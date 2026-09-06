// Package service — tactical_service_perimetre_test.go : LE PERIMETRE et L'AXE
// ESCOUADE de l'onglet Tactique (phase 4 bis, 2026-09-06).
//
// Trois frontieres, chacune invisible a l'ecran si elle cede :
//
//  1. la liste blanche et la composition descendent AU LECTEUR, sur les DEUX lectures ;
//  2. une liste VIDE est POSEE (restreinte, sans identifiant) — jamais confondue avec
//     « aucune restriction », qui servirait tout l'historique ;
//  3. l'axe « escouade » EST la composition choisie : refuse sans elle, et strictement
//     borne a ses xuids avec elle. Le KPI d'echange, lui, reste sur MON CAMP ENTIER
//     (decision utilisateur du 2026-09-06) : deux perimetres voisins qu'un seul
//     predicat partage ferait fusionner sans que rien ne le montre.
package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

// TestTacticalService_PerimetreTransmis : la liste blanche et la composition
// descendent TELLES QUELLES au lecteur, et LES DEUX lectures (positions, journal)
// recoivent le MEME perimetre — sinon le KPI d'echange porterait sur une population
// que la carte ne montre pas.
func TestTacticalService_PerimetreTransmis(t *testing.T) {
	repo := &mockTacticalRepo{pos: domain.TacticalPositions{Univers: universUnMatch("m1", domain.OutcomeWin)}}

	svc := NewTacticalService(repo, capsCompletes(), tsMoi)
	req := tsDemande(tsCarte, domain.TacticalQuestionKills, domain.TacticalQuiMoi, tsAmi)
	req.Scope.MatchIDs = []string{"m1", "m2"}
	if _, err := svc.Raster(context.Background(), req); err != nil {
		t.Fatalf("Raster: %v", err)
	}
	for nom, vu := range map[string]domain.TacticalQuery{"positions": repo.vuPos, "journal": repo.vuEv} {
		if vu.MapID != tsCarte || vu.PlayerXUID != tsMoi {
			t.Errorf("%s : demande = %+v, want carte/joueur inchanges", nom, vu)
		}
		if !vu.Matchs.Restreint() || !egalesXUID(vu.Matchs.IDs(), []string{"m1", "m2"}) {
			t.Errorf("%s : liste blanche = %v (restreinte=%v), want [m1 m2]",
				nom, vu.Matchs.IDs(), vu.Matchs.Restreint())
		}
		if !egalesXUID(vu.Coequipiers, []string{tsAmi}) {
			t.Errorf("%s : composition = %v, want [%s]", nom, vu.Coequipiers, tsAmi)
		}
	}
}

// TestTacticalService_ListeBlancheVide_Posee : une liste vide est POSEE au lecteur
// (restreinte, sans identifiant) et jamais confondue avec « aucune restriction ».
// C'est la seule frontiere qui empeche un perimetre vide de servir l'historique
// entier, et elle est invisible a l'ecran.
func TestTacticalService_ListeBlancheVide_Posee(t *testing.T) {
	repo := &mockTacticalRepo{pos: domain.TacticalPositions{Univers: universUnMatch("m1", domain.OutcomeWin)}}
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)

	if _, err := svc.Raster(context.Background(),
		tsDemande(tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi)); err != nil {
		t.Fatalf("Raster: %v", err)
	}
	if !repo.vuPos.Matchs.Restreint() {
		t.Errorf("liste blanche non posee : le lecteur servirait tout l'historique")
	}
	if len(repo.vuPos.Matchs.IDs()) != 0 {
		t.Errorf("liste blanche = %v, want vide", repo.vuPos.Matchs.IDs())
	}
}

// egalesXUID compare deux listes ordonnees de chaines.
func egalesXUID(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ─── L'AXE ESCOUADE = LA COMPOSITION CHOISIE ───────────────────────────────────

// TestTacticalService_Escouade_SansComposition_Refuse : sans coequipiers, l'axe n'a
// aucun contenu. Le REFUS est typé (400) plutot qu'un repli silencieux sur les
// coequipiers du match — qui repondrait a une AUTRE question sous le meme nom.
func TestTacticalService_Escouade_SansComposition_Refuse(t *testing.T) {
	repo := &mockTacticalRepo{pos: domain.TacticalPositions{Univers: universUnMatch("m1", domain.OutcomeWin)}}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)

	_, err := svc.Raster(context.Background(),
		tsDemande(tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiEscouade))
	if !errors.Is(err, domain.ErrTacticalEscouadeSansComposition) {
		t.Fatalf("err = %v, want ErrTacticalEscouadeSansComposition", err)
	}
	// La validation passe AVANT toute lecture : un axe sans contenu ne doit rien
	// faire lire.
	if repo.vuPos.PlayerXUID != "" {
		t.Errorf("le lecteur a ete appele malgre le refus : %+v", repo.vuPos)
	}
}

