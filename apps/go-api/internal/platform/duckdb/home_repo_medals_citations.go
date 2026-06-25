// Package duckdb — home_repo_medals_citations.go : médailles par match
// (Q26h), citations progressées (Q26i/Q26j), arme favorite (Q26k).
//
// Sous-module de home_repo.go (split god-file 2026-05-21).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/assets/static"
	"levelup/go-api/internal/domain"
)

// LoadFavoriteWeapon retourne le nom localisé et le nombre de kills de l'arme la plus
// utilisée par le joueur sur l'ensemble de ses matchs (Q26k).
//
// Dégradation : on tolère silencieusement sql.ErrNoRows (joueur sans kills) et la
// vue absente (instance fraîche pré-migration). Pour TOUTE autre erreur SQL (driver,
// connexion, scan), on log un WARN structuré pour transformer la cécité de l'ancien
// silent drop en observabilité. Le contrat externe reste ("", 0, nil) — le front
// affichera "—" comme avant — mais l'opérateur voit le vrai problème dans les logs.
func (r *HomeRepo) LoadFavoriteWeapon(ctx context.Context, locale string) (string, int, error) {
	var weaponID uint64
	var totalKills int

	// Sprint P7 / ADR 0016 : v_weapon_kills est shared-only, exécuter via
	// SharedReader (la conn player n'a plus shared attaché).
	if r.pdb == nil || r.pdb.SharedReader == nil {
		return "", 0, nil
	}
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "LoadFavoriteWeapon: SharedReader unavailable",
			"err", err, "xuid", r.pdb.XUID, "locale", locale)
		return "", 0, nil //nolint:nilerr // dégradation silencieuse côté contrat externe
	}
	err = sharedDB.QueryRowContext(ctx, Q26kFavoriteWeapon, r.pdb.XUID).Scan(&weaponID, &totalKills)
	release()
	if err != nil {
		if err != sql.ErrNoRows && !isTableNotFoundErr(err) {
			slog.WarnContext(ctx, "LoadFavoriteWeapon: query failed (silent degradation)",
				"err", err, "xuid", r.pdb.XUID, "locale", locale)
		}
		return "", 0, nil //nolint:nilerr // dégradation silencieuse côté contrat externe
	}

	// PASSAGE PRINCIPAL P4 : résolution via le resolver unifié (registre +
	// weapon_labels, parité). locale EN = name_en sinon le label FR (parité avec
	// l'ancien COALESCE(name_en, name_fr)).
	weaponName := ""
	if m, ok := resolveWeaponMeta(ctx, r.pdb.Metadata, r.pdb.TitleSlug, []int64{int64(weaponID)})[int64(weaponID)]; ok {
		if locale == "en" && m.nameEN != "" {
			weaponName = m.nameEN
		} else {
			weaponName = m.label
		}
	}
	if weaponName == "" {
		weaponName = "Inconnue"
		if locale == "en" {
			weaponName = "Unknown"
		}
	}
	return weaponName, totalKills, nil
}

