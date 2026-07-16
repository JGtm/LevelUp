// Package duckdb — citations_repo.go : accès DB pour les pages Citations et Commendations.
package duckdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// CitationsRepo implémente port.CitationsRepository.
type CitationsRepo struct {
	pdb *PlayerDB
}

// NewCitationsRepo crée un CitationsRepo pour un joueur.
func NewCitationsRepo(pdb *PlayerDB) *CitationsRepo {
	return &CitationsRepo{pdb: pdb}
}

// LoadCitationMappings charge les mappings de citations depuis metadata.duckdb (Q34).
// Utilise pdb.Metadata — pas pdb.Player.
func (r *CitationsRepo) LoadCitationMappings(ctx context.Context) ([]domain.CitationMappingRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.Metadata.Query(ctx, Q34CitationMappings)
	if err != nil {
		return nil, fmt.Errorf("LoadCitationMappings: %w", err)
	}
	defer rows.Close()

	enLocale := ctxkeys.Locale(ctx) == "en"
	var result []domain.CitationMappingRow
	for rows.Next() {
		var row domain.CitationMappingRow
		var displayEN string     // citation_name_display_en (COALESCE '' si NULL)
		var descriptionEN string // description_en (COALESCE '' si NULL)
		if err := rows.Scan(
			&row.NameNorm,
			&row.NameDisplay,
			&displayEN,
			&row.MappingType,
			&row.Category,
			&row.ImagePath,
			&row.Description,
			&descriptionEN,
			&row.TierTargets,
			&row.CompositeChildren,
		); err != nil {
			return nil, fmt.Errorf("LoadCitationMappings scan: %w", err)
		}
		// Locale-aware (GH4) : en UI anglaise, le nom anglais prime (sinon fallback FR)
		// et la description anglaise (description_en) prime ; si absente → description
		// masquée (nom seul, pointeur nil), jamais le FR (principe GH-5b). Les citations
		// Infinite étant copiées de H5, leur EN vient du seed — cf. ops.citationDisplayEN
		// / citationDescriptionEN.
		if enLocale {
			if displayEN != "" {
				row.NameDisplay = displayEN
			}
			if descriptionEN != "" {
				enDesc := descriptionEN
				row.Description = &enDesc
			} else {
				row.Description = nil
			}
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadCitationTotals charge les totaux agrégés depuis match_citations (Q35).
func (r *CitationsRepo) LoadCitationTotals(ctx context.Context) ([]domain.CitationTotalRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().QueryRecovered(ctx, Q35CitationTotals)
	if err != nil {
		return nil, fmt.Errorf("LoadCitationTotals: %w", err)
	}
	defer rows.Close()

	var result []domain.CitationTotalRow
	for rows.Next() {
		var row domain.CitationTotalRow
		if err := rows.Scan(
			&row.NameNorm,
			&row.Total,
		); err != nil {
			return nil, fmt.Errorf("LoadCitationTotals scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadMedalTotals charge les totaux de médailles depuis shared.medals_earned (Q36a).
func (r *CitationsRepo) LoadMedalTotals(ctx context.Context, xuid string) ([]domain.MedalEarnedRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadMedalTotals: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, Q36aMedalTotals, xuid)
	if err != nil {
		return nil, fmt.Errorf("LoadMedalTotals: %w", err)
	}
	defer rows.Close()

	var result []domain.MedalEarnedRow
	for rows.Next() {
		var row domain.MedalEarnedRow
		if err := rows.Scan(
			&row.MedalID,
			&row.TotalCount,
		); err != nil {
			return nil, fmt.Errorf("LoadMedalTotals scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadMedalCitationMappings charge les mappings médaille→citation depuis metadata.duckdb (Q36b).
// Utilise pdb.Metadata — pas pdb.Player.
func (r *CitationsRepo) LoadMedalCitationMappings(ctx context.Context) ([]domain.MedalCitationRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.Metadata.Query(ctx, Q36bMedalCitationMappings)
	if err != nil {
		return nil, fmt.Errorf("LoadMedalCitationMappings: %w", err)
	}
	defer rows.Close()

	var result []domain.MedalCitationRow
	for rows.Next() {
		var row domain.MedalCitationRow
		if err := rows.Scan(
			&row.MedalID,
			&row.NameDisplay,
			&row.Category,
			&row.ImagePath,
		); err != nil {
			return nil, fmt.Errorf("LoadMedalCitationMappings scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// LoadCitationMedalMappings charge les règles citation→medal_id pour le moteur de calcul (Q39).
// Utilise pdb.Metadata.
func (r *CitationsRepo) LoadCitationMedalMappings(ctx context.Context) ([]domain.CitationMedalMapping, error) {
	if r.pdb.Metadata == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.Metadata.Query(ctx, Q39CitationMedalMappings)
	if err != nil {
		return nil, fmt.Errorf("LoadCitationMedalMappings: %w", err)
	}
	defer rows.Close()

	var result []domain.CitationMedalMapping
	for rows.Next() {
		var m domain.CitationMedalMapping
		if err := rows.Scan(&m.NameNorm, &m.NameDisplay, &m.MedalID, &m.MappingType); err != nil {
			return nil, fmt.Errorf("LoadCitationMedalMappings scan: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// LoadMatchCitationsForView charge les top citations d'un match pour la vue
// détail (Q38).
//
// Pattern split cross-DB (post ADR 0016, fix incident 2026-05-26) :
//  1. Q38MatchViewCitationsPlayer sur pdb.Player → top 4 (norm, value)
//  2. Q26jCitationMappingsForNormsTemplate sur pdb.Metadata → display par norm
//  3. Merge en Go : COALESCE(display, norm) pour citation_name_display
//
// Le LEFT JOIN cross-DB de l'ancien Q38 levait silencieusement
// "Catalog Error: citation_mappings does not exist" (citation_mappings est
// dans metadata.duckdb, pas attachée aux conn player). Le caller capturait
// nil → page Match detail vide.
func (r *CitationsRepo) LoadMatchCitationsForView(ctx context.Context, matchID string) ([]domain.CitationMatchViewRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Étape 1 : top 4 citations bruts sur player DB.
	rows, err := r.pdb.ReadDB().QueryRecovered(ctx, Q38MatchViewCitationsPlayer, matchID)
	if err != nil {
		return nil, fmt.Errorf("LoadMatchCitationsForView player query: %w", err)
	}
	defer rows.Close()

	type rawRow struct {
		norm  string
		value int
	}
	var raws []rawRow
	normsSeen := make(map[string]struct{})
	for rows.Next() {
		var rr rawRow
		if err := rows.Scan(&rr.norm, &rr.value); err != nil {
			return nil, fmt.Errorf("LoadMatchCitationsForView scan: %w", err)
		}
		raws = append(raws, rr)
		normsSeen[rr.norm] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("LoadMatchCitationsForView rows.Err: %w", err)
	}
	if len(raws) == 0 {
		return nil, nil
	}

	// Étape 2 : lookup display côté metadata DB.
	norms := make([]string, 0, len(normsSeen))
	for n := range normsSeen {
		norms = append(norms, n)
	}
	metaMap := r.loadCitationMappingMeta(ctx, norms)

	// Étape 3 : merge — COALESCE(display, norm).
	result := make([]domain.CitationMatchViewRow, 0, len(raws))
	for _, rr := range raws {
		display := rr.norm
		if meta, ok := metaMap[rr.norm]; ok && meta.display != "" {
			display = meta.display
		}
		result = append(result, domain.CitationMatchViewRow{
			NameNorm:    rr.norm,
			NameDisplay: display,
			Value:       rr.value,
		})
	}
	return result, nil
}

// LoadMatchCitationsRich charge les citations d'un match avec cumul + métadonnées de paliers.
// Étape 1 : Q41 sur player DB → norm + delta + cumul.
// Étape 2 : Q26j sur metadata DB → display + image_path + tier_targets + description.
// Étape 3 : merge en Go (citation_mappings n'est pas attachée au player DB).
// Retour utilisable directement par analysis.BuildCitationSnippets.
func (r *CitationsRepo) LoadMatchCitationsRich(ctx context.Context, matchID string) ([]domain.HomeMatchCitationRaw, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := r.pdb.ReadDB().QueryRecovered(ctx, Q41SummaryTabCitations, matchID)
	if err != nil {
		return nil, nil //nolint:nilerr
	}
	defer rows.Close()

	type rawRow struct {
		norm       string
		delta      int
		cumulative int
	}
	var raws []rawRow
	normsSeen := make(map[string]struct{})
	for rows.Next() {
		var rr rawRow
		if err := rows.Scan(&rr.norm, &rr.delta, &rr.cumulative); err != nil {
			return nil, fmt.Errorf("LoadMatchCitationsRich scan: %w", err)
		}
		raws = append(raws, rr)
		normsSeen[rr.norm] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(raws) == 0 {
		return nil, nil
	}

	// Étape 2 : métadonnées sur metadata DB.
	norms := make([]string, 0, len(normsSeen))
	for n := range normsSeen {
		norms = append(norms, n)
	}
	metaMap := r.loadCitationMappingMeta(ctx, norms)

	// Étape 3 : merge. Display vide → BuildCitationSnippets filtre (norms internes
	// type _processed n'ont pas de mapping et doivent être ignorées).
	result := make([]domain.HomeMatchCitationRaw, 0, len(raws))
	for _, rr := range raws {
		meta := metaMap[rr.norm]
		result = append(result, domain.HomeMatchCitationRaw{
			Norm:        rr.norm,
			Display:     meta.display,
			Description: meta.description,
			ImagePath:   meta.imagePath,
			TierTargets: meta.tierTargets,
			Delta:       rr.delta,
			Cumulative:  rr.cumulative,
		})
	}
	return result, nil
}

// matchCommendationsRichQuery — toutes les commendations NATIVES gagnées par un
// joueur (xuid) sur UN match. ART-safe (lecture pure sur match_commendations,
// table INSERT-only keyée (match_id, xuid, commendation_id) : le `count` et le
// `progress` par-match sont immuables, pas de _latest requis). `progress` =
// cumul À VIE à ce match (nullable pré-migration → COALESCE 0). Ordre count DESC
// (tie-break commendation_id) pour une sélection top-N déterministe côté service.
const matchCommendationsRichQuery = `
SELECT mc.commendation_id, mc.count, COALESCE(mc.progress, 0) AS progress
FROM match_commendations mc
WHERE mc.match_id = ? AND mc.xuid = ? AND mc.count > 0
ORDER BY mc.count DESC, mc.commendation_id ASC`

// LoadMatchCommendationsRich charge les commendations NATIVES (Halo 5) gagnées par
// un joueur (xuid) sur UN match, enrichies nom + icône + tier_targets via
// commendation_definitions (metadata). Match-scoped, toutes les commendations (pas
// de top-N) : la sélection + le filtrage mastery sont faits au build côté service.
//
// Pattern split cross-DB (ADR 0016) : étape 1 sur SharedReader (match_commendations
// vit dans shared_matches_v2), étape 2 sur pdb.Metadata (commendation_definitions).
//
// Dégradation silencieuse (contrat = slice possiblement vide, jamais d'erreur) :
// SharedReader indisponible, table absente (Infinite / pré-migration), xuid vide.
// Pour Halo Infinite la table est vide → slice vide → l'onglet citations de la Match
// View reste alimenté par les citations dérivées (voie repo, NO-OP ici).
func (r *CitationsRepo) LoadMatchCommendationsRich(ctx context.Context, matchID, xuid string) ([]domain.HomeMatchCommendationRaw, error) {
	if r.pdb == nil || r.pdb.SharedReader == nil || strings.TrimSpace(matchID) == "" || strings.TrimSpace(xuid) == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, nil //nolint:nilerr // dégradation silencieuse côté contrat externe
	}
	rows, err := sharedDB.QueryContext(ctx, matchCommendationsRichQuery, matchID, xuid)
	if err != nil {
		release()
		return nil, nil //nolint:nilerr // table absente (Infinite) ou autre erreur SQL → vide
	}

	type rawRow struct {
		commID   string
		count    int
		progress int
	}
	var raws []rawRow
	idSeen := make(map[string]struct{})
	var commIDs []string
	for rows.Next() {
		var rr rawRow
		if err := rows.Scan(&rr.commID, &rr.count, &rr.progress); err != nil {
			continue
		}
		raws = append(raws, rr)
		if _, ok := idSeen[rr.commID]; !ok {
			idSeen[rr.commID] = struct{}{}
			commIDs = append(commIDs, rr.commID)
		}
	}
	rowsErr := rows.Err()
	rows.Close()
	release()
	if rowsErr != nil || len(raws) == 0 {
		return nil, nil
	}

	defs := loadCommendationDefsFromMetadata(ctx, r.pdb.Metadata, commIDs)
	result := make([]domain.HomeMatchCommendationRaw, 0, len(raws))
	for _, rr := range raws {
		d := defs[rr.commID]
		name := d.name
		if name == "" {
			name = rr.commID // dégradation : ID brut si la définition n'est pas seedée
		}
		result = append(result, domain.HomeMatchCommendationRaw{
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

type citationMappingMeta struct {
	display     string
	imagePath   string
	tierTargets string
	description string
}

func (r *CitationsRepo) loadCitationMappingMeta(ctx context.Context, norms []string) map[string]citationMappingMeta {
	result := make(map[string]citationMappingMeta, len(norms))
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
	enLocale := ctxkeys.Locale(ctx) == "en"
	for rows.Next() {
		var norm, display, displayEN, imagePath, tierTargets, description, descriptionEN string
		if err := rows.Scan(&norm, &display, &displayEN, &imagePath, &tierTargets, &description, &descriptionEN); err == nil {
			// Locale-aware (GH2-B2 + GH4) : sous UI EN, le nom anglais prime (fallback FR)
			// et la description EN (description_en, source commendations H5 officielles +
			// trad Infinite) prime. Si description_en est absente → nom seul (masquée) :
			// principe GH-5b « EN n'injecte jamais de FR ».
			if enLocale {
				if displayEN != "" {
					display = displayEN
				}
				description = descriptionEN
			}
			result[norm] = citationMappingMeta{
				display:     display,
				imagePath:   imagePath,
				tierTargets: tierTargets,
				description: description,
			}
		}
	}
	return result
}

// NOTE (campagne ART #23046) : WriteCitationsForMatch a été SUPPRIMÉ — c'était du
// dead code (aucun caller) portant un ON CONFLICT (match_id, citation_name_norm)
// désormais incompatible avec match_citations append-only (PK composite retirée).
// Le seul chemin d'écriture des citations est sync.writeCitations (génération).
