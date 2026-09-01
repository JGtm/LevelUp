// Package archlint — no_weapon_kills_sql_test.go : garde-rail de NON-RETOUR de la table
// `weapon_kills`.
//
// # CE QU'IL PROTÈGE
//
// Le 2026-09-01, `weapon_kills` et sa vue `v_weapon_kills` ont été SUPPRIMÉES du fichier
// Halo Infinite (`shared_drop_weapon_kills_v1`) : sur ce titre, l'arme d'un kill vient de
// la SOURCE DU DÉGÂT lue dans le film, jamais d'une table. Elles SURVIVENT pour Halo 5 —
// 550 926 lignes `confidence = 'native'` issues de la timeline de son API.
//
// Le risque n'est donc pas qu'on « reparle » de weapon_kills : c'est qu'un NOUVEAU
// chemin SQL la lise ou l'écrive sur une route que Halo Infinite peut emprunter. Là-bas
// elle n'existe plus : la requête échouerait en Catalog Error, et une requête qui joint
// plusieurs tables emporterait TOUT son résultat dans sa chute (l'incident du 25/07
// documenté en tête de Q12 — le scoreboard entier vidé pour une sous-requête d'armes).
//
// # CE QU'IL SCANNE, ET CE QU'IL NE SCANNE PAS
//
// Il cible les FORMES SQL (`FROM`, `JOIN`, `INTO`, `UPDATE`, `DELETE FROM`, `CREATE
// TABLE/VIEW`), jamais le simple littéral : `weapon_kills` apparaît dans une centaine de
// fichiers comme identifiant Go (`WeaponKillsRepo`), clé de statistique de citation
// (`weapon_kills:<id>`) ou nom de capability, toutes choses légitimes et sans rapport.
// Un scan du littéral nu serait ingérable et n'attraperait rien de plus.
//
// Modèle : no_art_patterns_test.go (scan de source, allowlist explicite et justifiée).

package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// weaponKillsSQLRE : une FORME SQL visant la table ou sa vue (les deux schémas confondus).
var weaponKillsSQLRE = regexp.MustCompile(
	`(?i)\b(?:FROM|JOIN|INTO|UPDATE|DELETE\s+FROM|TABLE|VIEW)\s+(?:"?\w+"?\.)?(?:v_)?weapon_kills(?:_v3)?\b`)

