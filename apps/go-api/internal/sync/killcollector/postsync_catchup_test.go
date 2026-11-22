package killcollector

import "testing"

// La borne de films par cycle de la source des frags suit la capacité FilmCatchUp du profil : le garde-fou de
// 200 films sous la capacité (le temps borne la passe), les 8 films d'avant hors d'elle.
func TestWithFilmCatchUp_BorneParCycle(t *testing.T) {
	h := NewPostSyncHook(t.TempDir(), FragsParCycleHeberge)
	if got := h.PerCycle(); got != 8 {
		t.Errorf("borne de départ = %d, attendu 8", got)
	}
	if got := h.WithFilmCatchUp(true).PerCycle(); got != 200 {
		t.Errorf("sous FilmCatchUp : borne = %d, attendu 200 (GardeFouFilmsParCycle)", got)
	}
	if got := h.WithFilmCatchUp(false).PerCycle(); got != FragsParCycleHeberge {
		t.Errorf("hors FilmCatchUp : borne = %d, attendu %d", got, FragsParCycleHeberge)
	}
}
