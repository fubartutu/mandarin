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
	"regexp"
	"strconv"
	"strings"
)

// RecipeIngredient is one ingredient line. Only Group and Text are written
// by clients; the server works out the rest from Text, using the scraper's
// trained parser when it's running (see ParseIngredients) and a simple one
// otherwise.
type RecipeIngredient struct {
	Group string `json:"group" maxLength:"250" doc:"The heading this line sits under, e.g. Sauce; empty for none."`
	Text  string `json:"text" maxLength:"500" doc:"The line as written, e.g. 2 cups (250g) plain flour, sifted."`

	Quantity float64 `json:"quantity" readOnly:"true" doc:"The amount to shop for, e.g. 250 (metric preferred; the top of a range). 0 when there's none."`
	Unit     string  `json:"unit" readOnly:"true" doc:"E.g. g, tbsp, cans; empty for a plain count."`
	Name     string  `json:"name" readOnly:"true" doc:"What to buy, e.g. plain flour."`
	Note     string  `json:"note" readOnly:"true" doc:"Preparation and comments, e.g. sifted."`
}

// The simple parser understands "<amount> [<unit>] <name>[, <note>]", like the
// shopping list's add box: "2 1/2 cups flour, sifted", "½ tsp salt", "3 eggs".
var (
	leadingAmount = regexp.MustCompile(`^(\d+\s+\d+/\d+|\d+/\d+|\d+(?:[.,]\d+)?)?\s*([½⅓⅔¼¾⅛])?\s*(.*)$`)
	vulgar        = map[string]float64{"½": 0.5, "⅓": 1.0 / 3, "⅔": 2.0 / 3, "¼": 0.25, "¾": 0.75, "⅛": 0.125}
	knownUnits    = map[string]string{
		"g": "g", "gram": "g", "grams": "g", "kg": "kg", "kilo": "kg", "kilos": "kg", "kilogram": "kg", "kilograms": "kg",
		"mg": "mg", "ml": "mL", "millilitre": "mL", "millilitres": "mL", "milliliter": "mL", "milliliters": "mL",
		"l": "L", "litre": "L", "litres": "L", "liter": "L", "liters": "L",
		"tsp": "tsp", "teaspoon": "tsp", "teaspoons": "tsp", "tbsp": "tbsp", "tablespoon": "tbsp", "tablespoons": "tbsp",
		"cup": "cup", "cups": "cups", "oz": "oz", "lb": "lb", "lbs": "lb",
		"can": "can", "cans": "cans", "tin": "tin", "tins": "tins", "packet": "packet", "packets": "packets",
		"pack": "pack", "packs": "packs", "bunch": "bunch", "bunches": "bunches", "clove": "clove", "cloves": "cloves",
		"pinch": "pinch", "handful": "handful", "slice": "slice", "slices": "slices", "sprig": "sprig", "sprigs": "sprigs",
	}
)

// parseSimply fills Quantity, Unit, Name and Note from Text.
func (i *RecipeIngredient) parseSimply() {
	i.Quantity, i.Unit, i.Name, i.Note = 0, "", i.Text, ""
	m := leadingAmount.FindStringSubmatch(i.Text)
	if m == nil {
		return
	}
	i.Quantity = parseNumber(m[1]) + vulgar[m[2]]
	rest := m[3]
	if i.Quantity > 0 {
		// "200g butter" and "200 g butter" both have a unit.
		word, after, _ := strings.Cut(strings.TrimLeft(rest, " "), " ")
		if unit, ok := knownUnits[strings.ToLower(strings.TrimSuffix(word, "."))]; ok && after != "" {
			i.Unit, rest = unit, after
		}
		rest = strings.TrimPrefix(strings.TrimSpace(rest), "of ")
	}
	name, note, _ := strings.Cut(rest, ",")
	if name = strings.TrimSpace(name); name != "" {
		i.Name = name
		i.Note = strings.TrimSpace(note)
	}
}

func parseNumber(s string) float64 {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", ".")
	if s == "" {
		return 0
	}
	whole, fraction, mixed := strings.Cut(s, " ")
	if !mixed {
		whole, fraction = "", s
	}
	total := 0.0
	if whole != "" {
		w, _ := strconv.ParseFloat(whole, 64)
		total = w
	}
	if num, den, ok := strings.Cut(fraction, "/"); ok {
		n, _ := strconv.ParseFloat(num, 64)
		d, _ := strconv.ParseFloat(den, 64)
		if d > 0 {
			total += n / d
		}
		return total
	}
	f, _ := strconv.ParseFloat(fraction, 64)
	return total + f
}
