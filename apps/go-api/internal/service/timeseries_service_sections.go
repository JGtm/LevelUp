// Package service — timeseries_service_sections.go : LES SECTIONS MIGRÉES DEPUIS LES AUTRES
// PAGES — « Portée des engagements » (onglet Résumé) et « Usages d'équipement » (onglet
// Progression) depuis la Synthèse le 2026-09-13, puis « Les formes retenues » (contexte
// SOLO, onglet Progression) depuis l'Escouade le 2026-09-19.
//
// AUCUN CALCUL NEUF ICI, ET C'EST LE POINT. Les trois blocs gardent le producteur de leur
// page d'origine (`buildWeaponRangeSection`, `squadagg.BuildEquipmentUsageBlock`,
// `squadagg.BuildSquadFormesBlock`) et le MÊME scope que le reste de la page — les matchs
// déjà filtrés. Recoder la lecture côté Timeseries
// aurait créé une seconde doctrine de scope, qui aurait divergé au premier correctif.
//
// Fichier séparé : timeseries_service.go tient le plafond des 500 lignes du dépôt.
package service

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squadformes"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/squadagg"
	"levelup/go-api/internal/service/teammates"
)

// WithWeaponRangeRepo injecte le loader de la section « Portée des engagements » (frags et
// morts MESURÉS, distance et dénivelé). Repo nil / titre sans positions par kill ⇒ section
// omise (dégradation gracieuse, jamais une section vide).
func (s *TimeseriesService) WithWeaponRangeRepo(repo port.WeaponRangeRepository) *TimeseriesService {
	s.weaponRangeRepo = repo
	return s
}

// WithEquipmentUsage injecte la source du résumé d'usage (vues _latest) et le résolveur
// d'amis configurés — la MÊME paire que la Synthèse et la page Sessions. Câblé gated par
// film.usage_summary ; repo nil ⇒ bloc servi avec Available=false et raison machine.
//
// `repoRoot` ne sert qu'au CATALOGUE D'ARMES du titre (nommage du detail par niveau) : vide,
// les armes s'affichent sous leur cle — la degradation ecrite partout ailleurs.
func (s *TimeseriesService) WithEquipmentUsage(
	repo port.SessionUsageRepository, friends teammates.FriendGamertagsResolver, repoRoot string,
) *TimeseriesService {
	s.sessionUsageRepo = repo
	s.usageFriends = friends
	s.repoRoot = repoRoot
	return s
}

// WithSquadFormes injecte les deux sources du bloc « Les formes retenues » — le MÊME couple
// que la page Escouade (`teammates.WithSquadFormes`), et pour la même raison : depuis le
// 2026-09-19 les neuf cartes du CONTEXTE SOLO vivent ici, l'Escouade ne gardant que son
// contexte escouade. Un second producteur aurait fait deux mesures du même geste.
//
// `objectives` nil ⇒ bloc sans cartes d'objectif (dégradation propre, jamais le bloc entier).
func (s *TimeseriesService) WithSquadFormes(
	usage port.SquadFormesUsageRepository, objectives port.SquadFormesObjectiveRepository,
) *TimeseriesService {
	s.formesUsageRepo = usage
	s.formesObjectiveRepo = objectives
	return s
}

// WithTimeseriesCoordination injecte les deux lecteurs du bloc « Coordination » et les
// capabilities du titre — les MÊMES que la page Sessions, pour le MÊME producteur.
func (s *TimeseriesService) WithTimeseriesCoordination(
	tactical port.TacticalRepository, appuis port.CoordinationRepository, caps games.CapabilityMap,
) *TimeseriesService {
	s.coordTactical = tactical
	s.coordAppuis = appuis
	s.coordCaps = caps
	return s
}

// WithMatchRange injecte le lecteur de portée « tout le lobby » et le xuid du joueur
// consulté — le nuage des rôles de portée de la page (D23-a). Câblage INCONDITIONNEL, pour
// la même raison que WithWeaponRangeRepo : le repo est le seul à savoir si ce titre a des
// positions par kill.
func (s *TimeseriesService) WithMatchRange(repo port.MatchRangeRepository, xuid string) *TimeseriesService {
	s.matchRangeRepo = repo
	s.matchRangeXUID = xuid
	return s
}

