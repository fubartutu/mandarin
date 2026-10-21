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
const (
	importTimeout = 45 * time.Second
	parseTimeout  = 10 * time.Second
)

// postScraper sends body as JSON to the scraper and decodes the answer into
// out. A scraper that's down gives ErrScraperUnavailable; one that says no
// gives ErrImportFailed with its reason.
func postScraper(ctx context.Context, path string, timeout time.Duration, body, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ScraperURL()+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ErrScraperUnavailable{}
	}
	defer resp.Body.Close()
	answer, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return ErrScraperUnavailable{}
	}
	if resp.StatusCode != http.StatusOK {
		// FastAPI puts the reason in "detail".
		var problem struct {
			Detail any `json:"detail"`
		}
		_ = json.Unmarshal(answer, &problem)
		if reason, ok := problem.Detail.(string); ok && reason != "" {
			return ErrImportFailed{Reason: reason}
		}
		return ErrImportFailed{Reason: fmt.Sprintf("The importer answered %d.", resp.StatusCode)}
	}
	return json.Unmarshal(answer, out)
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
