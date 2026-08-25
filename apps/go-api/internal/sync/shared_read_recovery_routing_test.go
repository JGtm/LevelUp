// Package sync — shared_read_recovery_routing_test.go : garde-rail anti-régression
// du lot « recovery des lectures shared best-effort » (2026-08-25, durci en revue R1).
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
// re-résolution, au lieu de conserver l'instantané.
//
// **Ce que ce test interdit** (revue R1 — la version initiale se contentait du MOT
// « RecoveringReader », qu'un simple commentaire satisfaisait, et laissait passer
// une ouverture DuckDB directe) :
//
//  1. toute acquisition de handle CONTOURNANTE dans les fichiers couverts :
//     ouverture DuckDB directe (`sql.Open(`), instantané (`OpenReadForQuery(`),
//     emprunt nu au cache (`LookupCachedDB(`). L'ouverture directe est
//     précisément l'incident 2026-06-03 (médias sans match) ;
//  2. la disparition de l'APPEL au construct sanctionné — `mustCall` exige la
//     construction réelle, pas sa mention en commentaire.
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
//     (`outstandingReads`) — ils restent des découvertes consignées.
//   - `OpenReadOnly(` n'est PAS interdit : `citations_backfill.go` en contient deux
//     usages ANTÉRIEURS au lot (metadata, et shared dans
//     RunBackfillCompositeOnlyCitations). Le second vise un chemin géré par le
//     provider et mérite son propre lot — consigné en découverte, pas traité ici.
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

// sharedReadRecoveryFile décrit un fichier protégé et ce qu'il doit APPELER.
type sharedReadRecoveryFile struct {
	rel string
	// mustCall : fragment d'APPEL (avec la parenthèse ouvrante) qui doit être
	// présent. Vide = aucune exigence positive (fichier couvert par le TYPE).
	mustCall string
}

// sharedReadRecoveryProtectedFiles — fichiers des deux familles corrigées, chemins
// relatifs à la racine du module (apps/go-api).
var sharedReadRecoveryProtectedFiles = []sharedReadRecoveryFile{
	// IndexMedia → shared_matches_v2 (chemin géré par un sharedprovider : la
	// reprise doit être cache-only, cf. read_recovery.go section INVARIANT).
	{rel: "internal/ops/media_associate.go", mustCall: "OpenRecoveringReader("},
	// Acquisition du handle shared_pve pour le batch citations.
	{rel: "internal/sync/citations_backfill.go", mustCall: "OpenRecoveringReader("},
	// loadPveStats : verrouillé par le TYPE (cf. test de signature ci-dessous).
	{rel: "internal/sync/citations.go"},
}

// sharedReadForbiddenConstructs — acquisitions de handle qui contournent le
// lecteur auto-réparant. Chaque entrée porte la raison de son interdiction.
var sharedReadForbiddenConstructs = map[string]string{
	"sql.Open(": "ouverture DuckDB directe hors cache/provider — incident 2026-06-03 (médias sans match) " +
		"et violation de l'invariant sharedprovider « unique owner du handle »",
	"OpenReadForQuery(": "instantané *sql.DB : meurt si le propriétaire ferme la handle pendant la lecture",
	"LookupCachedDB(":   "emprunt nu au cache, non possédant et sans reprise — même mort que l'instantané",
}

// sharedReadAllowEntry : exception DATÉE et justifiée, à la LIGNE près. Une
// occurrence n'est tolérée que si `rel` et `construct` correspondent ET que
// `needle` apparaît sur la ligne — ainsi seule l'acquisition visée est exemptée,
// pas tout le fichier (une NOUVELLE occurrence y serait quand même prise).
type sharedReadAllowEntry struct {
	rel       string
	construct string
	needle    string
	reason    string
}

// sharedReadAllowlist : exceptions au routage. Idéalement vide.
var sharedReadAllowlist = []sharedReadAllowEntry{
	// allowlist(2026-08-25) : emprunt du handle metadata déjà ouvert par le pool
	// serveur, ANTÉRIEUR à ce lot et hors de son périmètre (les deux familles
	// traitées sont shared_matches_v2 et shared_pve). metadata.duckdb n'est géré
	// par aucun sharedprovider. L'emprunt reste non possédant : consigné en
	// découverte, à traiter dans un lot dédié aux lectures metadata.
	{
		rel:       "internal/sync/citations_backfill.go",
		construct: "LookupCachedDB(",
		needle:    "metadataDBPath",
		reason:    "emprunt metadata preexistant au lot, hors perimetre",
	},
}

