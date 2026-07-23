// Package api — post_sync_deltas_test.go : tests des helpers de delta-detection
// et de l'émetteur EmitPostSyncDeltas via un emitter recording in-memory.
package wire

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/notifications"
)

// recordingEmitter capture les emit pour assertions in-memory.
type recordingEmitter struct {
	emitted []notifications.EmitInput
	failOn  notifications.Category
}

func (r *recordingEmitter) Emit(_ context.Context, in notifications.EmitInput) error {
	if r.failOn != "" && in.Category == r.failOn {
		return errInjected
	}
	r.emitted = append(r.emitted, in)
	return nil
}

// EmitCoalesced : le fake ne coalesce pas (délègue à Emit en ignorant window).
func (r *recordingEmitter) EmitCoalesced(ctx context.Context, in notifications.EmitInput, _ time.Duration) error {
	return r.Emit(ctx, in)
}

var errInjected = error_injected{}

type error_injected struct{}

func (error_injected) Error() string { return "injected" }

// ─── thresholdCrossed ────────────────────────────────────────────────────

func TestThresholdCrossed_Ascending(t *testing.T) {
	cases := []struct {
		name          string
		before, after float64
		step          float64
		wantCrossed   bool
		wantLevel     float64
	}{
		{"crosses 1.0 from 0.99", 0.99, 1.04, 0.05, true, 1.05}, // 0.99/0.05=19, 1.04/0.05=20 → bucket up → level=20*0.05=1.0; en réalité notre helper renvoie afterBucket * step
		{"no crossing, same bucket", 1.01, 1.04, 0.05, false, 0},
		{"descending ignored", 1.10, 0.99, 0.05, false, 0},
		{"equal ignored", 1.00, 1.00, 0.05, false, 0},
		{"large jump multiple steps", 0.50, 1.10, 0.05, true, 0},
		{"step=0 returns false", 1.0, 2.0, 0, false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			crossed, _ := thresholdCrossed(tc.before, tc.after, tc.step)
			if crossed != tc.wantCrossed {
				t.Errorf("crossed: got %v, want %v", crossed, tc.wantCrossed)
			}
		})
	}
}

func TestThresholdCrossed_LevelAccuracy(t *testing.T) {
	// 0.99 → bucket 19 ; 1.04 → bucket 20 ; level retourné = 20*0.05 = 1.00
	crossed, level := thresholdCrossed(0.99, 1.04, 0.05)
	if !crossed {
		t.Fatal("expected crossed=true")
	}
	if level < 0.99 || level > 1.01 {
		t.Errorf("level expected ~1.00, got %v", level)
	}
}

// ─── EmitPostSyncDeltas ─────────────────────────────────────────────────

func TestEmitPostSyncDeltas_NilGuards(t *testing.T) {
	em := &recordingEmitter{}
	// nil emitter
	EmitPostSyncDeltas(context.Background(), nil, "p1", &PlayerSnapshot{}, &PlayerSnapshot{}, nil, PostSyncDeltaOptions{})
	// nil before
	EmitPostSyncDeltas(context.Background(), em, "p1", nil, &PlayerSnapshot{}, nil, PostSyncDeltaOptions{})
	// nil after
	EmitPostSyncDeltas(context.Background(), em, "p1", &PlayerSnapshot{}, nil, nil, PostSyncDeltaOptions{})
	if len(em.emitted) != 0 {
		t.Errorf("expected no emits with nil args, got %d", len(em.emitted))
	}
}

func TestEmitPostSyncDeltas_NoChange_NoEmit(t *testing.T) {
	em := &recordingEmitter{}
	snap := &PlayerSnapshot{
		CurrentRank:        10,
		PersonalAwardCount: 5,
		CitationsCount:     3,
		KDRatio:            1.20,
		Winrate:            0.55,
	}
	EmitPostSyncDeltas(context.Background(), em, "p1", snap, snap, nil, PostSyncDeltaOptions{})
	if len(em.emitted) != 0 {
		t.Errorf("expected 0 emits when snapshots equal, got %d", len(em.emitted))
	}
}

