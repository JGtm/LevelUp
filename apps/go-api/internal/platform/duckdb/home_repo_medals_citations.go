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

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/assets/static"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
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
	// Titre a decodeur de film : l arme favorite vient de la SOURCE DU DEGAT — celle qui
	// voit l epee et le marteau, invisibles a la correlation de tirs (bascule 2026-09-01).
	if r.killSourceClassifier != nil {
		return r.favoriteWeaponFromDamageSource(ctx, locale)
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
	// weapon_labels). Le nom suit la locale par weaponResolved.displayLabel — depuis le
	// 2026-09-17 l'EN vient de weapon_name_labels.name_en (le libellé du TOML), plus de
	// weapon_labels.name_en (le nom brut d'URL image, jamais un libellé d'affichage).
	weaponName := ""
	if m, ok := resolveWeaponMeta(ctx, r.pdb.Metadata, r.pdb.TitleSlug, []int64{int64(weaponID)})[int64(weaponID)]; ok {
		weaponName = m.displayLabel(locale)
	}
	if weaponName == "" {
		weaponName = i18n.Msg(i18n.Parse(locale), "common.unknown_weapon", nil)
	}
	return weaponName, totalKills, nil
}

// LoadMatchMedals charge les médailles d'un joueur pour un lot de matchs (Q26h).
// Retourne un map match_id → []domain.RecentMatchMedal, trié par count DESC.
// Labels résolus via medal_translations (langue de la requête, repli `en`), citation_mappings en fallback.
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

	metaMap := resolveMedalLabels(ctx, r.pdb.Metadata, medalIDsList, ctxkeys.Locale(ctx))

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

// resolveMedalLabels résout les labels de médailles dans la langue de la requête (GH2-B6) : la
// difficulté vient de medal_definitions, label et description de medal_translations par la source
// unique loadMedalLabels (langue demandée, sinon `en`) — partagée avec la vue Match (GH-5b) et
// l'Explorer : sous UI EN, la tuile Home n'affiche jamais de français.
func resolveMedalLabels(ctx context.Context, db *DB, medalIDs []int64, locale string) map[int64]medalLabel {
	result := make(map[int64]medalLabel, len(medalIDs))
	if len(medalIDs) == 0 || db == nil {
		return result
	}

	difficulties := loadMedalDifficulties(ctx, db, medalIDs)
	labels, err := loadMedalLabels(ctx, db, medalIDs, locale)
	if err != nil {
		slog.ErrorContext(ctx, "home: textes des médailles illisibles, libellés repliés", "err", err)
	}
	for id, diff := range difficulties {
		if t := labels[id]; t.label != "" {
			result[id] = medalLabel{label: t.label, description: t.description, difficulty: diff}
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
	for id, label := range citationLabelsOrLog(ctx, db, missing, "home") {
		result[id] = medalLabel{label: label, difficulty: "Normal"}
	}
	return result
}

// loadMedalDifficulties lit la difficulté (colonne non textuelle de medal_definitions) des médailles
// `medalIDs`. Une lecture en échec est journalisée et rend ce qui a été lu : les libellés se replient
// alors sur citation_mappings.
func loadMedalDifficulties(ctx context.Context, db *DB, medalIDs []int64) map[int64]string {
	out := make(map[int64]string, len(medalIDs))
	q, args, ok := buildLookupQuery(
		`SELECT md.medal_name_id, COALESCE(NULLIF(TRIM(md.difficulty),''), 'Normal') AS difficulty
		 FROM medal_definitions md
		 WHERE md.medal_name_id IN (%s)`,
		medalIDs,
	)
	if !ok {
		return out
	}
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		slog.ErrorContext(ctx, "home: medal_definitions illisible, libellés repliés", "err", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var diff string
		if err := rows.Scan(&id, &diff); err != nil {
			slog.ErrorContext(ctx, "home: medal_definitions, ligne illisible", "err", err)
			continue
		}
		out[id] = diff
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "home: medal_definitions, lecture interrompue", "err", err)
	}
	return out
}

// loadMatchCitationRows lit sur la base joueur le gain de chaque citation à chacun des matchs
// `matchIDs` (Q26i), groupé par match, et les citations rencontrées. Dégradation silencieuse :
// table absente ou lecture en échec -> rien (map vide).
func (r *HomeRepo) loadMatchCitationRows(ctx context.Context, matchIDs []string) (map[string][]citationRowScan, []string) {
	placeholders := make([]string, len(matchIDs))
	args := make([]interface{}, len(matchIDs))
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(Q26iMatchCitationsTemplate, strings.Join(placeholders, ", "))

	rows, err := r.pdb.ReadDB().QueryRecovered(ctx, query, args...)
	if err != nil {
		if !isTableNotFoundErr(err) {
			slog.WarnContext(ctx, "LoadMatchCitations: lecture des gains par match en échec, citations omises",
				"xuid", r.pdb.XUID, "err", err)
		}
		return nil, nil // dégradation : table absente (instance fraîche) ou lecture en échec
	}
	defer rows.Close()

	rawByMatch := make(map[string][]citationRowScan)
	normsSeen := make(map[string]struct{})
	var norms []string
	for rows.Next() {
		var c citationRowScan
		if err := rows.Scan(&c.matchID, &c.norm, &c.delta); err != nil {
			continue
		}
		rawByMatch[c.matchID] = append(rawByMatch[c.matchID], c)
		if _, ok := normsSeen[c.norm]; !ok {
			normsSeen[c.norm] = struct{}{}
			norms = append(norms, c.norm)
		}
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(ctx, "LoadMatchCitations: lecture des gains par match interrompue, citations omises",
			"xuid", r.pdb.XUID, "err", err)
		return nil, nil
	}
	return rawByMatch, norms
}

// LoadMatchCitations charge les citations progressées pour un lot de matchs (Q26i + Q26j).
// Retourne un map match_id → []domain.HomeMatchCitationRaw, dégradation silencieuse.
func (r *HomeRepo) LoadMatchCitations(ctx context.Context, matchIDs []string) (map[string][]domain.HomeMatchCitationRaw, error) {
	result := make(map[string][]domain.HomeMatchCitationRaw)
	if len(matchIDs) == 0 || r.pdb == nil || r.pdb.Player == nil {
		return result, nil
	}

	// Étape 1 : charger les gains par match depuis match_citations (player DB).
	rawByMatch, norms := r.loadMatchCitationRows(ctx, matchIDs)
	if len(rawByMatch) == 0 {
		return result, nil
	}

	// Étape 2 : charger les métadonnées (display, image_path, tier_targets) depuis metadata,
	// et le cumul de chaque citation arrêté à chacun de ces matchs (citations_cumul.go).
	metaMap := loadCitationMetaOrLog(ctx, r.pdb.Metadata, norms)
	cumul, err := loadCitationCumulAfterMatch(ctx, r.pdb, norms)
	if err != nil {
		// Sans cumul arrêté au match, l'anneau et le palier seraient faux : pas de citation.
		slog.WarnContext(ctx, "LoadMatchCitations: cumul par match indisponible, citations omises",
			"xuid", r.pdb.XUID, "err", err)
		return result, nil
	}

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
				Cumulative:  cumul[analysis.CitationMatchKey{MatchID: matchID, Norm: c.norm}],
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
