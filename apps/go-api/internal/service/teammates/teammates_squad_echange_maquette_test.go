package teammates

// teammates_squad_echange_maquette_test.go — les DEUX mesures ajoutees le 2026-09-13 pour
// la page « L'echange » de l'Escouade (maquette 4c520da6) :
//
//	DelaiMedianMs    la tuile « Delai median » du bloc « Le compte » ;
//	TauxParSession   la carte « Taux d'echange par session ».
//
// Elles vivent dans un fichier a part parce qu'elles n'ont aucune des invariantes de la
// matrice ni de la distribution : ce sont deux DECOUPES de la meme mesure de couverture.

import (
	"context"
	"sort"
	"testing"

	"levelup/go-api/internal/domain"
)

// journalTroisEchanges pose un journal ou le camp subit QUATRE morts vengeables sur un seul
// match, toutes suivies d'une riposte, aux delais 1 000 / 3 000 / 4 000 / 8 000 ms.
//
// La derniere (8 000 ms) est HORS FENETRE : ce n'est pas un echange, et elle ne doit PAS
// entrer dans la mediane — l'y inclure la ferait passer de 3 000 (mediane de trois) a
// 3 500 (mediane de quatre), vers une population que le taux ne compte pas.
func journalTroisEchanges() domain.TacticalKillEvents {
	return domain.TacticalKillEvents{
		Univers: universDe("m1"),
		Events: []domain.KillEvent{
			// mort 1 : main tue a 1 000, venge par Ami a 2 000 -> delai 1 000
			{MatchID: "m1", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 1000},
			{MatchID: "m1", KillerXUID: "x_Ami", VictimXUID: "x_adv1", TimeMs: 2000},
			// mort 2 : Ami tue a 10 000, venge par main a 13 000 -> delai 3 000
			{MatchID: "m1", KillerXUID: "x_adv2", VictimXUID: "x_Ami", TimeMs: 10000},
			{MatchID: "m1", KillerXUID: "x_main", VictimXUID: "x_adv2", TimeMs: 13000},
			// mort 3 : main tue a 20 000, venge par Ami a 24 000 -> delai 4 000
			{MatchID: "m1", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 20000},
			{MatchID: "m1", KillerXUID: "x_Ami", VictimXUID: "x_adv1", TimeMs: 24000},
			// mort 4 : Ami tue a 30 000, riposte a 38 000 -> 8 000 ms, HORS FENETRE
			{MatchID: "m1", KillerXUID: "x_adv2", VictimXUID: "x_Ami", TimeMs: 30000},
			{MatchID: "m1", KillerXUID: "x_main", VictimXUID: "x_adv2", TimeMs: 38000},
		},
	}
}

func TestBuildSquadEchange_DelaiMedian(t *testing.T) {
	repo := &mockTacticalRepo{lecture: journalTroisEchanges()}
	svc := svcEchange(repo, capsFiables())

	got := svc.buildSquadEchange(context.Background(),
		echangeRows("m1"), echangeRows("m1"), "main", "x_main", echangeMates("Ami"))
	if got == nil {
		t.Fatal("section attendue, obtenu nil")
	}

	// Trois echanges DANS la fenetre : 1 000, 3 000, 4 000 -> mediane 3 000.
	if got.DelaiMedianMs != 3000 {
		t.Errorf("delai median = %d ms, attendu 3000", got.DelaiMedianMs)
	}
	// Garde-fou de scenario : si la riposte hors fenetre entrait dans la mediane, la
	// population serait 1 000 / 3 000 / 4 000 / 8 000 et la mediane vaudrait 3 500.
	if got.DelaiMedianMs == 3500 {
		t.Error("la riposte HORS FENETRE est entree dans la mediane : ce n'est pas un echange")
	}
}

func TestBuildSquadEchange_DelaiMedianZeroSansEchange(t *testing.T) {
	// Une seule mort, jamais vengee : aucun echange, donc aucune mediane a publier.
	repo := &mockTacticalRepo{lecture: domain.TacticalKillEvents{
		Univers: universDe("m1"),
		Events: []domain.KillEvent{
			{MatchID: "m1", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 1000},
		},
	}}
	svc := svcEchange(repo, capsFiables())

	got := svc.buildSquadEchange(context.Background(),
		echangeRows("m1"), echangeRows("m1"), "main", "x_main", echangeMates("Ami"))
	if got == nil {
		t.Fatal("section attendue, obtenu nil")
	}
	if got.DelaiMedianMs != 0 {
		t.Errorf("delai median = %d ms, attendu 0 (aucun echange) — le client ne rend pas la tuile",
			got.DelaiMedianMs)
	}
}

