// Package duckdb — home_repo_skill_peak.go : skill peak Home (CSR/LUSR).
//
// Charge le meilleur rating CSR ou LUSR pour la home, avec gestion de la
// phase de placement (10 matchs par playlist_group côté LUSR, 10 matchs
// côté Microsoft pour CSR via player_csr_snapshots). Inclut le builder
// d'URL de badge de rang (utilisé aussi par career_repo / match_history).
//
// Sous-module de home_repo.go (split god-file 2026-05-21).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path"
	"strconv"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/assets/static"
	"levelup/go-api/internal/domain"
	titlepkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite"
)

// Labels rating_type (UPPERCASE pour cohérence DB : match_skill_rank.rating_type).
// canonical.RatingType{CSR,LUSR} reste lowercase pour l'API publique.
const (
	ratingTypeLUSR = "LUSR"
	ratingTypeCSR  = "CSR"
)

// Tier names canoniques Halo Infinite (PascalCase pour l'affichage UI).
// Utilisés par canonicalHomeSkillTierName + tests de classification rang.
const (
	tierBronze    = "Bronze"
	tierGold      = "Gold"
	tierDiamond   = "Diamond"
	tierOnyx      = "Onyx"
	placementTier = "Placement"
)

// peakRow : scratch interne pour la classification CSR/LUSR + best per group.
type peakRow struct {
	matchID       string
	playlistGroup string // "_unknown" si NULL en DB
	ratingValue   float64
	ratingType    string // raw msr.rating_type
	tier          string
	subTier       int
	tierLabel     string
	recency       sql.NullTime
}

// peakRegistryInfo : projection Phase B match_registry pour classification CSR/LUSR
// + date d'obtention du pic (start_time canonique du match).
type peakRegistryInfo struct {
	isRanked     bool
	playlistName string
	pairName     string
	startTime    sql.NullTime // start_time canonique (COALESCE utc, local AT TIME ZONE UTC)
}

// loadHomeSkillPeak lit le meilleur rating CSR ou LUSR pour la home, avec
// gestion de la phase de placement (10 matchs par playlist_group côté LUSR,
// 10 matchs côté Microsoft pour CSR via player_csr_snapshots).
//
// Comportement :
//   - CSR : priorité à player_csr_snapshots.alltime_value (officiel Waypoint).
//     Si vide, fallback sur le chemin Phase A/B (Q26ePeakPhaseAPlayer +
//     classification CSR/LUSR en Go via classifyPeakType).
//   - LUSR : chemin Phase A/B (Q26ePeakPhaseAPlayer + classifyPeakType) uniquement.
//   - En placement (placement_remaining > 0) : retourne un row avec
//     BadgeImageURL=unranked_(10-remaining).png et MeasurementMatchesRemaining
//     non-nil ; le front affichera "En placement" sans inventer.
//   - Matured (placement_remaining = 0) : retourne rating + tier badge habituel.
func (r *HomeRepo) loadHomeSkillPeak(ctx context.Context, ratingType string) *domain.HomeSkillPeakRow {
	if r == nil || r.pdb == nil || r.pdb.Player == nil {
		return nil
	}

	// Pour CSR, lire depuis player_csr_snapshots (alltime officiel Waypoint).
	// Fallback sur match_skill_rank si la table est vide ou absente.
	if ratingType == "CSR" {
		if peak := r.loadCSRAlltimePeak(ctx); peak != nil {
			return peak
		}
	}

	playerRows, err := r.loadPeakPhaseA(ctx)
	if err != nil {
		if isTableNotFoundErr(err) {
			slog.DebugContext(ctx, "loadHomeSkillPeak: match_skill_rank missing",
				"rating_type", ratingType, "xuid", r.pdb.XUID, "err", err)
			return nil
		}
		slog.WarnContext(ctx, "loadHomeSkillPeak: Phase A failed (silent drop)",
			"rating_type", ratingType, "xuid", r.pdb.XUID, "err", err)
		return nil
	}
	if len(playerRows) == 0 {
		return nil
	}
	matchIDs := make([]string, 0, len(playerRows))
	for _, pr := range playerRows {
		matchIDs = append(matchIDs, pr.matchID)
	}
	registryByMatch := r.loadPeakPhaseB(ctx, matchIDs)
	return r.assemblePeak(ctx, playerRows, registryByMatch, ratingType)
}

