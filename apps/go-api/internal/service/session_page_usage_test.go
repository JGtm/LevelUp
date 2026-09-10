package service

// session_page_usage_test.go — l'attachement du bloc usage à la page session :
// capability absente (repo nil) ⇒ Available=false avec raison machine, jamais
// d'échec ; erreur de lecture ⇒ load_failed ; contexte escouade ⇒ coéquipiers
// suivis + lignes squad ; sous-bloc objectifs via le loader optionnel.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

type mockSessionUsageRepo struct {
	films        map[string]sessionusage.FilmRow
	players      []sessionusage.PlayerRow
	participants []sessionusage.ParticipantRow
	filmsErr     error
}

func (m *mockSessionUsageRepo) LoadUsageFilms(_ context.Context, _ []string) (map[string]sessionusage.FilmRow, error) {
	return m.films, m.filmsErr
}
func (m *mockSessionUsageRepo) LoadUsagePlayers(_ context.Context, _ []string) ([]sessionusage.PlayerRow, error) {
	return m.players, nil
}
func (m *mockSessionUsageRepo) LoadParticipants(_ context.Context, _ []string) ([]sessionusage.ParticipantRow, error) {
	return m.participants, nil
}

// mockObjectiveIndexWithRoles implémente port.ObjectiveIndexRepository ET la
// capability optionnelle objectiveRoleRowsLoader (comme duckdb.ObjectiveStatsRepo).
type mockObjectiveIndexWithRoles struct {
	roleRows []sessionusage.ObjectiveRow
}

func (m *mockObjectiveIndexWithRoles) LoadObjectiveIndexInputs(_ context.Context, _, _ []string) (map[string]narrative.ObjectiveIndexInput, error) {
	return map[string]narrative.ObjectiveIndexInput{}, nil
}
func (m *mockObjectiveIndexWithRoles) LoadObjectiveIndexInputsByGamertag(_ context.Context, _ []string, _ string) (narrative.ObjectiveIndexInput, error) {
	return narrative.ObjectiveIndexInput{}, nil
}
func (m *mockObjectiveIndexWithRoles) LoadObjectiveRoleRows(_ context.Context, _ []string) ([]sessionusage.ObjectiveRow, error) {
	return m.roleRows, nil
}

func usageTestMatches() []legacymatch.StatsMatchRow {
	return []legacymatch.StatsMatchRow{{MatchID: "m1"}, {MatchID: "m2"}}
}

func teamp(v int) *int { return &v }

func usageTestRepoMock() *mockSessionUsageRepo {
	return &mockSessionUsageRepo{
		films: map[string]sessionusage.FilmRow{
			"m1": {MatchID: "m1", DurationMS: 600000, PadUnnamed: 2},
		},
		players: []sessionusage.PlayerRow{
			{MatchID: "m1", XUID: "P", PadPickups: 2},
			{MatchID: "m1", XUID: "A", PadPickups: 1},
			{MatchID: "m1", XUID: "E1", PadPickups: 1},
		},
		participants: []sessionusage.ParticipantRow{
			{MatchID: "m1", XUID: "P", Gamertag: "Papa", TeamID: teamp(0), PresentAtCompletion: true},
			{MatchID: "m1", XUID: "A", Gamertag: "Alpha", TeamID: teamp(0), PresentAtCompletion: true},
			{MatchID: "m1", XUID: "E1", Gamertag: "Echo", TeamID: teamp(1), PresentAtCompletion: true},
			{MatchID: "m2", XUID: "P", Gamertag: "Papa", TeamID: teamp(0), PresentAtCompletion: true},
			{MatchID: "m2", XUID: "A", Gamertag: "Alpha", TeamID: teamp(0), PresentAtCompletion: true},
			{MatchID: "m2", XUID: "E1", Gamertag: "Echo", TeamID: teamp(1), PresentAtCompletion: true},
		},
	}
}

