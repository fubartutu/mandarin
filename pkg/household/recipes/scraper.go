// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package recipes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// The scraper is a separate container (household-scraper/ in the repo). It
// has no login, so it must only be reachable on the Docker network.
const defaultScraperURL = "http://mandarin-scraper:8000"

// ScraperURL is where the scraper listens: MANDARIN_SCRAPER_URL, or the
// container name on the shared Docker network.
func ScraperURL() string {
	if v := strings.TrimSpace(os.Getenv("MANDARIN_SCRAPER_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return defaultScraperURL
}

// How long to wait for the scraper. A recipe page can be slow to load;
// parsing lines takes well under a second.
// The AI on sampc may have to load its model first, so imports get minutes.
const (
	importTimeout = 5 * time.Minute
	parseTimeout  = 10 * time.Second
	listTimeout   = 30 * time.Second
)

// postScraper sends body as JSON to the scraper and decodes the answer into
// out. A scraper that's down gives ErrScraperUnavailable; one that says no
// gives ErrImportFailed with its reason.
func postScraper(ctx context.Context, path string, timeout time.Duration, body, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	answer, _, err := callScraper(ctx, http.MethodPost, path, timeout, bytes.NewReader(data))
	if err != nil {
		return err
	}
	return json.Unmarshal(answer, out)
}

// callScraper makes one request to the scraper and returns the answer's
// bytes and content type.
func callScraper(ctx context.Context, method, path string, timeout time.Duration, body io.Reader) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, ScraperURL()+path, body)
	if err != nil {
		return nil, "", err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", ErrScraperUnavailable{}
	}
	defer resp.Body.Close()
	answer, err := io.ReadAll(io.LimitReader(resp.Body, importedImageMax+1))
	if err != nil {
		return nil, "", ErrScraperUnavailable{}
	}
	if resp.StatusCode != http.StatusOK {
		// FastAPI puts the reason in "detail".
		var problem struct {
			Detail any `json:"detail"`
		}
		_ = json.Unmarshal(answer, &problem)
		if reason, ok := problem.Detail.(string); ok && reason != "" {
			return nil, "", ErrImportFailed{Reason: reason}
		}
		return nil, "", ErrImportFailed{Reason: fmt.Sprintf("The importer answered %d.", resp.StatusCode)}
	}
	return answer, resp.Header.Get("Content-Type"), nil
}

// Import fetches a recipe page through the scraper and returns it unsaved,
// with ImageURLs set, for the app to show as a preview.
func Import(ctx context.Context, pageURL string) (*Recipe, error) {
	pageURL = strings.TrimSpace(pageURL)
	if u, err := url.Parse(pageURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, ErrImportFailed{Reason: "That isn't a web address."}
	}
	// The scraper answers with the same field names as Recipe.
	r := &Recipe{}
	if err := postScraper(ctx, "/scrape", importTimeout, map[string]string{"url": pageURL}, r); err != nil {
		return nil, err
	}
	return finishImport(r)
}

// finishImport tidies an import preview the way a save would.
func finishImport(r *Recipe) (*Recipe, error) {
	if r.ImageURLs == nil {
		r.ImageURLs = []string{}
	}
	// Something odd on the page (a 600-line ingredient list, say) fails the
	// import with the reason, rather than failing the save later.
	var invalid ErrInvalidRecipe
	if err := r.normalize(); errors.As(err, &invalid) {
		return nil, ErrImportFailed(invalid)
	} else if err != nil {
		return nil, err
	}
	return r, nil
}

// ParseIngredients works out every line's amount and name with the
// scraper's trained parser. When the scraper is down the lines are left
// unparsed and the recipe's save falls back to the simple parser, so
// saving never depends on the scraper.
func ParseIngredients(ctx context.Context, ingredients []*RecipeIngredient) {
	lines := make([]string, 0, len(ingredients))
	for _, ing := range ingredients {
		if ing == nil {
			lines = append(lines, "")
			continue
		}
		ing.Quantity, ing.Unit, ing.Name, ing.Note = 0, "", "", ""
		lines = append(lines, ing.Text)
	}
	if len(lines) == 0 || len(lines) > maxIngredients {
		return
	}
	var parsed struct {
		Ingredients []*RecipeIngredient `json:"ingredients"`
	}
	if err := postScraper(ctx, "/parse", parseTimeout, map[string][]string{"lines": lines}, &parsed); err != nil {
		return
	}
	if len(parsed.Ingredients) != len(ingredients) {
		return
	}
	for i, ing := range ingredients {
		if ing == nil || parsed.Ingredients[i] == nil {
			continue
		}
		p := parsed.Ingredients[i]
		ing.Quantity, ing.Unit, ing.Name, ing.Note = p.Quantity, p.Unit, p.Name, p.Note
	}
}

// PaperlessDocument is a recipe scan in Paperless-ngx.
type PaperlessDocument struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	Created string `json:"created" doc:"When the document was created, as Paperless gives it."`
}

// PaperlessDocuments is the list of recipe scans.
type PaperlessDocuments struct {
	Documents   []*PaperlessDocument `json:"documents" doc:"Documents with the recipe tag, newest first."`
	AIAvailable bool                 `json:"ai_available" doc:"Whether imports can use the AI."`
}

// ListPaperless returns the recipe scans in Paperless, optionally searched.
func ListPaperless(ctx context.Context, search string) (*PaperlessDocuments, error) {
	answer, _, err := callScraper(ctx, http.MethodGet, "/paperless/documents?q="+url.QueryEscape(search), listTimeout, nil)
	if err != nil {
		return nil, err
	}
	docs := &PaperlessDocuments{}
	if err := json.Unmarshal(answer, docs); err != nil {
		return nil, err
	}
	if docs.Documents == nil {
		docs.Documents = []*PaperlessDocument{}
	}
	return docs, nil
}

// PaperlessThumbnail returns a scan's small preview and its content type.
func PaperlessThumbnail(ctx context.Context, docID int64) ([]byte, string, error) {
	return callScraper(ctx, http.MethodGet, fmt.Sprintf("/paperless/documents/%d/thumb", docID), listTimeout, nil)
}

// ImportPaperless reads a scan as a recipe and returns it unsaved. The
// scraper tries simple rules first and the AI when they fall short; aiFirst
// skips the rules (the app's "try the AI" button).
func ImportPaperless(ctx context.Context, docID int64, aiFirst bool) (*Recipe, error) {
	r := &Recipe{}
	path := fmt.Sprintf("/paperless/documents/%d/import?ai_first=%t", docID, aiFirst)
	if err := postScraper(ctx, path, importTimeout, struct{}{}, r); err != nil {
		return nil, err
	}
	return finishImport(r)
}

// paperlessImage matches the picture address a scan's import gives:
// the page is fetched from the scraper, which has the Paperless token.
var paperlessImage = regexp.MustCompile(`^paperless:documents/(\d+)/image$`)
