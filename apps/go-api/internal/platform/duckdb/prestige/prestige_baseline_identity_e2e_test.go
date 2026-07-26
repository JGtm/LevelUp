//go:build integration

// Tests d'intégration de la FRONTIÈRE D'IDENTITÉ Prestige ↔ données de match
// (correctif 2026-07-26).
//
// Ce que ces tests prouvent, sur de VRAIES tables shared_matches_v2 :
//
//  1. TestHaloBaselineProvider_XUIDvsPlayerSlug — un provider lié au XUID ramène
//     les matchs du joueur ; un provider lié au PLAYER_SLUG (le défaut corrigé)
//     n'en ramène AUCUN. C'est la démonstration nue des deux identités.
//  2. TestEvaluateForUser_ProgressesOnRealMatches — bout en bout, un objectif
//     dont le user_id est un player_slug PROGRESSE réellement puis passe
//     « acquis » avec crédit de PP. Ce test ÉCHOUE sur le code pré-correctif
//     (0 match remonté → EvalReasonInsufficient, NewValue 0, aucune transition).
//
// Le shared est seedé avec un DÉCOY (un autre xuid, valeurs très différentes) :
// une lecture qui ignorerait le filtre d'identité serait donc détectée aussi.

package prestige

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/prestige"
)

// Identités du scénario — volontairement RÉALISTES et de formes incompatibles :
// le slug est le gamertag applicatif (db_profiles.json), le xuid est numérique.
const (
	identityPlayerSlug = "JGtm"
	identityXUID       = "2533274823110022"
	identityDecoyXUID  = "2535469190789936"
	identityTitleSlug  = "halo_infinite"
)

// identityClock : horloge figée du scénario (évaluations déterministes).
var identityClock = time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)

// identitySeed décrit un lot de matchs à semer (regroupé en struct : CLAUDE.md
// n°5, ≤ 5 paramètres).
type identitySeed struct {
	Prefix string    // préfixe des match_id — évite toute collision entre lots
	XUID   string    // identité XBOX du participant
	Count  int       // nombre de matchs
	Kills  int       // frags par match
	Latest time.Time // date du match le plus récent (les suivants reculent d'1 h)
}

// seedIdentityMatches insère un lot de matchs dans le shared de test.
func seedIdentityMatches(t *testing.T, shared *duckdb.DB, s identitySeed) {
	t.Helper()
	ctx := context.Background()
	raw := shared.SQLDb()
	for i := 0; i < s.Count; i++ {
		id := s.Prefix + "_" + string(rune('a'+i))
		at := s.Latest.Add(-time.Duration(i) * time.Hour)
		if _, err := raw.ExecContext(ctx,
			`INSERT OR IGNORE INTO match_registry (match_id, start_time, start_time_utc)
			 VALUES (?, ?, ?)`, id, at, at); err != nil {
			t.Fatalf("seed match_registry %s: %v", id, err)
		}
		if _, err := raw.ExecContext(ctx,
			`INSERT INTO match_participants (match_id, xuid, gamertag, kills)
			 VALUES (?, ?, ?, ?)`, id, s.XUID, "GT_"+s.XUID, s.Kills); err != nil {
			t.Fatalf("seed match_participants %s/%s: %v", id, s.XUID, err)
		}
	}
}

