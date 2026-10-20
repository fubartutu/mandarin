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

package websocket

// Mandarin: the household shopping module's live updates. This file is new
// (not an edit to a Vikunja file); it lives in this package to reach the hub's
// connection list and the subscribable events.

// EventHouseholdShoppingChanged tells apps that shopping data changed and
// they should sync. Every household member may subscribe.
const EventHouseholdShoppingChanged = "household.shopping.changed"

func init() {
	validEvents[EventHouseholdShoppingChanged] = true
}

// PublishToAll sends an event to every connected user who subscribed to it.
func (h *Hub) PublishToAll(event string, data any) {
	h.mu.RLock()
	userIDs := make([]int64, 0, len(h.connections))
	for id := range h.connections {
		userIDs = append(userIDs, id)
	}
	h.mu.RUnlock()

	for _, id := range userIDs {
		h.PublishForUser(id, event, data)
	}
}
