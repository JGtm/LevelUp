// Package duckdb — match_view_repo_weapons.go : kills par arme (joueur +
// bulk scoreboard) + helpers lookup généraux. Découpé de match_view_repo.go
// (god-file split, refactor 2026-05-27).
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/weapons/filmshell"
)

type weaponMetaEntry struct {
	label         string
	canonicalName string
	// class / role / family : dimensions du registre (axe manipulation + fonction de
	// combat + famille), résolues dans la même passe que le label (resolveWeaponMeta les
	// porte déjà). Vides si l'arme est absente du registre. Alimentent la FragDistribution
	// par-match (family = TYPE de grenade au niveau 2, V72-15.2).
	class  string
	role   string
	family string
	// weaponKey : clé canonique du registre, niveau 2 des classes ventilées par ENGIN
	// (véhicule/tourelle — V73-3.2, où class/role/family valent tous « vehicle »).
	weaponKey string
}

// lookupWeaponMeta résout le nom d'affichage DANS LA LOCALE DE REQUÊTE (ctxkeys.Locale,
// cf. weaponResolved.displayLabel) + name_en + class/role depuis le registre.
// name_en est nécessaire pour construire l'URL image via AssetURLAdapter.WeaponImageURL ;
// class/role alimentent la FragDistribution par-match (sunburst v2).
func (r *MatchViewRepo) lookupWeaponMeta(ctx context.Context, weaponIDs []int64) map[int64]weaponMetaEntry {
	result := map[int64]weaponMetaEntry{}
	if len(weaponIDs) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return result
	}
	locale := ctxkeys.Locale(ctx)
	// PASSAGE PRINCIPAL P4 : résolution via le registre + weapon_labels (parité du
	// nom). On ne garde que les ids résolus (label non vide), comme l'ancien lookup.
	for id, m := range resolveWeaponMeta(ctx, r.pdb.Metadata, r.pdb.TitleSlug, weaponIDs) {
		if !m.label.Empty() {
			result[id] = weaponMetaEntry{
				label: m.displayLabel(locale), canonicalName: m.canonicalName, class: m.class, role: m.role,
				family: m.family, weaponKey: m.weaponKey,
			}
		}
	}
	return result
}

// lookupWeaponLabels : le seul nom d'affichage d'une arme, dans la locale de requête
// (arme favorite du scoreboard).
func (r *MatchViewRepo) lookupWeaponLabels(ctx context.Context, weaponIDs []int64) map[int64]string {
	labels := map[int64]string{}
	if len(weaponIDs) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return labels
	}
	locale := ctxkeys.Locale(ctx)
	for id, m := range resolveWeaponMeta(ctx, r.pdb.Metadata, r.pdb.TitleSlug, weaponIDs) {
		if !m.label.Empty() {
			labels[id] = m.displayLabel(locale)
		}
	}
	return labels
}

// lookupLabelsByID résout id → libellé par la requête paramétrée queryTemplate. Rend les libellés
// lus ET l'erreur d'une requête ou d'une lecture en échec : une erreur n'est JAMAIS une map vide.
// Un libellé vide est une absence de nom (ignorée), pas une panne. Sans identifiant ni base, map
// vide sans erreur.
func lookupLabelsByID(ctx context.Context, db *DB, queryTemplate string, ids []int64) (map[int64]string, error) {
	labels := map[int64]string{}
	query, args, ok := buildLookupQuery(queryTemplate, ids)
	if !ok || db == nil {
		return labels, nil
	}

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return labels, fmt.Errorf("lookupLabelsByID: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var label string
		if err := rows.Scan(&id, &label); err != nil {
			return labels, fmt.Errorf("lookupLabelsByID scan: %w", err)
		}
		if label != "" {
			labels[id] = label
		}
	}
	if err := rows.Err(); err != nil {
		return labels, fmt.Errorf("lookupLabelsByID rows: %w", err)
	}
	return labels, nil
}

