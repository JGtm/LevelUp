// Package sync — shared_read_recovery_routing_test.go : garde-rail anti-régression
// du lot « recovery des lectures shared best-effort » (2026-08-25).
//
// **Contexte.** `duckdb.OpenReadForQuery(path)` rend un INSTANTANÉ du `*sql.DB`.
// Quand il l'a EMPRUNTÉ au cache process (`LookupCachedDB`), l'emprunt est NON
// POSSÉDANT : le propriétaire peut fermer la handle sous les pieds du lecteur —
// B-swap RO→RW côté shared_matches_v2 (ADR 0016), fin du post-sync d'un autre
// joueur côté shared_pve (`RunPostSync`, PostSyncParallelism = 0). Le lecteur voit
// alors « sql: database is closed » EN COURS de requête. Deux familles mesurées en
// prod en août 2026 : ~67 ERROR `ops.IndexMedia` et 372 WARN `pve_stats`.
//
// **Le correctif** : ces chemins gardent le CHEMIN (`duckdb.RecoveringReader`,
// platform/duckdb/read_recovery.go) et rejouent la lecture UNE fois après
// ré-ouverture, au lieu de conserver l'instantané. Ce test échoue si un de ces
// fichiers reprend un `OpenReadForQuery` nu — la forme exacte de la régression.
//
// **Sans build tag `integration`** (comme no_art_patterns_test.go) : scan statique
// PUR — aucune connexion DuckDB, aucun CGO. Il DOIT tourner dans le gate par défaut
// `go test ./...`, sinon il ne protège rien.
//
// **Portée et limites (à connaître).**
//   - Couvre les 3 fichiers des DEUX familles corrigées, PAS tous les appelants
//     d'OpenReadForQuery du dépôt (pas de sweep repo-wide : décision de périmètre du
//     lot). Les autres appelants sont soit courts (une requête, aucune fenêtre
//     d'invalidation utile), soit protégés par le drain lecteurs de `SharedAccess`
//     (`outstandingReads`) — ils restent des découvertes consignées, pas des trous
//     couverts ici.
//   - Le vrai verrou de la famille citations est le TYPE : `loadPveStats` prend un
//     `*duckdbpkg.RecoveringReader`, donc on ne PEUT PLUS lui passer un `*sql.DB`
//     capturé au début du batch (erreur de compilation). Ce grep est la ceinture ;
//     l'assertion de signature ci-dessous vérifie que les bretelles tiennent.
package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sharedReadRecoveryProtectedFiles — fichiers des deux familles corrigées, chemins
// relatifs à la racine du module (apps/go-api).
var sharedReadRecoveryProtectedFiles = []string{
	"internal/ops/media_associate.go",     // IndexMedia → shared_matches_v2 (B-swap)
	"internal/sync/citations.go",          // loadPveStats → shared_pve
	"internal/sync/citations_backfill.go", // acquisition du handle shared_pve
}

// sharedReadRecoveryAllowlist — exceptions DATÉES et justifiées, idéalement vide.
// Clé = chemin relatif protégé ; valeur = justification. Une entrée neutralise le
// contrôle « pas d'OpenReadForQuery » pour ce fichier : ne l'ajouter qu'avec une
// raison écrite et une date, jamais pour faire passer un gate.
var sharedReadRecoveryAllowlist = map[string]string{}

// TestSharedBestEffortReadsUseRecoveringReader vérifie que les lectures shared
// best-effort corrigées n'ont pas re-régressé vers un instantané `OpenReadForQuery`.
func TestSharedBestEffortReadsUseRecoveringReader(t *testing.T) {
	root := findRepoRoot(t)

	for _, rel := range sharedReadRecoveryProtectedFiles {
		body := readProtectedFile(t, root, rel)

		if !strings.Contains(body, "RecoveringReader") {
			t.Errorf("%s ne référence plus RecoveringReader : la lecture shared best-effort "+
				"a perdu sa reprise sur invalidation (cf. platform/duckdb/read_recovery.go)", rel)
		}
		if reason, allowed := sharedReadRecoveryAllowlist[rel]; allowed {
			t.Logf("%s : contrôle OpenReadForQuery neutralisé (allowlist) — %s", rel, reason)
			continue
		}
		if strings.Contains(body, "OpenReadForQuery(") {
			t.Errorf("%s appelle OpenReadForQuery : l'instantané rendu meurt si le handle est "+
				"fermé pendant la requête (B-swap RO→RW, post-sync concurrent). Passer par "+
				"duckdb.OpenRecoveringReader + Do, ou justifier via sharedReadRecoveryAllowlist", rel)
		}
	}
}

// TestLoadPveStatsSignatureIsRecoveryTyped ferme PAR LE TYPE le vecteur qu'aucun grep
// ne voit : un `*sql.DB` shared_pve re-capturé pour toute la durée du batch citations.
func TestLoadPveStatsSignatureIsRecoveryTyped(t *testing.T) {
	root := findRepoRoot(t)
	body := readProtectedFile(t, root, "internal/sync/citations.go")

	const marker = "func loadPveStats("
	idx := strings.Index(body, marker)
	if idx < 0 {
		t.Fatalf("loadPveStats introuvable dans citations.go (renommée ? mettre ce garde-rail à jour)")
	}
	signature := body[idx:]
	if end := strings.Index(signature, ")"); end >= 0 {
		signature = signature[:end]
	}
	if !strings.Contains(signature, "*duckdbpkg.RecoveringReader") {
		t.Errorf("signature de loadPveStats = %q : elle doit prendre un *duckdbpkg.RecoveringReader "+
			"(un *sql.DB capturé au début du batch meurt quand un post-sync concurrent rend shared_pve)",
			strings.Join(strings.Fields(signature), " "))
	}
}

// readProtectedFile lit un fichier protégé, en échouant explicitement s'il a été
// déplacé ou supprimé (le garde-rail doit alors être mis à jour, pas contourné).
func readProtectedFile(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("fichier protégé %s illisible (déplacé ? mettre ce garde-rail à jour): %v", rel, err)
	}
	return string(b)
}