func TestAttachSessionUsage_CapabilityAbsente(t *testing.T) {
	svc := NewSessionPageService(nil) // repo usage jamais câblé
	var resp domain.SessionPageResponse
	svc.attachSessionUsage(context.Background(), &resp, usageTestMatches(), nil, domain.MatchContextSolo, "fr")
	if resp.Usage == nil {
		t.Fatal("Usage nil : le bloc doit être présent avec Available=false, pas absent")
	}
	if resp.Usage.Available || resp.Usage.UnavailableReason != domain.SessionUsageUnsupported {
		t.Errorf("bloc = %+v, attendu Available=false, raison %q", resp.Usage, domain.SessionUsageUnsupported)
	}
	if resp.Usage.MatchesTotal != 2 {
		t.Errorf("matches_total = %d, attendu 2 (le dénominateur reste dit)", resp.Usage.MatchesTotal)
	}
}

func TestAttachSessionUsage_SessionSansMatch(t *testing.T) {
	svc := NewSessionPageService(nil).WithSessionUsage(usageTestRepoMock(), "P", nil, "")
	var resp domain.SessionPageResponse
	svc.attachSessionUsage(context.Background(), &resp, nil, nil, domain.MatchContextSolo, "fr")
	if resp.Usage != nil {
		t.Errorf("Usage = %+v, attendu nil (session sans match)", resp.Usage)
	}
}

func TestAttachSessionUsage_ErreurDeLectureDegrade(t *testing.T) {
	repo := usageTestRepoMock()
	repo.filmsErr = errors.New("boom")
	svc := NewSessionPageService(nil).WithSessionUsage(repo, "P", nil, "")
	var resp domain.SessionPageResponse
	svc.attachSessionUsage(context.Background(), &resp, usageTestMatches(), nil, domain.MatchContextSolo, "fr")
	if resp.Usage == nil || resp.Usage.Available || resp.Usage.UnavailableReason != domain.SessionUsageLoadFailed {
		t.Errorf("bloc = %+v, attendu Available=false raison %q", resp.Usage, domain.SessionUsageLoadFailed)
	}
}

func TestAttachSessionUsage_ContexteEscouade(t *testing.T) {
	svc := NewSessionPageService(nil).
		WithSessionUsage(usageTestRepoMock(), "P", func(context.Context) []string { return []string{"Alpha"} }, "")
	svc.objectiveIndex = &mockObjectiveIndexWithRoles{roleRows: []sessionusage.ObjectiveRow{
		{MatchID: "m1", XUID: "P", Family: narrative.FamilyCTF, Take: 2},
		{MatchID: "m1", XUID: "A", Family: narrative.FamilyCTF, Take: 1},
	}}
	var resp domain.SessionPageResponse
	svc.attachSessionUsage(context.Background(), &resp, usageTestMatches(), nil, domain.MatchContextSquad, "fr")
	u := resp.Usage
	if u == nil || !u.Available {
		t.Fatalf("Usage = %+v, attendu bloc disponible", u)
	}
	if u.MatchesMeasured != 1 || u.MatchesTotal != 2 {
		t.Errorf("couverture = %d/%d, attendu 1/2", u.MatchesMeasured, u.MatchesTotal)
	}
	if len(u.SquadPlayers) != 1 || u.SquadPlayers[0].XUID != "A" || u.SquadPlayers[0].Gamertag != "Alpha" {
		t.Fatalf("squad_players = %+v, attendu [A/Alpha] (Echo est adverse)", u.SquadPlayers)
	}
	var pad *domain.SessionUsageMetric
	for i := range u.Metrics {
		if u.Metrics[i].Key == sessionusage.MetricPadPickups {
			pad = &u.Metrics[i]
		}
	}
	if pad == nil || len(pad.Squad) != 1 || pad.Squad[0].XUID != "A" || pad.Squad[0].Total != 1 {
		t.Errorf("lignes squad pad_pickups = %+v, attendu total 1 pour A", pad)
	}
	if u.Objectives == nil || len(u.Objectives.Roles) == 0 {
		t.Fatalf("objectifs = %+v, attendu sous-bloc via le loader optionnel", u.Objectives)
	}
	if u.Objectives.Roles[0].Squad == nil {
		t.Error("les rôles d'objectif doivent porter les lignes squad en contexte escouade")
	}
}