func TestEmitPostSyncDeltas_CareerRank(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{CurrentRank: 5, CurrentRankName: "Hero"}
	after := &PlayerSnapshot{CurrentRank: 6, CurrentRankName: "Onyx"}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil, PostSyncDeltaOptions{})
	if !hasCategory(em.emitted, notifications.CategoryCareerRank) {
		t.Error("expected career_rank emit when rank up")
	}
	// season_pass_level est déprécié depuis 2026-05-16 : ne doit plus être émis.
	if hasCategory(em.emitted, notifications.CategorySeasonPassLevel) {
		t.Error("season_pass_level must not be emitted anymore (deprecated 2026-05-16)")
	}
}

func TestEmitPostSyncDeltas_ObjectiveCompleted_AggregatedDelta(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{PersonalAwardCount: 10}
	after := &PlayerSnapshot{PersonalAwardCount: 13}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil, PostSyncDeltaOptions{})
	// B1/DP2 : SEUL objective_completed est émis ; objective_assigned supprimé.
	if !hasCategory(em.emitted, notifications.CategoryObjectiveCompleted) {
		t.Error("expected objective_completed")
	}
	if hasCategory(em.emitted, notifications.CategoryObjectiveAssigned) {
		t.Error("objective_assigned ne doit plus être émis (DP2)")
	}
}

// B3 (garde-rail) : aucun scénario post-sync ne doit émettre objective_assigned.
// Balaye un delta large sur tous les compteurs — si quelqu'un rebranche la
// catégorie sur un champ snapshot, ce test échoue.
func TestPostSyncNeverEmitsObjectiveAssigned(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{
		CurrentRank: 5, PersonalAwardCount: 10, CitationsCount: 3,
		ChallengePathsCount: 2, ChallengeCompletedCount: 1,
		CitationTotalEarnedTiers: 3, CitationMasteryCount: 1,
		SkillTierByPlaylist: map[string]string{"ranked-arena": "csr|Diamond|3"},
		KDRatio:             1.0, Winrate: 0.5,
	}
	after := &PlayerSnapshot{
		CurrentRank: 6, PersonalAwardCount: 13, CitationsCount: 5,
		ChallengePathsCount: 4, ChallengeCompletedCount: 3,
		BattlepassCompletedTracks: 1,
		CitationTotalEarnedTiers:  5, CitationMasteryCount: 2,
		SkillTierByPlaylist: map[string]string{"ranked-arena": "csr|Onyx|0"},
		KDRatio:             1.2, Winrate: 0.6,
	}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil, PostSyncDeltaOptions{})
	if hasCategory(em.emitted, notifications.CategoryObjectiveAssigned) {
		t.Error("garde-rail B3 : objective_assigned ré-émise (DP2 violé)")
	}
}

func TestEmitPostSyncDeltas_ChallengeCompleted_AndAdded(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{ChallengeCompletedCount: 1, ChallengePathsCount: 5}
	after := &PlayerSnapshot{ChallengeCompletedCount: 4, ChallengePathsCount: 7}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil, PostSyncDeltaOptions{})
	if !hasCategory(em.emitted, notifications.CategoryChallengeCompleted) {
		t.Error("expected challenge_completed (now wired on ChallengeCompletedCount)")
	}
	if !hasCategory(em.emitted, notifications.CategoryChallengeAdded) {
		t.Error("expected challenge_added")
	}
}

// challenge_completed n'est plus émis quand CitationsCount augmente seul
// (recâblage 2026-05-16 : challenge_completed = challenge_snapshots.status).
func TestEmitPostSyncDeltas_CitationsCountAloneDoesNotEmitChallenge(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{CitationsCount: 1}
	after := &PlayerSnapshot{CitationsCount: 8}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil, PostSyncDeltaOptions{})
	if hasCategory(em.emitted, notifications.CategoryChallengeCompleted) {
		t.Error("CitationsCount diff must no longer emit challenge_completed")
	}
}

