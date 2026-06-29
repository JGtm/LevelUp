package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// --- mocks ---

type mockCompareRepo struct {
	stats    *domain.NormalizedPlayerStats
	statsErr error
	xuid     string
	xuidErr  error
}

func (m *mockCompareRepo) GetLocalStats(_ context.Context, _, _ string) (*domain.NormalizedPlayerStats, error) {
	return m.stats, m.statsErr
}

func (m *mockCompareRepo) ResolveXUID(_ context.Context, _ string) (string, error) {
	return m.xuid, m.xuidErr
}

type mockStatsProvider struct {
	stats    *domain.NormalizedPlayerStats
	statsErr error
}

func (m *mockStatsProvider) FetchRemoteStats(_ context.Context, _, _ string) (*domain.NormalizedPlayerStats, error) {
	return m.stats, m.statsErr
}

// --- tests ---

func TestCompareService_BothLocal(t *testing.T) {
	statsA := &domain.NormalizedPlayerStats{
		Gamertag: "PlayerA",
		Matches:  100,
		WinRate:  0.60,
		KDR:      1.5,
	}
	statsB := &domain.NormalizedPlayerStats{
		Gamertag: "PlayerB",
		Matches:  80,
		WinRate:  0.50,
		KDR:      1.2,
	}

	callCount := 0
	repo := &mockCompareRepo{
		xuid: "xuid-b",
	}
	// GetLocalStats retourne A pour xuid "xuid-a", B pour xuid "xuid-b"
	repo.stats = statsA

	provider := &mockStatsProvider{statsErr: errors.New("not needed")}

	svc := NewCompareService(&mockCompareRepoAB{a: statsA, b: statsB}, provider, "xuid-a", "hi")
	_ = callCount

	resp, err := svc.GetPage(context.Background(), domain.CompareRequest{TargetGamertag: "PlayerB"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PlayerA.Gamertag != "PlayerA" {
		t.Errorf("expected PlayerA, got %s", resp.PlayerA.Gamertag)
	}
	if len(resp.Metrics) == 0 {
		t.Error("expected metrics to be populated")
	}
}

func TestCompareService_PlayerBNotFound(t *testing.T) {
	statsA := &domain.NormalizedPlayerStats{
		Gamertag: "PlayerA",
		Matches:  50,
	}

	repo := &mockCompareRepoAB{
		a:       statsA,
		bErr:    errors.New("not found"),
		xuidErr: errors.New("not found"),
	}
	provider := &mockStatsProvider{statsErr: errors.New("not found")}

	svc := NewCompareService(repo, provider, "xuid-a", "hi")
	_, err := svc.GetPage(context.Background(), domain.CompareRequest{TargetGamertag: "UnknownPlayer"})
	if err == nil {
		t.Error("expected error when player B not found, got nil")
	}
}

// TestCompareService_loadPlayerB_LiveResolveFillsXUID : un joueur B jamais croisé
// (ResolveXUID local vide) est résolu live → xuidB renseigné (permet l'enrichissement
// rang/CSR), tout en servant ses stats Waypoint.
func TestCompareService_loadPlayerB_LiveResolveFillsXUID(t *testing.T) {
	// xuidErr → ResolveXUID local renvoie "" (B jamais croisé) ; bErr → pas de stats
	// locales pour le xuid résolu live → fallback Waypoint.
	repo := &mockCompareRepoAB{
		a:       &domain.NormalizedPlayerStats{Gamertag: "A"},
		bErr:    errors.New("no local B"),
		xuidErr: errors.New("not found"),
	}
	provider := &mockStatsProvider{stats: &domain.NormalizedPlayerStats{Gamertag: "B"}}
	res := &stubResolver{xuid: "live-b-xuid"}
	svc := NewCompareService(repo, provider, "xuid-a", "hi").WithLiveGamertagResolver(res)

	stats, xuidB, err := svc.loadPlayerB(context.Background(), "NeverCrossedB")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.calls != 1 {
		t.Fatalf("résolveur live attendu 1 appel, got %d", res.calls)
	}
	if xuidB != "live-b-xuid" {
		t.Errorf("xuidB = %q, want live-b-xuid", xuidB)
	}
	if stats == nil || stats.Gamertag != "B" {
		t.Errorf("stats B (Waypoint) attendues, got %+v", stats)
	}
}

// TestCompareService_loadPlayerB_LocalSkipsLiveResolver : si B est résolu
// localement, le résolveur live n'est pas consulté.
func TestCompareService_loadPlayerB_LocalSkipsLiveResolver(t *testing.T) {
	repo := &mockCompareRepoAB{
		a:    &domain.NormalizedPlayerStats{Gamertag: "A"},
		b:    &domain.NormalizedPlayerStats{Gamertag: "B"},
		xuid: "local-b",
	}
	provider := &mockStatsProvider{statsErr: errors.New("not needed")}
	res := &stubResolver{xuid: "should-not-be-used"}
	svc := NewCompareService(repo, provider, "xuid-a", "hi").WithLiveGamertagResolver(res)

	_, xuidB, err := svc.loadPlayerB(context.Background(), "LocalB")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.calls != 0 {
		t.Fatalf("résolveur live ne doit pas être appelé si B local (calls=%d)", res.calls)
	}
	if xuidB != "local-b" {
		t.Errorf("xuidB = %q, want local-b", xuidB)
	}
}

// TestBuildMetrics_AvailabilityRemoteB vérifie que les métriques locale-only
// (ATH, max_killing_spree, etc.) sont marquées indisponibles côté B quand B
// est remote, et que les métriques fournies par Waypoint restent disponibles.
func TestBuildMetrics_AvailabilityRemoteB(t *testing.T) {
	a := domain.NormalizedPlayerStats{
		IsLocal:              true,
		Matches:              100,
		WinRate:              0.6,
		KDA:                  1.4,
		KDR:                  1.25,
		KillsPerGame:         10,
		DeathsPerGame:        8,
		Accuracy:             0.45,
		DamagePerGame:        2500,
		MaxKillingSpree:      12,
		AvgLifeSecs:          30,
		PerfectKillsPerGame:  0.2,
		HeadshotKillsPerGame: 3.0,
		PerfATH:              98,
		LusrATH:              1600,
		CareerRank:           150,
	}
	b := domain.NormalizedPlayerStats{
		IsLocal:       false,
		Matches:       50,
		WinRate:       0.5,
		KDA:           1.0,
		KDR:           1.0,
		KillsPerGame:  9,
		DeathsPerGame: 9,
		Accuracy:      0.4,
		DamagePerGame: 2200,
	}

	rows := buildMetrics(a, b, 225)
	byKey := make(map[string]domain.CompareMetricRow, len(rows))
	for _, r := range rows {
		byKey[r.Metric] = r
	}

	cases := []struct {
		metric    string
		wantAvail bool
	}{
		{"win_rate", true},
		{"kda", true},
		{"accuracy", true},
		{"damage_per_game", true},
		{"matches", true},
		{"max_killing_spree", false},
		{"avg_life_secs", false},
		{"perfect_kills_per_game", false},
		{"headshot_kills_per_game", false},
		{"perf_ath", false},
		{"lusr_ath", false},
		{"career_rank", false},
	}
	for _, c := range cases {
		row, ok := byKey[c.metric]
		if !ok {
			t.Errorf("metric %q absente du résultat", c.metric)
			continue
		}
		if row.ValueBAvailable != c.wantAvail {
			t.Errorf("metric %q : ValueBAvailable=%v, want %v", c.metric, row.ValueBAvailable, c.wantAvail)
		}
		if !row.ValueAAvailable {
			t.Errorf("metric %q : ValueAAvailable=false alors que A est local et renseigné", c.metric)
		}
	}
}

// TestMetricAvailability_CareerRankLiveNonLocal : le rang carrière est disponible
// dès value>0, y compris pour un non-local (rang récupéré en live), tandis que les
// autres métriques ATH restent local-only.
func TestMetricAvailability_CareerRankLiveNonLocal(t *testing.T) {
	if !metricAvailability(compareMetricCareerRank, 152, false, false) {
		t.Error("career_rank value>0 doit être disponible même pour un non-local (live)")
	}
	if metricAvailability(compareMetricCareerRank, 0, true, false) {
		t.Error("career_rank value 0 ne doit pas être disponible (ATH non calculé)")
	}
	if metricAvailability(compareMetricPerfATH, 98, false, false) {
		t.Error("perf_ath non-local ne doit pas être disponible (local-only)")
	}
}

// TestBuildMetrics_CareerRankLiveRemoteB : un joueur B non-local dont le rang a été
// récupéré en live (CareerRank>0) expose la ligne career_rank côté B.
func TestBuildMetrics_CareerRankLiveRemoteB(t *testing.T) {
	a := domain.NormalizedPlayerStats{IsLocal: true, Matches: 100, CareerRank: 200, KillsPerGame: 10, DeathsPerGame: 8}
	b := domain.NormalizedPlayerStats{IsLocal: false, Matches: 50, CareerRank: 152, KillsPerGame: 9, DeathsPerGame: 9}

	rows := buildMetrics(a, b, 225)
	var cr *domain.CompareMetricRow
	for i := range rows {
		if rows[i].Metric == compareMetricCareerRank {
			cr = &rows[i]
		}
	}
	if cr == nil {
		t.Fatal("ligne career_rank absente")
	}
	if !cr.ValueBAvailable {
		t.Error("career_rank de B (live, >0) doit être disponible")
	}
	if !cr.ValueAAvailable {
		t.Error("career_rank de A (ATH local) doit être disponible")
	}
}

// TestFillCareerRankLive couvre le helper qui complète le rang carrière d'un
// joueur B (local ou non-local) via le fetch live identity.
func TestFillCareerRankLive(t *testing.T) {
	svc := (&CompareService{}).WithLiveIdentity(&mockLiveIdentity{identity: &domain.HomeSpartanIdentityRow{RankNumber: 152}})
	remote := &domain.NormalizedPlayerStats{}
	svc.fillCareerRankLive(context.Background(), remote, "xuid-b")
	if remote.CareerRank != 152 {
		t.Errorf("CareerRank = %d, want 152 (live)", remote.CareerRank)
	}

	// Rang déjà présent (ex. ATH local) → non écrasé, pas de fetch.
	id := &mockLiveIdentity{identity: &domain.HomeSpartanIdentityRow{RankNumber: 1}}
	svc2 := (&CompareService{}).WithLiveIdentity(id)
	remote2 := &domain.NormalizedPlayerStats{CareerRank: 99}
	svc2.fillCareerRankLive(context.Background(), remote2, "xuid-b")
	if remote2.CareerRank != 99 || id.called {
		t.Errorf("rang existant ne doit pas être écrasé/refetché (rank=%d, called=%v)", remote2.CareerRank, id.called)
	}

	// Pas de provider → no-op.
	r3 := &domain.NormalizedPlayerStats{}
	(&CompareService{}).fillCareerRankLive(context.Background(), r3, "x")
	if r3.CareerRank != 0 {
		t.Errorf("sans provider, rien ne doit être écrit, got %d", r3.CareerRank)
	}
}

// TestCompareService_FetchCSRSummary : le helper retient le max des CSR courants
// ET all-time (avec libellés tier), et dégrade à vide sans provider/saison.
func TestCompareService_FetchCSRSummary(t *testing.T) {
	csr := CSRProviderFunc(func(_ context.Context, _, _ string) ([]domain.CareerPlaylistCSR, error) {
		return []domain.CareerPlaylistCSR{
			{Current: domain.CareerCSRRank{Value: 1450, Tier: "Diamond", SubTier: 2}, AllTime: domain.CareerCSRRank{Value: 1550, Tier: "Onyx"}},
			{Current: domain.CareerCSRRank{Value: 1600, Tier: "Onyx"}, AllTime: domain.CareerCSRRank{Value: 1700, Tier: "Onyx"}},
			{Current: domain.CareerCSRRank{Value: 1200, Tier: "Platinum", SubTier: 6}, AllTime: domain.CareerCSRRank{Value: 1300, Tier: "Diamond", SubTier: 1}},
		}, nil
	})
	svc := (&CompareService{}).WithCSR(csr, "CsrSeason13-1")
	sum := svc.fetchCSRSummary(context.Background(), "xuid")
	if sum.currentValue != 1600 || sum.currentLabel != "Onyx" {
		t.Errorf("current = (%v, %q), want (1600, Onyx)", sum.currentValue, sum.currentLabel)
	}
	if sum.allTimeValue != 1700 || sum.allTimeLabel != "Onyx" {
		t.Errorf("all-time = (%v, %q), want (1700, Onyx)", sum.allTimeValue, sum.allTimeLabel)
	}
	// Récupéré mais AUCUN classement (slice vide) → "Non classé" (≠ non récupéré).
	empty := CSRProviderFunc(func(_ context.Context, _, _ string) ([]domain.CareerPlaylistCSR, error) {
		return nil, nil
	})
	if got := (&CompareService{}).WithCSR(empty, "S13").fetchCSRSummary(context.Background(), "xuid"); got.currentLabel != csrUnrankedLabel || got.currentValue != 0 {
		t.Errorf("récupéré sans classement → 'Non classé', got (%v, %q)", got.currentValue, got.currentLabel)
	}
	// Non récupéré (pas de saison / pas de provider) → label VIDE (= N/A côté front).
	if got := (&CompareService{}).WithCSR(csr, "").fetchCSRSummary(context.Background(), "xuid"); got.currentLabel != "" {
		t.Errorf("sans saison → non récupéré (label vide), got %+v", got)
	}
	if got := (&CompareService{}).fetchCSRSummary(context.Background(), "xuid"); got.currentLabel != "" {
		t.Errorf("sans provider → non récupéré (label vide), got %+v", got)
	}
}

// TestCSRRankLabel : libellé FR tier + sous-palier romain ; Onyx sans sous-palier.
func TestCSRRankLabel(t *testing.T) {
	cases := []struct {
		rank domain.CareerCSRRank
		want string
	}{
		{domain.CareerCSRRank{Tier: "Gold", SubTier: 3}, "Or III"},
		{domain.CareerCSRRank{Tier: "Platinum", SubTier: 4}, "Platine IV"},
		{domain.CareerCSRRank{Tier: "Onyx", SubTier: 0}, "Onyx"},
		{domain.CareerCSRRank{Tier: ""}, ""},
	}
	for _, c := range cases {
		if got := csrRankLabel(c.rank); got != c.want {
			t.Errorf("csrRankLabel(%+v) = %q, want %q", c.rank, got, c.want)
		}
	}
}

// TestBuildMetrics_CSRRow : tri-état CSR porté par le libellé — classé (tier),
// "Non classé" (récupéré, value 0) et N/A (label vide = non récupéré).
func TestBuildMetrics_CSRRow(t *testing.T) {
	byKey := func(rows []domain.CompareMetricRow) map[string]domain.CompareMetricRow {
		m := make(map[string]domain.CompareMetricRow, len(rows))
		for _, r := range rows {
			m[r.Metric] = r
		}
		return m
	}

	// A et B classés → ligne visible, display = tier, vainqueur par valeur.
	a := domain.NormalizedPlayerStats{IsLocal: true, Matches: 100, HighestCSR: 1600, HighestCSRLabel: "Onyx", KillsPerGame: 10, DeathsPerGame: 8}
	b := domain.NormalizedPlayerStats{IsLocal: false, Matches: 50, HighestCSR: 1450, HighestCSRLabel: "Diamant II", KillsPerGame: 9, DeathsPerGame: 9}
	csr := byKey(buildMetrics(a, b, 225))[compareMetricCSR]
	if csr.Metric != compareMetricCSR {
		t.Fatal("ligne csr absente alors que A et B sont classés")
	}
	if !csr.ValueAAvailable || !csr.ValueBAvailable {
		t.Error("csr A et B (classés) doivent être disponibles")
	}
	if csr.DisplayA != "Onyx" || csr.DisplayB != "Diamant II" {
		t.Errorf("display csr = (%q,%q), want (Onyx, Diamant II)", csr.DisplayA, csr.DisplayB)
	}
	if csr.Winner != "a" {
		t.Errorf("csr winner = %q, want a (1600>1450)", csr.Winner)
	}

	// B "Non classé" (récupéré, value 0, label set) → disponible et affiché (pas N/A).
	bUnranked := domain.NormalizedPlayerStats{IsLocal: false, Matches: 50, HighestCSR: 0, HighestCSRLabel: csrUnrankedLabel}
	r := byKey(buildMetrics(a, bUnranked, 225))[compareMetricCSR]
	if !r.ValueBAvailable || r.DisplayB != csrUnrankedLabel {
		t.Errorf("csr B 'Non classé' doit être disponible et affiché, got avail=%v disp=%q", r.ValueBAvailable, r.DisplayB)
	}

	// B non récupéré (label vide) → N/A (indisponible).
	bNA := domain.NormalizedPlayerStats{IsLocal: false, Matches: 50, HighestCSR: 0, HighestCSRLabel: ""}
	if rr := byKey(buildMetrics(a, bNA, 225))[compareMetricCSR]; rr.ValueBAvailable {
		t.Error("csr B sans libellé (non récupéré) doit être N/A")
	}
}

// TestBuildMetrics_OCDRAlignedWithCombatYield vérifie que rendement/résistance du
// Face à face utilisent exactement les formules OC/DR canoniques (parité KPI bar
// home) et le bon sens de vainqueur (plus haut = mieux).
func TestBuildMetrics_OCDRAlignedWithCombatYield(t *testing.T) {
	a := domain.NormalizedPlayerStats{
		IsLocal: true, Matches: 100,
		KillsPerGame: 12, AssistsPerGame: 3, DeathsPerGame: 6,
		DamagePerGame: 2400, DamageTakenPerGame: 2100,
	}
	b := domain.NormalizedPlayerStats{
		IsLocal: true, Matches: 80,
		KillsPerGame: 8, AssistsPerGame: 2, DeathsPerGame: 9,
		DamagePerGame: 2600, DamageTakenPerGame: 1800,
	}

	rows := buildMetrics(a, b, 225)
	byKey := make(map[string]domain.CompareMetricRow, len(rows))
	for _, r := range rows {
		byKey[r.Metric] = r
	}

	wantA := analysis.ComputeCombatYieldFloat(12, 3, 2400, 2100, 6, 225)
	wantB := analysis.ComputeCombatYieldFloat(8, 2, 2600, 1800, 9, 225)

	rend, ok := byKey["rendement"]
	if !ok {
		t.Fatal("métrique rendement absente")
	}
	if rend.ValueA != wantA.OffensiveConversion || rend.ValueB != wantB.OffensiveConversion {
		t.Errorf("rendement = (%v,%v), want OC canonique (%v,%v)",
			rend.ValueA, rend.ValueB, wantA.OffensiveConversion, wantB.OffensiveConversion)
	}
	if rend.LessIsBetter {
		t.Error("rendement (OC) doit être plus-haut-=-mieux (LessIsBetter=false)")
	}
	if wantA.OffensiveConversion > wantB.OffensiveConversion && rend.Winner != "a" {
		t.Errorf("rendement winner = %q, want a (OC A > OC B)", rend.Winner)
	}

	res, ok := byKey["resistance"]
	if !ok {
		t.Fatal("métrique resistance absente")
	}
	if res.ValueA != wantA.DefensiveResistance || res.ValueB != wantB.DefensiveResistance {
		t.Errorf("resistance = (%v,%v), want DR canonique (%v,%v)",
			res.ValueA, res.ValueB, wantA.DefensiveResistance, wantB.DefensiveResistance)
	}
	if res.LessIsBetter {
		t.Error("resistance (DR) doit être plus-haut-=-mieux (LessIsBetter=false)")
	}
}

// TestBuildMetrics_AvailabilityATHZero vérifie que pour les métriques ATH,
// une valeur 0 sur un joueur local est considérée non disponible (ATH non calculé).
func TestBuildMetrics_AvailabilityATHZero(t *testing.T) {
	a := domain.NormalizedPlayerStats{
		IsLocal: true, Matches: 100, WinRate: 0.6, KillsPerGame: 10, DeathsPerGame: 8,
		PerfATH: 0, LusrATH: 0, CareerRank: 0,
	}
	b := domain.NormalizedPlayerStats{
		IsLocal: true, Matches: 80, WinRate: 0.5, KillsPerGame: 9, DeathsPerGame: 9,
		PerfATH: 50, LusrATH: 1200, CareerRank: 100,
	}

	rows := buildMetrics(a, b, 225)
	byKey := make(map[string]domain.CompareMetricRow, len(rows))
	for _, r := range rows {
		byKey[r.Metric] = r
	}

	for _, metric := range []string{"perf_ath", "lusr_ath", "career_rank"} {
		row, ok := byKey[metric]
		if !ok {
			t.Errorf("metric %q manquante (devrait rester car B a une valeur)", metric)
			continue
		}
		if row.ValueAAvailable {
			t.Errorf("metric %q : ValueAAvailable=true alors que A a 0 (ATH non calculé)", metric)
		}
		if !row.ValueBAvailable {
			t.Errorf("metric %q : ValueBAvailable=false alors que B a une valeur >0", metric)
		}
		if row.Winner != "" {
			t.Errorf("metric %q : Winner=%q, attendu chaîne vide (donnée partielle)", metric, row.Winner)
		}
	}
}

// TestBuildMetrics_IsLocalSample vérifie que les métriques locale-only
// deviennent disponibles côté B quand B est remote mais enrichi par un
// échantillon de matchs croisés (IsLocalSample=true).
func TestBuildMetrics_IsLocalSample(t *testing.T) {
	a := domain.NormalizedPlayerStats{
		IsLocal: true, Matches: 100, WinRate: 0.6, KDA: 1.4, KDR: 1.25,
		KillsPerGame: 10, DeathsPerGame: 8, Accuracy: 0.45, DamagePerGame: 2500,
		MaxKillingSpree: 12, AvgLifeSecs: 30, PerfectKillsPerGame: 0.2, HeadshotKillsPerGame: 3.0,
		PerfATH: 98, LusrATH: 1600, CareerRank: 150,
	}
	// B remote (IsLocal=false) mais enrichi par échantillon croisé.
	b := domain.NormalizedPlayerStats{
		IsLocal: false, IsLocalSample: true,
		Matches: 5, WinRate: 0.5, KDA: 1.0, KDR: 1.0,
		KillsPerGame: 9, DeathsPerGame: 9, Accuracy: 0.4, DamagePerGame: 2200,
		MaxKillingSpree: 8, AvgLifeSecs: 25, PerfectKillsPerGame: 0.1, HeadshotKillsPerGame: 2.5,
	}

	rows := buildMetrics(a, b, 225)
	byKey := make(map[string]domain.CompareMetricRow, len(rows))
	for _, r := range rows {
		byKey[r.Metric] = r
	}

	// Les 4 métriques locale-only doivent être marquées disponibles côté B.
	for _, metric := range []string{"max_killing_spree", "avg_life_secs", "perfect_kills_per_game", "headshot_kills_per_game"} {
		row, ok := byKey[metric]
		if !ok {
			t.Errorf("metric %q absente", metric)
			continue
		}
		if !row.ValueBAvailable {
			t.Errorf("metric %q : ValueBAvailable=false, attendu true (IsLocalSample)", metric)
		}
		if row.Winner == "" {
			t.Errorf("metric %q : Winner vide, attendu calculé (les deux côtés dispo)", metric)
		}
	}

	// L'ATH reste indisponible côté B même avec IsLocalSample.
	for _, metric := range []string{"perf_ath", "lusr_ath", "career_rank"} {
		row, ok := byKey[metric]
		if !ok {
			continue
		}
		if row.ValueBAvailable {
			t.Errorf("metric %q : ValueBAvailable=true, attendu false (ATH non dérivable d'un échantillon)", metric)
		}
	}
}

// mockCompareRepoAB — retourne stats différentes selon le xuid demandé.
type mockCompareRepoAB struct {
	a       *domain.NormalizedPlayerStats
	b       *domain.NormalizedPlayerStats
	bErr    error
	xuid    string
	xuidErr error
	mu      sync.Mutex
	calls   int
}

func (m *mockCompareRepoAB) GetLocalStats(_ context.Context, xuid, _ string) (*domain.NormalizedPlayerStats, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	if xuid == "xuid-b" || (m.a != nil && xuid != "xuid-a") {
		if m.bErr != nil {
			return nil, m.bErr
		}
		return m.b, nil
	}
	return m.a, nil
}

func (m *mockCompareRepoAB) ResolveXUID(_ context.Context, _ string) (string, error) {
	if m.xuidErr != nil {
		return "", m.xuidErr
	}
	if m.xuid != "" {
		return m.xuid, nil
	}
	return "xuid-b", nil
}

func (m *mockCompareRepoAB) GetPlayerATH(_ context.Context) (*domain.PlayerATH, error) {
	return &domain.PlayerATH{}, nil
}

func (m *mockCompareRepoAB) GetPlayerATHFor(_ context.Context, _, _ string) (*domain.PlayerATH, error) {
	return &domain.PlayerATH{}, nil
}

func (m *mockCompareRepoAB) GetEncounterStats(_ context.Context, _, _ string) (*domain.CompareEncounterStats, error) {
	return nil, nil
}

func (m *mockCompareRepoAB) GetCrossMatchSample(_ context.Context, _, _ string) (*domain.CrossMatchSample, error) {
	return nil, nil
}

// ─── F5 : Test de latence Compare P95 < 5s ───────────────────────────────────

// slowProvider simule une latence Waypoint configurable.
type slowProvider struct {
	delay    time.Duration
	stats    *domain.NormalizedPlayerStats
	statsErr error
}

func (s *slowProvider) FetchRemoteStats(_ context.Context, _, _ string) (*domain.NormalizedPlayerStats, error) {
	time.Sleep(s.delay)
	return s.stats, s.statsErr
}

// TestCompareService_Latency_P95 vérifie que GetPage s'exécute en < 5s
// même quand le provider Waypoint répond après un délai simulé.
// Le test est répété N fois pour estimer le P95.
func TestCompareService_Latency_P95(t *testing.T) {
	const N = 20
	const maxP95 = 5 * time.Second
	const simulatedDelay = 50 * time.Millisecond // latence mock — pas de vrai réseau

	statsA := &domain.NormalizedPlayerStats{Gamertag: "PlayerA", Matches: 100, WinRate: 0.6}
	statsB := &domain.NormalizedPlayerStats{Gamertag: "PlayerB", Matches: 80, WinRate: 0.5}

	repo := &mockCompareRepoAB{a: statsA, xuidErr: errors.New("not found")}
	provider := &slowProvider{delay: simulatedDelay, stats: statsB}
	svc := NewCompareService(repo, provider, "xuid-a", "halo_infinite")

	durations := make([]time.Duration, N)
	for i := range N {
		start := time.Now()
		_, err := svc.GetPage(context.Background(), domain.CompareRequest{TargetGamertag: "PlayerB"})
		durations[i] = time.Since(start)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
	}

	// Calcul P95 : trier et prendre le 95e percentile.
	sorted := make([]time.Duration, N)
	copy(sorted, durations)
	sortDurations(sorted)
	p95idx := int(float64(N)*0.95) - 1
	if p95idx < 0 {
		p95idx = 0
	}
	p95 := sorted[p95idx]

	if p95 > maxP95 {
		t.Errorf("P95 latence Compare = %v, dépasse le seuil de %v", p95, maxP95)
	}
}

func sortDurations(d []time.Duration) {
	for i := 1; i < len(d); i++ {
		for j := i; j > 0 && d[j] < d[j-1]; j-- {
			d[j], d[j-1] = d[j-1], d[j]
		}
	}
}

// TestLogBestEffortErr_LevelByErrorType vérifie que logBestEffortErr remonte
// `sql.ErrNoRows` en Debug (cas attendu : pas de data) mais toute autre erreur
// en Warn (anomalie). Anti-régression du bug 2026-05-26 où un Catalog Error
// SharedReader vivait sous Debug silencieux.
func TestLogBestEffortErr_LevelByErrorType(t *testing.T) {
	t.Parallel()

	var buf threadSafeBuffer
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	cases := []struct {
		name      string
		err       error
		wantLevel string
	}{
		{"sql.ErrNoRows → DEBUG", sql.ErrNoRows, "DEBUG"},
		{"wrapped sql.ErrNoRows → DEBUG", fmt.Errorf("query: %w", sql.ErrNoRows), "DEBUG"},
		{"generic err → WARN", errors.New("connection refused"), "WARN"},
		{"Catalog Error → WARN", errors.New("Catalog Error: Table does not exist"), "WARN"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()
			logBestEffortErr(context.Background(), "test", tc.err, "xuid", "Z")
			out := buf.String()
			if !strings.Contains(out, `"level":"`+tc.wantLevel+`"`) {
				t.Errorf("want level=%q in log, got: %s", tc.wantLevel, out)
			}
		})
	}
}

type threadSafeBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (b *threadSafeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *threadSafeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

func (b *threadSafeBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = b.buf[:0]
}
