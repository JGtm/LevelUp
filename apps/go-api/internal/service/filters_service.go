// Package service — FiltersService : résolution du contexte de filtres.
//
// Port Go du filter_service.py Python (apps/api/app/services/filter_service.py).
// Les données brutes sont chargées par le repo ; ce service applique la logique pure.
package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// Constantes expérience (ordre affiché dans l'UI).
var experienceLabels = []string{expTypePVPUnranked, expTypePVPRanked, expTypePVE}

// expTypeLabelEN mappe la VALUE canonique FR d'un type d'expérience vers son
// libellé d'affichage EN. La VALUE reste FR par contrat : la cascade front
// (EXPERIENCE_TO_CASCADE) et les matchers substring (applyExperienceFilter ici,
// experienceCounts dans useLocalFilterBar.tsx) matchent sur la VALUE FR — seul le
// LABEL est localisé (GH5-2, miroir de GH3-1 saisons). Libellés EN alignés sur le
// vocabulaire « Ranked »/« Unranked » des manifests (session/synthesis/career),
// en conservant la distinction PvP/PvE des 3 options.
var expTypeLabelEN = map[string]string{
	expTypePVPRanked:   "Ranked PvP",
	expTypePVPUnranked: "Unranked PvP",
	expTypePVE:         "PvE",
}

// experienceLabelForLocale retourne le libellé d'affichage d'une VALUE
// d'expérience dans la locale de requête. FR (défaut) = la value elle-même
// (canonique) ; EN = le libellé mappé (fallback sur la value si non mappée).
func experienceLabelForLocale(value, locale string) string {
	if strings.EqualFold(locale, "en") {
		if en, ok := expTypeLabelEN[value]; ok {
			return en
		}
	}
	return value
}

// localizeExperienceOptions projette le LABEL des options d'expérience vers la
// locale de requête en laissant la VALUE FR intacte (contrat cascade/substring).
// Appelé UNE fois au point d'entrée ctx-aware (Resolve) : la couche pure
// (ResolveFiltersFromRows) produit le canonique FR, le service le localise. GH5-2.
func localizeExperienceOptions(opts []domain.LabelValue, locale string) {
	for i := range opts {
		opts[i].Label = experienceLabelForLocale(opts[i].Value, locale)
	}
}

// experienceTypeOptionsForLocale construit les options d'expérience (LabelValue) à
// partir des VALUES canoniques FR distinctes (computeExplorerAvailableOptions) : LABEL
// localisé, VALUE FR intacte. Source UNIQUE partagée avec l'Omnibar (GH5-2) — réutilise
// experienceLabelForLocale (mêmes libellés EN, ZÉRO duplication). CONTRAT : la VALUE FR
// est la clé de filtre (req.ExperienceTypes → filterByExplorerExperienceTypes match
// exact) et alimente la cascade FR-hardcodée front (ExplorerPage.tsx rankedContext) :
// ne JAMAIS la localiser. Surface Explorer/Historique, miroir GH5-2 (GH6-1).
func experienceTypeOptionsForLocale(values []string, locale string) []domain.LabelValue {
	if len(values) == 0 {
		return nil
	}
	opts := make([]domain.LabelValue, len(values))
	for i, v := range values {
		opts[i] = domain.LabelValue{Label: experienceLabelForLocale(v, locale), Value: v}
	}
	return opts
}

// FiltersService calcule FilterContextResolved depuis les données du repo.
type FiltersService struct {
	repo      port.FiltersRepository
	titleSlug string
	catalog   seasonsCatalogLoader // optionnel : nil → aucun SeasonCount renvoyé
}

// NewFiltersService crée un FiltersService.
func NewFiltersService(repo port.FiltersRepository) *FiltersService {
	return &FiltersService{repo: repo}
}