// citation_tier / citation_mastery sur deltas CitationTotalEarnedTiers et
// CitationMasteryCount.
func TestEmitPostSyncDeltas_CitationTierAndMastery(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{CitationTotalEarnedTiers: 10, CitationMasteryCount: 2}
	after := &PlayerSnapshot{CitationTotalEarnedTiers: 13, CitationMasteryCount: 4}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil, PostSyncDeltaOptions{})
	if !hasCategory(em.emitted, notifications.CategoryCitationTier) {
		t.Error("expected citation_tier")
	}
	if !hasCategory(em.emitted, notifications.CategoryCitationMastery) {
		t.Error("expected citation_mastery")
	}
}

// battlepass_completed sur delta BattlepassCompletedTracks.
func TestEmitPostSyncDeltas_BattlepassCompleted(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{BattlepassCompletedTracks: 1}
	after := &PlayerSnapshot{BattlepassCompletedTracks: 2}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil, PostSyncDeltaOptions{})
	if !hasCategory(em.emitted, notifications.CategoryBattlepassCompleted) {
		t.Error("expected battlepass_completed")
	}
}

// skill_tier : 1 emit par playlist dont la signature change ; aucun emit si
// la signature est identique.
func TestEmitPostSyncDeltas_SkillTier(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{
		SkillTierByPlaylist: map[string]string{
			"ranked-arena":   "csr|Onyx|0",
			"ranked-doubles": "csr|Diamond|3",
		},
	}
	after := &PlayerSnapshot{
		SkillTierByPlaylist: map[string]string{
			"ranked-arena":   "csr|Onyx|0",      // identique → pas d'emit
			"ranked-doubles": "csr|Onyx|1",      // changé → emit
			"social-slayer":  "lusr|Onyx Pro|0", // nouveau → emit
		},
	}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil, PostSyncDeltaOptions{})
	count := countCategory(em.emitted, notifications.CategorySkillTier)
	if count != 2 {
		t.Errorf("expected 2 skill_tier emits (1 promoted + 1 new playlist), got %d", count)
	}
}

// ─── B9/B10/DP4 : skill_tier montées uniquement + dédup 24 h ─────────────────

func skillTierEmits(t *testing.T, before, after map[string]string, opts PostSyncDeltaOptions) int {
	t.Helper()
	em := &recordingEmitter{}
	b := &PlayerSnapshot{SkillTierByPlaylist: before}
	a := &PlayerSnapshot{SkillTierByPlaylist: after}
	EmitPostSyncDeltas(context.Background(), em, "p1", b, a, nil, opts)
	return countCategory(em.emitted, notifications.CategorySkillTier)
}

func TestEmitPostSyncDeltas_SkillTier_DemotionSilent(t *testing.T) {
	// Gold V → Gold IV : démotion entre rangs connus → 0.
	n := skillTierEmits(t,
		map[string]string{"ranked": "csr|Gold|5"},
		map[string]string{"ranked": "csr|Gold|4"},
		PostSyncDeltaOptions{})
	if n != 0 {
		t.Errorf("démotion Gold V→IV → 0 émission, got %d", n)
	}
}

func TestEmitPostSyncDeltas_SkillTier_PromotionEmits(t *testing.T) {
	// Gold VI → Platinum I : montée → 1.
	n := skillTierEmits(t,
		map[string]string{"ranked": "csr|Gold|6"},
		map[string]string{"ranked": "csr|Platinum|1"},
		PostSyncDeltaOptions{})
	if n != 1 {
		t.Errorf("montée Gold→Platinum → 1 émission, got %d", n)
	}
}

func TestEmitPostSyncDeltas_SkillTier_UnknownTierFailOpen(t *testing.T) {
	// Tier inconnu « Mythril » → fail-open, émet sur changement.
	n := skillTierEmits(t,
		map[string]string{"ranked": "csr|Gold|6"},
		map[string]string{"ranked": "csr|Mythril|1"},
		PostSyncDeltaOptions{})
	if n != 1 {
		t.Errorf("tier inconnu → fail-open émet sur changement, got %d", n)
	}
}

func TestEmitPostSyncDeltas_SkillTier_NewPlaylistEmits(t *testing.T) {
	// before non froid ; nouvelle playlist apparait → émet.
	n := skillTierEmits(t,
		map[string]string{"ranked-arena": "csr|Gold|3"},
		map[string]string{"ranked-arena": "csr|Gold|3", "new-playlist": "csr|Silver|2"},
		PostSyncDeltaOptions{})
	if n != 1 {
		t.Errorf("placement nouvelle playlist → 1 émission, got %d", n)
	}
}

