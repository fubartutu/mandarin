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
	"time"
)

// Item fields as sync sees them. quantity and unit travel together as
// "amount", so "2 kg" from one phone and "500 g" from another never mix into
// "2 g".
const (
	fieldName     = "name"
	fieldAmount   = "amount"
	fieldNote     = "note"
	fieldCategory = "category_id"
	fieldDone     = "done"
	fieldPosition = "position"
)

var itemFields = []string{fieldName, fieldAmount, fieldNote, fieldCategory, fieldDone, fieldPosition}

// fieldStamp records when an item field last changed (Unix milliseconds,
// as the phone's clock had it) and which member changed it.
type fieldStamp struct {
	At int64 `json:"at"`
	By int64 `json:"by"`
}

// fieldStamps maps a field name to its last change. Stored as JSON in
// shopping_items.field_times. Items from before sync existed have none,
// which counts as "changed at time 0": any change wins.
type fieldStamps map[string]fieldStamp

// stampAll marks every field as changed at at by by, for a new item.
func stampAll(at time.Time, by int64) fieldStamps {
	return fieldStamps(nil).with(itemFields, at, by)
}

// with returns a copy with fields stamped. The receiver is left alone, so
// a loaded item's stamps aren't changed by accident.
func (f fieldStamps) with(fields []string, at time.Time, by int64) fieldStamps {
	out := make(fieldStamps, len(itemFields))
	for k, v := range f {
		out[k] = v
	}
	for _, field := range fields {
		out[field] = fieldStamp{At: at.UnixMilli(), By: by}
	}
	return out
}

// changedFields lists the fields that differ between two versions of an item.
func changedFields(before, after *ShoppingItem) []string {
	var changed []string
	for _, field := range itemFields {
		if fieldValue(before, field) != fieldValue(after, field) {
			changed = append(changed, field)
		}
	}
	return changed
}

// amount is quantity and unit as one comparable value.
type amount struct {
	Quantity float64
	Unit     string
}

// fieldValue returns a field's value in a form that == can compare.
func fieldValue(item *ShoppingItem, field string) any {
	switch field {
	case fieldName:
		return item.Name
	case fieldAmount:
		return amount{item.Quantity, item.Unit}
	case fieldNote:
		return item.Note
	case fieldCategory:
		return item.CategoryID
	case fieldDone:
		return item.Done
	case fieldPosition:
		return item.Position
	}
	return nil
}

// setFieldValue writes a value fieldValue returned back onto an item.
func setFieldValue(item *ShoppingItem, field string, value any) {
	switch field {
	case fieldName:
		item.Name = value.(string)
	case fieldAmount:
		a := value.(amount)
		item.Quantity, item.Unit = a.Quantity, a.Unit
	case fieldNote:
		item.Note = value.(string)
	case fieldCategory:
		item.CategoryID = value.(int64)
	case fieldDone:
		item.Done = value.(bool)
	case fieldPosition:
		item.Position = value.(float64)
	}
}