// loadPeakPhaseA : query match_skill_rank sur pdb.Player (player-only).
// QueryRecovered (Phase 5 ART) : retry après Reopen si la handle player DB
// est invalidée (FATAL DuckDB).
func (r *HomeRepo) loadPeakPhaseA(ctx context.Context) ([]peakRow, error) {
	rows, err := r.pdb.Player.QueryRecovered(ctx, Q26ePeakPhaseAPlayer)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []peakRow
	for rows.Next() {
		var (
			matchID    string
			playlist   sql.NullString
			rating     sql.NullFloat64
			ratingType sql.NullString
			tier       sql.NullString
			subTier    sql.NullInt16
			tierLabel  sql.NullString
			recency    sql.NullTime
		)
		if err := rows.Scan(&matchID, &playlist, &rating, &ratingType, &tier, &subTier, &tierLabel, &recency); err != nil {
			return nil, err
		}
		if !rating.Valid {
			continue
		}
		pr := peakRow{
			matchID:       matchID,
			playlistGroup: "_unknown",
			ratingValue:   rating.Float64,
			ratingType:    optionalNullStringValue(ratingType),
			tier:          optionalNullStringValue(tier),
			subTier:       optionalNullInt16Value(subTier),
			tierLabel:     optionalNullStringValue(tierLabel),
			recency:       recency,
		}
		if playlist.Valid && strings.TrimSpace(playlist.String) != "" {
			pr.playlistGroup = playlist.String
		}
		out = append(out, pr)
	}
	return out, rows.Err()
}

// loadPeakPhaseB : enrichit avec match_registry via SharedReader.
func (r *HomeRepo) loadPeakPhaseB(ctx context.Context, matchIDs []string) map[string]peakRegistryInfo {
	out := make(map[string]peakRegistryInfo, len(matchIDs))
	if len(matchIDs) == 0 || r.pdb.SharedReader == nil {
		return out
	}
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "loadHomeSkillPeak: Phase B SharedReader unavailable",
			"xuid", r.pdb.XUID, "err", err)
		return out
	}
	defer release()

	query := fmt.Sprintf(Q26ePeakPhaseBRegistryTpl, Placeholders(len(matchIDs)))
	rows, err := sharedDB.QueryContext(ctx, query, ToAnySlice(matchIDs)...)
	if err != nil {
		slog.WarnContext(ctx, "loadHomeSkillPeak: Phase B query failed",
			"xuid", r.pdb.XUID, "err", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var (
			matchID      string
			isRanked     bool
			playlistName string
			pairName     string
			startTime    sql.NullTime
		)
		if err := rows.Scan(&matchID, &isRanked, &playlistName, &pairName, &startTime); err != nil {
			continue
		}
		out[matchID] = peakRegistryInfo{isRanked: isRanked, playlistName: playlistName, pairName: pairName, startTime: startTime}
	}
	return out
}