func TestEmitPostSyncDeltas_SkillTier_DedupWithin24h(t *testing.T) {
	now := time.Now().UTC()
	// Notif récente pour ranked / csr|Gold|5 émise il y a 1 h.
	params, _ := json.Marshal(map[string]any{
		"playlist_group": "ranked", "rating_type": "csr", "tier": "Gold", "sub_tier": 5,
	})
	recent := []notifications.Notification{{
		Category: notifications.CategorySkillTier, Params: params, CreatedAt: now.Add(-time.Hour),
	}}
	n := skillTierEmits(t,
		map[string]string{"ranked": "csr|Gold|4"},
		map[string]string{"ranked": "csr|Gold|5"},
		PostSyncDeltaOptions{RecentSkillTiers: recent, Now: now})
	if n != 0 {
		t.Errorf("montée déjà notifiée < 24 h → dédupée, got %d", n)
	}
}

// Séquence de flapping IV→V→IV→V→IV→V en < 24 h → 1 seule émission :
// la 1re montée émet, les démotions sont silencieuses, les montées suivantes
// sont dédupées.
func TestEmitPostSyncDeltas_SkillTier_FlappingCollapsesTo1(t *testing.T) {
	now := time.Now().UTC()
	var recent []notifications.Notification
	total := 0
	cycle := func(before, after string) {
		em := &recordingEmitter{}
		b := &PlayerSnapshot{SkillTierByPlaylist: map[string]string{"ranked": before}}
		a := &PlayerSnapshot{SkillTierByPlaylist: map[string]string{"ranked": after}}
		EmitPostSyncDeltas(context.Background(), em, "p1", b, a, nil,
			PostSyncDeltaOptions{RecentSkillTiers: recent, Now: now})
		for _, in := range em.emitted {
			if in.Category != notifications.CategorySkillTier {
				continue
			}
			total++
			p, _ := json.Marshal(in.Params)
			recent = append(recent, notifications.Notification{
				Category: notifications.CategorySkillTier, Params: p, CreatedAt: now,
			})
		}
	}
	cycle("csr|Gold|4", "csr|Gold|5") // montée → émet
	cycle("csr|Gold|5", "csr|Gold|4") // démotion → silencieuse
	cycle("csr|Gold|4", "csr|Gold|5") // montée → dédupée
	cycle("csr|Gold|5", "csr|Gold|4") // démotion → silencieuse
	cycle("csr|Gold|4", "csr|Gold|5") // montée → dédupée
	if total != 1 {
		t.Errorf("flapping IV↔V < 24 h → 1 émission, got %d", total)
	}
}

// splitSkillTier doit décomposer correctement la signature et tolérer les
// valeurs malformées.
func TestSplitSkillTier(t *testing.T) {
	cases := []struct {
		in              string
		wantType, wantT string
		wantSub         int
	}{
		{"csr|Onyx|0", "csr", "Onyx", 0},
		{"lusr|Diamond|3", "lusr", "Diamond", 3},
		{"", "", "", 0},
		{"malformed", "malformed", "", 0},
	}
	for _, c := range cases {
		gotType, gotT, gotSub := splitSkillTier(c.in)
		if gotType != c.wantType || gotT != c.wantT || gotSub != c.wantSub {
			t.Errorf("splitSkillTier(%q) = (%q,%q,%d), want (%q,%q,%d)",
				c.in, gotType, gotT, gotSub, c.wantType, c.wantT, c.wantSub)
		}
	}
}

func TestEmitPostSyncDeltas_ThresholdCrossed_KDRatio(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{KDRatio: 0.99, Winrate: 0.40}
	after := &PlayerSnapshot{KDRatio: 1.04, Winrate: 0.43}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil, PostSyncDeltaOptions{})
	if !hasCategory(em.emitted, notifications.CategoryThresholdCrossed) {
		t.Error("expected threshold_crossed (KD up)")
	}
	// Doit avoir 1 emit threshold (KD), pas 2 (winrate ne franchit pas le palier 0.05)
	count := countCategory(em.emitted, notifications.CategoryThresholdCrossed)
	if count != 1 {
		t.Errorf("expected exactly 1 threshold emit, got %d", count)
	}
}

