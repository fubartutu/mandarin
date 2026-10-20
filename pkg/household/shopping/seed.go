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
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/log"

	"xorm.io/xorm"
)

// DefaultCategories is the category order a new household starts with,
// roughly the walk through an Australian supermarket. Members can rename,
// reorder and add to it.
var DefaultCategories = []string{
	"Fruit & Veg",
	"Bakery",
	"Meat & Seafood",
	"Deli",
	"Dairy & Eggs",
	"Fridge",
	"Frozen",
	"Pantry",
	"Breakfast",
	"Snacks & Sweets",
	"Drinks",
	"Household & Cleaning",
	"Health & Beauty",
	"Baby & Kids",
	"Pets",
	"Clothing & Shoes",
	"Other",
}

// DefaultListTitle is the list a new household starts with.
const DefaultListTitle = "Groceries"

// Seed gives an empty household its starting categories and list. It only
// writes to empty tables, so it is safe to run on every start-up.
func Seed(s *xorm.Session) error {
	count, err := s.Count(&ShoppingCategory{})
	if err != nil {
		return err
	}
	if count == 0 {
		categories := make([]*ShoppingCategory, 0, len(DefaultCategories))
		for i, title := range DefaultCategories {
			categories = append(categories, &ShoppingCategory{Title: title, Position: float64(i + 1)})
		}
		if _, err := s.Insert(&categories); err != nil {
			return err
		}
	}

	count, err = s.Count(&ShoppingList{})
	if err != nil {
		return err
	}
	if count == 0 {
		if _, err := s.Insert(&ShoppingList{Title: DefaultListTitle, Position: 1}); err != nil {
			return err
		}
	}
	return nil
}

// Init runs at server start-up, after the migrations.
func Init() {
	if err := seedInTransaction(); err != nil {
		log.Fatalf("Could not seed shopping defaults: %s", err)
	}
}

// seedInTransaction is split out of Init so its deferred Close still runs:
// log.Fatalf exits the program immediately, skipping any defers.
func seedInTransaction() error {
	s := db.NewSession()
	defer s.Close()

	if err := Seed(s); err != nil {
		_ = s.Rollback()
		return err
	}
	return s.Commit()
}
