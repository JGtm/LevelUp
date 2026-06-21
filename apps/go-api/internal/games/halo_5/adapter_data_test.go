package halo_5

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/classification"
)

// fakeSource implemente h5Source sans reseau (injecte des reponses / erreurs).
type fakeSource struct {
	sr         *H5ServiceRecordResponse
	srErr      error
	ev         *h5MatchEventsResponse
	evErr      error
	matches    *H5MatchesResponse
	matchesErr error
	carnage    *H5CarnageResponse
	carnageErr error
}

func (f *fakeSource) GetServiceRecords(_ context.Context, _, _ string) (*H5ServiceRecordResponse, error) {
	return f.sr, f.srErr
}

func (f *fakeSource) GetPlayerMatches(_ context.Context, _ string, _, _ int) (*H5MatchesResponse, error) {
	return f.matches, f.matchesErr
}

func (f *fakeSource) GetMatchCarnage(_ context.Context, _, _ string) (*H5CarnageResponse, error) {
	return f.carnage, f.carnageErr
}

func (f *fakeSource) GetMatchEvents(_ context.Context, _ string) (*h5MatchEventsResponse, error) {
	return f.ev, f.evErr
}

// srcFactory enveloppe une source fixe en SourceFactory (le token de prod vient du
// ctx ; en test on court-circuite).
func srcFactory(s h5Source) SourceFactory {
	return func(context.Context) (h5Source, error) { return s, nil }
}

func mustServiceRecord(t *testing.T) *H5ServiceRecordResponse {
	t.Helper()
	var sr H5ServiceRecordResponse
	if err := json.Unmarshal([]byte(fixtureServiceRecord), &sr); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	return &sr
}

func TestAdapter_TitleSlugAndCapabilities(t *testing.T) {
	a := NewDataAdapter(srcFactory(&fakeSource{}), nil)
	if a.TitleSlug() != "halo_5" {
		t.Errorf("TitleSlug = %q, want halo_5", a.TitleSlug())
	}
	caps := a.Capabilities()
	// Phase 1a honnete : seul career.progression est cable.
	if !caps.Has(games.CapCareerProgression) {
		t.Errorf("career.progression devrait etre disponible (LoadCareerSnapshot cable)")
	}
	// match.history / citations = not_exposed tant que stub (pas de Has()==true menteur).
	if caps.Has(games.CapMatchHistory) {
		t.Errorf("match.history devrait etre not_exposed (LoadMatchSummaries stub)")
	}
	if caps.Has(games.CapCitationsEngine) {
		t.Errorf("citations.engine devrait etre not_exposed (decision B)")
	}
}

func TestAdapter_NilFactory_CapabilitiesDegraded(t *testing.T) {
	// Pas de source-factory -> rien n'est servable -> toutes les capabilities a not_exposed.
	caps := NewDataAdapter(nil, nil).Capabilities()
	for k, v := range caps {
		if v != games.CapNotExposed {
			t.Errorf("factory nil : capability %q = %q, want not_exposed", k, v)
		}
	}
}

func TestAdapter_LoadPlayerStats_Live(t *testing.T) {
	a := NewDataAdapter(srcFactory(&fakeSource{sr: mustServiceRecord(t)}), nil)
	s, err := a.LoadPlayerStats(context.Background(), "JGtm", canonical.StatsScope{})
	if err != nil {
		t.Fatalf("LoadPlayerStats: %v", err)
	}
	if s.MatchesPlayed != 3 || s.Kills != 20 || s.Deaths != 39 {
		t.Errorf("stats inattendues : %+v", s)
	}
	if s.Identity.XUID != "" || s.Identity.Gamertag != "JGtm" {
		t.Errorf("identite gamertag-keyee attendue : %+v", s.Identity)
	}
}

func TestAdapter_LoadCareerSnapshot_Live(t *testing.T) {
	a := NewDataAdapter(srcFactory(&fakeSource{sr: mustServiceRecord(t)}), nil)
	snap, err := a.LoadCareerSnapshot(context.Background(), "JGtm", canonical.CareerOptions{})
	if err != nil {
		t.Fatalf("LoadCareerSnapshot: %v", err)
	}
	if snap.RankName == nil || *snap.RankName != "Diamant 5" {
		t.Errorf("RankName = %v, want Diamant 5", snap.RankName)
	}
}

