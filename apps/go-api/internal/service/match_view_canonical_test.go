package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// errSentinel : erreur repo générique simulant "match introuvable" (le repo ne
// sait pas servir → routage vers la voie canonique).
var errSentinel = errors.New("repo: no rows")

// --- helpers ---
// ptrFloat est déclaré dans match_history_explorer_options_test.go (même package).

func ptrInt(v int) *int      { return &v }
func ptrBoolMV(v bool) *bool { return &v }

// fakeDetailAdapter implémente games.TitleDataAdapter mais ne sert QUE
// LoadMatchDetail (le reste dégrade en ErrCapabilityNotSupported). Suffit pour
// tester le routage repo-first / adapter-fallback de GetMatchView.
type fakeDetailAdapter struct {
	detail *canonical.MatchDetail
	err    error
}

func (f *fakeDetailAdapter) TitleSlug() string                 { return "halo_5" }
func (f *fakeDetailAdapter) Capabilities() games.CapabilityMap { return games.CapabilityMap{} }
func (f *fakeDetailAdapter) LoadMatchDetail(_ context.Context, _ string) (*canonical.MatchDetail, error) {
	return f.detail, f.err
}
func (f *fakeDetailAdapter) LoadPlayerStats(_ context.Context, _ string, _ canonical.StatsScope) (*canonical.PlayerStats, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadCareerSnapshot(_ context.Context, _ string, _ canonical.CareerOptions) (*canonical.CareerSnapshot, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadMatchSummaries(_ context.Context, _ []string) ([]canonical.MatchSummary, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadEncounters(_ context.Context, _ string) ([]canonical.EncounterRow, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadLUSRHistory(_ context.Context, _ string) ([]canonical.LUSRCheckpoint, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadTopMatches(_ context.Context, _ string) ([]canonical.CareerTopMatch, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadTargetRecentMatches(_ context.Context, _ string, _ int) ([]canonical.RecentMatchRow, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadParticipantStats(_ context.Context, _ string, _ []string) (*canonical.PlayerMatchSetStats, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadPlayerIntersection(_ context.Context, _, _ string) (*canonical.PlayerIntersection, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadTimeseries(_ context.Context, _ string, _ canonical.TimeseriesQuery) (*canonical.MetricSeries, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadMatchScoreboard(_ context.Context, _ string) ([]canonical.MatchParticipant, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadHighlightEvents(_ context.Context, _ string) ([]canonical.HighlightEvent, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadMatchEvents(_ context.Context, _ string, _ canonical.MatchEventOptions) (*canonical.MatchEventTimeline, error) {
	return nil, games.ErrCapabilityNotSupported
}
func (f *fakeDetailAdapter) LoadFriendsXUIDs(_ context.Context, _ string) ([]string, error) {
	return nil, games.ErrCapabilityNotSupported
}

// sampleH5Detail construit un MatchDetail h5-like (gamertag-keyé, xuid vide).
func sampleH5Detail() *canonical.MatchDetail {
	isRanked := true
	start := time.Date(2026, 6, 20, 18, 0, 0, 0, time.UTC)
	end := start.Add(7 * time.Minute)
	return &canonical.MatchDetail{
		MatchID:      "h5-match-1",
		StartedAtUTC: start,
		EndedAtUTC:   &end,
		Map:          &canonical.AssetReference{Kind: "map", ID: "map-guid-1", DefaultLabel: "Truth", Labels: map[string]string{"fr": "Vérité"}},
		Playlist:     &canonical.AssetReference{Kind: "playlist", ID: "pl-1", DefaultLabel: "Team Arena"},
		GameVariant:  &canonical.AssetReference{Kind: "game_variant", ID: "gv-1", DefaultLabel: "Slayer"},
		IsRanked:     &isRanked,
		MatchType:    canonical.MatchTypeRanked,
		Participants: []canonical.MatchParticipant{
			{
				Identity:      canonical.PlayerIdentity{Gamertag: "JGtm"},
				TeamID:        ptrInt(0),
				RankInMatch:   ptrInt(1),
				Outcome:       canonical.OutcomeWin,
				Score:         ptrInt(2500),
				PersonalScore: ptrInt(2500),
				Kills:         ptrInt(20),
				Deaths:        ptrInt(8),
				Assists:       ptrInt(5),
				HeadshotKills: ptrInt(7),
				KDA:           ptrFloat(13.6),
				DamageDealt:   ptrInt(3200),
				PerfectKills:  ptrInt(2),
			},
			{
				Identity:    canonical.PlayerIdentity{Gamertag: "Rival"},
				TeamID:      ptrInt(1),
				RankInMatch: ptrInt(5),
				Outcome:     canonical.OutcomeLoss,
				Kills:       ptrInt(10),
				Deaths:      ptrInt(18),
				Assists:     ptrInt(2),
				IsBot:       ptrBoolMV(false),
			},
		},
		Teams: []canonical.TeamSnapshot{
			{TeamID: 0, Score: ptrInt(50)},
			{TeamID: 1, Score: ptrInt(42)},
		},
	}
}

// TestBuildMatchViewFromCanonical_HeaderOutcomeScoreMap : un MatchDetail h5 factice
// → header outcome/score/map corrects, scoreboard complet, is_partial=true.
func TestBuildMatchViewFromCanonical_HeaderOutcomeScoreMap(t *testing.T) {
	svc := NewMatchViewService(nil, "")
	ctx := ctxkeys.WithViewerGamertag(context.Background(), "JGtm")
	resp := svc.buildMatchViewFromCanonical(ctx, sampleH5Detail())

	// Header : match id + map (FR prioritaire) + mode (game variant) + ranked.
	if resp.Header.MatchID != "h5-match-1" {
		t.Errorf("MatchID = %q, want h5-match-1", resp.Header.MatchID)
	}
	if resp.Header.MapUI != "Vérité" {
		t.Errorf("MapUI = %q, want 'Vérité' (label FR)", resp.Header.MapUI)
	}
	if resp.Header.MapID != "map-guid-1" {
		t.Errorf("MapID = %q, want map-guid-1", resp.Header.MapID)
	}
	if resp.Header.ModeUI != "Slayer" {
		t.Errorf("ModeUI = %q, want 'Slayer' (game variant)", resp.Header.ModeUI)
	}
	if resp.Header.PlaylistLabel != "Team Arena" {
		t.Errorf("PlaylistLabel = %q, want 'Team Arena'", resp.Header.PlaylistLabel)
	}
	if !resp.Header.IsRanked {
		t.Error("IsRanked = false, want true")
	}
	// Outcome du self (JGtm = Win = code 2).
	if resp.Header.OutcomeCode == nil || *resp.Header.OutcomeCode != domain.OutcomeWin {
		t.Errorf("OutcomeCode = %v, want %d (win)", resp.Header.OutcomeCode, domain.OutcomeWin)
	}
	if resp.Header.OutcomeColorToken != "outcome-win" {
		t.Errorf("OutcomeColorToken = %q, want 'outcome-win'", resp.Header.OutcomeColorToken)
	}
	// ScoreLabel : équipe du self (team 0, score 50) en premier.
	if resp.Header.ScoreLabel != "50 - 42" {
		t.Errorf("ScoreLabel = %q, want '50 - 42'", resp.Header.ScoreLabel)
	}
	// Durée jouable = 7 min = 420s.
	if resp.Header.PlayableDurationSeconds == nil || *resp.Header.PlayableDurationSeconds != 420 {
		t.Errorf("PlayableDurationSeconds = %v, want 420", resp.Header.PlayableDurationSeconds)
	}

	// IsPartial + raisons.
	if !resp.IsPartial {
		t.Error("IsPartial = false, want true")
	}
	if len(resp.PartialReasons) == 0 {
		t.Error("PartialReasons vide, attendu les raisons live")
	}

	// Scoreboard : 2 participants, is_me sur JGtm.
	if len(resp.TeamTab.Scoreboard) != 2 {
		t.Fatalf("scoreboard count = %d, want 2", len(resp.TeamTab.Scoreboard))
	}
	var meFound bool
	for _, row := range resp.TeamTab.Scoreboard {
		if row.Gamertag == "JGtm" {
			meFound = row.IsMe
			if row.Kills == nil || *row.Kills != 20 {
				t.Errorf("JGtm kills = %v, want 20", row.Kills)
			}
			if row.TeamSide == nil || *row.TeamSide != "t0" {
				t.Errorf("JGtm team_side = %v, want t0", row.TeamSide)
			}
		}
	}
	if !meFound {
		t.Error("aucune ligne scoreboard marquée is_me pour le viewer JGtm")
	}

	// Summary self KPIs.
	if resp.SummaryTab.KPIs.Kills == nil || *resp.SummaryTab.KPIs.Kills != 20 {
		t.Errorf("summary kills = %v, want 20", resp.SummaryTab.KPIs.Kills)
	}
	if resp.SummaryTab.PersonalResult.OutcomeColorToken != "outcome-win" {
		t.Errorf("summary outcome token = %q, want 'outcome-win'", resp.SummaryTab.PersonalResult.OutcomeColorToken)
	}

	// Rank dégradé (Skill nil).
	if resp.Rank.RatingType != "none" {
		t.Errorf("Rank.RatingType = %q, want 'none' (Skill nil)", resp.Rank.RatingType)
	}
}

// TestBuildMatchViewFromCanonical_NoViewer_DegradesSelfKeepsScoreboard : sans viewer
// dans le ctx, le self n'est pas identifiable → header/summary outcome dégradent,
// mais le scoreboard reste complet (toutes les lignes).
func TestBuildMatchViewFromCanonical_NoViewer_DegradesSelfKeepsScoreboard(t *testing.T) {
	svc := NewMatchViewService(nil, "")
	resp := svc.buildMatchViewFromCanonical(context.Background(), sampleH5Detail())

	if resp.Header.OutcomeCode != nil {
		t.Errorf("OutcomeCode = %v, want nil (self introuvable)", resp.Header.OutcomeCode)
	}
	if len(resp.TeamTab.Scoreboard) != 2 {
		t.Errorf("scoreboard count = %d, want 2 (complet malgré self absent)", len(resp.TeamTab.Scoreboard))
	}
	for _, row := range resp.TeamTab.Scoreboard {
		if row.IsMe {
			t.Errorf("ligne %q marquée is_me alors qu'aucun viewer", row.Gamertag)
		}
	}
	// ScoreLabel reste calculable depuis les équipes (ordre brut).
	if resp.Header.ScoreLabel == "" {
		t.Error("ScoreLabel vide alors que 2 équipes scorées")
	}
}

// TestBuildMatchViewFromCanonical_NilFields_NoPanic : participants vides, Skill nil,
// Map nil → aucune panique, dégradation en chaînes/pointeurs vides.
func TestBuildMatchViewFromCanonical_NilFields_NoPanic(t *testing.T) {
	svc := NewMatchViewService(nil, "")
	detail := &canonical.MatchDetail{MatchID: "empty"}
	resp := svc.buildMatchViewFromCanonical(context.Background(), detail)

	if resp.Header.MatchID != "empty" {
		t.Errorf("MatchID = %q, want empty", resp.Header.MatchID)
	}
	if resp.Header.MapUI != "" {
		t.Errorf("MapUI = %q, want empty (Map nil)", resp.Header.MapUI)
	}
	if len(resp.TeamTab.Scoreboard) != 0 {
		t.Errorf("scoreboard count = %d, want 0", len(resp.TeamTab.Scoreboard))
	}
	if !resp.IsPartial {
		t.Error("IsPartial = false, want true")
	}
}

// TestBuildMatchViewFromCanonical_NativeCommendations : les commendations natives
// (Halo 5 AXE B) du détail canonique peuplent CitationsTab.NativeCommendations et
// retirent partialReasonCitations. Sans commendation → onglet vide + raison présente.
func TestBuildMatchViewFromCanonical_NativeCommendations(t *testing.T) {
	svc := NewMatchViewService(nil, "")
	ctx := ctxkeys.WithViewerGamertag(context.Background(), "JGtm")

	// Cas 1 : commendations présentes.
	icon := "https://example.test/comm.png"
	detail := sampleH5Detail()
	detail.Commendations = []canonical.Commendation{
		{ID: "comm-uuid-1", Count: 3, Name: "Sharpshooter", IconURL: &icon},
		{ID: "comm-uuid-2", Count: 1}, // pas de définition → Name vide, IconURL nil
	}
	resp := svc.buildMatchViewFromCanonical(ctx, detail)
	got := resp.CitationsTab.NativeCommendations
	if len(got) != 2 {
		t.Fatalf("NativeCommendations = %d, want 2 — %+v", len(got), got)
	}
	if got[0].ID != "comm-uuid-1" || got[0].Count != 3 || got[0].Name != "Sharpshooter" || got[0].IconURL == nil {
		t.Errorf("commendation[0] mal projetée: %+v", got[0])
	}
	if got[1].Name != "" || got[1].IconURL != nil {
		t.Errorf("commendation[1] sans définition doit dégrader (Name='', IconURL=nil): %+v", got[1])
	}
	if containsReason(resp.PartialReasons, partialReasonCitations) {
		t.Errorf("partialReasonCitations ne doit PAS être présent quand des commendations existent: %v", resp.PartialReasons)
	}

	// Cas 2 : aucune commendation → onglet vide + raison citations présente.
	bare := svc.buildMatchViewFromCanonical(ctx, sampleH5Detail())
	if len(bare.CitationsTab.NativeCommendations) != 0 {
		t.Errorf("NativeCommendations doit être vide sans commendations: %+v", bare.CitationsTab.NativeCommendations)
	}
	if !containsReason(bare.PartialReasons, partialReasonCitations) {
		t.Errorf("partialReasonCitations attendu quand aucune commendation: %v", bare.PartialReasons)
	}
}

// TestBuildCanonicalCitationsTab_MasteryFilterAndProgress : parité Infinite sur les
// commendations natives — (a) MASQUAGE des commendations masterisées AVANT le match,
// (b) IsNewlyMastered pour celles franchies PENDANT, (c) anneau de progression sinon.
// Seuils réels : Sniper Rifle tier_targets="5,15,30,55,105" (validés sur les DB h5).
func TestBuildCanonicalCitationsTab_MasteryFilterAndProgress(t *testing.T) {
	const sniperTiers = "5,15,30,55,105"
	comms := []canonical.Commendation{
		// (a) masterisée AVANT (before=119-3=116 >= 105) → DOIT être masquée.
		{ID: "already", Name: "Already", Count: 3, Progress: 119, TierTargets: sniperTiers},
		// (b) masterisée PENDANT (before=100 < 105, progress=119 >= 105) → newly mastered.
		{ID: "newly", Name: "Newly", Count: 19, Progress: 119, TierTargets: sniperTiers},
		// (b') franchie pile (before=104 < 105, progress=105) → newly mastered, count=1.
		{ID: "newly-edge", Name: "NewlyEdge", Count: 1, Progress: 105, TierTargets: sniperTiers},
		// (c) en progression intermédiaire (progress=40, dans [30,55)) → anneau partiel.
		{ID: "progressing", Name: "Progressing", Count: 10, Progress: 40, TierTargets: sniperTiers},
		// (d) sans paliers connus → anneau vide (TierCount=0), jamais masquée.
		{ID: "no-tiers", Name: "NoTiers", Count: 2, Progress: 7, TierTargets: ""},
	}

	tab, unavailable := buildCanonicalCitationsTab(comms)
	if unavailable {
		t.Fatalf("section ne doit pas être indisponible : %d commendations visibles attendues", 4)
	}
	got := tab.NativeCommendations

	// (a) la commendation pré-masterisée est ABSENTE.
	byID := make(map[string]domain.MatchNativeCommendation, len(got))
	for _, c := range got {
		byID[c.ID] = c
	}
	if _, present := byID["already"]; present {
		t.Errorf("commendation masterisée AVANT le match doit être MASQUÉE, présente: %+v", byID["already"])
	}
	// 4 visibles : newly, newly-edge, progressing, no-tiers.
	if len(got) != 4 {
		t.Fatalf("NativeCommendations visibles = %d, want 4 — %+v", len(got), got)
	}

	// (b) franchies pendant → IsNewlyMastered + ProgressPct=100 + TierIndex==TierCount.
	for _, id := range []string{"newly", "newly-edge"} {
		c := byID[id]
		if !c.IsNewlyMastered {
			t.Errorf("%s: IsNewlyMastered attendu true, got %+v", id, c)
		}
		if c.ProgressPct != 100.0 {
			t.Errorf("%s: ProgressPct attendu 100, got %v", id, c.ProgressPct)
		}
		if c.TierCount != 5 || c.TierIndex != 5 {
			t.Errorf("%s: TierIndex/TierCount attendus 5/5, got %d/%d", id, c.TierIndex, c.TierCount)
		}
		if c.NextTierTarget != 0 {
			t.Errorf("%s: NextTierTarget attendu 0 (maîtrisé), got %d", id, c.NextTierTarget)
		}
	}

	// (c) en progression : pas newly, anneau partiel, prochain seuil = 55, tier 3/5.
	p := byID["progressing"]
	if p.IsNewlyMastered {
		t.Errorf("progressing: IsNewlyMastered doit être false, got %+v", p)
	}
	if p.TierCount != 5 || p.TierIndex != 3 {
		t.Errorf("progressing: TierIndex/TierCount attendus 3/5, got %d/%d", p.TierIndex, p.TierCount)
	}
	if p.NextTierTarget != 55 {
		t.Errorf("progressing: NextTierTarget attendu 55, got %d", p.NextTierTarget)
	}
	if p.Cumulative != 40 {
		t.Errorf("progressing: Cumulative attendu 40, got %d", p.Cumulative)
	}
	if !(p.ProgressPct > 0 && p.ProgressPct < 100) {
		t.Errorf("progressing: ProgressPct attendu dans (0,100), got %v", p.ProgressPct)
	}

	// (d) sans paliers : anneau vide, jamais masquée.
	nt := byID["no-tiers"]
	if nt.TierCount != 0 || nt.IsNewlyMastered || nt.ProgressPct != 0 {
		t.Errorf("no-tiers: anneau vide attendu (TierCount=0, !newly, pct=0), got %+v", nt)
	}

	// Tri count DESC : newly(19) avant progressing(10) avant no-tiers(2) avant newly-edge(1).
	if got[0].ID != "newly" {
		t.Errorf("tri count DESC attendu, got[0]=%s", got[0].ID)
	}

	// Toutes pré-masterisées → onglet vide + section signalée indisponible.
	allMastered := []canonical.Commendation{
		{ID: "m1", Count: 1, Progress: 200, TierTargets: sniperTiers},
		{ID: "m2", Count: 2, Progress: 150, TierTargets: sniperTiers},
	}
	emptyTab, unavail := buildCanonicalCitationsTab(allMastered)
	if !unavail || len(emptyTab.NativeCommendations) != 0 {
		t.Errorf("toutes pré-masterisées → onglet vide + indisponible, got unavail=%v len=%d",
			unavail, len(emptyTab.NativeCommendations))
	}
}

func containsReason(reasons []string, want string) bool {
	for _, r := range reasons {
		if r == want {
			return true
		}
	}
	return false
}

// TestGetMatchView_RoutesToCanonical_WhenRepoCannotServe : meta en erreur (repo ne
// sert pas le match) + adapter câblé → la voie canonique prend le relais.
func TestGetMatchView_RoutesToCanonical_WhenRepoCannotServe(t *testing.T) {
	repo := &mockMatchViewRepo{metaErr: errSentinel}
	svc := NewMatchViewService(repo, "").
		WithDataAdapter(&fakeDetailAdapter{detail: sampleH5Detail()}).
		WithViewerGamertag("JGtm")

	resp, err := svc.GetMatchView(context.Background(), "h5-match-1")
	if err != nil {
		t.Fatalf("attendu succès via voie canonique, erreur: %v", err)
	}
	if resp.Header.MatchID != "h5-match-1" {
		t.Errorf("MatchID = %q, want h5-match-1 (voie canonique)", resp.Header.MatchID)
	}
	if !resp.IsPartial {
		t.Error("IsPartial = false, want true (voie canonique)")
	}
}

// TestGetMatchView_NoRegression_WhenAdapterDegrades : meta en erreur + adapter qui
// dégrade (ErrCapabilityNotSupported) → l'erreur repo d'origine est remontée
// (comportement HINF / pré-canonique inchangé).
func TestGetMatchView_NoRegression_WhenAdapterDegrades(t *testing.T) {
	repo := &mockMatchViewRepo{metaErr: errSentinel}
	svc := NewMatchViewService(repo, "").
		WithDataAdapter(&fakeDetailAdapter{err: games.ErrCapabilityNotSupported}).
		WithViewerGamertag("JGtm")

	_, err := svc.GetMatchView(context.Background(), "x")
	if err == nil {
		t.Fatal("attendu l'erreur repo d'origine quand l'adapter dégrade")
	}
}
