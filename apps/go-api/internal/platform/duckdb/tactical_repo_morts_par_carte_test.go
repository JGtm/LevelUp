package duckdb

// tactical_repo_morts_par_carte_test.go — MES MORTS GROUPEES PAR CARTE (lot F,
// 2026-09-13), sur la VRAIE base migree (vues `_latest` comprises, aucune DDL recopiee).
//
// Ce que ces cas verrouillent, et le defaut que chacun attrape :
//
//	la FACE          seules MES morts sortent — mes KILLS peindraient l'endroit ou je
//	                 gagne des duels sous un calque cense montrer ou je les perds ;
//	la CARTE         chaque point sort avec SA carte : un groupement rate melangerait
//	                 deux plans ;
//	l'UNIVERS        un match ou je n'ai pas joue n'entre pas, meme sur ma carte ;
//	le PERIMETRE     la liste blanche s'applique ici comme aux trois autres lectures ;
//	les GARDES       passe non publiable et double kill au meme (tueur, instant) sont
//	                 ecartes, exactement comme dans `KillPositions`.

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

func TestTacticalRepo_MortsParCarte(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	got, err := NewTacticalRepo(pdb).MortsParCarte(context.Background(),
		domain.TacticalQuery{PlayerXUID: tacXUIDMoi})
	if err != nil {
		t.Fatalf("MortsParCarte: %v", err)
	}
	// Corpus : m1 (carte A) porte MA mort a (8,8) et MON kill a (4,4) ; m3 (carte B) ne
	// porte que MON kill. Seule la mort doit sortir, et seulement sur la carte A.
	if len(got) != 1 {
		t.Fatalf("cartes = %d (%v), want 1 : seule la carte A porte une de mes morts", len(got), got)
	}
	points := got[tacCarteA]
	if len(points) != 1 {
		t.Fatalf("points sur la carte A = %+v, want 1", points)
	}
	if points[0].MatchID != "m1" {
		t.Errorf("match = %q, want m1", points[0].MatchID)
	}
	if points[0].X != 8.0 || points[0].Y != 8.0 {
		t.Errorf("position = (%v, %v), want (8, 8) : la position de la VICTIME", points[0].X, points[0].Y)
	}
	if _, ok := got[tacCarteB]; ok {
		t.Errorf("la carte B sort alors qu'elle ne porte que MON kill : %+v", got[tacCarteB])
	}
}

func TestTacticalRepo_MortsParCarte_PerimetreEtGardes(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

	// m5 : ma mort sur la carte B, mais la passe N'EST PAS PUBLIABLE — attribution par
	// ligne non fiable, elle est ecartee.
	tacMatch(t, pdb, "m5", tacCarteB, base.Add(4*time.Hour))
	tacParticipant(t, pdb, "m5", tacXUIDMoi, 0, domain.OutcomeLoss)
	tacKill(t, pdb, "m5", tacXUIDAdv, tacXUIDMoi, 1000, false)
	tacPos(t, pdb, "m5", tacXUIDAdv, 1000, 1.0, 1.0, 3.0, 3.0)

	// m6 : DOUBLE KILL au meme (tueur, instant) — la position de victime n'est pas
	// attribuable, les DEUX lignes sont ecartees.
	tacMatch(t, pdb, "m6", tacCarteB, base.Add(5*time.Hour))
	tacParticipant(t, pdb, "m6", tacXUIDMoi, 0, domain.OutcomeLoss)
	tacParticipant(t, pdb, "m6", tacXUIDAmi, 0, domain.OutcomeLoss)
	tacKill(t, pdb, "m6", tacXUIDAdv, tacXUIDMoi, 2000, true)
	tacKill(t, pdb, "m6", tacXUIDAdv, tacXUIDAmi, 2000, true)
	tacPos(t, pdb, "m6", tacXUIDAdv, 2000, 1.0, 1.0, 5.0, 5.0)

	repo := NewTacticalRepo(pdb)

	got, err := repo.MortsParCarte(context.Background(), domain.TacticalQuery{PlayerXUID: tacXUIDMoi})
	if err != nil {
		t.Fatalf("MortsParCarte: %v", err)
	}
	if _, ok := got[tacCarteB]; ok {
		t.Errorf("la carte B sort : passe non publiable et double kill doivent etre ecartes (%+v)", got[tacCarteB])
	}

	// PERIMETRE : une liste blanche qui ne contient pas m1 vide la lecture.
	horsPerimetre, err := repo.MortsParCarte(context.Background(), domain.TacticalQuery{
		PlayerXUID: tacXUIDMoi, Matchs: domain.RestreindreAux([]string{"m3"}),
	})
	if err != nil {
		t.Fatalf("MortsParCarte (perimetre): %v", err)
	}
	if len(horsPerimetre) != 0 {
		t.Errorf("la liste blanche ne filtre pas : %+v", horsPerimetre)
	}
}

func TestTacticalRepo_MortsParCarte_XUIDVideRefuse(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	if _, err := NewTacticalRepo(pdb).MortsParCarte(context.Background(), domain.TacticalQuery{}); err == nil {
		t.Fatal("un xuid vide doit etre un REFUS, jamais un balayage de la table")
	}
}
