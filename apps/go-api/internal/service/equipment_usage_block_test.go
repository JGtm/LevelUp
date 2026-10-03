package service

// equipment_usage_block_test.go — l'orchestration du bloc « servi ou gâché » au
// grain période (étapes E5.5 et E6.1) : capability absente ⇒ raison machine et
// jamais d'échec, erreur de lecture ⇒ load_failed, scope vide ⇒ pas de bloc, et
// les amis configurés qui deviennent les parts « mes amis » des deux donuts.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// overviewRepoMock — un match mesuré à camp connu : moi (P), un ami (Alpha), un
// allié non déclaré (Bravo), un adversaire (Echo).
//
// P PORTE DEUX FAMILLES ET DEUX CANAUX (correction C1, 2026-09-10) : 3 murs, lus sur
// leurs POSES (seule famille qui engendre une pièce), et 2 capteurs, lus sur leurs
// CHARGES CONSOMMÉES. Une fixture qui ne porterait que du mur laisserait passer
// l'ancienne règle — celle qui lisait les poses pour TOUTES les familles.
func overviewRepoMock() *mockSessionUsageRepo {
	return &mockSessionUsageRepo{
		films: map[string]sessionusage.FilmRow{"m1": {MatchID: "m1", DurationMS: 600000}},
		players: []sessionusage.PlayerRow{
			{
				MatchID: "m1", XUID: "P", PadPickups: 2,
				DeployedByFamily: map[string]int{"wall": 1},
				SpentByFamily:    map[string]int{"sensor": 2},
				TakenByFamily:    map[string]int{"wall": 3, "sensor": 2},
				DroppedByFamily:  map[string]int{"wall": 1},
				KeptByFamily:     map[string]int{"wall": 1},
			},
			{MatchID: "m1", XUID: "A", PadPickups: 1, DeployedByFamily: map[string]int{"wall": 2}},
			{MatchID: "m1", XUID: "B", PadPickups: 4, DroppedByFamily: map[string]int{"wall": 1}},
			{MatchID: "m1", XUID: "E1", PadPickups: 3, DroppedByFamily: map[string]int{"wall": 5}},
		},
		participants: []sessionusage.ParticipantRow{
			{MatchID: "m1", XUID: "P", Gamertag: "Papa", TeamID: teamp(0), PresentAtCompletion: true},
			{MatchID: "m1", XUID: "A", Gamertag: "Alpha", TeamID: teamp(0), PresentAtCompletion: true},
			{MatchID: "m1", XUID: "B", Gamertag: "Bravo", TeamID: teamp(0), PresentAtCompletion: true},
			{MatchID: "m1", XUID: "E1", Gamertag: "Echo", TeamID: teamp(1), PresentAtCompletion: true},
		},
	}
}

func TestEquipmentUsageBlock_AmisConfiguresEtPartsDuDonut(t *testing.T) {
	block := buildEquipmentUsageBlock(context.Background(), equipmentUsageQuery{
		Repo: overviewRepoMock(), PlayerXUID: "P",
		MatchIDs: []string{"m1", "m2"}, FriendGamertags: []string{"Alpha"},
	})
	if block == nil || !block.Available {
		t.Fatalf("bloc = %+v, attendu disponible", block)
	}
	if block.MatchesMeasured != 1 || block.MatchesTotal != 2 {
		t.Errorf("couverture = %d/%d, attendu 1/2", block.MatchesMeasured, block.MatchesTotal)
	}
	if len(block.TrackedPlayers) != 1 || block.TrackedPlayers[0].Gamertag != "Alpha" {
		t.Fatalf("coéquipiers suivis = %+v, attendu [Alpha] (Bravo n'est pas un ami configuré)",
			block.TrackedPlayers)
	}
	eq := block.EquipmentParties
	if eq == nil {
		t.Fatal("comptes du donut équipement absents")
	}
	// moi 5 (3 murs + 2 capteurs CONSOMMÉS), l'ami 2, le reste de l'équipe (Bravo) 1,
	// eux 5 -> lobby 13. Lus sur les poses, les capteurs vaudraient 0 et le donut 11.
	if eq.Player != 5 || eq.Friends != 2 || eq.RestOfTeam != 1 || eq.Opponents != 5 || eq.LobbyTotal != 13 {
		t.Errorf("parts = %+v, attendu (moi 5, amis 2, reste 1, eux 5, lobby 13)", eq)
	}
	// Deux lignes joueur : moi puis l'ami suivi (publication E6.1).
	if len(block.Players) != 2 || block.Players[0].XUID != "P" || block.Players[1].XUID != "A" {
		t.Errorf("lignes joueur = %+v, attendu [P, A]", block.Players)
	}
}

