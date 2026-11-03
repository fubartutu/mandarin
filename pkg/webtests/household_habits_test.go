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

package webtests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/household/habits"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mandarin: /api/v2/habits webtests. "Today" is whatever the server says it
// is (progress.today), so the tests work on any date and in any time zone.

func setupHabitsEnv(t *testing.T) *echo.Echo {
	t.Helper()
	e, err := setupTestEnv()
	require.NoError(t, err)
	x, err := db.CreateTestEngine()
	require.NoError(t, err)
	require.NoError(t, x.Sync2(habits.GetTables()...))
	for _, table := range []string{"habits", "habit_groups", "habit_entries", "habit_skips", "habit_pauses"} {
		_, err = x.Exec("DELETE FROM " + table)
		require.NoError(t, err)
	}
	return e
}

type habitList struct {
	Items []habits.Habit
}

func makeHabit(t *testing.T, e *echo.Echo, token, body string) habits.Habit {
	t.Helper()
	rec := humaRequest(t, e, http.MethodPost, "/api/v2/habits", body, token, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	return decode[habits.Habit](t, rec.Body.Bytes())
}

func tickHabit(t *testing.T, e *echo.Echo, token string, id int64, uid, day string) habits.Habit {
	t.Helper()
	rec := humaRequest(t, e, http.MethodPost, fmt.Sprintf("/api/v2/habits/%d/entries", id),
		fmt.Sprintf(`{"uid":%q,"day":%q}`, uid, day), token, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	return decode[habits.Habit](t, rec.Body.Bytes())
}

func listHabits(t *testing.T, e *echo.Echo, token, query string) []habits.Habit {
	t.Helper()
	rec := humaRequest(t, e, http.MethodGet, "/api/v2/habits"+query, "", token, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	return decode[habitList](t, rec.Body.Bytes()).Items
}

// daysBefore returns the day n days before day (YYYY-MM-DD).
func daysBefore(t *testing.T, day string, n int) string {
	t.Helper()
	d, err := time.Parse("2006-01-02", day)
	require.NoError(t, err)
	return d.AddDate(0, 0, -n).Format("2006-01-02")
}

// humaJSON sends body as JSON.
func humaJSON(t *testing.T, e *echo.Echo, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	return humaRequest(t, e, method, path, string(raw), token, "")
}

// putHabit sends a full update made from h with change applied.
func putHabit(t *testing.T, e *echo.Echo, token string, h habits.Habit, change func(*habits.Habit)) *httptest.ResponseRecorder {
	t.Helper()
	change(&h)
	h.Progress = nil
	return humaJSON(t, e, http.MethodPut, fmt.Sprintf("/api/v2/habits/%d", h.ID), h, token)
}

func TestHouseholdHabits(t *testing.T) {
	e := setupHabitsEnv(t)
	user1 := humaTokenFor(t, &testuser1)

	h := makeHabit(t, e, user1, `{"title":"  Stretch ","kind":"daily"}`)
	assert.Equal(t, "Stretch", h.Title)
	assert.Equal(t, "1111111", h.Weekdays)
	assert.Equal(t, 1, h.Goal)
	assert.Equal(t, testuser1.ID, h.OwnerID)
	require.NotNil(t, h.Progress)
	assert.True(t, h.Progress.DueToday)
	assert.Empty(t, h.Progress.TodayEntries)
	today := h.Progress.Today

	// Ticking today makes it done; sending the same tick again changes nothing.
	uid := uuid.NewString()
	h = tickHabit(t, e, user1, h.ID, uid, today)
	assert.True(t, h.Progress.DoneToday)
	assert.Equal(t, 1, h.Progress.Streak)
	assert.Equal(t, []string{uid}, h.Progress.TodayEntries)
	h = tickHabit(t, e, user1, h.ID, uid, today)
	assert.Equal(t, 1, h.Progress.TodayCount)

	list := listHabits(t, e, user1, "")
	require.Len(t, list, 1)
	assert.True(t, list[0].Progress.DoneToday)

	// Taking the tick off; a second delete of the same uid is fine too.
	rec := humaRequest(t, e, http.MethodDelete, fmt.Sprintf("/api/v2/habits/%d/entries/%s", h.ID, uid), "", user1, "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	rec = humaRequest(t, e, http.MethodDelete, fmt.Sprintf("/api/v2/habits/%d/entries/%s", h.ID, uid), "", user1, "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())

	rec = humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v2/habits/%d", h.ID), "", user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	h = decode[habits.Habit](t, rec.Body.Bytes())
	assert.False(t, h.Progress.DoneToday)
	assert.Empty(t, rec.Header().Get("ETag"), "progress changes without Updated changing")

	// A full update: new title and Mon/Wed/Fri. The kind can't change.
	rec = putHabit(t, e, user1, h, func(h *habits.Habit) {
		h.Title = "Gym"
		h.Weekdays = "1010100"
		h.ReminderTimes = []string{"18:00", "07:30", "18:00"}
	})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	h = decode[habits.Habit](t, rec.Body.Bytes())
	assert.Equal(t, "Gym", h.Title)
	assert.Equal(t, "1010100", h.Weekdays)
	assert.Equal(t, []string{"07:30", "18:00"}, h.ReminderTimes)

	rec = putHabit(t, e, user1, h, func(h *habits.Habit) { h.Kind = habits.KindWeekly })
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	// PATCH works through the GET+PUT pair.
	rec = humaRequest(t, e, http.MethodPatch, fmt.Sprintf("/api/v2/habits/%d", h.ID), `{"color":"E8590C"}`, user1, "application/merge-patch+json")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "e8590c", decode[habits.Habit](t, rec.Body.Bytes()).Color)

	// Archived habits are left out unless asked for.
	rec = putHabit(t, e, user1, h, func(h *habits.Habit) { h.Archived = true })
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Empty(t, listHabits(t, e, user1, ""))
	assert.Len(t, listHabits(t, e, user1, "?archived=true"), 1)

	rec = humaRequest(t, e, http.MethodDelete, fmt.Sprintf("/api/v2/habits/%d", h.ID), "", user1, "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	rec = humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v2/habits/%d", h.ID), "", user1, "")
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHouseholdHabits_PersonalAndShared(t *testing.T) {
	e := setupHabitsEnv(t)
	user1 := humaTokenFor(t, &testuser1)
	user2 := humaTokenFor(t, &testuser2)

	mine := makeHabit(t, e, user1, `{"title":"Journal","kind":"daily"}`)
	bins := makeHabit(t, e, user1, `{"title":"Bins out","kind":"daily","weekdays":"0100000","shared":true}`)

	// user2 sees only the household routine.
	list := listHabits(t, e, user2, "")
	require.Len(t, list, 1)
	assert.Equal(t, bins.ID, list[0].ID)

	// user1's personal habit doesn't exist for user2, whatever they try.
	for _, req := range []struct{ method, path, body string }{
		{http.MethodGet, fmt.Sprintf("/api/v2/habits/%d", mine.ID), ""},
		{http.MethodDelete, fmt.Sprintf("/api/v2/habits/%d", mine.ID), ""},
		{http.MethodPost, fmt.Sprintf("/api/v2/habits/%d/entries", mine.ID), fmt.Sprintf(`{"uid":%q,"day":%q}`, uuid.NewString(), mine.Progress.Today)},
		{http.MethodPut, fmt.Sprintf("/api/v2/habits/%d/skips/%s", mine.ID, mine.Progress.Today), `{}`},
		{http.MethodGet, fmt.Sprintf("/api/v2/habits/%d/history?from=%s&to=%s", mine.ID, mine.Progress.Today, mine.Progress.Today), ""},
	} {
		rec := humaRequest(t, e, req.method, req.path, req.body, user2, "")
		assert.Equal(t, http.StatusNotFound, rec.Code, "%s %s: %s", req.method, req.path, rec.Body.String())
	}
	rec := putHabit(t, e, user2, mine, func(h *habits.Habit) { h.Title = "Mine now" })
	assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())

	// user2 can tick the routine; the tick records who did it.
	tickHabit(t, e, user2, bins.ID, uuid.NewString(), bins.Progress.Today)
	rec = humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v2/habits/%d/history?from=%s&to=%s", bins.ID, bins.Progress.Today, bins.Progress.Today), "", user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	hist := decode[habits.History](t, rec.Body.Bytes())
	require.Len(t, hist.Entries, 1)
	assert.Equal(t, testuser2.ID, hist.Entries[0].UserID)

	// user2 can change the routine but not make it personal: it would vanish for them.
	rec = putHabit(t, e, user2, bins, func(h *habits.Habit) { h.Title = "Bins and recycling" })
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	bins = decode[habits.Habit](t, rec.Body.Bytes())
	assert.Equal(t, testuser1.ID, bins.OwnerID, "the owner stays")
	rec = putHabit(t, e, user2, bins, func(h *habits.Habit) { h.Shared = false })
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	rec = putHabit(t, e, user1, bins, func(h *habits.Habit) { h.Shared = false })
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Empty(t, listHabits(t, e, user2, ""))

	// Link shares aren't household members.
	rec = humaRequest(t, e, http.MethodGet, "/api/v2/habits", "", menuLinkShareToken(t), "")
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
}

func TestHouseholdHabits_SkipPauseAndHistory(t *testing.T) {
	e := setupHabitsEnv(t)
	user1 := humaTokenFor(t, &testuser1)

	h := makeHabit(t, e, user1, `{"title":"Walk","kind":"daily"}`)
	today := h.Progress.Today
	yesterday := daysBefore(t, today, 1)

	// A late tick for yesterday counts from yesterday on.
	h = tickHabit(t, e, user1, h.ID, uuid.NewString(), yesterday)
	assert.Equal(t, 1, h.Progress.Streak)
	assert.True(t, h.Progress.DueToday)

	// Skipping today: not due, not missed. Skipping again changes the reason.
	rec := humaRequest(t, e, http.MethodPut, fmt.Sprintf("/api/v2/habits/%d/skips/%s", h.ID, today), `{"reason":"Raining"}`, user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	h = decode[habits.Habit](t, rec.Body.Bytes())
	assert.True(t, h.Progress.Skipped)
	assert.False(t, h.Progress.DueToday)
	rec = humaRequest(t, e, http.MethodPut, fmt.Sprintf("/api/v2/habits/%d/skips/%s", h.ID, today), `{"reason":"Storm"}`, user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v2/habits/%d/history?from=%s&to=%s", h.ID, daysBefore(t, today, 3), daysBefore(t, today, -5)), "", user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	hist := decode[habits.History](t, rec.Body.Bytes())
	assert.Equal(t, today, hist.To, "clipped to today")
	statuses := []string{}
	for _, p := range hist.Periods {
		statuses = append(statuses, p.Status)
	}
	assert.Equal(t, []string{habits.StatusBefore, habits.StatusBefore, habits.StatusDone, habits.StatusSkipped}, statuses)
	require.Len(t, hist.Skips, 1)
	assert.Equal(t, "Storm", hist.Skips[0].Reason)
	require.Len(t, hist.Entries, 1)

	rec = humaRequest(t, e, http.MethodDelete, fmt.Sprintf("/api/v2/habits/%d/skips/%s", h.ID, today), "", user1, "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())

	// Pausing from today; carrying on the same day leaves no pause behind.
	rec = putHabit(t, e, user1, h, func(h *habits.Habit) { h.Paused = true })
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	h = decode[habits.Habit](t, rec.Body.Bytes())
	assert.True(t, h.Paused)
	assert.False(t, h.Progress.DueToday)
	rec = putHabit(t, e, user1, h, func(h *habits.Habit) { h.Paused = false })
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	h = decode[habits.Habit](t, rec.Body.Bytes())
	assert.False(t, h.Paused)
	assert.True(t, h.Progress.DueToday)

	// Nothing can be ticked or skipped in the future.
	tomorrow := daysBefore(t, today, -1)
	rec = humaRequest(t, e, http.MethodPost, fmt.Sprintf("/api/v2/habits/%d/entries", h.ID), fmt.Sprintf(`{"uid":%q,"day":%q}`, uuid.NewString(), tomorrow), user1, "")
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	rec = humaRequest(t, e, http.MethodPut, fmt.Sprintf("/api/v2/habits/%d/skips/%s", h.ID, tomorrow), `{}`, user1, "")
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

func TestHouseholdHabits_Kinds(t *testing.T) {
	e := setupHabitsEnv(t)
	user1 := humaTokenFor(t, &testuser1)

	water := makeHabit(t, e, user1, `{"title":"Water","kind":"daily_count","goal":3}`)
	today := water.Progress.Today
	water = tickHabit(t, e, user1, water.ID, uuid.NewString(), today)
	water = tickHabit(t, e, user1, water.ID, uuid.NewString(), today)
	assert.Equal(t, 2, water.Progress.TodayCount)
	assert.False(t, water.Progress.DoneToday)
	water = tickHabit(t, e, user1, water.ID, uuid.NewString(), today)
	assert.True(t, water.Progress.DoneToday)
	assert.Len(t, water.Progress.TodayEntries, 3)

	run := makeHabit(t, e, user1, `{"title":"Run","kind":"weekly","goal":2,"weekdays":"1000000"}`)
	assert.Equal(t, "1111111", run.Weekdays, "weekly habits are due any day")
	run = tickHabit(t, e, user1, run.ID, uuid.NewString(), today)
	assert.Equal(t, 1, run.Progress.ThisWeek)
	assert.True(t, run.Progress.DueToday, "due until the week's goal is reached")

	snack := makeHabit(t, e, user1, `{"title":"Late snack","kind":"bad","goal":5}`)
	assert.Equal(t, 1, snack.Goal)
	snack = tickHabit(t, e, user1, snack.ID, uuid.NewString(), today)
	assert.Equal(t, 1, snack.Progress.TodayCount)
	assert.Zero(t, snack.Progress.Streak)
	rec := humaRequest(t, e, http.MethodPut, fmt.Sprintf("/api/v2/habits/%d/skips/%s", snack.ID, today), `{}`, user1, "")
	assert.Equal(t, http.StatusBadRequest, rec.Code, "bad habits can't be skipped")
}

func TestHouseholdHabits_Refusals(t *testing.T) {
	e := setupHabitsEnv(t)
	user1 := humaTokenFor(t, &testuser1)

	for _, body := range []string{
		`{"title":"","kind":"daily"}`,
		`{"title":"Stretch","kind":"sometimes"}`,
		`{"title":"Stretch","kind":"daily","weekdays":"0000000"}`,
		`{"title":"Stretch","kind":"daily","weekdays":"11111"}`,
		`{"title":"Water","kind":"daily_count","goal":100}`,
		`{"title":"Stretch","kind":"daily","color":"orange"}`,
		`{"title":"Stretch","kind":"daily","reminder_times":["7:30am"]}`,
		`{"title":"Stretch","kind":"daily","group_id":999}`,
	} {
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/habits", body, user1, "")
		assert.Contains(t, []int{http.StatusBadRequest, http.StatusNotFound, http.StatusUnprocessableEntity}, rec.Code, "%s: %s", body, rec.Body.String())
	}

	h := makeHabit(t, e, user1, `{"title":"Stretch","kind":"daily"}`)
	rec := humaRequest(t, e, http.MethodPost, fmt.Sprintf("/api/v2/habits/%d/entries", h.ID), fmt.Sprintf(`{"uid":"not-a-uuid","day":%q}`, h.Progress.Today), user1, "")
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())

	// A uid already used for another habit's tick is refused.
	other := makeHabit(t, e, user1, `{"title":"Read","kind":"daily"}`)
	uid := uuid.NewString()
	tickHabit(t, e, user1, h.ID, uid, h.Progress.Today)
	rec = humaRequest(t, e, http.MethodPost, fmt.Sprintf("/api/v2/habits/%d/entries", other.ID), fmt.Sprintf(`{"uid":%q,"day":%q}`, uid, h.Progress.Today), user1, "")
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())

	rec = humaRequest(t, e, http.MethodGet, fmt.Sprintf("/api/v2/habits/%d/history?from=2020-01-01&to=2026-01-01", h.ID), "", user1, "")
	assert.Equal(t, http.StatusBadRequest, rec.Code, "more than 400 days")
}

func TestHouseholdHabitGroups(t *testing.T) {
	e := setupHabitsEnv(t)
	user1 := humaTokenFor(t, &testuser1)
	user2 := humaTokenFor(t, &testuser2)

	rec := humaRequest(t, e, http.MethodPost, "/api/v2/habit-groups", `{"title":"Morning"}`, user1, "")
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	morning := decode[habits.HabitGroup](t, rec.Body.Bytes())

	h := makeHabit(t, e, user1, fmt.Sprintf(`{"title":"Stretch","kind":"daily","group_id":%d}`, morning.ID))
	assert.Equal(t, morning.ID, h.GroupID)

	// user2 can't see user1's personal group, nor put a habit in it.
	rec = humaRequest(t, e, http.MethodGet, "/api/v2/habit-groups", "", user2, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Empty(t, decode[struct{ Items []habits.HabitGroup }](t, rec.Body.Bytes()).Items)
	rec = humaRequest(t, e, http.MethodPost, "/api/v2/habits", fmt.Sprintf(`{"title":"Mine","kind":"daily","group_id":%d}`, morning.ID), user2, "")
	assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())

	// A household routine in user1's personal group can still be changed by user2.
	routine := makeHabit(t, e, user1, fmt.Sprintf(`{"title":"Feed the cat","kind":"daily","shared":true,"group_id":%d}`, morning.ID))
	rec = putHabit(t, e, user2, routine, func(h *habits.Habit) { h.Title = "Feed the cats" })
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	morning.Title = "Mornings"
	rec = humaJSON(t, e, http.MethodPut, fmt.Sprintf("/api/v2/habit-groups/%d", morning.ID), morning, user1)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "Mornings", decode[habits.HabitGroup](t, rec.Body.Bytes()).Title)

	// Deleting the group keeps its habits, ungrouped.
	rec = humaRequest(t, e, http.MethodDelete, fmt.Sprintf("/api/v2/habit-groups/%d", morning.ID), "", user1, "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	for _, h := range listHabits(t, e, user1, "") {
		assert.Zero(t, h.GroupID, h.Title)
	}
}

func TestHouseholdHabitsSync(t *testing.T) {
	e := setupHabitsEnv(t)
	user1 := humaTokenFor(t, &testuser1)
	user2 := humaTokenFor(t, &testuser2)

	water := makeHabit(t, e, user1, `{"title":"Water","kind":"daily_count","goal":3,"shared":true}`)
	mine := makeHabit(t, e, user1, `{"title":"Stretch","kind":"daily"}`)
	today := water.Progress.Today
	yesterday := daysBefore(t, today, 1)

	sync := func(token string, ops []map[string]any) habits.SyncResult {
		t.Helper()
		rec := humaJSON(t, e, http.MethodPost, "/api/v2/habits/sync", map[string]any{"ops": ops}, token)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		return decode[habits.SyncResult](t, rec.Body.Bytes())
	}
	find := func(r habits.SyncResult, id int64) *habits.Habit {
		t.Helper()
		for _, h := range r.Habits {
			if h.ID == id {
				return h
			}
		}
		return nil
	}

	// Ticks made offline, a late one for yesterday, and one taken back.
	a, b, c := uuid.NewString(), uuid.NewString(), uuid.NewString()
	ops := []map[string]any{
		{"id": "op1", "type": "tick", "habit_id": water.ID, "uid": a, "day": today},
		{"id": "op2", "type": "tick", "habit_id": water.ID, "uid": b, "day": today},
		{"id": "op3", "type": "untick", "habit_id": water.ID, "uid": b},
		{"id": "op4", "type": "tick", "habit_id": mine.ID, "uid": c, "day": yesterday},
		{"id": "op5", "type": "skip", "habit_id": mine.ID, "day": today},
	}
	r := sync(user2, ops[:3])
	assert.Empty(t, r.Rejected)
	require.Nil(t, find(r, mine.ID), "someone else's personal habit isn't sent")
	assert.Equal(t, []string{a}, find(r, water.ID).Progress.TodayEntries)

	r = sync(user1, ops)
	require.Empty(t, r.Rejected, "the same ops again change nothing and aren't refused")
	assert.Equal(t, 1, find(r, water.ID).Progress.TodayCount)
	got := find(r, mine.ID)
	assert.True(t, got.Progress.Skipped)
	assert.Equal(t, 1, got.Progress.Streak, "yesterday's late tick counts")
	assert.NotNil(t, r.Groups)

	// A change that can't apply is refused; the others still apply.
	r = sync(user2, []map[string]any{
		{"id": "bad1", "type": "tick", "habit_id": mine.ID, "uid": uuid.NewString(), "day": today},
		{"id": "bad2", "type": "tick", "habit_id": water.ID, "uid": uuid.NewString(), "day": "2999-01-01"},
		{"id": "ok", "type": "tick", "habit_id": water.ID, "uid": uuid.NewString(), "day": today},
	})
	require.Len(t, r.Rejected, 2)
	assert.Equal(t, "bad1", r.Rejected[0].OpID)
	assert.Equal(t, "bad2", r.Rejected[1].OpID)
	assert.NotEmpty(t, r.Rejected[1].Reason)
	assert.Equal(t, 2, find(r, water.ID).Progress.TodayCount)

	// No ops: just the current picture.
	r = sync(user1, []map[string]any{})
	assert.Len(t, r.Habits, 2)
}
