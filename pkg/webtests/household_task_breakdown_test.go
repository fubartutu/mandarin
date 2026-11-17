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
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"code.vikunja.io/api/pkg/household/breakdown"
	"code.vikunja.io/api/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mandarin: /api/v2/tasks/{task}/breakdown webtests.

func TestHouseholdTaskBreakdown(t *testing.T) {
	e, err := setupTestEnv()
	require.NoError(t, err)
	user1 := humaTokenFor(t, &testuser1)

	t.Run("steps become subtasks in the same project", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/tasks/1/breakdown",
			`{"mode":"steps","items":["Scalp the lawn","  ","Dethatch   the lawn"]}`, user1, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		task := decode[models.Task](t, rec.Body.Bytes())
		titles := map[string]*models.Task{}
		for _, sub := range task.RelatedTasks[models.RelationKindSubtask] {
			titles[sub.Title] = sub
		}
		require.Contains(t, titles, "Scalp the lawn")
		require.Contains(t, titles, "Dethatch the lawn")
		assert.Equal(t, int64(1), titles["Scalp the lawn"].ProjectID)

		// The child knows its parent.
		rec = humaRequest(t, e, http.MethodGet, "/api/v2/tasks/"+strconv.FormatInt(titles["Scalp the lawn"].ID, 10), "", user1, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		child := decode[models.Task](t, rec.Body.Bytes())
		require.Len(t, child.RelatedTasks[models.RelationKindParenttask], 1)
		assert.Equal(t, int64(1), child.RelatedTasks[models.RelationKindParenttask][0].ID)
	})

	t.Run("chunks become a checklist after the description", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/tasks/1/breakdown",
			`{"mode":"chunks","items":["Get the rake <from> the shed","Rake one strip"]}`, user1, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		task := decode[models.Task](t, rec.Body.Bytes())
		assert.Equal(t, `Lorem Ipsum<ul data-type="taskList">`+
			`<li data-type="taskItem" data-checked="false"><p>Get the rake &lt;from&gt; the shed</p></li>`+
			`<li data-type="taskItem" data-checked="false"><p>Rake one strip</p></li></ul>`, task.Description)
	})

	t.Run("nothing to add", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/tasks/1/breakdown", `{"mode":"chunks","items":[" "]}`, user1, "")
		assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	})

	t.Run("someone else's task", func(t *testing.T) {
		user2 := humaTokenFor(t, &testuser2)
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/tasks/1/breakdown", `{"mode":"steps","items":["x"]}`, user2, "")
		assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
		rec = humaRequest(t, e, http.MethodPost, "/api/v2/tasks/1/breakdown/suggest", `{"mode":"steps"}`, user2, "")
		assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	})
}

func TestHouseholdTaskBreakdownSuggest(t *testing.T) {
	e, err := setupTestEnv()
	require.NoError(t, err)
	user1 := humaTokenFor(t, &testuser1)

	var asked breakdown.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&asked)
		w.Header().Set("Content-Type", "application/json")
		if asked.Title == "task #3 high prio" {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"detail": "The AI isn't set up."}`))
			return
		}
		_, _ = w.Write([]byte(`{"steps": [{"title": "Scalp the lawn", "minutes": 90}]}`))
	}))
	t.Cleanup(server.Close)
	t.Setenv("MANDARIN_SCRAPER_URL", server.URL)

	rec := humaRequest(t, e, http.MethodPost, "/api/v2/tasks/1/breakdown/suggest", `{"mode":"steps"}`, user1, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	got := decode[models.BreakdownSuggestion](t, rec.Body.Bytes())
	assert.Equal(t, []models.BreakdownStep{{Title: "Scalp the lawn", Minutes: 90}}, got.Steps)
	assert.Equal(t, "steps", asked.Mode)
	assert.Equal(t, "task #1", asked.Title)
	assert.Equal(t, "Lorem Ipsum", asked.Description)
	assert.NotEmpty(t, asked.Project)

	t.Run("AI not set up", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/tasks/3/breakdown/suggest", `{"mode":"chunks"}`, user1, "")
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Body.String(), "isn't set up")
	})

	t.Run("scraper down", func(t *testing.T) {
		t.Setenv("MANDARIN_SCRAPER_URL", "http://127.0.0.1:1")
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/tasks/1/breakdown/suggest", `{"mode":"steps"}`, user1, "")
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code, rec.Body.String())
	})

	t.Run("bad mode", func(t *testing.T) {
		rec := humaRequest(t, e, http.MethodPost, "/api/v2/tasks/1/breakdown/suggest", `{"mode":"all"}`, user1, "")
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	})
}