// TestHaloBaselineProvider_XUIDvsPlayerSlug — les deux identités ne sont pas
// interchangeables, et c'est mesurable : même base, même métrique, seul le
// paramètre d'identité change.
func TestHaloBaselineProvider_XUIDvsPlayerSlug(t *testing.T) {
	shared := setupPrestigeDB(t, migration.TargetShared)
	latest := identityClock.Add(-time.Hour)
	seedIdentityMatches(t, shared, identitySeed{Prefix: "own", XUID: identityXUID, Count: 10, Kills: 12, Latest: latest})
	seedIdentityMatches(t, shared, identitySeed{Prefix: "decoy", XUID: identityDecoyXUID, Count: 10, Kills: 99, Latest: latest})

	ctx := context.Background()
	reader := duckdb.LegacySharedReader(shared)

	byXUID, err := NewHaloBaselineProvider(reader, identityXUID).
		RecentMatches(ctx, identityTitleSlug, "kills", 20)
	if err != nil {
		t.Fatalf("RecentMatches(xuid): %v", err)
	}
	if len(byXUID) != 10 {
		t.Fatalf("provider lié au XUID = %d matchs, want 10", len(byXUID))
	}
	for _, m := range byXUID {
		if m.MetricValue != 12 {
			t.Fatalf("valeur métrique = %v, want 12 (les lignes du decoy ne doivent PAS fuiter)", m.MetricValue)
		}
		if m.StartedAt.IsZero() {
			t.Errorf("StartedAt vide pour %s — le fragment timestamp canonique ne projette rien", m.MatchID)
		}
	}

	// Le défaut corrigé : alimenter le provider avec l'identité APPLICATIVE.
	bySlug, err := NewHaloBaselineProvider(reader, identityPlayerSlug).
		RecentMatches(ctx, identityTitleSlug, "kills", 20)
	if err != nil {
		t.Fatalf("RecentMatches(slug): %v", err)
	}
	if len(bySlug) != 0 {
		t.Fatalf("provider lié au PLAYER_SLUG = %d matchs, want 0 — "+
			"match_participants.xuid n'accepte que l'identité Xbox", len(bySlug))
	}

	// Provider sans identité : dégradation silencieuse INTERDITE côté logs, mais
	// contractuellement 0 match sans erreur (le service ne doit pas planter).
	none, err := NewHaloBaselineProvider(reader, "").
		RecentMatches(ctx, identityTitleSlug, "kills", 20)
	if err != nil || len(none) != 0 {
		t.Fatalf("provider sans xuid = (%d matchs, err=%v), want (0, nil)", len(none), err)
	}
}

// newIdentityService assemble un prestige.Service réel branché sur le VRAI
// HaloBaselineProvider (pas un stub) — c'est tout l'intérêt du test.
func newIdentityService(t *testing.T, shared *duckdb.DB) (prestige.Service, *PrestigeChallengeRepo) {
	t.Helper()
	playerDB := setupPrestigeDB(t, migration.TargetPlayer)
	socialDB := setupPrestigeDB(t, migration.TargetSharedSocial)
	challenges := NewPrestigeChallengeRepo(playerDB)

	svc := prestige.NewService(prestige.Deps{
		Tuning:        prestige.DefaultTuning(),
		Challenges:    challenges,
		Arcs:          NewPrestigeArcRepo(playerDB),
		Moments:       NewPrestigeMomentCardRepo(playerDB),
		Prestige:      NewPrestigeSocialRepo(socialDB),
		Telemetry:     NewPrestigeTelemetryRepo(playerDB),
		BaselineState: NewPrestigeBaselineStateRepo(playerDB),
		// LE point du test : le provider est lié au XUID, pendant que le défi
		// porte un user_id = player_slug.
		BaselineProvider: NewHaloBaselineProvider(duckdb.LegacySharedReader(shared), identityXUID),
		Now:              func() time.Time { return identityClock },
	})
	return svc, challenges
}

// identityChallenge : objectif « moyenne de frags sur les 5 derniers matchs ».
// user_id = player_slug, comme TOUTE la chaîne HTTP et le hook post-sync.
func identityChallenge(target float64) prestige.Challenge {
	return prestige.Challenge{
		ID:          "ch_identity_001",
		UserID:      identityPlayerSlug,
		TitleSlug:   identityTitleSlug,
		Metric:      "kills",
		Target:      target,
		WindowType:  prestige.WindowLastNMatches,
		WindowValue: "5",
		Cadence:     prestige.CadenceFree,
		EvalType:    prestige.EvalThreshold,
		Mode:        prestige.ModeLibre,
		Tier:        prestige.TierNormal,
		DataTier:    prestige.DataFull,
		Label:       "Frags moyens",
		Status:      prestige.StatusActive,
		CreatedAt:   identityClock.Add(-48 * time.Hour),
	}
}

