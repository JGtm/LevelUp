package port

import (
	"errors"
	"testing"
)

func TestWeaponKillFilters_Validate(t *testing.T) {
	t.Parallel()

	t.Run("valid with MatchIDs+Gamertag", func(t *testing.T) {
		t.Parallel()
		f := WeaponKillFilters{MatchIDs: []string{"m1"}, Gamertag: "main"}
		if err := f.Validate(); err != nil {
			t.Errorf("want nil, got %v", err)
		}
	})

	t.Run("valid with MatchIDs+XUIDs", func(t *testing.T) {
		t.Parallel()
		f := WeaponKillFilters{MatchIDs: []string{"m1"}, XUIDs: []string{"x_p1"}}
		if err := f.Validate(); err != nil {
			t.Errorf("want nil, got %v", err)
		}
	})

	t.Run("rejects no MatchIDs", func(t *testing.T) {
		t.Parallel()
		f := WeaponKillFilters{Gamertag: "main"}
		err := f.Validate()
		if !errors.Is(err, ErrWeaponKillFiltersTooBroad) {
			t.Errorf("want TooBroad, got %v", err)
		}
	})

	t.Run("rejects no Gamertag/XUIDs", func(t *testing.T) {
		t.Parallel()
		f := WeaponKillFilters{MatchIDs: []string{"m1"}}
		err := f.Validate()
		if !errors.Is(err, ErrWeaponKillFiltersTooBroad) {
			t.Errorf("want TooBroad, got %v", err)
		}
	})

	t.Run("rejects negative MinKills", func(t *testing.T) {
		t.Parallel()
		f := WeaponKillFilters{
			MatchIDs: []string{"m1"}, Gamertag: "main", MinKills: -1,
		}
		err := f.Validate()
		if !errors.Is(err, ErrWeaponKillFiltersInvalid) {
			t.Errorf("want Invalid, got %v", err)
		}
	})
}
