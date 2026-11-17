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

package models

// Mandarin: breaking a task down, for tasks too big to start. "steps" makes
// each item a subtask in the same project; "chunks" adds them as a checklist
// at the end of the description. A new file in this package because it uses
// Vikunja's unexported task helpers. The suggestions come from the AI
// (pkg/household/breakdown); this only saves what the person kept.

import (
	"html"
	"net/http"
	"strings"

	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// The kinds of breakdown.
const (
	BreakdownSteps  = "steps"
	BreakdownChunks = "chunks"
)

// MaxBreakdownItems is how many items one breakdown may add.
const MaxBreakdownItems = 15

// maxBreakdownItemLength keeps an item to a task title's sensible length.
const maxBreakdownItemLength = 250

// BreakdownStep is one suggested step or chunk.
type BreakdownStep struct {
	Title   string `json:"title" doc:"What to do, verb first."`
	Minutes int    `json:"minutes" doc:"A rough estimate of how long it takes; 0 when unknown."`
}

// BreakdownSuggestion is the AI's breakdown, for the person to edit before saving.
type BreakdownSuggestion struct {
	Steps []BreakdownStep `json:"steps"`
}

// CleanBreakdownItems trims items, joins runs of spaces, drops empty ones and
// cuts overlong ones.
func CleanBreakdownItems(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		// strings.Fields splits on any run of white space, newlines included.
		text := strings.Join(strings.Fields(item), " ")
		if text == "" {
			continue
		}
		if r := []rune(text); len(r) > maxBreakdownItemLength {
			text = string(r[:maxBreakdownItemLength])
		}
		out = append(out, text)
	}
	return out
}

// breakdownChecklist is items as a TipTap task list, the HTML Vikunja's
// editor writes for a checklist (so cards count "0 of 5").
func breakdownChecklist(items []string) string {
	var b strings.Builder
	b.WriteString(`<ul data-type="taskList">`)
	for _, item := range items {
		b.WriteString(`<li data-type="taskItem" data-checked="false"><p>`)
		b.WriteString(html.EscapeString(item))
		b.WriteString(`</p></li>`)
	}
	b.WriteString(`</ul>`)
	return b.String()
}

// BreakdownTask saves a breakdown of the task and returns the task as the
// API shows it (with its new subtasks in related_tasks). The caller checks
// write access to the task, which also allows creating tasks in its project.
func BreakdownTask(s *xorm.Session, a web.Auth, taskID int64, mode string, items []string) (*Task, error) {
	items = CleanBreakdownItems(items)
	if len(items) == 0 {
		return nil, ErrInvalidBreakdown{Reason: "Add at least one step."}
	}
	if len(items) > MaxBreakdownItems {
		return nil, ErrInvalidBreakdown{Reason: "That's too many steps for one go: keep it to 15 or fewer."}
	}
	parent, err := GetTaskSimple(s, &Task{ID: taskID})
	if err != nil {
		return nil, err
	}

	switch mode {
	case BreakdownSteps:
		for _, title := range items {
			child := &Task{Title: title, ProjectID: parent.ProjectID}
			// No assignees to copy; put it in the project's default bucket.
			if err := createTask(s, child, a, false, true); err != nil {
				return nil, err
			}
			// Creates the parenttask relation the other way round too.
			rel := &TaskRelation{TaskID: parent.ID, OtherTaskID: child.ID, RelationKind: RelationKindSubtask}
			if err := rel.Create(s, a); err != nil {
				return nil, err
			}
		}
	case BreakdownChunks:
		update := &Task{ID: taskID, Description: parent.Description + breakdownChecklist(items)}
		if err := update.updateSingleTask(s, a, []string{"description"}); err != nil {
			return nil, err
		}
	default:
		return nil, ErrInvalidBreakdown{Reason: "mode must be steps or chunks."}
	}

	task := &Task{ID: taskID}
	if err := task.ReadOne(s, a); err != nil {
		return nil, err
	}
	return task, nil
}

// ErrInvalidBreakdown is returned when a breakdown can't be saved as sent.
type ErrInvalidBreakdown struct {
	Reason string
}

func (err ErrInvalidBreakdown) Error() string {
	return "Invalid breakdown: " + err.Reason
}

// ErrCodeInvalidBreakdown holds the unique world-error code of this error.
// Task breakdown codes are 905xx.
const ErrCodeInvalidBreakdown = 90501

// HTTPError holds the http error description
func (err ErrInvalidBreakdown) HTTPError() web.HTTPError {
	return web.HTTPError{HTTPCode: http.StatusBadRequest, Code: ErrCodeInvalidBreakdown, Message: err.Reason}
}