// WithSeasonsCatalog injecte le résolveur unifié des saisons (TOML + DB +
// lazy fetch live). Source des SeasonCounts du popover SaisonPill côté
// frontend.
//
// Le titleSlug est nécessaire car le catalog est title-scoped (il faut
// passer le bon titleID au resolver pour que la lookup DB et le fetch live
// ciblent les bonnes données).
//
// Si non appelé → pas de season counts (dégradation gracieuse, le frontend
// affiche les saisons sans compteur ni folding).
func (s *FiltersService) WithSeasonsCatalog(titleSlug string, catalog *SeasonsCatalog) *FiltersService {
	s.titleSlug = titleSlug
	if catalog != nil { // garde concret fiable — évite le piège interface typed-nil
		s.catalog = catalog
	}
	return s
}

// Resolve charge les matchs du joueur et retourne le contexte résolu.
func (s *FiltersService) Resolve(
	ctx context.Context,
	input domain.FilterContextInput,
) (domain.FilterContextResolved, error) {
	rows, err := s.repo.LoadMatchesForFilters(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "load matches for filters", "err", err)
		return domain.FilterContextResolved{}, err
	}
	resolved := ResolveFiltersFromRows(rows, input)
	// Localise le LABEL des options d'expérience vers la locale de requête
	// (Value FR conservée — contrat cascade/substring). C'est le seul chemin prod
	// qui surface ces options à l'UI ; la couche pure reste canonique FR. GH5-2.
	localizeExperienceOptions(resolved.AvailableOptions.ExperienceTypes, ctxkeys.Locale(ctx))
	if s.catalog != nil && s.titleSlug != "" {
		// Le match_context a déjà été appliqué dans ResolveFiltersFromRowsAt.
		// Pour les SeasonCounts on veut le même périmètre : on ré-applique
		// match_context à rows et on calcule sur la base post-context.
		// Le catalog peut déclencher un fetch live + persist si la DB est
		// vide — best-effort, échec gracieux vers TOML statique.
		catalog := s.catalog.Load(ctx, s.titleSlug)
		if len(catalog) > 0 {
			seasonRows := applyMatchContextFilter(rows, input.MatchContext)
			windows := SeasonWindowsFromCatalog(catalog)
			resolved.SeasonCounts = BuildSeasonCounts(seasonRows, resolved.Effective.Cascade, windows)
		}
	}
	slog.DebugContext(ctx, "filters resolved",
		"rows_in", resolved.Counts.TotalMatchesBeforeFilters,
		"rows_out", resolved.Counts.TotalMatchesAfterFilters,
		"match_context", input.MatchContext,
		"filter_mode", input.FilterMode,
		"season_counts", len(resolved.SeasonCounts),
	)
	return resolved, nil
}

// ResolveMatchIDs charge les matchs du joueur et retourne, pour la sélection
// `input`, la liste ordonnée (start_time DESC) des match_id. Réutilise le même
// pipeline de filtrage que Resolve (donc match_context/sessions/cascade), ce qui
// garantit que le parcours "Voir les matchs" reste dans le périmètre — y compris
// solo/squad, que /neighbors ne sait pas filtrer.
func (s *FiltersService) ResolveMatchIDs(
	ctx context.Context,
	input domain.FilterContextInput,
) ([]string, error) {
	rows, err := s.repo.LoadMatchesForFilters(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "load matches for filter match-ids", "err", err)
		return nil, err
	}
	ids := FilteredMatchIDs(rows, input)
	slog.DebugContext(ctx, "filters match-ids resolved",
		"match_context", input.MatchContext,
		"filter_mode", input.FilterMode,
		"count", len(ids),
	)
	return ids, nil
}

// ResolveFiltersFromRows est la fonction pure testable sans repo.
// Utilise time.Now() pour les period presets ; pour la testabilité
// précise, voir ResolveFiltersFromRowsAt.
func ResolveFiltersFromRows(
	rows []domain.FilterMatchRow,
	input domain.FilterContextInput,
) domain.FilterContextResolved {
	return ResolveFiltersFromRowsAt(rows, input, time.Now())
}

