// Package service — synthesis_weapon_records.go : LA SECTION « RECORDS DE DISTANCE PAR ARME »
// de la Synthèse (plan .ai/V7.5/PLAN_RECORDS_DISTANCE_2026-09-20.md).
//
// FONCTION LIBRE, comme la portée (`weapon_range_section.go`) : elle prend son scope canonique
// en paramètre, et un service qui appellerait un autre service serait un couplage horizontal.
// Fichier séparé de synthesis_service.go, qui frôle le plafond de 500 lignes.
//
// # CE QUI SE DÉCIDE ICI, ET CE QUI NE S'Y CALCULE PAS
//
// Ici : le scope (les match_id du scope déjà filtré), le côté (les frags du joueur, jamais ses
// morts), l'EXCLUSION PAR CLASSE et son inventaire nommé, le régime d'échec best-effort,
// l'assemblage du bloc, l'hydratation des libellés et du contexte de match. Le record et la
// médiane vivent dans `internal/analysis` (purs) ; le SQL vit dans le repo ; la classe d'une
// arme vient du REGISTRE via le port, jamais d'une liste de clés en dur.
package service

import (
	"context"
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

// weaponRecordsExcludedClasses : les classes de registre pour lesquelles UNE DISTANCE N'A PAS
// DE SENS — un coup de crosse se donne au contact, une chute ou une bobine mesurent l'écart
// avec un point de décor, un répulseur pousse. Elles sortent de la règle mais restent NOMMÉES
// dans la section (D9). Posé le 2026-09-20 avec la section ; toute autre classe (véhicule,
// tourelle, grenade) tire ou projette, et son record se lit.
// Les clés viennent des constantes canoniques de domain (source unique des classes),
// jamais de littéraux recopiés.
var weaponRecordsExcludedClasses = map[string]bool{
	domain.FragClassMelee:         true,
	domain.FragClassEnvironmental: true,
	domain.FragClassEquipment:     true,
}

// weaponRecordsQuery — le repo, le titre, le joueur et le scope canonique DÉJÀ FILTRÉ.
type weaponRecordsQuery struct {
	Repo      port.WeaponRangeRepository
	TitleSlug string
	Gamertag  string
	Rows      []canonical.PlayerMatchRow
}

// buildWeaponRecordsSection charge et assemble la section pour un scope donné.
//
// BEST-EFFORT, même régime que la portée : nil (section omise) si le repo n'est pas câblé, si
// le joueur ou le scope est vide, si le titre ne produit pas de positions par kill
// (`games.ErrCapabilityNotSupported` -> Debug) ou si la lecture échoue (-> Warn). Une section
// absente ne casse jamais la page.
func buildWeaponRecordsSection(ctx context.Context, q weaponRecordsQuery) *domain.SynthesisWeaponRecords {
	defer timing.FromContext(ctx).Section("weapon_records")()
	if q.Repo == nil || q.Gamertag == "" || len(q.Rows) == 0 {
		return nil
	}
	scope := weaponRangeScope(q.Rows)
	filters := port.WeaponRangeFilters{MatchIDs: scope.matchIDs, Gamertag: q.Gamertag}

	kills, err := q.Repo.LoadWeaponRange(ctx, q.TitleSlug, filters)
	if err != nil {
		logWeaponRangeFailure(ctx, weaponRangeQuery{TitleSlug: q.TitleSlug, Gamertag: q.Gamertag},
			"records", len(scope.matchIDs), err)
		return nil
	}
	mine := keepMeasuredSide(kills, analysis.SideKiller)
	if len(mine) == 0 {
		slog.DebugContext(ctx, "records de distance — aucun frag mesure sur le scope",
			"title", q.TitleSlug, "gamertag", q.Gamertag, "match_count", len(scope.matchIDs))
		return nil
	}

	dims := resolveWeaponRecordsDimensions(ctx, q, mine)
	kept, excluded := splitWeaponRecordsByClass(mine, dims)
	records := analysis.WeaponDistanceRecords(kept, analysis.SideKiller)
	block := &domain.SynthesisWeaponRecords{
		Weapons:       toWeaponRecordRows(records, dims, indexCanonicalByMatch(q.Rows)),
		MeasuredKills: len(mine),
		TotalKills:    scope.totalKills,
		Excluded:      excluded,
	}
	hydrateWeaponRecordsLabels(ctx, q, block)
	slog.DebugContext(ctx, "records de distance par arme",
		"title", q.TitleSlug, "gamertag", q.Gamertag,
		"armes", len(block.Weapons), "ecartees", len(excluded), "frags_mesures", block.MeasuredKills)
	return block
}

// keepMeasuredSide ne garde que les frags d'un côté. Ordre d'entrée conservé.
func keepMeasuredSide(kills []analysis.MeasuredKill, side analysis.Side) []analysis.MeasuredKill {
	out := make([]analysis.MeasuredKill, 0, len(kills))
	for _, k := range kills {
		if k.Side == side {
			out = append(out, k)
		}
	}
	return out
}

// resolveWeaponRecordsDimensions traduit les clés d'arme en dimensions de registre. BEST-EFFORT :
// une panne rend une map vide (rien n'est écarté, les classes restent vides côté web) et se
// journalise en Warn — un registre muet n'est pas une raison de perdre la section.
func resolveWeaponRecordsDimensions(
	ctx context.Context, q weaponRecordsQuery, kills []analysis.MeasuredKill,
) map[string]port.WeaponDimensions {
	keys := make([]string, 0, 16)
	seen := make(map[string]bool, 16)
	for _, k := range kills {
		if !seen[k.WeaponKey] {
			seen[k.WeaponKey] = true
			keys = append(keys, k.WeaponKey)
		}
	}
	dims, err := q.Repo.ResolveWeaponDimensions(ctx, q.TitleSlug, keys)
	if err != nil {
		slog.WarnContext(ctx, "records de distance — dimensions non resolues (aucune exclusion)",
			"title", q.TitleSlug, "armes", len(keys), "err", err)
		return map[string]port.WeaponDimensions{}
	}
	if dims == nil {
		return map[string]port.WeaponDimensions{}
	}
	return dims
}

// splitWeaponRecordsByClass sépare les frags dont la classe d'arme a un sens de distance de
// ceux qui n'en ont pas, et NOMME ces derniers arme par arme (effectif décroissant, puis clé).
func splitWeaponRecordsByClass(
	kills []analysis.MeasuredKill, dims map[string]port.WeaponDimensions,
) ([]analysis.MeasuredKill, []domain.WeaponExcludedFromRecords) {
	kept := make([]analysis.MeasuredKill, 0, len(kills))
	counts := make(map[string]int)
	for _, k := range kills {
		if weaponRecordsExcludedClasses[dims[k.WeaponKey].Class] {
			counts[k.WeaponKey]++
			continue
		}
		kept = append(kept, k)
	}
	excluded := make([]domain.WeaponExcludedFromRecords, 0, len(counts))
	for key, n := range counts {
		excluded = append(excluded, domain.WeaponExcludedFromRecords{
			WeaponKey: key, Class: dims[key].Class, Measured: n,
		})
	}
	sort.Slice(excluded, func(i, j int) bool {
		if excluded[i].Measured != excluded[j].Measured {
			return excluded[i].Measured > excluded[j].Measured
		}
		return excluded[i].WeaponKey < excluded[j].WeaponKey
	})
	if len(excluded) == 0 {
		return kept, nil
	}
	return kept, excluded
}

// indexCanonicalByMatch indexe les résumés du scope par match_id : c'est là que la carte et la
// date du match du record se lisent, sans requête neuve.
func indexCanonicalByMatch(rows []canonical.PlayerMatchRow) map[string]canonical.MatchSummary {
	out := make(map[string]canonical.MatchSummary, len(rows))
	for _, r := range rows {
		out[r.Summary.MatchID] = r.Summary
	}
	return out
}

// toWeaponRecordRows projette les records purs en lignes de contrat, habillées de leur classe
// et du contexte du match source. L'ORDRE DE L'ANALYSE EST CONSERVÉ.
func toWeaponRecordRows(
	records []analysis.WeaponDistanceRecord,
	dims map[string]port.WeaponDimensions,
	byMatch map[string]canonical.MatchSummary,
) []domain.WeaponDistanceRecordRow {
	rows := make([]domain.WeaponDistanceRecordRow, 0, len(records))
	for _, r := range records {
		row := domain.WeaponDistanceRecordRow{
			WeaponKey: r.WeaponKey,
			Class:     dims[r.WeaponKey].Class,
			Measured:  r.Measured,
			MedianM:   r.MedianM,
			RecordM:   r.RecordM,
			Record:    domain.WeaponRecordFrag{MatchID: r.RecordMatchID, TimeMS: r.RecordTimeMS},
		}
		if s, ok := byMatch[r.RecordMatchID]; ok {
			if !s.StartedAtUTC.IsZero() {
				started := s.StartedAtUTC
				row.Record.StartedAt = &started
			}
			row.Record.MapLabel, row.Record.MapLabelEN = assetLabelPair(s.Map)
		}
		rows = append(rows, row)
	}
	return rows
}

// assetLabelPair rend le libellé FR et le libellé EN d'une référence d'asset canonique, avec
// repli sur le libellé par défaut. Aucun nom n'est inventé : un asset nil rend deux vides.
func assetLabelPair(ref *canonical.AssetReference) (fr, en string) {
	if ref == nil {
		return "", ""
	}
	fr, en = ref.Labels["fr"], ref.Labels["en"]
	if fr == "" {
		fr = ref.DefaultLabel
	}
	if en == "" {
		en = ref.DefaultLabel
	}
	return fr, en
}

// hydrateWeaponRecordsLabels pose les noms d'affichage des armes publiées ET écartées. Best-effort
// : une clé inconnue garde un libellé vide et le front retombe sur `WeaponKey` — jamais un nom
// inventé côté Go.
func hydrateWeaponRecordsLabels(ctx context.Context, q weaponRecordsQuery, block *domain.SynthesisWeaponRecords) {
	keys := make([]string, 0, len(block.Weapons)+len(block.Excluded))
	for _, w := range block.Weapons {
		keys = append(keys, w.WeaponKey)
	}
	for _, w := range block.Excluded {
		keys = append(keys, w.WeaponKey)
	}
	if len(keys) == 0 {
		return
	}
	labels, err := q.Repo.ResolveWeaponLabels(ctx, keys)
	if err != nil {
		slog.WarnContext(ctx, "records de distance — libelles non resolus (repli sur les cles)",
			"title", q.TitleSlug, "armes", len(keys), "err", err)
		return
	}
	for i := range block.Weapons {
		l := labels[block.Weapons[i].WeaponKey]
		block.Weapons[i].Label, block.Weapons[i].LabelEN = l.Label, l.LabelEN
	}
	for i := range block.Excluded {
		l := labels[block.Excluded[i].WeaponKey]
		block.Excluded[i].Label, block.Excluded[i].LabelEN = l.Label, l.LabelEN
	}
}