func TestAdapter_NilFactory_NotSupported(t *testing.T) {
	a := NewDataAdapter(nil, nil)
	if _, err := a.LoadPlayerStats(context.Background(), "JGtm", canonical.StatsScope{}); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("factory nil -> attendu ErrCapabilityNotSupported, got %v", err)
	}
	if _, err := a.LoadCareerSnapshot(context.Background(), "JGtm", canonical.CareerOptions{}); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("factory nil -> attendu ErrCapabilityNotSupported, got %v", err)
	}
}

func TestAdapter_NotFound_EmptyNotError(t *testing.T) {
	notFound := &HTTPError{StatusCode: http.StatusNotFound, URL: "x", Err: errors.New("absent")}
	a := NewDataAdapter(srcFactory(&fakeSource{srErr: notFound}), nil)
	s, err := a.LoadPlayerStats(context.Background(), "Inconnu", canonical.StatsScope{})
	if err != nil {
		t.Fatalf("404 ne doit pas etre une erreur metier : %v", err)
	}
	if s == nil || s.MatchesPlayed != 0 || s.Identity.Gamertag != "Inconnu" {
		t.Errorf("404 -> stats vides avec identite, got %+v", s)
	}
}

func TestAdapter_TokenExpired401_GracefulEmpty(t *testing.T) {
	// Un 401 (token expire/tourne) sur un endpoint read-only ne doit pas casser la
	// page : degradation gracieuse (vide identite-seule + warn), pas erreur dure.
	unauth := &HTTPError{StatusCode: http.StatusUnauthorized, URL: "x", Err: errors.New("expire")}
	a := NewDataAdapter(srcFactory(&fakeSource{srErr: unauth}), nil)
	s, err := a.LoadPlayerStats(context.Background(), "JGtm", canonical.StatsScope{})
	if err != nil {
		t.Fatalf("401 doit degrader gracieusement, pas erreur dure : %v", err)
	}
	if s == nil || s.MatchesPlayed != 0 {
		t.Errorf("401 -> stats vides, got %+v", s)
	}
}

func TestAdapter_TransportError_Propagated(t *testing.T) {
	a := NewDataAdapter(srcFactory(&fakeSource{srErr: errors.New("boom reseau")}), nil)
	if _, err := a.LoadPlayerStats(context.Background(), "JGtm", canonical.StatsScope{}); err == nil {
		t.Error("erreur transport (non-404/401) doit etre propagee")
	}
}

// TestNewSpartanTokenSource_NoToken : la factory de prod echoue proprement si le
// contexte ne porte pas de SpartanToken (pas de panique).
func TestNewSpartanTokenSource_NoToken(t *testing.T) {
	if _, err := NewSpartanTokenSource(context.Background()); err == nil {
		t.Error("attendu une erreur quand le SpartanToken est absent du contexte")
	}
}

// matchDetailFixtures construit un historique (1 entrée : matchID + mode arena +
// map/playlist/variant + durée) + une carnage (4 joueurs, 2 équipes, team 0
// gagnante) pour un viewer donné.
func matchDetailFixtures() (*H5MatchesResponse, *H5CarnageResponse) {
	matches := &H5MatchesResponse{
		Results: []H5MatchResult{
			{
				Id:                 H5MatchID{MatchId: "match-abc", GameMode: 1}, // 1 = arena
				HopperId:           "hopper-ranked-1",
				MapId:              "map-truth",
				GameBaseVariantId:  "variant-slayer",
				MatchDuration:      "PT9M30S",
				MatchCompletedDate: H5ISODate{ISO8601Date: "2023-05-01T20:09:30Z"},
				IsTeamGame:         true,
			},
			// 2e entrée bruit (ne doit jamais être renvoyée pour match-abc).
			{Id: H5MatchID{MatchId: "match-xyz", GameMode: 1}, MapId: "map-other"},
		},
	}
	carnage := &H5CarnageResponse{
		IsTeamGame: true,
		TeamStats:  []H5CarnageTeam{{TeamId: 0, Score: 50, Rank: 1}, {TeamId: 1, Score: 38, Rank: 2}},
		PlayerStats: []H5CarnagePlayer{
			{Player: H5PlayerRef{Gamertag: "JGtm"}, TeamId: 0, Rank: 2,
				PlayerScore: 1200, TotalKills: 18, TotalDeaths: 9, TotalAssists: 6,
				TotalHeadshots: 7, TotalShotsFired: 200, TotalShotsLanded: 110,
				TotalWeaponDamage: 4321.7, TotalMeleeKills: 2, TotalGrenadeKills: 1,
				TotalPowerWeaponKills: 3, TotalTimePlayed: "PT9M0S", AvgLifeTimeOfPlayer: "PT22.4S"},
			{Player: H5PlayerRef{Gamertag: "Mate"}, TeamId: 0, Rank: 1,
				TotalKills: 20, TotalDeaths: 8, TotalAssists: 3},
			{Player: H5PlayerRef{Gamertag: "Foe1"}, TeamId: 1, Rank: 3,
				TotalKills: 12, TotalDeaths: 14, TotalAssists: 5},
			{Player: H5PlayerRef{Gamertag: "Foe2"}, TeamId: 1, Rank: 4,
				TotalKills: 9, TotalDeaths: 15, TotalAssists: 2},
		},
	}
	return matches, carnage
}

