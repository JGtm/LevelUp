// Package duckdb — weapon_resolver.go : PASSAGE PRINCIPAL de la résolution
// d'arme (P4, cf. .ai/archive/V7/PLAN_P4_WEAPON_RESOLUTION.md).
//
// resolveWeaponMeta est l'unique point d'entrée : pour un lot de weapon_id (dans
// la metadata du titre courant), il renvoie le NOM d'affichage + les dimensions
// canoniques du registre (weapon_key / role / family / faction).
//
// POLITIQUE DE NOM = SOURCE UNIQUE keyée par weapon_key (V72-06), PAR LANGUE (plan i18n 8.4/8.7).
// Le nom d'affichage vient de la table `weapon_name_labels` (title_slug, weapon_key, lang, name),
// seedée depuis config/titles/{slug}/mappings/weapon_names.toml
// (games/weapons.ReconcileNameLabels). Résolution :
// weapon_id → weapon_ids → weapon_key → noms par langue. Toutes les variantes brutes d'une
// même arme (« FRAG GRENADE » / « Frag Grenade », skins…) retombent sur UNE seule
// traduction → tue le mismatch keyé par nom EN brut de l'ancien modèle.
//
// Règle de repli UNIQUE (weapon_names.go) : le SQL ne rend que des noms par langue ; le nom curé de
// weapon_name_labels l'emporte, langue par langue, sur celui de weapon_label_names (clé weapon_id,
// posé par l'API Halo ou le seed) ; la langue demandée se choisit ensuite par `Text.Pick` (langue
// demandée, sinon la langue de repli), jamais par un COALESCE croisé fr/en en SQL. Le registre
// `weapons` ne porte PLUS de nom d'affichage (son name_fr « inventé » a été retiré) : il ne fournit
// que les dimensions (weapon_key / role / class / family / faction). weapon_label_names reste aussi
// la source du nom canonique anglais (URL image, AssetURLAdapter) et le repli de nom des ids SANS
// weapon_key (sentinelles 0/1/2 = grenade/mêlée/véhicule, ids inconnus).
//
// Robustesse (best-effort, jamais de panic) : registre (weapons/weapon_ids) absent
// (vieux schéma, metadata de test non migrée) → résolution par weapon_label_names seule ;
// weapon_name_labels absente (metadata non encore seedée) → nom depuis weapon_label_names,
// dimensions du registre préservées. En prod les tables existent dès le boot.
package duckdb

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"levelup/go-api/internal/i18n"
)

// weaponResolved — résolution unifiée d'une arme.
type weaponResolved struct {
	// label : nom d'affichage par langue (noms curés puis noms par identifiant, fusionnés langue par
	// langue par mergeWeaponNames). Vide exactement quand aucune source n'a de nom ; une langue
	// sans nom se replie à la lecture (Pick), pas ici.
	label i18n.Text
	// canonicalName : nom anglais canonique du jeu, clé de l'URL image (AssetURLAdapter) — PAS
	// un libellé d'affichage, jamais affiché tel quel.
	canonicalName string
	weaponKey     string // clé canonique du registre ("hinf_br75") ; "" si inconnu
	role          string // dimension registre (automatic/precision/…) ; "" si inconnu
	class         string // dimension registre — axe manipulation (shoulder/sidearm/heavy/
	// melee/grenade/…) ; "" si inconnu. Peuplé dans la MÊME passe que role
	// (jointure weapons w déjà présente) → +1 colonne, 0 round-trip DB.
	family  string // dimension registre (battle_rifle/…) ; "" si inconnu
	faction string // dimension registre (human/covenant/…) ; "" si inconnu
}