func TestAttachSessionUsage_ContexteSoloSansLigneSquad(t *testing.T) {
	svc := NewSessionPageService(nil).WithSessionUsage(usageTestRepoMock(), "P", nil, "")
	var resp domain.SessionPageResponse
	svc.attachSessionUsage(context.Background(), &resp, usageTestMatches(), nil, domain.MatchContextSolo, "fr")
	u := resp.Usage
	if u == nil || !u.Available {
		t.Fatalf("Usage = %+v, attendu bloc disponible", u)
	}
	if len(u.SquadPlayers) != 0 {
		t.Errorf("squad_players = %+v, attendu vide en solo", u.SquadPlayers)
	}
	for _, m := range u.Metrics {
		if len(m.Squad) != 0 {
			t.Errorf("métrique %s porte des lignes squad en solo : %+v", m.Key, m.Squad)
		}
	}
	if u.Objectives != nil {
		t.Errorf("objectifs = %+v, attendu nil (aucun loader câblé)", u.Objectives)
	}
}

// ─── Session comparée (D8, plan de lisibilité 2026-09-09) ────────────────────────

// Le drawer de comparaison n'affichait rien à droite parce que le bloc n'était calculé
// que pour la session courante. Ces deux tests fixent la règle : le bloc comparé est
// servi quand — et seulement quand — des matchs comparés sont passés.
func TestAttachSessionUsage_SessionCompareeServieQuandLeDrawerEstOuvert(t *testing.T) {
	svc := NewSessionPageService(nil).WithSessionUsage(usageTestRepoMock(), "P", nil, "")
	var resp domain.SessionPageResponse
	svc.attachSessionUsage(context.Background(), &resp, usageTestMatches(), usageTestMatches(),
		domain.MatchContextSolo, "fr")
	if resp.Usage == nil || resp.CompareUsage == nil {
		t.Fatalf("Usage = %v, CompareUsage = %v : les deux colonnes doivent porter un bloc",
			resp.Usage, resp.CompareUsage)
	}
	// Les deux blocs sont calculés séparément : ce ne doit JAMAIS être le même pointeur
	// (un bloc partagé se mettrait à mentir dès que les deux sessions divergent).
	if resp.Usage == resp.CompareUsage {
		t.Error("Usage et CompareUsage partagent le même bloc")
	}
	if resp.CompareUsage.MatchesTotal != len(usageTestMatches()) {
		t.Errorf("compare matches_total = %d, attendu %d",
			resp.CompareUsage.MatchesTotal, len(usageTestMatches()))
	}
}

func TestAttachSessionUsage_DrawerFermeNeSertAucunBlocCompare(t *testing.T) {
	svc := NewSessionPageService(nil).WithSessionUsage(usageTestRepoMock(), "P", nil, "")
	var resp domain.SessionPageResponse
	svc.attachSessionUsage(context.Background(), &resp, usageTestMatches(), nil,
		domain.MatchContextSolo, "fr")
	if resp.CompareUsage != nil {
		t.Errorf("CompareUsage = %+v, attendu nil hors comparaison", resp.CompareUsage)
	}
}

