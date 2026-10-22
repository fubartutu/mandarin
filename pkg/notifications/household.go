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

package notifications

// Mandarin (new file in this package, so upstream merges don't conflict):
// lets the household push package see each notification as it's sent, and
// push it to the user's phones. See pkg/household/push.

var householdHook func(notifiable Notifiable, notification Notification)

// SetHouseholdHook registers the function Notify calls for every
// notification it's about to send.
func SetHouseholdHook(f func(notifiable Notifiable, notification Notification)) {
	householdHook = f
}

// OrHouseholdPush is true when emailEnabled is, or push is on, so crons that
// only ran for email (the overdue summary) also run for push. Sending mail
// still checks the mailer itself.
func OrHouseholdPush(emailEnabled bool) bool {
	return emailEnabled || householdHook != nil
}
