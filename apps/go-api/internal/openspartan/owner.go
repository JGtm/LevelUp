package openspartan

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// filenameXUIDRegex matches the conventional OpenSpartan filename `{xuid}.db`.
// Xbox Live XUIDs are 64-bit integers, observed at 16–17 decimal digits.
var filenameXUIDRegex = regexp.MustCompile(`^(\d{16,17})\.db$`)

// playerIDXuidRegex matches the API form `xuid(<digits>)` used in
// MatchStats.Players[].PlayerId and PlayerMatchStats.Value[].Id.
var playerIDXuidRegex = regexp.MustCompile(`^xuid\((\d+)\)$`)

// bareDigitsXUIDRegex matches a raw decimal XUID without surrounding wrapper.
var bareDigitsXUIDRegex = regexp.MustCompile(`^\d{16,17}$`)

// extractXUIDFromFilename returns the digit string captured from a hint that
// looks like a path or filename of the form `{xuid}.db`, or an empty string
// when no match is found. Backslashes are normalised to slashes so that paths
// captured on Windows (`C:\users\me\{xuid}.db`) extract correctly when the
// process runs on Linux (where filepath.Base treats only `/` as a separator).
func extractXUIDFromFilename(hint string) string {
	hint = strings.ReplaceAll(hint, "\\", "/")
	base := filepath.Base(hint)
	m := filenameXUIDRegex.FindStringSubmatch(base)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// ParseXUID accepts either the API wrapper form `xuid(<digits>)` or a bare
// 16–17 digit decimal string, and returns the digits on success or "" otherwise.
//
// Exported so sibling packages (e.g. mapper) can extract a bare XUID from
// any of the wrapped forms the Halo API uses.
func ParseXUID(s string) string {
	s = strings.TrimSpace(s)
	if m := playerIDXuidRegex.FindStringSubmatch(s); len(m) >= 2 {
		return m[1]
	}
	if bareDigitsXUIDRegex.MatchString(s) {
		return s
	}
	return ""
}

// DetectOwner runs up to three heuristics to identify the owner XUID:
//
//  1. Filename hint — an OpenSpartan database is conventionally named
//     `{xuid}.db`. When `filenameHint` (or the reader's own path, if hint is
//     empty) matches the pattern, the extracted XUID is a candidate.
//  2. Most-frequent human XUID across all matches — in a personal database
//     the owner appears in nearly every match, so the XUID that shows up the
//     most often among `PlayerType=1` entries is the strongest signal.
//  3. CacheMeta lookup — some grunt builds persist the owner XUID under a
//     key whose name contains "xuid".
//
// The returned Confidence reflects agreement:
//   - High   when filename hint and frequency winner agree on the same XUID;
//   - Medium when a single strong heuristic (frequency, or filename alone)
//     produced a candidate;
//   - Low    when only the CacheMeta fallback produced a candidate.
//
// Returns "" + ConfidenceNone + ErrOwnerUndetected when no heuristic matched.
func (r *Reader) DetectOwner(ctx context.Context, filenameHint string) (xuid string, conf Confidence, err error) {
	r.mu.Lock()
	closed := r.closed
	r.mu.Unlock()
	if closed {
		return "", ConfidenceNone, ErrReaderClosed
	}

	hint := filenameHint
	if hint == "" {
		hint = r.path
	}
	fromFilename := extractXUIDFromFilename(hint)

	fromFrequency, err := r.mostFrequentHumanXUID(ctx)
	if err != nil {
		return "", ConfidenceNone, err
	}

	switch {
	case fromFilename != "" && fromFilename == fromFrequency:
		slog.Info("openspartan: owner detected", "xuid", fromFilename, "confidence", "high")
		return fromFilename, ConfidenceHigh, nil
	case fromFrequency != "":
		slog.Info("openspartan: owner detected", "xuid", fromFrequency, "confidence", "medium", "via", "frequency")
		return fromFrequency, ConfidenceMedium, nil
	case fromFilename != "":
		slog.Info("openspartan: owner detected", "xuid", fromFilename, "confidence", "medium", "via", "filename")
		return fromFilename, ConfidenceMedium, nil
	}

	fromCache, _ := r.xuidFromCacheMeta(ctx)
	if fromCache != "" {
		slog.Info("openspartan: owner detected", "xuid", fromCache, "confidence", "low", "via", "cache_meta")
		return fromCache, ConfidenceLow, nil
	}
	slog.Warn("openspartan: owner detection exhausted all heuristics", "path", r.path)
	return "", ConfidenceNone, ErrOwnerUndetected
}

// mostFrequentHumanXUID scans MatchStats.ResponseBody, extracts every
// human Player (PlayerType=1), and returns the XUID that appears in the
// largest number of distinct matches. Empty string when no human player was
// found across the database (very unusual).
func (r *Reader) mostFrequentHumanXUID(ctx context.Context) (string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT ResponseBody FROM MatchStats`)
	if err != nil {
		return "", fmt.Errorf("openspartan: query MatchStats for owner detection: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var body []byte
		if err := rows.Scan(&body); err != nil {
			return "", fmt.Errorf("openspartan: scan MatchStats row: %w", err)
		}
		// Decode only the Players array — saves memory & CPU on 50+ field payloads.
		var probe struct {
			Players []struct {
				PlayerID   string `json:"PlayerId"`
				PlayerType int    `json:"PlayerType"`
			} `json:"Players"`
		}
		if err := json.Unmarshal(body, &probe); err != nil {
			// Malformed row: skip, do not abort the whole detection.
			continue
		}
		seenInMatch := make(map[string]struct{}, len(probe.Players))
		for _, p := range probe.Players {
			if p.PlayerType != 1 {
				continue
			}
			xuid := ParseXUID(p.PlayerID)
			if xuid == "" {
				continue
			}
			if _, ok := seenInMatch[xuid]; ok {
				continue
			}
			seenInMatch[xuid] = struct{}{}
			counts[xuid]++
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("openspartan: iterate MatchStats: %w", err)
	}
	if len(counts) == 0 {
		return "", nil
	}

	type pair struct {
		xuid  string
		count int
	}
	pairs := make([]pair, 0, len(counts))
	for x, c := range counts {
		pairs = append(pairs, pair{x, c})
	}
	// Highest count wins; deterministic tie-break by lexicographic XUID.
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count != pairs[j].count {
			return pairs[i].count > pairs[j].count
		}
		return pairs[i].xuid < pairs[j].xuid
	})
	return pairs[0].xuid, nil
}

// xuidFromCacheMeta looks for an XUID stored in CacheMeta under any key
// whose name contains "xuid" (case-insensitive). Missing table is treated as
// "no signal" rather than an error — CacheMeta is optional in the schema.
func (r *Reader) xuidFromCacheMeta(ctx context.Context) (string, error) {
	hasTable, err := tableExists(ctx, r.db, "CacheMeta")
	if err != nil {
		return "", err
	}
	if !hasTable {
		return "", nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT key, value FROM CacheMeta`)
	if err != nil {
		return "", fmt.Errorf("openspartan: query CacheMeta: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var key, value sql.NullString
		if err := rows.Scan(&key, &value); err != nil {
			return "", fmt.Errorf("openspartan: scan CacheMeta: %w", err)
		}
		if !key.Valid || !value.Valid {
			continue
		}
		if !strings.Contains(strings.ToLower(key.String), "xuid") {
			continue
		}
		if x := ParseXUID(value.String); x != "" {
			return x, nil
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("openspartan: iterate CacheMeta: %w", err)
	}
	return "", nil
}

// tableExists reports whether a non-temp table with the given name is present.
func tableExists(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var found string
	err := db.QueryRowContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&found)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("openspartan: check table %q: %w", name, err)
	}
	return found == name, nil
}
