package port

import (
	"errors"
	"testing"
)

func TestPersonalScoreAwardsFilters_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid avec MatchIDs+XUIDs", func(t *testing.T) {
		t.Parallel()
		f := PersonalScoreAwardsFilters{MatchIDs: []string{"m1"}, XUIDs: []string{"x1"}}
		if err := f.Validate(); err != nil {
			t.Errorf("want nil, got %v", err)
		}
	})

	t.Run("rejette MatchIDs vide", func(t *testing.T) {
		t.Parallel()
		f := PersonalScoreAwardsFilters{XUIDs: []string{"x1"}}
		err := f.Validate()
		if !errors.Is(err, ErrPersonalScoreAwardsFiltersTooBroad) {
			t.Errorf("want TooBroad, got %v", err)
		}
	})

	t.Run("rejette XUIDs vide", func(t *testing.T) {
		t.Parallel()
		f := PersonalScoreAwardsFilters{MatchIDs: []string{"m1"}}
		err := f.Validate()
		if !errors.Is(err, ErrPersonalScoreAwardsFiltersTooBroad) {
			t.Errorf("want TooBroad, got %v", err)
		}
	})
}
