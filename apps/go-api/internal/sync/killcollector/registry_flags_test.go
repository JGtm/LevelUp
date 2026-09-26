package killcollector

// registry_flags_test.go — LA TABLE DE CORRESPONDANCE outcome -> marqueur.
//
// Elle est PURE, donc testable sans base, et c est la moitie du sens du demenagement du
// 2026-09-01 : dire quel etat du film justifie quel marqueur, et surtout lesquels n en
// justifient AUCUN. Un `default` trop large poserait « film absent » sur un film present mais
// muet, et le retirerait pour toujours des deux rattrapages.

import (
	"testing"

	"levelup/go-api/internal/sync/matchflags"
)

func TestMarquerFilmParOutcome(t *testing.T) {
	cas := []struct {
		nom      string
		outcome  KillSourceOutcome
		morts    int
		bit      int
		aMarquer bool
		pourquoi string
	}{
		{"film absent", OutcomeNoFilm, 0, matchflags.MBitFilmAbsent, true,
			"marqueur TERMINAL : sans lui les rattrapages redemandent le film a vie"},
		{"passe ecrite", OutcomeWritten, 12, matchflags.MBitWeaponKills, true,
			"le detail par arme existe pour ce match"},
		{"passe vide", OutcomeWritten, 0, 0, false,
			"garde bit-honnete : pas de ligne, pas de bit (sinon bit menteur)"},
		{"film muet", OutcomeNoKillFeed, 0, 0, false,
			"le film EXISTE : le marquer absent le retirerait a tort des rattrapages"},
		{"abandon delai", OutcomeTimeout, 0, 0, false,
			"etat transitoire : le match doit rester candidat"},
		{"capability absente", OutcomeNotSupported, 0, 0, false,
			"hors titre : rien a affirmer sur son film"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			bit, aMarquer := marquerFilmParOutcome(c.outcome, c.morts)
			if aMarquer != c.aMarquer {
				t.Fatalf("aMarquer = %v, attendu %v — %s", aMarquer, c.aMarquer, c.pourquoi)
			}
			if aMarquer && bit != c.bit {
				t.Errorf("bit = %d, attendu %d — %s", bit, c.bit, c.pourquoi)
			}
		})
	}
}
