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
// **Extension du 2026-08-26 (lot « ouvertures RO hors invariant provider »).** Le
// périmètre couvre désormais les 3 sites qui acquéraient un handle HORS cache /
// hors provider, chacun avec le patron le plus simple qui respecte l'invariant :
//
//  1. `internal/sync/citations_backfill.go` — `RunBackfillCompositeOnlyCitations`
//     ouvrait shared_matches_v2 en `OpenReadOnly` FORCÉ et gardait l'entrée `ro:`
//     pendant TOUTE la boucle de recalcul. Routé en `OpenReadForQuery` borné à la
//     seule requête de tri chrono (acquisition → requête → release) ;
//  2. `internal/ops/media.go` — `IndexMedia` empruntait `LookupCachedDB` avec un
//     FALLBACK `sql.Open` nu. Routé en `OpenReadWriteShared` (c'est un WRITER :
//     DDL + INSERT + CHECKPOINT), qui couvre les deux branches via le cache ;
//  3. `internal/ops/healthcheck.go` — `checkDuckDB` faisait un `sql.Open` avec
//     `?access_mode=read_only`, qui échoue en « different configuration » sur une
//     DB tenue en RW. Routé en `OpenReadForQuery`.
//
// **Portée et limites (à connaître).**
//   - Couvre les 5 fichiers des familles corrigées, PAS tous les appelants
//     d'OpenReadForQuery du dépôt (pas de sweep repo-wide : décision de périmètre du
//     lot). Les autres appelants sont soit courts (une requête, aucune fenêtre
//     d'invalidation utile), soit protégés par le drain lecteurs de `SharedAccess`
//     (`outstandingReads`) — ils restent des découvertes consignées.
//   - Le set d'interdits est PAR FICHIER : socle commun, `alsoForbidden` (ajouts)
//     et `exemptFromBase` (retraits justifiés). `OpenReadForQuery(` est dans le
//     socle parce que l'INSTANTANÉ qu'il rend meurt quand la lecture DURE (boucle
//     de matchs, itération de rows) ; il est au contraire le remède PRESCRIT quand
//     l'acquisition est bornée à une requête unique — d'où les retraits explicites
//     sur `healthcheck.go` (fichier entier) et la ligne de tri chrono de
//     `citations_backfill.go` (allowlist à la ligne près).
//   - Le vrai verrou de la famille citations est le TYPE : `loadPveStats` prend un
//     `*duckdbpkg.RecoveringReader`, donc on ne PEUT PLUS lui passer un `*sql.DB`
//     capturé au début du batch (erreur de compilation). Ce grep est la ceinture ;
//     l'assertion de signature ci-dessous vérifie que les bretelles tiennent.
//   - Reste HORS périmètre et consigné en découverte : les autres `sql.Open("duckdb"`
//     de `internal/ops` (archive, backup, backup_service, diagnose, media_hls,
//     restore, seed*, snapshot_read).
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
	// alsoForbidden : constructs interdits EN PLUS du socle commun, propres à ce
	// fichier. Permet d'interdire une acquisition sur un chemin géré par un
	// provider sans faire rougir un usage légitime ailleurs.
	alsoForbidden map[string]string
	// exemptFromBase : constructs du socle commun qui NE s'appliquent PAS à ce
	// fichier, avec leur justification. Contrepartie soustractive d'alsoForbidden :
	// le socle interdit l'INSTANTANÉ (OpenReadForQuery) parce qu'il meurt sur une
	// lecture longue — mais c'est le remède prescrit quand l'acquisition est bornée
	// à une requête unique. Chaque retrait est nominatif : le reste du socle
	// continue de mordre sur le fichier.
	exemptFromBase map[string]string
}

