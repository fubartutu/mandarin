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

package push

// Done and Snooze buttons on task notifications. A button tap runs in the
// phone's service worker, which can't read the app's login token, so each
// notification carries a signed action token instead: it says which user and
// task it's for, the task's due date when it was sent, and when it expires.
// POST /api/v2/push/actions takes the token and the button, no login needed.
//
// A token can't do Done twice: the server ignores it once the task is done or
// its due date has moved (a repeating task completed elsewhere), so an old
// notification can't skip a second occurrence.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/user"

	"xorm.io/xorm"
)

const (
	// ActionDone ticks the task off.
	ActionDone = "done"
	// ActionSnooze reminds the user again in SnoozeFor.
	ActionSnooze = "snooze"

	// SnoozeFor is how long Snooze waits.
	SnoozeFor = time.Hour
	// actionTokenTTL is how long a notification's buttons keep working.
	actionTokenTTL = 48 * time.Hour
)

// Action is a button on a notification (the Web Notification API's
// NotificationAction). Chrome on Android shows at most two.
type Action struct {
	Action string `json:"action"`
	Title  string `json:"title"`
}

var taskActions = []Action{
	{Action: ActionDone, Title: "Done"},
	{Action: ActionSnooze, Title: "Snooze 1h"},
}

// actionClaims is what an action token says. Short JSON keys keep the push
// payload small (push services cap it at about 4 KB).
type actionClaims struct {
	User    int64 `json:"u"`
	Task    int64 `json:"t"`
	Due     int64 `json:"d"` // unix seconds; 0 = no due date
	Expires int64 `json:"e"` // unix seconds
}

// dueUnix is a due date as stored in a token; 0 when there is none.
func dueUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

// actionKey is the signing key, derived from the server's secret so nothing
// new has to be stored. The label keeps it different from the login tokens'.
func actionKey() []byte {
	m := hmac.New(sha256.New, []byte(config.ServiceSecret.GetString()))
	m.Write([]byte("mandarin push actions"))
	return m.Sum(nil)
}

func signature(payload string) []byte {
	m := hmac.New(sha256.New, actionKey())
	m.Write([]byte(payload))
	return m.Sum(nil)
}

// NewActionToken makes the token for a notification about task, sent to userID.
// Exported for the route tests.
func NewActionToken(userID int64, task int64, due time.Time, now time.Time) string {
	raw, err := json.Marshal(actionClaims{User: userID, Task: task, Due: dueUnix(due), Expires: now.Add(actionTokenTTL).Unix()})
	if err != nil {
		return "" // can't happen with a struct of int64s; no token means no buttons
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	return payload + "." + base64.RawURLEncoding.EncodeToString(signature(payload))
}

// parseActionToken checks the signature and expiry and returns the claims.
func parseActionToken(token string, now time.Time) (*actionClaims, error) {
	payload, sig, ok := strings.Cut(token, ".")
	if !ok {
		return nil, ErrInvalidActionToken{}
	}
	got, err := base64.RawURLEncoding.DecodeString(sig)
	// hmac.Equal compares in constant time, so timing doesn't leak the signature.
	if err != nil || !hmac.Equal(got, signature(payload)) {
		return nil, ErrInvalidActionToken{}
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, ErrInvalidActionToken{}
	}
	c := &actionClaims{}
	if err := json.Unmarshal(raw, c); err != nil || c.User == 0 || c.Task == 0 {
		return nil, ErrInvalidActionToken{}
	}
	if now.Unix() > c.Expires {
		return nil, ErrInvalidActionToken{Expired: true}
	}
	return c, nil
}

// addTaskActions puts the Done and Snooze buttons on a message about one
// task, for the user it goes to. Messages about no single task stay as they are.
func addTaskActions(msg *Message, userID int64, now time.Time) {
	if msg == nil || msg.taskID == 0 {
		return
	}
	token := NewActionToken(userID, msg.taskID, msg.taskDue, now)
	if token == "" {
		return
	}
	msg.Actions = taskActions
	msg.ActionToken = token
}

// ActionInput is a button tap from a notification.
type ActionInput struct {
	Token  string `json:"token" minLength:"1" maxLength:"1000" doc:"The action_token the notification came with."`
	Action string `json:"action" enum:"done,snooze" doc:"Which button was tapped."`
}

// ActionResult says what the tap did.
type ActionResult struct {
	Result string `json:"result" enum:"done,snoozed,unchanged" doc:"done or snoozed; unchanged when the task was already done or has moved on to another date since the notification was sent."`
	// SnoozedUntil is when the reminder comes back, for snoozed.
	SnoozedUntil *time.Time `json:"snoozed_until,omitempty" doc:"When the reminder comes back (snoozed only)."`
}

// DoAction carries out a notification button for the user in the token, with
// their permissions. Done may queue task events on s: send them after commit.
func DoAction(s *xorm.Session, in *ActionInput, now time.Time) (*ActionResult, error) {
	c, err := parseActionToken(in.Token, now)
	if err != nil {
		return nil, err
	}
	u, err := user.GetUserByIDForAuth(s, c.User)
	if err != nil {
		// Deleted or disabled since the notification went out.
		return nil, ErrInvalidActionToken{}
	}
	task, err := models.GetTaskByIDSimple(s, c.Task)
	if err != nil {
		return nil, err
	}
	can, err := (&models.Task{ID: task.ID}).CanUpdate(s, u)
	if err != nil {
		return nil, err
	}
	if !can {
		return nil, models.ErrGenericForbidden{}
	}
	if task.Done || dueUnix(task.DueDate) != c.Due {
		return &ActionResult{Result: "unchanged"}, nil
	}

	switch in.Action {
	case ActionDone:
		if err := models.MarkTaskDone(s, u, task.ID); err != nil {
			return nil, err
		}
		return &ActionResult{Result: "done"}, nil
	case ActionSnooze:
		at := now.Add(SnoozeFor).Truncate(time.Minute)
		if err := snooze(s, u.ID, task.ID, at); err != nil {
			return nil, err
		}
		return &ActionResult{Result: "snoozed", SnoozedUntil: &at}, nil
	}
	return nil, ErrInvalidActionToken{} // unreachable: Huma checks the enum
}