// TestAttachSessionUsage_LesTroisIssuesRemontentAuContrat — étape E3 : les quatre
// ventilations lues par le repo (taken/spent/kept/dropped) deviennent une grandeur
// "equipment_<famille>" avec son remplissage et ses deux taux de référence. Le
// service ne calcule rien lui-même : ce test vérifie que rien ne se perd EN ROUTE
// entre la ligne de base et le bloc servi.
//
// DEUX FAMILLES, DEUX CANAUX (correction C1, 2026-09-10) : le mur se lit sur ses
// POSES (seule famille qui engendre une pièce), le capteur sur ses CONSOMMATIONS.
// La colonne `spent_json` est chargée par le repo depuis `us4` — jusqu'à cette
// correction elle n'avait AUCUN lecteur et le capteur remontait « utilisé 0 ».
func TestAttachSessionUsage_LesTroisIssuesRemontentAuContrat(t *testing.T) {
	repo := usageTestRepoMock()
	repo.players = []sessionusage.PlayerRow{
		{
			MatchID: "m1", XUID: "P",
			DeployedByFamily: map[string]int{"wall": 1},
			SpentByFamily:    map[string]int{"sensor": 2},
			TakenByFamily:    map[string]int{"wall": 3, "sensor": 3},
			KeptByFamily:     map[string]int{"wall": 1},
			DroppedByFamily:  map[string]int{"wall": 1, "sensor": 1},
		},
		// L'allié utilise tout ce qu'il prend : la référence « reste de mon
		// équipe » vaut 100 %, et elle ne me contient pas (décision P7).
		{
			MatchID: "m1", XUID: "A",
			DeployedByFamily: map[string]int{"wall": 2},
			TakenByFamily:    map[string]int{"wall": 2},
		},
		// L'adversaire ne fait que lâcher : « eux » vaut 0 %.
		{
			MatchID: "m1", XUID: "E1",
			TakenByFamily:   map[string]int{"wall": 2},
			DroppedByFamily: map[string]int{"wall": 2},
		},
	}
	svc := NewSessionPageService(nil).WithSessionUsage(repo, "P", nil, "")
	var resp domain.SessionPageResponse
	svc.attachSessionUsage(context.Background(), &resp, usageTestMatches(), nil, domain.MatchContextSolo, "fr")
	if resp.Usage == nil || !resp.Usage.Available {
		t.Fatalf("Usage = %+v, attendu bloc disponible", resp.Usage)
	}
	m := grandeurEquipement(t, resp.Usage.Metrics, "wall")
	if m.Outcomes == nil {
		t.Fatal("equipment_wall servie SANS ses issues — le remplissage de la barre est perdu")
	}
	o := m.Outcomes
	if o.Used != 1 || o.Kept != 1 || o.Dropped != 1 || o.Taken != 3 {
		t.Errorf("issues servies = %+v, attendu 1/1/1 pour 3 prises", o)
	}
	if o.TeammatesUsedRatePct == nil || *o.TeammatesUsedRatePct != 100 {
		t.Errorf("TeammatesUsedRatePct = %v, attendu 100 %%", o.TeammatesUsedRatePct)
	}
	if o.OpponentsUsedRatePct == nil || *o.OpponentsUsedRatePct != 0 {
		t.Errorf("OpponentsUsedRatePct = %v, attendu 0 %%", o.OpponentsUsedRatePct)
	}

	// LE CAPTEUR : aucune pose, deux charges consommées, un lâcher. Servi sur les
	// poses, il remonterait « utilisé 0 · gardé 0 · lâché 1 » pour une barre de 1
	// au lieu de 3 (constat C1 de la revue de la vague 5).
	sensor := grandeurEquipement(t, resp.Usage.Metrics, "sensor")
	if sensor.Outcomes == nil {
		t.Fatal("equipment_sensor servie SANS ses issues")
	}
	so := sensor.Outcomes
	if so.Used != 2 || so.Kept != 0 || so.Dropped != 1 || so.Taken != 3 {
		t.Errorf("issues du capteur = %+v, attendu 2 utilisés / 0 gardé / 1 lâché pour 3 prises", so)
	}
	if sensor.PlayerTotal != 3 {
		t.Errorf("PlayerTotal du capteur = %v, attendu 3", sensor.PlayerTotal)
	}
}

// grandeurEquipement — la grandeur "equipment_<famille>" du bloc servi, ou un échec
// nommé : une famille ABSENTE est le symptôme même du constat C1 (metricKeys
// n'ouvre une ligne que sur un total non nul).
func grandeurEquipement(
	t *testing.T, metrics []domain.SessionUsageMetric, family string,
) *domain.SessionUsageMetric {
	t.Helper()
	for i := range metrics {
		if metrics[i].Key == sessionusage.MetricEquipmentPrefix+family {
			return &metrics[i]
		}
	}
	t.Fatalf("aucune grandeur equipment_%s dans le bloc servi", family)
	return nil
}