// assemblePeak : filtre par effective_type, groupe, sélectionne le best matured.
// Phase 6 : threshold paramétré (CSR=lookup season ou default, LUSR=10). Pipeline
// filtre+regroupement+sélection peak avec branches effective_type (CSR/LUSR/derived)
// + threshold (season vs default vs LUSR=10).
//
//nolint:gocyclo // cohésion du flow filtre→group→select, splitter casse la lisibilité.
func (r *HomeRepo) assemblePeak(ctx context.Context, playerRows []peakRow, registryByMatch map[string]peakRegistryInfo, ratingType string) *domain.HomeSkillPeakRow {
	want := strings.ToUpper(strings.TrimSpace(ratingType))
	// LUSR garde son seuil interne de 10 (algorithme local). CSR utilise la
	// saison courante du HomeRepo (configurée via WithCSRThresholds) ou le
	// default S3+ (=5) si non câblé.
	threshold := 10
	if want == "CSR" {
		threshold = r.csrThreshold(r.currentCSRSID)
	}

	type groupBest struct {
		row        peakRow
		matchCount int
	}
	byGroup := make(map[string]*groupBest)
	for _, pr := range playerRows {
		if classifyPeakType(pr, registryByMatch) != want {
			continue
		}
		gb, ok := byGroup[pr.playlistGroup]
		if !ok {
			gb = &groupBest{row: pr}
			byGroup[pr.playlistGroup] = gb
		}
		gb.matchCount++
		if isBetterPeak(pr, gb.row) {
			gb.row = pr
		}
	}
	if len(byGroup) == 0 {
		return nil
	}

	var chosen *groupBest
	for _, gb := range byGroup {
		switch {
		case chosen == nil:
			chosen = gb
		case gb.matchCount >= threshold && chosen.matchCount < threshold:
			chosen = gb
		case (gb.matchCount >= threshold) == (chosen.matchCount >= threshold):
			if gb.row.ratingValue > chosen.row.ratingValue {
				chosen = gb
			}
		}
	}
	remaining := threshold - chosen.matchCount
	if remaining < 0 {
		remaining = 0
	}

	peak := &domain.HomeSkillPeakRow{RatingValue: chosen.row.ratingValue}
	totalCopy := threshold
	peak.PlacementTotal = &totalCopy
	if remaining > 0 {
		peak.BadgeImageURL = buildHomeSkillPeakBadgeURLForThreshold("", "", 0, r.titleSlug(), remaining, threshold)
		remCopy := remaining
		peak.MeasurementMatchesRemaining = &remCopy
		return peak
	}
	peak.BadgeImageURL = buildHomeSkillPeakBadgeURLForThreshold(chosen.row.tier, chosen.row.tierLabel, chosen.row.subTier, r.titleSlug(), 0, threshold)
	// Tier+subTier bruts conservés pour la bande ordinale (analysis.SkillTierBand).
	peak.Tier = chosen.row.tier
	peak.SubTier = chosen.row.subTier
	if strings.TrimSpace(chosen.row.tierLabel) != "" {
		peak.TierLabel = stringPtr(chosen.row.tierLabel)
	}
	// Date d'obtention du pic : start_time canonique du match retenu (Phase B).
	// Dégradation gracieuse si le match est absent du registry (ex. m2 legacy
	// sans ligne registry) : champ laissé nil + trace Debug, jamais d'erreur.
	if info, ok := registryByMatch[chosen.row.matchID]; ok && info.startTime.Valid {
		t := info.startTime.Time
		peak.PeakAchievedAt = &t
	} else {
		slog.DebugContext(ctx, "assemblePeak: date d'obtention du pic indisponible (match hors registry)",
			"rating_type", want, "match_id", chosen.row.matchID, "xuid", r.pdb.XUID)
	}
	zero := 0
	peak.MeasurementMatchesRemaining = &zero
	return peak
}

// classifyPeakType : heuristique CSR/LUSR (is_ranked, sinon fallback si
// playlist_name/pair_name contient 'ranked'). Portée en Go depuis l'ancienne
// query monolithique Q26e (supprimée, split en Phase A/B).
func classifyPeakType(pr peakRow, registryByMatch map[string]peakRegistryInfo) string {
	if info, ok := registryByMatch[pr.matchID]; ok {
		if info.isRanked ||
			strings.Contains(strings.ToLower(info.playlistName), "ranked") ||
			strings.Contains(strings.ToLower(info.pairName), "ranked") {
			return ratingTypeCSR
		}
		return ratingTypeLUSR
	}
	rt := strings.ToUpper(strings.TrimSpace(pr.ratingType))
	if rt == ratingTypeCSR {
		return ratingTypeCSR
	}
	return ratingTypeLUSR
}

