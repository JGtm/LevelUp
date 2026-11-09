package weapons

// name_labels.go — SOURCE UNIQUE des noms d'armes affiches (V72-06).
//
// Le nom d'affichage d'une arme est keye par weapon_key (identifiant canonique stable), lu
// depuis config/titles/{slug}/mappings/weapon_names.toml et seede dans la table metadata
// `weapon_name_labels` (title_slug, weapon_key, lang, name) : UNE LIGNE PAR LANGUE du fichier
// (ADR 0040 D-5, plan i18n 8.4). Le resolver fait weapon_id -> weapon_ids -> weapon_key -> noms par
// langue depuis cette table (internal/platform/duckdb/weapon_resolver.go).
//
// Avant : 3 sources se marchaient dessus, keyees par le nom EN brut (weapon_labels,
// name_fr invente du registre, priorite du resolver) — mismatch "FRAG GRENADE" vs
// "Frag Grenade" cote H5. Ce reconcile unifie tout par weapon_key.
//
// Table CREE ICI (pas via une migration versionnee) : c'est un cache derive d'un
// fichier versionne, reconcilie a CHAQUE boot (meme motif que ReconcileRegistry
// — le registre saute tout step deja « done », donc un ajout au fichier n'atteindrait
// jamais une DB prod migree). Referentiel STATIQUE, PK simple, zero writer concurrent,
// hors surface ART #23645. Ecriture SELECT-puis-UPDATE-ou-INSERT par ligne : une correction
// du fichier se propage au boot suivant, et une langue posee ailleurs (hors fichier) n'est
// jamais effacee.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sort"

	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/migration"
)

// ensureWeaponNameLabelsTable cree weapon_name_labels et les tables de noms par langue
// (idempotent). Toujours appelee au boot pour qu'elle EXISTE meme si le fichier de traduction
// manque (le resolver la joint ; sans elle, il retomberait sur weapon_label_names seul).
func ensureWeaponNameLabelsTable(db *sql.DB) error {
	if err := migration.MigrateWeaponNames(db); err != nil {
		return err
	}
	return migration.ExecScript(db, migration.WeaponTablesDDL())
}

// nameKey est la cle d'une ligne de weapon_name_labels dans la lecture prealable.
type nameKey struct {
	weaponKey string
	lang      string
}

// ReconcileNameLabels seede weapon_name_labels pour `titleSlug` depuis le
// fichier weapon_names.toml a `tomlPath`, une ligne par langue non vide. Idempotent. La table est
// TOUJOURS creee (meme si le fichier manque) pour que le resolver puisse la joindre.
// Fichier absent/illisible → table laissee en l'etat (log WARN, best-effort — le
// resolver retombe sur weapon_label_names pour le nom, sans casser l'UI). Retourne le
// nombre de lignes (arme, langue) du fichier.
func ReconcileNameLabels(db *sql.DB, titleSlug, tomlPath string) (int, error) {
	if db == nil {
		return 0, nil
	}
	ctx := migration.BootCtx()
	if err := ensureWeaponNameLabelsTable(db); err != nil {
		return 0, fmt.Errorf("ensure weapon_name_labels: %w", err)
	}
	set, err := mappings.LoadWeaponNamesFromFile(tomlPath)
	if err != nil {
		slog.WarnContext(ctx, "weapon_name_labels: fichier de traduction absent/illisible, table laissée en l'état",
			"title", titleSlug, "path", tomlPath, "err", err)
		return 0, nil
	}
	existing, err := loadNameLabels(ctx, db, titleSlug)
	if err != nil {
		return 0, err
	}
	n, written := 0, 0
	for key, text := range set.Names() {
		for _, lang := range sortedLangs(text) {
			name := text[lang]
			if name == "" {
				continue
			}
			n++
			changed, err := writeNameLabel(ctx, db, existing, titleSlug, key, string(lang), name)
			if err != nil {
				return n, fmt.Errorf("seed weapon_name_labels %s/%s/%s: %w", titleSlug, key, lang, err)
			}
			if changed {
				written++
			}
		}
	}
	slog.InfoContext(ctx, "weapon_name_labels reconciled", "title", titleSlug, "rows", n, "written", written)
	return n, nil
}

// sortedLangs rend les langues d'un texte dans l'ordre alphabetique (ecriture deterministe).
func sortedLangs(t i18n.Text) []i18n.Locale {
	out := make([]i18n.Locale, 0, len(t))
	for l := range t {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// loadNameLabels lit les lignes deja posees pour le titre : (arme, langue) -> nom.
func loadNameLabels(ctx context.Context, db *sql.DB, titleSlug string) (map[nameKey]string, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT weapon_key, lang, name FROM weapon_name_labels WHERE title_slug = ?`, titleSlug)
	if err != nil {
		return nil, fmt.Errorf("read weapon_name_labels %s: %w", titleSlug, err)
	}
	defer rows.Close()
	out := map[nameKey]string{}
	for rows.Next() {
		var k nameKey
		var name string
		if err := rows.Scan(&k.weaponKey, &k.lang, &name); err != nil {
			return nil, fmt.Errorf("scan weapon_name_labels %s: %w", titleSlug, err)
		}
		out[k] = name
	}
	return out, rows.Err()
}

// writeNameLabel pose une ligne (arme, langue) : inchangee si elle porte deja ce nom, UPDATE si
// elle porte un autre nom, INSERT sinon. Rend true quand une ecriture a eu lieu.
func writeNameLabel(ctx context.Context, db *sql.DB, existing map[nameKey]string, slug, key, lang, name string) (bool, error) {
	cur, ok := existing[nameKey{key, lang}]
	switch {
	case ok && cur == name:
		return false, nil
	case ok:
		_, err := db.ExecContext(ctx,
			`UPDATE weapon_name_labels SET name = ? WHERE title_slug = ? AND weapon_key = ? AND lang = ?`,
			name, slug, key, lang)
		return err == nil, err
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO weapon_name_labels (title_slug, weapon_key, lang, name) VALUES (?, ?, ?, ?)`,
		slug, key, lang, name)
	return err == nil, err
}
