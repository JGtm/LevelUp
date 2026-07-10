// filters_options.go — calcul des options disponibles + comptes par catégorie.
//
// Extrait de filters_service.go pour respecter la limite 500L/module.
// Contient la logique de groupage par catégorie (sessions, experience, playlists,
// modes, cartes) qui alimente les dropdowns côté UI.
//
// Sémantique des counts (validée plan smart-filter-counts) :
//
//	Pour chaque option X d'une catégorie K, count = nombre de matchs si la sélection
//	finale de K incluait X en plus de ce qui est déjà coché — c.-à-d. K IN (selected ∪ {X}).
//	Pour une option déjà cochée : count = total sélection actuelle.
//	Pour une option non cochée  : count = matchs si on l'ajoute.
package service

import (
	"sort"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/mappings"
)

// presetSpec décrit un preset période côté backend (mirror du tableau frontend
// PERIOD_PRESETS dans apps/web/src/components/shell/_filter_pills/_hooks.ts).
type presetSpec struct {
	id   string
	days int // 0 = "Toutes" (pas de cutoff)
}

var periodPresets = []presetSpec{
	{id: "7d", days: 7},
	{id: "30d", days: 30},
	{id: "90d", days: 90},
	{id: "all", days: 0},
}

// ---------------------------------------------------------------------------
// Sessions
// ---------------------------------------------------------------------------

// buildSessionOptions agrège les sessions présentes dans `rows` (rows post-match_context)
// et calcule deux counts par session :
//   - MatchCount         : nombre de matchs bruts dans la session
//   - MatchCountFiltered : nombre de matchs si on cochait CETTE session avec la cascade
//     active (sans le filtre period, car period + sessions sont mutuellement exclusifs)
//
// La distinction permet à l'UI de masquer les sessions vides post-cascade tout en gardant
// l'info brute pour les autres consommateurs.
func buildSessionOptions(rows []domain.FilterMatchRow, cascade domain.CascadeFilter) domain.SessionOptions {
	type aggEntry struct {
		count      int
		isSquad    bool
		latestAt   time.Time // start_time du match le plus récent de la session
		earliestAt time.Time // start_time du match le plus ancien de la session
	}
	agg := make(map[string]aggEntry)
	sessionID := make(map[string]string) // label → session_id

	for _, r := range rows {
		lbl := derefStr(r.SessionLabel)
		if lbl == "" {
			continue
		}
		e := agg[lbl]
		e.count++
		if r.IsWithFriends {
			e.isSquad = true
		}
		if r.StartTime != nil {
			if r.StartTime.After(e.latestAt) {
				e.latestAt = *r.StartTime
			}
			if e.earliestAt.IsZero() || r.StartTime.Before(e.earliestAt) {
				e.earliestAt = *r.StartTime
			}
		}
		agg[lbl] = e
		if sid := derefStr(r.SessionID); sid != "" {
			sessionID[lbl] = sid
		}
	}

	// Counts post-cascade par session — sans filtre period (puisque sélectionner
	// une session annule le filter period).
	cascadeFiltered := applyCascadeFilter(rows, cascade)
	cascadeCount := make(map[string]int, len(agg))
	for _, r := range cascadeFiltered {
		if lbl := derefStr(r.SessionLabel); lbl != "" {
			cascadeCount[lbl]++
		}
	}

	labels := make([]string, 0, len(agg))
	for lbl := range agg {
		labels = append(labels, lbl)
	}
	// Tri par date réelle décroissante (la plus récente en tête).
	sort.Slice(labels, func(i, j int) bool {
		return agg[labels[i]].latestAt.After(agg[labels[j]].latestAt)
	})

	// Init explicite à [] : un slice nil sérialise en JSON `null` et crashe le
	// front typé non-nullable. Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
	all := make([]domain.SessionOption, 0, len(labels))
	soloLabels := make([]string, 0)
	squadLabels := make([]string, 0)
	for _, lbl := range labels {
		e := agg[lbl]
		// Masquer les sessions d'un seul match : non listées dans la SessionPill
		// (Solo) ni la carte des counts Escouade. Cf. minListedSessionMatches.
		if e.count < minListedSessionMatches {
			continue
		}
		sid := sessionID[lbl]
		if sid == "" {
			sid = lbl
		}
		opt := domain.SessionOption{
			Label:              lbl,
			SessionID:          sid,
			MatchCount:         e.count,
			MatchCountFiltered: cascadeCount[lbl],
			IsSquad:            e.isSquad,
			StartedAtUTC:       e.earliestAt,
			EndedAtUTC:         e.latestAt,
		}
		all = append(all, opt)
		if e.isSquad {
			squadLabels = append(squadLabels, lbl)
		} else {
			soloLabels = append(soloLabels, lbl)
		}
	}
	return domain.SessionOptions{
		AllSessions: all,
		SoloLabels:  soloLabels,
		SquadLabels: squadLabels,
	}
}

