package service

// tactical_service_groupes_test.go — UNE CARTE PAR GROUPE dans l'onglet Tactique (plan cartes
// variantes, E3). Le port est double : ce qui se verifie ici est l'orchestration (somme des rangees,
// plancher recalcule, identifiants passes au lecteur, vignette du groupe), le SQL a ses tests
// `:memory:` (platform/duckdb).

import (
	"context"
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

type lecteurDeGroupesTactique struct{ groups domain.GroupesDeCartes }

func (l lecteurDeGroupesTactique) MapGroups(context.Context) (domain.GroupesDeCartes, error) {
	return l.groups, nil
}

// groupeThunderhead : la base « base-a » (deux identifiants) et sa variante « var-heavies ».
func groupeThunderhead() lecteurDeGroupesTactique {
	return lecteurDeGroupesTactique{groups: domain.NewGroupesDeCartes(map[string]string{
		"var-heavies": "base-a", "base-b": "base-a",
	}).AvecLibelles(map[string]i18n.Text{
		"base-a": {"fr": "Tonnerre", "en": "Thunderhead"},
	})}
}

// La base et sa variante sont UNE carte : parties sommees, plancher recalcule sur la somme, libelle de la
// base, identifiant de base. Une carte hors groupe reste seule.
func TestMapsPlayed_GroupeFusionneEtPlancherSurLaSomme(t *testing.T) {
	repo := &mockTacticalRepo{maps: []domain.TacticalMapRow{
		{MapID: "base-a", MapName: i18n.FromColumns("Thunderhead", "Tonnerre"), Matchs: 6, Victoires: 4, Defaites: 2},
		{MapID: "var-heavies", MapName: i18n.FromColumns("Thunderhead Heavies", "Du lourd sur Thunderhead"), Matchs: 5, Victoires: 1, Defaites: 4},
		{MapID: "seule", MapName: i18n.FromColumns("The Pit", "La Fosse"), Matchs: 7, Victoires: 3, Defaites: 4},
	}}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi).WithMapGroups(groupeThunderhead())
	page, err := svc.MapsPlayed(context.Background(), domain.TacticalScope{})
	if err != nil {
		t.Fatalf("MapsPlayed: %v", err)
	}
	if len(page.Cartes) != 2 {
		t.Fatalf("cartes = %d, attendu 2 (un groupe + une carte seule) : %+v", len(page.Cartes), page.Cartes)
	}
	g := page.Cartes[0]
	if g.MapID != "base-a" || g.Matchs != 11 || g.Victoires != 5 || g.Defaites != 6 {
		t.Errorf("groupe = %+v, attendu base-a, 11 parties, 5 V, 6 D", g)
	}
	if g.SousPlancher {
		t.Errorf("11 parties >= plancher %d : le groupe ne doit pas etre sous le plancher alors que ses parts le sont",
			domain.PlancherMatchsParCarte)
	}
	if g.MapName != "Tonnerre" || g.MapLabel != "Tonnerre" || g.MapFilterKey != "Tonnerre" {
		t.Errorf("libelles = %q / %q, attendu ceux de la BASE (aucune mention de la variante)", g.MapName, g.MapLabel)
	}
	if s := page.Cartes[1]; s.MapID != "seule" || !s.SousPlancher {
		t.Errorf("carte seule = %+v, attendu inchangee (7 < plancher)", s)
	}
	// Sans lecteur : le comportement d'avant, une rangee par identifiant.
	avant, _ := NewTacticalService(repo, capsCompletes(), tsMoi).MapsPlayed(context.Background(), domain.TacticalScope{})
	if len(avant.Cartes) != 3 {
		t.Errorf("sans groupe : %d cartes, attendu 3", len(avant.Cartes))
	}
}

// Une lecture de placement ouverte sur l'identifiant de base lit TOUT le groupe, et la meme vaut pour une
// ouverture par un identifiant de variante.
func TestRaster_LitTousLesIdentifiantsDuGroupe(t *testing.T) {
	repo := &mockTacticalRepo{pos: domain.TacticalPositions{Univers: universUnMatch("m1", domain.OutcomeWin)}}
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi).WithMapGroups(groupeThunderhead())
	for _, ouverture := range []string{"base-a", "var-heavies"} {
		repo.vuPos = domain.TacticalQuery{}
		if _, err := svc.Raster(context.Background(),
			tsDemande(repo, ouverture, domain.TacticalQuestionMorts, domain.TacticalQuiMoi)); err != nil {
			t.Fatalf("Raster(%s): %v", ouverture, err)
		}
		want := []string{"base-a", "base-b", "var-heavies"}
		if !reflect.DeepEqual(repo.vuPos.MapIDs, want) {
			t.Errorf("ouverture par %s : MapIDs = %v, attendu %v", ouverture, repo.vuPos.MapIDs, want)
		}
	}
	// Carte hors groupe : son identifiant seul.
	repo.vuPos = domain.TacticalQuery{}
	if _, err := svc.Raster(context.Background(),
		tsDemande(repo, "seule", domain.TacticalQuestionMorts, domain.TacticalQuiMoi)); err != nil {
		t.Fatalf("Raster(seule): %v", err)
	}
	if !reflect.DeepEqual(repo.vuPos.MapIDs, []string{"seule"}) {
		t.Errorf("carte seule : MapIDs = %v, attendu [seule]", repo.vuPos.MapIDs)
	}
}

// La vignette d'un groupe est le mini-plan de TOUT le groupe : les morts de la variante et de la base se
// superposent, et le denominateur « par match » compte les matchs des deux.
func TestMapsPlayed_VignetteDuGroupeReunitLesMorts(t *testing.T) {
	repo := &mockTacticalRepo{
		maps: []domain.TacticalMapRow{
			{MapID: "base-a", MapName: i18n.FromColumns("Thunderhead", ""), Matchs: 6, Victoires: 3, Defaites: 3},
			{MapID: "var-heavies", MapName: i18n.FromColumns("Thunderhead Heavies", ""), Matchs: 6, Victoires: 3, Defaites: 3},
		},
		mortsParCarte: map[string][]domain.PositionSample{
			"base-a":      {{MatchID: "m1", X: 2, Y: 2}, {MatchID: "m2", X: 2.2, Y: 2.2}},
			"var-heavies": {{MatchID: "m3", X: 2.4, Y: 2.4}},
		},
	}
	page, err := NewTacticalService(repo, capsCompletes(), tsMoi).WithMapGroups(groupeThunderhead()).
		MapsPlayed(context.Background(), domain.TacticalScope{MatchIDs: []string{"m1", "m2", "m3"}})
	if err != nil {
		t.Fatalf("MapsPlayed: %v", err)
	}
	if len(page.Cartes) != 1 || len(page.Cartes[0].Cellules) == 0 {
		t.Fatalf("cartes = %+v, attendu une vignette peinte", page.Cartes)
	}
	if got := page.Cartes[0].Cellules[0].Matchs; got != 3 {
		t.Errorf("matchs distincts de la cellule = %d, attendu 3 (deux de la base, un de la variante)", got)
	}
}