// sharedReadRecoveryProtectedFiles — fichiers des deux familles corrigées, chemins
// relatifs à la racine du module (apps/go-api).
var sharedReadRecoveryProtectedFiles = []sharedReadRecoveryFile{
	// IndexMedia → shared_matches_v2 (chemin géré par un sharedprovider : la
	// reprise doit être cache-only, cf. read_recovery.go section INVARIANT).
	{
		rel:      "internal/ops/media_associate.go",
		mustCall: "OpenRecoveringReader(",
		alsoForbidden: map[string]string{
			// Interdit ICI et pas dans le socle commun : c'est la forme MODERNE de
			// l'incident 2026-06-03 (l'ouverture RO forcée d'un fichier tenu en RW),
			// et sur un chemin géré par un provider elle peut faire échouer
			// l'OpenReadWrite du swap → StateError. citations_backfill.go en contient
			// deux usages ANTÉRIEURS au lot, qui relèvent de leur propre lot.
			"OpenReadOnly(": "ouverture RO forcée sur un chemin géré par un sharedprovider — " +
				"forme moderne de l'incident 2026-06-03, et fait échouer l'OpenReadWrite du swap",
		},
	},
	// Acquisition du handle shared_pve pour le batch citations, ET (depuis le lot
	// RO-invariant du 2026-08-26) emprunt borné de shared_matches_v2 pour le tri
	// chrono du recalcul composite-only.
	{
		rel:      "internal/sync/citations_backfill.go",
		mustCall: "OpenRecoveringReader(",
		alsoForbidden: map[string]string{
			// Ce fichier tourne dans le CLI (process séparé, provider nil) : le vrai
			// coût d'une ouverture RO forcée y est le VERROU FICHIER cross-process,
			// tenu pendant toute la durée de vie du handle, qui bloque le swapToRW
			// du SERVEUR (« Could not set lock on file »). L'emprunt borné
			// OpenReadForQuery réduit la fenêtre à la seule requête. Les DEUX
			// emprunts metadata (metadata.duckdb n'est géré par aucun provider)
			// restent tolérés via l'allowlist ci-dessous, à la ligne près.
			"OpenReadOnly(": "ouverture RO forcée tenue longtemps : verrou fichier cross-process qui bloque " +
				"le swapToRW du serveur pendant toute la duree de vie du handle — emprunt borne prescrit",
		},
	},
	// loadPveStats : verrouillé par le TYPE (cf. test de signature ci-dessous).
	{rel: "internal/sync/citations.go"},
	// IndexMedia → shared_social (ou player DB en mode legacy) : chemin d'ÉCRITURE
	// (DDL + INSERT + UPDATE + CHECKPOINT), donc handle RW obligatoire. Le socle
	// s'applique en entier : ni bare connect, ni emprunt nu, ni instantané RO.
	{
		rel:      "internal/ops/media.go",
		mustCall: "OpenReadWriteShared(",
		alsoForbidden: map[string]string{
			"OpenReadOnly(": "IndexMedia écrit (DDL + INSERT + CHECKPOINT) : un handle RO y échouerait " +
				"à l'exécution, et l'ouverture RO forcée viole l'invariant du cache mono-process",
		},
	},
	// checkDuckDB : une seule requête COUNT par DB diagnostiquée, sur des chemins
	// que le process peut tenir en RW (pool, sharedprovider, writer de sync).
	{
		rel:      "internal/ops/healthcheck.go",
		mustCall: "OpenReadForQuery(",
		exemptFromBase: map[string]string{
			// Retrait NOMINATIF (2026-08-26) : ici l'instantané n'a pas de fenêtre
			// de mort — acquisition, un COUNT, release. C'est le remède prescrit
			// du lot, l'interdire reviendrait à interdire le correctif. `sql.Open(`
			// et `LookupCachedDB(` restent interdits sur ce fichier.
			"OpenReadForQuery(": "acquisition BORNÉE à une requête unique (COUNT de diagnostic) : " +
				"remède prescrit du lot RO-invariant, pas une lecture longue",
		},
		alsoForbidden: map[string]string{
			// Honnêteté (revue 2026-08-26) : au site d'appel réel (CLI, cache vide),
			// OpenReadForQuery fait la MÊME ouverture RO sur cache miss — on
			// n'interdit pas un mécanisme éradiqué, on impose le canal canonique
			// (emprunt in-process si un handle est tenu, connecteur custom, refcount).
			"OpenReadOnly(": "hors canal canonique — meme ouverture que OpenReadForQuery sur cache miss, " +
				"mais sans l'emprunt in-process ni le connecteur custom : rester sur le canal sanctionne",
		},
	},
}

// sharedReadForbiddenConstructs — socle commun : acquisitions de handle qui
// contournent le lecteur auto-réparant, interdites dans TOUS les fichiers
// couverts. Chaque entrée porte la raison de son interdiction.
var sharedReadForbiddenConstructs = map[string]string{
	"sql.Open(": "ouverture DuckDB directe hors cache/provider — incident 2026-06-03 (médias sans match) " +
		"et violation de l'invariant sharedprovider « unique owner du handle »",
	"OpenReadForQuery(": "instantané *sql.DB : meurt si le propriétaire ferme la handle pendant la lecture",
	"LookupCachedDB(":   "emprunt nu au cache, non possédant et sans reprise — même mort que l'instantané",
}

