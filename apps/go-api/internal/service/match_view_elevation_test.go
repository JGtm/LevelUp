package service

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// viewerKillCount est le DÉNOMINATEUR de la réserve de couverture du bloc Dénivelé. Ce test
// verrouille la distinction qui compte : « pas de ligne » et « pas de compteur » rendent 0,
// et 0 se lit alors « on ne sait pas », jamais « zéro frag ».
func TestViewerKillCount(t *testing.T) {
	if got := viewerKillCount(nil); got != 0 {
		t.Errorf("sans ligne de scoreboard : %d, attendu 0", got)
	}
	if got := viewerKillCount(&domain.MatchScoreboardRow{}); got != 0 {
		t.Errorf("ligne sans compteur de frags : %d, attendu 0", got)
	}
	kills := 14
	if got := viewerKillCount(&domain.MatchScoreboardRow{Kills: &kills}); got != 14 {
		t.Errorf("compteur lu = %d, attendu 14", got)
	}
}
