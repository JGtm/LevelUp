package analysis

// match_range_elevation_test.go — LE DÉNIVELÉ du profil de portée (proposition E1,
// 2026-09-22).
//
// Ce que ces tests verrouillent :
//
//  1. la médiane de dénivelé du lobby se calcule SUR LES FRAGS, jamais comme la moyenne des
//     médianes par joueur — la même règle que la médiane de portée ;
//  2. le SIGNE n'est pas redressé : `killer_z - victim_z` positif = frag depuis le haut ;
//  3. `elevation_lobby_delta_m` est bien `joueur - lobby`.

import (
	"math"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

func mreKill(killer string, timeMS int64, dist, dz float64) MeasuredKill {
	return MeasuredKill{
		MatchID: "m1", KillerXUID: killer, TimeMS: timeMS,
		Side: SideKiller, DistanceM: dist, DeltaZ: dz,
	}
}

func mreProfil(t *testing.T, kills []MeasuredKill) domain.MatchRangeProfile {
	t.Helper()
	out := MatchRangeProfiles(MatchRangeInput{
		Kills:        kills,
		Matches:      []MatchRangeMatch{{MatchID: "m1", PlayedAt: time.Now().UTC()}},
		Publish:      map[string]string{"a": "A"},
		PublishOrder: []string{"a"},
	})
	if len(out) != 1 || len(out[0].Players) != 1 {
		t.Fatalf("profils = %+v, want un profil et un joueur", out)
	}
	return out[0]
}

func TestMatchRangeProfiles_DeniveleSurLesFragsEtSigne(t *testing.T) {
	// « a » frague quatre fois DEPUIS LE HAUT (+4, +6 -> médiane +5) ; « b » trois fois
	// depuis le bas (-10). Le lobby porte 5 frags : -10,-10,-10,+4,+6 -> médiane -10. Une
	// moyenne des médianes par joueur aurait rendu -2,5 : c'est exactement ce que ce test
	// interdit.
	p := mreProfil(t, []MeasuredKill{
		mreKill("a", 1, 10, 4),
		mreKill("a", 2, 10, 6),
		mreKill("b", 3, 30, -10),
		mreKill("b", 4, 30, -10),
		mreKill("b", 5, 30, -10),
	})
	if p.LobbyElevationMedianM == nil {
		t.Fatal("lobby_elevation_median_m nil, want une mediane")
	}
	if math.Abs(*p.LobbyElevationMedianM-(-10)) > 1e-9 {
		t.Errorf("lobby_elevation_median_m = %v, want -10 (mediane SUR LES FRAGS)",
			*p.LobbyElevationMedianM)
	}
	j := p.Players[0]
	if j.ElevationMedianM == nil || math.Abs(*j.ElevationMedianM-5) > 1e-9 {
		t.Fatalf("elevation_median_m = %v, want +5 (frag depuis le haut, signe conserve)",
			j.ElevationMedianM)
	}
	if j.ElevationLobbyDeltaM == nil || math.Abs(*j.ElevationLobbyDeltaM-15) > 1e-9 {
		t.Errorf("elevation_lobby_delta_m = %v, want +15 (joueur - lobby)",
			j.ElevationLobbyDeltaM)
	}
}

func TestMatchRangeProfiles_DeniveleAPlatResteUneMesure(t *testing.T) {
	// Tout le monde au même niveau : 0 m est une MESURE (« il frague à plat »), pas une
	// absence — les trois champs sont servis à zéro.
	p := mreProfil(t, []MeasuredKill{
		mreKill("a", 1, 12, 0),
		mreKill("a", 2, 14, 0),
	})
	j := p.Players[0]
	for nom, got := range map[string]*float64{
		"lobby_elevation_median_m": p.LobbyElevationMedianM,
		"elevation_median_m":       j.ElevationMedianM,
		"elevation_lobby_delta_m":  j.ElevationLobbyDeltaM,
	} {
		if got == nil {
			t.Errorf("%s = nil, want 0 (a plat est une mesure)", nom)
			continue
		}
		if *got != 0 {
			t.Errorf("%s = %v, want 0", nom, *got)
		}
	}
}