// ResolveFiltersFromRowsAt est la fonction pure testable avec injection de l'heure
// courante (pour les comptes de presets de période).
//
// Phase I plan catalogue : applique le filtre `match_context` (solo/squad/all)
// avant tout autre traitement. C'est un filtre transversal qui restreint la
// population de matchs selon le contexte de la page appelante.
func ResolveFiltersFromRowsAt(
	rows []domain.FilterMatchRow,
	input domain.FilterContextInput,
	now time.Time,
) domain.FilterContextResolved {
	// Phase I : filtre solo/squad appliqué tôt pour réduire la population
	// avant tous les calculs (sessions, cascade, etc.).
	rows = applyMatchContextFilter(rows, input.MatchContext)

	totalBefore := len(rows)
	effective := normalizeInput(input)

	if totalBefore == 0 {
		return emptyResolved(effective, buildSessionOptions(rows, effective.Cascade))
	}

	// Migre les valeurs cascade stockées en anglais vers les noms FR.
	effective = migrateCascadeFromRows(rows, effective)

	// Sessions enrichies (count + count post-cascade) — calculées sur rows
	// post-match_context pour rester indépendantes du filter_mode courant.
	sessionOpts := buildSessionOptions(rows, effective.Cascade)

	// Period presets — counts si on switchait en mode period sur ce preset
	// (indépendants du filter_mode courant et du filtre sessions).
	periodPresetCounts := buildPeriodPresetCounts(rows, effective.Cascade, now)

	// Filtre temporel (session OU période) puis cascade.
	temporal, filtered := splitTemporalFiltered(rows, effective)

	// Options disponibles (avant cascade) avec counts OR.
	available := buildAvailableOptions(temporal, effective.Cascade)

	return domain.FilterContextResolved{
		Effective:        effective,
		AvailableOptions: available,
		SessionOptions:   sessionOpts,
		Counts: domain.FilterCounts{
			TotalMatchesBeforeFilters: totalBefore,
			TotalMatchesAfterFilters:  len(filtered),
		},
		PeriodPresets: periodPresetCounts,
	}
}

// migrateCascadeFromRows migre les valeurs cascade stockées en anglais vers les
// noms FR (modes, cartes, playlists), à partir des traductions présentes dans
// les rows. Transparent si aucune traduction n'est disponible (map vide).
func migrateCascadeFromRows(rows []domain.FilterMatchRow, effective domain.FilterContextInput) domain.FilterContextInput {
	if len(effective.Cascade.Modes) > 0 {
		if tr := buildModeTranslationMap(rows); len(tr) > 0 {
			effective.Cascade.Modes = migrateCascadeValues(effective.Cascade.Modes, tr)
		}
	}
	if len(effective.Cascade.Maps) > 0 {
		if tr := buildMapTranslationMap(rows); len(tr) > 0 {
			effective.Cascade.Maps = migrateCascadeValues(effective.Cascade.Maps, tr)
		}
	}
	if len(effective.Cascade.Playlists) > 0 {
		if tr := buildPlaylistTranslationMap(rows); len(tr) > 0 {
			effective.Cascade.Playlists = migrateCascadeValues(effective.Cascade.Playlists, tr)
		}
	}
	return effective
}

// splitTemporalFiltered applique le filtre temporel (session si au moins une
// session pickée, sinon période) puis la cascade. Retourne (temporal pré-cascade
// pour available_options, filtered post-cascade). Symétrie avec applyAllFilters
// (match_history_service.go) : applySessionFilter dès qu'une session est pickée,
// peu importe filter_mode.
func splitTemporalFiltered(rows []domain.FilterMatchRow, effective domain.FilterContextInput) (temporal, filtered []domain.FilterMatchRow) {
	if hasPickedSessions(effective.Sessions) {
		temporal = applySessionFilter(rows, effective.Sessions)
	} else {
		temporal = applyPeriodFilter(rows, effective.Period)
	}
	return temporal, applyCascadeFilter(temporal, effective.Cascade)
}

