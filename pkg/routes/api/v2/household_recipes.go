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

package apiv2

// Mandarin: /api/v2 routes for the household recipe box. A new file, like
// household_shopping.go. Permissions live on the model in
// pkg/household/recipes.

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/household/recipes"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/web/handler"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/conditional"
	"xorm.io/xorm"
)

func init() { AddRouteRegistrar(RegisterHouseholdRecipeRoutes) }

const recipeAccess = " Recipes belong to the household: every user on the server can read and change all of them; link shares cannot."

type recipeListBody struct {
	Body Paginated[*recipes.Recipe]
}

type recipeReadBody struct {
	recipes.Recipe
	MaxPermission models.Permission `json:"max_permission" readOnly:"true" doc:"The caller's permission on this recipe; always 2 (admin) for household members."`
}

type recipeURLInput struct {
	Body struct {
		URL string `json:"url" minLength:"1" maxLength:"2000" doc:"The web address."`
	}
}

type recipeImageUploadInput struct {
	ID int64 `path:"recipe"`
	// Same allow-list as the avatar upload: the formats the server can decode.
	RawBody huma.MultipartFormFiles[struct {
		Image huma.FormFile `form:"image" contentType:"image/png,image/jpeg,image/gif,image/tiff,image/bmp,application/octet-stream" required:"true" doc:"The picture. PNG, JPEG, GIF, TIFF or BMP; shrunk to fit 1600px and stored as JPEG."`
	}]
}

type recipeImagePath struct {
	ID      int64 `path:"recipe"`
	ImageID int64 `path:"image"`
}

type recipeImageResponse struct {
	ContentType  string `header:"Content-Type"`
	CacheControl string `header:"Cache-Control"`
	Body         []byte
}

// RegisterHouseholdRecipeRoutes wires recipes and their images onto the Huma API.
func RegisterHouseholdRecipeRoutes(api huma.API) {
	tags := []string{"recipes"}

	Register(api, huma.Operation{
		OperationID: "recipes-list",
		Summary:     "List recipes",
		Description: "By title. The search (q) matches titles, tags and ingredients." + recipeAccess,
		Method:      http.MethodGet, Path: "/recipes", Tags: tags,
	}, recipesList)
	Register(api, huma.Operation{
		OperationID: "recipes-read",
		Summary:     "Get a recipe",
		Description: "Sends an ETag; pass it as If-None-Match on a later read to get a 304 Not Modified.",
		Method:      http.MethodGet, Path: "/recipes/{recipe}", Tags: tags,
	}, recipesRead)
	Register(api, huma.Operation{
		OperationID: "recipes-create",
		Summary:     "Add a recipe",
		Description: "Send each ingredient's text (and group); the server works out amounts and names. Add pictures afterwards under /recipes/{recipe}/images.",
		Method:      http.MethodPost, Path: "/recipes", Tags: tags,
	}, recipesCreate)
	Register(api, huma.Operation{
		OperationID: "recipes-update",
		Summary:     "Update a recipe",
		Description: "Replaces everything except the pictures. Use PATCH for a partial update.",
		Method:      http.MethodPut, Path: "/recipes/{recipe}", Tags: tags,
	}, recipesUpdate)
	Register(api, huma.Operation{
		OperationID: "recipes-delete",
		Summary:     "Delete a recipe",
		Description: "Also deletes its pictures.",
		Method:      http.MethodDelete, Path: "/recipes/{recipe}", Tags: tags,
	}, recipesDelete)
	Register(api, huma.Operation{
		OperationID: "recipes-import",
		Summary:     "Read a recipe from a web page",
		Description: "Fetches the page through the scraper container and returns the recipe without saving it, with image_urls listing the pictures found. " +
			"Show it for checking, then save it with POST /recipes. 422 when there's no recipe on the page; 503 when the scraper isn't running.",
		Method: http.MethodPost, Path: "/recipes/import", Tags: tags,
		DefaultStatus: http.StatusOK, // nothing is created
	}, recipesImport)

	Register(api, huma.Operation{
		OperationID: "recipes-paperless-list",
		Summary:     "List recipe scans in Paperless-ngx",
		Description: "Documents with the recipe tag, newest first; q searches their text. 422 with a reason when Paperless isn't set up on the scraper.",
		Method:      http.MethodGet, Path: "/recipes/paperless", Tags: tags,
	}, recipesPaperlessList)
	Register(api, huma.Operation{
		OperationID: "recipes-paperless-thumb",
		Summary:     "Get a recipe scan's thumbnail",
		Method:      http.MethodGet, Path: "/recipes/paperless/{document}/thumb", Tags: tags,
		Responses: map[string]*huma.Response{
			"200": {
				Description: "Paperless's small preview (WebP or PNG).",
				Content: map[string]*huma.MediaType{
					"image/webp": {Schema: &huma.Schema{Type: huma.TypeString, Format: "binary"}},
				},
			},
		},
	}, recipesPaperlessThumb)
	Register(api, huma.Operation{
		OperationID: "recipes-paperless-import",
		Summary:     "Read a recipe scan from Paperless-ngx",
		Description: "Returns the recipe without saving it, like /recipes/import. Simple rules read the scan's text first and the AI on sampc takes over when they fall short; ai_first skips the rules. " +
			"image_urls holds the scanned page; fetch it after saving like any other.",
		Method: http.MethodPost, Path: "/recipes/paperless/{document}/import", Tags: tags,
		DefaultStatus: http.StatusOK, // nothing is created
	}, recipesPaperlessImport)

	Register(api, withUploadLimits(huma.Operation{
		OperationID: "recipes-image-upload",
		Summary:     "Add a picture to a recipe",
		Description: "Added last. The picture is shrunk to fit 1600px, turned upright from its EXIF orientation, and stored as JPEG. At most 4 per recipe.",
		Method:      http.MethodPost, Path: "/recipes/{recipe}/images", Tags: tags,
		DefaultStatus: http.StatusOK,
	}), recipesImageUpload)
	Register(api, huma.Operation{
		OperationID: "recipes-image-fetch",
		Summary:     "Add a picture to a recipe from the web",
		Description: "Downloads the picture (e.g. one of an import's image_urls) and adds it like an upload.",
		Method:      http.MethodPost, Path: "/recipes/{recipe}/images/fetch", Tags: tags,
		DefaultStatus: http.StatusOK,
	}, recipesImageFetch)
	Register(api, huma.Operation{
		OperationID: "recipes-image-get",
		Summary:     "Get a recipe picture",
		Description: "Returns the JPEG bytes. A picture never changes, so it may be cached for good.",
		Method:      http.MethodGet, Path: "/recipes/{recipe}/images/{image}", Tags: tags,
		Responses: map[string]*huma.Response{
			"200": {
				Description: "The picture as JPEG.",
				Content: map[string]*huma.MediaType{
					"image/jpeg": {Schema: &huma.Schema{Type: huma.TypeString, Format: "binary"}},
				},
			},
		},
	}, recipesImageGet)
	Register(api, huma.Operation{
		OperationID: "recipes-image-main",
		Summary:     "Make a picture the main one",
		Description: "Moves it to the front of image_ids; lists show the first picture.",
		Method:      http.MethodPost, Path: "/recipes/{recipe}/images/{image}/main", Tags: tags,
		DefaultStatus: http.StatusOK,
	}, recipesImageMain)
	Register(api, huma.Operation{
		OperationID: "recipes-image-delete",
		Summary:     "Remove a recipe picture",
		Method:      http.MethodDelete, Path: "/recipes/{recipe}/images/{image}", Tags: tags,
	}, recipesImageDelete)
}