// LoadMatchMedals charge les médailles d'un joueur pour un lot de matchs (Q26h).
// Retourne un map match_id → []domain.RecentMatchMedal, trié par count DESC.
// Labels résolus via medal_definitions (name_fr) en priorité, citation_mappings en fallback.
func (r *HomeRepo) LoadMatchMedals(ctx context.Context, matchIDs []string) (map[string][]domain.RecentMatchMedal, error) {
	result := make(map[string][]domain.RecentMatchMedal)
	if len(matchIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(matchIDs))
	args := make([]interface{}, 0, len(matchIDs)+1)
	args = append(args, r.pdb.XUID)
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := fmt.Sprintf(Q26hMatchMedalsTemplate, strings.Join(placeholders, ", "))

	// Phase 3 plan stabilisation 2026-05-22 : migré de pdb.ReadDB() (player
	// conn) vers SharedReader.Get(). shared.medals_earned vit dans
	// shared_matches_v2 et l'ATTACH shared sur la player conn a été retiré
	// (ADR 0016). Référence sans préfixe `shared.` désormais.
	if r.pdb.SharedReader == nil {
		return result, nil
	}
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return result, nil
	}
	defer release()
	rows, err := sharedDB.QueryContext(ctx, query, args...)
	if err != nil {
		return result, nil // dégradation silencieuse
	}
	defer rows.Close()

	type rawRow struct {
		matchID string
		medalID int64
		count   int
	}
	var rawRows []rawRow
	var medalIDsList []int64
	seen := make(map[int64]struct{})
	for rows.Next() {
		var rr rawRow
		if err := rows.Scan(&rr.matchID, &rr.medalID, &rr.count); err != nil {
			continue
		}
		rawRows = append(rawRows, rr)
		if _, ok := seen[rr.medalID]; !ok {
			seen[rr.medalID] = struct{}{}
			medalIDsList = append(medalIDsList, rr.medalID)
		}
	}
	if err := rows.Err(); err != nil {
		return result, nil
	}

	metaMap := resolveMedalLabels(ctx, r.pdb.Metadata, medalIDsList)

	slug := r.titleSlug()
	for _, rr := range rawRows {
		meta := metaMap[rr.medalID]
		png, sp := static.MedalImage(slug, rr.medalID)
		m := domain.RecentMatchMedal{
			MedalID:     rr.medalID,
			Name:        meta.label,
			Count:       rr.count,
			Description: meta.description,
			Difficulty:  meta.difficulty,
		}
		if sp != nil {
			m.SpriteSheet, m.SpriteLeft, m.SpriteTop, m.SpriteWidth, m.SpriteHeight =
				sp.SheetURL, sp.Left, sp.Top, sp.Width, sp.Height
		} else {
			m.ImageURL = png
		}
		result[rr.matchID] = append(result[rr.matchID], m)
	}
	return result, nil
}

// medalLabel contient le nom localisé et la description d'une médaille.
type medalLabel struct {
	label       string
	description string
	difficulty  string
}

// resolveMedalLabels résout les labels de médailles avec la chaîne BCP-47 complète :
//
//	medal_translations (fr-FR) → medal_definitions.name_fr
//	→ medal_translations (en-US) → medal_definitions.name_en
//
// Miroir de resolve_medal_name(id, lang) dans src/data/medal_definitions.py.
func resolveMedalLabels(ctx context.Context, db *DB, medalIDs []int64) map[int64]medalLabel {
	result := make(map[int64]medalLabel, len(medalIDs))
	if len(medalIDs) == 0 || db == nil {
		return result
	}

	// Chaîne BCP-47 : medal_translations (fr-FR, en-US) > medal_definitions (name_fr, name_en).
	q, mArgs, ok := buildLookupQuery(
		`SELECT md.medal_name_id,
		        COALESCE(
		            NULLIF(TRIM(mt_fr.name),''),
		            NULLIF(TRIM(md.name_fr),''),
		            NULLIF(TRIM(mt_en.name),''),
		            NULLIF(TRIM(md.name_en),'')
		        ) AS label,
		        COALESCE(
		            NULLIF(TRIM(md.description_fr),''),
		            NULLIF(TRIM(md.description_en),''),
		            ''
		        ) AS description,
		        COALESCE(NULLIF(TRIM(md.difficulty),''), 'Normal') AS difficulty
		 FROM medal_definitions md
		 LEFT JOIN medal_translations mt_fr
		     ON mt_fr.medal_name_id = md.medal_name_id AND mt_fr.lang = 'fr-FR'
		 LEFT JOIN medal_translations mt_en
		     ON mt_en.medal_name_id = md.medal_name_id AND mt_en.lang = 'en-US'
		 WHERE md.medal_name_id IN (%s)`,
		medalIDs,
	)
	if !ok {
		return result
	}
	mRows, err := db.Query(ctx, q, mArgs...)
	if err != nil {
		return result
	}
	defer mRows.Close()
	for mRows.Next() {
		var id int64
		var name, desc, diff string
		if err := mRows.Scan(&id, &name, &desc, &diff); err == nil && name != "" {
			result[id] = medalLabel{label: name, description: desc, difficulty: diff}
		}
	}

	// Fallback citation_mappings pour les IDs absents de medal_definitions
	// (corrige la tuile de match Home qui affichait des libellés vides). Source
	// unique partagée avec la vue Match et l'Explorer : medal_citation_fallback.go.
	missing := make([]int64, 0)
	for _, id := range medalIDs {
		if _, ok := result[id]; !ok {
			missing = append(missing, id)
		}
	}
	for id, label := range lookupMedalCitationLabels(ctx, db, missing) {
		result[id] = medalLabel{label: label, difficulty: "Normal"}
	}
	return result
}