func TestEquipmentUsageBlock_CapabiliteAbsenteEtScopeVide(t *testing.T) {
	// Repo nil = titre sans film.usage_summary : réponse partielle propre.
	block := buildEquipmentUsageBlock(context.Background(), equipmentUsageQuery{
		PlayerXUID: "P", MatchIDs: []string{"m1"},
	})
	if block == nil || block.Available || block.UnavailableReason != domain.SessionUsageUnsupported {
		t.Errorf("bloc = %+v, attendu indisponible/unsupported", block)
	}
	// Scope sans match : rien à publier du tout.
	if got := buildEquipmentUsageBlock(context.Background(), equipmentUsageQuery{
		Repo: overviewRepoMock(), PlayerXUID: "P",
	}); got != nil {
		t.Errorf("bloc = %+v, attendu nil sur un scope vide", got)
	}
}

func TestEquipmentUsageBlock_ErreurDeLectureDegradeSansEchouer(t *testing.T) {
	repo := overviewRepoMock()
	repo.filmsErr = errors.New("vue indisponible")
	block := buildEquipmentUsageBlock(context.Background(), equipmentUsageQuery{
		Repo: repo, PlayerXUID: "P", MatchIDs: []string{"m1"},
	})
	if block == nil || block.Available || block.UnavailableReason != domain.SessionUsageLoadFailed {
		t.Errorf("bloc = %+v, attendu indisponible/load_failed", block)
	}
}

// TestTimeseriesPage_AttacheLeBlocEquipement — le bloc voyage avec la réponse EXISTANTE de
// la page Séries temporelles (jamais un endpoint dédié), sur le scope FILTRÉ, et les amis
// configurés y deviennent la part « mes amis ». Il a quitté la Synthèse le 2026-09-13 :
// c'est l'onglet Progression qui l'affiche désormais, mais ni le producteur ni le scope
// n'ont changé.
func TestTimeseriesPage_AttacheLeBlocEquipement(t *testing.T) {
	svc := NewTimeseriesService(nil).
		WithEquipmentUsage(overviewRepoMock(), func(context.Context) []string { return []string{"Alpha"} }, "")
	svc.playerXUID = "P"

	var resp domain.TimeseriesPageResponse
	svc.attachMigratedSections(context.Background(), &resp, eqUsageCanonRows("m1"), "fr", equipesDuScope{})

	block := resp.EquipmentUsage
	if block == nil || !block.Available {
		t.Fatalf("bloc équipement = %+v, attendu disponible", block)
	}
	if block.MatchesMeasured != 1 || block.MatchesTotal != 1 {
		t.Errorf("couverture = %d/%d, attendu 1/1", block.MatchesMeasured, block.MatchesTotal)
	}
	if len(block.TrackedPlayers) != 1 || block.TrackedPlayers[0].Gamertag != "Alpha" {
		t.Errorf("coéquipiers suivis = %+v, attendu [Alpha]", block.TrackedPlayers)
	}
	wall := block.Families[0]
	if wall.FamilyKey != "wall" || wall.Used != 1 || wall.Kept != 1 || wall.Dropped != 1 {
		t.Errorf("première famille = %+v, attendu wall (1, 1, 1)", wall)
	}
	// La deuxième famille est le CAPTEUR, servi sur ses charges consommées : sans elle, la
	// page n'aurait aucune ligne pour un équipement pris deux fois et utilisé deux fois
	// (constat C1 de la revue de la vague 5).
	if len(block.Families) != 2 {
		t.Fatalf("familles = %+v, attendu deux lignes (mur puis capteur)", block.Families)
	}
	sensor := block.Families[1]
	if sensor.FamilyKey != "sensor" || sensor.Used != 2 || sensor.Kept != 0 || sensor.Dropped != 0 {
		t.Errorf("deuxième famille = %+v, attendu sensor (2, 0, 0)", sensor)
	}
}

