package grammar

import "testing"

// repli_anticipation_test.go — le repli `repli_liaison_par_anticipation` est COMPTE jusqu a la
// marche du frame-processeur (lot J8.1 du plan de suite d audit, constat GA1-2 : le compte
// existait dans le monde, sans aucun appelant, et l artefact ne le disait pas).

// TestLeRepliDAnticipationEstCompteParLaMarche : un delta d un slot jamais lie, que la table
// anticipee declare dans un chunk POSTERIEUR, est lie par le repli ; la marche en rend le compte.
// Un slot que la table ne declare pas est rejete hors datum et ne compte pas.
//
// MUTATION : `return 0` dans [movementStateScanner.liaisonsDuRepliDAnticipation] — ROUGE.
func TestLeRepliDAnticipationEstCompteParLaMarche(t *testing.T) {
	w := NewWorld(&Registry{Archetypes: []Archetype{{Index: 0}, {Index: 1}, {Index: 2}, {Index: 3}}})
	tab := NouvelleTableAnticipee()
	tab.entrees[cleAnticipee{slot: 77, tete: 1}] = []declarationAnticipee{{chunk: 5, ti: 3}}
	tab.entrees[cleAnticipee{slot: 78, tete: 1}] = []declarationAnticipee{{chunk: 6, ti: 2}}
	w.PoserTableAnticipee(tab)
	w.PoserChunkCourant(2)
	cfg := FrameConfig{IDLowBits: 13, Profil: ProfilDeBalayageParDefaut(), Obs: NouvelleObservation()}

	for _, id := range []uint32{1<<30 | 77, 1<<30 | 78} {
		if _, rejete := rejetDeVue(recDelta, id, w, cfg); rejete {
			t.Fatalf("eid %#x rejete : la table anticipee le declare, le repli doit le lier", id)
		}
	}
	if _, rejete := rejetDeVue(recDelta, 1<<30|79, w, cfg); !rejete {
		t.Fatal("eid du slot 79 accepte : aucune image-cle ne le declare, il doit etre rejete hors datum")
	}
	sc := &movementStateScanner{obs: cfg.Obs}
	if got := sc.liaisonsDuRepliDAnticipation(); got != 2 {
		t.Fatalf("la marche rend %d liaison(s) par anticipation, attendu 2 : le compte du repli "+
			"n arrive pas a la cuisson", got)
	}
	if got := cfg.Obs.LiaisonsParRepliDAnticipation[3] + cfg.Obs.LiaisonsParRepliDAnticipation[2]; got != 2 {
		t.Fatalf("l observation compte %d liaison(s), attendu 2", got)
	}
}
