// Package service — session_page_usage.go : le bloc « usages d'équipement,
// socles et objectifs » de la page détail de session (chantier session-usage S2,
// .ai/HANDOFF_SESSION_USAGE_BDD_2026-09-04.md §5/S2).
//
// PATRON D'ATTACHEMENT (miroir IntensityRows/FirstBlood) : repo optionnel injecté
// à la DI, attaché à la réponse existante de POST .../pages/sessions/detail —
// PAS d'endpoint dédié. Capability film.usage_summary absente ⇒ repo nil ⇒ bloc
// Available=false avec raison machine (réponse partielle propre, jamais un 500).
// Le contexte Solo/Escouade est résolu EN AMONT par Filters.MatchContext : le
// bloc agrège les matchs de la session AFFICHÉE ; en contexte escouade il porte
// en plus une ligne par coéquipier suivi.
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/teammates"
)

// objectiveRoleRowsLoader est la capability OPTIONNELLE du repo objectifs :
// lignes (match, joueur, famille) projetées par rôle, les deux camps. Seul
// duckdb.ObjectiveStatsRepo l'implémente ; sans elle (titre sans
// match.objective.stats), le sous-bloc Objectives est simplement omis.
type objectiveRoleRowsLoader interface {
	LoadObjectiveRoleRows(ctx context.Context, matchIDs []string) ([]sessionusage.ObjectiveRow, error)
}

// flagGrabsNetLoader est la capability OPTIONNELLE qui sert les prises nettes de
// drapeau. Interface SÉPARÉE de celle ci-dessus, et non une méthode de plus :
// les deux grandeurs viennent de deux tables alimentées par deux producteurs
// (l'API pour les rôles, le film pour les prises nettes), et un montage qui n'a
// que l'une doit pouvoir servir l'autre sans l'implémenter.
type flagGrabsNetLoader interface {
	LoadFlagGrabsNet(ctx context.Context, matchIDs []string) ([]sessionusage.FlagGrabsNetRow, error)
}

// WithSessionUsage injecte le repo du résumé d'usage S1 (vues _latest), le xuid
// du joueur suivi et le résolveur d'amis configurés (restriction des coéquipiers
// suivis, même source que l'accueil). Câblé UNIQUEMENT pour les titres portant
// film.usage_summary (registry_pages, jamais slug==) — nil ⇒ bloc indisponible.
//
// repoRoot sert au SEUL catalogue d'armes du titre (session_page_usage_labels.go) :
// il voyage avec ce wiring-là parce qu'il n'a d'utilité que pour ce bloc. Vide ⇒ les
// familles de socle gardent leur clé hexadécimale, rendu valide et sans erreur.
func (s *SessionPageService) WithSessionUsage(
	repo port.SessionUsageRepository, xuid string, friends teammates.FriendGamertagsResolver,
	repoRoot string,
) *SessionPageService {
	s.sessionUsageRepo = repo
	s.usageXUID = xuid
	s.usageFriends = friends
	s.repoRoot = repoRoot
	return s
}

// attachSessionUsage attache le bloc usage de la session COURANTE et, en mode
// comparaison, celui de la session COMPARÉE — MIROIR d'attachSessionEventBlocks, qui
// sert déjà ses deux blocs event-based aux deux sessions.
//
// POURQUOI LES DEUX (revue de lisibilité 2026-09-09, D8) : le drawer de comparaison
// montrait ces cartes à gauche et rien à droite, non par choix de lecture mais parce que
// le bloc n'était jamais calculé pour la session comparée. Les deux blocs sont
// comparables sans retraitement — toutes leurs grandeurs sont normalisées (cadences par
// dix minutes, parts en pourcentage), aucune n'est un total de session.
//
// `compareMatches` vide (drawer fermé) ⇒ CompareUsage reste nil, et rien ne se rend à
// droite : l'absence dit tout, aucun drapeau n'est nécessaire.
func (s *SessionPageService) attachSessionUsage(
	ctx context.Context, resp *domain.SessionPageResponse,
	matches, compareMatches []legacymatch.StatsMatchRow, matchContext, locale string,
) {
	resp.Usage = s.buildSessionUsage(ctx, matches, matchContext, locale)
	if len(compareMatches) > 0 {
		resp.CompareUsage = s.buildSessionUsage(ctx, compareMatches, matchContext, locale)
	}
}

