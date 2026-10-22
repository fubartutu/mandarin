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
	"testing"
	"time"

	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

func TestShoppingSummaries(t *testing.T) {
	service := fakeService(t)
	s := setup(t)
	addDevice(t, s, user2, "https://web.push.apple.com/two")
	s = setup0(t)
	_, err := SavePreferences(s, user2, &Preferences{Reminders: true, QuietHours: true, ShoppingListIDs: []int64{1}})
	require.NoError(t, err)

	// user1 adds three things to list 1 and one to list 2 (not followed);
	// user2's own adds never notify user2.
	for _, name := range []string{"Milk", "Eggs", "Bread"} {
		require.NoError(t, queueShoppingAdd(s, 1, name, user1.ID))
	}
	require.NoError(t, queueShoppingAdd(s, 2, "Socks", user1.ID))
	require.NoError(t, queueShoppingAdd(s, 1, "Coffee", user2.ID))
	_, err = s.Exec("INSERT INTO shopping_lists (id, title, position, created_by_id, created, updated) VALUES (1, 'Groceries', 1, 0, ?, ?)", time.Now(), time.Now())
	require.NoError(t, err)
	// Midday Sydney time, so quiet hours don't apply.
	noon := time.Date(2026, 10, 22, 12, 0, 0, 0, sydney(t))
	inSydneyAddedAt(t, s, noon)

	sendShoppingSummaries(noon)
	assert.Empty(t, service.requests, "still settling")

	sendShoppingSummaries(noon.Add(settle + time.Minute))
	require.Len(t, service.requests, 1, "one summary")

	// Nothing left to send.
	sendShoppingSummaries(noon.Add(settle + 2*time.Minute))
	assert.Len(t, service.requests, 1)
}

func TestShoppingSummaryWaitsForMorning(t *testing.T) {
	service := fakeService(t)
	s := setup(t)
	addDevice(t, s, user2, "https://web.push.apple.com/two")
	s = setup0(t)
	_, err := SavePreferences(s, user2, &Preferences{QuietHours: true, ShoppingListIDs: []int64{1}})
	require.NoError(t, err)
	require.NoError(t, queueShoppingAdd(s, 1, "Milk", user1.ID))
	late := time.Date(2026, 10, 22, 23, 30, 0, 0, sydney(t))
	inSydneyAddedAt(t, s, late.Add(-time.Hour))

	sendShoppingSummaries(late)
	assert.Empty(t, service.requests, "quiet hours")
}

func TestShoppingMessage(t *testing.T) {
	msg := shoppingMessage([]string{"Alex"}, []string{"milk", "eggs", "bread", "jam", "tea"}, "Groceries", 1)
	assert.Equal(t, "Groceries", msg.Title)
	assert.Equal(t, "Alex added milk, eggs and 3 more", msg.Body)
	assert.Equal(t, "/shopping/1", msg.URL)
}

func TestQuietHours(t *testing.T) {
	u := &user.User{Timezone: "Australia/Sydney"}
	at := func(h int) time.Time { return time.Date(2026, 10, 22, h, 0, 0, 0, sydney(t)) }
	assert.True(t, inQuietHours(at(22), u))
	assert.True(t, inQuietHours(at(6), u))
	assert.False(t, inQuietHours(at(7), u))
	assert.False(t, inQuietHours(at(21), u))
}

// inSydneyAddedAt puts user2 in Sydney time and dates the queued adds, then commits.
func inSydneyAddedAt(t *testing.T, s *xorm.Session, at time.Time) {
	t.Helper()
	_, err := s.Exec("UPDATE users SET timezone = 'Australia/Sydney' WHERE id = 2")
	require.NoError(t, err)
	_, err = s.Exec("UPDATE push_shopping_pending SET created = ?", at)
	require.NoError(t, err)
	require.NoError(t, s.Commit())
}

func sydney(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Australia/Sydney")
	require.NoError(t, err)
	return loc
}