// forbiddenFor retourne le set effectif de constructs interdits pour un fichier :
// socle commun − exemptions nominatives + interdictions propres au fichier.
func forbiddenFor(f sharedReadRecoveryFile) map[string]string {
	out := make(map[string]string, len(sharedReadForbiddenConstructs)+len(f.alsoForbidden))
	for k, v := range sharedReadForbiddenConstructs {
		if _, exempt := f.exemptFromBase[k]; exempt {
			continue
		}
		out[k] = v
	}
	for k, v := range f.alsoForbidden {
		out[k] = v
	}
	return out
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
	// allowlist(2026-08-26) : les DEUX ouvertures RO de metadata.duckdb du fichier
	// (RunBackfillCitations + RunBackfillCompositeOnlyCitations). metadata n'est
	// géré par AUCUN sharedprovider : il n'y a pas de swap RO→RW à casser, et le
	// handle est possédé (Close apparié), pas emprunté. L'interdiction
	// `OpenReadOnly(` ajoutée au fichier vise shared_matches_v2, pas metadata.
	{
		rel:       "internal/sync/citations_backfill.go",
		construct: "OpenReadOnly(",
		needle:    "metadataDBPath",
		reason:    "ouverture metadata (aucun sharedprovider sur ce chemin), handle possede + Close apparie",
	},
	// allowlist(2026-08-26) : emprunt BORNÉ de shared_matches_v2 pour le tri chrono
	// du recalcul composite-only (sortMatchIDsChronoOnShared). L'instantané que le
	// socle interdit n'a ici aucune fenêtre de mort : acquisition → une requête →
	// release. L'exception est à la LIGNE : une 2e acquisition dans ce fichier
	// serait quand même prise.
	{
		rel:       "internal/sync/citations_backfill.go",
		construct: "OpenReadForQuery(",
		needle:    "e.sharedDBPath",
		reason:    "emprunt borne a la requete de tri chrono (acquisition -> requete -> release)",
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
		forbidden := forbiddenFor(f)

		if f.mustCall != "" && !containsInCode(body, f.mustCall) {
			t.Errorf("%s n'appelle plus %s (hors commentaire) : l'acquisition du handle DuckDB a "+
				"régressé vers un chemin non sanctionné (cf. platform/duckdb/read_recovery.go, "+
				"section INVARIANT)", f.rel, f.mustCall)
		}
		for i, line := range strings.Split(body, "\n") {
			if isGoCommentLine(line) {
				continue
			}
			for construct, why := range forbidden {
				if !strings.Contains(line, construct) || isSharedReadAllowed(f.rel, construct, line) {
					continue
				}
				t.Errorf("%s:%d contient %q — %s. Router par le patron sanctionné du site "+
					"(OpenRecoveringReader + Do pour une lecture longue, OpenReadForQuery borné à "+
					"une requête, OpenReadWriteShared pour un writer), ou justifier via "+
					"sharedReadAllowlist (exception datée, à la ligne près)",
					f.rel, i+1, construct, why)
			}
		}
	}
}

// TestSharedReadExceptionsAreAlive refuse les exceptions devenues obsolètes. Une
// allowlist (ou une exemption de socle) qui ne correspond plus à rien ne protège
// personne : elle ne fait qu'élargir silencieusement la surface tolérée le jour où
// un construct du même nom réapparaît dans le fichier pour une AUTRE raison.
func TestSharedReadExceptionsAreAlive(t *testing.T) {
	root := findRepoRoot(t)

	protectedBody := func(rel string) (string, bool) {
		for _, f := range sharedReadRecoveryProtectedFiles {
			if f.rel == rel {
				return readProtectedFile(t, root, rel), true
			}
		}
		return "", false
	}

	for _, a := range sharedReadAllowlist {
		body, ok := protectedBody(a.rel)
		if !ok {
			t.Errorf("allowlist : %s n'est plus un fichier protégé — entrée à retirer", a.rel)
			continue
		}
		found := false
		for _, line := range strings.Split(body, "\n") {
			if !isGoCommentLine(line) && strings.Contains(line, a.construct) && strings.Contains(line, a.needle) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("allowlist obsolète : aucune ligne de code de %s ne contient à la fois %q et %q "+
				"(raison enregistrée : %q) — retirer l'entrée", a.rel, a.construct, a.needle, a.reason)
		}
	}

	for _, f := range sharedReadRecoveryProtectedFiles {
		if len(f.exemptFromBase) == 0 {
			continue
		}
		body := readProtectedFile(t, root, f.rel)
		for construct, reason := range f.exemptFromBase {
			if _, inBase := sharedReadForbiddenConstructs[construct]; !inBase {
				t.Errorf("exemptFromBase[%q] sur %s ne correspond à aucun construct du socle commun "+
					"(faute de frappe ? l'exemption ne retire rien)", construct, f.rel)
				continue
			}
			if !containsInCode(body, construct) {
				t.Errorf("exemption obsolète : %s n'utilise plus %q (raison enregistrée : %q) — "+
					"retirer l'exemption pour que le socle remorde", f.rel, construct, reason)
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