// buildSessionUsage calcule le bloc usage d'UNE session. Best-effort : une erreur de
// lecture est loggée PUIS dégradée en Available=false (raison machine) — jamais d'échec
// de la page. Session sans match ⇒ nil.
func (s *SessionPageService) buildSessionUsage(
	ctx context.Context, matches []legacymatch.StatsMatchRow, matchContext, locale string,
) *domain.SessionUsageBlock {
	if len(matches) == 0 {
		return nil
	}
	if s.sessionUsageRepo == nil || s.usageXUID == "" {
		return &domain.SessionUsageBlock{
			UnavailableReason: domain.SessionUsageUnsupported, MatchesTotal: len(matches),
		}
	}
	ids := matchIDsFromStatsRows(matches)
	films, filmsErr := s.sessionUsageRepo.LoadUsageFilms(ctx, ids)
	players, playersErr := s.sessionUsageRepo.LoadUsagePlayers(ctx, ids)
	participants, partErr := s.sessionUsageRepo.LoadParticipants(ctx, ids)
	for _, err := range []error{filmsErr, playersErr, partErr} {
		if err != nil {
			slog.ErrorContext(ctx, "session page: usage block load failed", "err", err,
				"match_count", len(ids))
			return &domain.SessionUsageBlock{
				UnavailableReason: domain.SessionUsageLoadFailed, MatchesTotal: len(matches),
			}
		}
	}

	tc := sessionusage.BuildTeamContext(s.usageXUID, participants)
	in := buildSessionUsageInput(s.usageXUID, matches, films, players, tc)
	// Un match mesuré sans échelle de temps (duration_ms <= 0 et aucun repli
	// stats) est exclu des cadences par ComputeUsage — jamais en silence.
	if n := countMeasuredWithoutDuration(in.Matches); n > 0 {
		slog.WarnContext(ctx, "session page: measured matches without time scale excluded from rates",
			"matches_without_duration", n, "match_count", len(ids))
	}
	var squad []domain.SessionUsageSquadPlayer
	if matchContext == domain.MatchContextSquad {
		squad = sessionusage.ResolveTrackedSquad(s.usageXUID, ids, participants, s.friendGamertags(ctx))
		for _, member := range squad {
			in.SquadXUIDs = append(in.SquadXUIDs, member.XUID)
		}
	}
	block := sessionusage.ComputeUsage(in)
	block.SquadPlayers = squad
	s.attachSessionObjectives(ctx, &block, ids, tc, in.SquadXUIDs)
	s.attachPadTiers(ctx, &block, ids, tc)
	s.resolvePadFamilyLabels(ctx, &block, locale)
	s.resolvePadTierWeaponLabels(ctx, &block, locale)
	return &block
}

// attachSessionObjectives renseigne le sous-bloc objectifs (lecture seule de
// match_objective_stats_latest, les deux camps). Omis sans repo objectifs (titre
// sans match.objective.stats) ou sur erreur (best-effort, loggé).
func (s *SessionPageService) attachSessionObjectives(
	ctx context.Context, block *domain.SessionUsageBlock,
	matchIDs []string, tc sessionusage.TeamContext, squadXUIDs []string,
) {
	loader, ok := s.objectiveIndex.(objectiveRoleRowsLoader)
	if !ok {
		return
	}
	rows, err := loader.LoadObjectiveRoleRows(ctx, matchIDs)
	if err != nil {
		slog.WarnContext(ctx, "session page: objective role rows unavailable", "err", err)
		return
	}
	block.Objectives = sessionusage.ComputeObjectives(sessionusage.ObjectivesInput{
		PlayerXUID: s.usageXUID,
		SquadXUIDs: squadXUIDs,
		Rows:       rows,
		PlayerTeam: tc.PlayerTeam,
		TeamOf:     tc.TeamOf,
		TeamSize:   tc.TeamSize,
		LobbySize:  tc.LobbySize,
	})
	s.attachFlagGrabsNet(ctx, block, matchIDs, tc)
}

// attachFlagGrabsNet ajoute les PRISES NETTES de drapeau au bloc objectifs.
//
// APRÈS ComputeObjectives, et pas dedans : la grandeur vient d'une autre table
// et d'un autre producteur (le film), elle n'est mesurée que sur les matchs dont
// l'artefact a été lu, et elle porte ses propres dénominateurs. Sans bloc
// objectifs (scope sans mode à objectif), il n'y a rien à attacher.
//
// Best-effort et DIT : montage sans ce loader (titre qui ne produit pas la
// grandeur) ⇒ silence ; lecture en échec ⇒ WARN, le reste du bloc est servi.
func (s *SessionPageService) attachFlagGrabsNet(
	ctx context.Context, block *domain.SessionUsageBlock,
	matchIDs []string, tc sessionusage.TeamContext,
) {
	if block.Objectives == nil {
		return
	}
	loader, ok := s.objectiveIndex.(flagGrabsNetLoader)
	if !ok {
		return
	}
	rows, err := loader.LoadFlagGrabsNet(ctx, matchIDs)
	if err != nil {
		slog.WarnContext(ctx, "session page: prises nettes indisponibles", "err", err)
		return
	}
	block.Objectives.FlagGrabsNet = sessionusage.ComputeFlagGrabsNet(sessionusage.FlagGrabsNetInput{
		Rows:              rows,
		PlayerXUID:        s.usageXUID,
		MatchesFlagFamily: matchsFamilleDrapeau(block.Objectives),
		PlayerTeam:        tc.PlayerTeam,
		TeamOf:            tc.TeamOf,
	})
}