// TestAdapter_LoadMatchDetail_Live vérifie le payload MatchDetail complet à partir
// d'une source fake (historique + carnage) + viewer gamertag dans le contexte.
func TestAdapter_LoadMatchDetail_Live(t *testing.T) {
	matches, carnage := matchDetailFixtures()
	// Classifier : hopper-ranked-1 ∈ set classé → IsRanked &true sur le header.
	ranked := classification.NewSetClassifier([]string{"hopper-ranked-1"}, nil)
	a := NewDataAdapter(srcFactory(&fakeSource{matches: matches, carnage: carnage}), nil).
		WithRankedClassifier(ranked)

	ctx := ctxkeys.WithViewerGamertag(context.Background(), "JGtm")
	d, err := a.LoadMatchDetail(ctx, "match-abc")
	if err != nil {
		t.Fatalf("LoadMatchDetail: %v", err)
	}
	if d == nil {
		t.Fatal("MatchDetail nil")
	}
	if d.MatchID != "match-abc" {
		t.Errorf("MatchID = %q, want match-abc", d.MatchID)
	}
	// Refs header (réutilisent le mapper summary).
	if d.Map == nil || d.Map.ID != "map-truth" {
		t.Errorf("Map = %v, want ref map-truth", d.Map)
	}
	if d.Playlist == nil || d.Playlist.ID != "hopper-ranked-1" {
		t.Errorf("Playlist = %v, want ref hopper-ranked-1", d.Playlist)
	}
	if d.GameVariant == nil || d.GameVariant.ID != "variant-slayer" {
		t.Errorf("GameVariant = %v, want ref variant-slayer", d.GameVariant)
	}
	if d.IsRanked == nil || !*d.IsRanked {
		t.Errorf("IsRanked = %v, want &true (hopper classé)", d.IsRanked)
	}
	if d.MatchType != canonical.MatchTypeRanked {
		t.Errorf("MatchType = %q, want ranked", d.MatchType)
	}
	if d.StartedAtUTC.IsZero() {
		t.Error("StartedAtUTC ne doit pas être zéro (fin − durée)")
	}
	if d.EndedAtUTC == nil {
		t.Error("EndedAtUTC doit être renseigné (début + durée)")
	}
	// Roster : 4 participants, AUCUN sauté (canonique in-memory, pas de PK xuid).
	if len(d.Participants) != 4 {
		t.Fatalf("Participants = %d, want 4", len(d.Participants))
	}
	// Teams : 2 équipes avec scores.
	if len(d.Teams) != 2 {
		t.Fatalf("Teams = %d, want 2", len(d.Teams))
	}
	// Viewer JGtm : K/D/A bruts + outcome WIN (son équipe 0 = Rank 1, même si son
	// Rank individuel = 2).
	var jg *canonical.MatchParticipant
	for i := range d.Participants {
		if d.Participants[i].Identity.Gamertag == "JGtm" {
			jg = &d.Participants[i]
		}
	}
	if jg == nil {
		t.Fatal("participant JGtm introuvable")
	}
	if jg.Kills == nil || *jg.Kills != 18 || jg.Deaths == nil || *jg.Deaths != 9 || jg.Assists == nil || *jg.Assists != 6 {
		t.Errorf("K/D/A JGtm KO: %+v", jg)
	}
	if jg.Outcome != canonical.OutcomeWin {
		t.Errorf("Outcome JGtm = %q, want win (rang d'équipe prime)", jg.Outcome)
	}
	if jg.Identity.XUID != "" {
		t.Errorf("identité h5 gamertag-keyée : XUID doit être vide, got %q", jg.Identity.XUID)
	}
	// Non-fabrication : Accuracy / DamageTaken nil (absents de l'API h5).
	if jg.Accuracy != nil || jg.DamageTaken != nil {
		t.Errorf("h5 ne fabrique pas Accuracy/DamageTaken: acc=%v dt=%v", jg.Accuracy, jg.DamageTaken)
	}
	// DamageDealt présent (dégâts infligés), arrondi int.
	if jg.DamageDealt == nil || *jg.DamageDealt != 4321 {
		t.Errorf("DamageDealt JGtm = %v, want 4321", jg.DamageDealt)
	}
	// Skill nil (CSR pré/post par match = phase ultérieure), Limitations déclarées.
	if d.Skill != nil {
		t.Errorf("Skill doit être nil (phase ultérieure), got %+v", d.Skill)
	}
	if len(d.Limitations) == 0 {
		t.Error("Limitations doivent déclarer les gaps h5 (accuracy/damage_taken/skill)")
	}
}

