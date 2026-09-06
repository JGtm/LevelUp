// Package service — tactical_service_echange_test.go : les PORTES de capability,
// le KPI d'echange, et la grille d'entree.
//
// Scinde de tactical_service_test.go le 2026-09-06 : celui-ci avait franchi les 500
// lignes sous les ajouts de la revue (meme discipline que la scission de
// merge_test.go en phase 1). Le mock du port et les fabriques de capabilities
// restent chez le voisin, meme paquet.
package service

import (
	"context"
	"errors"
	"math"
	"testing"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// ─── LE KPI D'ECHANGE ──────────────────────────────────────────────────────────

// TestTacticalService_Echange_EchantillonFaible : le KPI est une domain.Couverture
// (jamais un taux nu), et sous 30 morts vengeables il porte le drapeau
// d'echantillon faible — il se lit, il ne classe personne.
func TestTacticalService_Echange_EchantillonFaible(t *testing.T) {
	repo := &mockTacticalRepo{pos: domain.TacticalPositions{Univers: universUnMatch("m1", domain.OutcomeWin)}}
	repo.ev = domain.TacticalKillEvents{Univers: universUnMatch("m1", domain.OutcomeWin)}
	// Deux morts de MON camp : la mienne (vengee par l'ami en 3 s) et celle de
	// l'ami (non vengee). Plus une mort adverse, qui ne doit PAS entrer au
	// denominateur de ma couverture.
	repo.ev.Events = []domain.KillEvent{
		{MatchID: "m1", KillerXUID: tsAdv, VictimXUID: tsMoi, TimeMs: 1000},
		{MatchID: "m1", KillerXUID: tsAmi, VictimXUID: tsAdv, TimeMs: 4000},
		{MatchID: "m1", KillerXUID: tsAdv2, VictimXUID: tsAmi, TimeMs: 20000},
		// T6 — une mort de MON CAMP que PERSONNE ne revendique (chute, hors-limites,
		// grenade perdue). Elle n'est pas vengeable : la compter comme un echec
		// punirait l'equipe pour une mort que personne ne pouvait echanger.
		{MatchID: "m1", KillerXUID: "", VictimXUID: tsMoi, TimeMs: 30000},
	}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)

	got, err := svc.Raster(context.Background(), tsDemande(tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
	if err != nil {
		t.Fatalf("Raster: %v", err)
	}
	if got.Echange == nil {
		t.Fatal("Echange nil alors que film.kill_source est declaree")
	}
	if got.Echange.N != 2 || got.Echange.Brut != 1 {
		t.Errorf("couverture = %d/%d, want 1 vengee sur 2 vengeables DE MON CAMP "+
			"(ni la mort de l'adversaire ni ma mort SANS TUEUR n'y entrent)",
			got.Echange.Brut, got.Echange.N)
	}
	if math.Abs(got.Echange.Taux-0.5) > 1e-9 {
		t.Errorf("Taux = %v, want 0,5 (unite 0..1, jamais un pourcentage)", got.Echange.Taux)
	}
	if !got.Echange.EchantillonFaible {
		t.Errorf("2 morts vengeables (< %d) : echantillon faible attendu", coordination.SeuilEchantillonFaible)
	}
	if got.Echange.ParMatch != 1.0 {
		t.Errorf("ParMatch = %v, want 1,0 (1 vengee sur 1 match retenu)", got.Echange.ParMatch)
	}
}

// TestTacticalService_SansKillSource_EchangeSilencieux : le titre mesure les
// positions mais pas la source des morts — la lecture de placement est servie, le
// KPI est ABSENT. Un zero se lirait comme une contre-performance.
func TestTacticalService_SansKillSource_EchangeSilencieux(t *testing.T) {
	repo := &mockTacticalRepo{pos: domain.TacticalPositions{Univers: universUnMatch("m1", domain.OutcomeWin)}}
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)

	got, err := svc.Raster(context.Background(), tsDemande(tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
	if err != nil {
		t.Fatalf("Raster: %v", err)
	}
	if got.Echange != nil {
		t.Errorf("Echange = %+v, want nil (aucune provenance de journal fiable)", got.Echange)
	}
	// Le journal EST lu quand meme : la couverture de localisation est une propriete
	// de la MESURE, pas un KPI — elle ne depend d'aucune capability.
	if repo.vuEv.MapID != tsCarte {
		t.Error("le journal des morts doit etre lu pour la couverture, meme sans KPI d'echange")
	}
}

// TestTacticalService_EchangeEnEchec_LectureServie : une panne du journal des
// morts est journalisee et degrade le SEUL KPI — elle n'emporte pas la lecture de
// placement avec elle.
func TestTacticalService_EchangeEnEchec_LectureServie(t *testing.T) {
	repo := &mockTacticalRepo{
		pos:   domain.TacticalPositions{Univers: universUnMatch("m1", domain.OutcomeWin)},
		errEv: errors.New("journal illisible"),
	}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)

	got, err := svc.Raster(context.Background(), tsDemande(tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
	if err != nil {
		t.Fatalf("Raster: %v, want la lecture servie malgre l'echec du KPI", err)
	}
	if got.Echange != nil {
		t.Errorf("Echange = %+v, want nil", got.Echange)
	}
	if got.MatchsRetenus != 1 {
		t.Errorf("MatchsRetenus = %d, want 1", got.MatchsRetenus)
	}
}

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
		_, err := svc.Raster(context.Background(), tsDemande(tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
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

	got, err := svc.Raster(context.Background(), tsDemande(tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
	if err != nil {
		t.Fatalf("positions NATIVES (match.events.spatial) : err = %v, want une lecture servie", err)
	}
	if len(got.Cellules) != 1 || celluleEn(got.Cellules, 10.0, 10.0) == nil {
		t.Errorf("cellules = %+v, want la cellule (10,10)", got.Cellules)
	}
}

// TestTacticalService_EchangeDeuxProvenances : le journal des morts est exploitable
// soit par la source de degat du film (`film.kill_source`), soit par un kill-feed
// natif declare `supported` — mais PAS `degraded`.
//
// POURQUOI LE STRICT : `Has` accepte `degraded`, et Halo Infinite declare justement
// `match.killfeed.per_kill = degraded` (kills simultanes possiblement omis) — soit
// exactement le defaut qui fabrique de faux echanges, une mort omise dans la
// fenetre de 5 s se lisant « non vengee ».
func TestTacticalService_EchangeDeuxProvenances(t *testing.T) {
	cas := []struct {
		nom  string
		caps games.CapabilityMap
		veut bool
	}{
		{"source de degat du film", games.CapabilityMap{
			games.CapMatchEventsSpatial: games.CapSupported,
			games.CapFilmKillSource:     games.CapSupported}, true},
		{"kill-feed natif supported", games.CapabilityMap{
			games.CapMatchEventsSpatial:   games.CapSupported,
			games.CapMatchKillfeedPerKill: games.CapSupported}, true},
		{"kill-feed natif DEGRADED", games.CapabilityMap{
			games.CapMatchEventsSpatial:   games.CapSupported,
			games.CapMatchKillfeedPerKill: games.CapDegraded}, false},
		{"aucune des deux", games.CapabilityMap{
			games.CapMatchEventsSpatial: games.CapSupported}, false},
	}
	for _, c := range cas {
		repo := &mockTacticalRepo{pos: domain.TacticalPositions{Univers: universUnMatch("m1", domain.OutcomeWin)}}
		repo.ev = domain.TacticalKillEvents{Univers: universUnMatch("m1", domain.OutcomeWin)}
		svc := NewTacticalService(repo, c.caps, tsMoi)
		got, err := svc.Raster(context.Background(), tsDemande(tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
		if err != nil {
			t.Fatalf("%s: %v", c.nom, err)
		}
		if (got.Echange != nil) != c.veut {
			t.Errorf("%s: Echange servi = %v, want %v", c.nom, got.Echange != nil, c.veut)
		}
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
	svc := NewTacticalService(nil, capsCompletes(), tsMoi)
	if _, err := svc.MapsPlayed(context.Background(), domain.TacticalScope{}); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("MapsPlayed sans lecteur: err = %v, want ErrCapabilityNotSupported", err)
	}
	if _, err := svc.Raster(context.Background(), tsDemande(tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi)); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("Raster sans lecteur: err = %v, want ErrCapabilityNotSupported", err)
	}
}
