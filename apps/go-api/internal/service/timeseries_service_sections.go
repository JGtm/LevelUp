// Package service — timeseries_service_sections.go : LES SECTIONS DE L'ONGLET « USAGES » ET DE LA
// PROGRESSION QUI VIENNENT D'AUTRES PAGES — « Portée des engagements » (producteur de la
// Synthèse), « Les formes retenues » (contexte SOLO, producteur de l'Escouade), la coordination
// et les rôles de portée (producteurs de la page Sessions) — puis l'Emprise solo, les vies et
// l'emblème (fichiers voisins).
//
// AUCUN CALCUL NEUF ICI, ET C'EST LE POINT. Les blocs gardent le producteur de leur page
// d'origine (`buildWeaponRangeSection`, `squadagg.BuildSquadFormesBlock`…) et le MÊME scope que le
// reste de la page — les matchs déjà filtrés. Recoder la lecture côté Timeseries aurait créé une
// seconde doctrine de scope, qui aurait divergé au premier correctif.
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
)

// WithWeaponRangeRepo injecte le loader de la section « Portée des engagements » (frags et
// morts MESURÉS, distance et dénivelé). Repo nil / titre sans positions par kill ⇒ section
// omise (dégradation gracieuse, jamais une section vide).
func (s *TimeseriesService) WithWeaponRangeRepo(repo port.WeaponRangeRepository) *TimeseriesService {
	s.weaponRangeRepo = repo
	return s
}

// WithUsageSummary injecte la source du résumé d'usage (vues _latest), lue UNE fois par requête et
// partagée par l'Emprise et les formes retenues. Câblé gated par film.usage_summary ; repo nil ⇒
// l'Emprise dit `film_unsupported` et les formes se retirent.
//
// `repoRoot` sert aux CATALOGUES du titre (noms d'armes) : vide, les armes s'affichent sous leur
// cle — la degradation ecrite partout ailleurs.
func (s *TimeseriesService) WithUsageSummary(repo port.SessionUsageRepository, repoRoot string) *TimeseriesService {
	s.sessionUsageRepo = repo
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

// attachMigratedSections pose les blocs de l'onglet Usages sur la réponse, depuis le scope canonique déjà
// filtré. Best-effort de bout en bout : chaque producteur rend nil plutôt que de casser la page.
// `equipes` : les participants du scope, déjà lus par la page (cf. lireEquipesDuScope).
func (s *TimeseriesService) attachMigratedSections(
	ctx context.Context, resp *domain.TimeseriesPageResponse,
	filteredCanon []canonical.PlayerMatchRow, locale string, equipes equipesDuScope,
) {
	stop := timing.FromContext(ctx).Section("weapon_range")
	resp.WeaponRange = buildWeaponRangeSection(ctx, weaponRangeQuery{
		Repo: s.weaponRangeRepo, TitleSlug: s.titleSlug, Gamertag: s.gamertag, Rows: filteredCanon,
	})
	stop()
	// LES TROIS LECTURES DU RÉSUMÉ D'USAGE, UNE FOIS pour les blocs qui les lisent (ADR 0036 I4).
	lu := s.lireUsageDuScope(ctx, synthesisMatchIDs(filteredCanon))
	// « Les formes retenues », contexte SOLO : les match_id de la fenêtre. `SelectedGamertags`
	// reste vide — cette page n'a pas d'escouade, et les cartes du contexte escouade ne s'y
	// montent pas.
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
		Lectures:     lu,
	})
	stop()
	s.attachEmprise(ctx, resp, filteredCanon, locale, lu)
	s.attachEmblem(ctx, resp)
	s.attachLives(ctx, resp, filteredCanon)
	s.attachCoordination(ctx, resp, filteredCanon, equipes)
	s.attachMatchRange(ctx, resp, filteredCanon, locale)
}

// lireUsageDuScope fait les trois lectures du résumé d'usage de la fenêtre, sous leur section de
// durée ; nil sans repo (titre sans `film.usage_summary`) ou sans match — chaque bloc dégrade alors
// comme il le fait seul.
func (s *TimeseriesService) lireUsageDuScope(ctx context.Context, matchIDs []string) *squadagg.LecturesUsage {
	if s.sessionUsageRepo == nil || len(matchIDs) == 0 {
		return nil
	}
	defer timing.FromContext(ctx).Section("usage_summary")()
	return squadagg.LireUsage(ctx, s.sessionUsageRepo, matchIDs)
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
	equipes equipesDuScope,
) {
	matchIDs := synthesisMatchIDs(filteredCanon)
	if len(matchIDs) == 0 {
		return
	}
	var teamSize map[string]int
	if equipes.lues {
		teamSize = equipes.tc.TeamSize
	}
	resp.Coordination = buildCoordinationBlock(ctx, coordinationQuery{
		Tactical:   s.coordTactical,
		Appuis:     s.coordAppuis,
		Caps:       s.coordCaps,
		PlayerXUID: s.playerXUID,
		MatchIDs:   matchIDs,
		TeamSize:   teamSize,
		Soirees:    soireesDesRows(filteredCanon),
	})
}

// equipesDuScope — LES PARTICIPANTS DU SCOPE, lus UNE FOIS par requête (lot L5a du plan
// perf, 2026-09-23). La courbe d'équipe du profil d'intensité et l'effectif de camp de la
// coordination posaient la même question aux mêmes matchs : deux lectures identiques de
// `match_participants` pour une réponse.
//
// `lues` faux : port non câblé, joueur inconnu, scope vide ou lecture en échec (journalisée)
// — les deux consommateurs dégradent alors comme avant (courbe d'équipe absente, bloc de
// coordination sans parité), jamais une valeur inventée.
type equipesDuScope struct {
	tc   sessionusage.TeamContext
	lues bool
}

// lireEquipesDuScope fait LA lecture, sous sa propre section (`participants`).
func (s *TimeseriesService) lireEquipesDuScope(ctx context.Context, matchIDs []string) equipesDuScope {
	defer timing.FromContext(ctx).Section("participants")()
	if s.formesUsageRepo == nil || s.playerXUID == "" || len(matchIDs) == 0 {
		return equipesDuScope{}
	}
	participants, err := s.formesUsageRepo.LoadParticipants(ctx, matchIDs)
	if err != nil {
		slog.WarnContext(ctx, "timeseries_participants_en_echec", "degrade", "courbe_equipe,parite_coordination",
			"err", err, "matchs", len(matchIDs))
		return equipesDuScope{}
	}
	return equipesDuScope{tc: sessionusage.BuildTeamContext(s.playerXUID, participants), lues: true}
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
