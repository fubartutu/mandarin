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
	"bytes"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/household/shopping"
	"code.vikunja.io/api/pkg/log"
	"code.vikunja.io/api/pkg/user"

	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

var (
	user1 = &user.User{ID: 1}
	user2 = &user.User{ID: 2}
)

func TestMain(m *testing.M) {
	log.InitLogger()
	config.InitDefaultConfig()
	config.ServicePublicURL.Set("https://mandarin.example.com/")
	user.InitTests()
	x, err := db.CreateTestEngine()
	if err != nil {
		log.Fatal(err)
	}
	if err := x.Sync2(append(GetTables(), shopping.GetTables()...)...); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func setup(t *testing.T) *xorm.Session {
	t.Helper()
	require.NoError(t, db.LoadFixtures())
	s := db.NewSession()
	t.Cleanup(func() { s.Close() })
	for _, table := range []string{"push_devices", "push_preferences", "push_vapid_keys", "push_shopping_pending"} {
		_, err := s.Exec("DELETE FROM " + table)
		require.NoError(t, err)
	}
	require.NoError(t, s.Commit())
	return db.NewSession()
}

// fakePushService records pushes and answers with status for each endpoint.
type fakePushService struct {
	mu       sync.Mutex
	requests []*http.Request
	bodies   [][]byte
	status   map[string]int
}

func (f *fakePushService) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	body, _ := io.ReadAll(r.Body)
	f.requests = append(f.requests, r)
	f.bodies = append(f.bodies, body)
	code := http.StatusCreated
	if c, ok := f.status[r.URL.String()]; ok {
		code = c
	}
	return &http.Response{StatusCode: code, Body: io.NopCloser(bytes.NewReader(nil))}, nil
}

func fakeService(t *testing.T) *fakePushService {
	t.Helper()
	f := &fakePushService{status: map[string]int{}}
	original := httpClient
	httpClient = f
	t.Cleanup(func() { httpClient = original })
	return f
}

// A real browser key pair for the subscription (from webpush-go's own tests).
const (
	testP256dh = "BNNL5ZaTfK81qhXOx23-wewhigUeFb632jN6LvRWCFH1ubQr77FE_9qV1FuojuRmHP42zmf34rXgW80OvUVDgTk"
	testAuth   = "zqbxT6JKstKSY9JKibZLSQ"
)

func addDevice(t *testing.T, s *xorm.Session, u *user.User, endpoint string) *Device {
	t.Helper()
	in := &DeviceInput{Endpoint: endpoint, Name: "iPhone · Safari"}
	in.Keys.P256dh, in.Keys.Auth = testP256dh, testAuth
	d, err := SaveDevice(s, u, in)
	require.NoError(t, err)
	require.NoError(t, s.Commit())
	return d
}