func recipesList(ctx context.Context, in *struct{ ListParams }) (*recipeListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &recipes.Recipe{}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	list, ok := result.([]*recipes.Recipe)
	if !ok {
		return nil, fmt.Errorf("recipes ReadAll returned unexpected type %T", result)
	}
	return &recipeListBody{Body: NewPaginated(list, total, in.Page, in.PerPage)}, nil
}

func recipesRead(ctx context.Context, in *struct {
	ID int64 `path:"recipe"`
	conditional.Params
}) (*singleReadBody[recipeReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	r := &recipes.Recipe{ID: in.ID}
	maxPermission, err := handler.DoReadOne(ctx, r, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &recipeReadBody{Recipe: *r, MaxPermission: models.Permission(maxPermission)}
	return conditionalReadResponse(&in.Params, body, r.Updated, maxPermission)
}

func recipesCreate(ctx context.Context, in *struct {
	Body recipes.Recipe
}) (*singleBody[recipes.Recipe], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	// Parsed before the database session opens: it's a call to the scraper.
	recipes.ParseIngredients(ctx, in.Body.Ingredients)
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[recipes.Recipe]{Body: &in.Body}, nil
}

func recipesUpdate(ctx context.Context, in *struct {
	ID   int64 `path:"recipe"`
	Body recipeReadBody
}) (*singleBody[recipes.Recipe], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	r := &in.Body.Recipe
	r.ID = in.ID // URL wins over body
	recipes.ParseIngredients(ctx, r.Ingredients)
	if err := handler.DoUpdate(ctx, r, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[recipes.Recipe]{Body: r}, nil
}

func recipesDelete(ctx context.Context, in *struct {
	ID int64 `path:"recipe"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &recipes.Recipe{ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}

func recipesImport(ctx context.Context, in *recipeURLInput) (*singleBody[recipes.Recipe], error) {
	// Only members may make the server fetch pages.
	if err := requireRecipeMember(ctx); err != nil {
		return nil, err
	}
	r, err := recipes.Import(ctx, in.Body.URL)
	if err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[recipes.Recipe]{Body: r}, nil
}

// requireRecipeMember stops anyone else making the server fetch pages or
// read Paperless.
func requireRecipeMember(ctx context.Context) error {
	a, err := authFromCtx(ctx)
	if err != nil {
		return err
	}
	if can, _ := (&recipes.Recipe{}).CanCreate(nil, a); !can {
		return huma.Error403Forbidden("forbidden")
	}
	return nil
}

type recipePaperlessListBody struct {
	Body *recipes.PaperlessDocuments
}

func recipesPaperlessList(ctx context.Context, in *struct {
	Q string `query:"q" maxLength:"250" doc:"Words to search the scans' text for."`
}) (*recipePaperlessListBody, error) {
	if err := requireRecipeMember(ctx); err != nil {
		return nil, err
	}
	docs, err := recipes.ListPaperless(ctx, in.Q)
	if err != nil {
		return nil, translateDomainError(err)
	}
	return &recipePaperlessListBody{Body: docs}, nil
}

func recipesPaperlessThumb(ctx context.Context, in *struct {
	DocumentID int64 `path:"document"`
}) (*recipeImageResponse, error) {
	if err := requireRecipeMember(ctx); err != nil {
		return nil, err
	}
	data, contentType, err := recipes.PaperlessThumbnail(ctx, in.DocumentID)
	if err != nil {
		return nil, translateDomainError(err)
	}
	return &recipeImageResponse{ContentType: contentType, CacheControl: "private, max-age=3600", Body: data}, nil
}

func recipesPaperlessImport(ctx context.Context, in *struct {
	DocumentID int64 `path:"document"`
	Body       struct {
		AIFirst bool `json:"ai_first,omitempty" doc:"Go straight to the AI instead of trying the simple rules first."`
	}
}) (*singleBody[recipes.Recipe], error) {
	if err := requireRecipeMember(ctx); err != nil {
		return nil, err
	}
	r, err := recipes.ImportPaperless(ctx, in.DocumentID, in.Body.AIFirst)
	if err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[recipes.Recipe]{Body: r}, nil
}

// changeRecipe runs change in a session after checking the caller may
// update the recipe, and returns the recipe as it ends up. The image
// actions aren't CRUD, so they manage the session themselves.
func changeRecipe(ctx context.Context, id int64, change func(s *xorm.Session, r *recipes.Recipe) error) (*singleBody[recipes.Recipe], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()

	r := &recipes.Recipe{ID: id}
	can, err := r.CanUpdate(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if !can {
		_ = s.Rollback()
		return nil, huma.Error403Forbidden("forbidden")
	}
	if err := change(s, r); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[recipes.Recipe]{Body: r}, nil
}

func recipesImageUpload(ctx context.Context, in *recipeImageUploadInput) (*singleBody[recipes.Recipe], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	src := in.RawBody.Data().Image
	defer func() { _ = src.Close() }()
	return changeRecipe(ctx, in.ID, func(s *xorm.Session, r *recipes.Recipe) error {
		return r.AddImage(ctx, s, a, src)
	})
}

func recipesImageFetch(ctx context.Context, in *struct {
	ID int64 `path:"recipe"`
	recipeURLInput
}) (*singleBody[recipes.Recipe], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := requireRecipeMember(ctx); err != nil {
		return nil, err
	}
	// Downloaded before the session opens, so a slow site doesn't hold the
	// database (SQLite allows one writer at a time).
	data, err := recipes.DownloadImage(ctx, in.Body.URL)
	if err != nil {
		return nil, translateDomainError(err)
	}
	return changeRecipe(ctx, in.ID, func(s *xorm.Session, r *recipes.Recipe) error {
		return r.AddImage(ctx, s, a, bytes.NewReader(data))
	})
}

func recipesImageMain(ctx context.Context, in *recipeImagePath) (*singleBody[recipes.Recipe], error) {
	return changeRecipe(ctx, in.ID, func(s *xorm.Session, r *recipes.Recipe) error {
		return r.MakeImageMain(s, in.ImageID)
	})
}

func recipesImageDelete(ctx context.Context, in *recipeImagePath) (*emptyBody, error) {
	if _, err := changeRecipe(ctx, in.ID, func(s *xorm.Session, r *recipes.Recipe) error {
		return r.DeleteImage(s, in.ImageID)
	}); err != nil {
		return nil, err
	}
	return &emptyBody{}, nil
}

func recipesImageGet(ctx context.Context, in *recipeImagePath) (*recipeImageResponse, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()

	r := &recipes.Recipe{ID: in.ID}
	if _, _, err := r.CanRead(s, a); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	_ = s.Commit()
	data, err := r.Image(in.ImageID)
	if err != nil {
		return nil, translateDomainError(err)
	}
	// Private: it needs a login. Image ids are never reused, so cache for good.
	return &recipeImageResponse{ContentType: "image/jpeg", CacheControl: "private, max-age=31536000, immutable", Body: data}, nil
}
