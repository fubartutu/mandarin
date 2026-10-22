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

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

func TestDevices(t *testing.T) {
	s := setup(t)
	d := addDevice(t, s, user1, "https://web.push.apple.com/abc")

	s = setup0(t)
	devices, err := ListDevices(s, user1)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, "iPhone · Safari", devices[0].Name)

	// The same browser subscribing again under another login moves it.
	addDevice(t, s, user2, "https://web.push.apple.com/abc")
	devices, err = ListDevices(s, user1)
	require.NoError(t, err)
	assert.Empty(t, devices)

	_, err = SaveDevice(s, user1, &DeviceInput{Endpoint: "http://insecure.example.com/x"})
	require.ErrorAs(t, err, &ErrInvalidDevice{})
	require.ErrorAs(t, DeleteDevice(s, user1, d.ID), &ErrDeviceDoesNotExist{}, "not theirs any more")
	require.NoError(t, DeleteDevice(s, user2, d.ID))
}

// setup0 is a fresh session without clearing the tables.
func setup0(t *testing.T) *xorm.Session {
	t.Helper()
	s := db.NewSession()
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSend(t *testing.T) {
	service := fakeService(t)
	s := setup(t)
	addDevice(t, s, user1, "https://web.push.apple.com/live")
	addDevice(t, s, user1, "https://fcm.googleapis.com/gone")
	addDevice(t, s, user2, "https://web.push.apple.com/someone-else")
	service.status["https://fcm.googleapis.com/gone"] = http.StatusGone

	sent, err := Send(context.Background(), user1.ID, &Message{Title: "Milk", Body: "Reminder", URL: "/tasks/1", Tag: "task-1"})
	require.NoError(t, err)
	assert.Equal(t, 1, sent)
	require.Len(t, service.requests, 2, "only user1's devices")

	r := service.requests[0]
	assert.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "vapid t="), "signed with the server's key")
	assert.Equal(t, "aes128gcm", r.Header.Get("Content-Encoding"), "encrypted for the device")
	assert.NotContains(t, string(service.bodies[0]), "Milk", "the push service can't read it")
	assert.Equal(t, "task-1", r.Header.Get("Topic"))

	s = setup0(t)
	devices, err := ListDevices(s, user1)
	require.NoError(t, err)
	require.Len(t, devices, 1, "the gone device is forgotten")
	assert.NotNil(t, devices[0].LastSent)
}

func TestMessages(t *testing.T) {
	due := time.Now().Add(2 * time.Hour)
	msg, wants := messageFor(&models.ReminderDueNotification{
		User:    &user.User{ID: 1, Timezone: "Australia/Sydney"},
		Task:    &models.Task{ID: 12, Title: "Renew car rego", DueDate: due},
		Project: &models.Project{Title: "Car"},
	})
	require.NotNil(t, msg)
	assert.Equal(t, "Renew car rego", msg.Title)
	assert.True(t, strings.HasPrefix(msg.Body, "Reminder · Car · due "), msg.Body)
	assert.Equal(t, "/tasks/12", msg.URL)
	assert.True(t, wants(&Preferences{Reminders: true}))
	assert.False(t, wants(&Preferences{Reminders: false}))

	week := time.Now().Add(-7 * 24 * time.Hour)
	msg, wants = messageFor(&models.UndoneTasksOverdueNotification{
		Assigned: map[int64]*models.Task{
			1: {ID: 1, Title: "Bins", DueDate: week},
			2: {ID: 2, Title: "Tax", DueDate: week.Add(-time.Hour)},
		},
		Followed: map[int64]*models.Task{
			2: {ID: 2, Title: "Tax", DueDate: week.Add(-time.Hour)},
			3: {ID: 3, Title: "Dentist", DueDate: week.Add(time.Hour)},
			4: {ID: 4, Title: "Gutters", DueDate: week.Add(2 * time.Hour)},
		},
	})
	require.NotNil(t, msg)
	assert.Equal(t, "4 overdue tasks", msg.Title)
	assert.Equal(t, "Tax, Bins, Dentist and 1 more", msg.Body, "longest overdue first, no repeats")
	assert.True(t, wants(&Preferences{Overdue: true}))

	msg, _ = messageFor(&models.TaskCommentNotification{})
	assert.Nil(t, msg, "comments don't push")
}
