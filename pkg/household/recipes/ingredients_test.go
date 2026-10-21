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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSimply(t *testing.T) {
	cases := []struct {
		text     string
		quantity float64
		unit     string
		name     string
		note     string
	}{
		{"2 1/2 cups plain flour, sifted", 2.5, "cups", "plain flour", "sifted"},
		{"200g butter", 200, "g", "butter", ""},
		{"½ tsp salt", 0.5, "tsp", "salt", ""},
		{"1 ½ tbsp oil", 1.5, "tbsp", "oil", ""},
		{"3 eggs", 3, "", "eggs", ""},
		{"1 pinch of salt", 1, "pinch", "salt", ""},
		{"1,5 L milk", 1.5, "L", "milk", ""},
		{"Salt and pepper, to taste", 0, "", "Salt and pepper", "to taste"},
		{"2 g", 2, "", "g", ""}, // a lone unit is the name
	}
	for _, c := range cases {
		i := &RecipeIngredient{Text: c.text}
		i.parseSimply()
		assert.Equal(t, RecipeIngredient{Text: c.text, Quantity: c.quantity, Unit: c.unit, Name: c.name, Note: c.note}, *i, c.text)
	}
}
