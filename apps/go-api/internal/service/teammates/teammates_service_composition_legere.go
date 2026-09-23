// Package teammates — teammates_service_composition_legere.go : LES SESSIONS D'UNE
// COMPOSITION, SANS LA PAGE (lot perf L4b, 2026-09-23 —
// .ai/PLAN_PERF_CHARGEMENTS_2026-09-23.md §9, D4b.1).
//
// # LE DÉFAUT
//
// Le sélecteur de sessions de l'Escouade et l'ancrage sur la dernière session de la
// composition ne lisaient que deux champs de POST /pages/teammates
// (composition_sessions, latest_composition_session) : la page entière partait d'abord sur
// tout l'historique, puis l'ancrage la relançait sur la bonne session
// (.ai/ETAT_DES_LIEUX_PERF_CHARGEMENTS_2026-09-23.md §1.2 et C3).
//
// # CE QUE CETTE LECTURE FAIT, ET SEULEMENT CELA
//
// Ces deux champs ne dépendent d'aucun filtre de la page (session, période, cascade) ni
// d'aucune section : ils se calculent sur l'historique COMPLET de la composition. Leurs
// seules lectures :
//   - Q29 (top coéquipiers) : résout un gamertag en xuid, forme l'extraPool de la
//     composition exacte et nomme le coéquipier responsable d'un match écarté ;
//   - Q30 par coéquipier (matchs communs, historique complet) ;
//   - Q32b (équipe alliée du joueur principal) SOUS L'OPTION composition exacte seulement :
//     hors option aucun match n'est écarté, et GetPage ne lit l'équipe que pour ses sections ;
//   - sans coéquipier : l'historique du joueur principal (ses sessions escouade).
//
// Q29 et Q32b lisent l'annuaire du lot L2 (plus de jointure v_gamertag_lookup). Le calcul
// passe ensuite par les MÊMES fonctions que GetPage : intersectSquadRowsByMatchID,
// filterExactComposition, buildCompositionSessionEntries, wrapSessionLabelsAsComposition.
// La parité avec GetPage est verrouillée par teammates_service_composition_legere_test.go.
package teammates

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

// compositionLue : ce que les lectures d'une composition rendent au calcul de ses sessions.
type compositionLue struct {
	topRows []domain.TopTeammateRow
	// roster : intersection des matchs communs (historique complet), AVANT l'option
	// composition exacte — le rosterRowsForTimeline de GetPage.
	roster []domain.SquadMatchRow
	// selectedXUIDs : les xuids des coéquipiers résolus (collectSelectedXUIDs de GetPage).
	selectedXUIDs []string
}

// filtreDeComposition : le roster après l'option composition exacte, et ce qu'il faut pour
// nommer les responsables d'un écart (nil hors option ou sans équipe lisible).
type filtreDeComposition struct {
	kept, excluded []domain.SquadMatchRow
	teamByMatch    map[string]map[string]struct{}
	extraPool      map[string]struct{}
}

// CompositionSessions : cf. l'en-tête du fichier. Mêmes erreurs fatales que GetPage (Q29 ;
// l'historique du joueur principal quand aucun coéquipier n'est désigné), mêmes
// dégradations pour le reste — un coéquipier introuvable, ou dont les matchs communs ne se
// lisent pas, sort de l'intersection ; une équipe alliée illisible laisse le roster non
// filtré — et une requête annulée rend l'erreur du contexte, jamais un résultat partiel.
func (s *TeammatesService) CompositionSessions(
	ctx context.Context, playerXUID string, gamertags []string, exact bool,
) ([]domain.CompositionSessionEntry, string, error) {
	// La copie par requête de GetPage (lot L2) : ce chemin n'emprunte aujourd'hui aucune des
	// lectures qu'elle mémorise (Q32, LoadFor), mais une lecture partagée qui s'y ajouterait
	// passerait par la même mémoire que la page.
	s, _ = s.pourLaRequete()
	if len(gamertags) == 0 {
		return s.sessionsEscouadeDuPrincipal(ctx)
	}
	stop := timing.FromContext(ctx).Section("top_teammates")
	topRows, err := s.repo.LoadTopTeammates(ctx, playerXUID)
	stop()
	if err != nil {
		return nil, "", fmt.Errorf("TeammatesService: %w", err)
	}
	compo, err := s.lireComposition(ctx, playerXUID, gamertags, topRows)
	if err != nil {
		return nil, "", err
	}
	f := s.appliquerCompositionExacte(ctx, playerXUID, compo, exact)
	if err := ctx.Err(); err != nil {
		return nil, "", fmt.Errorf("TeammatesService: requete annulee: %w", err)
	}

	stop = timing.FromContext(ctx).Section("composition_sessions")
	sessions := buildCompositionSessionEntries(f.kept, compo.roster, f.excluded, f.teamByMatch, f.extraPool, topRows)
	stop()
	var latest string
	if len(sessions) > 0 {
		latest = sessions[0].Label
	}
	slog.DebugContext(ctx, "teammates.composition_sessions_resolved",
		"player", s.gamertag,
		"selected_count", len(gamertags),
		"shared_matches", len(f.kept),
		"excluded_matches", len(f.excluded),
		"composition_sessions", len(sessions),
		"latest_session", latest,
	)
	return sessions, latest, nil
}

