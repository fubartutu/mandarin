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

package habits

// Offline ticking: a phone queues ticks and skips while it has no signal and
// sends them here when it does (frontend household/habits/sync). The answer
// is the whole picture again (habits with progress, and routines): a
// household's habits are few, so there's no change feed like shopping's.

import (
	"errors"
	"fmt"

	"code.vikunja.io/api/pkg/web"

	"xorm.io/xorm"
)

// Sync op types.
const (
	OpTick   = "tick"
	OpUntick = "untick"
	OpSkip   = "skip"
	OpUnskip = "unskip"
)

// SyncOp is one change made on a phone, possibly offline.
type SyncOp struct {
	ID      string `json:"id" minLength:"1" maxLength:"36" doc:"The change's own id, made by the phone; rejected changes are listed by it."`
	Type    string `json:"type" enum:"tick,untick,skip,unskip" doc:"tick and untick add or take off one tick (by uid); skip and unskip a day."`
	HabitID int64  `json:"habit_id" doc:"The habit changed."`
	UID     string `json:"uid,omitempty" maxLength:"36" doc:"tick and untick: the tick's uid, made by the phone."`
	Day     string `json:"day,omitempty" maxLength:"10" doc:"tick, skip and unskip: the day, YYYY-MM-DD, on the phone's calendar."`
}

// RejectedOp is a change that couldn't be applied, e.g. its habit was deleted.
type RejectedOp struct {
	OpID   string `json:"op_id" doc:"The id of the change."`
	Reason string `json:"reason" doc:"Why, in words for people."`
}

// SyncResult is everything the phone shows on the Habits page.
type SyncResult struct {
	Habits   []*Habit      `json:"habits" doc:"The caller's habits and the household routines that aren't archived, with progress as of today."`
	Groups   []*HabitGroup `json:"groups" doc:"The caller's routines and the shared ones."`
	Rejected []*RejectedOp `json:"rejected" doc:"Changes that weren't applied; the others were."`
}

// httpError is what the package's (and Vikunja's) domain errors have: an
// answer for people. A change failing with one is rejected, not fatal.
type httpError interface {
	HTTPError() web.HTTPError
}

// Sync applies the ops in order, then returns the caller's habits and
// routines. An op that fails for a reason people can act on (no such habit,
// a day in the future) is listed in Rejected and the rest still apply.
// Sending the same ops again changes nothing: ticks are kept by uid, a skip
// is per day, and taking off a tick that's gone is fine.
func Sync(s *xorm.Session, a web.Auth, ops []*SyncOp) (*SyncResult, error) {
	out := &SyncResult{Habits: []*Habit{}, Groups: []*HabitGroup{}, Rejected: []*RejectedOp{}}
	for _, op := range ops {
		err := applyOp(s, a, op)
		var he httpError
		switch {
		case err == nil:
		case errors.As(err, &he):
			out.Rejected = append(out.Rejected, &RejectedOp{OpID: op.ID, Reason: he.HTTPError().Message})
		default:
			return nil, err
		}
	}

	habits, _, _, err := (&Habit{}).ReadAll(s, a, "", -1, 0)
	if err != nil {
		return nil, err
	}
	groups, _, _, err := (&HabitGroup{}).ReadAll(s, a, "", -1, 0)
	if err != nil {
		return nil, err
	}
	var ok bool
	if out.Habits, ok = habits.([]*Habit); !ok {
		return nil, fmt.Errorf("habits ReadAll returned %T", habits)
	}
	if out.Groups, ok = groups.([]*HabitGroup); !ok {
		return nil, fmt.Errorf("habit groups ReadAll returned %T", groups)
	}
	return out, nil
}

func applyOp(s *xorm.Session, a web.Auth, op *SyncOp) error {
	var err error
	switch op.Type {
	case OpTick:
		_, err = AddEntry(s, a, op.HabitID, op.UID, op.Day)
	case OpUntick:
		_, err = RemoveEntry(s, a, op.HabitID, op.UID)
	case OpSkip:
		_, err = SetSkip(s, a, op.HabitID, op.Day, "")
	case OpUnskip:
		_, err = RemoveSkip(s, a, op.HabitID, op.Day)
	default:
		err = ErrInvalidHabit{Reason: "Unknown change type."}
	}
	return err
}