// attachMigratedSections pose les trois blocs sur la réponse, depuis le scope canonique déjà
// filtré. Best-effort de bout en bout : chaque producteur rend nil plutôt que de casser la page.
func (s *TimeseriesService) attachMigratedSections(
	ctx context.Context, resp *domain.TimeseriesPageResponse,
	filteredCanon []canonical.PlayerMatchRow, locale string,
) {
	stop := timing.FromContext(ctx).Section("weapon_range")
	resp.WeaponRange, resp.Elevation = buildWeaponRangeSections(ctx, weaponRangeQuery{
		Repo: s.weaponRangeRepo, TitleSlug: s.titleSlug, Gamertag: s.gamertag, Rows: filteredCanon,
	})
	stop()
	stop = timing.FromContext(ctx).Section("equipment_usage")
	resp.EquipmentUsage = buildEquipmentUsageBlock(ctx, equipmentUsageQuery{
		Repo:            s.sessionUsageRepo,
		PlayerXUID:      s.playerXUID,
		MatchIDs:        synthesisMatchIDs(filteredCanon),
		FriendGamertags: s.timeseriesFriendGamertags(ctx),
		// De quoi NOMMER les armes du detail par niveau, DANS LA LANGUE DE LA REQUETE.
		RepoRoot:  s.repoRoot,
		TitleSlug: s.titleSlug,
		Locale:    locale,
	})
	stop()
	// « Les formes retenues », contexte SOLO : MÊMES match_id que le bloc d'usage
	// ci-dessus. `SelectedGamertags` reste vide — cette page n'a pas d'escouade, et les
	// cartes du contexte escouade ne s'y montent pas.
	stop = timing.FromContext(ctx).Section("squad_formes")
	resp.SquadFormes = squadagg.BuildSquadFormesBlock(ctx, squadagg.SquadFormesQuery{
		Repo:         s.formesUsageRepo,
		Objectives:   s.formesObjectiveRepo,
		PlayerXUID:   s.playerXUID,
		MainGamertag: s.gamertag,
		Metas:        timeseriesFormesMetas(filteredCanon, locale),
		RepoRoot:     s.repoRoot,
		TitleSlug:    s.titleSlug,
		Locale:       locale,
	})
	stop()
	s.attachCoordination(ctx, resp, filteredCanon)
	s.attachMatchRange(ctx, resp, filteredCanon, locale)
}

// attachMatchRange pose le nuage des rôles de portée du joueur consulté sur la FENÊTRE DE
// LA PAGE (lot U, décision D23-a).
//
// UN SEUL JOUEUR PUBLIÉ, UN LOBBY ENTIER MESURÉ : la médiane de référence de chaque match
// porte sur tous ses frags mesurés (c'est elle qui neutralise la carte et le mode), mais
// seule la ligne du joueur consulté est servie — cette page n'est pas un tableau du lobby.
func (s *TimeseriesService) attachMatchRange(
	ctx context.Context, resp *domain.TimeseriesPageResponse,
	filteredCanon []canonical.PlayerMatchRow, locale string,
) {
	defer timing.FromContext(ctx).Section("range_profiles")()
	if s.matchRangeXUID == "" {
		return
	}
	resp.RangeProfiles = buildMatchRangeBlock(ctx, matchRangeQuery{
		Repo:         s.matchRangeRepo,
		TitleSlug:    s.titleSlug,
		Matches:      timeseriesRangeScope(filteredCanon, locale),
		Publish:      map[string]string{s.matchRangeXUID: s.gamertag},
		PublishOrder: []string{s.matchRangeXUID},
		Scope:        "timeseries",
	})
}

// timeseriesRangeScope projette le scope canonique en scope de portée, DU PLUS ANCIEN AU
// PLUS RÉCENT — l'axe des x du nuage, le même ordre que l'Escouade. Le nom de carte vient
// du canonique déjà chargé et DANS LA LANGUE DE LA REQUÊTE (même arbitrage que
// timeseriesFormesMetas) : ré-interroger la base aurait fait deux libellés du même match.
func timeseriesRangeScope(rows []canonical.PlayerMatchRow, locale string) []analysis.MatchRangeMatch {
	out := make([]analysis.MatchRangeMatch, 0, len(rows))
	for _, r := range rows {
		out = append(out, analysis.MatchRangeMatch{
			MatchID:  r.Summary.MatchID,
			PlayedAt: r.Summary.StartedAtUTC,
			MapName:  labelPourLocale(r.Summary.Map, locale),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].PlayedAt.Equal(out[j].PlayedAt) {
			return out[i].PlayedAt.Before(out[j].PlayedAt)
		}
		return out[i].MatchID < out[j].MatchID
	})
	return out
}

// attachCoordination pose le bloc « Coordination » de la page, groupé PAR SOIRÉE.
//
// L'EFFECTIF DE CAMP VIENT DU MÊME `sessionusage.BuildTeamContext` que le bloc d'usage et
// que la courbe d'intensité d'équipe : une seconde définition de « mon équipe » aurait
// affiché deux effectifs pour le même match (réserve R1).
//
// DÉGRADATION NOMMÉE : port des participants non câblé ou lecture en échec ⇒ le bloc est
// servi SANS parité (les deux grandeurs restent justes, seule la référence 1/n manque),
// jamais une parité inventée.
func (s *TimeseriesService) attachCoordination(
	ctx context.Context, resp *domain.TimeseriesPageResponse, filteredCanon []canonical.PlayerMatchRow,
) {
	matchIDs := synthesisMatchIDs(filteredCanon)
	if len(matchIDs) == 0 {
		return
	}
	resp.Coordination = buildCoordinationBlock(ctx, coordinationQuery{
		Tactical:   s.coordTactical,
		Appuis:     s.coordAppuis,
		Caps:       s.coordCaps,
		PlayerXUID: s.playerXUID,
		MatchIDs:   matchIDs,
		TeamSize:   s.coordinationTeamSizes(ctx, matchIDs),
		Soirees:    soireesDesRows(filteredCanon),
	})
}

