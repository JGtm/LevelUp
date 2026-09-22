// Package service — session_page_range.go : le bloc « portée des engagements » de la page
// détail de session (plan .ai/V7.5/PLAN_AJUSTEMENTS_SUPP_PRE_V75_2026-09-21.md, lots N2 et
// U, décisions D22-4 et D23-4).
//
// PATRON D'ATTACHEMENT identique au bloc usage : repo OPTIONNEL injecté à la DI, attaché à
// la réponse existante de POST .../pages/sessions/detail — pas d'endpoint dédié, pas de clé
// de cache de plus.
//
// # CE QUI EST PUBLIÉ, ET CE QUI NE L'EST PAS
//
// Un profil par match de la session : la médiane du LOBBY (calculée sur tous les frags
// mesurés du match, camp adverse compris) et UNE SEULE ligne de joueur, celle du joueur
// consulté. Le lobby sert de RÉFÉRENTIEL — c'est ce qui rend l'écart lisible d'une carte à
// l'autre (D22-4) — il n'est jamais publié nominativement : la page Session n'est pas un
// tableau du lobby.
//
// # LA PÉRIODE DE RÉFÉRENCE (lot U)
//
// Les bandes de rôle d'une soirée calculées SUR la soirée sont tautologiques : une session
// entièrement jouée au fusil de précision produit quand même un « Front ». `RangeReference`
// sert donc les mêmes profils sur la PÉRIODE — les matchs du FILTRE de la page, la MÊME
// référence que l'habituel des usages et de la coordination
// (`sessionBlocksScope.ReferenceMatches`), et non une seconde notion de « d'habitude »
// libre de diverger au premier réglage. La fenêtre est bornée à [rangeReferenceWindow]
// matchs les plus récents, les matchs des sessions AFFICHÉES étant toujours retenus : le
// client met leur fenêtre en surbrillance dans le nuage, elle ne peut donc pas en sortir.
//
// UN SEUL BLOC POUR LES DEUX COLONNES : la référence dépend du filtre, pas de la session
// affichée — il n'y a pas de `compare_range_reference`.
//
// # BEST-EFFORT, TOUJOURS
//
// Repo non câblé (titre sans décodeur de film), xuid inconnu, lecture en échec, aucun match
// décodé : le bloc reste nil et la page se rend (cf. buildMatchRangeBlock).
package service

