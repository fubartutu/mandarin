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

// Mandarin: /api/v2 routes for the household menu. A new file, like
// household_recipes.go. Permissions live on the model in pkg/household/menu.

import (
	"context"
	"fmt"
	"net/http"

	"code.vikunja.io/api/pkg/household/menu"
	"code.vikunja.io/api/pkg/web/handler"

	"github.com/danielgtaylor/huma/v2"
)

func init() { AddRouteRegistrar(RegisterHouseholdMenuRoutes) }

const menuAccess = " There is one menu for the whole household: every user on the server can read and change all of it; link shares cannot."

type menuListBody struct {
	Body Paginated[*menu.MenuEntry]
}

// RegisterHouseholdMenuRoutes wires the menu's entries onto the Huma API.
func RegisterHouseholdMenuRoutes(api huma.API) {
	tags := []string{"menu"}

	Register(api, huma.Operation{
		OperationID: "menu-list",
		Summary:     "List menu entries",
		Description: "The entries planned from one day to another (both included, at most 62 days), by day, then meal (breakfast, lunch, dinner), then position." + menuAccess,
		Method:      http.MethodGet, Path: "/menu", Tags: tags,
	}, menuList)
	Register(api, huma.Operation{
		OperationID: "menu-create",
		Summary:     "Add a menu entry",
		Description: "Added last in its day's meal." + menuAccess,
		Method:      http.MethodPost, Path: "/menu", Tags: tags,
	}, menuCreate)
	Register(api, huma.Operation{
		OperationID: "menu-update",
		Summary:     "Change a menu entry",
		Description: "Changes only the fields sent: servings on a recipe entry, the text of a note entry, or the day: a moved entry goes last in the same meal on its new day. The meal stays as it is." + menuAccess,
		Method:      http.MethodPut, Path: "/menu/{entry}", Tags: tags,
	}, menuUpdate)
	Register(api, huma.Operation{
		OperationID: "menu-delete",
		Summary:     "Delete a menu entry",
		Method:      http.MethodDelete, Path: "/menu/{entry}", Tags: tags,
	}, menuDelete)
}

func menuList(ctx context.Context, in *struct {
	ListParams
	From string `query:"from" required:"true" format:"date" doc:"The first day, YYYY-MM-DD."`
	To   string `query:"to" required:"true" format:"date" doc:"The last day, YYYY-MM-DD."`
}) (*menuListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &menu.MenuEntry{From: in.From, To: in.To}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	list, ok := result.([]*menu.MenuEntry)
	if !ok {
		return nil, fmt.Errorf("menu ReadAll returned unexpected type %T", result)
	}
	return &menuListBody{Body: NewPaginated(list, total, in.Page, in.PerPage)}, nil
}

func menuCreate(ctx context.Context, in *struct {
	Body menu.MenuEntry
}) (*singleBody[menu.MenuEntry], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[menu.MenuEntry]{Body: &in.Body}, nil
}

func menuUpdate(ctx context.Context, in *struct {
	ID   int64 `path:"entry"`
	Body menu.EntryChanges
}) (*singleBody[menu.MenuEntry], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	// CanUpdate loads the stored entry; Update applies the changes to it.
	entry := &menu.MenuEntry{ID: in.ID, Changes: in.Body}
	if err := handler.DoUpdate(ctx, entry, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[menu.MenuEntry]{Body: entry}, nil
}

func menuDelete(ctx context.Context, in *struct {
	ID int64 `path:"entry"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &menu.MenuEntry{ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}
