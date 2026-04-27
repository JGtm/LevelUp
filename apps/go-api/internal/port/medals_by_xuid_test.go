package port

import (
	"errors"
	"testing"
)

func TestMedalsByXUIDFilters_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid with MatchIDs+XUIDs", func(t *testing.T) {
		t.Parallel()
		f := MedalsByXUIDFilters{MatchIDs: []string{"m1"}, XUIDs: []string{"x1"}}
		if err := f.Validate(); err != nil {
			t.Errorf("want nil, got %v", err)
		}
	})

	t.Run("rejects no MatchIDs", func(t *testing.T) {
		t.Parallel()
		f := MedalsByXUIDFilters{XUIDs: []string{"x1"}}
		err := f.Validate()
		if !errors.Is(err, ErrMedalsByXUIDFiltersTooBroad) {
			t.Errorf("want TooBroad, got %v", err)
		}
	})

	t.Run("rejects no XUIDs", func(t *testing.T) {
		t.Parallel()
		f := MedalsByXUIDFilters{MatchIDs: []string{"m1"}}
		err := f.Validate()
		if !errors.Is(err, ErrMedalsByXUIDFiltersTooBroad) {
			t.Errorf("want TooBroad, got %v", err)
		}
	})

	t.Run("rejects negative Limit", func(t *testing.T) {
		t.Parallel()
		f := MedalsByXUIDFilters{
			MatchIDs: []string{"m1"}, XUIDs: []string{"x1"}, Limit: -1,
		}
		err := f.Validate()
		if !errors.Is(err, ErrMedalsByXUIDFiltersInvalid) {
			t.Errorf("want Invalid, got %v", err)
		}
	})
}