// TestEvaluateForUser_ProgressesOnRealMatches — LE test de non-régression du
// défaut d'identité. Sur le code pré-correctif, RecentMatches filtrait
// `WHERE mp.xuid = 'JGtm'` : 0 ligne, donc EvalReasonInsufficient et NewValue 0
// à la phase 1, et aucune transition à la phase 2. Les deux phases échouent.
func TestEvaluateForUser_ProgressesOnRealMatches(t *testing.T) {
	shared := setupPrestigeDB(t, migration.TargetShared)
	old := identityClock.Add(-24 * time.Hour)
	seedIdentityMatches(t, shared, identitySeed{Prefix: "own", XUID: identityXUID, Count: 10, Kills: 12, Latest: old})
	seedIdentityMatches(t, shared, identitySeed{Prefix: "decoy", XUID: identityDecoyXUID, Count: 10, Kills: 99, Latest: old})

	ctx := context.Background()
	svc, challenges := newIdentityService(t, shared)
	if err := challenges.Create(ctx, identityChallenge(14)); err != nil {
		t.Fatalf("create challenge: %v", err)
	}

	// ── Phase 1 : 10 matchs à 12 frags, cible 14 → l'objectif PROGRESSE ──
	outcomes, err := svc.EvaluateForUser(ctx, identityPlayerSlug, identityTitleSlug)
	if err != nil {
		t.Fatalf("EvaluateForUser (phase 1): %v", err)
	}
	if len(outcomes) != 1 {
		t.Fatalf("outcomes = %d, want 1", len(outcomes))
	}
	got := outcomes[0]
	if got.NewValue != 12 {
		t.Errorf("valeur mesurée = %v, want 12 — l'objectif ne progresse pas sur les matchs réels "+
			"(symptôme exact du défaut d'identité slug/xuid)", got.NewValue)
	}
	if got.Reason != prestige.EvalReasonProgress {
		t.Errorf("raison = %q, want %q", got.Reason, prestige.EvalReasonProgress)
	}
	if got.NewStatus != prestige.StatusActive {
		t.Errorf("statut = %q, want %q (cible 14 non atteinte avec 12)", got.NewStatus, prestige.StatusActive)
	}

	// ── Phase 2 : 10 matchs de plus à 20 frags → moyenne 16 ≥ 14 → acquis ──
	seedIdentityMatches(t, shared, identitySeed{
		Prefix: "own2", XUID: identityXUID, Count: 10, Kills: 20,
		Latest: identityClock.Add(-time.Hour),
	})

	outcomes, err = svc.EvaluateForUser(ctx, identityPlayerSlug, identityTitleSlug)
	if err != nil {
		t.Fatalf("EvaluateForUser (phase 2): %v", err)
	}
	if len(outcomes) != 1 {
		t.Fatalf("outcomes (phase 2) = %d, want 1", len(outcomes))
	}
	got = outcomes[0]
	if got.NewStatus != prestige.StatusCompleted {
		t.Fatalf("statut = %q (valeur %v), want %q", got.NewStatus, got.NewValue, prestige.StatusCompleted)
	}
	if got.Reason != prestige.EvalReasonTargetReached {
		t.Errorf("raison = %q, want %q", got.Reason, prestige.EvalReasonTargetReached)
	}
	if got.PPCredited <= 0 {
		t.Errorf("PP crédités = %d, want > 0 (palier normal, data_tier full)", got.PPCredited)
	}

	// La transition est PERSISTÉE (le défi ne sera pas re-crédité au cycle suivant).
	stored, err := challenges.Get(ctx, "ch_identity_001")
	if err != nil {
		t.Fatalf("get challenge: %v", err)
	}
	if stored.Status != prestige.StatusCompleted {
		t.Errorf("statut persisté = %q, want %q", stored.Status, prestige.StatusCompleted)
	}
}