// ---------------------------------------------------------------------------
// Period presets — counts si l'utilisateur switchait en mode period
// ---------------------------------------------------------------------------

// buildPeriodPresetCounts retourne, pour chaque preset (7j/30j/90j/Toutes),
// le nombre de matchs qu'il contiendrait avec la cascade active.
//
// Calcul indépendant du filter_mode courant : on simule "et si on switchait
// en mode period avec ce preset ?". Ignore donc tout filtre session.
//
// `rows` doit déjà avoir le match_context appliqué.
// `now` est injecté pour la testabilité.
func buildPeriodPresetCounts(rows []domain.FilterMatchRow, cascade domain.CascadeFilter, now time.Time) []domain.PeriodPresetCount {
	rowsCascade := applyCascadeFilter(rows, cascade)
	out := make([]domain.PeriodPresetCount, 0, len(periodPresets))
	for _, p := range periodPresets {
		count := 0
		if p.days == 0 {
			count = len(rowsCascade) // "Toutes"
		} else {
			cutoff := now.AddDate(0, 0, -p.days)
			for _, r := range rowsCascade {
				if r.StartTime != nil && !r.StartTime.Before(cutoff) {
					count++
				}
			}
		}
		out = append(out, domain.PeriodPresetCount{PresetID: p.id, Days: p.days, Count: count})
	}
	return out
}

// ---------------------------------------------------------------------------
// Season counts — counts si l'utilisateur sélectionnait une saison
// ---------------------------------------------------------------------------

// SeasonWindow est la projection minimale d'une saison nécessaire au calcul
// du nombre de matchs qu'elle couvre. Construit depuis le mappings registry
// via SeasonsFromAssets.
type SeasonWindow struct {
	ID    string
	Start time.Time
	End   *time.Time // nil = saison ouverte (compte tout >= Start)
}

// BuildSeasonCounts retourne, pour chaque saison du catalog, le nombre de
// matchs dont StartTime ∈ [season.Start, season.End) ET qui matchent la
// cascade active.
//
// Symétrique de buildPeriodPresetCounts. Sert au folding "+N saisons sans
// matchs" côté frontend (popover SaisonPill).
//
// `rows` doit déjà avoir le match_context appliqué.
// Si `seasons` est vide → retourne nil (pas d'allocation).
func BuildSeasonCounts(rows []domain.FilterMatchRow, cascade domain.CascadeFilter, seasons []SeasonWindow) []domain.SeasonCount {
	if len(seasons) == 0 {
		return nil
	}
	rowsCascade := applyCascadeFilter(rows, cascade)
	out := make([]domain.SeasonCount, 0, len(seasons))
	for _, s := range seasons {
		count := 0
		for _, r := range rowsCascade {
			if r.StartTime == nil {
				continue
			}
			if r.StartTime.Before(s.Start) {
				continue
			}
			if s.End != nil && !r.StartTime.Before(*s.End) {
				continue
			}
			count++
		}
		out = append(out, domain.SeasonCount{SeasonID: s.ID, Count: count})
	}
	return out
}

