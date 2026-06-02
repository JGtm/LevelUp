// Package sync — contract_test.go : invariants observables d'un cycle de
// sync, que l'implémentation soit V1 (engine.go) ou V2 (v2/orchestrator.go).
//
// Ces tests définissent le CONTRAT que toute implémentation cycle doit
// respecter. Cf. ADR 0020 § "Tests anti-régression critiques".
//
// État D0 : tous les tests sont skippés. L'activation s'enchaîne avec
// les deliverables :
//   - V1 adapter (T_V1_*) : activé en D6 (intégration scheduler)
//   - V2 réel (T_V2_*)    : activé phase par phase (D1..D5)
//   - Parité V1↔V2        : activée en D6 (cible go/no-go shadow run)
//
// Chaque test garde un commentaire "TODO(Dx)" qui pointe le deliverable
// qui doit le décoincer.
//
// Black-box (package sync_test) pour éviter le cycle d'import
// sync ↔ sync/v2 (D6.4 — v2 importe sync pour les types HaloClient).
package sync_test

import (
	"testing"

	syncv2 "levelup/go-api/internal/sync/v2"
)

// CycleRunner abstrait l'exécution d'un cycle complet pour N joueurs.
// V1 sera adapté via une closure qui boucle sur RunDelta + post-sync ;
// V2 utilise syncv2.CycleOrchestrator directement.
//
// L'objet retourné (CycleObservation) capture l'état observable post-cycle
// (compteurs API call, contenu DB, warnings) sans exposer l'implémentation.
type CycleRunner interface {
	RunCycle(t *testing.T, fixture CycleFixture) CycleObservation
}

// CycleFixture décrit le scénario de test : joueurs + dataset API attendu.
// Les implémentations concrètes (D1) ajouteront la fake API server et le
// dataset DB pré-rempli.
type CycleFixture struct {
	Players []FixturePlayer
	// MatchesPerPlayer : pour chaque player_slug, la liste de match_ids
	// que l'API doit retourner (ordre reverse-chronologique). Inclure les
	// matchs partagés (même match_id chez plusieurs joueurs) permet de
	// tester la cross-player dedup.
	MatchesPerPlayer map[string][]string
	// KnownMatches : matchs déjà présents en DB avant le cycle (par
	// player_slug). Sert à tester la détection delta + skip.
	KnownMatches map[string][]string
}

// FixturePlayer minimaliste pour le contract test.
type FixturePlayer struct {
	Gamertag   string
	XUID       string
	PlayerSlug string
}

// CycleObservation capture l'état observable après un cycle, sans
// dépendance à V1 ou V2 spécifiquement.
type CycleObservation struct {
	// APICallsByMatch : nombre d'appels GetMatchStats émis par match_id.
	// Invariant critique : doit être ≤ 1 par match (cross-player dedup).
	APICallsByMatch map[string]int
	// SharedMatchRegistry : match_ids présents dans shared.match_registry.
	SharedMatchRegistry []string
	// SharedParticipants : pour chaque match_id, les xuids présents dans
	// shared.match_participants.
	SharedParticipants map[string][]string
	// PlayerEnrichment : pour chaque player_slug, les match_ids présents
	// dans son player_match_enrichment.
	PlayerEnrichment map[string][]string
	// Warnings émis pendant le cycle (best-effort, pour assertions
	// négatives type "aucun warning citation").
	Warnings []string
}

// ──────────────────────────────────────────────────────────────────────
// Contract tests — invariants observables d'un cycle valide
// ──────────────────────────────────────────────────────────────────────

// TestContract_AllAPIMatchesPersisted vérifie que tous les matchs retournés
// par l'API qui ne sont pas dans KnownMatches finissent en
// shared.match_registry.
//
// TODO(D6) : activer pour V1 via adapter ; activer pour V2 quand Phase 5
// (Persist) est livrée.
func TestContract_AllAPIMatchesPersisted(t *testing.T) {
	t.Skip("contract suite scaffold — activée en D6 du plan ADR 0020")
}

// TestContract_ParticipantsContainTrackedXUIDs vérifie que pour chaque match
// retenu, shared.match_participants contient une ligne pour chaque xuid
// tracké qui a participé.
//
// Critique pour la cross-player dedup : sans cette ligne, le prochain
// loadKnownMatchIDs du joueur ne reconnaîtra pas le match comme connu.
//
// TODO(D6) : activer pour V1 et V2.
func TestContract_ParticipantsContainTrackedXUIDs(t *testing.T) {
	t.Skip("contract suite scaffold — activée en D6 du plan ADR 0020")
}

// TestContract_PlayerEnrichmentMatchesParticipation vérifie que pour chaque
// joueur, son player_match_enrichment contient une ligne par match auquel
// son xuid a participé (et seulement ceux-là).
//
// TODO(D6) : activer pour V1 et V2.
func TestContract_PlayerEnrichmentMatchesParticipation(t *testing.T) {
	t.Skip("contract suite scaffold — activée en D6 du plan ADR 0020")
}

// TestContract_NoDuplicateRows vérifie qu'aucune table critique
// (match_registry, match_participants, player_match_enrichment) ne contient
// de lignes dupliquées (PK respectées).
//
// Anti-régression directe pour les patterns ART qui re-insèrent (cf. ADR
// 0019).
//
// TODO(D6) : activer pour V1 et V2.
func TestContract_NoDuplicateRows(t *testing.T) {
	t.Skip("V1 actif : TestContract_NoDuplicateRows_V1 (package sync, dataset PVP+PVE) ; scaffold V2 en attente")
}