// FilteredMatchIDs retourne les match_id de la sélection — mêmes filtres que
// ResolveFiltersFromRowsAt (match_context → temporel → cascade) — ordonnés par
// start_time DESC (récent d'abord, comme la chronologie de navigation). Les rows
// sans start_time sont reléguées en fin. Alimente le bouton "Voir les matchs" :
// la liste explicite permet un parcours prev/next exact, là où /neighbors
// (shared-only) ne sait pas filtrer solo/squad ni les sessions (player DB).
func FilteredMatchIDs(rows []domain.FilterMatchRow, input domain.FilterContextInput) []string {
	rows = applyMatchContextFilter(rows, input.MatchContext)
	if len(rows) == 0 {
		return nil
	}
	effective := migrateCascadeFromRows(rows, normalizeInput(input))
	_, filtered := splitTemporalFiltered(rows, effective)

	sort.SliceStable(filtered, func(i, j int) bool {
		ti, tj := filtered[i].StartTime, filtered[j].StartTime
		switch {
		case ti == nil:
			return false // i sans date → relégué après j
		case tj == nil:
			return true // j sans date → i avant
		default:
			return ti.After(*tj) // DESC : récent d'abord
		}
	})

	ids := make([]string, 0, len(filtered))
	for i := range filtered {
		if filtered[i].MatchID != "" {
			ids = append(ids, filtered[i].MatchID)
		}
	}
	return ids
}

// applyMatchContextFilter restreint les rows selon le contexte de la page :
//   - "solo"  : ne garde que les matchs avec is_with_friends = false
//   - "squad" : ne garde que les matchs avec is_with_friends = true
//   - "all" / vide : retourne rows tel quel
//
// Phase I plan catalogue. Filtre transversal indépendant de la cascade.
func applyMatchContextFilter(rows []domain.FilterMatchRow, matchContext string) []domain.FilterMatchRow {
	switch matchContext {
	case domain.MatchContextSolo:
		out := make([]domain.FilterMatchRow, 0, len(rows))
		for _, r := range rows {
			if !r.IsWithFriends {
				out = append(out, r)
			}
		}
		return out
	case domain.MatchContextSquad:
		out := make([]domain.FilterMatchRow, 0, len(rows))
		for _, r := range rows {
			if r.IsWithFriends {
				out = append(out, r)
			}
		}
		return out
	}
	// "all" ou vide → pas de filtre.
	return rows
}

// ---------------------------------------------------------------------------
// Helpers d'enrichissement
// ---------------------------------------------------------------------------

func modeUI(row domain.FilterMatchRow) string {
	raw := derefStr(row.PairNameFR)
	if raw == "" {
		raw = derefStr(row.PairName)
	}
	return analysis.NormalizeModeLabel(raw)
}

func mapUI(row domain.FilterMatchRow) string {
	if v := derefStr(row.MapNameFR); v != "" {
		return v
	}
	return derefStr(row.MapName)
}

