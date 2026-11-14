// boot_compaction.go — compaction des passes supersédées au démarrage (capacité
// CompactPassesAtStartup, profil desktop).
//
// Un serveur hébergé compacte la nuit, serveur arrêté, par scripts/compact-passes-nuit.sh. Une
// application de bureau n'a ni cette tâche ni ligne de commande : sans compaction au démarrage,
// chaque redécodage de film ajoute une passe que toutes les lectures de vue `_latest` paient
// ensuite. Le démarrage est le seul moment où le processus est écrivain unique des bases partagées
// sans qu'aucune lecture ne les tienne : les migrations viennent de finir, les bases ne sont pas
// encore ouvertes par les fournisseurs.
//
// Le travail est celui de `levelup compact-passes` : migration.CompactSupersededPasses (table
// reconstruite à côté, échange transactionnel vérifié, jamais de DELETE), précédée de la même
// copie du fichier fermé (migration.BackupBeforeUpgrade). Seul l'enchaînement diffère : mesure de
// la jauge d'abord (rien ne se passe sous le seuil), budget de temps, journal structuré, et un
// échec n'empêche jamais le démarrage (ERROR, démarrage normal).
//
// Budget : la coquille attend `ready` au plus 120 s (STARTUP_TIMEOUT) et le contrat d'événements
// n'a pas de phase de maintenance. La compaction s'arrête donc à compactionBootBudget ; le contexte
// annulé fait rouler en arrière la table en cours (chaque table a sa transaction), les tables déjà
// compactées le restent, et la jauge de la fois suivante reprend ce qui reste.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/diskfree"
	"levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"
)

// compactionBootBudget borne la compaction du démarrage : STARTUP_TIMEOUT de la coquille moins
// la marge des migrations, de l'ouverture des bases et du reste de l'initialisation.
const compactionBootBudget = 75 * time.Second

// compactBootDeps réunit ce que compactPassesAtBoot lit et appelle (injectable en test).
type compactBootDeps struct {
	slug       string
	path       string
	backupsDir string
	now        func() time.Time
	// measure rend la jauge de chaque table du registre présente dans la base, en lecture seule, et la
	// version de schéma de la base (entre dans le nom de la copie).
	measure func(ctx context.Context, path, slug string) (jauges []migration.JaugePasses, version int, err error)
	// backup copie la base FERMÉE.
	backup func(ctx context.Context, p migration.BackupParams) error
	// compact ouvre la base en écriture, compacte chaque table du registre, la referme.
	compact func(ctx context.Context, path string) ([]migration.CompactionTable, error)
}

// newCompactBootDeps câble la production pour la base partagée d'un titre.
func newCompactBootDeps(slug, path, backupsDir string) compactBootDeps {
	return compactBootDeps{
		slug:       slug,
		path:       path,
		backupsDir: backupsDir,
		now:        time.Now,
		measure:    measureSharedPasses,
		backup: func(ctx context.Context, p migration.BackupParams) error {
			p.FreeBytes = func(path string) (uint64, error) {
				free, _, err := diskfree.Free(path)
				return free, err
			}
			_, err := migration.BackupBeforeUpgrade(ctx, p)
			return err
		},
		compact: compactSharedPasses,
	}
}

// runCompactBoot est la compaction d'un titre ; variable pour que les tests y substituent un double.
var runCompactBoot = compactPassesAtBoot

// compactPassesAtBootAllTitles compacte la base partagée de chaque titre du registre dont la
// jauge dépasse le seuil, si le profil l'offre. Jamais fatale. Le budget est celui du démarrage
// entier, partagé entre les titres dans l'ordre du registre.
func compactPassesAtBootAllTitles(ctx context.Context, cfg *config.AppConfig, pr *title.PathResolver, reg *title.Registry) {
	if !cfg.Capabilities.CompactPassesAtStartup {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, compactionBootBudget)
	defer cancel()
	for _, d := range reg.All() {
		path := pr.SharedDBPath(d.Slug)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		deps := newCompactBootDeps(d.Slug, path, pr.MigrationBackupsDir())
		if err := runCompactBoot(ctx, deps); err != nil {
			slog.ErrorContext(ctx, "compaction au démarrage échouée, démarrage poursuivi",
				"titleSlug", d.Slug, "path", path, "err", err)
		}
	}
}

