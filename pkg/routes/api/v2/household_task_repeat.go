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

// Mandarin: /api/v2 routes for a repeating task's household settings: its end
// (an end date and/or a number of times). A new file; the model is pkg/models/household_task_repeat.go.
// Not CRUDable (one row per task, no list), so the handlers check the task's
// permissions themselves.

import (
	"context"
	"net/http"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/events"
	"code.vikunja.io/api/pkg/models"

	"github.com/danielgtaylor/huma/v2"
)

func init() { AddRouteRegistrar(RegisterHouseholdTaskRepeatRoutes) }

// RegisterHouseholdTaskRepeatRoutes wires a task's repeat settings onto the Huma API.
func RegisterHouseholdTaskRepeatRoutes(api huma.API) {
	tags := []string{"tasks"}

	Register(api, huma.Operation{
		OperationID: "tasks-repeat-read",
		Summary:     "Get a task's repeat settings",
		Description: "When a repeating task stops repeating. A task without settings gets empty ones " +
			"(until null, repeats_left 0) rather than a 404. Needs read access to the task.",
		Method: http.MethodGet,
		Path:   "/tasks/{task}/repeat",
		Tags:   tags,
	}, tasksRepeatRead)
	Register(api, huma.Operation{
		OperationID: "tasks-repeat-update",
		Summary:     "Set a task's repeat settings",
		Description: "Replaces the task's repeat settings; empty ones remove them. It only has an effect while the " +
			"task repeats. Completing the last occurrence clears the task's repeat and this end, " +
			"so the task stays done. Needs write access to the task.",
		Method: http.MethodPut,
		Path:   "/tasks/{task}/repeat",
		Tags:   tags,
	}, tasksRepeatUpdate)
	Register(api, huma.Operation{
		OperationID: "tasks-skip",
		Summary:     "Skip a repeating task's occurrence",
		Description: "Moves a repeating task to its next occurrence without marking it done: dates and reminders " +
			"move as completing would move them, and the number of times left doesn't count down. When the " +
			"end date comes before the next occurrence, the series ends: the task is completed and stops " +
			"repeating. 400 for a task that doesn't repeat. Needs write access to the task.",
		Method:        http.MethodPost,
		Path:          "/tasks/{task}/skip",
		DefaultStatus: http.StatusOK, // an action, not a create
		Tags:          tags,
	}, tasksSkip)
}

func tasksRepeatRead(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"The numeric id of the task."`
}) (*singleBody[models.TaskRepeat], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewReadSession()
	defer s.Close()

	can, _, err := (&models.Task{ID: in.TaskID}).CanRead(s, a)
	if err != nil {
		return nil, translateDomainError(err)
	}
	if !can {
		return nil, huma.Error403Forbidden("forbidden")
	}
	re, err := models.GetTaskRepeat(s, in.TaskID)
	if err != nil {
		return nil, translateDomainError(err)
	}
	if err := re.FillNext(s); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.TaskRepeat]{Body: re}, nil
}

func tasksRepeatUpdate(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"The numeric id of the task."`
	Body   models.TaskRepeat
}) (*singleBody[models.TaskRepeat], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()

	can, err := (&models.Task{ID: in.TaskID}).CanUpdate(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if !can {
		_ = s.Rollback()
		return nil, huma.Error403Forbidden("forbidden")
	}
	re := &in.Body
	re.TaskID = in.TaskID // the URL wins over the body
	if err := models.SetTaskRepeat(s, re); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := re.FillNext(s); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.TaskRepeat]{Body: re}, nil
}

func tasksSkip(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"The numeric id of the task."`
}) (*singleBody[models.Task], error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	s := db.NewSession()
	defer s.Close()

	can, err := (&models.Task{ID: in.TaskID}).CanUpdate(s, a)
	if err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if !can {
		_ = s.Rollback()
		return nil, huma.Error403Forbidden("forbidden")
	}
	task, err := models.SkipTaskOccurrence(s, a, in.TaskID)
	if err != nil {
		_ = s.Rollback()
		events.CleanupPending(s)
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		events.CleanupPending(s)
		return nil, translateDomainError(err)
	}
	// Task.Update queued its TaskUpdatedEvent on the session; send it now
	// it's committed, so open apps refresh.
	events.DispatchPending(ctx, s)
	return &singleBody[models.Task]{Body: task}, nil
}
