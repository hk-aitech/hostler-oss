// Package dblock — SQLite lock retry helper.
//
// Lives in a location that both pkg/db and internal/adapters/sqlite can
// import (no reverse dependency). Adds a buffer on top of
// modernc.org/sqlite's busy_timeout(5000ms) to absorb transient races.
//
// Usage:
//
//	res, err := dblock.Exec(s.db, query, args...)
//	rows, err := dblock.Query(s.db, query, args...)
package dblock

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	sqlitelib "modernc.org/sqlite"
	sqlite3lib "modernc.org/sqlite/lib"
)

const (
	// envKey — opt-out env var. value=off disables retries (fail immediately).
	envKey = "HSTL_DB_LOCK_RETRY"
	// retryMax — maximum retry attempts.
	retryMax = 3
)

// retryDelays — exponential backoff. 100ms / 250ms / 500ms = ~850ms total.
var retryDelays = []time.Duration{
	100 * time.Millisecond,
	250 * time.Millisecond,
	500 * time.Millisecond,
}

// IsDisabled reports whether retries are disabled via the env opt-out.
func IsDisabled() bool {
	return strings.EqualFold(os.Getenv(envKey), "off")
}

// IsBusy reports whether err is a SQLite busy/lock error.
func IsBusy(err error) bool {
	if err == nil {
		return false
	}
	var sqliteErr *sqlitelib.Error
	if errors.As(err, &sqliteErr) {
		c := sqliteErr.Code()
		if c == sqlite3lib.SQLITE_BUSY || c == sqlite3lib.SQLITE_LOCKED {
			return true
		}
	}
	msg := err.Error()
	return strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "SQLITE_BUSY") ||
		strings.Contains(msg, "database table is locked")
}

// Exec calls database.Exec with sleep+retry on SQLITE_BUSY.
//
// On retry, prints a single WARN line on stderr (keeps JSON stdout clean).
// Returns the last error after retryMax failed attempts.
//
// Opt-out: env HSTL_DB_LOCK_RETRY=off → fail immediately.
func Exec(database *sql.DB, query string, args ...any) (sql.Result, error) {
	if IsDisabled() {
		return database.Exec(query, args...)
	}
	var lastErr error
	for attempt := 0; attempt <= retryMax; attempt++ {
		res, err := database.Exec(query, args...)
		if err == nil {
			if attempt > 0 {
				fmt.Fprintf(os.Stderr,
					"[db] lock retry success — attempt %d/%d\n",
					attempt, retryMax,
				)
			}
			return res, nil
		}
		if !IsBusy(err) {
			return nil, err
		}
		lastErr = err
		if attempt < retryMax {
			delay := retryDelays[attempt]
			fmt.Fprintf(os.Stderr,
				"[db] lock retry %d/%d — sleeping %v: %v\n",
				attempt+1, retryMax, delay, err,
			)
			time.Sleep(delay)
		}
	}
	return nil, fmt.Errorf("DB lock not absorbed (all %d retries locked): %w — check for other hstl processes or retry shortly",
		retryMax, lastErr)
}

// Query calls database.Query with sleep+retry on SQLITE_BUSY.
func Query(database *sql.DB, query string, args ...any) (*sql.Rows, error) {
	if IsDisabled() {
		return database.Query(query, args...)
	}
	var lastErr error
	for attempt := 0; attempt <= retryMax; attempt++ {
		rows, err := database.Query(query, args...)
		if err == nil {
			if attempt > 0 {
				fmt.Fprintf(os.Stderr,
					"[db] lock retry success — attempt %d/%d\n",
					attempt, retryMax,
				)
			}
			return rows, nil
		}
		if !IsBusy(err) {
			return nil, err
		}
		lastErr = err
		if attempt < retryMax {
			delay := retryDelays[attempt]
			fmt.Fprintf(os.Stderr,
				"[db] lock retry %d/%d — sleeping %v: %v\n",
				attempt+1, retryMax, delay, err,
			)
			time.Sleep(delay)
		}
	}
	return nil, fmt.Errorf("DB lock not absorbed (all %d retries locked): %w — check for other hstl processes or retry shortly",
		retryMax, lastErr)
}
