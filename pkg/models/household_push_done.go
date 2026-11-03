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

// Mandarin (new file, so upstream merges don't conflict): marks a task done
// for the "Done" button on push notifications (pkg/household/push). It's here
// because only this package can update a single column of a task.

import (
	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// MarkTaskDone sets only the task's done flag, as ticking it off in the app
// does: a repeating task moves to its next occurrence (with Mandarin's repeat
// rules and ends) and stays open. The caller checks permissions, and sends the
// queued events after committing.
func MarkTaskDone(s *xorm.Session, a web.Auth, taskID int64) error {
	t := &Task{ID: taskID, Done: true}
	return t.updateSingleTask(s, a, []string{"done"})
}
