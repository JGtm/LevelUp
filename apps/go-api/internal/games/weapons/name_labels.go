package weapons

// name_labels.go — SOURCE UNIQUE des noms d'armes affiches (V72-06).
//
// Le nom d'affichage d'une arme est desormais keye par weapon_key (identifiant
// canonique stable), lu depuis config/titles/{slug}/mappings/weapon_names.toml et
// seede dans la table metadata `weapon_name_labels` (title_slug, weapon_key,
// name_en, name_fr). Le resolver fait weapon_id -> weapon_ids -> weapon_key ->
// {en, fr} depuis cette table (internal/platform/duckdb/weapon_resolver.go).
//
// Avant : 3 sources se marchaient dessus, keyees par le nom EN brut (weapon_labels,
// name_fr invente du registre, priorite du resolver) — mismatch "FRAG GRENADE" vs
// "Frag Grenade" cote H5. Ce reconcile unifie tout par weapon_key.
//
// Table CREE ICI (pas via une migration versionnee) : c'est un cache derive d'un
// fichier versionne, reconcilie a CHAQUE boot (meme motif que ReconcileRegistry
// — le registre saute tout step deja « done », donc un ajout au fichier n'atteindrait
// jamais une DB prod migree). Referentiel STATIQUE, PK simple, zero writer concurrent,
// hors surface ART #23046. INSERT OR REPLACE pour que les corrections du fichier se
// propagent au boot suivant.

import (
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/migration"
)

// ensureWeaponNameLabelsTable cree la table weapon_name_labels (idempotent). Toujours
// appelee au boot pour qu'elle EXISTE meme si le fichier de traduction manque (le
// resolver la joint ; sans elle, il retomberait sur weapon_labels seul, sans rôle).
func ensureWeaponNameLabelsTable(db *sql.DB) error {
	return migration.ExecScript(db, `
		CREATE TABLE IF NOT EXISTS weapon_name_labels (
			title_slug VARCHAR NOT NULL,
			weapon_key VARCHAR NOT NULL,
			name_en    VARCHAR NOT NULL,
			name_fr    VARCHAR NOT NULL,
			PRIMARY KEY (title_slug, weapon_key)
		);
	`)
}

// ReconcileNameLabels seede weapon_name_labels pour `titleSlug` depuis le
// fichier weapon_names.toml a `tomlPath`. Idempotent (INSERT OR REPLACE). La table est
// TOUJOURS creee (meme si le fichier manque) pour que le resolver puisse la joindre.
// Fichier absent/illisible → table laissee en l'etat (log WARN, best-effort — le
// resolver retombe sur weapon_labels pour le nom, sans casser l'UI). Retourne le
// nombre de lignes ecrites.
func ReconcileNameLabels(db *sql.DB, titleSlug, tomlPath string) (int, error) {
	if db == nil {
		return 0, nil
	}
	if err := ensureWeaponNameLabelsTable(db); err != nil {
		return 0, fmt.Errorf("ensure weapon_name_labels: %w", err)
	}
	set, err := mappings.LoadWeaponNamesFromFile(tomlPath)
	if err != nil {
		slog.WarnContext(migration.BootCtx(), "weapon_name_labels: fichier de traduction absent/illisible, table laissée en l'état",
			"title", titleSlug, "path", tomlPath, "err", err)
		return 0, nil
	}
	n := 0
	for key, nm := range set.Names() {
		if _, err := db.ExecContext(migration.BootCtx(),
			`INSERT OR REPLACE INTO weapon_name_labels (title_slug, weapon_key, name_en, name_fr) VALUES (?, ?, ?, ?)`,
			titleSlug, key, nm.En, nm.Fr); err != nil {
			return n, fmt.Errorf("seed weapon_name_labels %s/%s: %w", titleSlug, key, err)
		}
		n++
	}
	slog.InfoContext(migration.BootCtx(), "weapon_name_labels reconciled", "title", titleSlug, "rows", n)
	return n, nil
}
