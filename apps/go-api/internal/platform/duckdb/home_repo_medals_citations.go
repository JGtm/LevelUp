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
	"strconv"
	"strings"

	"levelup/go-api/internal/assets/static"
	"levelup/go-api/internal/domain"
)

// homeMedalIconURL retourne l'URL d'une icône de médaille à partir de son ID.
// slug est le slug du titre du joueur (cf. HomeRepo.titleSlug()).
func homeMedalIconURL(slug string, medalID int64) string {
	return static.URL(static.KindMedal, slug, strconv.FormatInt(medalID, 10), ".png")
}

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

	// Résolution du label depuis metadata.
	// Contournement driver : database/sql ne supporte pas uint64 avec bit63=1.
	// weapon_id est une valeur interne (pas user input) → littéral décimal sûr.
	nameCol := "COALESCE(name_fr, name_en, '')"
	if locale == "en" {
		nameCol = "COALESCE(name_en, name_fr, '')"
	}
	var weaponName string
	metaErr := r.pdb.Metadata.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM weapon_labels WHERE weapon_id = %d", nameCol, weaponID), //nolint:gosec
	).Scan(&weaponName)
	if metaErr != nil || weaponName == "" {
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

	for _, rr := range rawRows {
		meta := metaMap[rr.medalID]
		result[rr.matchID] = append(result[rr.matchID], domain.RecentMatchMedal{
			MedalID:     rr.medalID,
			Name:        meta.label,
			Count:       rr.count,
			Description: meta.description,
			ImageURL:    homeMedalIconURL(r.titleSlug(), rr.medalID),
			Difficulty:  meta.difficulty,
		})
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
