// Package service — tactical_service_portes_test.go : les PORTES de capability de la
// lecture de placement, la liste des cartes jouees, et le service sans lecteur.
//
// Cas deplaces TELS QUELS, noms inchanges, de tactical_service_echange_test.go quand le KPI
// d'echange a quitte l'onglet (plan Tactique v2, L9). Le mock du port et les fabriques de
// capabilities restent chez les voisins, meme paquet.
package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// ─── LES PORTES ────────────────────────────────────────────────────────────────

// TestTacticalService_AucunePositionLisible_Capability : aucune des DEUX
// provenances de positions — ErrCapabilityNotSupported (503 propre).
func TestTacticalService_AucunePositionLisible_Capability(t *testing.T) {
	repo := &mockTacticalRepo{pos: domain.TacticalPositions{Univers: universUnMatch("m1", domain.OutcomeWin)}}
	for nom, caps := range map[string]games.CapabilityMap{
		"map vide":              {},
		"map nil":               nil,
		"kill_source seule":     {games.CapFilmKillSource: games.CapSupported},
		"capture non exposee":   {games.CapFilmKillPositions: games.CapNotExposed},
		"spatial non expose":    {games.CapMatchEventsSpatial: games.CapNotExposed},
		"les deux non exposees": {games.CapFilmKillPositions: games.CapNotExposed, games.CapMatchEventsSpatial: games.CapNotExposed},
		"killfeed natif seul":   {games.CapMatchKillfeedPerKill: games.CapSupported},
	} {
		svc := NewTacticalService(repo, caps, tsMoi)
		_, err := svc.Raster(context.Background(), tsDemande(repo, tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
		if !errors.Is(err, games.ErrCapabilityNotSupported) {
			t.Errorf("%s: err = %v, want ErrCapabilityNotSupported", nom, err)
		}
	}
}

// TestTacticalService_PositionsNatives_RasterServi : LE test de la correction R1.
//
// Un titre qui remplit `kill_positions` NATIVEMENT (Halo 5 : `match.events.spatial
// = supported`, aucun decodeur de film, donc AUCUNE declaration de
// `film.kill_positions` — qui gouverne la capture par le film) doit etre servi. La
// version precedente lui rendait un 503 alors que la jointure marche integralement.
func TestTacticalService_PositionsNatives_RasterServi(t *testing.T) {
	repo := &mockTacticalRepo{}
	repo.pos.Univers = domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	for _, id := range []string{"m1", "m2", "m3"} {
		u := universUnMatch(id, domain.OutcomeWin)
		repo.pos.Univers.Matchs = append(repo.pos.Univers.Matchs, u.Matchs...)
		repo.pos.Univers.Equipes[id] = u.Equipes[id]
		repo.pos.Points = append(repo.pos.Points, domain.TacticalKillPosition{
			MatchID: id, KillerXUID: tsAdv, VictimXUID: tsMoi,
			KillerX: 1.0, KillerY: 1.0, VictimX: 10.0, VictimY: 10.0,
		})
	}
	caps := games.CapabilityMap{games.CapMatchEventsSpatial: games.CapSupported}
	svc := NewTacticalService(repo, caps, tsMoi)

	got, err := svc.Raster(context.Background(), tsDemande(repo, tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
	if err != nil {
		t.Fatalf("positions NATIVES (match.events.spatial) : err = %v, want une lecture servie", err)
	}
	if len(got.Cellules) != 1 || celluleEn(got.Cellules, 10.0, 10.0) == nil {
		t.Errorf("cellules = %+v, want la cellule (10,10)", got.Cellules)
	}
}

// ─── L'ECRAN D'ENTREE ──────────────────────────────────────────────────────────

// TestTacticalService_MapsPlayed_Plancher : le drapeau « sous le plancher » est
// pose ICI (regle produit), pas dans la requete SQL.
func TestTacticalService_MapsPlayed_Plancher(t *testing.T) {
	repo := &mockTacticalRepo{maps: []domain.TacticalMapRow{
		{MapID: "a", MapName: "Aquarius", Matchs: domain.PlancherMatchsParCarte, Victoires: 6, Defaites: 4},
		{MapID: "b", MapName: "Bazaar", Matchs: domain.PlancherMatchsParCarte - 1, Victoires: 5, Defaites: 4},
	}}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)

	page, err := svc.MapsPlayed(context.Background(), domain.TacticalScope{})
	if err != nil {
		t.Fatalf("MapsPlayed: %v", err)
	}
	if page.PlancherMatchs != domain.PlancherMatchsParCarte {
		t.Errorf("PlancherMatchs = %d, want %d", page.PlancherMatchs, domain.PlancherMatchsParCarte)
	}
	if len(page.Cartes) != 2 {
		t.Fatalf("cartes = %d, want 2", len(page.Cartes))
	}
	if page.Cartes[0].SousPlancher {
		t.Errorf("carte a %d matchs : le plancher est ATTEINT, pas franchi", page.Cartes[0].Matchs)
	}
	if !page.Cartes[1].SousPlancher {
		t.Errorf("carte a %d matchs : sous le plancher attendu", page.Cartes[1].Matchs)
	}
	if page.Cartes[0].Victoires != 6 || page.Cartes[0].Defaites != 4 {
		t.Errorf("V/D non transmis : %+v", page.Cartes[0])
	}
}

// TestTacticalService_MapsPlayed_PerimetreTransmis : la grille d'entree porte le
// MEME perimetre que les rasters — liste blanche et composition descendent au lecteur
// tel quel, et la carte n'y est jamais posee (l'ecran porte sur toutes les cartes).
// Sans cela, la grille proposerait des cartes qui n'ont aucun match une fois ouvertes.
func TestTacticalService_MapsPlayed_PerimetreTransmis(t *testing.T) {
	repo := &mockTacticalRepo{}

	svc := NewTacticalService(repo, capsCompletes(), tsMoi)
	scope := domain.TacticalScope{MatchIDs: []string{"m1", "m2"}, Coequipiers: []string{tsAmi}}
	if _, err := svc.MapsPlayed(context.Background(), scope); err != nil {
		t.Fatalf("MapsPlayed: %v", err)
	}
	if !repo.vuMaps.Matchs.Restreint() || !egalesXUID(repo.vuMaps.Matchs.IDs(), []string{"m1", "m2"}) {
		t.Errorf("liste blanche = %v (restreinte=%v), want [m1 m2]",
			repo.vuMaps.Matchs.IDs(), repo.vuMaps.Matchs.Restreint())
	}
	if !egalesXUID(repo.vuMaps.Coequipiers, []string{tsAmi}) {
		t.Errorf("composition = %v, want [%s]", repo.vuMaps.Coequipiers, tsAmi)
	}
	if repo.vuMaps.PlayerXUID != tsMoi {
		t.Errorf("joueur transmis = %q, want %q", repo.vuMaps.PlayerXUID, tsMoi)
	}
	if repo.vuMaps.MapID != "" {
		t.Errorf("MapID = %q, want vide : la grille porte sur TOUTES les cartes", repo.vuMaps.MapID)
	}
}

// TestTacticalService_SansLecteur : un titre sans lecteur cable degrade en
// capability absente, jamais en panique.
func TestTacticalService_SansLecteur(t *testing.T) {
	repo := &mockTacticalRepo{}
	svc := NewTacticalService(nil, capsCompletes(), tsMoi)
	if _, err := svc.MapsPlayed(context.Background(), domain.TacticalScope{}); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("MapsPlayed sans lecteur: err = %v, want ErrCapabilityNotSupported", err)
	}
	if _, err := svc.Raster(context.Background(), tsDemande(repo, tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi)); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("Raster sans lecteur: err = %v, want ErrCapabilityNotSupported", err)
	}
}
