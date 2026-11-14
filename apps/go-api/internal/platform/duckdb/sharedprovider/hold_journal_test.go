package sharedprovider

import (
	"testing"
	"time"
)

// Un épisode de dépassements d'une même clé = une seule ligne ; l'épisode suivant (après le
// silence) rend le nombre de dépassements tus ; une autre clé a son propre épisode.
func TestJournalDepassements_UneLigneParEpisode(t *testing.T) {
	horloge := time.Date(2026, 11, 14, 8, 0, 0, 0, time.UTC)
	j := &journalDepassements{now: func() time.Time { return horloge }}

	lignes := 0
	for i := 0; i < 200; i++ {
		horloge = horloge.Add(2 * time.Second)
		if ok, _ := j.signaler("match_recompute/lusr"); ok {
			lignes++
		}
	}
	if lignes != 1 {
		t.Fatalf("200 dépassements en un épisode : %d lignes, attendu 1", lignes)
	}
	if ok, _ := j.signaler("autre"); !ok {
		t.Error("une autre clé doit ouvrir son propre épisode")
	}

	horloge = horloge.Add(episodeSilence + time.Second)
	ok, tus := j.signaler("match_recompute/lusr")
	if !ok || tus != 199 {
		t.Errorf("épisode suivant : journaliser=%v tus=%d, attendu true et 199", ok, tus)
	}
}

// La table d'une clé à cardinalité libre reste bornée.
func TestJournalDepassements_TableBornee(t *testing.T) {
	j := &journalDepassements{}
	for i := 0; i < 3*episodesMax; i++ {
		j.signaler(time.Duration(i).String())
	}
	if len(j.episodes) > episodesMax {
		t.Errorf("table non bornée : %d entrées (max %d)", len(j.episodes), episodesMax)
	}
}

// vider rend les dépassements tus de l'épisode en cours et remet le journal à zéro.
func TestJournalDepassements_ViderRendLesTus(t *testing.T) {
	horloge := time.Date(2026, 11, 14, 8, 0, 0, 0, time.UTC)
	j := &journalDepassements{now: func() time.Time { return horloge }}
	for i := 0; i < 5; i++ {
		horloge = horloge.Add(time.Second)
		j.signaler("a")
	}
	j.signaler("b") // un seul dépassement : rien de tu
	if got := j.vider(); len(got) != 1 || got["a"] != 4 {
		t.Errorf("vider = %v, attendu {a:4}", got)
	}
	if got := j.vider(); len(got) != 0 {
		t.Errorf("second vider = %v, attendu vide", got)
	}
	if ok, _ := j.signaler("a"); !ok {
		t.Error("après vider, le dépassement suivant doit ouvrir un épisode")
	}
}
