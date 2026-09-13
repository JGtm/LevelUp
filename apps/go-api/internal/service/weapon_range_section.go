// Package service — weapon_range_section.go : LA SECTION « PORTÉE PAR ARME »
// (plan .ai/PLAN_DUELS_PORTEE_2026-09-06.md, lot 4).
//
// SECTION MIGRÉE LE 2026-09-13 : elle a quitté la Synthèse pour l'onglet Résumé des Séries
// temporelles, sans changer de producteur. Le chargement vit dans une FONCTION LIBRE
// (`buildWeaponRangeSection`) prenant son scope canonique en paramètre — un service qui
// appellerait un autre service serait un couplage horizontal (skill arch-rules).
//
// Fichier séparé de synthesis_service.go, qui frôle déjà le plafond de 500 lignes du dépôt.
// La frontière suit la donnée : ce qui est propre à la portée vit ici, l'orchestration de la
// page reste là-bas.
//
// # CE QUI SE DÉCIDE ICI, ET CE QUI NE S'Y CALCULE PAS
//
// Ici : le scope (les match_id du scope déjà filtré par période, D10), le régime d'échec
// best-effort, l'assemblage du bloc de réponse et l'hydratation des libellés. Les percentiles,
// le seuil de publication, la ventilation du dénivelé et l'appariement entame -> coup fatal
// vivent dans `internal/analysis` (purs, testables seuls) ; le SQL vit dans le repo. Aucune
// des trois frontières ne bouge ici.
package service

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// weaponRangeQuery — tout ce dont la section a besoin : son repo, le titre, le joueur et le
// scope canonique DÉJÀ FILTRÉ par la page appelante. Aucune des deux pages ne refiltre ici :
// chacune pose la même question à ses propres matchs.
type weaponRangeQuery struct {
	Repo      port.WeaponRangeRepository
	TitleSlug string
	Gamertag  string
	Rows      []canonical.PlayerMatchRow
}

// buildWeaponRangeSection charge et assemble la section pour un scope donné.
//
// BEST-EFFORT, EXACTEMENT COMME loadWeaponAccuracy : nil (section omise) si le repo n'est pas
// câblé, si le joueur ou le scope est vide, si le titre ne produit pas de positions par kill
// (`games.ErrCapabilityNotSupported` -> Debug, c'est une absence légitime) ou si la lecture
// échoue (-> Warn, c'est une anomalie). Une section absente ne casse jamais la page.
func buildWeaponRangeSection(ctx context.Context, q weaponRangeQuery) *domain.SynthesisWeaponRange {
	if q.Repo == nil || q.Gamertag == "" || len(q.Rows) == 0 {
		return nil
	}
	scope := weaponRangeScope(q.Rows)
	filters := port.WeaponRangeFilters{MatchIDs: scope.matchIDs, Gamertag: q.Gamertag}

	kills, err := q.Repo.LoadWeaponRange(ctx, q.TitleSlug, filters)
	if err != nil {
		logWeaponRangeFailure(ctx, q, "portee", len(scope.matchIDs), err)
		return nil
	}
	if len(kills) == 0 {
		// Scope réel mais aucun frag mesuré : le décodeur n'a pas (encore) couvert ces
		// matchs. Pas une panne, pas une section vide — pas de section.
		slog.DebugContext(ctx, "portee par arme — aucun frag mesure sur le scope",
			"title", q.TitleSlug, "gamertag", q.Gamertag, "match_count", len(scope.matchIDs))
		return nil
	}

	// L'ENTAME EST UN BONUS, SON ABSENCE N'EMPORTE PAS LA SECTION. Sa couverture est
	// partielle par construction tant que le backfill n'a pas tourné (D5) : la portée se
	// publie sans elle, et le bloc Opening reste nil plutôt que de valoir zéro.
	openings, err := q.Repo.LoadWeaponOpening(ctx, q.TitleSlug, filters)
	if err != nil {
		logWeaponRangeFailure(ctx, q, "entame", len(scope.matchIDs), err)
		openings = nil
	}

	block := buildWeaponRangeBlock(kills, openings, scope)
	hydrateWeaponRangeLabels(ctx, q, block)
	slog.DebugContext(ctx, "portee par arme",
		"title", q.TitleSlug, "gamertag", q.Gamertag,
		"armes", len(block.Weapons), "frags_mesures", block.MeasuredKills,
		"morts_mesurees", block.MeasuredDeaths, "entames", len(openings))
	return block
}

