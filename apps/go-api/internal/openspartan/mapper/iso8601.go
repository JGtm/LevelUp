package mapper

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// iso8601DurationRe matches the subset of ISO 8601 duration the Halo API
// uses: `PT[<H>H][<M>M][<S>S]`, where the seconds component may carry a
// decimal fraction. Days/weeks/years are not observed and not supported.
//
// Examples seen in real responses:
//
//	PT11M59.2855382S
//	PT11M59.25S
//	PT1M8.2S
//	PT46S
//	PT10M
//	PT1H30M
//
// Returns ErrInvalidDuration when the input does not match the expected
// shape or carries no time component at all.
var iso8601DurationRe = regexp.MustCompile(
	`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?$`)

// ParseISO8601Duration parses a Halo-flavoured ISO 8601 duration string into
// a time.Duration. An empty input returns (0, nil) — many payloads omit
// optional duration fields.
func ParseISO8601Duration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	m := iso8601DurationRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("%w: %q", ErrInvalidDuration, s)
	}
	hours, minutes, seconds := m[1], m[2], m[3]
	if hours == "" && minutes == "" && seconds == "" {
		return 0, fmt.Errorf("%w: %q has no time component", ErrInvalidDuration, s)
	}
	var total time.Duration
	if hours != "" {
		h, err := strconv.Atoi(hours)
		if err != nil {
			return 0, fmt.Errorf("%w: hours %q: %v", ErrInvalidDuration, hours, err)
		}
		total += time.Duration(h) * time.Hour
	}
	if minutes != "" {
		mi, err := strconv.Atoi(minutes)
		if err != nil {
			return 0, fmt.Errorf("%w: minutes %q: %v", ErrInvalidDuration, minutes, err)
		}
		total += time.Duration(mi) * time.Minute
	}
	if seconds != "" {
		sec, err := strconv.ParseFloat(seconds, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: seconds %q: %v", ErrInvalidDuration, seconds, err)
		}
		total += time.Duration(sec * float64(time.Second))
	}
	return total, nil
}

// DurationSeconds is a convenience wrapper that returns the duration in
// whole seconds (rounded down), suitable for the `INTEGER` columns the v6
// schema uses (duration_seconds, playable_duration_seconds,
// time_played_seconds).
func DurationSeconds(s string) (int, error) {
	d, err := ParseISO8601Duration(s)
	if err != nil {
		return 0, err
	}
	return int(d / time.Second), nil
}

// DurationSecondsFloat returns the duration in fractional seconds, suitable
// for FLOAT columns (avg_life_seconds).
func DurationSecondsFloat(s string) (float64, error) {
	d, err := ParseISO8601Duration(s)
	if err != nil {
		return 0, err
	}
	return float64(d) / float64(time.Second), nil
}