// displayLabel rend le nom d'affichage dans la locale de requête : `label.Pick` (langue demandée,
// sinon langue de repli) — la même règle que tout autre texte de jeu.
//
// Sert les surfaces qui publient UN SEUL nom par arme (Match view : kills par arme du
// viewer, arme favorite du scoreboard, sources de dégât du film) — avant le 2026-09-17
// elles servaient label (FR) quelle que soit la locale, et « Tourelle LMG du Falcon » /
// « Apparition » sortaient sous UI EN. Même doctrine que lookupMedalMeta (GH-5b) : la
// locale de requête (ctxkeys.Locale ← X-LevelUp-Locale) décide côté serveur.
func (w weaponResolved) displayLabel(locale string) string {
	return w.label.Pick(i18n.Parse(locale))
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
	// table absente) et on retombe directement sur weapon_label_names seul (parité).
	if !weaponRegistryAvailable(ctx, meta) {
		return resolveWeaponLabelsOnly(ctx, meta, unique)
	}
	parts := make([]string, len(unique))
	for i, id := range unique {
		// Contournement driver (uint64 bit63=1) : id_value = littéral décimal string.
		parts[i] = "('" + strconv.FormatUint(uint64(id), 10) + "')" //nolint:gosec
	}
	// titleSlug = identifiant de titre interne (jamais user input) → littéral sûr.
	// Dimensions seules ici ; les noms viennent de deux lecteurs par langue (weapon_names.go).
	query := "SELECT ids.v," +
		" COALESCE(w.weapon_key, '') AS weapon_key," +
		" COALESCE(w.role, '') AS role," +
		" COALESCE(w.class, '') AS class," +
		" COALESCE(w.family_key, '') AS family," +
		" COALESCE(w.faction, '') AS faction" +
		" FROM (VALUES " + strings.Join(parts, ", ") + ") AS ids(v)" +
		" LEFT JOIN weapon_ids wi ON wi.title_slug = '" + titleSlug + "' AND wi.id_value = ids.v" +
		" LEFT JOIN weapons w ON w.title_slug = wi.title_slug AND w.weapon_key = wi.weapon_key"
	rows, err := meta.Query(ctx, query)
	if err != nil {
		// weaponRegistryAvailable a confirmé la présence des tables du registre : une
		// erreur ici n'est donc PAS un simple « schéma non migré » mais une anomalie de
		// requête (SQL invalide, colonne renommée, conn timeout) — à SIGNALER avant la
		// dégradation best-effort (parité loaders Synthesis/Session), pas à avaler. Le
		// fallback weapon_label_names seul (parité nom) reste servi pour ne pas casser l'UI.
		slog.Log(ctx, degradedLookupLevel(ctx, err), "weapon resolver: unified registry query failed, falling back to labels-only",
			"title", titleSlug, "weapon_ids", len(unique), "err", err)
		return resolveWeaponLabelsOnly(ctx, meta, unique)
	}
	defer rows.Close()
	for rows.Next() {
		var idStr, weaponKey, role, class, family, faction string
		if err := rows.Scan(&idStr, &weaponKey, &role, &class, &family, &faction); err != nil {
			slog.Log(ctx, degradedLookupLevel(ctx, err),
				"weapon resolver: ligne du registre illisible ignorée", "title", titleSlug, "err", err)
			continue
		}
		u, perr := strconv.ParseUint(idStr, 10, 64)
		if perr != nil {
			slog.WarnContext(ctx, "weapon resolver: identifiant d'arme illisible ignoré",
				"title", titleSlug, "id", idStr, "err", perr)
			continue
		}
		out[int64(u)] = weaponResolved{weaponKey: weaponKey, role: role, class: class, family: family, faction: faction}
	}
	if err := rows.Err(); err != nil {
		slog.Log(ctx, degradedLookupLevel(ctx, err),
			"weapon resolver: itération interrompue, armes partielles", "title", titleSlug,
			"resolved", len(out), "err", err)
	}
	attachWeaponNames(ctx, meta, titleSlug, unique, out)
	return out
}