// logWeaponRangeFailure distingue l'absence légitime de l'anomalie — parité loadWeaponAccuracy.
// Une capability manquante en Warn noierait les vrais bugs SQL sur un titre sans décodeur.
func logWeaponRangeFailure(
	ctx context.Context, q weaponRangeQuery, lecture string, matchCount int, err error,
) {
	if errors.Is(err, games.ErrCapabilityNotSupported) {
		slog.DebugContext(ctx, "portee par arme — capability absente",
			"lecture", lecture, "title", q.TitleSlug, "gamertag", q.Gamertag)
		return
	}
	slog.WarnContext(ctx, "portee par arme — lecture en echec (best-effort, section degradee)",
		"lecture", lecture, "title", q.TitleSlug, "gamertag", q.Gamertag,
		"match_count", matchCount, "err", err)
}

// weaponRangeScopeInfo : le scope de la section — ses matchs et les DEUX totaux du joueur.
//
// LES TOTAUX VIENNENT DU SCOPE CANONIQUE, JAMAIS DE LA TABLE DE POSITIONS. C'est ce qui permet
// de publier « 1 214 frags mesurés sur 1 602 » : le dénominateur est ce que le joueur a
// réellement fait, le numérateur ce que le décodeur a su placer. Les tirer tous deux de la
// même source ferait toujours 100 % et cacherait la couverture.
type weaponRangeScopeInfo struct {
	matchIDs    []string
	totalKills  int
	totalDeaths int
}

// weaponRangeScope projette le scope filtré. Un match sans compteur (nil) n'ajoute rien plutôt
// que zéro : l'absence de donnée n'est pas une performance nulle.
func weaponRangeScope(filteredCanon []canonical.PlayerMatchRow) weaponRangeScopeInfo {
	sc := weaponRangeScopeInfo{matchIDs: make([]string, 0, len(filteredCanon))}
	for _, r := range filteredCanon {
		sc.matchIDs = append(sc.matchIDs, r.Summary.MatchID)
		if r.Self.Kills != nil {
			sc.totalKills += *r.Self.Kills
		}
		if r.Self.Deaths != nil {
			sc.totalDeaths += *r.Self.Deaths
		}
	}
	return sc
}

// hydrateWeaponRangeLabels remplit les noms d'affichage des armes publiées ET de celles
// écartées par le seuil.
//
// LES DEUX LISTES, ET C'EST VOULU : la ligne « sous le seuil » NOMME les armes (« Hydra (6) »),
// sans quoi elle dirait « 3 armes » et n'apprendrait rien. Best-effort : une clé que la
// metadata ne connaît pas garde un libellé vide, et le front retombe sur `WeaponKey` — jamais
// un nom inventé côté Go (aucun libellé FR/EN en dur, règle transverse multi-titre).
func hydrateWeaponRangeLabels(
	ctx context.Context, q weaponRangeQuery, block *domain.SynthesisWeaponRange,
) {
	keys := collectWeaponKeys(block)
	if len(keys) == 0 {
		return
	}
	labels, err := q.Repo.ResolveWeaponLabels(ctx, keys)
	if err != nil {
		slog.WarnContext(ctx, "portee par arme — libelles non resolus (repli sur les cles)",
			"title", q.TitleSlug, "armes", len(keys), "err", err)
		return
	}
	for i := range block.Weapons {
		l := labels[block.Weapons[i].WeaponKey]
		block.Weapons[i].Label, block.Weapons[i].LabelEN = l.Label, l.LabelEN
	}
	hydrateBelowThreshold(block.BelowThresholdKills, labels)
	hydrateBelowThreshold(block.BelowThresholdDeaths, labels)
}

// hydrateBelowThreshold pose le libellé de chaque arme écartée. L'ORDRE N'EST PAS TOUCHÉ : il
// vient de l'agrégat (effectif décroissant), et le rejouer ici en ferait une seconde doctrine
// de tri qui divergerait au premier changement.
func hydrateBelowThreshold(rows []domain.WeaponBelowThreshold, labels map[string]port.WeaponLabel) {
	for i := range rows {
		l := labels[rows[i].WeaponKey]
		rows[i].Label, rows[i].LabelEN = l.Label, l.LabelEN
	}
}

// collectWeaponKeys rend les clés à traduire, dédupliquées, dans l'ordre de première apparition.
func collectWeaponKeys(block *domain.SynthesisWeaponRange) []string {
	keys := make([]string, 0, len(block.Weapons))
	seen := make(map[string]bool, len(block.Weapons))
	ajouter := func(k string) {
		if k != "" && !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	for _, w := range block.Weapons {
		ajouter(w.WeaponKey)
	}
	for _, w := range block.BelowThresholdKills {
		ajouter(w.WeaponKey)
	}
	for _, w := range block.BelowThresholdDeaths {
		ajouter(w.WeaponKey)
	}
	return keys
}
