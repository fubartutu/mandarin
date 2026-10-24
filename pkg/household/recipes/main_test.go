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

package recipes

import (
	"os"
	"testing"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/files"
	"code.vikunja.io/api/pkg/household/menu"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

var (
	user1     = &user.User{ID: 1}
	linkShare = &models.LinkSharing{ID: 1}
)

// TestMain sets up an in-memory SQLite database with Vikunja's users and
// file tables plus ours. The scraper is pointed at a closed port, so tests
// that don't start a fake one see it as down.
func TestMain(m *testing.M) {
	log.InitLogger()
	config.InitDefaultConfig()
	files.InitTests()
	user.InitTests()
	_ = os.Setenv("MANDARIN_SCRAPER_URL", "http://127.0.0.1:1")

	x, err := db.CreateTestEngine()
	if err != nil {
		log.Fatal(err)
	}
	if err := x.Sync2(GetTables()...); err != nil {
		log.Fatal(err)
	}
	// Deleting a recipe turns its menu entries into notes.
	if err := x.Sync2(menu.GetTables()...); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func setup(t *testing.T) *xorm.Session {
	t.Helper()
	require.NoError(t, db.LoadFixtures())
	s := db.NewSession()
	t.Cleanup(func() { s.Close() })
	_, err := s.Exec("DELETE FROM recipes")
	require.NoError(t, err)
	return s
}

func lines(texts ...string) []*RecipeIngredient {
	out := make([]*RecipeIngredient, 0, len(texts))
	for _, text := range texts {
		out = append(out, &RecipeIngredient{Text: text})
	}
	return out
}

func addRecipe(t *testing.T, s *xorm.Session, title string, ingredients ...string) *Recipe {
	t.Helper()
	r := &Recipe{Title: title, Ingredients: lines(ingredients...), Steps: []string{"Cook it."}}
	require.NoError(t, r.Create(s, user1))
	return r
}
