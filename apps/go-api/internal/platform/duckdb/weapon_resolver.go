// Package duckdb — weapon_resolver.go : PASSAGE PRINCIPAL de la résolution
// d'arme (P4, cf. .ai/PLAN_P4_WEAPON_RESOLUTION.md).
//
// resolveWeaponMeta est l'unique point d'entrée : pour un lot de weapon_id (dans
// la metadata du titre courant), il renvoie le NOM d'affichage + les dimensions
// canoniques du registre (weapon_key / role / family / faction).
//
// POLITIQUE DE NOM = SOURCE UNIQUE keyée par weapon_key (V72-06). Le nom d'affichage
// vient de la table `weapon_name_labels` (title_slug, weapon_key, name_en, name_fr),
// seedée depuis config/titles/{slug}/mappings/weapon_names.toml
// (games/weapons.ReconcileNameLabels). Résolution :
// weapon_id → weapon_ids → weapon_key → {en, fr}. Toutes les variantes brutes d'une
// même arme (« FRAG GRENADE » / « Frag Grenade », skins…) retombent sur UNE seule
// traduction → tue le mismatch keyé par nom EN brut de l'ancien modèle.
//
// Ordre du label : wnl.name_fr > wnl.name_en > wl.name_fr > wl.name_en. Le registre
// `weapons` ne porte PLUS de nom d'affichage (son name_fr « inventé » a été retiré) :
// il ne fournit que les dimensions (weapon_key / role / class / family / faction).
// weapon_labels reste utilisé pour name_en (URL image, AssetURLAdapter) et comme
// repli de nom pour les ids SANS weapon_key (sentinelles 0/1/2 = grenade/mêlée/
// véhicule, ids inconnus). Sur Halo Infinite, weapon_names.toml reprend EXACTEMENT le
// name_fr historique de weapon_labels → résolution byte-INCHANGÉE (parité préservée).
//
// Robustesse (best-effort, jamais de panic) : registre (weapons/weapon_ids) absent
// (vieux schéma, metadata de test non migrée) → résolution weapon_labels seule ;
// weapon_name_labels absente (metadata non encore seedée) → nom depuis weapon_labels,
// dimensions du registre préservées. En prod les 3 tables existent dès le boot.
package duckdb

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
)

// weaponResolved — résolution unifiée d'une arme.
type weaponResolved struct {
	label   string // nom d'affichage FR-first (parité weapon_labels-first, repli EN)
	nameEN  string // pour l'URL image (AssetURLAdapter) — PAS un libellé d'affichage
	labelEN string // nom d'affichage EN-first (repli FR) — V2.1, 2026-08-29, cf. label ci-
	// dessus : MÊME source (weapon_name_labels puis weapon_labels), ordre de priorité
	// inversé. Distinct de nameEN (celui-ci reste réservé à l'URL image, jamais affiché
	// tel quel). Vide exactement quand label est vide (mêmes 4 sources sous-jacentes).
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
	// label = SOURCE UNIQUE keyée par weapon_key (weapon_name_labels via weapon_ids)
	// quand la table existe (toujours vrai en prod, créée au boot par
	// ReconcileNameLabels) ; weapon_labels seul en repli pour le nom (ids sans
	// weapon_key, ou metadata non seedée en test). Le registre `weapons` ne fournit
	// PLUS de nom (dimensions seules). name_en reste weapon_labels (URL image). label ""
	// → id inconnu de toutes les sources (caller décide).
	// labelEN (V2.1, 2026-08-29) : MÊME 4 sources, ordre EN-first — pour qu'un lecteur EN
	// voie le nom EN de l'objet plutôt que le FR servi par label (D2 du plan retours
	// utilisateur). Vide exactement quand label l'est (mêmes sources sous-jacentes).
	labelExpr := "COALESCE(NULLIF(wl.name_fr,''), NULLIF(wl.name_en,''), '')"
	labelENExpr := "COALESCE(NULLIF(wl.name_en,''), NULLIF(wl.name_fr,''), '')"
	nameJoin := ""
	if weaponNameLabelsAvailable(ctx, meta) {
		labelExpr = "COALESCE(NULLIF(wnl.name_fr,''), NULLIF(wnl.name_en,''), NULLIF(wl.name_fr,''), NULLIF(wl.name_en,''), '')"
		labelENExpr = "COALESCE(NULLIF(wnl.name_en,''), NULLIF(wl.name_en,''), NULLIF(wnl.name_fr,''), NULLIF(wl.name_fr,''), '')"
		nameJoin = " LEFT JOIN weapon_name_labels wnl ON wnl.title_slug = wi.title_slug AND wnl.weapon_key = wi.weapon_key"
	}
	query := "SELECT ids.v," +
		" " + labelExpr + " AS label," +
		" " + labelENExpr + " AS label_en," +
		" COALESCE(wl.name_en, '') AS name_en," +
		" COALESCE(w.weapon_key, '') AS weapon_key," +
		" COALESCE(w.role, '') AS role," +
		" COALESCE(w.class, '') AS class," +
		" COALESCE(w.family_key, '') AS family," +
		" COALESCE(w.faction, '') AS faction" +
		" FROM (VALUES " + strings.Join(parts, ", ") + ") AS ids(v)" +
		" LEFT JOIN weapon_labels wl ON wl.weapon_id = CAST(ids.v AS UBIGINT)" +
		" LEFT JOIN weapon_ids wi ON wi.title_slug = '" + titleSlug + "' AND wi.id_value = ids.v" +
		" LEFT JOIN weapons w ON w.title_slug = wi.title_slug AND w.weapon_key = wi.weapon_key" +
		nameJoin
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
		var idStr, label, labelEN, nameEN, weaponKey, role, class, family, faction string
		if err := rows.Scan(&idStr, &label, &labelEN, &nameEN, &weaponKey, &role, &class, &family, &faction); err != nil {
			continue
		}
		u, perr := strconv.ParseUint(idStr, 10, 64)
		if perr != nil {
			continue
		}
		out[int64(u)] = weaponResolved{
			label: label, labelEN: labelEN, nameEN: nameEN, weaponKey: weaponKey,
			role: role, class: class, family: family, faction: faction,
		}
	}
	return out
}

