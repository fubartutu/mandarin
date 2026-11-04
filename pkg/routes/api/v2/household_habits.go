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

// Mandarin: /api/v2 routes for habits and routines. A new file, like
// household_menu.go. Habits and groups are CRUD models; ticks, skips and the
// history are actions in pkg/household/habits that check the caller may
// see the habit themselves.

import (
	"context"
	"fmt"
	"net/http"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/household/habits"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/web/handler"

	"github.com/danielgtaylor/huma/v2"
	"xorm.io/xorm"
)

func init() { AddRouteRegistrar(RegisterHouseholdHabitRoutes) }

const habitAccess = " Callers see their own habits and every household routine (shared habit); " +
	"others' personal habits answer 404. Link shares can't use habits."

type habitListBody struct {
	Body Paginated[*habits.Habit]
}

type habitReadBody struct {
	habits.Habit
	MaxPermission models.Permission `json:"max_permission" readOnly:"true" doc:"The caller's permission on this habit; always 2 (admin) for one they can see."`
}

type habitGroupListBody struct {
	Body Paginated[*habits.HabitGroup]
}

// RegisterHouseholdHabitRoutes wires habits, routines, ticks and skips onto
// the Huma API.
func RegisterHouseholdHabitRoutes(api huma.API) {
	tags := []string{"habits"}

	Register(api, huma.Operation{
		OperationID: "habits-list",
		Summary:     "List habits",
		Description: "The caller's habits and the household routines, each with its progress as of today " +
			"(in the caller's time zone), by group then position. All on one page." + habitAccess,
		Method: http.MethodGet, Path: "/habits", Tags: tags,
	}, habitsList)
	Register(api, huma.Operation{
		OperationID: "habits-read",
		Summary:     "Get a habit",
		Description: "The habit with its progress as of today." + habitAccess,
		Method:      http.MethodGet, Path: "/habits/{habit}", Tags: tags,
	}, habitsRead)
	Register(api, huma.Operation{
		OperationID: "habits-create",
		Summary:     "Make a habit",
		Description: "The caller becomes its owner. It goes last in its group." + habitAccess,
		Method:      http.MethodPost, Path: "/habits", Tags: tags,
	}, habitsCreate)
	Register(api, huma.Operation{
		OperationID: "habits-update",
		Summary:     "Change a habit",
		Description: "Replaces the editable fields (send them all). The kind can't change. " +
			"Setting paused starts or ends a pause from today." + habitAccess,
		Method: http.MethodPut, Path: "/habits/{habit}", Tags: tags,
	}, habitsUpdate)
	Register(api, huma.Operation{
		OperationID: "habits-delete",
		Summary:     "Delete a habit",
		Description: "Deletes it with all its ticks, skips and pauses. Archive it instead to keep the history." + habitAccess,
		Method:      http.MethodDelete, Path: "/habits/{habit}", Tags: tags,
	}, habitsDelete)

	Register(api, huma.Operation{
		OperationID: "habits-history",
		Summary:     "Get a habit's history",
		Description: "The habit's grid from one day to another (at most 400 days; days after today are left out), " +
			"with the ticks and skips in that range." + habitAccess,
		Method: http.MethodGet, Path: "/habits/{habit}/history", Tags: tags,
	}, habitsHistory)
	Register(api, huma.Operation{
		OperationID: "habits-entries-create",
		Summary:     "Tick a habit",
		Description: "Adds one tick on a day (today or earlier). Sending a uid again changes nothing, " +
			"so a tick can be retried safely. Returns the habit with its new progress." + habitAccess,
		Method: http.MethodPost, Path: "/habits/{habit}/entries", Tags: tags,
	}, habitsEntryCreate)
	Register(api, huma.Operation{
		OperationID: "habits-entries-delete",
		Summary:     "Take a tick off",
		Description: "Deletes one tick by its uid. A uid that is already gone is not an error." + habitAccess,
		Method:      http.MethodDelete, Path: "/habits/{habit}/entries/{uid}", Tags: tags,
	}, habitsEntryDelete)
	Register(api, huma.Operation{
		OperationID: "habits-skips-update",
		Summary:     "Skip a day",
		Description: "Skips the day (for a weekly habit, the week the day is in), so it counts as neither " +
			"done nor missed. Skipping again only changes the reason. Bad habits can't be skipped. " +
			"Returns the habit with its new progress." + habitAccess,
		Method: http.MethodPut, Path: "/habits/{habit}/skips/{day}", Tags: tags,
	}, habitsSkipSet)
	Register(api, huma.Operation{
		OperationID: "habits-skips-delete",
		Summary:     "Un-skip a day",
		Description: "For a weekly habit, un-skips the week the day is in." + habitAccess,
		Method:      http.MethodDelete, Path: "/habits/{habit}/skips/{day}", Tags: tags,
	}, habitsSkipDelete)

	Register(api, huma.Operation{
		OperationID: "habits-sync",
		Summary:     "Sync ticks and skips from a phone",
		Description: "Applies the phone's queued ticks and skips in order (they may have been made offline), " +
			"then returns the caller's habits (not archived, with progress) and routines. A change that can't be " +
			"applied is listed in rejected and skipped; the rest still apply. Safe to repeat: ticks are kept by uid." + habitAccess,
		Method: http.MethodPost, Path: "/habits/sync", Tags: tags,
		DefaultStatus: http.StatusOK, // an action, not a create
	}, habitsSync)

	groupTags := []string{"habits"}
	Register(api, huma.Operation{
		OperationID: "habit-groups-list",
		Summary:     "List routines",
		Description: "The caller's habit groups and the shared ones, by position." + habitAccess,
		Method:      http.MethodGet, Path: "/habit-groups", Tags: groupTags,
	}, habitGroupsList)
	Register(api, huma.Operation{
		OperationID: "habit-groups-create",
		Summary:     "Make a routine",
		Description: "A named group of habits, e.g. Morning. It goes after the other groups.",
		Method:      http.MethodPost, Path: "/habit-groups", Tags: groupTags,
	}, habitGroupsCreate)
	Register(api, huma.Operation{
		OperationID: "habit-groups-update",
		Summary:     "Change a routine",
		Description: "Replaces the title, position and sharing. Only the owner can make a shared group personal again.",
		Method:      http.MethodPut, Path: "/habit-groups/{group}", Tags: groupTags,
	}, habitGroupsUpdate)
	Register(api, huma.Operation{
		OperationID: "habit-groups-delete",
		Summary:     "Delete a routine",
		Description: "Its habits stay, without a group.",
		Method:      http.MethodDelete, Path: "/habit-groups/{group}", Tags: groupTags,
	}, habitGroupsDelete)
}

