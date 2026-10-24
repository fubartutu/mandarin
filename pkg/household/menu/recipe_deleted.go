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

package menu

import (
	"strings"
	"time"

	"xorm.io/xorm"
)

const deletedRecipeNote = "Deleted recipe"

// RecipeDeleted turns every entry planned with the recipe, past and future,
// into a note carrying its title, so the day isn't left empty and past weeks
// still read properly. Day, meal and position stay. It's called by the
// recipe's Delete in the same session (transaction), before the recipe row
// goes; nothing is written when the recipe was never planned (SQLite allows
// one writer at a time).
func RecipeDeleted(s *xorm.Session, recipeID int64, title string) error {
	planned, err := s.Where("recipe_id = ?", recipeID).Exist(&MenuEntry{})
	if err != nil || !planned {
		return err
	}
	_, err = s.Table(&MenuEntry{}).Where("recipe_id = ?", recipeID).Update(map[string]any{
		"recipe_id": 0,
		"servings":  0,
		"note":      noteFromTitle(title),
		"updated":   time.Now(),
	})
	return err
}

// noteFromTitle trims the title and cuts it to the note's length, counting
// runes (characters) rather than bytes so a multi-byte letter isn't split.
func noteFromTitle(title string) string {
	note := []rune(strings.TrimSpace(title))
	if len(note) > maxNoteLength {
		note = note[:maxNoteLength]
	}
	if len(note) == 0 {
		return deletedRecipeNote
	}
	return strings.TrimSpace(string(note))
}
