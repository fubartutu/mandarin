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
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"code.vikunja.io/api/pkg/files"
	"code.vikunja.io/api/pkg/household"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// DownloadImage fetches a picture from the web, e.g. one an import found.
// Call it before opening a database session: it can take a while.
//
// Any household member can make the server fetch any address. That's fine
// for a household server, but it's why this needs a login.
func DownloadImage(ctx context.Context, imageURL string) ([]byte, error) {
	if m := paperlessImage.FindStringSubmatch(strings.TrimSpace(imageURL)); m != nil {
		data, _, err := callScraper(ctx, http.MethodGet, "/paperless/documents/"+m[1]+"/image", importTimeout, nil)
		return data, err
	}
	u, err := url.Parse(strings.TrimSpace(imageURL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, ErrImportFailed{Reason: "That isn't a web address."}
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	// Some image hosts refuse requests without a browser-like user agent.
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Mandarin recipe importer)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, ErrImportFailed{Reason: "Couldn't download the picture."}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, ErrImportFailed{Reason: fmt.Sprintf("The picture's site answered %d.", resp.StatusCode)}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, importedImageMax+1))
	if err != nil {
		return nil, ErrImportFailed{Reason: "Couldn't download the picture."}
	}
	if len(data) > importedImageMax {
		return nil, ErrImportFailed{Reason: "The picture is too big."}
	}
	return data, nil
}

// AddImage stores src (shrunk to fit 1600px, as JPEG) as the recipe's last
// image. Callers must check CanUpdate first.
func (r *Recipe) AddImage(s *xorm.Session, a web.Auth, src io.Reader) error {
	stored, err := getRecipe(s, r.ID)
	if err != nil {
		return err
	}
	if len(stored.ImageIDs) >= maxImages {
		return ErrTooManyImages{}
	}
	f, err := household.StoreJPEG(s, a, src, imageMaxEdge, "recipe.jpg")
	if errors.Is(err, household.ErrNotAnImage) {
		return ErrNotAnImage{}
	}
	if err != nil {
		return err
	}
	stored.ImageIDs = append(stored.ImageIDs, f.ID)
	return r.saveImages(s, stored)
}

// Image returns one of the recipe's images as JPEG bytes. Callers must check
// CanRead first.
func (r *Recipe) Image(imageID int64) ([]byte, error) {
	if !slices.Contains(r.ImageIDs, imageID) {
		return nil, ErrImageDoesNotExist{ImageID: imageID}
	}
	f := &files.File{ID: imageID}
	if err := f.LoadFileByID(); err != nil {
		if files.IsErrFileDoesNotExist(err) {
			return nil, ErrImageDoesNotExist{ImageID: imageID}
		}
		return nil, err
	}
	defer f.File.Close()
	return io.ReadAll(f.File)
}

// DeleteImage removes one image. Callers must check CanUpdate first.
func (r *Recipe) DeleteImage(s *xorm.Session, imageID int64) error {
	stored, err := getRecipe(s, r.ID)
	if err != nil {
		return err
	}
	i := slices.Index(stored.ImageIDs, imageID)
	if i < 0 {
		return ErrImageDoesNotExist{ImageID: imageID}
	}
	if err := deleteFile(s, imageID); err != nil {
		return err
	}
	stored.ImageIDs = slices.Delete(stored.ImageIDs, i, i+1)
	return r.saveImages(s, stored)
}

// MakeImageMain moves an image to the front, so it's the one shown in lists.
func (r *Recipe) MakeImageMain(s *xorm.Session, imageID int64) error {
	stored, err := getRecipe(s, r.ID)
	if err != nil {
		return err
	}
	i := slices.Index(stored.ImageIDs, imageID)
	if i < 0 {
		return ErrImageDoesNotExist{ImageID: imageID}
	}
	stored.ImageIDs = append([]int64{imageID}, slices.Delete(stored.ImageIDs, i, i+1)...)
	return r.saveImages(s, stored)
}

func (r *Recipe) saveImages(s *xorm.Session, stored *Recipe) error {
	if _, err := s.ID(stored.ID).Cols("image_ids").Update(stored); err != nil {
		return err
	}
	*r = *stored
	return fillCreators(s, r)
}