import (
	"context"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// rangeReferenceWindow borne la période de référence de la portée aux N matchs les plus
// récents du filtre. Le nuage de D23-4 en dessine 30 : au-delà, les points se recouvrent et
// la charge utile grossit sans rien ajouter à la lecture.
const rangeReferenceWindow = 30

// WithMatchRange injecte le lecteur de portée « tout le lobby » et le xuid du joueur
// consulté. Câblé INCONDITIONNELLEMENT : c'est le repo qui sait si ce titre a des positions
// par kill (il rend games.ErrCapabilityNotSupported), et un `if capability` ici prendrait la
// même décision à deux endroits qui divergeraient — même doctrine que le câblage Timeseries
// de WeaponRangeRepo.
func (s *SessionPageService) WithMatchRange(repo port.MatchRangeRepository, xuid string) *SessionPageService {
	s.matchRangeRepo = repo
	s.matchRangeXUID = xuid
	return s
}

// attachSessionRange attache le bloc portée de la session COURANTE, celui de la session
// COMPARÉE — miroir d'attachSessionUsage : les deux colonnes du drawer parlent des mêmes
// matchs — puis la période de référence partagée par les deux.
func (s *SessionPageService) attachSessionRange(
	ctx context.Context, resp *domain.SessionPageResponse, sc sessionBlocksScope,
) {
	resp.RangeProfiles = s.buildSessionRange(ctx, sc.Matches, "session")
	if len(sc.CompareMatches) > 0 {
		resp.CompareRangeProfiles = s.buildSessionRange(ctx, sc.CompareMatches, "session_comparee")
	}
	resp.RangeReference = s.buildRangeReference(ctx, sc)
}

// buildSessionRange calcule le bloc portée d'UNE session. nil dans tous les cas où il n'y a
// rien à dire — jamais un bloc vide, qui se lirait comme une mesure à zéro.
func (s *SessionPageService) buildSessionRange(
	ctx context.Context, matches []legacymatch.StatsMatchRow, scope string,
) *domain.MatchRangeBlock {
	if s.matchRangeXUID == "" {
		return nil
	}
	return buildMatchRangeBlock(ctx, matchRangeQuery{
		Repo:         s.matchRangeRepo,
		TitleSlug:    s.titleSlug,
		Matches:      sessionRangeScope(matches),
		Publish:      map[string]string{s.matchRangeXUID: s.gamertag},
		PublishOrder: []string{s.matchRangeXUID},
		Scope:        scope,
	})
}

// buildRangeReference calcule la période de référence : les mêmes profils sur la fenêtre du
// filtre, plus les bandes de rôle et la médiane de période.
//
// RÉFÉRENCE TAUTOLOGIQUE : quand la fenêtre se réduit aux matchs de la session affichée, les
// bandes retomberaient sur la session elle-même — le bloc entier est alors OMIS, exactement
// comme le repère d'habituel de la coordination (cf. cibleDHabituel).
func (s *SessionPageService) buildRangeReference(
	ctx context.Context, sc sessionBlocksScope,
) *domain.RangeReferenceBlock {
	if s.matchRangeXUID == "" {
		return nil
	}
	fenetre := rangeReferenceMatches(sc)
	if len(fenetre) == 0 || memeScope(matchIDsFromStatsRows(fenetre), matchIDsFromStatsRows(sc.Matches)) {
		return nil
	}
	bloc := buildMatchRangeBlock(ctx, matchRangeQuery{
		Repo:         s.matchRangeRepo,
		TitleSlug:    s.titleSlug,
		Matches:      sessionRangeScope(fenetre),
		Publish:      map[string]string{s.matchRangeXUID: s.gamertag},
		PublishOrder: []string{s.matchRangeXUID},
		Scope:        "reference",
	})
	if bloc == nil {
		return nil
	}
	bandes := analysis.MatchRangeRoleBands(bloc.Profiles, s.matchRangeXUID)
	return &domain.RangeReferenceBlock{
		Profiles:           bloc.Profiles,
		RoleLowM:           bandes.LowM,
		RoleHighM:          bandes.HighM,
		PeriodMedianDeltaM: bandes.MedianM,
		MatchesMeasured:    len(bloc.Profiles),
		MatchesTotal:       len(fenetre),
	}
}

// rangeReferenceMatches rend la fenêtre de référence : les matchs des sessions AFFICHÉES
// (toujours retenus, pour que leur surbrillance existe dans le nuage) complétés, en
// remontant du plus récent, jusqu'à [rangeReferenceWindow] matchs du filtre. Sortie du plus
// ancien au plus récent — l'axe des x du nuage.
func rangeReferenceMatches(sc sessionBlocksScope) []legacymatch.StatsMatchRow {
	retenus := make(map[string]legacymatch.StatsMatchRow, rangeReferenceWindow)
	for _, lot := range [][]legacymatch.StatsMatchRow{sc.Matches, sc.CompareMatches} {
		for _, m := range lot {
			retenus[m.MatchID] = m
		}
	}
	reste := make([]legacymatch.StatsMatchRow, 0, len(sc.ReferenceMatches))
	for _, m := range sc.ReferenceMatches {
		if _, vu := retenus[m.MatchID]; !vu {
			reste = append(reste, m)
		}
	}
	sort.SliceStable(reste, func(i, j int) bool { return reste[i].StartTime.After(reste[j].StartTime) })
	for i := range reste {
		if len(retenus) >= rangeReferenceWindow {
			break
		}
		retenus[reste[i].MatchID] = reste[i]
	}
	out := make([]legacymatch.StatsMatchRow, 0, len(retenus))
	for _, m := range retenus {
		out = append(out, m)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].StartTime.Equal(out[j].StartTime) {
			return out[i].StartTime.Before(out[j].StartTime)
		}
		return out[i].MatchID < out[j].MatchID
	})
	return out
}

// sessionRangeScope projette des lignes de match en scope de portée, dans leur ordre.
func sessionRangeScope(matches []legacymatch.StatsMatchRow) []analysis.MatchRangeMatch {
	out := make([]analysis.MatchRangeMatch, 0, len(matches))
	for i := range matches {
		out = append(out, analysis.MatchRangeMatch{
			MatchID:  matches[i].MatchID,
			PlayedAt: matches[i].StartTime,
			MapName:  sessionRangeMapName(matches[i]),
		})
	}
	return out
}

// sessionRangeMapName rend le nom de carte de l'étiquette, en préférant la traduction FR
// quand la base la porte — la MÊME préférence que les lignes du tableau de session.
func sessionRangeMapName(row legacymatch.StatsMatchRow) string {
	if row.MapNameFR != "" {
		return row.MapNameFR
	}
	return row.MapName
}