func TestEmitPostSyncDeltas_ThresholdCrossed_NoEmitOnDescent(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{KDRatio: 1.10, Winrate: 0.55}
	after := &PlayerSnapshot{KDRatio: 0.99, Winrate: 0.49}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil, PostSyncDeltaOptions{})
	if hasCategory(em.emitted, notifications.CategoryThresholdCrossed) {
		t.Error("threshold_crossed should NOT emit on descent")
	}
}

func TestEmitPostSyncDeltas_PersonalRecord_SkippedWithoutPDB(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{BestKDA: 2.0}
	after := &PlayerSnapshot{BestKDA: 4.5, BestKDAMatchID: "abc"}
	// pdb nil → personal_record skip
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil, PostSyncDeltaOptions{})
	if hasCategory(em.emitted, notifications.CategoryPersonalRecord) {
		t.Error("personal_record should be skipped when pdb=nil")
	}
}

// ─── B1 : test régression routes notifications (revue 2026-04-29) ──────
//
// Avant le fix B1, EmitPostSyncDeltas émettait des TargetRoute pointant vers
// 3 routes inexistantes côté front : /defis, /help/changelog, /sync.
// Ce test vérifie que toutes les TargetRoute émises matchent un préfixe de
// la whitelist des routes valides documentées dans routeTree.gen.ts.
//
// Si une nouvelle route fantôme réapparaît, ce test échouera avec un message
// nominatif. Test de non-régression au sens politique transverse.

// validPlayerSubpaths : sous-chemins acceptés sous la racine joueur title-scopée
// (/t/{titleSlug}/players/{slug}/…). Ce sont des routes front RÉELLES — les deux
// entrées relocalisées (lot A 2026-07-23) `/stats/synthesis` et `/career/citations`
// remplacent les anciens stubs de redirection `/synthesis` et `/citations` (hop).
var validPlayerSubpaths = []string{
	"/stats/synthesis",
	"/objectifs",
	"/objectifs/index",
	"/ascension",
	"/ascension/realisations",
	"/palmares",
	"/palmares/season-pass",
	"/palmares/prestige",
	"/palmares/relations",
	"/palmares/compare",
	"/career",
	"/career/season-pass", // route actuelle (vs. /palmares/season-pass legacy)
	"/career/citations",   // 2026-07-23 — page Citations sous la section Carrière
	"/match",
	"/matches",
	"/media",
	"/stats",
	"/stats/query",
	"/explorer",
	"/sessions",
	"/squad",
	"/squad/v2",
	"/timeseries",
	"/teammates",
	"/compare",
}

// targetRouteIsValid retourne true si route correspond au format title-scopé
// /t/{titleSlug}/players/{slug}/<sous-chemin valide> (lot A) ou à une route
// agnostique connue. Le sous-chemin doit figurer dans validPlayerSubpaths.
func targetRouteIsValid(route string) bool {
	if route == "" {
		return true // pas de TargetRoute = pas de risque
	}
	if route == "/changelog" || route == "/settings" {
		return true
	}
	// Format title-scopé : /t/{titleSlug}/players/{slug}/<sous-chemin>.
	const tPrefix = "/t/"
	if !strings.HasPrefix(route, tPrefix) {
		return false
	}
	// Retire /t/{titleSlug} → reste /players/{slug}/<sous-chemin>.
	afterTitle := strings.IndexByte(route[len(tPrefix):], '/')
	if afterTitle < 0 {
		return false
	}
	rest := route[len(tPrefix)+afterTitle:]
	const pPrefix = "/players/"
	if !strings.HasPrefix(rest, pPrefix) {
		return false
	}
	rest = rest[len(pPrefix):] // {slug}/<sous-chemin>
	slash := strings.IndexByte(rest, '/')
	if slash < 0 {
		// /t/{title}/players/{slug} sans sous-chemin — accepté mais inutile en notification
		return true
	}
	subPath := rest[slash:]
	for _, valid := range validPlayerSubpaths {
		if subPath == valid {
			return true
		}
	}
	return false
}