// TestTimeseriesPage_SansCapabiliteLeBlocDitPourquoi — titre sans film.usage_summary (repo
// non câblé) : réponse partielle propre, jamais un 500 ni un bloc muet. La portée des
// engagements, elle, s'OMET (nil) : deux contrats de dégradation distincts, voulus.
func TestTimeseriesPage_SansCapabiliteLeBlocDitPourquoi(t *testing.T) {
	svc := NewTimeseriesService(nil)
	svc.playerXUID = "P"

	var resp domain.TimeseriesPageResponse
	svc.attachMigratedSections(context.Background(), &resp, eqUsageCanonRows("m1"), "fr", equipesDuScope{})

	if resp.EquipmentUsage == nil || resp.EquipmentUsage.Available ||
		resp.EquipmentUsage.UnavailableReason != domain.SessionUsageUnsupported {
		t.Errorf("bloc = %+v, attendu indisponible/unsupported", resp.EquipmentUsage)
	}
	if resp.WeaponRange != nil {
		t.Errorf("portée = %+v, attendu nil (repo non câblé)", resp.WeaponRange)
	}
}

// eqUsageCanonRows — un scope canonique minimal : seuls les match_id comptent ici.
func eqUsageCanonRows(ids ...string) []canonical.PlayerMatchRow {
	rows := make([]canonical.PlayerMatchRow, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, canonical.PlayerMatchRow{Summary: canonical.MatchSummary{MatchID: id}})
	}
	return rows
}

// TestEquipmentUsageBlock_LitLesNiveauxDArmes — LE SECOND CHEMIN DE LECTURE, PROUVE.
//
// Constat de revue (2026-09-14) : ce chemin n'etait garde que par un GREP sur le source. Une
// mutation en no-op (`attacherNiveauxDArmes` qui ne fait rien) laissait `squadagg` et
// `teammates` verts, et les pages Escouade et Timeseries n auraient jamais montre un niveau.
//
// Ce test passe par le VRAI producteur avec un repo qui rend des lignes, et verifie que le
// bloc arrive au contrat avec SES propres denominateurs.
func TestEquipmentUsageBlock_LitLesNiveauxDArmes(t *testing.T) {
	repo := overviewRepoMock()
	repo.padTiers = []sessionusage.PadTierRow{
		{
			MatchID: "m1", XUID: "P", Tier: domain.PadTierPower,
			WeaponFamily: "9d6aaed2", Pickups: 3, PadsConfirmed: 4, PadsTotal: 5,
		},
		{MatchID: "m1", XUID: "A", Tier: domain.PadTierNoPickup, PadsConfirmed: 4, PadsTotal: 5},
	}
	block := buildEquipmentUsageBlock(context.Background(), equipmentUsageQuery{
		Repo: repo, PlayerXUID: "P", MatchIDs: []string{"m1"},
	})
	if block == nil || block.PadTiers == nil {
		t.Fatal("le bloc des niveaux d'armes n'est pas publie sur le chemin Escouade/Timeseries : " +
			"la lecture produit y est debranchee, et la table se remplirait sans qu'aucun de ces " +
			"deux ecrans ne la lise")
	}
	// LES DENOMINATEURS SONT LES SIENS, jamais ceux du resume d'usage voisin.
	if block.PadTiers.MatchesTotal != 1 || block.PadTiers.MatchesMeasured != 1 {
		t.Errorf("couverture des niveaux = %d/%d, attendu 1/1",
			block.PadTiers.MatchesMeasured, block.PadTiers.MatchesTotal)
	}
	if len(block.PadTiers.Tiers) != 1 || block.PadTiers.Tiers[0].Tier != domain.PadTierPower {
		t.Fatalf("niveaux publies : %+v", block.PadTiers.Tiers)
	}
	if block.PadTiers.Tiers[0].PlayerTotal != 3 {
		t.Errorf("prises du joueur = %v, attendu 3", block.PadTiers.Tiers[0].PlayerTotal)
	}
}

// TestEquipmentUsageBlock_SansLigneAucunBlocDeNiveaux — « pas encore mesure » ne se publie pas
// comme « zero ».
func TestEquipmentUsageBlock_SansLigneAucunBlocDeNiveaux(t *testing.T) {
	block := buildEquipmentUsageBlock(context.Background(), equipmentUsageQuery{
		Repo: overviewRepoMock(), PlayerXUID: "P", MatchIDs: []string{"m1"},
	})
	if block != nil && block.PadTiers != nil {
		t.Errorf("bloc de niveaux servi sans aucune ligne : %+v", block.PadTiers)
	}
}