// isBetterPeak : ordre rating DESC, recency DESC, sub_tier DESC, match_id DESC.
func isBetterPeak(candidate, current peakRow) bool {
	if candidate.ratingValue != current.ratingValue {
		return candidate.ratingValue > current.ratingValue
	}
	if candidate.recency.Valid && current.recency.Valid && !candidate.recency.Time.Equal(current.recency.Time) {
		return candidate.recency.Time.After(current.recency.Time)
	}
	if candidate.recency.Valid != current.recency.Valid {
		return candidate.recency.Valid
	}
	if candidate.subTier != current.subTier {
		return candidate.subTier > current.subTier
	}
	return candidate.matchID > current.matchID
}

// loadCSRAlltimePeak lit le meilleur CSR alltime depuis player_csr_snapshots.
// Si aucun alltime_value > 0 n'existe (joueur en cours de placement sur sa
// première playlist ranked), on rend un row placement avec
// BadgeImageURL=unranked_N.png basé sur le MIN(current_measurement_remaining)
// (la playlist la plus avancée dans son placement) pour que la home affiche
// "En placement N/10" au lieu de "Aucune partie classée".
func (r *HomeRepo) loadCSRAlltimePeak(ctx context.Context) *domain.HomeSkillPeakRow {
	// Phase 6 : threshold de la saison courante pour les calculs CSR.
	threshold := r.csrThreshold(r.currentCSRSID)

	// Sélection du pic EN GO via l'ordinal canonique analysis.CSRTierOrdinal
	// (palier → sous-palier → valeur de départage). Q26csrAlltimePeak renvoie tous
	// les snapshots éligibles (palier renseigné OU valeur > 0) → couvre les titres
	// tier-only (valeur=0, ex. H5 "Diamant V") sans dupliquer l'ordre des paliers.
	if bestTier, bestSub, bestVal, ok := r.pickBestCSRAlltime(ctx); ok {
		peak := &domain.HomeSkillPeakRow{RatingValue: bestVal}
		// Tier+subTier bruts conservés pour la bande ordinale (analysis.SkillTierBand).
		peak.Tier = bestTier
		peak.SubTier = bestSub
		peak.BadgeImageURL = buildHomeSkillPeakBadgeURLForThreshold(bestTier, "", bestSub, r.titleSlug(), 0, threshold)
		// Palier all-time CSR : libellé FR + sous-palier romain ("Diamant III"),
		// au lieu du tier brut anglais sans sous-palier. FR-first comme tous les
		// libellés CSR (sync.formatCSRTierLabel). Partagé Home + Explorer.
		if bestTier != "" {
			var subPtr *int
			if bestSub >= 1 && bestSub <= 6 {
				s := bestSub
				subPtr = &s
			}
			if lbl := analysis.BuildCSRTierLabelFromEN(bestTier, subPtr, true); lbl != nil {
				peak.TierLabel = lbl
			} else {
				peak.TierLabel = stringPtr(bestTier)
			}
		}
		zero := 0
		peak.MeasurementMatchesRemaining = &zero
		totalCopy := threshold
		peak.PlacementTotal = &totalCopy
		// Date d'obtention du pic : corrélée via l'historique par-match
		// match_csrs (l'API Waypoint alltime_value n'expose PAS la date). On
		// retrouve la PREMIÈRE atteinte du pic (tier+sub_tier[+valeur]) dans
		// match_csrs_latest ⋈ match_registry. Reste nil seulement si aucun match
		// ne corrèle (pic atteint AVANT notre tracking) ou shared indisponible —
		// dégradation gracieuse, jamais d'erreur propagée (best-effort).
		peak.PeakAchievedAt = r.loadCSRPeakDate(ctx, bestTier, bestSub, bestVal)
		return peak
	}

	// Pas d'alltime : tenter de récupérer l'état de placement le plus avancé.
	// MIN(current_measurement_remaining) = playlist la plus proche de la fin
	// du placement (threshold → 0). Si pas de snapshot du tout : retourner nil
	// pour laisser Q26e fallback prendre la suite.
	//
	// NOTE : à terme, on pourrait lire le season_id ici aussi pour appliquer le
	// threshold de cette saison-là spécifiquement. Pour l'instant on prend le
	// threshold de la saison courante — acceptable dans 99% des cas (joueur en
	// placement = saison récente).
	var minRemaining sql.NullInt32
	rows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, `
		SELECT MIN(current_measurement_remaining)
		FROM player_csr_snapshots_latest
		WHERE current_measurement_remaining IS NOT NULL
		  AND current_measurement_remaining > 0
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	if err := rows.Scan(&minRemaining); err != nil || !minRemaining.Valid {
		return nil
	}
	remaining := int(minRemaining.Int32)
	peak := &domain.HomeSkillPeakRow{RatingValue: 0}
	peak.BadgeImageURL = buildHomeSkillPeakBadgeURLForThreshold("", "", 0, r.titleSlug(), remaining, threshold)
	peak.MeasurementMatchesRemaining = &remaining
	totalCopy := threshold
	peak.PlacementTotal = &totalCopy
	return peak
}

// pickBestCSRAlltime parcourt les snapshots candidats (Q26csrAlltimePeak) et
// retourne le meilleur pic via l'ordinal canonique analysis.CSRTierOrdinal —
// départage : palier, puis sous-palier, puis valeur. ok=false si aucun candidat.
// Title-agnostic : un titre tier-only (valeur=0) reste classé par son palier ;
// un titre à valeur (Infinite) départage les ex-aequo de palier/sous-palier par
// la valeur. Pas de CASE SQL : l'ordre des paliers a UNE seule source (Go).
func (r *HomeRepo) pickBestCSRAlltime(ctx context.Context) (tier string, sub int, val float64, ok bool) {
	rows, err := r.pdb.ReadDB().QueryRecovered(ctx, Q26csrAlltimePeak)
	if err != nil {
		return "", 0, 0, false
	}
	defer func() { _ = rows.Close() }()
	bestOrd := -1
	for rows.Next() {
		var v sql.NullFloat64
		var t sql.NullString
		var st sql.NullInt16
		if err := rows.Scan(&v, &t, &st); err != nil {
			continue
		}
		cTier := optionalNullStringValue(t)
		cSub := optionalNullInt16Value(st)
		cVal := 0.0
		if v.Valid {
			cVal = v.Float64
		}
		cOrd := analysis.CSRTierOrdinal(cTier)
		better := !ok ||
			cOrd > bestOrd ||
			(cOrd == bestOrd && cSub > sub) ||
			(cOrd == bestOrd && cSub == sub && cVal > val)
		if better {
			tier, sub, val, bestOrd, ok = cTier, cSub, cVal, cOrd, true
		}
	}
	return tier, sub, val, ok
}

// csrRatingMatchEpsilon : tolérance d'égalité sur rating_value lors de la
// corrélation du pic all-time. rating_value (FLOAT) et bestVal portent des
// valeurs entières CSR ; une demi-unité couvre l'imprécision flottante sans
// jamais chevaucher deux paliers de valeur distincts.
const csrRatingMatchEpsilon = 0.5

// loadCSRPeakDate corrèle le pic CSR all-time (tier+sub_tier[+valeur]) avec
// l'historique par-match match_csrs_latest (DB shared) pour retrouver la
// PREMIÈRE date d'atteinte du pic — MIN(start_time canonique du match). L'API
// Waypoint (alltime_value) n'expose pas cette date : la corrélation est la
// seule voie.
//
// Title-agnostic : un titre tier-only (val<=0, ex. Halo 5 dont l'API ne fournit
// pas la valeur numérique) corrèle sur tier+sub_tier seuls ; un titre à valeur
// (Infinite) resserre en plus sur rating_value (± csrRatingMatchEpsilon).
//
// Best-effort : retourne nil (sans erreur propagée) si le shared reader est
// indisponible, si le scan échoue, ou si aucun match ne corrèle (pic atteint
// AVANT le début de notre tracking). Chaque cas laisse une trace Debug.
//
// Accès shared conforme au modèle mono-process (ADR 0016) : le site d'appel
// (loadCSRAlltimePeak) ne tient AUCUN reader shared ouvert (pickBestCSRAlltime
// et la requête placement lisent la player DB via ReadDB), donc on ouvre puis
// libère ici sans imbrication.
func (r *HomeRepo) loadCSRPeakDate(ctx context.Context, tier string, sub int, val float64) *time.Time {
	if r == nil || r.pdb == nil || strings.TrimSpace(tier) == "" {
		return nil
	}
	sharedDB, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.DebugContext(ctx, "loadCSRPeakDate: shared reader indisponible (date du pic non corrélée)",
			"tier", tier, "sub_tier", sub, "xuid", r.pdb.XUID, "err", err)
		return nil
	}
	defer release()

	// MIN(start_time canonique) = première atteinte du pic. Lecture via la vue
	// _latest (règle ART n°2). Timezone canonique via StartTimeCanonicalSQL.
	query := `
		SELECT MIN(` + StartTimeCanonicalSQL("mr") + `)
		FROM match_csrs_latest mc
		JOIN match_registry mr ON mc.match_id = mr.match_id
		WHERE mc.xuid = ?
		  AND mc.tier = ?
		  AND mc.sub_tier = ?`
	args := []any{r.pdb.XUID, tier, sub}
	if val > 0 {
		query += `
		  AND ABS(mc.rating_value - ?) < ?`
		args = append(args, val, csrRatingMatchEpsilon)
	}

	var achieved sql.NullTime
	if err := sharedDB.QueryRowContext(ctx, query, args...).Scan(&achieved); err != nil {
		slog.DebugContext(ctx, "loadCSRPeakDate: corrélation de la date du pic échouée (scan)",
			"tier", tier, "sub_tier", sub, "xuid", r.pdb.XUID, "err", err)
		return nil
	}
	if !achieved.Valid {
		slog.DebugContext(ctx, "loadCSRPeakDate: aucun match ne corrèle le pic CSR all-time (pic pré-tracking)",
			"tier", tier, "sub_tier", sub, "xuid", r.pdb.XUID)
		return nil
	}
	t := achieved.Time
	return &t
}

// unrankedBadgeURL retourne l'URL du badge unranked_N.png pour un placement à
// seuil 10 (compat historique). Préfèrer unrankedBadgeURLForThreshold pour les
// nouveaux callers qui veulent supporter le seuil dynamique par saison.
func unrankedBadgeURL(placementsCompleted int, titleSlug string) *string {
	return unrankedBadgeURLForThreshold(placementsCompleted, 10, titleSlug)
}

// unrankedBadgeURLForThreshold retourne l'URL du badge unranked en mappant
// proportionnellement la progression sur les 10 images disponibles
// (unranked_0.png .. unranked_9.png).
//
// Phase 6 du plan pipeline CSR : depuis Season 3 (2023-03-07) Halo utilise
// un seuil 5 au lieu de 10. On recycle les images existantes via un mapping
// régulier :
//
//	threshold=10 : completed * 10 / 10 = identité (0,1,2,3,4,5,6,7,8,9)
//	threshold=5  : completed * 10 / 5  = 0,2,4,6,8 (5 images utilisées)
//
// N est ensuite clampé [0, 9] pour les bornes (completed négatif ou ≥ threshold).
//
// Résolution du titre : les badges unranked_N.png sont des visuels GÉNÉRIQUES LevelUp
// (gris, N/seuil de placement), PAS des insignes CSR title-specific. Ils ne sont livrés
// que sous le dossier du titre par défaut (static/ranks/halo_infinite/). Un titre
// additionnel (ex. halo_5) n'a pas de dossier ranks/ propre : composer l'URL avec son
// slug produisait /static/ranks/halo_5/unranked_0.png → 404 (file-server nu, sans
// fallback) → le front affichait « ? » sur les playlists « Non classé » H5. On résout
// donc toujours vers le titre par défaut — cohérent avec le fallback static HINF des
// insignes CSR matured (buildHomeSkillPeakBadgeURLForThreshold ci-dessus).
//
// builder d'insignes matured ; les badges unranked sont des assets partagés → toujours
// résolus sous le titre par défaut (voir doc ci-dessus).
//
//nolint:unparam // titleSlug conservé pour la symétrie de signature title-aware avec le
func unrankedBadgeURLForThreshold(placementsCompleted, threshold int, titleSlug string) *string {
	if threshold <= 0 {
		threshold = 10 // garde-fou
	}
	// Mapping proportionnel : completed * 10 / threshold.
	n := (placementsCompleted * 10) / threshold
	if n < 0 {
		n = 0
	}
	if n > 9 {
		n = 9
	}
	url := static.URL(static.KindCSRRank, titlepkg.DefaultSlug, fmt.Sprintf("unranked_%d", n), ".png")
	return &url
}

// csrBadgeResolver (optionnel, posé au boot) résout l'URL d'insigne CSR d'un titre
// ADDITIONNEL par (titleSlug, tier EN « Diamond/Onyx… », sous-palier 1-6) depuis sa
// metadata (table csr_designations, URLs CDN officielles). Retourne "" pour le titre
// par défaut (Halo Infinite) → le chemin static HINF reste la source. Mécanisme
// global (zéro changement de signature des 4 call-sites du builder).
var csrBadgeResolver func(titleSlug, tier string, subTier int) string

// SetCSRBadgeResolver pose le résolveur d'insignes CSR par titre. Idempotent (boot).
func SetCSRBadgeResolver(f func(titleSlug, tier string, subTier int) string) {
	csrBadgeResolver = f
}

// buildHomeSkillPeakBadgeURL construit l'URL du badge de rang (compat seuil 10).
// Wrapper de buildHomeSkillPeakBadgeURLForThreshold. Préfèrer la version
// "ForThreshold" pour les nouveaux callers conscients du seuil dynamique.
func buildHomeSkillPeakBadgeURL(tier string, tierLabel string, subTier int, titleSlug string, measurementMatchesRemaining int) *string {
	return buildHomeSkillPeakBadgeURLForThreshold(tier, tierLabel, subTier, titleSlug, measurementMatchesRemaining, 10)
}

// TitleSkillBadgeURL résout l'URL du badge CSR de façon title-aware : pour un
// titre additionnel (ex. Halo 5), l'URL CDN officielle issue de csr_designations
// via csrBadgeResolver ; sinon le chemin static HINF (titre par défaut). C'est le
// SEUL endroit où le slug pilote l'URL — injecté dans le package analysis (pur)
// par la couche boot/service qui connaît le titre.
//
// tierEN : tier capitalisé ("Bronze".."Diamond","Onyx"). subTier : 0 pour Onyx,
// 1..6 sinon. Retourne "" si aucune URL constructible (analysis → SkillRankImageURL
// laissé vide, dégradation gracieuse).
func TitleSkillBadgeURL(slug, tierEN string, subTier int) string {
	u := buildHomeSkillPeakBadgeURL(tierEN, "", subTier, slug, 0)
	if u == nil {
		return ""
	}
	return *u
}

// buildHomeSkillPeakBadgeURLForThreshold construit l'URL du badge avec seuil
// dynamique pour le calcul de l'image placement. Phase 6 du plan pipeline CSR.
func buildHomeSkillPeakBadgeURLForThreshold(tier string, tierLabel string, subTier int, titleSlug string, measurementMatchesRemaining, threshold int) *string {
	normalizedTier, normalizedSubTier := normalizeHomeSkillPeakBadgeParts(tier, tierLabel, subTier)
	if normalizedTier == "" {
		if measurementMatchesRemaining > 0 {
			if threshold <= 0 {
				threshold = 10
			}
			completed := threshold - measurementMatchesRemaining
			return unrankedBadgeURLForThreshold(completed, threshold, titleSlug)
		}
		return nil
	}
	// Titres additionnels (ex. Halo 5) : insigne CSR depuis leur metadata
	// (csr_designations, URLs CDN officielles). Additif — le résolveur renvoie "" pour
	// le titre par défaut (HINF) → chemin static HINF ci-dessous strictement inchangé.
	if csrBadgeResolver != nil {
		sub := normalizedSubTier
		if strings.EqualFold(normalizedTier, "Onyx") {
			sub = 1 // Onyx = palier unique (csr_designations tier_id=1)
		}
		if u := csrBadgeResolver(titleSlug, normalizedTier, sub); u != "" {
			return &u
		}
	}
	// P5.4 (gap #9, ADR 0012) : déléguer à halo_infinite.AssetURLAdapter pour
	// le format `120px-HINF-CSR_*` (Halo-only). Évite la duplication du format.
	adapter := halo_infinite.NewAssetURLAdapter()
	var rawURL string
	if strings.EqualFold(normalizedTier, "Onyx") {
		rawURL = adapter.CSRRankImageURLOnyx()
	} else {
		if normalizedSubTier < 1 || normalizedSubTier > 6 {
			return nil
		}
		rawURL = adapter.CSRRankImageURL(normalizedTier, normalizedSubTier)
	}
	if rawURL == "" {
		return nil
	}
	// Quand titleSlug != adapter.TitleSlug() (LUSR cross-titre), recomposer
	// le path sans slug. L'adapter renvoie /static/ranks/<slug>/<id>.png.
	if titleSlug == "" {
		// Strip /static/ranks/halo_infinite/ → /static/ranks/
		prefix := static.MountPoint + "/" + static.Folder(static.KindCSRRank) + "/" + adapter.TitleSlug() + "/"
		if strings.HasPrefix(rawURL, prefix) {
			rawURL = path.Join(static.MountPoint, static.Folder(static.KindCSRRank), strings.TrimPrefix(rawURL, prefix))
		}
	}
	return &rawURL
}

func normalizeHomeSkillPeakBadgeParts(tier string, tierLabel string, subTier int) (string, int) {
	normalizedTier := canonicalHomeSkillTierName(tier)
	derivedSubTier := subTier
	if strings.TrimSpace(tierLabel) != "" {
		parts := strings.Fields(strings.TrimSpace(tierLabel))
		if normalizedTier == "" && len(parts) > 0 {
			normalizedTier = canonicalHomeSkillTierName(parts[0])
		}
		if derivedSubTier <= 0 && len(parts) > 1 {
			derivedSubTier = parseHomeSkillPeakSubTier(parts[len(parts)-1])
		}
	}
	return normalizedTier, derivedSubTier
}

func canonicalHomeSkillTierName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "bronze":
		return tierBronze
	case "silver":
		return "Silver"
	case "gold":
		return tierGold
	case "platinum":
		return "Platinum"
	case "diamond":
		return tierDiamond
	case "onyx":
		return tierOnyx
	default:
		return ""
	}
}

func parseHomeSkillPeakSubTier(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	if numeric, err := strconv.Atoi(trimmed); err == nil {
		return numeric
	}
	switch strings.ToUpper(trimmed) {
	case "I":
		return 1
	case "II":
		return 2
	case "III":
		return 3
	case "IV":
		return 4
	case "V":
		return 5
	case "VI":
		return 6
	default:
		return 0
	}
}
