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

package push

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/cron"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/household/shopping"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/user"

	"xorm.io/xorm"
)

// Shopping summaries: when someone adds items to a list, everyone else who
// turned summaries on for that list gets one push once the adding has
// stopped for a while: "Alex added milk, eggs and 3 more to Groceries".

const (
	// settle is how long the list must be quiet before the summary goes.
	settle = 10 * time.Minute
	// Quiet hours, in each person's time zone: summaries wait until quietEnd.
	quietStart = 22
	quietEnd   = 7
)

// pendingShoppingAdd is one added item waiting to go into a summary.
type pendingShoppingAdd struct {
	ID          int64     `xorm:"bigint autoincr not null unique pk"`
	RecipientID int64     `xorm:"bigint not null index"`
	ListID      int64     `xorm:"bigint not null"`
	ActorID     int64     `xorm:"bigint not null"`
	ItemName    string    `xorm:"varchar(250) not null"`
	Created     time.Time `xorm:"created not null"`
}

func (*pendingShoppingAdd) TableName() string {
	return "push_shopping_pending"
}

func hookShopping() {
	shopping.OnItemAdded = queueShoppingAdd
	if err := cron.Schedule("* * * * *", func() { sendShoppingSummaries(time.Now()) }); err != nil {
		log.Errorf("Could not schedule shopping summaries: %s", err)
	}
}

// queueShoppingAdd notes the item for everyone (but the person adding it)
// who wants summaries of this list.
func queueShoppingAdd(s *xorm.Session, listID int64, name string, actorID int64) error {
	all := []*Preferences{}
	if err := s.Where("user_id != ?", actorID).Find(&all); err != nil {
		return err
	}
	for _, p := range all {
		if !slices.Contains(p.ShoppingListIDs, listID) {
			continue
		}
		if _, err := s.Insert(&pendingShoppingAdd{RecipientID: p.UserID, ListID: listID, ActorID: actorID, ItemName: name}); err != nil {
			return err
		}
	}
	return nil
}

type summaryKey struct{ recipient, list int64 }

// sendShoppingSummaries sends each (person, list) whose adds have settled,
// outside the person's quiet hours, and clears what it sent.
func sendShoppingSummaries(now time.Time) {
	s := db.NewSession()
	pending := []*pendingShoppingAdd{}
	err := s.OrderBy("id").Find(&pending)
	_ = s.Commit()
	s.Close()
	if err != nil {
		log.Errorf("Shopping summaries: %s", err)
		return
	}

	groups := map[summaryKey][]*pendingShoppingAdd{}
	for _, p := range pending {
		k := summaryKey{p.RecipientID, p.ListID}
		groups[k] = append(groups[k], p)
	}
	for k, adds := range groups {
		if now.Sub(adds[len(adds)-1].Created) < settle {
			continue // still adding
		}
		if err := sendSummary(k, adds, now); err != nil {
			log.Errorf("Shopping summary for user %d: %s", k.recipient, err)
		}
	}
}

func sendSummary(k summaryKey, adds []*pendingShoppingAdd, now time.Time) error {
	s := db.NewSession()
	defer s.Close()
	prefs, err := preferencesFor(s, k.recipient)
	if err != nil {
		return err
	}
	recipient, err := user.GetUserByID(s, k.recipient)
	if err != nil {
		return err
	}
	if prefs.QuietHours && inQuietHours(now, recipient) {
		return s.Commit() // they wait until the morning
	}

	var title string
	if _, err := s.Table("shopping_lists").Where("id = ?", k.list).Cols("title").Get(&title); err != nil {
		return err
	}
	actors, err := actorNames(s, adds)
	if err != nil {
		return err
	}
	ids := make([]int64, 0, len(adds))
	for _, a := range adds {
		ids = append(ids, a.ID)
	}
	// Cleared first: a summary that fails to send isn't retried forever.
	if _, err := s.In("id", ids).Delete(&pendingShoppingAdd{}); err != nil {
		return err
	}
	if err := s.Commit(); err != nil {
		return err
	}
	// Turned off meanwhile, or the list is gone: nothing to say.
	if !slices.Contains(prefs.ShoppingListIDs, k.list) || title == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_, err = Send(ctx, k.recipient, shoppingMessage(actors, itemNames(adds), title, k.list))
	return err
}

func itemNames(adds []*pendingShoppingAdd) []string {
	names := []string{}
	for _, a := range adds {
		if !slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(n, a.ItemName) }) {
			names = append(names, a.ItemName)
		}
	}
	return names
}

func actorNames(s *xorm.Session, adds []*pendingShoppingAdd) ([]string, error) {
	ids := []int64{}
	for _, a := range adds {
		if !slices.Contains(ids, a.ActorID) {
			ids = append(ids, a.ActorID)
		}
	}
	users, err := user.GetUsersByIDs(s, ids)
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, id := range ids {
		if u, ok := users[id]; ok {
			names = append(names, u.GetName())
		}
	}
	return names, nil
}

// shoppingMessage: "Alex added milk, eggs and 3 more to Groceries".
func shoppingMessage(actors, items []string, list string, listID int64) *Message {
	who := strings.Join(actors, " and ")
	if who == "" {
		who = "Someone"
	}
	what := strings.Join(items[:min(2, len(items))], ", ")
	if len(items) > 2 {
		what += fmt.Sprintf(" and %d more", len(items)-2)
	}
	return &Message{
		Title: list,
		Body:  fmt.Sprintf("%s added %s", who, what),
		URL:   fmt.Sprintf("/shopping/%d", listID),
		Tag:   fmt.Sprintf("shopping-%d", listID),
		ttl:   24 * time.Hour,
	}
}

func inQuietHours(now time.Time, u *user.User) bool {
	loc := config.GetTimeZone()
	if u != nil && u.Timezone != "" {
		if l, err := time.LoadLocation(u.Timezone); err == nil {
			loc = l
		}
	}
	h := now.In(loc).Hour()
	return h >= quietStart || h < quietEnd
}
