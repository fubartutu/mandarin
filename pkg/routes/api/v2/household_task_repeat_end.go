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

// Mandarin: /api/v2 routes for a repeating task's end (an end date and/or a
// number of times). A new file; the model is pkg/models/household_task_repeat_end.go.
// Not CRUDable (one row per task, no list), so the handlers check the task's
// permissions themselves.

import (
	"context"
	"net/http"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"

	"github.com/danielgtaylor/huma/v2"
)

func init() { AddRouteRegistrar(RegisterHouseholdTaskRepeatEndRoutes) }

// RegisterHouseholdTaskRepeatEndRoutes wires a task's repeat end onto the Huma API.
func RegisterHouseholdTaskRepeatEndRoutes(api huma.API) {
	tags := []string{"tasks"}

	Register(api, huma.Operation{
		OperationID: "tasks-repeat-end-read",
		Summary:     "Get a task's repeat end",
		Description: "When a repeating task stops repeating. A task without one gets an empty end " +
			"(until null, repeats_left 0) rather than a 404. Needs read access to the task.",
		Method: http.MethodGet,
		Path:   "/tasks/{task}/repeat-end",
		Tags:   tags,
	}, tasksRepeatEndRead)
	Register(api, huma.Operation{
		OperationID: "tasks-repeat-end-update",
		Summary:     "Set a task's repeat end",
		Description: "Replaces the task's repeat end; an empty one removes it. It only has an effect while the " +
			"task repeats. Completing the last occurrence clears the task's repeat and this end, " +
			"so the task stays done. Needs write access to the task.",
		Method: http.MethodPut,
		Path:   "/tasks/{task}/repeat-end",
		Tags:   tags,
	}, tasksRepeatEndUpdate)
}

func tasksRepeatEndRead(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"The numeric id of the task."`
}) (*singleBody[models.TaskRepeatEnd], error) {
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
	re, err := models.GetTaskRepeatEnd(s, in.TaskID)
	if err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.TaskRepeatEnd]{Body: re}, nil
}

func tasksRepeatEndUpdate(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"The numeric id of the task."`
	Body   models.TaskRepeatEnd
}) (*singleBody[models.TaskRepeatEnd], error) {
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
	if err := models.SetTaskRepeatEnd(s, re); err != nil {
		_ = s.Rollback()
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		return nil, translateDomainError(err)
	}
	return &singleBody[models.TaskRepeatEnd]{Body: re}, nil
}
