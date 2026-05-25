// Package v2 — persist.go : Phase 5 du pipeline V2 (ADR 0020).
//
// Phase 5 = construction d'un méga-batch unique à partir des buffers
// Phase 3 (SharedMatchData) + Phase 4 (PlayerEnrichmentData), et
// délégation à un CycleBatchPersister qui écrit en transaction.
//
// Pas de goroutines ici, pas de parallélisme — c'est délibéré : un
// SEUL writer prend le shared lease UNE seule fois pour persister TOUT
// le cycle. C'est ce qui élimine la contention 60s du drain timeout V1.
//
// L'implémentation CycleBatchPersister V1-bridge (D6) construit les
// MatchBatch correspondants via persist.BatchBuilder et les submit à
// la queue async existante. Pour D4, on test contre un mock.
package v2

import (
	"context"
	"fmt"
	"time"
)

// CycleBatch agrège toutes les données fetchées d'un cycle V2, prêtes
// à être persistées en transaction. Construit par RunPersist à partir
// des outputs de Phase 3 et Phase 4.
//
// Invariants à respecter par les implémentations CycleBatchPersister :
//   - Idempotence : appeler PersistCycle 2× sur le même CycleBatch ne
//     duplique aucune ligne (SharedPersister utilise EXISTS(match_id),
//     PlayerPersister idem).
//   - Atomicité par DB cible : chaque base (shared, player_X) est
//     écrite dans UNE transaction. Crash mid-write → rollback complet
//     de cette base, le WAL persiste pour retry au prochain boot.
//   - Aucun UPDATE/UPSERT sur les tables critiques (cf. ADR 0019) — uniquement INSERT.
type CycleBatch struct {
	// CycleID identifie ce batch pour diagnostic + WAL recovery. Format
	// libre, recommandé : "v2-cycle-<timestamp_ns>" (lexicographically
	// ordered, unique cross-restart).
	CycleID string

	// Matches : 1 entrée par match unique fetché en Phase 3.
	Matches map[string]SharedMatchData

	// Enrichments : indexation par joueur, puis par match. Phase 5 itère
	// par joueur lors de la construction des sous-batches player.*.
	Enrichments map[string]map[string]PlayerEnrichmentData

	// PlayerBySlug : map des PlayerProfile du cycle courant indexée par
	// PlayerSlug. Permet au persister de résoudre Gamertag+XUID depuis
	// SharedMatchData.Fetcher (= PlayerSlug) sans dépendre d'un state
	// statique. Renseignée par l'orchestrator avant PersistCycle.
	PlayerBySlug map[string]PlayerProfile

	// BuiltAt timestamp de construction du batch (pour métriques).
	BuiltAt time.Time
}

// CycleBatchPersister persiste un CycleBatch complet en respectant les
// invariants d'idempotence et d'atomicité par DB cible.
//
// L'implémentation V1-bridge (D6) traduit CycleBatch → persist.MatchBatch
// par match (un batch par match, comme aujourd'hui) et les submit en
// rafale à la queue. Le worker monothread traite en série mais SANS
// concurrence avec d'autres syncs (le cycle est seul à écrire à cet
// instant). Variante future : 1 méga-MatchBatch = 1 TX par DB (gain
// supplémentaire à mesurer).
type CycleBatchPersister interface {
	PersistCycle(ctx context.Context, batch CycleBatch) error
}

// PersistResult agrège les métriques de Phase 5.
//
// Sémantique transactionnelle stricte : sur succès, on compte tous les
// inputs ; sur erreur, on compte 0 (rollback). Pas de demi-mesure.
type PersistResult struct {
	MatchesPersisted     int // len(batch.Matches) si succès, 0 si erreur
	EnrichmentsPersisted int // somme des len(map) si succès, 0 sinon
	Duration             time.Duration
	Err                  error // global error si PersistCycle échoue
}

// RunPersist construit le CycleBatch à partir des outputs Phase 3+4
// et délègue à persister.PersistCycle.
//
// Sémantique :
//   - Pas de goroutines : 1 writer, 1 batch, 1 appel.
//   - CycleID auto-généré ("v2-cycle-<unix_nano>") pour traçabilité.
//   - Sur succès : counts == len(input). Sur erreur : counts == 0
//     (sémantique transactionnelle stricte, le caller voit Err != nil).
//   - Si fetched.Matches et enrichments.Enrichments sont vides : skip
//     l'appel persister (return PersistResult zéro), pas d'erreur.
//
// playerBySlug : map des PlayerProfile du cycle, passée dans le
// CycleBatch pour permettre au persister de résoudre Gamertag+XUID
// depuis le PlayerSlug du fetcher canonical (cf. persist_v1bridge.go).
func RunPersist(
	ctx context.Context,
	fetched FetchSharedResult,
	enrichments FetchPlayerResult,
	playerBySlug map[string]PlayerProfile,
	persister CycleBatchPersister,
) PersistResult {
	start := time.Now()

	totalEnrichments := 0
	for _, m := range enrichments.Enrichments {
		totalEnrichments += len(m)
	}

	if len(fetched.Matches) == 0 && totalEnrichments == 0 {
		return PersistResult{Duration: time.Since(start)}
	}

	batch := CycleBatch{
		CycleID:      fmt.Sprintf("v2-cycle-%d", time.Now().UnixNano()),
		Matches:      fetched.Matches,
		Enrichments:  enrichments.Enrichments,
		PlayerBySlug: playerBySlug,
		BuiltAt:      time.Now(),
	}

	if err := persister.PersistCycle(ctx, batch); err != nil {
		return PersistResult{
			Duration: time.Since(start),
			Err:      fmt.Errorf("persist cycle %s: %w", batch.CycleID, err),
		}
	}

	return PersistResult{
		MatchesPersisted:     len(batch.Matches),
		EnrichmentsPersisted: totalEnrichments,
		Duration:             time.Since(start),
	}
}