// matchsFamilleDrapeau — les matchs de la famille DRAPEAU de la session, dénominateur
// de couverture des prises nettes.
//
// PAS `MatchesWithObjectives`, ET C'EST UNE CORRECTION DE REVUE : ce compteur-là porte
// TOUTES les familles à objectif. L'employer disait « mesuré sur 2 des 7 matchs » sur une
// session de cinq parties de Bastion et deux de drapeau — et attribuait au film l'absence
// de cinq matchs qui n'ont tout simplement pas de drapeau.
//
// La famille se lit sur le bloc déjà calculé (`ComputeObjectives` la publie avec son
// compte de matchs) : aucune requête de plus, et un seul discriminant de famille dans
// tout le produit.
func matchsFamilleDrapeau(obj *domain.SessionObjectivesBlock) int {
	if obj == nil {
		return 0
	}
	for i := range obj.Families {
		if obj.Families[i].Family == string(narrative.FamilyCTF) {
			return obj.Families[i].Matches
		}
	}
	return 0
}

// friendGamertags résout la liste des amis configurés (vide = aucune
// restriction sur les coéquipiers suivis, même convention que l'accueil).
func (s *SessionPageService) friendGamertags(ctx context.Context) []string {
	if s.usageFriends == nil {
		return nil
	}
	return s.usageFriends(ctx)
}

// countMeasuredWithoutDuration compte les matchs mesurés restés SANS durée
// après le repli stats : ils sortent des cadences (règle C6, voir ComputeUsage)
// et l'appelant le logge une fois par session.
func countMeasuredWithoutDuration(matches []sessionusage.MatchInput) int {
	n := 0
	for i := range matches {
		if matches[i].Measured && matches[i].DurationSeconds <= 0 {
			n++
		}
	}
	return n
}

// buildSessionUsageInput assemble l'entrée de sessionusage.ComputeUsage : un
// MatchInput par match de la session (ordre d'affichage), mesuré s'il a une
// ligne film. L'assemblage commun vit dans sessionusage.BuildMatchInputs (le bloc
// de période s'en sert aussi) ; ce qui suit n'ajoute que ce qui est PROPRE à la
// session — l'échelle de temps des cadences, avec repli sur la durée côté stats
// quand le film n'en a pas (un match resté sans durée est exclu des cadences par
// ComputeUsage, totaux et parts conservés) — et les compteurs de grain match.
func buildSessionUsageInput(
	playerXUID string, matches []legacymatch.StatsMatchRow,
	films map[string]sessionusage.FilmRow, players []sessionusage.PlayerRow,
	tc sessionusage.TeamContext,
) sessionusage.Input {
	in := sessionusage.Input{
		PlayerXUID: playerXUID,
		Matches:    sessionusage.BuildMatchInputs(matchIDsFromStatsRows(matches), films, players, tc),
	}
	// BuildMatchInputs conserve l'ordre reçu : in.Matches[i] est le match matches[i].
	for i := range in.Matches {
		m := &in.Matches[i]
		if !m.Measured {
			continue
		}
		film := films[m.MatchID]
		m.DurationSeconds = float64(film.DurationMS) / 1000
		if m.DurationSeconds <= 0 && matches[i].TimePlayedSeconds != nil {
			m.DurationSeconds = float64(*matches[i].TimePlayedSeconds)
		}
		m.PadUnnamed = film.PadUnnamed
		m.PowerupPickups = film.PowerupPickups
	}
	return in
}

// attachPadTiers ajoute au bloc usage LES PRISES DE SOCLE PAR NIVEAU D'ARME.
//
// EN DEHORS DE ComputeUsage, et pas dedans : la grandeur vient d'une autre table
// (`match_pad_pickups_by_tier`) et d'un autre producteur (une passe de lecture
// d'artefact distincte du résumé d'usage). Elle n'est mesurée que sur les matchs
// dont cette passe a eu lieu, et elle porte ses propres dénominateurs — les
// verser dans les métriques du bloc ferait compter un match non projeté comme un
// match sans prise.
//
// Best-effort et DIT : montage sans ce loader (titre qui ne produit pas la
// grandeur) ⇒ silence ; lecture en échec ⇒ WARN, le reste du bloc est servi.
func (s *SessionPageService) attachPadTiers(
	ctx context.Context, block *domain.SessionUsageBlock,
	matchIDs []string, tc sessionusage.TeamContext,
) {
	if s.sessionUsageRepo == nil {
		return
	}
	rows, err := s.sessionUsageRepo.LoadPadTiers(ctx, matchIDs)
	if err != nil {
		slog.WarnContext(ctx, "session page: niveaux d'armes indisponibles", "err", err)
		return
	}
	block.PadTiers = sessionusage.ComputePadTiers(sessionusage.PadTiersInput{
		Rows:       rows,
		PlayerXUID: s.usageXUID,
		PlayerTeam: tc.PlayerTeam,
		TeamOf:     tc.TeamOf,
	})
}
