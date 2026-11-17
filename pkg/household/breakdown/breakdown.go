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

// Package breakdown asks the AI how to break a task down, for tasks too big
// to start. The AI lives in the scraper container (household-scraper/
// breakdown.py); saving what the person keeps is models.BreakdownTask.
package breakdown

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"code.vikunja.io/api/pkg/household/recipes"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// The AI on sampc may have to load its model first.
const suggestTimeout = 3 * time.Minute

// Request is what the scraper's POST /breakdown takes.
type Request struct {
	Mode        string   `json:"mode"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Project     string   `json:"project"`
	Parent      string   `json:"parent"`
	Siblings    []string `json:"siblings"`
}

// RequestFor gathers what the AI needs to know about the task: its title and
// notes, its project and, for a subtask, its parent and the parent's other
// subtasks (so chunks stay within this stage). The caller checks read access.
func RequestFor(s *xorm.Session, a web.Auth, taskID int64, mode string) (*Request, error) {
	task := &models.Task{ID: taskID}
	if err := task.ReadOne(s, a); err != nil {
		return nil, err
	}
	req := &Request{Mode: mode, Title: task.Title, Description: task.Description, Siblings: []string{}}
	if project, err := models.GetProjectSimpleByID(s, task.ProjectID); err == nil {
		req.Project = project.Title
	}
	parents := task.RelatedTasks[models.RelationKindParenttask]
	if len(parents) > 0 {
		req.Parent = parents[0].Title
		parent := &models.Task{ID: parents[0].ID}
		// A parent the person can't read just gives no siblings.
		if can, _, err := parent.CanRead(s, a); err == nil && can {
			if err := parent.ReadOne(s, a); err == nil {
				for _, sibling := range parent.RelatedTasks[models.RelationKindSubtask] {
					req.Siblings = append(req.Siblings, sibling.Title)
				}
			}
		}
	}
	return req, nil
}

// Suggest asks the scraper's AI. Call it with no database session open: it
// can take minutes, and SQLite allows only one writer.
func Suggest(ctx context.Context, req *Request) (*models.BreakdownSuggestion, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, suggestTimeout)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, recipes.ScraperURL()+"/breakdown", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, ErrAIUnavailable{}
	}
	defer resp.Body.Close()
	answer, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, ErrAIUnavailable{}
	}
	if resp.StatusCode != http.StatusOK {
		// FastAPI puts the reason in "detail" (a list of problems for a bad request).
		var problem struct {
			Detail any `json:"detail"`
		}
		_ = json.Unmarshal(answer, &problem)
		reason, _ := problem.Detail.(string)
		if reason == "" {
			reason = fmt.Sprintf("The AI answered %d.", resp.StatusCode)
		}
		return nil, ErrSuggestFailed{Reason: reason, Unavailable: resp.StatusCode == http.StatusServiceUnavailable}
	}
	out := &models.BreakdownSuggestion{}
	if err := json.Unmarshal(answer, out); err != nil {
		return nil, ErrSuggestFailed{Reason: "The AI's answer couldn't be read."}
	}
	if out.Steps == nil {
		out.Steps = []models.BreakdownStep{}
	}
	return out, nil
}