func habitsList(ctx context.Context, in *struct {
	ListParams
	Archived bool `query:"archived" doc:"Include archived habits."`
}) (*habitListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &habits.Habit{ShowArchived: in.Archived}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	list, ok := result.([]*habits.Habit)
	if !ok {
		return nil, fmt.Errorf("habits ReadAll returned unexpected type %T", result)
	}
	return &habitListBody{Body: NewPaginated(list, total, in.Page, in.PerPage)}, nil
}

func habitsRead(ctx context.Context, in *struct {
	ID int64 `path:"habit"`
}) (*singleBody[habitReadBody], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	h := &habits.Habit{ID: in.ID}
	maxPermission, err := handler.DoReadOne(ctx, h, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	// No ETag, unlike most reads: the progress changes with every tick and
	// at midnight, neither of which touches Updated.
	return &singleBody[habitReadBody]{Body: &habitReadBody{Habit: *h, MaxPermission: models.Permission(maxPermission)}}, nil
}

func habitsCreate(ctx context.Context, in *struct {
	Body habits.Habit
}) (*singleBody[habits.Habit], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[habits.Habit]{Body: &in.Body}, nil
}

func habitsUpdate(ctx context.Context, in *struct {
	ID   int64 `path:"habit"`
	Body habitReadBody
}) (*singleBody[habits.Habit], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	h := &in.Body.Habit
	h.ID = in.ID // URL wins over body
	if err := handler.DoUpdate(ctx, h, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[habits.Habit]{Body: h}, nil
}

func habitsDelete(ctx context.Context, in *struct {
	ID int64 `path:"habit"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &habits.Habit{ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}

// inHabitSession runs one habit action in its own transaction. The actions
// check the caller may see the habit (there is no generic handler here).
func inHabitSession[T any](do func(s *xorm.Session) (*T, error)) (*T, error) {
	s := db.NewSession()
	defer s.Close()
	out, err := do(s)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return out, nil
}

func habitsHistory(ctx context.Context, in *struct {
	ID   int64  `path:"habit"`
	From string `query:"from" required:"true" format:"date" doc:"The first day, YYYY-MM-DD."`
	To   string `query:"to" required:"true" format:"date" doc:"The last day, YYYY-MM-DD."`
}) (*singleBody[habits.History], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	hist, err := inHabitSession(func(s *xorm.Session) (*habits.History, error) {
		return habits.GetHistory(s, a, in.ID, in.From, in.To)
	})
	if err != nil {
		return nil, err
	}
	return &singleBody[habits.History]{Body: hist}, nil
}

func habitsEntryCreate(ctx context.Context, in *struct {
	ID   int64 `path:"habit"`
	Body struct {
		UID string `json:"uid" format:"uuid" doc:"A new UUID made by the app for this tick."`
		Day string `json:"day" format:"date" doc:"The day it counts for, YYYY-MM-DD: today or earlier."`
	}
}) (*singleBody[habits.Habit], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	h, err := inHabitSession(func(s *xorm.Session) (*habits.Habit, error) {
		return habits.AddEntry(s, a, in.ID, in.Body.UID, in.Body.Day)
	})
	if err != nil {
		return nil, err
	}
	return &singleBody[habits.Habit]{Body: h}, nil
}

func habitsEntryDelete(ctx context.Context, in *struct {
	ID  int64  `path:"habit"`
	UID string `path:"uid"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := inHabitSession(func(s *xorm.Session) (*habits.Habit, error) {
		return habits.RemoveEntry(s, a, in.ID, in.UID)
	}); err != nil {
		return nil, err
	}
	return &emptyBody{}, nil
}

func habitsSkipSet(ctx context.Context, in *struct {
	ID   int64  `path:"habit"`
	Day  string `path:"day" format:"date" doc:"The day to skip, YYYY-MM-DD: today or earlier."`
	Body struct {
		Reason string `json:"reason" maxLength:"250" doc:"Why, optional."`
	}
}) (*singleBody[habits.Habit], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	h, err := inHabitSession(func(s *xorm.Session) (*habits.Habit, error) {
		return habits.SetSkip(s, a, in.ID, in.Day, in.Body.Reason)
	})
	if err != nil {
		return nil, err
	}
	return &singleBody[habits.Habit]{Body: h}, nil
}

func habitsSkipDelete(ctx context.Context, in *struct {
	ID  int64  `path:"habit"`
	Day string `path:"day" format:"date" doc:"The skipped day, YYYY-MM-DD."`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := inHabitSession(func(s *xorm.Session) (*habits.Habit, error) {
		return habits.RemoveSkip(s, a, in.ID, in.Day)
	}); err != nil {
		return nil, err
	}
	return &emptyBody{}, nil
}

func habitGroupsList(ctx context.Context, in *ListParams) (*habitGroupListBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	result, _, total, err := handler.DoReadAll(ctx, &habits.HabitGroup{}, a, in.Q, in.Page, in.PerPage)
	if err != nil {
		return nil, translateDomainError(err)
	}
	list, ok := result.([]*habits.HabitGroup)
	if !ok {
		return nil, fmt.Errorf("habit groups ReadAll returned unexpected type %T", result)
	}
	return &habitGroupListBody{Body: NewPaginated(list, total, in.Page, in.PerPage)}, nil
}

func habitGroupsCreate(ctx context.Context, in *struct {
	Body habits.HabitGroup
}) (*singleBody[habits.HabitGroup], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoCreate(ctx, &in.Body, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[habits.HabitGroup]{Body: &in.Body}, nil
}

func habitGroupsUpdate(ctx context.Context, in *struct {
	ID   int64 `path:"group"`
	Body habits.HabitGroup
}) (*singleBody[habits.HabitGroup], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	g := &in.Body
	g.ID = in.ID // URL wins over body
	if err := handler.DoUpdate(ctx, g, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[habits.HabitGroup]{Body: g}, nil
}

func habitGroupsDelete(ctx context.Context, in *struct {
	ID int64 `path:"group"`
}) (*emptyBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if err := handler.DoDelete(ctx, &habits.HabitGroup{ID: in.ID}, a); err != nil {
		return nil, translateDomainError(err)
	}
	return &emptyBody{}, nil
}

func habitsSync(ctx context.Context, in *struct {
	Body struct {
		Ops []*habits.SyncOp `json:"ops" maxItems:"500" doc:"Queued changes, oldest first."`
	}
}) (*singleBody[habits.SyncResult], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	// One transaction: the changes apply together, and the answer includes them.
	result, err := inHabitSession(func(s *xorm.Session) (*habits.SyncResult, error) {
		return habits.Sync(s, a, in.Body.Ops)
	})
	if err != nil {
		return nil, err
	}
	return &singleBody[habits.SyncResult]{Body: result}, nil
}
