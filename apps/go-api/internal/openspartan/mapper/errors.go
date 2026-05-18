package mapper

import "errors"

var (
	// ErrInvalidMatch is returned when a ParsedMatch is missing fundamental
	// fields that make registry/participant mapping impossible (e.g. empty
	// MatchID, no Players).
	ErrInvalidMatch = errors.New("mapper: invalid match payload")

	// ErrFutureMatch is returned when MatchInfo.StartTime is in the future,
	// which indicates corruption in the source database.
	ErrFutureMatch = errors.New("mapper: match start_time is in the future")

	// ErrInvalidDuration is returned when an ISO 8601 duration string cannot
	// be parsed.
	ErrInvalidDuration = errors.New("mapper: invalid ISO 8601 duration")
)
