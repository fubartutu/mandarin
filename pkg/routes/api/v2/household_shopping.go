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

// Mandarin: /api/v2 routes for the household shopping module. This file is
// new (not an edit to a Vikunja file): it lives in this package because the
// v2 helpers it uses (authFromCtx, translateDomainError, the envelopes) are
// unexported. Permissions live on the models in pkg/household/shopping.

import (
	"context"
	"fmt"
	"net/http"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/events"
	"code.vikunja.io/api/pkg/household/shopping"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/web/handler"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/conditional"
)

func init() { AddRouteRegistrar(RegisterHouseholdShoppingRoutes) }

// RegisterHouseholdShoppingRoutes wires shopping lists, items and categories onto the Huma API.
func RegisterHouseholdShoppingRoutes(api huma.API) {
	registerShoppingListRoutes(api)
	registerShoppingItemRoutes(api)
	registerShoppingCategoryRoutes(api)
	registerShoppingShopRoutes(api)
	registerShoppingCatalogRoutes(api)
	registerShoppingSyncRoutes(api)
}

const shoppingAccess = " Shopping data belongs to the household: every user on the server can read and change all of it; link shares cannot."

// ---- Lists ----

type shoppingListListBody struct {
	Body Paginated[*shopping.ShoppingList]
}

type shoppingListReadBody struct {
	shopping.ShoppingList
	MaxPermission models.Permission `json:"max_permission" readOnly:"true" doc:"The caller's permission on this list; always 2 (admin) for household members."`
}

// ShoppingClearDoneResult is the response of the clear-done action.
type ShoppingClearDoneResult struct {
	Deleted int64 `json:"deleted" doc:"How many ticked items were removed."`
}

type shoppingClearDoneBody struct {
	Body *ShoppingClearDoneResult
}

func registerShoppingListRoutes(api huma.API) {
	tags := []string{"shopping"}

	Register(api, huma.Operation{
		OperationID: "shopping-lists-list",
		Summary:     "List shopping lists",
		Description: "Returns the household's shopping lists ordered by position." + shoppingAccess,
		Method:      http.MethodGet, Path: "/shopping/lists", Tags: tags,
	}, shoppingListsList)
	Register(api, huma.Operation{
		OperationID: "shopping-lists-read",
		Summary:     "Get a shopping list",
		Description: "Sends an ETag; pass it as If-None-Match on a later read to get a 304 Not Modified.",
		Method:      http.MethodGet, Path: "/shopping/lists/{list}", Tags: tags,
	}, shoppingListsRead)
	Register(api, huma.Operation{
		OperationID: "shopping-lists-create",
		Summary:     "Create a shopping list",
		Description: "The caller is recorded as the creator; the list is shared with the whole household.",
		Method:      http.MethodPost, Path: "/shopping/lists", Tags: tags,
	}, shoppingListsCreate)
	Register(api, huma.Operation{
		OperationID: "shopping-lists-update",
		Summary:     "Update a shopping list",
		Description: "Replaces the title and position. Use PATCH for a partial update.",
		Method:      http.MethodPut, Path: "/shopping/lists/{list}", Tags: tags,
	}, shoppingListsUpdate)
	Register(api, huma.Operation{
		OperationID: "shopping-lists-delete",
		Summary:     "Delete a shopping list",
		Description: "Deletes the list and every item on it.",
		Method:      http.MethodDelete, Path: "/shopping/lists/{list}", Tags: tags,
	}, shoppingListsDelete)
	Register(api, huma.Operation{
		OperationID: "shopping-lists-clear-done",
		Summary:     "Clear ticked items",
		Description: "Deletes every ticked (done) item on the list and returns how many were removed.",
		Method:      http.MethodPost, Path: "/shopping/lists/{list}/clear-done", Tags: tags,
		DefaultStatus: http.StatusOK, // an action, not a create
	}, shoppingListsClearDone)
}

func shoppingListsList(ctx context.Context, in *struct{ ListParams }) (*shoppingListListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &shopping.ShoppingList{}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	lists, ok := result.([]*shopping.ShoppingList)
	if !ok {
		return nil, fmt.Errorf("shopping lists ReadAll returned unexpected type %T", result)
	}
	return &shoppingListListBody{Body: NewPaginated(lists, total, in.Page, in.PerPage)}, nil
}

