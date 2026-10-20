//go:build cgo

// Package api — build_queue_derivations_cgo_test.go : LE DEPOT D'UN OUVRIER DECLENCHE LES
// DERIVATIONS (constat A1 du registre v2).
//
// # Ce que ce test tient, et pourquoi il n'existait pas
//
// Les trois dérivations post-cuisson (coup d'envoi mesuré, résumé d'usage, statistiques
// d'Assaut) n'étaient câblées que sur la branche « construction locale » de
// `replayartifacts.Run` — or `local` est REFUSÉ en production par construction
// (`replaybuild.DecidePlacement`), et le placement par défaut y est `worker`. Le jour de
// l'activation de l'ouvrier distant, `match_usage_*`, `match_bomb_stats` et `real_start_time`
// seraient restés vides SANS QU'AUCUN COMPTEUR NE LE DISE.
//
// Le maillon manquant était exactement celui-ci : `StoreBuildArtifact` valide, range, compte,
// journalise — et rendait la main. Ce test prouve qu'il DÉRIVE, avec l'identité du job et le
// chemin RANGÉ.
//
// # Pourquoi le seam, et ce qu'il ne masque pas
//
// Le chemin réel des dérivations acquiert un writer shared (lease + B-swap) : l'exercer ici
// demanderait une base partagée réelle et un provider, c'est-à-dire de tester les persisters
// une seconde fois. Le seam `replayDerivationsFn` (nil en production, même parti pris que
// `replayJobFactsFn`) isole LA question de ce test : le rangeur appelle-t-il les dérivations,
// avec quoi. Ce que les dérivations écrivent est éprouvé chez elles
// (`replayartifacts/usage_integration_test.go`, `bombstats_integration_test.go`).
//
// Le store monitoring est RÉEL (DuckDB temporaire) : le job réclamé est un vrai job, et le
// rangement passe par `replaybuild.StoreArtifact` — d'où le tag cgo.
package wire

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/sync/replayartifacts"
)

const (
	derivWorker  = "ouvrier-derivations"
	derivMatchID = "9f8e7d6c-5b4a-4938-8271-0a1b2c3d4e5f"
)

// TestStoreBuildArtifact_DeclencheLesDerivations : le cas nominal du chemin OUVRIER.
func TestStoreBuildArtifact_DeclencheLesDerivations(t *testing.T) {
	reg := buildQueueRegistry(t)
	var vus []replayartifacts.ArtefactRange
	var vuSlug string
	appels := 0
	reg.replayDerivationsFn = func(_ context.Context, slug string, ranges []replayartifacts.ArtefactRange) {
		appels++
		vuSlug = slug
		vus = append(vus, ranges...)
	}

	job := enfilerJobDerivations(t, reg, derivMatchID)
	claimed, err := reg.monitoringStore.ClaimBuildJob(context.Background(), derivWorker, "host", "v")
	if err != nil || claimed == nil {
		t.Fatalf("ClaimBuildJob: err=%v job=%v", err, claimed)
	}

	blob := artefactMinimal(t, derivMatchID)
	recu, err := reg.StoreBuildArtifact(context.Background(), job.JobID, derivWorker, blob)
	if err != nil {
		t.Fatalf("StoreBuildArtifact: %v", err)
	}

	if appels != 1 {
		t.Fatalf("dérivations appelées %d fois pour UN artefact rangé, attendu 1 — "+
			"c'est exactement le maillon du constat A1", appels)
	}
	if vuSlug != titlePkg.DefaultSlug {
		t.Errorf("titre passé aux dérivations = %q, attendu %q", vuSlug, titlePkg.DefaultSlug)
	}
	if len(vus) != 1 {
		t.Fatalf("%d artefact(s) transmis, attendu 1", len(vus))
	}
	// L'IDENTITÉ VIENT DU JOB, PAS DU DOCUMENT : c'est la règle de `StoreBuildArtifact`, et
	// les dérivations écrivent en base sous cette clé (match_registry est indexé par le
	// match_id COMPLET).
	if vus[0].MatchID != derivMatchID {
		t.Errorf("match_id transmis = %q, attendu celui du JOB %q", vus[0].MatchID, derivMatchID)
	}
	// LE CHEMIN EST CELUI DU FICHIER RANGÉ, et il existe : projeter le blob candidat
	// écrirait en base ce que le disque ne porte pas (StoreArtifact peut refuser).
	if _, statErr := os.Stat(vus[0].Path); statErr != nil {
		t.Errorf("chemin transmis introuvable sur disque (%s) : %v", vus[0].Path, statErr)
	}
	attendu := titlePkg.NewPathResolver(reg.cfg.RepoRoot).
		ReplayArtifactPath(titlePkg.DefaultSlug, derivMatchID)
	if vus[0].Path != attendu {
		t.Errorf("chemin transmis = %q, attendu la place canonique %q", vus[0].Path, attendu)
	}
	if recu.MatchID != derivMatchID {
		t.Errorf("reçu.MatchID = %q, attendu %q", recu.MatchID, derivMatchID)
	}
}

// TestStoreBuildArtifact_ArtefactRefuseNeDerivePas : un dépôt REFUSÉ (identité qui ne
// correspond pas au job) ne doit rien dériver — sinon on projetterait un artefact que le
// rangeur vient d'écarter.
func TestStoreBuildArtifact_ArtefactRefuseNeDerivePas(t *testing.T) {
	reg := buildQueueRegistry(t)
	appels := 0
	reg.replayDerivationsFn = func(context.Context, string, []replayartifacts.ArtefactRange) { appels++ }

	job := enfilerJobDerivations(t, reg, derivMatchID)
	if _, err := reg.monitoringStore.ClaimBuildJob(context.Background(), derivWorker, "host", "v"); err != nil {
		t.Fatalf("ClaimBuildJob: %v", err)
	}
	// Un document qui revendique un AUTRE match : `StoreArtifact` refuse.
	blob := artefactMinimal(t, "00000000-1111-4222-8333-444444444444")
	if _, err := reg.StoreBuildArtifact(context.Background(), job.JobID, derivWorker, blob); err == nil {
		t.Fatal("un artefact d'un autre match doit être REFUSÉ")
	}
	if appels != 0 {
		t.Errorf("dérivations appelées %d fois sur un dépôt refusé, attendu 0", appels)
	}
}

// enfilerJobDerivations met un job de rejeu en file (sans résolution de manifeste : hors sujet).
func enfilerJobDerivations(t *testing.T, reg *ServiceRegistry, matchID string) domain.BuildQueueJob {
	t.Helper()
	job, created, err := reg.monitoringStore.EnqueueBuildJob(context.Background(), ops.EnqueueBuildJobRequest{
		JobType:   string(domain.JobTypeReplayBuild),
		TitleSlug: titlePkg.DefaultSlug,
		MatchID:   matchID,
	})
	if err != nil || !created {
		t.Fatalf("mise en file: err=%v created=%v", err, created)
	}
	return job
}

// artefactMinimal forge les octets d'un artefact VALIDE pour un match — une trajectoire suffit,
// ce qui est testé est le déclenchement, pas le décodage.
func artefactMinimal(t *testing.T, matchID string) []byte {
	t.Helper()
	blob, err := json.Marshal(transportDoc(matchID))
	if err != nil {
		t.Fatalf("marshal artefact: %v", err)
	}
	return blob
}