// compactPassesAtBoot mesure la base partagée d'un titre et, si une table dépasse
// migration.SeuilPassesParMatch, la copie puis la compacte. Rend l'erreur de la première étape qui
// échoue ; rien n'est compacté sans copie préalable réussie.
func compactPassesAtBoot(ctx context.Context, d compactBootDeps) error {
	jauges, version, err := d.measure(ctx, d.path, d.slug)
	if err != nil {
		return fmt.Errorf("mesure de la jauge des passes: %w", err)
	}
	var tables []string
	for _, j := range jauges {
		if j.AuDessusDuSeuil() {
			tables = append(tables, j.Table)
		}
	}
	if len(tables) == 0 {
		slog.DebugContext(ctx, "compaction au démarrage : aucune table au-dessus du seuil", "titleSlug", d.slug)
		return nil
	}
	debut := d.now()
	slog.InfoContext(ctx, "compaction au démarrage : tables au-dessus du seuil",
		"titleSlug", d.slug, "tables", tables, "seuil", migration.SeuilPassesParMatch)
	if err := d.backup(ctx, migration.BackupParams{
		DBPath: d.path, DestDir: d.backupsDir, Version: version, Now: d.now(),
	}); err != nil {
		return fmt.Errorf("copie préalable, rien n'est compacté: %w", err)
	}
	rapports, err := d.compact(ctx, d.path)
	for _, r := range rapports {
		if r.Status != migration.CompactionCompactee {
			continue
		}
		slog.InfoContext(ctx, "compaction au démarrage : table compactée",
			"titleSlug", d.slug, "table", r.Table, "brutes", r.Raw, "apres", r.After,
			"duree_ms", r.Duration.Milliseconds())
	}
	if err != nil {
		return fmt.Errorf("compaction après %s: %w", d.now().Sub(debut).Round(time.Millisecond), err)
	}
	slog.InfoContext(ctx, "compaction au démarrage terminée",
		"titleSlug", d.slug, "duree_ms", d.now().Sub(debut).Milliseconds())
	return nil
}

// measureSharedPasses lit la jauge de chaque table du registre sur une poignée de lecture, et la
// version de schéma de la base partagée.
func measureSharedPasses(ctx context.Context, path, slug string) ([]migration.JaugePasses, int, error) {
	db, release, err := duckdb.OpenReadForQuery(path)
	if err != nil {
		return nil, 0, err
	}
	defer release()
	st, err := migration.InspectSchema(ctx, db, slug, migration.TargetShared)
	if err != nil {
		return nil, 0, err
	}
	return mesurerTables(ctx, db), st.Version, nil
}

// mesurerTables mesure chaque table du registre ; une mesure en échec est journalisée et la table
// est ignorée : « non mesuré » ne déclenche rien.
func mesurerTables(ctx context.Context, db *sql.DB) []migration.JaugePasses {
	var out []migration.JaugePasses
	for _, table := range migration.CompactableTables() {
		j, presente, err := migration.MesurerPasses(ctx, db, table)
		if err != nil {
			slog.WarnContext(ctx, "compaction au démarrage : mesure d'une table échouée, table ignorée",
				"table", table, "err", err)
			continue
		}
		if presente {
			out = append(out, j)
		}
	}
	return out
}

// compactSharedPasses ouvre la base en écriture, compacte chaque table du registre, la referme.
func compactSharedPasses(ctx context.Context, path string) ([]migration.CompactionTable, error) {
	// Ouverture par le point d'entrée de la base partagée (sync.OpenSharedDB), le même que le moteur de
	// sync : le schéma est déjà monté par les migrations qui précèdent, son contrôle est sans effet.
	handle, err := syncpkg.OpenSharedDB(path)
	if err != nil {
		return nil, fmt.Errorf("ouverture en écriture: %w", err)
	}
	rapports, errCompact := migration.CompactSupersededPasses(ctx, handle.SQLDb(), false)
	if errClose := handle.Close(); errClose != nil {
		slog.ErrorContext(ctx, "compaction au démarrage : fermeture de la base", "path", path, "err", errClose)
	}
	return rapports, errCompact
}