// TestAdapter_LoadMatchDetail_Degradations couvre les chemins de dégradation
// gracieuse → ErrCapabilityNotSupported (le service Part B retombe sur le repo).
func TestAdapter_LoadMatchDetail_Degradations(t *testing.T) {
	matches, carnage := matchDetailFixtures()

	cases := []struct {
		name    string
		factory SourceFactory
		ctx     context.Context
		matchID string
	}{
		{
			name:    "viewer_gamertag_absent",
			factory: srcFactory(&fakeSource{matches: matches, carnage: carnage}),
			ctx:     context.Background(), // pas de ViewerGamertag
			matchID: "match-abc",
		},
		{
			name:    "matchID_vide",
			factory: srcFactory(&fakeSource{matches: matches, carnage: carnage}),
			ctx:     ctxkeys.WithViewerGamertag(context.Background(), "JGtm"),
			matchID: "   ",
		},
		{
			name:    "match_introuvable_dans_historique",
			factory: srcFactory(&fakeSource{matches: matches, carnage: carnage}),
			ctx:     ctxkeys.WithViewerGamertag(context.Background(), "JGtm"),
			matchID: "match-inconnu",
		},
		{
			name:    "carnage_vide",
			factory: srcFactory(&fakeSource{matches: matches, carnage: &H5CarnageResponse{}}),
			ctx:     ctxkeys.WithViewerGamertag(context.Background(), "JGtm"),
			matchID: "match-abc",
		},
		{
			name:    "source_nil_factory",
			factory: nil,
			ctx:     ctxkeys.WithViewerGamertag(context.Background(), "JGtm"),
			matchID: "match-abc",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := NewDataAdapter(tc.factory, nil)
			d, err := a.LoadMatchDetail(tc.ctx, tc.matchID)
			if d != nil {
				t.Errorf("MatchDetail doit être nil en dégradation, got %+v", d)
			}
			if !errors.Is(err, games.ErrCapabilityNotSupported) {
				t.Errorf("attendu ErrCapabilityNotSupported, got %v", err)
			}
		})
	}
}

// TestAdapter_LoadMatchDetail_CarnageHTTPErrorDegrades : un 404 sur la carnage (le
// match a disparu / token tourné) dégrade gracieusement, pas d'erreur dure.
func TestAdapter_LoadMatchDetail_CarnageHTTPErrorDegrades(t *testing.T) {
	matches, _ := matchDetailFixtures()
	notFound := &HTTPError{StatusCode: http.StatusNotFound, URL: "x", Err: errors.New("absent")}
	a := NewDataAdapter(srcFactory(&fakeSource{matches: matches, carnageErr: notFound}), nil)
	ctx := ctxkeys.WithViewerGamertag(context.Background(), "JGtm")
	d, err := a.LoadMatchDetail(ctx, "match-abc")
	if d != nil {
		t.Errorf("404 carnage → MatchDetail nil, got %+v", d)
	}
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("404 carnage doit dégrader en ErrCapabilityNotSupported, got %v", err)
	}
}
