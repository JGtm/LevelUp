package archlint

// match_registry_ddl_types_test.go — LES DEUX DDL DE match_registry DISENT LE MÊME TYPE.
//
// POURQUOI (2026-09-16). `match_registry` est déclarée à DEUX endroits : la migration de
// création title-owned (`internal/games/halo_infinite/migrations/steps_shared_core.go`) et le
// schéma de secours du moteur (`internal/sync/schema.go`). Elles ont divergé sans que rien ne
// le voie : la première annonçait `team_{0,1}_score INTEGER`, la seconde SMALLINT — et les
// bases réelles, créées par la seconde forme, rejetaient à l'INSERT tout match dont un score
// d'équipe dépasse 32 767 (Baptême du feu). Deux matchs perdus POUR TOUS LES JOUEURS.
//
// LA RÈGLE. Pour chaque colonne déclarée DANS LES DEUX DDL, le type doit être identique. Les
// colonnes propres à une seule DDL ne sont pas comparées (les deux schémas n'ont jamais eu la
// même surface : `playlist_name_fr` d'un côté, `season_id` de l'autre) — mais un plancher de
// colonnes communes garde le parseur honnête : s'il ne trouve plus rien, c'est lui qui est
// cassé, pas le code.
//
// Mutation qui doit le faire rougir : remettre `team_0_score SMALLINT` dans l'une des deux.

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// ddlMatchRegistry : les deux sources à comparer (chemin relatif à apps/go-api).
var ddlMatchRegistry = []string{
	"internal/games/halo_infinite/migrations/steps_shared_core.go",
	"internal/sync/schema.go",
}

// colonnesCommunesPlancher : mesuré le 2026-09-16 (26 colonnes communes). Un parseur qui rend
// moins ne compare plus rien d'utile.
const colonnesCommunesPlancher = 20