// TestEmitPostSyncDeltas_AllTargetRoutesValid verifie que toutes les
// TargetRoute emises pour un large delta correspondent a des routes valides.
func TestEmitPostSyncDeltas_AllTargetRoutesValid(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{
		CurrentRank:               5,
		PersonalAwardCount:        10,
		CitationsCount:            3,
		ChallengePathsCount:       2,
		ChallengeCompletedCount:   1,
		BattlepassCompletedTracks: 0,
		CitationTotalEarnedTiers:  3,
		CitationMasteryCount:      0,
		SkillTierByPlaylist:       map[string]string{"ranked-arena": "csr|Diamond|3"},
		KDRatio:                   1.0,
		Winrate:                   0.50,
		BestKDA:                   3.0,
	}
	after := &PlayerSnapshot{
		CurrentRank:               6,                                               // → career_rank
		PersonalAwardCount:        15,                                              // → objective_completed/assigned
		CitationsCount:            5,                                               // (legacy — n'émet plus challenge_completed)
		ChallengePathsCount:       4,                                               // → challenge_added
		ChallengeCompletedCount:   3,                                               // → challenge_completed
		BattlepassCompletedTracks: 1,                                               // → battlepass_completed
		CitationTotalEarnedTiers:  5,                                               // → citation_tier
		CitationMasteryCount:      1,                                               // → citation_mastery
		SkillTierByPlaylist:       map[string]string{"ranked-arena": "csr|Onyx|0"}, // → skill_tier
		KDRatio:                   1.20,                                            // → threshold_crossed (KD)
		Winrate:                   0.60,                                            // → threshold_crossed (winrate)
		BestKDA:                   5.5,                                             // → personal_record (best_kda)
	}
	EmitPostSyncDeltas(context.Background(), em, "test-player", before, after, nil, PostSyncDeltaOptions{TitleSlug: "halo_infinite"})

	if len(em.emitted) == 0 {
		t.Fatalf("expected at least one emit for the wide delta")
	}

	for _, in := range em.emitted {
		if !targetRouteIsValid(in.TargetRoute) {
			t.Errorf("category=%s emits invalid TargetRoute=%q (route fantome ?)",
				in.Category, in.TargetRoute)
		}
	}
}

// TestEmitPostSyncDeltas_NoFantomRoutes verifie nominativement que les 3
// routes fantomes du bug B1 ne sont plus utilisees.
func TestEmitPostSyncDeltas_NoFantomRoutes(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{CitationsCount: 0, ChallengePathsCount: 0}
	after := &PlayerSnapshot{CitationsCount: 5, ChallengePathsCount: 4}
	EmitPostSyncDeltas(context.Background(), em, "test-player", before, after, nil, PostSyncDeltaOptions{TitleSlug: "halo_infinite"})

	// Routes fantômes B1 (format title-scopé post-lot A) : ne doivent JAMAIS réapparaître.
	fantomRoutes := []string{
		"/t/halo_infinite/players/test-player/defis",
		"/help/changelog",
		"/t/halo_infinite/players/test-player/sync",
	}
	for _, in := range em.emitted {
		for _, fantom := range fantomRoutes {
			if in.TargetRoute == fantom {
				t.Errorf("regression B1 : TargetRoute fantome %q reapparue (category=%s)",
					fantom, in.Category)
			}
		}
	}
}

// ─── A5 : anti-burst cold-start ─────────────────────────────────────────

// (a) before froid + after riche → 0 émission (au lieu de 22).
func TestEmitPostSyncDeltas_ColdStart_SuppressesAll(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{} // tous compteurs à 0 → froid
	after := &PlayerSnapshot{
		CurrentRank:               42,
		PersonalAwardCount:        3434,
		ChallengeCompletedCount:   50,
		BattlepassCompletedTracks: 10,
		CitationTotalEarnedTiers:  200,
		SkillTierByPlaylist:       map[string]string{"ranked-arena": "csr|Onyx|0"},
		KDRatio:                   1.5,
	}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil, PostSyncDeltaOptions{})
	if len(em.emitted) != 0 {
		t.Errorf("cold-start doit supprimer toutes les émissions, got %d", len(em.emitted))
	}
}

