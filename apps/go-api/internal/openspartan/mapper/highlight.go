package mapper

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"levelup/go-api/internal/openspartan"
)

// MapHighlight projects one OpenSpartan HighlightEvents.ResponseBody row into
// a HighlightEventRow.
//
// In real OpenSpartan databases the `xuid` field of a highlight event is
// emitted as a JSON number, not the API's `xuid(<digits>)` wrapper. The
// parser tolerates either form.
//
// The raw JSON is always preserved in RawJSON so downstream queries can
// reach fields the row type doesn't surface (e.g. `medal_value`, `medal_name`,
// `is_medal`).
//
// Returns ErrInvalidMatch when matchID is empty, or a wrapped error when the
// payload cannot be parsed at all.
func MapHighlight(matchID string, rawJSON []byte) (HighlightEventRow, error) {
	if strings.TrimSpace(matchID) == "" {
		return HighlightEventRow{}, fmt.Errorf("%w: missing matchID", ErrInvalidMatch)
	}
	if len(rawJSON) == 0 {
		return HighlightEventRow{}, fmt.Errorf("%w: empty highlight payload", ErrInvalidMatch)
	}
	var probe struct {
		EventType string          `json:"event_type"`
		TimeMs    *int            `json:"time_ms"`
		XUID      json.RawMessage `json:"xuid"`
		TypeHint  *int            `json:"type_hint"`
	}
	if err := json.Unmarshal(rawJSON, &probe); err != nil {
		return HighlightEventRow{}, fmt.Errorf("unmarshal highlight: %w", err)
	}
	if strings.TrimSpace(probe.EventType) == "" {
		return HighlightEventRow{}, fmt.Errorf("%w: missing event_type", ErrInvalidMatch)
	}

	row := HighlightEventRow{
		MatchID:   matchID,
		EventType: probe.EventType,
		TimeMs:    probe.TimeMs,
		TypeHint:  probe.TypeHint,
		RawJSON:   string(rawJSON),
	}
	if xuid := decodeHighlightXUID(probe.XUID); xuid != "" {
		row.XUID = &xuid
	}
	return row, nil
}

// decodeHighlightXUID accepts the various shapes the xuid field has been
// observed under and returns the bare-digit XUID, or "" if none matched.
//
// Shapes:
//   - JSON number:           2533274945467756
//   - JSON string of digits: "2533274945467756"
//   - JSON string wrapped:   "xuid(2533274945467756)"
func decodeHighlightXUID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// Try as a string first (quoted in the JSON).
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return openspartan.ParseXUID(asString)
	}
	// Fall back to a number — int64 to fit Xbox 64-bit XUIDs.
	var asInt int64
	if err := json.Unmarshal(raw, &asInt); err == nil {
		if asInt <= 0 {
			return ""
		}
		return openspartan.ParseXUID(strconv.FormatInt(asInt, 10))
	}
	return ""
}