// weaponRegistryAvailable vérifie (sans log d'erreur) que les tables du registre
// (weapons/weapon_ids) existent dans cette metadata. QueryRow sur information_schema
// (toujours présent) → pas d'ERROR loggée si le registre manque.
func weaponRegistryAvailable(ctx context.Context, meta *DB) bool {
	var n int
	err := meta.QueryRow(ctx,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('weapons','weapon_ids')").Scan(&n)
	return err == nil && n >= 2
}

// weaponNameLabelsAvailable vérifie (sans log) la présence de weapon_name_labels, la
// SOURCE UNIQUE des noms keyée par weapon_key (V72-06). Toujours vraie en prod (créée
// au boot par ReconcileNameLabels) ; absente seulement sur une metadata non
// seedée (test) → le resolver sert alors le nom depuis weapon_labels, dims préservées.
func weaponNameLabelsAvailable(ctx context.Context, meta *DB) bool {
	var n int
	err := meta.QueryRow(ctx,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'weapon_name_labels'").Scan(&n)
	return err == nil && n >= 1
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
		" COALESCE(name_en, name_fr, CAST(weapon_id AS VARCHAR)) AS label_en," +
		" COALESCE(name_en, '') AS name_en" +
		" FROM weapon_labels WHERE weapon_id IN (" + strings.Join(parts, ",") + ")"
	rows, err := meta.Query(ctx, query)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id UBigint
		var label, labelEN, nameEN string
		if err := rows.Scan(&id, &label, &labelEN, &nameEN); err == nil && label != "" {
			out[id.Int64()] = weaponResolved{label: label, labelEN: labelEN, nameEN: nameEN}
		}
	}
	return out
}

// offArsenalMeta — dimensions d'une cle de registre HORS ARSENAL (cf.
// killsource_class_repo.go) : sa classe et son libelle d'affichage.
type offArsenalMeta struct {
	class   string
	label   string
	labelEN string // meme libelle, EN-first (repli FR) — V2.1, 2026-08-29, cf. weaponResolved.labelEN
}

// resolveOffArsenalKeys resout un lot de weapon_key et NE GARDE QUE celles qui n'ont
// AUCUN identifiant numerique dans `weapon_ids`.
//
// POURQUOI CE FILTRE EST LE COEUR DE LA FONCTION, et pas une option. Ces cles servent une
// SECONDE voie de comptage des kills (source de degat du film), en parallele de
// l'attribution arme-a-feu qui, elle, part d'un `weapon_id`. Une cle qui porte un id
// numerique est donc visible des DEUX voies : la compter ici la compterait deux fois.
// Le filtre `wi.weapon_key IS NULL` est la garantie STRUCTURELLE que ca n'arrive pas —
// pas une liste de classes ecrite a la main, qui derive au premier ajout de registre.
// Le garde-rail qui tient la propriete cote registre : weapons.TestHorsArsenalHINFSansIdNumerique.
//
// Meme politique de nom que resolveWeaponMeta (source unique keyee par weapon_key) :
// weapon_name_labels FR > EN, vide si la metadata n'est pas seedee. Best-effort, jamais
// de panic : registre absent -> map vide (le lecteur ne remonte alors aucune ligne).
func resolveOffArsenalKeys(ctx context.Context, meta *DB, titleSlug string, keys []string) map[string]offArsenalMeta {
	out := map[string]offArsenalMeta{}
	if meta == nil || len(keys) == 0 || !weaponRegistryAvailable(ctx, meta) {
		return out
	}
	labelExpr := "''"
	labelENExpr := "''"
	nameJoin := ""
	if weaponNameLabelsAvailable(ctx, meta) {
		labelExpr = "COALESCE(NULLIF(wnl.name_fr,''), NULLIF(wnl.name_en,''), '')"
		labelENExpr = "COALESCE(NULLIF(wnl.name_en,''), NULLIF(wnl.name_fr,''), '')"
		nameJoin = " LEFT JOIN weapon_name_labels wnl ON wnl.title_slug = w.title_slug AND wnl.weapon_key = w.weapon_key"
	}
	args := make([]any, 0, len(keys)+1)
	args = append(args, titleSlug)
	for _, k := range keys {
		args = append(args, k)
	}
	query := "SELECT w.weapon_key, COALESCE(w.class, '') AS class, " + labelExpr + " AS label, " + labelENExpr + " AS label_en" +
		" FROM weapons w" +
		" LEFT JOIN weapon_ids wi ON wi.title_slug = w.title_slug AND wi.weapon_key = w.weapon_key" +
		nameJoin +
		" WHERE w.title_slug = ? AND wi.weapon_key IS NULL AND w.weapon_key IN (" + Placeholders(len(keys)) + ")"
	rows, err := meta.Query(ctx, query, args...)
	if err != nil {
		// weaponRegistryAvailable a confirme les tables : une erreur ici est une anomalie
		// de requete, pas un schema non migre. On la SIGNALE avant de degrader.
		slog.WarnContext(ctx, "weapon resolver: off-arsenal key query failed",
			"title", titleSlug, "keys", len(keys), "err", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var key, class, label, labelEN string
		if err := rows.Scan(&key, &class, &label, &labelEN); err != nil {
			continue
		}
		out[key] = offArsenalMeta{class: class, label: label, labelEN: labelEN}
	}
	return out
}

// resolveWeaponKeyLabelsAny résout un lot de weapon_key vers {class, label,
// labelEN} SANS restriction arsenal — contrairement à resolveOffArsenalKeys,
// qui ne garde QUE les clés sans id numérique (anti-double-comptage avec
// weapon_kills, cf. son en-tête). Utilisée par un lecteur qui ne recoupe JAMAIS
// avec weapon_kills — le POC distance par arme (G.3, platform/duckdb/
// kill_distance_repo.go), qui a besoin du libellé de TOUTES les armes
// (arme à feu du registre COMME hors arsenal), pas seulement des secondes.
//
// PAS de jointure weapon_ids ici, et c'est délibéré : une arme à feu peut avoir
// PLUSIEURS id numériques (variantes) → une jointure LEFT JOIN weapon_ids ferait
// un fan-out (N lignes pour 1 weapon_key) exactement le bug que
// resolveOffArsenalKeys évite par son filtre `wi.weapon_key IS NULL`. `weapons`
// est unique par (title_slug, weapon_key) — la sélectionner seule ne fan-out
// jamais.
//
// Même politique de nom que les deux résolveurs sœurs : weapon_name_labels
// FR>EN, vide si la metadata n'est pas seedée. Best-effort, jamais de panic.
func resolveWeaponKeyLabelsAny(ctx context.Context, meta *DB, titleSlug string, keys []string) map[string]offArsenalMeta {
	out := map[string]offArsenalMeta{}
	if meta == nil || len(keys) == 0 || !weaponRegistryAvailable(ctx, meta) {
		return out
	}
	labelExpr := "''"
	labelENExpr := "''"
	nameJoin := ""
	if weaponNameLabelsAvailable(ctx, meta) {
		labelExpr = "COALESCE(NULLIF(wnl.name_fr,''), NULLIF(wnl.name_en,''), '')"
		labelENExpr = "COALESCE(NULLIF(wnl.name_en,''), NULLIF(wnl.name_fr,''), '')"
		nameJoin = " LEFT JOIN weapon_name_labels wnl ON wnl.title_slug = w.title_slug AND wnl.weapon_key = w.weapon_key"
	}
	args := make([]any, 0, len(keys)+1)
	args = append(args, titleSlug)
	for _, k := range keys {
		args = append(args, k)
	}
	query := "SELECT w.weapon_key, COALESCE(w.class, '') AS class, " + labelExpr + " AS label, " + labelENExpr + " AS label_en" +
		" FROM weapons w" +
		nameJoin +
		" WHERE w.title_slug = ? AND w.weapon_key IN (" + Placeholders(len(keys)) + ")"
	rows, err := meta.Query(ctx, query, args...)
	if err != nil {
		slog.WarnContext(ctx, "weapon resolver: any-key query failed",
			"title", titleSlug, "keys", len(keys), "err", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var key, class, label, labelEN string
		if err := rows.Scan(&key, &class, &label, &labelEN); err != nil {
			continue
		}
		out[key] = offArsenalMeta{class: class, label: label, labelEN: labelEN}
	}
	return out
}
