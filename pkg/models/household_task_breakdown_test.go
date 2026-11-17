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

package models

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanBreakdownItems(t *testing.T) {
	long := strings.Repeat("é", 300)
	got := CleanBreakdownItems([]string{"  Get the\n rake ", "", "\t", long})
	assert.Equal(t, []string{"Get the rake", strings.Repeat("é", maxBreakdownItemLength)}, got)
}

func TestBreakdownChecklist(t *testing.T) {
	got := breakdownChecklist([]string{"A & B"})
	assert.Equal(t, `<ul data-type="taskList"><li data-type="taskItem" data-checked="false"><p>A &amp; B</p></li></ul>`, got)
	// The same checklist Vikunja's own helpers recognise (they reset it when a task repeats).
	assert.Equal(t, got, resetDescriptionChecklist(got))
}