// weaponKillsSQLAllowlist : les chemins (relatifs à apps/go-api/) où une forme SQL sur
// `weapon_kills` est LÉGITIME, avec sa raison. Toute entrée ajoutée ici doit être datée et
// justifiée ; une liste qui grossit signale que la table revient par la fenêtre.
//
// Trois familles, et trois seulement :
//
//	(1) le SCHÉMA          création, ALTER, append-only, suppression — le DDL doit pouvoir
//	                       nommer la table qu'il gère ;
//	(2) HALO 5             son producteur natif, son backfill de kill_kind, son adaptateur
//	                       de lecture — le titre qui CONSERVE la table ;
//	(3) les REPLIS PARTAGÉS du code inter-titres : la même fonction sert les deux titres,
//	                       branche « titre sans décodeur de film » comprise. Leur chemin ne
//	                       contient PAS `halo_5` — c'est du code partagé — d'où cette liste
//	                       nominative plutôt qu'un filtre de répertoire.
var weaponKillsSQLAllowlist = map[string]string{
	// (1) schéma
	"internal/migration/steps_shared.go":                                        "DDL : création historique de weapon_kills (registre partagé — Halo 5 en dépend)",
	"internal/migration/steps_shared_append_only_weapon_kills.go":               "DDL : append-only generation_id + vue v_weapon_kills (conservé, Halo 5 le lit)",
	"internal/migration/steps_shared_h5_weapon_kill_kind.go":                    "DDL : colonne kill_kind propre à Halo 5 + recréation de sa vue",
	"internal/migration/helpers_export.go":                                      "commentaires d'exemple des helpers de migration",
	"internal/games/halo_infinite/migrations/steps_shared_core.go":              "DDL : créateur de schéma shared title-owned (la table naît puis est droppée)",
	"internal/games/halo_infinite/migrations/steps_shared_drop_weapon_kills.go": "DDL : LA suppression elle-même (Halo Infinite uniquement, OnlyTitles)",
	// (2) Halo 5
	"internal/games/halo_5/livesync/kill_kind_backfill.go":        "Halo 5 : backfill de kill_kind sur SA table",
	"internal/platform/duckdb/halo5/halo5_match_events_source.go": "Halo 5 : source d'événements adossée à SA table",
	"cmd/h5-sync/main.go":     "CLI Halo 5",
	"cmd/h5-backfill/main.go": "CLI Halo 5",
	// (3) replis partagés — branche « titre sans décodeur de film », c'est-à-dire Halo 5
	"internal/platform/duckdb/weapon_kills_repo.go":      "repo historique : sert Halo 5 (le lecteur Halo Infinite est killsource_weapon_kills_repo.go)",
	"internal/platform/duckdb/queries_match_detail.go":   "repli Q-détail par joueur (branche sans classifier de source)",
	"internal/platform/duckdb/queries_home_citations.go": "repli arme favorite de l'accueil (branche sans classifier)",
	"internal/platform/duckdb/explorer_repo.go":          "repli top armes de la cible (branche sans classifier)",
	"internal/sync/citations.go":                         "repli de la statistique weapon_stat du moteur de citations",
	"internal/api/wire/registry_weapon_coverage.go":      "repli de la page admin de couverture (titre sans décodeur)",
	"internal/sync/writes.go":                            "InsertWeaponKills : écriture du producteur natif Halo 5",
	"internal/persist/shared_persister.go":               "persister partagé : insertion des lignes natives Halo 5",
	// Outils d'ops et de diagnostic — hors flux applicatif, exécution manuelle.
	"internal/ops/seed_demo.go":                  "seed de la base de démo (recrée le schéma legacy en entier)",
	"internal/ops/seed_demo_synthetic_shared.go": "seed synthétique",
	"internal/ops/snapshot_read.go":              "lecture d'un snapshot exporté (schéma figé au moment de l'export)",
	"internal/sync/testutil/fixture.go":          "fixture de test partagée (monte le schéma legacy complet)",
	"cmd/diag_db_health/main.go":                 "diagnostic manuel",
	"cmd/diag_recent_match_sync/main.go":         "diagnostic manuel",
	"cmd/diag_audit_crash/main.go":               "diagnostic manuel",
	"cmd/audit_coverage/main.go":                 "audit manuel de couverture",
}

// TestNoNewWeaponKillsSQL échoue si une forme SQL sur weapon_kills apparaît hors
// allowlist. Les fichiers de test sont exclus : ils forgent leurs propres bases.
func TestNoNewWeaponKillsSQL(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // .../apps/go-api

	var violations []string
	err := filepath.WalkDir(goAPIRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if n := d.Name(); n == "vendor" || n == "node_modules" || n == "data" || n == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(goAPIRoot, path)
		rel = filepath.ToSlash(rel)
		if _, allowed := weaponKillsSQLAllowlist[rel]; allowed {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
				continue // un commentaire qui explique la suppression est légitime
			}
			if weaponKillsSQLRE.MatchString(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+" — "+trimmed)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("forme(s) SQL sur weapon_kills hors allowlist :\n  - %s\n"+
			"→ sur Halo Infinite la table N'EXISTE PLUS (shared_drop_weapon_kills_v1, 2026-09-01) : "+
			"lire la source de dégât (match_kill_events_latest). Si le site sert Halo 5 ou le "+
			"schéma, l'ajouter à weaponKillsSQLAllowlist avec sa raison et sa date.",
			strings.Join(violations, "\n  - "))
	}
}