// lireComposition lit, pour chaque coéquipier désigné, ses matchs communs avec le joueur
// principal (Q30, historique complet), puis les intersecte. Mêmes résolution et écarts que
// la boucle de GetPage (buildTeammateRowWithMatches) : un coéquipier introuvable ou dont la
// lecture échoue n'entre pas dans l'intersection.
func (s *TeammatesService) lireComposition(
	ctx context.Context, playerXUID string, gamertags []string, topRows []domain.TopTeammateRow,
) (compositionLue, error) {
	defer timing.FromContext(ctx).Section("squad_matches")()
	var sets [][]domain.SquadMatchRow
	compo := compositionLue{topRows: topRows}
	for _, gt := range gamertags {
		if err := ctx.Err(); err != nil {
			return compositionLue{}, fmt.Errorf("TeammatesService: requete annulee: %w", err)
		}
		xuid, ok := s.xuidDuCoequipier(ctx, playerXUID, gt, topRows)
		if !ok {
			continue
		}
		rows, err := s.repo.LoadSquadMatches(ctx, playerXUID, xuid)
		if err != nil {
			// DEBUG quand la requête a pris fin (client parti, lot perf L9-go) : pas une panne.
			slog.Log(ctx, observability.LevelUnlessCanceled(ctx, err, slog.LevelError), "teammates_load_squad_matches_failed",
				"player_xuid", playerXUID, "teammate_xuid", xuid, "gamertag", gt, "err", err)
			continue
		}
		sets = append(sets, rows)
		if xuid != "" {
			compo.selectedXUIDs = append(compo.selectedXUIDs, xuid)
		}
	}
	if err := ctx.Err(); err != nil {
		return compositionLue{}, fmt.Errorf("TeammatesService: requete annulee: %w", err)
	}
	compo.roster = intersectSquadRowsByMatchID(sets)
	return compo, nil
}

// xuidDuCoequipier résout un gamertag désigné comme buildTeammateRowWithMatches, dont c'est
// la seconde copie (CLAUDE.md n°6 : deux au plus ; la parité est testée) : top 50 (Q29)
// sans tenir compte de la casse, sinon les alias (LookupXUIDByGamertag). ok=false : le
// coéquipier est introuvable (ou la recherche a échoué) et sort de la composition.
func (s *TeammatesService) xuidDuCoequipier(
	ctx context.Context, playerXUID, gamertag string, topRows []domain.TopTeammateRow,
) (string, bool) {
	for _, r := range topRows {
		if strings.EqualFold(r.Gamertag, gamertag) {
			if r.XUID != "" {
				return r.XUID, true
			}
			break
		}
	}
	resolved, found, err := s.repo.LookupXUIDByGamertag(ctx, gamertag)
	if err != nil {
		slog.WarnContext(ctx, "teammates_gamertag_lookup_failed",
			"player_xuid", playerXUID, "gamertag", gamertag, "err", err)
		return "", false
	}
	if !found {
		// Pas une erreur : GetPage le signale déjà en WARN pour la même composition.
		slog.DebugContext(ctx, "teammates_gamertag_not_found",
			"player_xuid", playerXUID, "gamertag", gamertag, "top_rows_count", len(topRows))
		return "", false
	}
	return resolved, true
}

// appliquerCompositionExacte applique l'option composition exacte au roster comme GetPage :
// l'extraPool (top coéquipiers et amis, hors composition et hors joueur principal), l'équipe
// alliée lue une fois (Q32b) sur les matchs du roster, puis filterExactComposition. Hors
// option, sans coéquipier résolu ou sans équipe lisible : roster intact, aucun écart.
func (s *TeammatesService) appliquerCompositionExacte(
	ctx context.Context, playerXUID string, compo compositionLue, exact bool,
) filtreDeComposition {
	f := filtreDeComposition{kept: compo.roster}
	if !exact || len(compo.selectedXUIDs) == 0 {
		return f
	}
	var friendGTs []string
	if s.friendGamertags != nil {
		friendGTs = s.friendGamertags(ctx)
	}
	f.extraPool = buildExtraPoolXUIDs(compo.topRows, resolveFriendXUIDs(friendGTs, compo.topRows), compo.selectedXUIDs, playerXUID)
	// Échec de lecture journalisé en ERROR par loadMainTeamAllies, comme pour la page (qui le
	// publie en plus dans ses data_issues ; cette réponse-ci n'en porte pas).
	_, f.teamByMatch = s.loadMainTeamAllies(ctx, playerXUID, collectMatchIDs(compo.roster), true, nil)
	if f.teamByMatch == nil {
		return f
	}
	f.kept, f.excluded = filterExactComposition(compo.roster, f.teamByMatch, f.extraPool, compo.selectedXUIDs)
	return f
}

// sessionsEscouadeDuPrincipal : sans coéquipier, les sessions escouade du joueur principal
// sur tout son historique — celles que GetPage publie (wrapSessionLabelsAsComposition de
// SessionLabels.Squad) — et aucune dernière session : GetPage n'en publie pas hors
// composition.
func (s *TeammatesService) sessionsEscouadeDuPrincipal(ctx context.Context) ([]domain.CompositionSessionEntry, string, error) {
	if s.playerMatchesRepo == nil || s.titleSlug == "" || s.gamertag == "" {
		return nil, "", fmt.Errorf("TeammatesService: PlayerMatchesRepo non câblé (P4.3 finale exige le wiring DI)")
	}
	stop := timing.FromContext(ctx).Section("player_matches")
	canonicalRows, err := s.playerMatchesRepo.LoadPlayerMatches(ctx, s.titleSlug, s.gamertag, port.PlayerMatchFilters{})
	stop()
	if err != nil {
		return nil, "", fmt.Errorf("TeammatesService synthesis: %w", err)
	}
	stop = timing.FromContext(ctx).Section("composition_sessions")
	labels := extractSynthesisSessionLabels(analysis.SynthesisMatchRowsFromCanonical(canonicalRows))
	sessions := wrapSessionLabelsAsComposition(labels.Squad)
	stop()
	return sessions, "", nil
}