// SeasonsFromAssets projette les entrées du kind "season" d'un AssetMappingSet
// en []SeasonWindow exploitable par BuildSeasonCounts.
//
// Filtre les entrées dont StartDate est nil (les autres kinds n'en auront
// pas, mais le filtrage rend le helper robuste à toute évolution du TOML).
// Préserve l'ordre AllOfKind (DisplayOrder croissant).
func SeasonsFromAssets(assets *mappings.AssetMappingSet) []SeasonWindow {
	if assets == nil {
		return nil
	}
	entries := assets.AllOfKind("season")
	out := make([]SeasonWindow, 0, len(entries))
	for _, e := range entries {
		if e.StartDate == nil {
			continue
		}
		out = append(out, SeasonWindow{
			ID:    e.ID,
			Start: *e.StartDate,
			End:   e.EndDate,
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// Options disponibles + counts OR (cascade)
// ---------------------------------------------------------------------------

// rowExperienceLabel dérive le label d'expérience d'un FilterMatchRow.
// Doit rester cohérent avec synthesisExperienceLabel (teammates_service.go).
func rowExperienceLabel(r domain.FilterMatchRow) string {
	if r.IsFirefight {
		return expTypePVE
	}
	if r.IsRanked {
		return expTypePVPRanked
	}
	return expTypePVPUnranked
}

// buildAvailableOptions calcule les options disponibles pour les 4 catégories
// (experience_types, playlists, modes, cartes) avec un compte par option selon
// la sémantique OR.
//
// Pour chaque option X de catégorie K :
//
//	count(X) = nb matchs avec K IN (selected_K ∪ {X}) + tous les autres filtres cascade actifs
//
// Cascade DESCENDANTE conservée pour le périmètre des options visibles :
//   - Experience : visible si présent dans `rows` (post période/sessions/match_context)
//   - Playlists  : visible si présent dans rows post-Experience
//   - Modes      : visible si présent dans rows post-(Experience + Playlists)
//   - Cartes     : visible si présent dans rows post-(Experience + Playlists + Modes)
func buildAvailableOptions(rows []domain.FilterMatchRow, c domain.CascadeFilter) domain.AvailableFilterOptions {
	// Experience types : valeurs présentes dans `rows`, ordre canonique.
	expOpts := buildExperienceOptions(rows, c)

	// Playlists — au-dessus dans la cascade : Experience appliqué, pas Playlists.
	rowsExp := applyExperienceFilter(rows, c.ExperienceTypes)
	playlistOpts := uniqueLabelValuesWithORCounts(rowsExp, playlistUI, c.Playlists, func(rs []domain.FilterMatchRow) []domain.FilterMatchRow {
		rs = filterBySet(rs, c.Modes, modeUI)
		rs = filterBySet(rs, c.Maps, mapUI)
		return rs
	})

	// Modes — Experience + Playlists appliqués, pas Modes.
	rowsPl := filterBySet(rowsExp, c.Playlists, playlistUI)
	modeOpts := uniqueLabelValuesWithORCounts(rowsPl, modeUI, c.Modes, func(rs []domain.FilterMatchRow) []domain.FilterMatchRow {
		return filterBySet(rs, c.Maps, mapUI)
	})

	// Cartes — Experience + Playlists + Modes appliqués, pas Maps.
	rowsMo := filterBySet(rowsPl, c.Modes, modeUI)
	mapOpts := uniqueLabelValuesWithORCounts(rowsMo, mapUI, c.Maps, func(rs []domain.FilterMatchRow) []domain.FilterMatchRow {
		return rs
	})

	return domain.AvailableFilterOptions{
		ExperienceTypes: expOpts,
		Playlists:       playlistOpts,
		Modes:           modeOpts,
		Maps:            mapOpts,
	}
}

// buildExperienceOptions : options Experience avec counts OR.
// Experience est en haut de la cascade : pour chaque label X, count = matchs si
// experience IN (selected ∪ {X}) avec playlists/modes/maps actifs.
func buildExperienceOptions(rows []domain.FilterMatchRow, c domain.CascadeFilter) []domain.LabelValue {
	existing := make(map[string]struct{}, 3)
	for _, r := range rows {
		existing[rowExperienceLabel(r)] = struct{}{}
	}
	out := make([]domain.LabelValue, 0, len(experienceLabels))
	selectedSet := stringSliceToSet(c.ExperienceTypes)
	for _, lbl := range experienceLabels {
		if _, ok := existing[lbl]; !ok {
			continue
		}
		// Sélection simulée : selected ∪ {lbl}
		simExp := unionWith(selectedSet, lbl)
		rs := applyExperienceFilter(rows, simExp)
		rs = filterBySet(rs, c.Playlists, playlistUI)
		rs = filterBySet(rs, c.Modes, modeUI)
		rs = filterBySet(rs, c.Maps, mapUI)
		// Label = value canonique FR ; localisé vers la locale de requête au point
		// d'entrée (FiltersService.Resolve → localizeExperienceOptions). Value FR par
		// contrat (cascade EXPERIENCE_TO_CASCADE + matchers substring). GH5-2.
		out = append(out, domain.LabelValue{Label: lbl, Value: lbl, Count: len(rs)})
	}
	return out
}

// uniqueLabelValuesWithORCounts : valeurs uniques de la catégorie K extraites par `fn`,
// triées alphabétiquement, chacune assortie de son count selon la sémantique OR.
//
// `rowsHorsK` doit être pré-filtré par les catégories AU-DESSUS de K dans la cascade
// (Experience pour Playlists, Experience+Playlists pour Modes, etc.) mais PAS par K elle-même.
//
// `selectedK` : valeurs déjà cochées dans K.
// `applyDownCats` : applique les catégories EN-DESSOUS de K (ex: pour Modes, applique Maps).
func uniqueLabelValuesWithORCounts(
	rowsHorsK []domain.FilterMatchRow,
	fn func(domain.FilterMatchRow) string,
	selectedK []string,
	applyDownCats func([]domain.FilterMatchRow) []domain.FilterMatchRow,
) []domain.LabelValue {
	// Étape 1 : valeurs uniques visibles dans rowsHorsK.
	seen := make(map[string]struct{})
	for _, r := range rowsHorsK {
		if v := fn(r); v != "" {
			seen[v] = struct{}{}
		}
	}
	vals := make([]string, 0, len(seen))
	for v := range seen {
		vals = append(vals, v)
	}
	sort.Strings(vals)

	// Étape 2 : pour chaque valeur X, count = len(applyDownCats(filter(rowsHorsK, K IN (selected ∪ {X})))).
	selectedSet := stringSliceToSet(selectedK)
	out := make([]domain.LabelValue, 0, len(vals))
	for _, x := range vals {
		simSet := copyStringSet(selectedSet)
		simSet[x] = struct{}{}
		filtered := filterByValueSet(rowsHorsK, fn, simSet)
		filtered = applyDownCats(filtered)
		out = append(out, domain.LabelValue{Label: x, Value: x, Count: len(filtered)})
	}
	return out
}

// ---------------------------------------------------------------------------
// Helpers internes
// ---------------------------------------------------------------------------

func stringSliceToSet(s []string) map[string]struct{} {
	out := make(map[string]struct{}, len(s))
	for _, v := range s {
		out[v] = struct{}{}
	}
	return out
}

func copyStringSet(in map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(in)+1)
	for k := range in {
		out[k] = struct{}{}
	}
	return out
}

func unionWith(set map[string]struct{}, extra string) []string {
	out := make([]string, 0, len(set)+1)
	for k := range set {
		out = append(out, k)
	}
	if _, ok := set[extra]; !ok {
		out = append(out, extra)
	}
	return out
}

// filterByValueSet : variante de filterBySet qui prend un set déjà construit
// (évite de reconstruire le set à chaque appel dans une boucle de counts).
func filterByValueSet(rows []domain.FilterMatchRow, fn func(domain.FilterMatchRow) string, set map[string]struct{}) []domain.FilterMatchRow {
	if len(set) == 0 {
		return rows
	}
	out := rows[:0:0]
	for _, r := range rows {
		if _, ok := set[fn(r)]; ok {
			out = append(out, r)
		}
	}
	return out
}