// LoadMatchCitations charge les citations progressées pour un lot de matchs (Q26i + Q26j).
// Retourne un map match_id → []domain.HomeMatchCitationRaw, dégradation silencieuse.
func (r *HomeRepo) LoadMatchCitations(ctx context.Context, matchIDs []string) (map[string][]domain.HomeMatchCitationRaw, error) {
	result := make(map[string][]domain.HomeMatchCitationRaw)
	if len(matchIDs) == 0 || r.pdb == nil || r.pdb.Player == nil {
		return result, nil
	}

	// Étape 1 : charger les deltas + cumulatifs depuis match_citations (player DB).
	placeholders := make([]string, len(matchIDs))
	args := make([]interface{}, len(matchIDs))
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(Q26iMatchCitationsTemplate, strings.Join(placeholders, ", "))

	rows, err := r.pdb.ReadDB().Query(ctx, query, args...)
	if err != nil {
		if isTableNotFoundErr(err) {
			return result, nil
		}
		return result, nil // dégradation silencieuse
	}
	defer rows.Close()

	type citIntermediate struct {
		norm       string
		delta      int
		cumulative int
	}
	rawByMatch := make(map[string][]citIntermediate)
	normsSeen := make(map[string]struct{})
	for rows.Next() {
		var matchID, norm string
		var delta, cumulative int
		if err := rows.Scan(&matchID, &norm, &delta, &cumulative); err != nil {
			continue
		}
		rawByMatch[matchID] = append(rawByMatch[matchID], citIntermediate{norm, delta, cumulative})
		normsSeen[norm] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return result, nil
	}
	if len(rawByMatch) == 0 {
		return result, nil
	}

	// Étape 2 : charger les métadonnées (display, image_path, tier_targets) depuis metadata.
	norms := make([]string, 0, len(normsSeen))
	for n := range normsSeen {
		norms = append(norms, n)
	}
	metaMap := r.loadCitationMappingMeta(ctx, norms)

	// Étape 3 : merger.
	for matchID, cits := range rawByMatch {
		for _, c := range cits {
			meta := metaMap[c.norm]
			var imgPath *string
			if meta.imagePath != "" {
				imgPath = &meta.imagePath
			}
			result[matchID] = append(result[matchID], domain.HomeMatchCitationRaw{
				Norm:        c.norm,
				Display:     meta.display,
				Description: meta.description,
				ImagePath:   safeStringValue(imgPath),
				TierTargets: meta.tierTargets,
				Delta:       c.delta,
				Cumulative:  c.cumulative,
			})
		}
	}
	return result, nil
}

// homeMatchCommendationsTopN : nombre de commendations natives retenues par match
// pour le slot TopCitations de la MatchCard (parité maxCitationSnippets côté service).
const homeMatchCommendationsTopN = 3