func playlistUI(row domain.FilterMatchRow) string {
	return derefStr(row.PlaylistName)
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ---------------------------------------------------------------------------
// Filtres
// ---------------------------------------------------------------------------

func applyPeriodFilter(rows []domain.FilterMatchRow, p domain.PeriodInput) []domain.FilterMatchRow {
	if p.StartDate == nil && p.EndDate == nil {
		return rows
	}
	out := rows[:0:0]
	for _, r := range rows {
		if r.StartTime == nil {
			continue
		}
		t := *r.StartTime
		if p.StartDate != nil && t.Before(*p.StartDate) {
			continue
		}
		if p.EndDate != nil {
			end := p.EndDate.Add(24*time.Hour - time.Second)
			if t.After(end) {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

func applySessionFilter(rows []domain.FilterMatchRow, s domain.SessionsFilter) []domain.FilterMatchRow {
	keep := make(map[string]struct{})
	add := func(lbl string) {
		if lbl != "" {
			keep[lbl] = struct{}{}
		}
	}
	add(derefStr(s.PickedSessionLabel))
	add(derefStr(s.PickedSoloSessionLabel))
	add(derefStr(s.PickedSquadSessionLabel))
	for _, lbl := range s.PickedSessions {
		add(lbl)
	}
	if len(keep) == 0 {
		return rows
	}
	// `picked_sessions` reçoit soit le label ("30/04/2026 18:30—22:15 (12)"),
	// soit le session_id stocké dans pme.session_id ("1", "2", …) selon le
	// composant frontend qui appelle (SessionMultiSelect envoie des labels,
	// FilterOmnibar.SessionPill envoie des session_id). On accepte les deux
	// pour rester agnostique côté serveur — sinon les pages utilisant SessionPill
	// retournent systématiquement 0 match en production (session_id ≠ label).
	out := rows[:0:0]
	for _, r := range rows {
		if id := derefStr(r.SessionID); id != "" {
			if _, ok := keep[id]; ok {
				out = append(out, r)
				continue
			}
		}
		if lbl := derefStr(r.SessionLabel); lbl != "" {
			if _, ok := keep[lbl]; ok {
				out = append(out, r)
			}
		}
	}
	return out
}

func applyCascadeFilter(rows []domain.FilterMatchRow, c domain.CascadeFilter) []domain.FilterMatchRow {
	rows = applyExperienceFilter(rows, c.ExperienceTypes)
	rows = filterBySet(rows, c.Playlists, playlistUI)
	rows = filterBySet(rows, c.Modes, modeUI)
	rows = filterBySet(rows, c.Maps, mapUI)
	return rows
}

// applyExperienceFilter garde les rows dont le type d'expérience est demandé.
// Matche par SUBSTRING sur la VALUE (FR canonique « PVP non classé » / « PVP
// classé » / « PVE »), jamais sur le Label : la Value reste FR par contrat même
// quand le Label est localisé côté service (GH5-2). Ne PAS remplacer ces
// littéraux FR ni localiser la Value — la cascade front en dépend.
func applyExperienceFilter(rows []domain.FilterMatchRow, types []string) []domain.FilterMatchRow {
	if len(types) == 0 || len(types) >= len(experienceLabels) {
		return rows
	}
	wantPVE, wantRanked, wantUnranked := false, false, false
	for _, t := range types {
		tl := strings.ToLower(t)
		switch {
		case strings.Contains(tl, "pve") || strings.Contains(tl, "firefight"):
			wantPVE = true
		// "non classé" must be checked before "classé" — it is a substring of it.
		case strings.Contains(tl, "non classé") || strings.Contains(tl, "non-classé") || strings.Contains(tl, "unranked"):
			wantUnranked = true
		case strings.Contains(tl, "classé") || strings.Contains(tl, "ranked"):
			wantRanked = true
		default:
			wantUnranked = true
		}
	}
	out := rows[:0:0]
	for _, r := range rows {
		switch {
		case r.IsFirefight && wantPVE:
			out = append(out, r)
		case !r.IsFirefight && r.IsRanked && wantRanked:
			out = append(out, r)
		case !r.IsFirefight && !r.IsRanked && wantUnranked:
			out = append(out, r)
		}
	}
	return out
}

func filterBySet(rows []domain.FilterMatchRow, values []string, fn func(domain.FilterMatchRow) string) []domain.FilterMatchRow {
	if len(values) == 0 {
		return rows
	}
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	out := rows[:0:0]
	for _, r := range rows {
		if _, ok := set[fn(r)]; ok {
			out = append(out, r)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Helpers mineurs
// ---------------------------------------------------------------------------

func normalizeInput(in domain.FilterContextInput) domain.FilterContextInput {
	p := in.Period
	if p.StartDate != nil && p.EndDate != nil && p.EndDate.Before(*p.StartDate) {
		p.StartDate, p.EndDate = p.EndDate, p.StartDate
	}
	in.Period = p
	// Garantit slices non-nil sur les champs JSON sans omitempty : un slice nil
	// Go sérialise en `null`, ce qui viole le contrat avec le frontend typé
	// non-nullable. Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
	if in.Cascade.ExperienceTypes == nil {
		in.Cascade.ExperienceTypes = []string{}
	}
	if in.Cascade.Playlists == nil {
		in.Cascade.Playlists = []string{}
	}
	if in.Cascade.Modes == nil {
		in.Cascade.Modes = []string{}
	}
	if in.Cascade.Maps == nil {
		in.Cascade.Maps = []string{}
	}
	if in.Sessions.PickedSessions == nil {
		in.Sessions.PickedSessions = []string{}
	}
	return in
}

func emptyResolved(effective domain.FilterContextInput, sess domain.SessionOptions) domain.FilterContextResolved {
	// Label = value canonique FR ici ; localisé vers la locale de requête au point
	// d'entrée (FiltersService.Resolve → localizeExperienceOptions). Value FR par
	// contrat (cascade/substring — ne PAS localiser la Value). GH5-2.
	expOpts := make([]domain.LabelValue, len(experienceLabels))
	for i, lbl := range experienceLabels {
		expOpts[i] = domain.LabelValue{Label: lbl, Value: lbl, Count: 0}
	}
	presets := make([]domain.PeriodPresetCount, 0, len(periodPresets))
	for _, p := range periodPresets {
		presets = append(presets, domain.PeriodPresetCount{PresetID: p.id, Days: p.days, Count: 0})
	}
	// Init explicite des 4 slices : un slice Go nil sérialise en JSON `null`,
	// ce qui crashe le front (`opts.filter` sur null). Cf. filters_jsonshape_test.go.
	return domain.FilterContextResolved{
		Effective: effective,
		AvailableOptions: domain.AvailableFilterOptions{
			ExperienceTypes: expOpts,
			Playlists:       []domain.LabelValue{},
			Modes:           []domain.LabelValue{},
			Maps:            []domain.LabelValue{},
		},
		SessionOptions: sess,
		Counts:         domain.FilterCounts{},
		PeriodPresets:  presets,
	}
}

// migrateCascadeValues remplace chaque valeur par sa traduction FR si disponible.
func migrateCascadeValues(values []string, tr map[string]string) []string {
	out := make([]string, len(values))
	for i, v := range values {
		if fr, ok := tr[v]; ok {
			out[i] = fr
		} else {
			out[i] = v
		}
	}
	return out
}

// buildModeTranslationMap construit une map EN→FR depuis les FilterMatchRows déjà enrichis.
// Si applyModeFRTranslations a tournée, PairNameFR contient un nom FR pur ("Assassin")
// tandis que PairName contient encore le nom brut ("Arena:Slayer").
func buildModeTranslationMap(rows []domain.FilterMatchRow) map[string]string {
	tr := make(map[string]string, 8)
	for _, row := range rows {
		en := analysis.NormalizeModeLabel(derefStr(row.PairName))
		fr := analysis.NormalizeModeLabel(derefStr(row.PairNameFR))
		if en != "" && fr != "" && en != fr {
			tr[en] = fr
		}
	}
	return tr
}

// buildMapTranslationMap construit une map EN→FR pour les cartes.
// MapName = nom brut EN, MapNameFR = nom enrichi par applyMapFRTranslations.
func buildMapTranslationMap(rows []domain.FilterMatchRow) map[string]string {
	tr := make(map[string]string, 8)
	for _, row := range rows {
		en := derefStr(row.MapName)
		fr := derefStr(row.MapNameFR)
		if en != "" && fr != "" && en != fr {
			tr[en] = fr
		}
	}
	return tr
}

// buildPlaylistTranslationMap construit une map EN→FR pour les playlists.
// PlaylistNameEN = nom brut EN, PlaylistName = nom enrichi par applyPlaylistFRTranslations.
func buildPlaylistTranslationMap(rows []domain.FilterMatchRow) map[string]string {
	tr := make(map[string]string, 8)
	for _, row := range rows {
		en := derefStr(row.PlaylistNameEN)
		fr := derefStr(row.PlaylistName)
		if en != "" && fr != "" && en != fr {
			tr[en] = fr
		}
	}
	return tr
}
