// Package duckdb — weapon_resolver.go : PASSAGE PRINCIPAL de la résolution
// d'arme (P4, cf. .ai/PLAN_P4_WEAPON_RESOLUTION.md).
//
// resolveWeaponMeta est l'unique point d'entrée : pour un lot de weapon_id (dans
// la metadata du titre courant), il renvoie le NOM d'affichage + les dimensions
// canoniques du registre (weapon_key / role / family / faction).
//
// POLITIQUE DE NOM = PARITÉ PURE (décision user 2026-06-23 : « genre BR75 ») : le
// nom (label / name_en) vient EXCLUSIVEMENT de weapon_labels (name_fr > name_en,
// exactement comme avant) ; le registre n'apporte QUE les dimensions (weapon_key /
// role / family / faction) et n'influence JAMAIS le nom affiché. Donc zéro
// changement de nom, et aucune arme nouvelle ne « surgit ». label = "" si l'id est
// inconnu de weapon_labels (le caller filtre / fallback comme avant).
//
// Robustesse : si le registre (weapons/weapon_ids) est absent (vieux schéma,
// metadata de test non migrée), on retombe sur une résolution weapon_labels seule
// — best-effort, jamais de panic, parité garantie.
package duckdb

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
)

// weaponResolved — résolution unifiée d'une arme.
type weaponResolved struct {
	label     string // nom d'affichage (parité weapon_labels-first)
	nameEN    string // pour l'URL image (AssetURLAdapter)
	weaponKey string // clé canonique du registre ("hinf_br75") ; "" si inconnu
	role      string // dimension registre (automatic/precision/…) ; "" si inconnu
	class     string // dimension registre — axe manipulation (shoulder/sidearm/heavy/
	// melee/grenade/…) ; "" si inconnu. Peuplé dans la MÊME passe que role
	// (jointure weapons w déjà présente) → +1 colonne, 0 round-trip DB.
	family  string // dimension registre (battle_rifle/…) ; "" si inconnu
	faction string // dimension registre (human/covenant/…) ; "" si inconnu
}

// resolveWeaponMeta résout un lot de weapon_id (titre courant). Map vide si meta
// nil ou ids vide. Best-effort.
func resolveWeaponMeta(ctx context.Context, meta *DB, titleSlug string, weaponIDs []int64) map[int64]weaponResolved {
	out := map[int64]weaponResolved{}
	if meta == nil || len(weaponIDs) == 0 {
		return out
	}
	unique := uniqueInt64s(weaponIDs)
	// Garde silencieux : si le registre n'est pas dans cette metadata (vieux schéma,
	// test non migré), on évite la requête unifiée (qui logguerait une ERROR sur
	// table absente) et on retombe directement sur weapon_labels seul (parité).
	if !weaponRegistryAvailable(ctx, meta) {
		return resolveWeaponLabelsOnly(ctx, meta, unique)
	}
	parts := make([]string, len(unique))
	for i, id := range unique {
		// Contournement driver (uint64 bit63=1) : id_value = littéral décimal string.
		parts[i] = "('" + strconv.FormatUint(uint64(id), 10) + "')" //nolint:gosec
	}
	// titleSlug = identifiant de titre interne (jamais user input) → littéral sûr.
	// label/name_en = weapon_labels SEUL (parité pure) ; le registre (w) ne donne
	// QUE les dimensions. label "" → id inconnu de weapon_labels (caller décide).
	query := "SELECT ids.v," +
		" COALESCE(wl.name_fr, wl.name_en, '') AS label," +
		" COALESCE(wl.name_en, '') AS name_en," +
		" COALESCE(w.weapon_key, '') AS weapon_key," +
		" COALESCE(w.role, '') AS role," +
		" COALESCE(w.class, '') AS class," +
		" COALESCE(w.family_key, '') AS family," +
		" COALESCE(w.faction, '') AS faction" +
		" FROM (VALUES " + strings.Join(parts, ", ") + ") AS ids(v)" +
		" LEFT JOIN weapon_labels wl ON wl.weapon_id = CAST(ids.v AS UBIGINT)" +
		" LEFT JOIN weapon_ids wi ON wi.title_slug = '" + titleSlug + "' AND wi.id_value = ids.v" +
		" LEFT JOIN weapons w ON w.title_slug = wi.title_slug AND w.weapon_key = wi.weapon_key"
	rows, err := meta.Query(ctx, query)
	if err != nil {
		// weaponRegistryAvailable a confirmé la présence des tables du registre : une
		// erreur ici n'est donc PAS un simple « schéma non migré » mais une anomalie de
		// requête (SQL invalide, colonne renommée, conn timeout) — à SIGNALER avant la
		// dégradation best-effort (parité loaders Synthesis/Session), pas à avaler. Le
		// fallback weapon_labels seul (parité nom) reste servi pour ne pas casser l'UI.
		slog.WarnContext(ctx, "weapon resolver: unified registry query failed, falling back to labels-only",
			"title", titleSlug, "weapon_ids", len(unique), "err", err)
		return resolveWeaponLabelsOnly(ctx, meta, unique)
	}
	defer rows.Close()
	for rows.Next() {
		var idStr, label, nameEN, weaponKey, role, class, family, faction string
		if err := rows.Scan(&idStr, &label, &nameEN, &weaponKey, &role, &class, &family, &faction); err != nil {
			continue
		}
		u, perr := strconv.ParseUint(idStr, 10, 64)
		if perr != nil {
			continue
		}
		out[int64(u)] = weaponResolved{
			label: label, nameEN: nameEN, weaponKey: weaponKey,
			role: role, class: class, family: family, faction: faction,
		}
	}
	return out
}

// weaponRegistryAvailable vérifie (sans log d'erreur) que les tables du registre
// existent dans cette metadata. QueryRow sur information_schema (toujours présent)
// → pas d'ERROR loggée si le registre manque.
func weaponRegistryAvailable(ctx context.Context, meta *DB) bool {
	var n int
	err := meta.QueryRow(ctx,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('weapons','weapon_ids')").Scan(&n)
	return err == nil && n >= 2
}

// resolveWeaponLabelsOnly — fallback parité : weapon_labels seul (name_fr>name_en),
// sans dimensions. Utilisé si le registre n'existe pas dans la metadata cible.
func resolveWeaponLabelsOnly(ctx context.Context, meta *DB, uniqueIDs []int64) map[int64]weaponResolved {
	out := map[int64]weaponResolved{}
	if meta == nil || len(uniqueIDs) == 0 {
		return out
	}
	parts := make([]string, len(uniqueIDs))
	for i, id := range uniqueIDs {
		parts[i] = strconv.FormatUint(uint64(id), 10) //nolint:gosec
	}
	query := "SELECT weapon_id," +
		" COALESCE(name_fr, name_en, CAST(weapon_id AS VARCHAR)) AS label," +
		" COALESCE(name_en, '') AS name_en" +
		" FROM weapon_labels WHERE weapon_id IN (" + strings.Join(parts, ",") + ")"
	rows, err := meta.Query(ctx, query)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id UBigint
		var label, nameEN string
		if err := rows.Scan(&id, &label, &nameEN); err == nil && label != "" {
			out[id.Int64()] = weaponResolved{label: label, nameEN: nameEN}
		}
	}
	return out
}
