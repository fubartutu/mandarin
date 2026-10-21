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

package backup

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm"
)

func TestMain(m *testing.M) {
	log.InitLogger()
	config.InitDefaultConfig()
	if _, err := db.CreateTestEngine(); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

// unpack reads an archive into name → contents.
func unpack(t *testing.T, path string) map[string][]byte {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	tr := tar.NewReader(gz)
	out := map[string][]byte{}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return out
		}
		require.NoError(t, err)
		data, err := io.ReadAll(tr)
		require.NoError(t, err)
		out[h.Name] = data
	}
}

func TestRun(t *testing.T) {
	// Something to find in the copy.
	s := db.NewSession()
	_, err := s.Exec("CREATE TABLE IF NOT EXISTS backup_probe (note TEXT)")
	require.NoError(t, err)
	_, err = s.Exec("INSERT INTO backup_probe VALUES ('milk')")
	require.NoError(t, err)
	require.NoError(t, s.Commit())
	s.Close()

	files := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(files, "12"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(files, "12", "34"), []byte("photo"), 0o600))
	config.FilesType.Set("local")
	config.FilesBasePath.Set(files)

	dir := t.TempDir()
	settings := Settings{Dir: dir, KeepDays: 14}
	now := time.Date(2026, 10, 22, 3, 0, 0, 0, time.UTC)

	// Old archives: one just inside the 14 days, one just outside, and a stranger.
	for _, name := range []string{"mandarin-2026-10-09.tar.gz", "mandarin-2026-10-08.tar.gz", "notes.txt"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600))
	}

	path, err := Run(settings, now)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "mandarin-2026-10-22.tar.gz"), path)

	contents := unpack(t, path)
	assert.Equal(t, []byte("photo"), contents["files/12/34"])
	require.Contains(t, contents, "vikunja.db")

	// The database copy opens and has the row.
	copyPath := filepath.Join(t.TempDir(), "copy.db")
	require.NoError(t, os.WriteFile(copyPath, contents["vikunja.db"], 0o600))
	engine, err := xorm.NewEngine("sqlite3", copyPath)
	require.NoError(t, err)
	defer engine.Close()
	var note string
	_, err = engine.SQL("SELECT note FROM backup_probe").Get(&note)
	require.NoError(t, err)
	assert.Equal(t, "milk", note)

	left, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := []string{}
	for _, e := range left {
		names = append(names, e.Name())
	}
	assert.ElementsMatch(t, []string{"mandarin-2026-10-09.tar.gz", "mandarin-2026-10-22.tar.gz", "notes.txt"}, names,
		"older than 14 days pruned; the snapshot and other files left alone")

	// Running again the same day replaces the archive.
	_, err = Run(settings, now.Add(time.Hour))
	require.NoError(t, err)
}

func TestFromEnv(t *testing.T) {
	t.Setenv("MANDARIN_BACKUP_DIR", "")
	_, ok, err := FromEnv()
	require.NoError(t, err)
	assert.False(t, ok, "off without a folder")

	t.Setenv("MANDARIN_BACKUP_DIR", "/backups")
	t.Setenv("MANDARIN_BACKUP_KEEP_DAYS", "7")
	t.Setenv("MANDARIN_BACKUP_AT", "02:30")
	s, ok, err := FromEnv()
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, Settings{Dir: "/backups", KeepDays: 7, Hour: 2, Minute: 30}, s)

	t.Setenv("MANDARIN_BACKUP_AT", "3am")
	_, _, err = FromEnv()
	assert.Error(t, err)
}
