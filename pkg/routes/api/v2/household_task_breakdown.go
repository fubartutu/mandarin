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

// Mandarin: /api/v2 routes for breaking a task down into steps (subtasks) or
// small chunks (a checklist). A new file; the AI call is pkg/household/breakdown,
// saving is models.BreakdownTask. Custom actions, so the handlers check the
// task's permissions themselves.

import (
	"context"
	"net/http"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/events"
	"code.vikunja.io/api/pkg/household/breakdown"
	"code.vikunja.io/api/pkg/models"

	"github.com/danielgtaylor/huma/v2"
)

func init() { AddRouteRegistrar(RegisterHouseholdTaskBreakdownRoutes) }

// RegisterHouseholdTaskBreakdownRoutes wires task breakdown onto the Huma API.
func RegisterHouseholdTaskBreakdownRoutes(api huma.API) {
	tags := []string{"tasks"}

	Register(api, huma.Operation{
		OperationID: "tasks-breakdown-suggest",
		Summary:     "Suggest a breakdown of a task",
		Description: "Asks the AI (in the mandarin-scraper container) how to break the task down. Nothing is saved: " +
			"the answer is for the person to edit and then send to POST /tasks/{task}/breakdown. The AI is told the " +
			"task's notes, project and, for a subtask, its parent and the parent's other subtasks. Can take a while " +
			"when the AI has to start up. 503 when the AI isn't set up or can't be reached. Needs read access to the task.",
		Method:        http.MethodPost,
		Path:          "/tasks/{task}/breakdown/suggest",
		DefaultStatus: http.StatusOK, // nothing is created
		Tags:          tags,
	}, tasksBreakdownSuggest)
	Register(api, huma.Operation{
		OperationID: "tasks-breakdown",
		Summary:     "Break a task down",
		Description: "Saves a breakdown. mode \"steps\" creates each step as a subtask in the task's project, in " +
			"order; \"chunks\" adds them as a checklist at the end of the task's description. At most 15 items; " +
			"empty ones are dropped. Returns the task with its new subtasks or description. Needs write access to the task.",
		Method:        http.MethodPost,
		Path:          "/tasks/{task}/breakdown",
		DefaultStatus: http.StatusOK, // returns the existing task, changed
		Tags:          tags,
	}, tasksBreakdown)
}

type breakdownSuggestBody struct {
	Body models.BreakdownSuggestion
}

func tasksBreakdownSuggest(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"The numeric id of the task."`
	Body   struct {
		Mode string `json:"mode" enum:"steps,chunks" doc:"steps: the main stages of a big job; chunks: small, easy-to-start actions for one stage."`
	}
}) (*breakdownSuggestBody, error) {
	a, err := authFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	// Gather what the AI needs, then close the session before the (slow) AI call.
	req, err := func() (*breakdown.Request, error) {
		s := db.NewReadSession()
		defer s.Close()
		can, _, err := (&models.Task{ID: in.TaskID}).CanRead(s, a)
		if err != nil {
			return nil, err
		}
		if !can {
			return nil, huma.Error403Forbidden("forbidden")
		}
		return breakdown.RequestFor(s, a, in.TaskID, in.Body.Mode)
	}()
	if err != nil {
		return nil, translateDomainError(err)
	}

	suggestion, err := breakdown.Suggest(ctx, req)
	if err != nil {
		return nil, translateDomainError(err)
	}
	return &breakdownSuggestBody{Body: *suggestion}, nil
}

func tasksBreakdown(ctx context.Context, in *struct {
	TaskID int64 `path:"task" doc:"The numeric id of the task."`
	Body   struct {
		Mode  string   `json:"mode" enum:"steps,chunks" doc:"steps: each item becomes a subtask; chunks: the items become a checklist in the description."`
		Items []string `json:"items" maxItems:"15" doc:"The steps or chunks, in order."`
	}
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
	task, err := models.BreakdownTask(s, a, in.TaskID, in.Body.Mode, in.Body.Items)
	if err != nil {
		_ = s.Rollback()
		events.CleanupPending(s)
		return nil, translateDomainError(err)
	}
	if err := s.Commit(); err != nil {
		events.CleanupPending(s)
		return nil, translateDomainError(err)
	}
	// Task creates and updates queued their events on the session; send them
	// now it's committed, so open apps refresh.
	events.DispatchPending(ctx, s)
	return &singleBody[models.Task]{Body: task}, nil
}