// isGoCommentLine : ligne de commentaire ligne-à-ligne. Le garde-rail juge le
// CODE, pas la prose — sans ça il attrape ses propres explications (« passer par
// X, PAS Y »), ce qui est précisément le travers qu'il combat. Limite assumée :
// les commentaires de bloc /* */ ne sont pas reconnus (le style Go du dépôt les
// n'utilise pas) ; une occurrence qui y serait cachée déclencherait un faux
// positif, à traiter par l'allowlist.
func isGoCommentLine(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "//")
}

// containsInCode cherche needle hors des lignes de commentaire — « exiger
// l'APPEL réel, pas sa mention » (revue R1).
func containsInCode(body, needle string) bool {
	for _, line := range strings.Split(body, "\n") {
		if !isGoCommentLine(line) && strings.Contains(line, needle) {
			return true
		}
	}
	return false
}

// isSharedReadAllowed indique si une occurrence est couverte par une exception.
func isSharedReadAllowed(rel, construct, line string) bool {
	for _, a := range sharedReadAllowlist {
		if a.rel == rel && a.construct == construct && strings.Contains(line, a.needle) {
			return true
		}
	}
	return false
}

// TestSharedBestEffortReadsUseRecoveringReader vérifie que les lectures shared
// best-effort corrigées n'ont pas re-régressé vers une acquisition contournante.
func TestSharedBestEffortReadsUseRecoveringReader(t *testing.T) {
	root := findRepoRoot(t)

	for _, f := range sharedReadRecoveryProtectedFiles {
		body := readProtectedFile(t, root, f.rel)

		if f.mustCall != "" && !containsInCode(body, f.mustCall) {
			t.Errorf("%s n'appelle plus %s (hors commentaire) : la lecture shared best-effort a perdu "+
				"sa reprise sur invalidation (cf. platform/duckdb/read_recovery.go)", f.rel, f.mustCall)
		}
		for i, line := range strings.Split(body, "\n") {
			if isGoCommentLine(line) {
				continue
			}
			for construct, why := range sharedReadForbiddenConstructs {
				if !strings.Contains(line, construct) || isSharedReadAllowed(f.rel, construct, line) {
					continue
				}
				t.Errorf("%s:%d contient %q — %s. Passer par duckdb.OpenRecoveringReader + Do, "+
					"ou justifier via sharedReadAllowlist (exception datée, à la ligne près)",
					f.rel, i+1, construct, why)
			}
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

// TestProviderManagedReadIsCacheOnly verrouille le correctif P0 de la revue R1 au
// SITE d'appel : shared_matches_v2 est géré par un sharedprovider, sa reprise doit
// être déclarée ReopenCacheOnly. Avec ReopenAllowed, une ré-ouverture RO émise dans
// la fenêtre de swap peut faire échouer l'OpenReadWrite du provider (StateError).
// Le contrôle porte sur la LIGNE D'APPEL, pas sur le fichier : une prose du type
// « ReopenCacheOnly, PAS ReopenAllowed » doit rester écrivable sans faire rougir
// le gate (faux positif attrapé au premier run des gates R1).
func TestProviderManagedReadIsCacheOnly(t *testing.T) {
	const rel = "internal/ops/media_associate.go"
	root := findRepoRoot(t)
	body := readProtectedFile(t, root, rel)

	calls := 0
	for i, line := range strings.Split(body, "\n") {
		if isGoCommentLine(line) || !strings.Contains(line, "OpenRecoveringReader(") {
			continue
		}
		calls++
		if !strings.Contains(line, "ReopenCacheOnly") {
			t.Errorf("%s:%d — l'appel doit déclarer ReopenCacheOnly : shared_matches_v2 est géré par "+
				"un sharedprovider, une ouverture RO pendant le swap fait échouer son OpenReadWrite "+
				"(provider → StateError, lectures shared en 503)", rel, i+1)
		}
		if strings.Contains(line, "ReopenAllowed") {
			t.Errorf("%s:%d — ReopenAllowed est interdit sur un chemin géré par un sharedprovider "+
				"(cf. read_recovery.go, section INVARIANT)", rel, i+1)
		}
	}
	if calls == 0 {
		t.Errorf("aucun appel OpenRecoveringReader( trouvé dans %s — garde-rail à mettre à jour", rel)
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
