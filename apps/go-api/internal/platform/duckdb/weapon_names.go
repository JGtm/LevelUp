package duckdb

// weapon_names.go — lecture des NOMS d'armes par langue (ADR 0040 D-4/D-5, plan i18n 8.4 et 8.7).
//
// Deux tables, une ligne par (arme, langue) chacune, la langue étant un code de config/locales.toml :
//   - weapon_name_labels (title_slug, weapon_key, lang, name) : noms CURÉS, copie du TOML
//     weapon_names.toml, clés par weapon_key ;
//   - weapon_label_names (weapon_id, lang, name) : noms posés par l'API Halo ou le seed, clés par
//     weapon_id (identifiant du film), repli des ids sans weapon_key (sentinelles, ids inconnus).
//
// UNE SEULE RÈGLE DE REPLI, pour tous les lecteurs de noms d'armes de ce paquet : le SQL ne rend
// que les noms PAR LANGUE, jamais de repli croisé fr/en. mergeWeaponNames fusionne les deux
// sources langue par langue (le nom curé l'emporte sur l'autre dans une même langue), puis la
// langue demandée se choisit par `Text.Pick` (langue demandée, sinon la langue de repli).

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"levelup/go-api/internal/i18n"
)

// weaponNamesChunk borne la taille d'une liste IN (identifiants ou clés d'armes).
const weaponNamesChunk = 500

// mergeWeaponNames fusionne les noms curés d'une arme (par clé) et ceux de son identifiant, langue
// par langue : un nom curé non vide l'emporte sur le nom de l'autre source dans la MÊME langue ;
// une langue que seule l'autre source porte reste servie. Jamais de nom inventé : une langue
// absente des deux reste absente (Pick la replie ensuite sur la langue de repli).
func mergeWeaponNames(curated, byID i18n.Text) i18n.Text {
	out := make(i18n.Text, len(curated)+len(byID))
	for l, v := range byID {
		if v != "" {
			out[l] = v
		}
	}
	for l, v := range curated {
		if v != "" {
			out[l] = v
		}
	}
	return out
}

// weaponTableAvailable vérifie (sans log d'erreur) qu'une table de noms existe dans cette
// metadata : information_schema est toujours présent, donc pas d'ERROR sur table absente.
func weaponTableAvailable(ctx context.Context, meta *DB, table string) bool {
	var n int
	err := meta.QueryRow(ctx,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", table).Scan(&n)
	return err == nil && n >= 1
}

// weaponNameRow est une ligne lue d'une table de noms : clé, code de langue, nom.
type weaponNameRow struct{ key, lang, name string }

// scanWeaponNames exécute `query` (colonnes : clé, langue, nom) et range chaque nom sous la
// langue de son code dans `out`. Une ligne dont le code n'est pas une langue de la liste est
// ignorée, et comptée dans un seul journal par appel.
func scanWeaponNames(ctx context.Context, meta *DB, query string, args []any, table string, out map[string]i18n.Text) {
	rows, err := meta.Query(ctx, query, args...)
	if err != nil {
		slog.Log(ctx, degradedLookupLevel(ctx, err),
			table+": lecture des noms d'armes en échec, armes sans nom de cette table",
			"table", table, "err", err)
		return
	}
	defer rows.Close()
	unknown := 0
	for rows.Next() {
		var r weaponNameRow
		if err := rows.Scan(&r.key, &r.lang, &r.name); err != nil {
			slog.Log(ctx, degradedLookupLevel(ctx, err),
				table+": ligne de nom d'arme illisible ignorée", "table", table, "err", err)
			continue
		}
		loc, ok := i18n.Lookup(r.lang)
		if !ok {
			unknown++
			continue
		}
		t := out[r.key]
		if t == nil {
			t = i18n.Text{}
			out[r.key] = t
		}
		if name := strings.TrimSpace(r.name); name != "" {
			t[loc] = name
		}
	}
	if err := rows.Err(); err != nil {
		slog.Log(ctx, degradedLookupLevel(ctx, err),
			table+": itération interrompue, noms d'armes partiels", "table", table, "err", err, "resolved", len(out))
	}
	if unknown > 0 {
		slog.WarnContext(ctx, table+": lignes d'une langue absente de config/locales.toml ignorées",
			"table", table, "rows", unknown)
	}
}

// loadCuratedWeaponNames lit les noms curés (weapon_name_labels) du titre pour un lot de
// weapon_key. Table absente (metadata non seedée) : map vide, sans bruit.
func loadCuratedWeaponNames(ctx context.Context, meta *DB, titleSlug string, keys []string) map[string]i18n.Text {
	out := map[string]i18n.Text{}
	if meta == nil || len(keys) == 0 {
		return out
	}
	if !weaponTableAvailable(ctx, meta, "weapon_name_labels") {
		slog.DebugContext(ctx, "weapon_name_labels: table absente, noms curés ignorés")
		return out
	}
	for start := 0; start < len(keys); start += weaponNamesChunk {
		chunk := keys[start:min(start+weaponNamesChunk, len(keys))]
		args := append([]any{titleSlug}, ToAnySlice(chunk)...)
		scanWeaponNames(ctx, meta,
			"SELECT weapon_key, lang, name FROM weapon_name_labels WHERE title_slug = ? AND weapon_key IN ("+
				Placeholders(len(chunk))+")",
			args, "weapon_name_labels", out)
	}
	return out
}

// loadWeaponLabelNames lit les noms par langue de weapon_label_names pour un lot de weapon_id,
// rangés sous l'identifiant en décimal. Les identifiants sont injectés en littéraux (cast UBIGINT
// non bindable proprement quand le bit 63 est posé, cf. resolveWeaponMeta).
func loadWeaponLabelNames(ctx context.Context, meta *DB, ids []int64) map[int64]i18n.Text {
	out := map[int64]i18n.Text{}
	if meta == nil || len(ids) == 0 {
		return out
	}
	if !weaponTableAvailable(ctx, meta, "weapon_label_names") {
		slog.DebugContext(ctx, "weapon_label_names: table absente, noms par identifiant ignorés")
		return out
	}
	byKey := map[string]i18n.Text{}
	for start := 0; start < len(ids); start += weaponNamesChunk {
		chunk := ids[start:min(start+weaponNamesChunk, len(ids))]
		parts := make([]string, len(chunk))
		for i, id := range chunk {
			parts[i] = strconv.FormatUint(uint64(id), 10) //nolint:gosec // reinterpretation bit-a-bit, cf. UBigint
		}
		scanWeaponNames(ctx, meta,
			"SELECT CAST(weapon_id AS VARCHAR), lang, name FROM weapon_label_names WHERE weapon_id IN ("+
				strings.Join(parts, ",")+")",
			nil, "weapon_label_names", byKey)
	}
	for k, t := range byKey {
		u, err := strconv.ParseUint(k, 10, 64)
		if err != nil {
			slog.WarnContext(ctx, "weapon names: identifiant d'arme illisible ignoré", "id", k, "err", err)
			continue
		}
		out[int64(u)] = t //nolint:gosec // reinterpretation bit-a-bit, cf. UBigint
	}
	return out
}