// (b) delta objectifs = 25 (> cap) → supprimé, les autres deltas du même cycle passent.
func TestEmitPostSyncDeltas_ImplausibleDelta_Suppressed(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{PersonalAwardCount: 5, ChallengePathsCount: 3}
	after := &PlayerSnapshot{PersonalAwardCount: 30, ChallengePathsCount: 5} // PSA +25 (>20), paths +2
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil, PostSyncDeltaOptions{})
	if hasCategory(em.emitted, notifications.CategoryObjectiveCompleted) {
		t.Error("delta PSA=25 (>cap) doit être supprimé")
	}
	if !hasCategory(em.emitted, notifications.CategoryChallengeAdded) {
		t.Error("le delta challenge_added=2 du même cycle doit passer")
	}
}

// (c) delta = 5 → émis.
func TestEmitPostSyncDeltas_PlausibleDelta_Emitted(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{PersonalAwardCount: 10}
	after := &PlayerSnapshot{PersonalAwardCount: 15}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil, PostSyncDeltaOptions{})
	if !hasCategory(em.emitted, notifications.CategoryObjectiveCompleted) {
		t.Error("delta PSA=5 (<cap) doit être émis")
	}
}

// (d) career_rank previous=0 → supprimé.
func TestEmitPostSyncDeltas_CareerRank_PreviousZero_Suppressed(t *testing.T) {
	em := &recordingEmitter{}
	// PersonalAwardCount non nul → snapshot non froid, on isole la garde career_rank.
	before := &PlayerSnapshot{CurrentRank: 0, PersonalAwardCount: 5}
	after := &PlayerSnapshot{CurrentRank: 5, PersonalAwardCount: 5, CurrentRankName: "Onyx"}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil, PostSyncDeltaOptions{})
	if hasCategory(em.emitted, notifications.CategoryCareerRank) {
		t.Error("career_rank previous=0 doit être supprimé")
	}
}

// (e) career_rank 190→192 → émis.
func TestEmitPostSyncDeltas_CareerRank_RealRankUp_Emitted(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{CurrentRank: 190}
	after := &PlayerSnapshot{CurrentRank: 192, CurrentRankName: "Onyx"}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil, PostSyncDeltaOptions{})
	if !hasCategory(em.emitted, notifications.CategoryCareerRank) {
		t.Error("career_rank 190→192 doit être émis")
	}
}

// (f) before froid ET after froid (nouveau joueur vide) → 0 émission, pas de warn cold-start.
func TestEmitPostSyncDeltas_BothCold_NoEmitNoWarn(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)

	em := &recordingEmitter{}
	EmitPostSyncDeltas(context.Background(), em, "p1", &PlayerSnapshot{}, &PlayerSnapshot{}, nil, PostSyncDeltaOptions{})
	if len(em.emitted) != 0 {
		t.Errorf("before+after froids → 0 émission, got %d", len(em.emitted))
	}
	if strings.Contains(buf.String(), "cold-start") {
		t.Error("aucun warn cold-start attendu quand after est aussi froid")
	}
}

func TestSnapshotLooksCold(t *testing.T) {
	if !snapshotLooksCold(nil) {
		t.Error("nil doit être considéré froid")
	}
	if !snapshotLooksCold(&PlayerSnapshot{}) {
		t.Error("snapshot vide doit être froid")
	}
	if snapshotLooksCold(&PlayerSnapshot{PersonalAwardCount: 1}) {
		t.Error("PSA=1 ne doit pas être froid")
	}
	if snapshotLooksCold(&PlayerSnapshot{SkillTierByPlaylist: map[string]string{"a": "b"}}) {
		t.Error("skill tier présent ne doit pas être froid")
	}
}

// ─── helpers de test ────────────────────────────────────────────────────

func hasCategory(items []notifications.EmitInput, c notifications.Category) bool {
	for _, it := range items {
		if it.Category == c {
			return true
		}
	}
	return false
}

func countCategory(items []notifications.EmitInput, c notifications.Category) int {
	n := 0
	for _, it := range items {
		if it.Category == c {
			n++
		}
	}
	return n
}