func buildLookupQuery(queryTemplate string, ids []int64) (string, []interface{}, bool) {
	uniqueIDs := uniqueInt64s(ids)
	if len(uniqueIDs) == 0 {
		return "", nil, false
	}

	placeholders := make([]string, 0, len(uniqueIDs))
	args := make([]interface{}, 0, len(uniqueIDs))
	for _, id := range uniqueIDs {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	return fmt.Sprintf(queryTemplate, strings.Join(placeholders, ",")), args, true
}

func uniqueInt64s(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	unique := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

// GetMatchBulkWeaponKills retourne les kills par arme de tous les joueurs (Q28).
// Applique la fusion variante→canonique par xuid (regroupe Duelist Energy Sword
// + Elite Bloodblade + Energy Sword sous le même canonique pour chaque joueur).
// Exécutée sur SharedReader (ADR 0016) — Q28 lit weapon_kills (shared-only).
//
// `v_weapon_kills` est une relation PROPRE AU TITRE (le fichier d'un titre à décodeur de film ne la
// porte pas, cf. `sharedSnapshotTitleOwnedRaw`) : son absence du lecteur est une absence constatée
// par sonde (nil), pas une erreur. Un lecteur indisponible, une sonde ou une requête en échec, ou une
// ligne illisible remontent.
func (r *MatchViewRepo) GetMatchBulkWeaponKills(ctx context.Context, matchID string) ([]domain.BulkWeaponKillRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Titre a decodeur de film : l arme vient de la SOURCE DU DEGAT, pas d une
	// correlation de tirs (bascule du 2026-09-01). Aucun `slug ==` — c est la presence
	// du traducteur, injectee par capability au cablage, qui decide.
	if r.killSourceClassifier != nil {
		return r.bulkWeaponKillsFromSource(ctx, matchID)
	}

	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchBulkWeaponKills: shared reader: %w", err)
	}
	defer release()

	present, err := relationExists(ctx, sharedDB, "v_weapon_kills")
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchBulkWeaponKills: %w", err)
	}
	if !present {
		slog.DebugContext(ctx, "match_view: v_weapon_kills absente de ce lecteur, aucune arme par joueur",
			"match_id", matchID)
		return nil, nil
	}

	rows, err := sharedDB.QueryContext(ctx, Q28BulkWeaponKills, matchID)
	if err != nil {
		return nil, fmt.Errorf("MatchViewRepo.GetMatchBulkWeaponKills: Q28: %w", err)
	}
	defer rows.Close()

	type key struct {
		xuid string
		wid  int64
	}
	killsByKey := make(map[key]int)
	mechanicByKey := make(map[key]int) // sous-ensemble non-arme (kill_kind <> 'weapon'), V72-15.3
	ordered := make([]key, 0, 32)
	for rows.Next() {
		var xuid string
		var widU uint64
		var kills, mechanicKills int
		if err := rows.Scan(&xuid, &widU, &kills, &mechanicKills); err != nil {
			return nil, fmt.Errorf("MatchViewRepo.GetMatchBulkWeaponKills scan: %w", err)
		}
		canonicalU := widU
		if canon, ok := filmshell.WeaponFusionMapID[widU]; ok {
			canonicalU = canon
		}
		k := key{xuid: xuid, wid: int64(canonicalU)} //nolint:gosec
		if _, seen := killsByKey[k]; !seen {
			ordered = append(ordered, k)
		}
		killsByKey[k] += kills
		mechanicByKey[k] += mechanicKills
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	results := make([]domain.BulkWeaponKillRaw, 0, len(ordered))
	weaponIDs := make([]int64, 0, len(ordered))
	for _, k := range ordered {
		results = append(results, domain.BulkWeaponKillRaw{
			XUID:          k.xuid,
			WeaponID:      k.wid,
			Kills:         killsByKey[k],
			MechanicKills: mechanicByKey[k],
		})
		weaponIDs = append(weaponIDs, k.wid)
	}

	labelBulkWeaponKills(results, r.lookupWeaponMeta(ctx, weaponIDs))
	return results, nil
}

// labelBulkWeaponKills pose le libellé et les dimensions du registre sur chaque ligne. Une arme
// absente de metadata.weapon_labels (variante) garde son weapon_id en chaîne : le frontend n'a
// jamais à gérer un weapon_label vide (`??` ne retombe pas sur "").
func labelBulkWeaponKills(results []domain.BulkWeaponKillRaw, weapMeta map[int64]weaponMetaEntry) {
	for i := range results {
		if m, ok := weapMeta[results[i].WeaponID]; ok {
			results[i].WeaponLabel = m.label
			results[i].Class = m.class
			results[i].Role = m.role
			results[i].Family = m.family
			results[i].WeaponKey = m.weaponKey
			continue
		}
		results[i].WeaponLabel = strconv.FormatInt(results[i].WeaponID, 10)
	}
}
