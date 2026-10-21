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

// Package backup writes Mandarin's nightly backups: the SQLite database and
// the uploaded files (photos, recipe pictures, attachments) in one dated
// archive, mandarin-YYYY-MM-DD.tar.gz, with old archives pruned.
//
// Settings (environment):
//
//	MANDARIN_BACKUP_DIR        where to write; backups are off without it
//	MANDARIN_BACKUP_KEEP_DAYS  how many days to keep; default 14
//	MANDARIN_BACKUP_AT         when, as HH:MM; default 03:00. In Mandarin's time zone
//	                           (VIKUNJA_SERVICE_TIMEZONE, e.g. Australia/Sydney), else UTC.
//
// To restore: stop Mandarin, put the archive's vikunja.db in place of the
// database (and delete any vikunja.db-wal and -shm beside it), copy files/
// over the files folder, and start Mandarin again.
package backup

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"code.vikunja.io/api/pkg/config"
	"code.vikunja.io/api/pkg/cron"
	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/log"
)

// Settings is what a backup run needs.
type Settings struct {
	Dir      string
	KeepDays int
	Hour     int
	Minute   int
}

// FromEnv reads the settings. ok is false when backups are off.
func FromEnv() (s Settings, ok bool, err error) {
	s = Settings{Dir: strings.TrimSpace(os.Getenv("MANDARIN_BACKUP_DIR")), KeepDays: 14, Hour: 3}
	if s.Dir == "" {
		return s, false, nil
	}
	if v := strings.TrimSpace(os.Getenv("MANDARIN_BACKUP_KEEP_DAYS")); v != "" {
		if s.KeepDays, err = strconv.Atoi(v); err != nil || s.KeepDays < 1 {
			return s, false, fmt.Errorf("MANDARIN_BACKUP_KEEP_DAYS must be a whole number of days, got %q", v)
		}
	}
	if v := strings.TrimSpace(os.Getenv("MANDARIN_BACKUP_AT")); v != "" {
		at, perr := time.Parse("15:04", v)
		if perr != nil {
			return s, false, fmt.Errorf("MANDARIN_BACKUP_AT must look like 03:00, got %q", v)
		}
		s.Hour, s.Minute = at.Hour(), at.Minute()
	}
	return s, true, nil
}

// running stops two backups overlapping (the nightly one and a catch-up).
var running sync.Mutex

// RegisterCron schedules the nightly backup, and makes one shortly after
// start-up when there's none from today (a new install, or the server was
// off at backup time). Call after cron.Init.
func RegisterCron() {
	settings, ok, err := FromEnv()
	if err != nil {
		log.Errorf("Mandarin backups are off: %s", err)
		return
	}
	if !ok {
		log.Info("Mandarin backups are off: set MANDARIN_BACKUP_DIR to turn them on.")
		return
	}
	if config.DatabaseType.GetString() != "sqlite" {
		log.Errorf("Mandarin backups only support SQLite, not %s.", config.DatabaseType.GetString())
		return
	}
	run := func() {
		if _, err := Run(settings, time.Now().In(config.GetTimeZone())); err != nil {
			log.Errorf("Mandarin backup failed: %s", err)
		}
	}
	// CRON_TZ: the time is in Mandarin's time zone (service.timezone), not
	// the container's, which is usually UTC.
	zone := config.GetTimeZone().String()
	if err := cron.Schedule(fmt.Sprintf("CRON_TZ=%s %d %d * * *", zone, settings.Minute, settings.Hour), run); err != nil {
		log.Errorf("Could not schedule Mandarin backups: %s", err)
		return
	}
	go func() {
		time.Sleep(time.Minute) // let start-up (and migrations) settle
		today := filepath.Join(settings.Dir, archiveName(time.Now().In(config.GetTimeZone())))
		if _, err := os.Stat(today); errors.Is(err, fs.ErrNotExist) {
			run()
		}
	}()
	log.Infof("Mandarin backups: nightly at %02d:%02d (%s) into %s, kept %d days.", settings.Hour, settings.Minute, zone, settings.Dir, settings.KeepDays)
}

func archiveName(day time.Time) string {
	return "mandarin-" + day.Format("2006-01-02") + ".tar.gz"
}

var archivePattern = regexp.MustCompile(`^mandarin-(\d{4}-\d{2}-\d{2})\.tar\.gz$`)

// Run writes today's archive (replacing one from earlier today) and prunes
// old ones. It returns the archive's path.
func Run(settings Settings, now time.Time) (string, error) {
	running.Lock()
	defer running.Unlock()

	if err := os.MkdirAll(settings.Dir, 0o750); err != nil {
		return "", err
	}
	// A consistent copy of the live database: SQLite's VACUUM INTO reads it in
	// one transaction while Mandarin keeps running. Copying the file itself
	// could catch it half-written.
	snapshot := filepath.Join(settings.Dir, ".mandarin-backup.db")
	_ = os.Remove(snapshot)
	defer os.Remove(snapshot)
	// Straight on the engine: db.NewSession opens a transaction, and SQLite
	// can't VACUUM inside one. CreateDBEngine returns the running engine.
	engine, err := db.CreateDBEngine()
	if err != nil {
		return "", err
	}
	if _, err := engine.Exec("VACUUM INTO ?", snapshot); err != nil {
		return "", fmt.Errorf("copying the database: %w", err)
	}

	final := filepath.Join(settings.Dir, archiveName(now))
	partial := final + ".part"
	if err := writeArchive(partial, snapshot, filesDir()); err != nil {
		_ = os.Remove(partial)
		return "", err
	}
	if err := os.Rename(partial, final); err != nil {
		return "", err
	}

	if info, err := os.Stat(final); err == nil {
		log.Infof("Mandarin backup written: %s (%.1f MB)", final, float64(info.Size())/1e6)
	}
	return final, prune(settings, now)
}

// filesDir is the uploaded files' folder, or "" when they're not stored on
// this machine (S3).
func filesDir() string {
	if config.FilesType.GetString() != "local" {
		return ""
	}
	return config.FilesBasePath.GetString()
}

// writeArchive packs the database copy as vikunja.db and the files folder
// as files/ into a .tar.gz.
func writeArchive(path, database, files string) error {
	out, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	defer out.Close()
	gz := gzip.NewWriter(out)
	tw := tar.NewWriter(gz)

	if err := addFile(tw, database, "vikunja.db"); err != nil {
		return err
	}
	if files != "" {
		err := filepath.WalkDir(files, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.Type().IsRegular() {
				return nil // folders are implied by the paths; skip anything else
			}
			rel, err := filepath.Rel(files, p)
			if err != nil {
				return err
			}
			return addFile(tw, p, "files/"+filepath.ToSlash(rel))
		})
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("packing the files: %w", err)
		}
	}

	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	return out.Close()
}

func addFile(tw *tar.Writer, path, name string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = name
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	_, err = io.Copy(tw, f)
	return err
}

// prune deletes archives older than KeepDays. Only files named like ours
// are touched, so the folder can hold other things.
func prune(settings Settings, now time.Time) error {
	entries, err := os.ReadDir(settings.Dir)
	if err != nil {
		return err
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	oldest := today.AddDate(0, 0, -(settings.KeepDays - 1))
	for _, e := range entries {
		m := archivePattern.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		day, err := time.ParseInLocation("2006-01-02", m[1], now.Location())
		if err != nil || !day.Before(oldest) {
			continue
		}
		if err := os.Remove(filepath.Join(settings.Dir, e.Name())); err != nil {
			return err
		}
		log.Infof("Mandarin backup removed (older than %d days): %s", settings.KeepDays, e.Name())
	}
	return nil
}
