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
	"sort"
	"strings"
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/notifications"
	"code.vikunja.io/api/pkg/user"

	"github.com/SherClockHolmes/webpush-go"
)

func hookNotifications() {
	notifications.SetHouseholdHook(func(notifiable notifications.Notifiable, n notifications.Notification) {
		msg, wants := messageFor(n)
		if msg == nil {
			return // not something that pushes
		}
		userID := notifiable.RouteForDB()
		// In the background: Vikunja's cron holds a database session, and a
		// push service can take a while to answer.
		go func() {
			s := db.NewSession()
			prefs, err := preferencesFor(s, userID)
			_ = s.Commit()
			s.Close()
			if err != nil {
				log.Errorf("Push: could not read preferences of user %d: %s", userID, err)
				return
			}
			if !wants(prefs) {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			if _, err := Send(ctx, userID, msg); err != nil {
				log.Errorf("Push to user %d failed: %s", userID, err)
			}
		}()
	})
}

// messageFor turns the notifications that push into a message, and says
// which preference turns it on. Everything else returns nil.
func messageFor(n notifications.Notification) (*Message, func(*Preferences) bool) {
	switch n := n.(type) {
	case *models.ReminderDueNotification:
		if n.Task == nil {
			return nil, nil
		}
		return reminderMessage(n.Task, n.Project, n.User), func(p *Preferences) bool { return p.Reminders }
	case *models.UndoneTasksOverdueNotification:
		tasks := make([]*models.Task, 0, len(n.Assigned)+len(n.Followed))
		seen := map[int64]bool{}
		for _, group := range []map[int64]*models.Task{n.Assigned, n.Followed} {
			for id, t := range group {
				if !seen[id] {
					seen[id] = true
					tasks = append(tasks, t)
				}
			}
		}
		return overdueMessage(tasks), func(p *Preferences) bool { return p.Overdue }
	case *models.UndoneTaskOverdueNotification:
		if n.Task == nil {
			return nil, nil
		}
		return overdueMessage([]*models.Task{n.Task}), func(p *Preferences) bool { return p.Overdue }
	}
	return nil, nil
}

func reminderMessage(task *models.Task, project *models.Project, u *user.User) *Message {
	parts := []string{}
	if project != nil && project.Title != "" {
		parts = append(parts, project.Title)
	}
	if !task.DueDate.IsZero() {
		parts = append(parts, "due "+when(task.DueDate, u))
	}
	return &Message{
		Title:   task.Title,
		Body:    strings.Join(append([]string{"Reminder"}, parts...), " · "),
		URL:     fmt.Sprintf("/tasks/%d", task.ID),
		Tag:     fmt.Sprintf("task-%d", task.ID),
		ttl:     6 * time.Hour,
		urgency: webpush.UrgencyHigh,
	}
}

// when is a due date as a person reads it in their own time zone:
// "today 3:30pm", "tomorrow 9:00am", "Fri 24 Oct 5:00pm".
func when(t time.Time, u *user.User) string {
	loc := config.GetTimeZone()
	if u != nil && u.Timezone != "" {
		if l, err := time.LoadLocation(u.Timezone); err == nil {
			loc = l
		}
	}
	t = t.In(loc)
	now := time.Now().In(loc)
	clock := strings.ToLower(t.Format("3:04PM"))
	switch day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc); {
	case day.Equal(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)):
		return "today " + clock
	case day.Equal(time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)):
		return "tomorrow " + clock
	default:
		return t.Format("Mon 2 Jan ") + clock
	}
}

func overdueMessage(tasks []*models.Task) *Message {
	if len(tasks) == 0 {
		return nil
	}
	// Longest overdue first.
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].DueDate.Before(tasks[j].DueDate) })
	titles := make([]string, 0, 3)
	for _, t := range tasks[:min(3, len(tasks))] {
		titles = append(titles, t.Title)
	}
	body := strings.Join(titles, ", ")
	if len(tasks) > 3 {
		body += fmt.Sprintf(" and %d more", len(tasks)-3)
	}
	title := fmt.Sprintf("%d overdue tasks", len(tasks))
	url := "/"
	if len(tasks) == 1 {
		title = "1 overdue task"
		url = fmt.Sprintf("/tasks/%d", tasks[0].ID)
	}
	return &Message{Title: title, Body: body, URL: url, Tag: "overdue", ttl: 12 * time.Hour}
}