// reDebutMatchRegistry repère l'ouverture de la DDL, quelle que soit la casse et l'indentation.
var reDebutMatchRegistry = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(IF\s+NOT\s+EXISTS\s+)?match_registry\s*\(`)

// reColonne : `nom TYPE ...` — le type est le deuxième mot, sans sa longueur éventuelle.
var reColonne = regexp.MustCompile(`^([a-z_][a-z0-9_]*)\s+([A-Za-z]+)`)

// typesMatchRegistry parse la DDL de match_registry d'un fichier et rend colonne -> type.
func typesMatchRegistry(t *testing.T, chemin string) map[string]string {
	t.Helper()
	brut, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("lecture de %s : %v", chemin, err)
	}
	texte := string(brut)
	pos := reDebutMatchRegistry.FindStringIndex(texte)
	if pos == nil {
		t.Fatalf("%s : aucune DDL match_registry trouvée — le parseur ou le fichier a changé", chemin)
	}
	corps := texte[pos[1]:]
	types := map[string]string{}
	profondeur := 1
	for _, ligne := range strings.Split(corps, "\n") {
		nette := strings.TrimSpace(ligne)
		profondeur += strings.Count(nette, "(") - strings.Count(nette, ")")
		if profondeur <= 0 {
			break
		}
		if nette == "" || strings.HasPrefix(nette, "--") || strings.HasPrefix(nette, "//") {
			continue
		}
		// Contraintes de table (PRIMARY KEY (...), UNIQUE (...)) : pas des colonnes.
		majuscule := strings.ToUpper(nette)
		if strings.HasPrefix(majuscule, "PRIMARY KEY") || strings.HasPrefix(majuscule, "UNIQUE") ||
			strings.HasPrefix(majuscule, "FOREIGN KEY") || strings.HasPrefix(majuscule, "CONSTRAINT") {
			continue
		}
		m := reColonne.FindStringSubmatch(nette)
		if m == nil {
			continue
		}
		types[m[1]] = strings.ToUpper(m[2])
	}
	if len(types) == 0 {
		t.Fatalf("%s : DDL match_registry parsée sans aucune colonne", chemin)
	}
	return types
}

// TestMatchRegistryDDLTypesIdentiques — LE RATCHET.
func TestMatchRegistryDDLTypesIdentiques(t *testing.T) {
	racine := racineGoAPI(t)
	migrationTypes := typesMatchRegistry(t, filepath.Join(racine, ddlMatchRegistry[0]))
	schemaTypes := typesMatchRegistry(t, filepath.Join(racine, ddlMatchRegistry[1]))

	var communes []string
	for colonne := range migrationTypes {
		if _, ok := schemaTypes[colonne]; ok {
			communes = append(communes, colonne)
		}
	}
	sort.Strings(communes)
	if len(communes) < colonnesCommunesPlancher {
		t.Fatalf("%d colonne(s) commune(s) aux deux DDL, plancher %d (mesure du 2026-09-16) — "+
			"le parseur ne compare plus rien", len(communes), colonnesCommunesPlancher)
	}

	for _, colonne := range communes {
		if migrationTypes[colonne] != schemaTypes[colonne] {
			t.Errorf("match_registry.%s : %s dans %s, %s dans %s — les deux DDL doivent dire le "+
				"MÊME type ; une divergence ne se voit qu'au premier INSERT rejeté, et le match "+
				"est alors perdu pour TOUS les joueurs (plan 2026-09-16, étape 3)",
				colonne, migrationTypes[colonne], ddlMatchRegistry[0],
				schemaTypes[colonne], ddlMatchRegistry[1])
		}
	}
}

// ── Les FIXTURES de test aussi disent le même type ───────────────────────────────────────
//
// POURQUOI (D6, plan robustesse 2026-09-16). Douze fichiers de test recopient la DDL de
// `match_registry`. Deux d'entre eux portaient encore `team_{0,1}_score SMALLINT` : une DDL de
// test recopiée dérive en silence, et un test qui s'exécute sur un schéma que la production
// n'a plus ne prouve rien (leçon consignée : « DDL de test recopiées = dérive indétectable »).
//
// LA RÈGLE. Tout `*_test.go` du module qui contient `CREATE TABLE ... match_registry (` et NE
// porte PAS le marqueur `match_registry-ddl: legacy` déclare, pour les colonnes COMMUNES avec
// la DDL de production, les MÊMES types. Aucun plancher de colonnes : une fixture peut n'en
// déclarer que trois, elles doivent simplement dire vrai.
//
// L'échappatoire est nominative et motivée : un test qui DOIT partir d'une base périmée (celui
// de l'étape d'élargissement) écrit `// match_registry-ddl: legacy — <raison>` au-dessus de sa
// DDL. C'est une exemption lisible, pas une allowlist centralisée qui s'allonge sans qu'on s'en
// aperçoive.
//
// Mutation qui doit le faire rougir : remettre `team_0_score SMALLINT` dans une fixture non
// marquée.

// marqueurDDLLegacy autorise une fixture à porter la DDL d'avant l'élargissement.
const marqueurDDLLegacy = "match_registry-ddl: legacy"

func TestMatchRegistryFixturesDeTestAlignees(t *testing.T) {
	racine := racineGoAPI(t)
	production := typesMatchRegistry(t, filepath.Join(racine, ddlMatchRegistry[0]))

	var examines int
	err := filepath.WalkDir(racine, func(chemin string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		brut, lerr := os.ReadFile(chemin)
		if lerr != nil {
			return lerr
		}
		texte := string(brut)
		if reDebutMatchRegistry.FindStringIndex(texte) == nil {
			return nil
		}
		if strings.Contains(texte, marqueurDDLLegacy) {
			return nil
		}
		examines++
		rel, _ := filepath.Rel(racine, chemin)
		gele := deriveFixturesGelees[filepath.ToSlash(rel)]
		for colonne, typeFixture := range typesFixtureMatchRegistry(texte) {
			typeProd, commune := production[colonne]
			if !commune {
				continue
			}
			tolere, gelee := gele[colonne]
			if typeFixture == typeProd {
				if gelee {
					t.Errorf("%s : match_registry.%s est réaligné sur la production (%s) — retirer "+
						"son entrée de deriveFixturesGelees : ce gel ne doit que rétrécir", rel, colonne, typeProd)
				}
				continue
			}
			if gelee && typeFixture == tolere {
				continue
			}
			t.Errorf("%s : match_registry.%s déclaré %s alors que la production dit %s — une "+
				"fixture qui ment sur le schéma teste un monde qui n'existe pas (marquer la DDL "+
				"avec « %s — <raison> » si elle doit rester périmée)",
				rel, colonne, typeFixture, typeProd, marqueurDDLLegacy)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours du module : %v", err)
	}
	if examines == 0 {
		t.Fatal("aucune fixture de test avec une DDL match_registry trouvée — le parseur ou " +
			"l'arborescence a changé")
	}
	t.Logf("fixtures de test examinées (hors DDL legacy marquées) : %d", examines)
}

// typesFixtureMatchRegistry parse TOUTES les DDL match_registry d'un texte et rend
// colonne -> type. Tolérant à la mise en forme : les fixtures écrivent aussi bien une colonne
// par ligne qu'une DDL entière sur une seule ligne — le parseur des deux DDL de production, lui,
// est ligne à ligne. Rend une map vide quand rien n'est reconnu (une fixture qui ne déclare
// aucune colonne typée n'a rien à contredire).
func typesFixtureMatchRegistry(texte string) map[string]string {
	types := map[string]string{}
	reste := texte
	for {
		pos := reDebutMatchRegistry.FindStringIndex(reste)
		if pos == nil {
			return types
		}
		corps := reste[pos[1]:]
		profondeur := 1
		fin := len(corps)
		for i, r := range corps {
			switch r {
			case '(':
				profondeur++
			case ')':
				profondeur--
			}
			if profondeur == 0 {
				fin = i
				break
			}
		}
		for _, champ := range decouperChamps(corps[:fin]) {
			nette := strings.TrimSpace(champ)
			majuscule := strings.ToUpper(nette)
			if nette == "" || strings.HasPrefix(nette, "--") || strings.HasPrefix(nette, "//") ||
				strings.HasPrefix(majuscule, "PRIMARY KEY") || strings.HasPrefix(majuscule, "UNIQUE") ||
				strings.HasPrefix(majuscule, "FOREIGN KEY") || strings.HasPrefix(majuscule, "CONSTRAINT") {
				continue
			}
			if m := reColonne.FindStringSubmatch(nette); m != nil {
				types[m[1]] = strings.ToUpper(m[2])
			}
		}
		reste = corps[fin:]
	}
}

// decouperChamps découpe le corps d'une DDL sur les virgules de PROFONDEUR 0 (une virgule
// dans `DECIMAL(10, 2)` ne sépare pas deux colonnes), les sauts de ligne faisant aussi office
// de séparateur.
func decouperChamps(corps string) []string {
	var champs []string
	var courant strings.Builder
	profondeur := 0
	for _, r := range corps {
		switch r {
		case '(':
			profondeur++
		case ')':
			profondeur--
		case ',', '\n':
			if profondeur == 0 {
				champs = append(champs, courant.String())
				courant.Reset()
				continue
			}
		}
		courant.WriteRune(r)
	}
	return append(champs, courant.String())
}

// deriveFixturesGelees — DÉRIVE PRÉEXISTANTE, GELÉE LE 2026-09-16, QUI NE PEUT QUE DIMINUER.
//
// Le ratchet ci-dessus, appliqué pour la première fois à toutes les fixtures, a mesuré 45
// divergences de type dans 25 fichiers de test — AUCUNE ne porte sur les scores d'équipe (le
// sujet du lot) : ce sont les horodatages (`TIMESTAMP` vs `TIMESTAMPTZ`, dans les DEUX sens),
// `backfill_completed` (BIGINT vs INTEGER) et `player_count` (INTEGER vs SMALLINT). Les
// réaligner touche la sémantique de fuseau de chaque test concerné : c'est un chantier en soi,
// hors du périmètre de ce lot (consigné en §10 du plan robustesse 2026-09-16).
//
// CE GEL EST UN RATCHET, PAS UNE AMNISTIE : toute divergence NOUVELLE rougit, et une entrée
// dont la fixture a été réalignée rougit AUSSI (« entrée périmée ») — la liste ne peut donc
// que rétrécir. Clé : chemin relatif à apps/go-api (séparateurs `/`) → colonne → type TOLÉRÉ
// dans la fixture.
var deriveFixturesGelees = map[string]map[string]string{
	"internal/api/handlers/media_e2e_realdb_test.go": {
		"end_time_utc":   "TIMESTAMP",
		"start_time_utc": "TIMESTAMP",
	},
	"internal/api/wire/registry_monitoring_freshness_grouped_test.go": {
		"start_time_utc": "TIMESTAMP",
	},
	"internal/games/halo_5/livesync/events_backfill_surfaces_test.go": {
		"start_time_utc": "TIMESTAMP",
	},
	"internal/games/halo_5/livesync/kill_kind_backfill_test.go": {
		"start_time_utc": "TIMESTAMP",
	},
	"internal/migration/steps_shared_rebuild_match_participants_test.go": {
		"end_time_utc":   "TIMESTAMP",
		"player_count":   "INTEGER",
		"start_time_utc": "TIMESTAMP",
	},
	"internal/ops/archive_restore_cgo_test.go": {
		"start_time": "TIMESTAMPTZ",
	},
	"internal/ops/data_quality_cgo_test.go": {
		"backfill_completed": "BIGINT",
	},
	"internal/ops/snapshot_integration_test.go": {
		"start_time": "TIMESTAMPTZ",
	},
	"internal/ops/snapshot_read_shared_integration_test.go": {
		"end_time":     "TIMESTAMPTZ",
		"player_count": "INTEGER",
		"start_time":   "TIMESTAMPTZ",
	},
	"internal/persist/events_completion_persister_test.go": {
		"backfill_completed": "BIGINT",
	},
	"internal/platform/duckdb/prestige/prestige_squad_match_provider_playlist_fr_test.go": {
		"start_time_utc": "TIMESTAMP",
	},
	"internal/platform/duckdb/replay_facts_repo_test.go": {
		"start_time_utc": "TIMESTAMP",
	},
	"internal/platform/duckdb/repo_test.go": {
		"last_updated_at": "TIMESTAMPTZ",
		"start_time":      "TIMESTAMPTZ",
	},
	"internal/sync/art_rebuild_regression_test.go": {
		"end_time_utc":   "TIMESTAMP",
		"player_count":   "INTEGER",
		"start_time_utc": "TIMESTAMP",
	},
	"internal/sync/backfill_integration_test.go": {
		"start_time": "TIMESTAMPTZ",
	},
	"internal/sync/backfill_missing_test.go": {
		"start_time": "TIMESTAMPTZ",
	},
	"internal/sync/backfill_shared_test.go": {
		"end_time":   "TIMESTAMPTZ",
		"start_time": "TIMESTAMPTZ",
	},
	"internal/sync/convergence_backfill_events_integration_test.go": {
		"backfill_completed": "BIGINT",
		"start_time":         "TIMESTAMPTZ",
	},
	"internal/sync/csr_backfill_from_shared_integration_test.go": {
		"start_time": "TIMESTAMPTZ",
	},
	"internal/sync/csr_backfill_integration_test.go": {
		"start_time": "TIMESTAMPTZ",
	},
	"internal/sync/events_replay_test.go": {
		"start_time":     "TIMESTAMPTZ",
		"start_time_utc": "TIMESTAMP",
	},
	"internal/sync/film_retry_policy_test.go": {
		"backfill_completed": "BIGINT",
	},
	"internal/sync/golden_test.go": {
		"start_time":     "TIMESTAMPTZ",
		"start_time_utc": "TIMESTAMP",
	},
	"internal/sync/lusrdb_helpers_test.go": {
		"start_time": "TIMESTAMPTZ",
	},
	"internal/sync/performance_integration_test.go": {
		"start_time": "TIMESTAMPTZ",
	},
	"internal/sync/recompute_after_art_rebuild_test.go": {
		"start_time": "TIMESTAMPTZ",
	},
	"internal/sync/shared_rw_guard_test.go": {
		"backfill_completed": "BIGINT",
	},
	"internal/sync/skill/skill_rating_loaders_test.go": {
		"start_time": "TIMESTAMPTZ",
	},
	"internal/sync/snapshot/snapshot_readiness_integration_test.go": {
		"backfill_completed": "BIGINT",
		"start_time":         "TIMESTAMPTZ",
	},
	"internal/sync/snapshot/snapshot_shared_reader_test.go": {
		"end_time":     "TIMESTAMPTZ",
		"player_count": "INTEGER",
		"start_time":   "TIMESTAMPTZ",
	},
}