// TestContract_CrossPlayerDedupOneAPICallPerMatch est l'invariant central
// qui justifie la refonte V2 : pour un match joué par P1+P2, exactement 1
// appel GetMatchStats API doit être émis sur le cycle.
//
// V1 échoue probablement ce test à cause du worker BatchQueue monothread
// qui ne commit pas en DB avant que P2 ne fasse son loadKnown (cf. ADR
// 0020 § Causes racines).
//
// TODO(D6) : activer pour V2 ; en V1 ce test sert de baseline (XFAIL
// documenté avant la bascule prod).
func TestContract_CrossPlayerDedupOneAPICallPerMatch(t *testing.T) {
	t.Skip("V1 actif : TestContract_CrossPlayerDedup_V1 (package sync, via loadKnownMatchIDs/shared.match_participants) ; scaffold V2 en attente")
}

// TestContract_PartialFailureIsolation vérifie qu'un échec sur un joueur
// (token expiré, API 500) n'empêche pas les autres joueurs de terminer
// leur cycle avec succès.
//
// TODO(D6) : activer pour V1 et V2.
func TestContract_PartialFailureIsolation(t *testing.T) {
	t.Skip("V1 actif : TestContract_PartialFailureIsolation_V1 (package sync) ; scaffold V2 en attente")
}

// TestContract_CycleIdempotent vérifie que rejouer le même cycle V2
// immédiatement après produit le même état DB (rien inséré 2 fois, pas
// d'erreur PK).
//
// TODO(D6) : activer pour V1 et V2.
func TestContract_CycleIdempotent(t *testing.T) {
	t.Skip("V1 actif : TestContract_CycleIdempotent_V1 (package sync) ; scaffold V2 en attente")
}

// ──────────────────────────────────────────────────────────────────────
// Anti-régression incident spécifiques
// ──────────────────────────────────────────────────────────────────────

// TestContract_HaloAPIURLFormatXUID est l'anti-régression de l'incident
// mai 2026 (14 jours de sync à inserted=0). L'URL /matches doit utiliser
// xuid(NNN), pas le gamertag brut.
//
// L'invariant V1 est désormais ACTIF : voir TestContract_HaloAPIURLFormatXUID_V1
// (package sync, via RunDelta + capture de l'arg GetMatchHistory). Ce scaffold
// black-box (fake server) reste skippé en attendant la suite V2.
func TestContract_HaloAPIURLFormatXUID(t *testing.T) {
	t.Skip("V1 actif : TestContract_HaloAPIURLFormatXUID_V1 (package sync) ; scaffold V2 en attente")
}

// TestContract_MetadataDSNAlignment est l'anti-régression du bug citations
// 2026-05-25 : aucune ouverture concurrente de metadata.duckdb avec des
// configurations différentes (RO vs RW) ne doit avoir lieu pendant un
// cycle.
//
// TODO(D6) : activer pour V1 et V2. Probablement nécessite un hook
// duckdbpkg pour compter les Open* et leurs cache keys.
func TestContract_MetadataDSNAlignment(t *testing.T) {
	t.Skip("contract suite scaffold — activée en D6 du plan ADR 0020")
}

// TestContract_DrainVisibilityPhase5ToPhase6 est l'anti-régression du bug
// drain non-visible : après Phase 5 (Persist), la lecture en Phase 6
// (PostSync) doit voir tous les writes (pas de WAL pending qui occulte).
//
// Spécifique V2. En V1, ce test n'a pas de sens.
//
// TODO(D5) : activer quand Phase 5 + Phase 6 sont câblées en V2.
func TestContract_V2_DrainVisibilityPhase5ToPhase6(t *testing.T) {
	t.Skip("contract suite scaffold — activée en D5 du plan ADR 0020")
}

// ──────────────────────────────────────────────────────────────────────
// Smoke tests D0 (vérifient le scaffold lui-même)
// ──────────────────────────────────────────────────────────────────────

// TestD0_StubOrchestratorReturnsErrNotImplemented vérifie que le stub V2
// retourne bien ErrNotImplemented (sentinelle pour le fallback scheduler).
//
// Ce test reste actif au-delà de D0 — il documente que le stub doit
// rester disponible jusqu'à D6 (avant on a besoin de pouvoir tester le
// dispatch scheduler sans avoir l'implémentation complète).
func TestD0_StubOrchestratorReturnsErrNotImplemented(t *testing.T) {
	orch := syncv2.NewStubOrchestrator()
	_, err := orch.Run(t.Context(), []syncv2.PlayerProfile{
		{Gamertag: "TestPlayer", XUID: "0000000000000001", PlayerSlug: "testplayer"},
	})
	if err == nil {
		t.Fatal("attendu ErrNotImplemented, reçu nil")
	}
	if err != syncv2.ErrNotImplemented {
		t.Fatalf("attendu syncv2.ErrNotImplemented, reçu %v", err)
	}
}

// TestD0_StubOrchestratorReturnsCycleResultWithEmptyMaps vérifie que le
// stub retourne un CycleResult exploitable (maps initialisées) même sur
// erreur — pour ne pas paniquer le scheduler en dispatch D6.
func TestD0_StubOrchestratorReturnsCycleResultWithEmptyMaps(t *testing.T) {
	orch := syncv2.NewStubOrchestrator()
	res, _ := orch.Run(t.Context(), nil)
	if res.PerPlayer == nil {
		t.Error("CycleResult.PerPlayer doit être initialisée, pas nil")
	}
	if res.PhaseDurations == nil {
		t.Error("CycleResult.PhaseDurations doit être initialisée, pas nil")
	}
}
