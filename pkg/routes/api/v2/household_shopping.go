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