// rowsAvecSessions rend des SquadMatchRow portant un libelle de session : c'est la maille
// de decoupe de `TauxParSession` (la meme que celle du nuage d'isolement).
func rowsAvecSessions(parSession map[string][]string) []domain.SquadMatchRow {
	label := map[string]string{}
	ids := []string{}
	for l, matchs := range parSession {
		for _, id := range matchs {
			label[id] = l
			ids = append(ids, id)
		}
	}
	// L'ordre des libelles servis est celui du match le plus ancien de chaque session :
	// trier les ids rend l'ordre attendu deterministe (echangeRows horodate en croissant).
	sort.Strings(ids)
	out := echangeRows(ids...)
	for i := range out {
		l := label[out[i].MatchID]
		out[i].SessionLabel = &l
	}
	return out
}

func TestBuildSquadEchange_TauxParSession(t *testing.T) {
	// Deux soirees. m1 : une mort vengeable, vengee. m2 : une mort vengeable, JAMAIS
	// vengee. Le taux par session doit donc valoir 1 puis 0 — et surtout pas la moyenne
	// des deux, qui est ce que l'agregat dit deja.
	repo := &mockTacticalRepo{lecture: domain.TacticalKillEvents{
		Univers: universDe("m1", "m2"),
		Events: []domain.KillEvent{
			{MatchID: "m1", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 1000},
			{MatchID: "m1", KillerXUID: "x_Ami", VictimXUID: "x_adv1", TimeMs: 2000},
			{MatchID: "m2", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 1000},
		},
	}}
	svc := svcEchange(repo, capsFiables())

	rows := rowsAvecSessions(map[string][]string{"soir A": {"m1"}, "soir B": {"m2"}})
	got := svc.buildSquadEchange(context.Background(),
		rows, rows, "main", "x_main", echangeMates("Ami"))
	if got == nil {
		t.Fatal("section attendue, obtenu nil")
	}
	if len(got.TauxParSession) != 2 {
		t.Fatalf("taux par session = %+v, attendu 2 points", got.TauxParSession)
	}
	if got.TauxParSession[0].SessionLabel != "soir A" || got.TauxParSession[0].Couverture.Taux != 1 {
		t.Errorf("1er point = %+v, attendu {soir A, taux 1}", got.TauxParSession[0])
	}
	if got.TauxParSession[1].SessionLabel != "soir B" || got.TauxParSession[1].Couverture.Taux != 0 {
		t.Errorf("2e point = %+v, attendu {soir B, taux 0}", got.TauxParSession[1])
	}
	// Le brut et son denominateur voyagent AVEC le taux : jamais un float nu.
	if got.TauxParSession[0].Couverture.N != 1 || got.TauxParSession[0].Couverture.Brut != 1 {
		t.Errorf("1er point sans son compte : %+v", got.TauxParSession[0].Couverture)
	}
	if got.TauxParSession[0].MatchsMesures != 1 {
		t.Errorf("matchs mesures du 1er point = %d, attendu 1", got.TauxParSession[0].MatchsMesures)
	}
	// Ici filtre == historique : les deux soirees sont dans le filtre.
	for i, p := range got.TauxParSession {
		if !p.DansLeFiltre {
			t.Errorf("point %d (%s) hors filtre alors que filtre == historique", i, p.SessionLabel)
		}
	}
}