func shoppingListsRead(ctx context.Context, in *struct {
	ID int64 `path:"list"`
	conditional.Params
}) (*singleReadBody[shoppingListReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	list := &shopping.ShoppingList{ID: in.ID}
	maxPermission, err := handler.DoReadOne(ctx, list, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &shoppingListReadBody{ShoppingList: *list, MaxPermission: models.Permission(maxPermission)}
	return conditionalReadResponse(&in.Params, body, list.Updated, maxPermission)
}

func shoppingListsCreate(ctx context.Context, in *struct {
	Body shopping.ShoppingList
}) (*singleBody[shopping.ShoppingList], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[shopping.ShoppingList]{Body: &in.Body}, nil
}

func shoppingListsUpdate(ctx context.Context, in *struct {
	ID   int64 `path:"list"`
	Body shoppingListReadBody
}) (*singleBody[shopping.ShoppingList], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	list := &in.Body.ShoppingList
	list.ID = in.ID // URL wins over body
	if err := handler.DoUpdate(ctx, list, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[shopping.ShoppingList]{Body: list}, nil
}

func shoppingListsDelete(ctx context.Context, in *struct {
	ID int64 `path:"list"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &shopping.ShoppingList{ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}

// Custom action: no generic handler.Do*, so this handler checks the
// permission and manages the session itself.
func shoppingListsClearDone(ctx context.Context, in *struct {
	ID int64 `path:"list"`
}) (*shoppingClearDoneBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()
	// Drops the "shopping changed" events queued by a failed attempt.
	defer events.CleanupPending(s)

	list := &shopping.ShoppingList{ID: in.ID}
	can, err := list.CanUpdate(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if !can {
		_ = s.Rollback()
		return nil, huma.Error403Forbidden("forbidden")
	}
	deleted, err := list.ClearDoneItems(s)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	events.DispatchPending(ctx, s)
	return &shoppingClearDoneBody{Body: &ShoppingClearDoneResult{Deleted: deleted}}, nil
}

// ---- Items ----

type shoppingItemListBody struct {
	Body Paginated[*shopping.ShoppingItem]
}

type shoppingItemReadBody struct {
	shopping.ShoppingItem
	MaxPermission models.Permission `json:"max_permission" readOnly:"true" doc:"The caller's permission on this item; always 2 (admin) for household members."`
}

func registerShoppingItemRoutes(api huma.API) {
	tags := []string{"shopping"}

	Register(api, huma.Operation{
		OperationID: "shopping-items-list",
		Summary:     "List the items on a shopping list",
		Description: "Not-done items come first, then ticked ones; within each, by position. Clients group them by category_id.",
		Method:      http.MethodGet, Path: "/shopping/lists/{list}/items", Tags: tags,
	}, shoppingItemsList)
	Register(api, huma.Operation{
		OperationID: "shopping-items-read",
		Summary:     "Get a shopping item",
		Description: "404 if the item is not on the list in the URL.",
		Method:      http.MethodGet, Path: "/shopping/lists/{list}/items/{item}", Tags: tags,
	}, shoppingItemsRead)
	Register(api, huma.Operation{
		OperationID: "shopping-items-create",
		Summary:     "Add an item to a shopping list",
		Description: "category_id must be 0 or an existing category.",
		Method:      http.MethodPost, Path: "/shopping/lists/{list}/items", Tags: tags,
	}, shoppingItemsCreate)
	Register(api, huma.Operation{
		OperationID: "shopping-items-update",
		Summary:     "Update a shopping item",
		Description: "Replaces the item's fields; use PATCH to change one (e.g. {\"done\": true} to tick it). Ticking sets done_at, unticking clears it. Items can't move between lists.",
		Method:      http.MethodPut, Path: "/shopping/lists/{list}/items/{item}", Tags: tags,
	}, shoppingItemsUpdate)
	Register(api, huma.Operation{
		OperationID: "shopping-items-delete",
		Summary:     "Delete a shopping item",
		Method:      http.MethodDelete, Path: "/shopping/lists/{list}/items/{item}", Tags: tags,
	}, shoppingItemsDelete)
}

func shoppingItemsList(ctx context.Context, in *struct {
	ListID int64 `path:"list"`
	ListParams
}) (*shoppingItemListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &shopping.ShoppingItem{ListID: in.ListID}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	items, ok := result.([]*shopping.ShoppingItem)
	if !ok {
		return nil, fmt.Errorf("shopping items ReadAll returned unexpected type %T", result)
	}
	return &shoppingItemListBody{Body: NewPaginated(items, total, in.Page, in.PerPage)}, nil
}

func shoppingItemsRead(ctx context.Context, in *struct {
	ListID int64 `path:"list"`
	ID     int64 `path:"item"`
	conditional.Params
}) (*singleReadBody[shoppingItemReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	item := &shopping.ShoppingItem{ListID: in.ListID, ID: in.ID}
	maxPermission, err := handler.DoReadOne(ctx, item, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &shoppingItemReadBody{ShoppingItem: *item, MaxPermission: models.Permission(maxPermission)}
	return conditionalReadResponse(&in.Params, body, item.Updated, maxPermission)
}

func shoppingItemsCreate(ctx context.Context, in *struct {
	ListID int64 `path:"list"`
	Body   shopping.ShoppingItem
}) (*singleBody[shopping.ShoppingItem], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	in.Body.ListID = in.ListID
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[shopping.ShoppingItem]{Body: &in.Body}, nil
}

func shoppingItemsUpdate(ctx context.Context, in *struct {
	ListID int64 `path:"list"`
	ID     int64 `path:"item"`
	Body   shoppingItemReadBody
}) (*singleBody[shopping.ShoppingItem], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	item := &in.Body.ShoppingItem
	item.ListID = in.ListID // URL wins over body
	item.ID = in.ID
	if err := handler.DoUpdate(ctx, item, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[shopping.ShoppingItem]{Body: item}, nil
}

func shoppingItemsDelete(ctx context.Context, in *struct {
	ListID int64 `path:"list"`
	ID     int64 `path:"item"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &shopping.ShoppingItem{ListID: in.ListID, ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}

// ---- Categories ----

type shoppingCategoryListBody struct {
	Body Paginated[*shopping.ShoppingCategory]
}

type shoppingCategoryReadBody struct {
	shopping.ShoppingCategory
	MaxPermission models.Permission `json:"max_permission" readOnly:"true" doc:"The caller's permission on this category; always 2 (admin) for household members."`
}

func registerShoppingCategoryRoutes(api huma.API) {
	tags := []string{"shopping"}

	Register(api, huma.Operation{
		OperationID: "shopping-categories-list",
		Summary:     "List shopping categories",
		Description: "Returns the household's categories in display order; one order is shared by every list." + shoppingAccess,
		Method:      http.MethodGet, Path: "/shopping/categories", Tags: tags,
	}, shoppingCategoriesList)
	Register(api, huma.Operation{
		OperationID: "shopping-categories-read",
		Summary:     "Get a shopping category",
		Method:      http.MethodGet, Path: "/shopping/categories/{id}", Tags: tags,
	}, shoppingCategoriesRead)
	Register(api, huma.Operation{
		OperationID: "shopping-categories-create",
		Summary:     "Create a shopping category",
		Method:      http.MethodPost, Path: "/shopping/categories", Tags: tags,
	}, shoppingCategoriesCreate)
	Register(api, huma.Operation{
		OperationID: "shopping-categories-update",
		Summary:     "Update a shopping category",
		Description: "Replaces the title and position. Use PATCH for a partial update.",
		Method:      http.MethodPut, Path: "/shopping/categories/{id}", Tags: tags,
	}, shoppingCategoriesUpdate)
	Register(api, huma.Operation{
		OperationID: "shopping-categories-delete",
		Summary:     "Delete a shopping category",
		Description: "Items in the category become uncategorised (category_id 0); they are not deleted.",
		Method:      http.MethodDelete, Path: "/shopping/categories/{id}", Tags: tags,
	}, shoppingCategoriesDelete)
}

func shoppingCategoriesList(ctx context.Context, in *struct{ ListParams }) (*shoppingCategoryListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &shopping.ShoppingCategory{}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	categories, ok := result.([]*shopping.ShoppingCategory)
	if !ok {
		return nil, fmt.Errorf("shopping categories ReadAll returned unexpected type %T", result)
	}
	return &shoppingCategoryListBody{Body: NewPaginated(categories, total, in.Page, in.PerPage)}, nil
}

func shoppingCategoriesRead(ctx context.Context, in *struct {
	ID int64 `path:"id"`
	conditional.Params
}) (*singleReadBody[shoppingCategoryReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	category := &shopping.ShoppingCategory{ID: in.ID}
	maxPermission, err := handler.DoReadOne(ctx, category, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &shoppingCategoryReadBody{ShoppingCategory: *category, MaxPermission: models.Permission(maxPermission)}
	return conditionalReadResponse(&in.Params, body, category.Updated, maxPermission)
}

func shoppingCategoriesCreate(ctx context.Context, in *struct {
	Body shopping.ShoppingCategory
}) (*singleBody[shopping.ShoppingCategory], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[shopping.ShoppingCategory]{Body: &in.Body}, nil
}

func shoppingCategoriesUpdate(ctx context.Context, in *struct {
	ID   int64 `path:"id"`
	Body shoppingCategoryReadBody
}) (*singleBody[shopping.ShoppingCategory], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	category := &in.Body.ShoppingCategory
	category.ID = in.ID
	if err := handler.DoUpdate(ctx, category, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[shopping.ShoppingCategory]{Body: category}, nil
}

func shoppingCategoriesDelete(ctx context.Context, in *struct {
	ID int64 `path:"id"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &shopping.ShoppingCategory{ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}

// ---- Shops ----

type shoppingShopListBody struct {
	Body Paginated[*shopping.ShoppingShop]
}

type shoppingShopReadBody struct {
	shopping.ShoppingShop
	MaxPermission models.Permission `json:"max_permission" readOnly:"true" doc:"The caller's permission on this shop; always 2 (admin) for household members."`
}

func registerShoppingShopRoutes(api huma.API) {
	tags := []string{"shopping"}

	Register(api, huma.Operation{
		OperationID: "shopping-shops-list",
		Summary:     "List a shopping list's shops",
		Description: "Shops are views of their list: each hides the items marked not stocked there. The list read already includes its shops.",
		Method:      http.MethodGet, Path: "/shopping/lists/{list}/shops", Tags: tags,
	}, shoppingShopsList)
	Register(api, huma.Operation{
		OperationID: "shopping-shops-read",
		Summary:     "Get a shop",
		Description: "404 if the shop doesn't belong to the list in the URL.",
		Method:      http.MethodGet, Path: "/shopping/lists/{list}/shops/{shop}", Tags: tags,
	}, shoppingShopsRead)
	Register(api, huma.Operation{
		OperationID: "shopping-shops-create",
		Summary:     "Add a shop to a shopping list",
		Description: "Without a position the shop's tab goes last.",
		Method:      http.MethodPost, Path: "/shopping/lists/{list}/shops", Tags: tags,
	}, shoppingShopsCreate)
	Register(api, huma.Operation{
		OperationID: "shopping-shops-update",
		Summary:     "Update a shop",
		Description: "Replaces the title and position. Use PATCH for a partial update.",
		Method:      http.MethodPut, Path: "/shopping/lists/{list}/shops/{shop}", Tags: tags,
	}, shoppingShopsUpdate)
	Register(api, huma.Operation{
		OperationID: "shopping-shops-delete",
		Summary:     "Delete a shop",
		Description: "Also removes every not-stocked mark for the shop. The list's items are untouched.",
		Method:      http.MethodDelete, Path: "/shopping/lists/{list}/shops/{shop}", Tags: tags,
	}, shoppingShopsDelete)
}

func shoppingShopsList(ctx context.Context, in *struct {
	ListID int64 `path:"list"`
	ListParams
}) (*shoppingShopListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &shopping.ShoppingShop{ListID: in.ListID}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	shops, ok := result.([]*shopping.ShoppingShop)
	if !ok {
		return nil, fmt.Errorf("shopping shops ReadAll returned unexpected type %T", result)
	}
	return &shoppingShopListBody{Body: NewPaginated(shops, total, in.Page, in.PerPage)}, nil
}

func shoppingShopsRead(ctx context.Context, in *struct {
	ListID int64 `path:"list"`
	ID     int64 `path:"shop"`
	conditional.Params
}) (*singleReadBody[shoppingShopReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	shop := &shopping.ShoppingShop{ListID: in.ListID, ID: in.ID}
	maxPermission, err := handler.DoReadOne(ctx, shop, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &shoppingShopReadBody{ShoppingShop: *shop, MaxPermission: models.Permission(maxPermission)}
	return conditionalReadResponse(&in.Params, body, shop.Updated, maxPermission)
}

func shoppingShopsCreate(ctx context.Context, in *struct {
	ListID int64 `path:"list"`
	Body   shopping.ShoppingShop
}) (*singleBody[shopping.ShoppingShop], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	in.Body.ListID = in.ListID
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[shopping.ShoppingShop]{Body: &in.Body}, nil
}

func shoppingShopsUpdate(ctx context.Context, in *struct {
	ListID int64 `path:"list"`
	ID     int64 `path:"shop"`
	Body   shoppingShopReadBody
}) (*singleBody[shopping.ShoppingShop], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	shop := &in.Body.ShoppingShop
	shop.ListID = in.ListID
	shop.ID = in.ID
	if err := handler.DoUpdate(ctx, shop, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[shopping.ShoppingShop]{Body: shop}, nil
}

func shoppingShopsDelete(ctx context.Context, in *struct {
	ListID int64 `path:"list"`
	ID     int64 `path:"shop"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &shopping.ShoppingShop{ListID: in.ListID, ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}

// ---- Catalogue ----

type shoppingCatalogListBody struct {
	Body Paginated[*shopping.ShoppingCatalogItem]
}

type shoppingCatalogReadBody struct {
	shopping.ShoppingCatalogItem
	MaxPermission models.Permission `json:"max_permission" readOnly:"true" doc:"The caller's permission on this entry; always 2 (admin) for household members."`
}

type shoppingPhotoUploadInput struct {
	ID int64 `path:"id"`
	// Same allow-list as the avatar upload: the formats the server can decode.
	RawBody huma.MultipartFormFiles[struct {
		Photo huma.FormFile `form:"photo" contentType:"image/png,image/jpeg,image/gif,image/tiff,image/bmp,application/octet-stream" required:"true" doc:"The photo. PNG, JPEG, GIF, TIFF or BMP; shrunk to fit 1024px and stored as JPEG."`
	}]
}

type shoppingPhotoResponse struct {
	ContentType  string `header:"Content-Type"`
	CacheControl string `header:"Cache-Control"`
	Body         []byte
}

func registerShoppingCatalogRoutes(api huma.API) {
	tags := []string{"shopping"}

	Register(api, huma.Operation{
		OperationID: "shopping-catalog-list",
		Summary:     "List the item catalogue",
		Description: "Every item name the household has used plus the pre-loaded ones, most often and most recently added first. q filters by name. Clients usually fetch it all for autocomplete." + shoppingAccess,
		Method:      http.MethodGet, Path: "/shopping/catalog", Tags: tags,
	}, shoppingCatalogList)
	Register(api, huma.Operation{
		OperationID: "shopping-catalog-read",
		Summary:     "Get a catalogue entry",
		Method:      http.MethodGet, Path: "/shopping/catalog/{id}", Tags: tags,
	}, shoppingCatalogRead)
	Register(api, huma.Operation{
		OperationID: "shopping-catalog-create",
		Summary:     "Add a catalogue entry",
		Description: "Adding an item to a list does this automatically; use this to add one without listing it. Names are unique ignoring case (409 if taken).",
		Method:      http.MethodPost, Path: "/shopping/catalog", Tags: tags,
	}, shoppingCatalogCreate)
	Register(api, huma.Operation{
		OperationID: "shopping-catalog-update",
		Summary:     "Update a catalogue entry",
		Description: "Rename it, or change its category, remembered quantity/unit and not-stocked shops (replaced as a whole). Use PATCH, e.g. {\"not_stocked_shop_ids\": [3]}.",
		Method:      http.MethodPut, Path: "/shopping/catalog/{id}", Tags: tags,
	}, shoppingCatalogUpdate)
	Register(api, huma.Operation{
		OperationID: "shopping-catalog-delete",
		Summary:     "Delete a catalogue entry",
		Description: "Also deletes its photo and not-stocked marks. Items already on lists keep their name and lose the link.",
		Method:      http.MethodDelete, Path: "/shopping/catalog/{id}", Tags: tags,
	}, shoppingCatalogDelete)

	Register(api, withUploadLimits(huma.Operation{
		OperationID: "shopping-catalog-photo-upload",
		Summary:     "Set a catalogue entry's photo",
		Description: "Replaces any earlier photo. The image is shrunk to fit 1024px, turned upright from its EXIF orientation, and stored as JPEG.",
		Method:      http.MethodPut, Path: "/shopping/catalog/{id}/photo", Tags: tags,
		DefaultStatus: http.StatusOK,
	}), shoppingCatalogPhotoUpload)
	Register(api, huma.Operation{
		OperationID: "shopping-catalog-photo-get",
		Summary:     "Get a catalogue entry's photo",
		Description: "Returns the JPEG bytes; 404 if the entry has no photo.",
		Method:      http.MethodGet, Path: "/shopping/catalog/{id}/photo", Tags: tags,
		Responses: map[string]*huma.Response{
			"200": {
				Description: "The photo as JPEG.",
				Content: map[string]*huma.MediaType{
					"image/jpeg": {Schema: &huma.Schema{Type: huma.TypeString, Format: "binary"}},
				},
			},
		},
	}, shoppingCatalogPhotoGet)
	Register(api, huma.Operation{
		OperationID: "shopping-catalog-photo-delete",
		Summary:     "Remove a catalogue entry's photo",
		Method:      http.MethodDelete, Path: "/shopping/catalog/{id}/photo", Tags: tags,
	}, shoppingCatalogPhotoDelete)
}

func shoppingCatalogList(ctx context.Context, in *struct{ ListParams }) (*shoppingCatalogListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &shopping.ShoppingCatalogItem{}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	entries, ok := result.([]*shopping.ShoppingCatalogItem)
	if !ok {
		return nil, fmt.Errorf("shopping catalog ReadAll returned unexpected type %T", result)
	}
	return &shoppingCatalogListBody{Body: NewPaginated(entries, total, in.Page, in.PerPage)}, nil
}

func shoppingCatalogRead(ctx context.Context, in *struct {
	ID int64 `path:"id"`
	conditional.Params
}) (*singleReadBody[shoppingCatalogReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	entry := &shopping.ShoppingCatalogItem{ID: in.ID}
	maxPermission, err := handler.DoReadOne(ctx, entry, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	body := &shoppingCatalogReadBody{ShoppingCatalogItem: *entry, MaxPermission: models.Permission(maxPermission)}
	return conditionalReadResponse(&in.Params, body, entry.Updated, maxPermission)
}

func shoppingCatalogCreate(ctx context.Context, in *struct {
	Body shopping.ShoppingCatalogItem
}) (*singleBody[shopping.ShoppingCatalogItem], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[shopping.ShoppingCatalogItem]{Body: &in.Body}, nil
}

func shoppingCatalogUpdate(ctx context.Context, in *struct {
	ID   int64 `path:"id"`
	Body shoppingCatalogReadBody
}) (*singleBody[shopping.ShoppingCatalogItem], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	entry := &in.Body.ShoppingCatalogItem
	entry.ID = in.ID
	if err := handler.DoUpdate(ctx, entry, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[shopping.ShoppingCatalogItem]{Body: entry}, nil
}

func shoppingCatalogDelete(ctx context.Context, in *struct {
	ID int64 `path:"id"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &shopping.ShoppingCatalogItem{ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}

// The photo actions aren't CRUD, so they check permissions and manage the
// session themselves (see the api-v2-routes skill).

func shoppingCatalogPhotoUpload(ctx context.Context, in *shoppingPhotoUploadInput) (*singleBody[shopping.ShoppingCatalogItem], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()
	// Drops the "shopping changed" events queued by a failed attempt.
	defer events.CleanupPending(s)

	entry := &shopping.ShoppingCatalogItem{ID: in.ID}
	can, err := entry.CanUpdate(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if !can {
		_ = s.Rollback()
		return nil, huma.Error403Forbidden("forbidden")
	}

	src := in.RawBody.Data().Photo
	defer func() { _ = src.Close() }()
	if err := entry.SetPhoto(ctx, s, a, src); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	events.DispatchPending(ctx, s)
	return &singleBody[shopping.ShoppingCatalogItem]{Body: entry}, nil
}

func shoppingCatalogPhotoGet(ctx context.Context, in *struct {
	ID int64 `path:"id"`
}) (*shoppingPhotoResponse, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()

	entry := &shopping.ShoppingCatalogItem{ID: in.ID}
	if _, _, err := entry.CanRead(s, a); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	data, err := entry.Photo(s)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err) // ErrNoPhoto becomes a 404
	}
	_ = s.Commit()
	// Private: the photo needs a login. Short max-age because it can be replaced.
	return &shoppingPhotoResponse{ContentType: "image/jpeg", CacheControl: "private, max-age=300", Body: data}, nil
}

func shoppingCatalogPhotoDelete(ctx context.Context, in *struct {
	ID int64 `path:"id"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()
	// Drops the "shopping changed" events queued by a failed attempt.
	defer events.CleanupPending(s)

	entry := &shopping.ShoppingCatalogItem{ID: in.ID}
	can, err := entry.CanUpdate(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if !can {
		_ = s.Rollback()
		return nil, huma.Error403Forbidden("forbidden")
	}
	if err := entry.DeletePhoto(s); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	events.DispatchPending(ctx, s)
	return &emptyBody{}, nil
}

// ---- Sync ----

// shoppingSyncInput is what a phone sends: its cursor and its queued changes.
type shoppingSyncInput struct {
	Body struct {
		Cursor int64                     `json:"cursor" minimum:"0" doc:"The cursor from the last sync; 0 for a full copy."`
		Ops    []shopping.ShoppingSyncOp `json:"ops" maxItems:"500" doc:"Changes made on the device since the last sync, oldest first. Send at most 500 at a time."`
	}
}

type shoppingSyncBody struct {
	Body *shopping.ShoppingSyncResult
}

func registerShoppingSyncRoutes(api huma.API) {
	Register(api, huma.Operation{
		OperationID: "shopping-sync",
		Summary:     "Sync shopping data with a device",
		Description: "Applies the device's queued changes in order, then returns everything that changed since its cursor (all data when the cursor is 0). " +
			"Per item field the latest change wins; crossing edits are recorded in the conflict log. A change that can't be applied is listed in rejected and skipped; the rest still apply. " +
			"Safe to repeat: send the same changes again if the answer was lost." + shoppingAccess,
		Method: http.MethodPost, Path: "/shopping/sync", Tags: []string{"shopping"},
		DefaultStatus: http.StatusOK, // an action, not a create
	}, shoppingSync)
}

func shoppingSync(ctx context.Context, in *shoppingSyncInput) (*shoppingSyncBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	// One transaction: the changes apply together, and the answer is a
	// consistent snapshot that includes them.
	s := db.NewSession()
	defer s.Close()
	defer events.CleanupPending(s)

	rejected, err := shopping.ApplySyncOps(s, a, in.Body.Ops)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	result, err := shopping.PullChanges(s, a, in.Body.Cursor)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	result.Rejected = rejected
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	events.DispatchPending(ctx, s)
	return &shoppingSyncBody{Body: result}, nil
}