// homeMatchCommendationsQuery — top N commendations natives gagnées par le joueur
// (xuid) sur un lot de matchs, ordonnées count DESC par match. ART-safe (lecture
// pure sur match_commendations, table INSERT-only keyée (match_id, xuid,
// commendation_id) — pas de _latest nécessaire : le `count` par-match est immuable).
// Le tie-break par commendation_id rend la sélection top-N déterministe.
//
// `progress` = total cumulatif À VIE à ce match (valeur absolue 343, nullable pour
// les lignes pré-migration → COALESCE 0). Alimente le palier/masterisé read-time.
const homeMatchCommendationsQueryTemplate = `
SELECT match_id, commendation_id, count, progress
FROM (
    SELECT mc.match_id              AS match_id,
           mc.commendation_id       AS commendation_id,
           mc.count                 AS count,
           COALESCE(mc.progress, 0) AS progress,
           ROW_NUMBER() OVER (
               PARTITION BY mc.match_id
               ORDER BY mc.count DESC, mc.commendation_id ASC
           ) AS rn
    FROM match_commendations mc
    WHERE mc.xuid = ? AND mc.match_id IN (%s) AND mc.count > 0
) ranked
WHERE rn <= %d`

// LoadMatchCommendations charge les commendations NATIVES gagnées par le joueur sur un
// lot de matchs (Halo 5 : shared.match_commendations), top N par match (count DESC),
// enrichies nom + icône via commendation_definitions (metadata). Retourne un map
// match_id → []domain.HomeMatchCommendationRaw.
//
// Dégradation silencieuse (contrat externe = map possiblement vide, jamais d'erreur
// propagée) : SharedReader indisponible, table match_commendations absente (instance
// pré-migration / titre sans commendations natives), ou erreur SQL. Pour Halo Infinite
// la table est vide → map vide → le slot TopCitations reste alimenté par les citations
// dérivées (cf. enrichMatchesWithCommendations, fallback title-agnostic).
func (r *HomeRepo) LoadMatchCommendations(ctx context.Context, matchIDs []string) (map[string][]domain.HomeMatchCommendationRaw, error) {
	result := make(map[string][]domain.HomeMatchCommendationRaw)
	if len(matchIDs) == 0 || r.pdb == nil || r.pdb.SharedReader == nil || strings.TrimSpace(r.pdb.XUID) == "" {
		return result, nil
	}

	placeholders := make([]string, len(matchIDs))
	args := make([]interface{}, 0, len(matchIDs)+1)
	args = append(args, r.pdb.XUID)
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := fmt.Sprintf(homeMatchCommendationsQueryTemplate, strings.Join(placeholders, ", "), homeMatchCommendationsTopN)

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		// Transitoire (handle partagé indisponible) : dégradation silencieuse, trace Debug.
		slog.DebugContext(ctx, "LoadMatchCommendations: SharedReader indisponible (dégradation)",
			"err", err, "xuid", r.pdb.XUID)
		return result, nil
	}
	defer release()
	rows, err := sharedDB.QueryContext(ctx, query, args...)
	if err != nil {
		// Table absente = attendu pour les titres sans commendations natives (ex. Halo
		// Infinite) → Debug (sinon spam à chaque Home). Toute autre erreur SQL → Warn.
		if isTableNotFoundErr(err) {
			slog.DebugContext(ctx, "LoadMatchCommendations: table match_commendations absente (titre sans natif)",
				"xuid", r.pdb.XUID)
		} else {
			slog.WarnContext(ctx, "LoadMatchCommendations: query échouée (dégradation silencieuse)",
				"err", err, "xuid", r.pdb.XUID)
		}
		return result, nil
	}
	defer rows.Close()

	type rawRow struct {
		matchID  string
		commID   string
		count    int
		progress int
	}
	var rawRows []rawRow
	idSeen := make(map[string]struct{})
	var commIDs []string
	for rows.Next() {
		var rr rawRow
		if err := rows.Scan(&rr.matchID, &rr.commID, &rr.count, &rr.progress); err != nil {
			continue
		}
		rawRows = append(rawRows, rr)
		if _, ok := idSeen[rr.commID]; !ok {
			idSeen[rr.commID] = struct{}{}
			commIDs = append(commIDs, rr.commID)
		}
	}
	if err := rows.Err(); err != nil || len(rawRows) == 0 {
		return result, nil
	}

	defs := r.loadCommendationDefs(ctx, commIDs)
	for _, rr := range rawRows {
		d := defs[rr.commID]
		name := d.name
		if name == "" {
			name = rr.commID // dégradation : ID court si la définition n'est pas seedée
		}
		result[rr.matchID] = append(result[rr.matchID], domain.HomeMatchCommendationRaw{
			ID:          rr.commID,
			Name:        name,
			IconURL:     d.iconURL,
			Count:       rr.count,
			Progress:    rr.progress,
			TierTargets: d.tierTargets,
		})
	}
	return result, nil
}

