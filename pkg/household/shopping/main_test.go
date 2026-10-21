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

package shopping

import (
	"os"
	"testing"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/files"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

// Fixture users from pkg/db/fixtures/users.yml, and a link share that must
// never get at household data.
var (
	user1     = &user.User{ID: 1}
	user2     = &user.User{ID: 2}
	linkShare = &models.LinkSharing{ID: 1}
)

// TestMain runs once before this package's tests: an in-memory SQLite
// database with Vikunja's user tables (and fixture users) plus ours.
func TestMain(m *testing.M) {
	log.InitLogger()
	config.InitDefaultConfig()
	// Order matters: each InitTests replaces the fixture set, and we need the
	// users. Same order as pkg/models' TestMain.
	files.InitTests() // photo storage
	user.InitTests()

	x, err := db.CreateTestEngine()
	if err != nil {
		log.Fatal(err)
	}
	if err := x.Sync2(GetTables()...); err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

// setup gives each test a freshly seeded household and a session.
// t.Cleanup closes the session when the test ends.
func setup(t *testing.T) *xorm.Session {
	t.Helper()
	require.NoError(t, db.LoadFixtures())

	s := db.NewSession()
	t.Cleanup(func() { s.Close() })

	for _, table := range []string{"shopping_items", "shopping_lists", "shopping_categories", "shopping_shops", "shopping_catalog_items", "shopping_not_stocked", "shopping_changes", "shopping_conflicts", "shopping_item_aliases"} {
		_, err := s.Exec("DELETE FROM " + table)
		require.NoError(t, err)
	}
	require.NoError(t, Seed(s))
	return s
}

// groceries returns the list Seed created.
func groceries(t *testing.T, s *xorm.Session) *ShoppingList {
	t.Helper()
	l := &ShoppingList{}
	has, err := s.Where("title = ?", DefaultListTitle).Get(l)
	require.NoError(t, err)
	require.True(t, has)
	return l
}

// addShop adds a shop to a list as user1.
func addShop(t *testing.T, s *xorm.Session, listID int64, title string) *ShoppingShop {
	t.Helper()
	shop := &ShoppingShop{ListID: listID, Title: title}
	require.NoError(t, shop.Create(s, user1))
	return shop
}

// catalogEntry looks a catalogue entry up by name.
func catalogEntry(t *testing.T, s *xorm.Session, name string) *ShoppingCatalogItem {
	t.Helper()
	entry, exists, err := findCatalogItemByName(s, name)
	require.NoError(t, err)
	require.True(t, exists, "catalogue has %q", name)
	return entry
}

// addItem inserts an item as user1 and fails the test on error.
func addItem(t *testing.T, s *xorm.Session, item *ShoppingItem) *ShoppingItem {
	t.Helper()
	can, err := item.CanCreate(s, user1)
	require.NoError(t, err)
	require.True(t, can)
	require.NoError(t, item.Create(s, user1))
	return item
}
