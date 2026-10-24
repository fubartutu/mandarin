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

package household

import (
	"testing"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/models"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsMember(t *testing.T) {
	assert.True(t, IsMember(&user.User{ID: 1}))
	assert.False(t, IsMember(&models.LinkSharing{ID: 1}))
	assert.False(t, IsMember(nil))
	assert.False(t, IsMember(&user.User{}), "an unsaved user is not a member")
}

func TestRequireMember(t *testing.T) {
	require.NoError(t, RequireMember(&user.User{ID: 1}))
	require.ErrorIs(t, RequireMember(&models.LinkSharing{ID: 1}), models.ErrGenericForbidden{})
	require.ErrorIs(t, RequireMember(nil), models.ErrGenericForbidden{})
}

func TestLimitFromPage(t *testing.T) {
	previous := config.ServiceMaxItemsPerPage.GetInt()
	config.ServiceMaxItemsPerPage.Set(50)
	t.Cleanup(func() { config.ServiceMaxItemsPerPage.Set(previous) })

	for _, c := range []struct {
		name                string
		page, perPage       int
		wantLimit, wantFrom int
	}{
		{
			name: "page 0 means everything",
		},
		{
			name:    "a negative page means everything",
			page:    -1,
			perPage: 10,
		},
		{
			name:      "the server default per page",
			page:      1,
			wantLimit: 50,
		},
		{
			name:      "the asked-for page size",
			page:      3,
			perPage:   10,
			wantLimit: 10,
			wantFrom:  20,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			limit, start := LimitFromPage(c.page, c.perPage)
			assert.Equal(t, c.wantLimit, limit)
			assert.Equal(t, c.wantFrom, start)
		})
	}
}