// coordinationTeamSizes rend l'effectif de mon camp par match, ou nil (pas de parité).
func (s *TimeseriesService) coordinationTeamSizes(ctx context.Context, matchIDs []string) map[string]int {
	defer timing.FromContext(ctx).Section("team_sizes")()
	if s.formesUsageRepo == nil || s.playerXUID == "" {
		return nil
	}
	participants, err := s.formesUsageRepo.LoadParticipants(ctx, matchIDs)
	if err != nil {
		slog.WarnContext(ctx, "timeseries_coordination_participants_en_echec",
			"err", err, "matchs", len(matchIDs))
		return nil
	}
	return sessionusage.BuildTeamContext(s.playerXUID, participants).TeamSize
}

// soireesDesRows groupe les matchs du scope par SOIRÉE, dans l'ordre chronologique du
// premier match de chaque soirée.
//
// UN MATCH SANS SESSION N'ENTRE DANS AUCUNE SOIRÉE : la frise se lit par soirée, pas par
// match isolé — la même règle que la frise d'échange de l'Escouade. Les libellés sont
// rendus TRIÉS par l'instant du plus ancien match : l'ordre d'itération d'une map n'est
// pas un ordre, et une frise temporelle dont les bâtons changent de place à chaque appel
// ne se lit pas.
func soireesDesRows(rows []canonical.PlayerMatchRow) []coordinationSoiree {
	parLabel := map[string][]string{}
	plusAncien := map[string]time.Time{}
	for _, r := range rows {
		if r.Enrichment.SessionLabel == nil || *r.Enrichment.SessionLabel == "" {
			continue
		}
		label := *r.Enrichment.SessionLabel
		parLabel[label] = append(parLabel[label], r.Summary.MatchID)
		if t, ok := plusAncien[label]; !ok || r.Summary.StartedAtUTC.Before(t) {
			plusAncien[label] = r.Summary.StartedAtUTC
		}
	}
	labels := make([]string, 0, len(parLabel))
	for l := range parLabel {
		labels = append(labels, l)
	}
	sort.Slice(labels, func(i, j int) bool { return plusAncien[labels[i]].Before(plusAncien[labels[j]]) })
	out := make([]coordinationSoiree, 0, len(labels))
	for _, l := range labels {
		out = append(out, coordinationSoiree{Label: l, MatchIDs: parLabel[l]})
	}
	return out
}

// timeseriesFormesMetas nomme chaque match du scope pour le bloc « formes retenues ».
//
// L'IDENTITÉ D'AFFICHAGE VIENT DU CANONIQUE DÉJÀ CHARGÉ (même arbitrage que la page
// Escouade, qui la prend de son historique) : ré-interroger la base aurait fait deux
// libellés possibles du même match. Un match sans carte ni mode nommés garde son heure —
// l'écran montre alors l'heure seule, jamais un « inconnu ».
func timeseriesFormesMetas(rows []canonical.PlayerMatchRow, locale string) []squadformes.MatchMeta {
	out := make([]squadformes.MatchMeta, 0, len(rows))
	for _, r := range rows {
		meta := squadformes.MatchMeta{
			MatchID:   r.Summary.MatchID,
			StartTime: r.Summary.StartedAtUTC.UTC().Format("2006-01-02T15:04:05Z"),
		}
		if r.Summary.PairMode != nil {
			meta.ModeLabel = labelPourLocale(r.Summary.PairMode, locale)
		}
		if r.Summary.Map != nil {
			meta.MapLabel = labelPourLocale(r.Summary.Map, locale)
		}
		out = append(out, meta)
	}
	return out
}

// timeseriesFriendGamertags résout les amis du joueur (nil = aucun ami déclaré).
func (s *TimeseriesService) timeseriesFriendGamertags(ctx context.Context) []string {
	if s.usageFriends == nil {
		return nil
	}
	return s.usageFriends(ctx)
}

// labelPourLocale rend le libellé d'un asset DANS LA LANGUE DE LA REQUÊTE, et retombe sur
// le libellé par défaut quand la locale n'est pas traduite — jamais un identifiant machine
// à l'écran.
func labelPourLocale(ref *canonical.AssetReference, locale string) string {
	if ref == nil {
		return ""
	}
	if l, ok := ref.Labels[locale]; ok && l != "" {
		return l
	}
	return ref.DefaultLabel
}