// TestBuildSquadEchange_TauxParSessionCouvreToutLHistorique cadenasse la decision
// utilisateur du 2026-09-22 : la frise montre TOUTES les soirees de la composition, pas
// seulement celles du filtre, et chaque point dit s'il est dans le filtre courant.
//
// Avant ce lot, filtrer sur une soiree — l'usage nominal de la page — rendait UN baton
// sans aucune population autour de lui.
func TestBuildSquadEchange_TauxParSessionCouvreToutLHistorique(t *testing.T) {
	// Trois soirees dans l'historique, UNE SEULE retenue par le filtre (m2 / « soir B »).
	repo := &mockTacticalRepo{lecture: domain.TacticalKillEvents{
		Univers: universDe("m1", "m2", "m3"),
		Events: []domain.KillEvent{
			// soir A : une mort vengeable, vengee.
			{MatchID: "m1", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 1000},
			{MatchID: "m1", KillerXUID: "x_Ami", VictimXUID: "x_adv1", TimeMs: 2000},
			// soir B : une mort vengeable, JAMAIS vengee.
			{MatchID: "m2", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 1000},
			// soir C : une mort vengeable, vengee.
			{MatchID: "m3", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 1000},
			{MatchID: "m3", KillerXUID: "x_Ami", VictimXUID: "x_adv1", TimeMs: 3000},
		},
	}}
	svc := svcEchange(repo, capsFiables())

	historique := rowsAvecSessions(map[string][]string{
		"soir A": {"m1"}, "soir B": {"m2"}, "soir C": {"m3"},
	})
	filtre := rowsAvecSessions(map[string][]string{"soir B": {"m2"}})
	got := svc.buildSquadEchange(context.Background(),
		filtre, historique, "main", "x_main", echangeMates("Ami"))
	if got == nil {
		t.Fatal("section attendue, obtenu nil")
	}
	// LES TROIS SOIREES SONT LA, dans l'ordre chronologique — pas la seule du filtre.
	if len(got.TauxParSession) != 3 {
		t.Fatalf("taux par session = %+v, attendu les 3 soirees de l'historique", got.TauxParSession)
	}
	attendu := []struct {
		label string
		dans  bool
	}{{"soir A", false}, {"soir B", true}, {"soir C", false}}
	for i, a := range attendu {
		p := got.TauxParSession[i]
		if p.SessionLabel != a.label {
			t.Errorf("point %d = %q, attendu %q (ordre chronologique)", i, p.SessionLabel, a.label)
		}
		if p.DansLeFiltre != a.dans {
			t.Errorf("point %q : dans_le_filtre = %v, attendu %v", p.SessionLabel, p.DansLeFiltre, a.dans)
		}
	}
	// La mesure de chaque soiree reste la sienne : vengee / pas vengee / vengee.
	if got.TauxParSession[0].Couverture.Taux != 1 || got.TauxParSession[1].Couverture.Taux != 0 {
		t.Errorf("taux par soiree = %+v, attendu 1 puis 0", got.TauxParSession)
	}
}

func TestBuildSquadEchange_TauxParSessionIgnoreLesSessionsSansMort(t *testing.T) {
	// m2 est retenu par le filtre et mesure, mais PERSONNE n'y meurt du camp : la session
	// n'a pas un taux nul, elle n'a pas de taux. Un zero s'y lirait comme une
	// contre-performance.
	repo := &mockTacticalRepo{lecture: domain.TacticalKillEvents{
		Univers: universDe("m1", "m2"),
		Events: []domain.KillEvent{
			{MatchID: "m1", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 1000},
			{MatchID: "m1", KillerXUID: "x_Ami", VictimXUID: "x_adv1", TimeMs: 2000},
			// m2 : seul un adversaire meurt — aucune mort de MON camp.
			{MatchID: "m2", KillerXUID: "x_main", VictimXUID: "x_adv1", TimeMs: 1000},
		},
	}}
	svc := svcEchange(repo, capsFiables())

	rows := rowsAvecSessions(map[string][]string{"soir A": {"m1"}, "soir B": {"m2"}})
	got := svc.buildSquadEchange(context.Background(),
		rows, rows, "main", "x_main", echangeMates("Ami"))
	if got == nil {
		t.Fatal("section attendue, obtenu nil")
	}
	if len(got.TauxParSession) != 1 || got.TauxParSession[0].SessionLabel != "soir A" {
		t.Fatalf("taux par session = %+v, attendu le SEUL point « soir A »", got.TauxParSession)
	}
}

func TestBuildSquadEchange_TauxParSessionVideSansLibelle(t *testing.T) {
	// Aucun libelle de session : la carte ne se lit pas par match isole. Une tranche vide,
	// jamais un point fabrique.
	repo := &mockTacticalRepo{lecture: journalTroisEchanges()}
	svc := svcEchange(repo, capsFiables())

	got := svc.buildSquadEchange(context.Background(),
		echangeRows("m1"), echangeRows("m1"), "main", "x_main", echangeMates("Ami"))
	if got == nil {
		t.Fatal("section attendue, obtenu nil")
	}
	if len(got.TauxParSession) != 0 {
		t.Errorf("taux par session = %+v, attendu vide (aucun libelle de session)", got.TauxParSession)
	}
}
