package analysis

// camps_canonical.go — MON CAMP ET L'AUTRE CAMP d'une ligne canonique : les deux équipes du match
// (`Summary.Teams`, team_id 0 et 1), mon camp lu sur `Self.TeamID` (0 quand il manque). Source
// unique du score « mon camp d'abord » de l'accueil, des Sessions et du détail d'une zone de
// l'onglet Tactique. Garde-rail : archlint/camps_canonical_single_source_test.go.

import (
	"strings"

	"levelup/go-api/internal/games/canonical"
)

// CampsDuMatch rend mon équipe puis l'autre ; faux quand le match ne porte pas ses deux équipes.
func CampsDuMatch(r canonical.PlayerMatchRow) (mien, autre *canonical.TeamSnapshot, ok bool) {
	var t0, t1 *canonical.TeamSnapshot
	for i := range r.Summary.Teams {
		switch r.Summary.Teams[i].TeamID {
		case 0:
			t0 = &r.Summary.Teams[i]
		case 1:
			t1 = &r.Summary.Teams[i]
		}
	}
	if t0 == nil || t1 == nil {
		return nil, nil, false
	}
	if r.Self.TeamID != nil && *r.Self.TeamID == 1 {
		return t1, t0, true
	}
	return t0, t1, true
}

// EntreeDeScoreCanonique rend l'entrée du score d'équipe d'une ligne canonique, mon camp d'abord ;
// la variante du match décide si le résultat se lit en manches (`roundsDecide`, ADR 0032). Faux
// quand le match ne porte pas ses deux équipes.
func EntreeDeScoreCanonique(r canonical.PlayerMatchRow, roundsDecide map[string]bool) (TeamScoreInput, bool) {
	mien, autre, ok := CampsDuMatch(r)
	if !ok {
		return TeamScoreInput{}, false
	}
	return TeamScoreInput{
		MyPoints: mien.Score, EnemyPoints: autre.Score,
		MyRoundsWon: mien.RoundsWon, EnemyRoundsWon: autre.RoundsWon,
		RoundsTotal:  r.Summary.RoundsTotal,
		RoundsDecide: roundsDecide[strings.TrimSpace(variantNameOf(r))],
	}, true
}