// TestTacticalService_Escouade_CompositionVideDeBlancs_Refuse : une composition qui
// ne porte que du vide N'EST PAS une composition. Sans ce nettoyage, un client qui
// envoie [""] ouvrirait l'axe sur un ensemble qui ne designe personne — donc sur une
// carte vide presentee comme une mesure.
func TestTacticalService_Escouade_CompositionVideDeBlancs_Refuse(t *testing.T) {
	repo := &mockTacticalRepo{pos: domain.TacticalPositions{Univers: universUnMatch("m1", domain.OutcomeWin)}}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)

	_, err := svc.Raster(context.Background(),
		tsDemande(tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiEscouade, "", "  "))
	if !errors.Is(err, domain.ErrTacticalEscouadeSansComposition) {
		t.Errorf("err = %v, want ErrTacticalEscouadeSansComposition", err)
	}
}

// TestTacticalService_Escouade_CibleExactementLaComposition : LE test de l'axe.
// Trois coequipiers tombent dans le match ; la composition n'en nomme QU'UN — seul
// son point est peint. Le predicat « meme equipe que moi » d'avant le 2026-09-06 en
// aurait peint trois.
func TestTacticalService_Escouade_CibleExactementLaComposition(t *testing.T) {
	const ami2, ami3 = "xuid(21)", "xuid(22)"
	repo := &mockTacticalRepo{}
	repo.pos.Univers = domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	for _, id := range []string{"m1", "m2", "m3"} {
		u := universUnMatch(id, domain.OutcomeWin)
		repo.pos.Univers.Matchs = append(repo.pos.Univers.Matchs, u.Matchs...)
		equipe := u.Equipes[id]
		equipe[ami2] = equipe[tsAmi] // les deux autres sont DANS MON EQUIPE, eux aussi
		equipe[ami3] = equipe[tsAmi]
		repo.pos.Univers.Equipes[id] = equipe
		repo.pos.Points = append(repo.pos.Points,
			domain.TacticalKillPosition{MatchID: id, KillerXUID: tsAdv, VictimXUID: tsAmi,
				KillerX: 1.0, KillerY: 1.0, VictimX: 4.0, VictimY: 4.0},
			domain.TacticalKillPosition{MatchID: id, KillerXUID: tsAdv, VictimXUID: ami2,
				KillerX: 1.0, KillerY: 1.0, VictimX: 40.0, VictimY: 40.0},
			domain.TacticalKillPosition{MatchID: id, KillerXUID: tsAdv, VictimXUID: ami3,
				KillerX: 1.0, KillerY: 1.0, VictimX: 60.0, VictimY: 60.0},
		)
	}
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)

	got, err := svc.Raster(context.Background(),
		tsDemande(tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiEscouade, tsAmi))
	if err != nil {
		t.Fatalf("Raster(escouade): %v", err)
	}
	if len(got.Cellules) != 1 || celluleEn(got.Cellules, 4.0, 4.0) == nil {
		t.Fatalf("cellules = %+v, want la seule (4,4) — les deux autres coequipiers du "+
			"match ne sont PAS dans la composition choisie", got.Cellules)
	}
}

// TestTacticalService_Echange_PorteSurLeCampEntier : le KPI d'echange ne retrecit
// PAS avec la composition (decision utilisateur du 2026-09-06 : « le KPI reste sur
// mon camp entier »). Un coequipier hors composition qui meurt sans etre venge doit
// peser sur le taux — sinon nommer deux joueurs dans la barre de filtres changerait
// un taux qui ne parle pas d'eux.
func TestTacticalService_Echange_PorteSurLeCampEntier(t *testing.T) {
	const ami2 = "xuid(21)"
	repo := &mockTacticalRepo{pos: domain.TacticalPositions{Univers: universUnMatch("m1", domain.OutcomeWin)}}
	repo.pos.Points = []domain.TacticalKillPosition{{
		MatchID: "m1", KillerXUID: tsAdv, VictimXUID: tsMoi,
		KillerX: 1.0, KillerY: 1.0, VictimX: 10.0, VictimY: 10.0,
	}}
	equipes := domain.EquipesParMatch{"m1": {}}
	for x, t := range repo.pos.Univers.Equipes["m1"] {
		equipes["m1"][x] = t
	}
	equipes["m1"][ami2] = equipes["m1"][tsAmi] // dans mon equipe, HORS composition
	repo.pos.Univers.Equipes = equipes
	repo.ev = domain.TacticalKillEvents{
		Univers: domain.TacticalUnivers{Matchs: repo.pos.Univers.Matchs, Equipes: equipes},
		Events: []domain.KillEvent{
			// Deux morts de mon camp, aucune vengee : le taux vaut 0 sur DEUX morts.
			{MatchID: "m1", VictimXUID: tsAmi, KillerXUID: tsAdv, TimeMs: 1000},
			{MatchID: "m1", VictimXUID: ami2, KillerXUID: tsAdv, TimeMs: 2000},
		},
	}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)

	got, err := svc.Raster(context.Background(),
		tsDemande(tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi, tsAmi))
	if err != nil {
		t.Fatalf("Raster: %v", err)
	}
	if got.Echange == nil {
		t.Fatalf("Echange = nil, want une couverture servie")
	}
	if got.Echange.N != 2 {
		t.Errorf("morts vengeables = %d, want 2 (mon camp ENTIER, pas la seule composition)",
			got.Echange.N)
	}
}