// attachWeaponNames pose, sur chaque arme résolue, ses noms par langue (nom curé de sa weapon_key,
// sinon nom de son identifiant, langue par langue) et son nom canonique anglais.
func attachWeaponNames(ctx context.Context, meta *DB, titleSlug string, ids []int64, out map[int64]weaponResolved) {
	keys := make([]string, 0, len(out))
	for _, w := range out {
		if w.weaponKey != "" {
			keys = append(keys, w.weaponKey)
		}
	}
	curated := loadCuratedWeaponNames(ctx, meta, titleSlug, keys)
	byID := loadWeaponLabelNames(ctx, meta, ids)
	for id, w := range out {
		w.label = mergeWeaponNames(curated[w.weaponKey], byID[id])
		w.canonicalName = byID[id][i18n.Fallback]
		out[id] = w
	}
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

// resolveWeaponLabelsOnly — fallback parité : weapon_label_names seul, sans dimensions. Utilisé
// si le registre n'existe pas dans la metadata cible. N'inscrit que les armes qui portent au
// moins un nom.
func resolveWeaponLabelsOnly(ctx context.Context, meta *DB, uniqueIDs []int64) map[int64]weaponResolved {
	out := map[int64]weaponResolved{}
	if meta == nil || len(uniqueIDs) == 0 {
		return out
	}
	for id, names := range loadWeaponLabelNames(ctx, meta, uniqueIDs) {
		if names.Empty() {
			continue
		}
		out[id] = weaponResolved{label: names, canonicalName: names[i18n.Fallback]}
	}
	return out
}

// offArsenalMeta — dimensions minimales d'une cle de registre : sa classe et son libelle
// d'affichage par langue. Le nom rappelle son premier usage (les cles hors arsenal) ; elle sert
// desormais tout lecteur qui n'a besoin que de ces champs.
type offArsenalMeta struct {
	class string
	label i18n.Text // memes noms par langue que weaponResolved.label (noms cures seuls)
}

// resolveWeaponKeyLabelsAny résout un lot de weapon_key vers {class, label} SANS
// restriction d'arsenal. Utilisée par le POC distance par arme (G.3,
// platform/duckdb/kill_distance_repo.go), qui a besoin du libellé de TOUTES les armes —
// arme à feu du registre COMME objet hors arsenal.
//
// ⚠ SA SŒUR `resolveOffArsenalKeys` A ÉTÉ SUPPRIMÉE le 2026-09-01. Elle ne gardait que les
// clés SANS identifiant numérique, pour départager DEUX voies de comptage concurrentes
// (arme à feu depuis `weapon_kills`, hors arsenal depuis la source de dégât). Les deux ont
// FUSIONNÉ (décision D11) : il n'y a plus qu'une voie, donc plus rien à départager, et la
// fonction n'avait plus d'appelant.
//
// PAS de jointure weapon_ids ici, et c'est délibéré : une arme à feu peut avoir PLUSIEURS
// id numériques (variantes) → un LEFT JOIN nu ferait un fan-out (N lignes pour 1
// weapon_key). `weapons` est unique par (title_slug, weapon_key) — la sélectionner seule ne
// fan-out jamais. C'est la même raison qui fait employer `MIN(wi.id_value)` à
// `resolveWeaponKeyDimensions`, qui a besoin de l'id.
//
// Même politique de nom que les deux résolveurs sœurs : weapon_name_labels, par langue (repli à
// la lecture par Text.Pick), vide si la metadata n'est pas seedée. Best-effort, jamais de panic.
func resolveWeaponKeyLabelsAny(ctx context.Context, meta *DB, titleSlug string, keys []string) map[string]offArsenalMeta {
	out := map[string]offArsenalMeta{}
	if meta == nil || len(keys) == 0 || !weaponRegistryAvailable(ctx, meta) {
		return out
	}
	args := make([]any, 0, len(keys)+1)
	args = append(args, titleSlug)
	for _, k := range keys {
		args = append(args, k)
	}
	query := "SELECT w.weapon_key, COALESCE(w.class, '') AS class" +
		" FROM weapons w" +
		" WHERE w.title_slug = ? AND w.weapon_key IN (" + Placeholders(len(keys)) + ")"
	rows, err := meta.Query(ctx, query, args...)
	if err != nil {
		slog.WarnContext(ctx, "weapon resolver: any-key query failed",
			"title", titleSlug, "keys", len(keys), "err", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var key, class string
		if err := rows.Scan(&key, &class); err != nil {
			slog.Log(ctx, degradedLookupLevel(ctx, err),
				"weapon resolver: ligne de clé illisible ignorée", "title", titleSlug, "err", err)
			continue
		}
		out[key] = offArsenalMeta{class: class}
	}
	if err := rows.Err(); err != nil {
		slog.Log(ctx, degradedLookupLevel(ctx, err),
			"weapon resolver: itération interrompue, clés partielles", "title", titleSlug, "err", err)
	}
	for key, names := range loadCuratedWeaponNames(ctx, meta, titleSlug, keys) {
		if m, ok := out[key]; ok {
			m.label = names
			out[key] = m
		}
	}
	return out
}

// weaponKeyResolved — les dimensions d'une cle de registre, plus son identifiant numerique
// CANONIQUE quand elle en porte un.
//
// POURQUOI L'IDENTIFIANT NUMERIQUE EST ICI. La source de degat rend une CLE, jamais un id.
// Or `port.WeaponKillRow` en porte un, et deux consommateurs s'en servent : l'URL de
// l'image d'arme (`AssetURLAdapter.WeaponImageURL`) et rien d'autre depuis que les
// agregats sont keyes par `weapon_key`. Resoudre la cle vers son id canonique
// (`MIN(id_value)`, deterministe) rend donc l'image SANS demander au lecteur de connaitre
// le film. Une cle SANS entree dans `weapon_ids` a `numericID` = 0 (garde-rail
// weapons.TestHorsArsenalHINFSansIdNumerique : la liste EXACTE de ces cles).
//
// « SANS ID » N EST PLUS « HORS ARSENAL » depuis le 2026-09-24 (retours du rejeu, lot M6.4) : la
// bobine a fusion UNSC, classe `environmental`, porte les identifiants de l objet que le joueur
// tient. Un lecteur qui doit ecarter les objets (arme favorite, top armes) le fait par la CLASSE,
// regle nommee `domain.IsFavoriteWeaponCandidate` — jamais par `numericID == 0`.
type weaponKeyResolved struct {
	class     string
	role      string
	family    string
	label     i18n.Text // noms cures par langue ; le repli se fait a la lecture (Text.Pick)
	numericID int64     // 0 = cle sans identifiant numerique (cf. l en-tete : pas un test d arsenal)
}

// displayLabel : même contrat que weaponResolved.displayLabel, pour les lecteurs keyés par
// weapon_key (sources de dégât du film).
func (w weaponKeyResolved) displayLabel(locale string) string {
	return w.label.Pick(i18n.Parse(locale))
}

// resolveWeaponKeyDimensions resout un lot de weapon_key vers leurs dimensions completes.
//
// SANS le filtre `wi.weapon_key IS NULL` de l ex-`resolveOffArsenalKeys` (supprimee le
// 2026-09-01) : ce filtre servait a
// departager DEUX voies de comptage concurrentes (arme a feu contre hors arsenal) ; il n'y
// en a plus qu'une (D11 du plan du 2026-09-01), donc plus rien a departager. L'agregat
// `MIN(wi.id_value)` remplace la jointure fan-out : une arme peut porter plusieurs ids
// (variantes, skins) et un LEFT JOIN nu rendrait N lignes pour une cle.
//
// Best-effort, jamais de panic : registre absent -> map vide (le lecteur ne remonte alors
// aucune ligne, et le sunburst retombe sur « Non attribue »).
func resolveWeaponKeyDimensions(ctx context.Context, meta *DB, titleSlug string, keys []string) map[string]weaponKeyResolved {
	out := map[string]weaponKeyResolved{}
	if meta == nil || len(keys) == 0 || !weaponRegistryAvailable(ctx, meta) {
		return out
	}
	args := make([]any, 0, len(keys)+1)
	args = append(args, titleSlug)
	for _, k := range keys {
		args = append(args, k)
	}
	query := "SELECT w.weapon_key, COALESCE(MIN(w.class), '') AS class," +
		" COALESCE(MIN(w.role), '') AS role, COALESCE(MIN(w.family_key), '') AS family," +
		" COALESCE(MIN(wi.id_value), '') AS id_value" +
		" FROM weapons w" +
		" LEFT JOIN weapon_ids wi ON wi.title_slug = w.title_slug AND wi.weapon_key = w.weapon_key" +
		" WHERE w.title_slug = ? AND w.weapon_key IN (" + Placeholders(len(keys)) + ")" +
		" GROUP BY w.weapon_key"
	rows, err := meta.Query(ctx, query, args...)
	if err != nil {
		// weaponRegistryAvailable a confirme les tables : une erreur ici est une anomalie
		// de requete, pas un schema non migre. On la SIGNALE avant de degrader.
		slog.WarnContext(ctx, "weapon resolver: key dimensions query failed",
			"title", titleSlug, "keys", len(keys), "err", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var key, class, role, family, idValue string
		if err := rows.Scan(&key, &class, &role, &family, &idValue); err != nil {
			slog.Log(ctx, degradedLookupLevel(ctx, err),
				"weapon resolver: ligne de dimensions illisible ignorée", "title", titleSlug, "err", err)
			continue
		}
		r := weaponKeyResolved{class: class, role: role, family: family}
		if u, perr := strconv.ParseUint(idValue, 10, 64); perr == nil {
			r.numericID = int64(u) //nolint:gosec // reinterpretation bit-a-bit, cf. UBigint
		}
		out[key] = r
	}
	if err := rows.Err(); err != nil {
		slog.Log(ctx, degradedLookupLevel(ctx, err),
			"weapon resolver: itération interrompue, dimensions partielles", "title", titleSlug, "err", err)
	}
	for key, names := range loadCuratedWeaponNames(ctx, meta, titleSlug, keys) {
		if r, ok := out[key]; ok {
			r.label = names
			out[key] = r
		}
	}
	return out
}
