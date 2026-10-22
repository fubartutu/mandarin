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
	"encoding/json"
	"net/http"
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/log"

	"github.com/SherClockHolmes/webpush-go"
)

// Message is what a device shows. The service worker turns it into a
// notification (src/household/sw.ts in the frontend).
type Message struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	// URL is the app path to open when the notification is tapped, e.g. /tasks/12.
	URL string `json:"url"`
	// Tag replaces an earlier notification with the same tag instead of stacking.
	Tag string `json:"tag,omitempty"`

	ttl     time.Duration
	urgency webpush.Urgency
}

// httpClient sends the pushes. Tests replace it.
var httpClient webpush.HTTPClient = &http.Client{Timeout: 30 * time.Second}

// Send pushes msg to every device of the user and returns how many the push
// services accepted. Devices the push service says are gone are forgotten.
// It opens its own database sessions: call it outside a transaction.
func Send(ctx context.Context, userID int64, msg *Message) (int, error) {
	s := db.NewSession()
	k, err := keys(s)
	devices := []*Device{}
	if err == nil {
		err = s.Where("user_id = ?", userID).Find(&devices)
	}
	if err == nil {
		err = s.Commit()
	}
	s.Close()
	if err != nil || len(devices) == 0 {
		return 0, err
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}
	options := &webpush.Options{
		HTTPClient: httpClient,
		// Apple requires a contact; the server's public address is one.
		Subscriber:      config.ServicePublicURL.GetString(),
		VAPIDPublicKey:  k.PublicKey,
		VAPIDPrivateKey: k.PrivateKey,
		TTL:             int(max(msg.ttl, time.Hour).Seconds()),
		Urgency:         msg.urgency,
		Topic:           topic(msg.Tag),
	}
	if options.Urgency == "" {
		options.Urgency = webpush.UrgencyNormal
	}

	sent, gone := 0, []int64{}
	reached := []int64{}
	for _, d := range devices {
		sub := &webpush.Subscription{Endpoint: d.Endpoint, Keys: webpush.Keys{P256dh: d.P256dh, Auth: d.Auth}}
		resp, err := webpush.SendNotificationWithContext(ctx, payload, sub, options)
		if err != nil {
			log.Warningf("Push to device %d of user %d failed: %s", d.ID, userID, err)
			continue
		}
		_ = resp.Body.Close()
		switch {
		case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
			gone = append(gone, d.ID) // unsubscribed, or the app was removed
		case resp.StatusCode >= 400:
			log.Warningf("Push to device %d of user %d: the push service answered %d", d.ID, userID, resp.StatusCode)
		default:
			sent++
			reached = append(reached, d.ID)
		}
	}

	s = db.NewSession()
	defer s.Close()
	if len(gone) > 0 {
		if _, err := s.In("id", gone).Delete(&Device{}); err != nil {
			_ = s.Rollback()
			return sent, err
		}
		log.Infof("Forgot %d push device(s) of user %d that no longer exist", len(gone), userID)
	}
	if len(reached) > 0 {
		now := time.Now()
		if _, err := s.In("id", reached).Cols("last_sent").Update(&Device{LastSent: &now}); err != nil {
			_ = s.Rollback()
			return sent, err
		}
	}
	return sent, s.Commit()
}

// topic turns a tag into a Topic header (at most 32 URL-safe characters),
// so a push still waiting on the push service is replaced, not stacked.
func topic(tag string) string {
	out := make([]rune, 0, 32)
	for _, r := range tag {
		if len(out) == 32 {
			break
		}
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			out = append(out, r)
		}
	}
	return string(out)
}

// SendTest pushes a test notification to the caller's devices.
func SendTest(ctx context.Context, userID int64) (int, error) {
	return Send(ctx, userID, &Message{
		Title: "Mandarin",
		Body:  "Notifications work on this device. 🍊",
		URL:   "/user/settings/household-notifications",
		Tag:   "test",
		ttl:   time.Hour,
	})
}
