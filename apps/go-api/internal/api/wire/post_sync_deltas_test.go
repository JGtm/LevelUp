// Package api — post_sync_deltas_test.go : tests des helpers de delta-detection
// et de l'émetteur EmitPostSyncDeltas via un emitter recording in-memory.
package wire

import (
	"context"
	"testing"

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
	EmitPostSyncDeltas(context.Background(), nil, "p1", &PlayerSnapshot{}, &PlayerSnapshot{}, nil)
	// nil before
	EmitPostSyncDeltas(context.Background(), em, "p1", nil, &PlayerSnapshot{}, nil)
	// nil after
	EmitPostSyncDeltas(context.Background(), em, "p1", &PlayerSnapshot{}, nil, nil)
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
	EmitPostSyncDeltas(context.Background(), em, "p1", snap, snap, nil)
	if len(em.emitted) != 0 {
		t.Errorf("expected 0 emits when snapshots equal, got %d", len(em.emitted))
	}
}

func TestEmitPostSyncDeltas_CareerRank(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{CurrentRank: 5, CurrentRankName: "Hero"}
	after := &PlayerSnapshot{CurrentRank: 6, CurrentRankName: "Onyx"}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil)
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
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil)
	// Doit emit objective_completed ET objective_assigned (delta>0 sur PSA)
	if !hasCategory(em.emitted, notifications.CategoryObjectiveCompleted) {
		t.Error("expected objective_completed")
	}
	if !hasCategory(em.emitted, notifications.CategoryObjectiveAssigned) {
		t.Error("expected objective_assigned (MVP : delta = both)")
	}
}

func TestEmitPostSyncDeltas_ChallengeCompleted_AndAdded(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{ChallengeCompletedCount: 1, ChallengePathsCount: 5}
	after := &PlayerSnapshot{ChallengeCompletedCount: 4, ChallengePathsCount: 7}
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil)
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
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil)
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
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil)
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
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil)
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
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil)
	count := countCategory(em.emitted, notifications.CategorySkillTier)
	if count != 2 {
		t.Errorf("expected 2 skill_tier emits (1 promoted + 1 new playlist), got %d", count)
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
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil)
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
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil)
	if hasCategory(em.emitted, notifications.CategoryThresholdCrossed) {
		t.Error("threshold_crossed should NOT emit on descent")
	}
}

func TestEmitPostSyncDeltas_PersonalRecord_SkippedWithoutPDB(t *testing.T) {
	em := &recordingEmitter{}
	before := &PlayerSnapshot{BestKDA: 2.0}
	after := &PlayerSnapshot{BestKDA: 4.5, BestKDAMatchID: "abc"}
	// pdb nil → personal_record skip
	EmitPostSyncDeltas(context.Background(), em, "p1", before, after, nil)
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

// validPlayerSubpaths : sous-chemins acceptés sous /players/{slug}/.
var validPlayerSubpaths = []string{
	"/synthesis",
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
	"/citations",          // 2026-05-16 — pages citations/commendations
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

// targetRouteIsValid retourne true si route correspond a un prefixe valide
// (whitelist) ou a /players/{*}/<sous-chemin valide>.
func targetRouteIsValid(route string) bool {
	if route == "" {
		return true // pas de TargetRoute = pas de risque
	}
	if route == "/changelog" || route == "/settings" {
		return true
	}
	// /players/{slug}/<sous-chemin>
	const prefix = "/players/"
	if len(route) <= len(prefix) || route[:len(prefix)] != prefix {
		return false
	}
	rest := route[len(prefix):]
	// Skip the slug segment.
	slash := -1
	for i, c := range rest {
		if c == '/' {
			slash = i
			break
		}
	}
	if slash < 0 {
		// /players/{slug} sans sous-chemin — accepté mais inutile en notification
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
	EmitPostSyncDeltas(context.Background(), em, "test-player", before, after, nil)

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
	EmitPostSyncDeltas(context.Background(), em, "test-player", before, after, nil)

	fantomRoutes := []string{
		"/players/test-player/defis",
		"/help/changelog",
		"/players/test-player/sync",
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
