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
	"fmt"
	"net/http"

	"code.vikunja.io/api/pkg/web"
)

// Recipe error codes are 901xx (shopping has 900xx).

// ErrRecipeDoesNotExist is returned when a recipe id is unknown.
type ErrRecipeDoesNotExist struct {
	RecipeID int64
}

func (err ErrRecipeDoesNotExist) Error() string {
	return fmt.Sprintf("Recipe does not exist [RecipeID: %d]", err.RecipeID)
}

// ErrCodeRecipeDoesNotExist holds the unique world-error code of this error
const ErrCodeRecipeDoesNotExist = 90101

// HTTPError holds the http error description
func (err ErrRecipeDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusNotFound, Code: ErrCodeRecipeDoesNotExist, Message: "This recipe does not exist."}
}

// ErrInvalidRecipe is returned when a recipe breaks one of the limits that
// the API schema can't express (e.g. the length of one ingredient line).
type ErrInvalidRecipe struct {
	Reason string
}

func (err ErrInvalidRecipe) Error() string {
	return "Invalid recipe: " + err.Reason
}

// ErrCodeInvalidRecipe holds the unique world-error code of this error
const ErrCodeInvalidRecipe = 90102

// HTTPError holds the http error description
func (err ErrInvalidRecipe) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusBadRequest, Code: ErrCodeInvalidRecipe, Message: err.Reason}
}

// ErrScraperUnavailable is returned when the scraper container can't be reached.
type ErrScraperUnavailable struct{}

func (err ErrScraperUnavailable) Error() string {
	return "Recipe scraper is unavailable"
}

// ErrCodeScraperUnavailable holds the unique world-error code of this error
const ErrCodeScraperUnavailable = 90103

// HTTPError holds the http error description
func (err ErrScraperUnavailable) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusServiceUnavailable, Code: ErrCodeScraperUnavailable, Message: "The recipe importer isn't running. Check the mandarin-scraper container."}
}

// ErrImportFailed is returned when the scraper couldn't get a recipe from the page.
type ErrImportFailed struct {
	Reason string
}

func (err ErrImportFailed) Error() string {
	return "Recipe import failed: " + err.Reason
}

// ErrCodeImportFailed holds the unique world-error code of this error
const ErrCodeImportFailed = 90104

// HTTPError holds the http error description
func (err ErrImportFailed) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusUnprocessableEntity, Code: ErrCodeImportFailed, Message: err.Reason}
}

// ErrImageDoesNotExist is returned when an image id isn't one of the recipe's images.
type ErrImageDoesNotExist struct {
	ImageID int64
}

func (err ErrImageDoesNotExist) Error() string {
	return fmt.Sprintf("Recipe image does not exist [ImageID: %d]", err.ImageID)
}

// ErrCodeImageDoesNotExist holds the unique world-error code of this error
const ErrCodeImageDoesNotExist = 90105

// HTTPError holds the http error description
func (err ErrImageDoesNotExist) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusNotFound, Code: ErrCodeImageDoesNotExist, Message: "This recipe has no such image."}
}

// ErrNotAnImage is returned when an uploaded or fetched picture can't be read as an image.
type ErrNotAnImage struct{}

func (err ErrNotAnImage) Error() string {
	return "Not an image"
}

// ErrCodeNotAnImage holds the unique world-error code of this error
const ErrCodeNotAnImage = 90106

// HTTPError holds the http error description
func (err ErrNotAnImage) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusBadRequest, Code: ErrCodeNotAnImage, Message: "That isn't an image (PNG, JPEG, GIF, TIFF or BMP)."}
}

// ErrTooManyImages is returned when a recipe already has the most images allowed.
type ErrTooManyImages struct{}

func (err ErrTooManyImages) Error() string {
	return "Too many recipe images"
}

// ErrCodeTooManyImages holds the unique world-error code of this error
const ErrCodeTooManyImages = 90107

// HTTPError holds the http error description
func (err ErrTooManyImages) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusBadRequest, Code: ErrCodeTooManyImages, Message: fmt.Sprintf("A recipe can have at most %d images.", maxImages)}
}