// commendationDef contient nom localisé + icône + paliers d'une commendation native
// par UUID.
type commendationDef struct {
	name        string
	iconURL     string
	tierTargets string // CSV croissant des seuils de paliers (vide si Meta/Daily ou non seedé)
}

// loadCommendationDefs résout nom (FR > EN) + icône CDN + tier_targets pour un ensemble
// d'UUID de commendations depuis commendation_definitions (metadata h5). IDs inconnus
// absents de la map. Dégradation : metadata nil / table absente → map vide.
//
// tier_targets est COALESCE'é vide : nullable + colonne possiblement absente sur une
// DB pré-migration. Pour absorber le cas « colonne absente » (metadata h5 provisionnée
// avant l'ajout de tier_targets), on retombe sur une requête sans tier_targets si le
// SELECT échoue (table présente, colonne manquante).
func (r *HomeRepo) loadCommendationDefs(ctx context.Context, ids []string) map[string]commendationDef {
	out := make(map[string]commendationDef, len(ids))
	if len(ids) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return out
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	in := strings.Join(placeholders, ", ")
	query := `SELECT commendation_id,
	                 COALESCE(NULLIF(TRIM(name_fr), ''), name_en) AS name,
	                 COALESCE(icon_url, '') AS icon_url,
	                 COALESCE(tier_targets, '') AS tier_targets
	          FROM commendation_definitions
	          WHERE commendation_id IN (` + in + `)`
	rows, err := r.pdb.Metadata.Query(ctx, query, args...)
	if err != nil {
		// Colonne tier_targets absente (DB pré-migration) → retry sans elle (le
		// progrès dégradera proprement en anneau vide).
		queryNoTiers := `SELECT commendation_id,
		                        COALESCE(NULLIF(TRIM(name_fr), ''), name_en) AS name,
		                        COALESCE(icon_url, '') AS icon_url
		                 FROM commendation_definitions
		                 WHERE commendation_id IN (` + in + `)`
		rows, err = r.pdb.Metadata.Query(ctx, queryNoTiers, args...)
		if err != nil {
			return out // dégradation : titre sans table commendation_definitions
		}
		defer rows.Close()
		for rows.Next() {
			var id, name, icon string
			if err := rows.Scan(&id, &name, &icon); err != nil {
				continue
			}
			out[id] = commendationDef{name: name, iconURL: icon}
		}
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, name, icon, tiers string
		if err := rows.Scan(&id, &name, &icon, &tiers); err != nil {
			continue
		}
		out[id] = commendationDef{name: name, iconURL: icon, tierTargets: tiers}
	}
	return out
}

type citationMeta struct {
	display     string
	imagePath   string
	tierTargets string
	description string
}

// loadCitationMappingMeta interroge citation_mappings sur pdb.Metadata pour un ensemble de norms.
func (r *HomeRepo) loadCitationMappingMeta(ctx context.Context, norms []string) map[string]citationMeta {
	result := make(map[string]citationMeta, len(norms))
	if len(norms) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return result
	}

	placeholders := make([]string, len(norms))
	args := make([]interface{}, len(norms))
	for i, n := range norms {
		placeholders[i] = "?"
		args[i] = n
	}
	query := fmt.Sprintf(Q26jCitationMappingsForNormsTemplate, strings.Join(placeholders, ", "))

	rows, err := r.pdb.Metadata.Query(ctx, query, args...)
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var norm, display, imagePath, tierTargets, description string
		if err := rows.Scan(&norm, &display, &imagePath, &tierTargets, &description); err != nil {
			continue
		}
		result[norm] = citationMeta{display: display, imagePath: imagePath, tierTargets: tierTargets, description: description}
	}
	return result
}
