// Package service — coordination_block.go : LE BLOC « COORDINATION », servi À L'IDENTIQUE
// à la page Sessions (une case par match) et aux Séries temporelles (un point par soirée).
//
// UN SEUL PRODUCTEUR POUR LES DEUX PAGES, et c'est le point du fichier. Les deux posent la
// même question à deux mailles ; deux constructeurs auraient donné deux définitions de
// « morts de mon camp » libres de diverger au premier correctif — le défaut que la
// migration des sections transverses (timeseries_service_sections.go) a déjà évité pour la
// portée des engagements et l'usage d'équipement.
//
// ─── DEUX LECTURES, JAMAIS UNE PAR MATCH ──────────────────────────────────────────────
//
//	le JOURNAL DES MORTS   port.TacticalRepository.KillEvents, UNE fois, sur la liste
//	                       blanche du scope : il rend l'univers (matchs retenus, drapeau
//	                       « mesuré », table des équipes) ET les événements ;
//	les APPUIS             port.CoordinationRepository.LoadAppuis, UNE fois, sur la même
//	                       liste.
//
// La maille SOIRÉE se découpe ensuite EN GO (coordination.Restreindre) : une requête par
// soirée aurait été un N+1 sur la même fenêtre.
//
// ─── L'EFFECTIF DE CAMP VIENT DE L'APPELANT ───────────────────────────────────────────
//
// Les deux pages construisent déjà un `sessionusage.TeamContext` pour leurs autres blocs.
// Le recompter ici depuis la table des équipes de l'univers en aurait donné une SECONDE
// définition (tous les participants à camp connu, contre les joueurs présents à la fin) :
// la parité du bloc et le `team_size` de la bande d'usage auraient affiché deux effectifs
// pour le même match.
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// coordinationSoiree — une soirée de la frise temporelle : son libellé et ses matchs, déjà
// ordonnés chronologiquement par l'appelant.
type coordinationSoiree struct {
	Label    string
	MatchIDs []string
}

// coordinationQuery — tout ce dont le bloc a besoin. `Soirees` vide ⇒ maille MATCH (page
// Sessions) ; renseigné ⇒ maille SOIRÉE (Séries temporelles).
type coordinationQuery struct {
	Tactical   port.TacticalRepository
	Appuis     port.CoordinationRepository
	Caps       games.CapabilityMap
	PlayerXUID string
	MatchIDs   []string
	// TeamSize : effectif de MON camp par match, tel que l'appelant l'a déjà calculé
	// (sessionusage.TeamContext.TeamSize). Clé absente = camp inconnu (FFA) : le match
	// n'a alors pas de parité, jamais un 1 inventé (réserve R1).
	TeamSize map[string]int
	Soirees  []coordinationSoiree
}

// buildCoordinationBlock assemble le bloc. Best-effort de bout en bout : un lecteur absent
// ou en échec rend un bloc INDISPONIBLE avec sa raison machine — jamais un bloc de zéros,
// et jamais l'échec de la page.
//
// nil (champ omis) dans un seul cas : le scope est vide. Il n'y a alors rien à dire, et un
// bloc « indisponible » sur une page sans match serait un message de panne sur un état
// normal.
func buildCoordinationBlock(ctx context.Context, q coordinationQuery) *domain.CoordinationBlock {
	if len(q.MatchIDs) == 0 {
		return nil
	}
	if q.Tactical == nil || q.PlayerXUID == "" || !games.JournalDesMortsFiable(q.Caps) {
		return &domain.CoordinationBlock{
			UnavailableReason: domain.CoordinationUnsupported,
			FenetreMs:         coordination.FenetreEchangeMs,
			MatchesTotal:      len(q.MatchIDs),
		}
	}
	lecture, err := q.Tactical.KillEvents(ctx, domain.TacticalQuery{
		PlayerXUID: q.PlayerXUID,
		Matchs:     domain.RestreindreAux(q.MatchIDs),
	})
	if err != nil {
		slog.ErrorContext(ctx, "coordination: journal des morts en echec", "err", err,
			"match_count", len(q.MatchIDs))
		return &domain.CoordinationBlock{
			UnavailableReason: domain.CoordinationLoadFailed,
			FenetreMs:         coordination.FenetreEchangeMs,
			MatchesTotal:      len(q.MatchIDs),
		}
	}

	entree := coordinationEntree(q, lecture)
	entree.Appuis = q.chargerAppuis(ctx)

	bloc := coordination.Bloc(entree)
	if len(q.Soirees) > 0 {
		// La frise temporelle ne peint pas de cases par match : ses bâtons sont les
		// soirées, et publier les deux ferait voyager une série que rien ne lit.
		bloc.PerMatch = nil
		bloc.Sessions = soireesDuScope(entree, q.Soirees)
	}
	slog.InfoContext(ctx, "coordination_bloc",
		"player", q.PlayerXUID, "matchs", bloc.MatchesTotal, "matchs_mesures", bloc.MatchesMeasured,
		"morts_de_camp", bloc.Riposte.TeamDeaths, "morts_ripostees", bloc.Riposte.TeamDeathsAvenged,
		"mes_frags_mesures", bloc.Appui.OnMePrepare.N, "soirees", len(bloc.Sessions))
	return &bloc
}

// chargerAppuis lit les appuis du scope. Lecteur absent ou en échec ⇒ aucune ligne, loggé :
// le versant appui a alors des dénominateurs vides (que la couverture publie), et le
// versant riposte reste servi. Dégrader UN sujet vaut mieux que retirer le bloc entier.
func (q coordinationQuery) chargerAppuis(ctx context.Context) []domain.CoordinationAppuiRow {
	if q.Appuis == nil {
		slog.DebugContext(ctx, "coordination: aucun lecteur d'appuis cable",
			"player", q.PlayerXUID)
		return nil
	}
	rows, err := q.Appuis.LoadAppuis(ctx, q.MatchIDs)
	if err != nil {
		slog.ErrorContext(ctx, "coordination: appuis en echec", "err", err,
			"match_count", len(q.MatchIDs))
		return nil
	}
	return rows
}

// coordinationEntree projette la lecture de base en entrée du calcul pur.
//
// L'UNIVERS FAIT FOI SUR LES MATCHS, pas la liste demandée : un match_id que le lecteur
// n'a pas retenu (hors périmètre du joueur) n'a ni équipes ni drapeau de mesure, et le
// compter au total gonflerait le dénominateur d'une couverture avec des matchs qui
// n'existent pas pour cette lecture.
func coordinationEntree(q coordinationQuery, lecture domain.TacticalKillEvents) domain.CoordinationEntree {
	out := domain.CoordinationEntree{
		MoiXUID: q.PlayerXUID,
		Equipes: lecture.Univers.Equipes,
		Kills:   lecture.Events,
		Matchs:  make([]domain.CoordinationMatch, 0, len(lecture.Univers.Matchs)),
	}
	for _, m := range lecture.Univers.Matchs {
		match := domain.CoordinationMatch{MatchID: m.MatchID, Mesure: m.Mesure}
		if n, ok := q.TeamSize[m.MatchID]; ok && n > 0 {
			size := n
			match.TeamSize = &size
		}
		out.Matchs = append(out.Matchs, match)
	}
	return out
}

// soireesDuScope rend un point par soirée, dans l'ordre reçu (chronologique).
//
// UNE SOIRÉE SANS AUCUN MATCH MESURÉ N'A PAS DE POINT : elle n'a pas un taux nul, elle n'a
// pas de taux. Même doctrine que l'omission d'un bloc entier — et la trame du temps ne
// ment pas pour autant, puisque le bâton absent se distingue d'un bâton à zéro.
func soireesDuScope(entree domain.CoordinationEntree, soirees []coordinationSoiree) []domain.CoordinationSessionPoint {
	out := make([]domain.CoordinationSessionPoint, 0, len(soirees))
	for _, s := range soirees {
		bloc := coordination.Bloc(coordination.Restreindre(entree, s.MatchIDs))
		if !bloc.Available {
			continue
		}
		out = append(out, domain.CoordinationSessionPoint{
			SessionLabel:    s.Label,
			MatchesMeasured: bloc.MatchesMeasured,
			MatchesTotal:    bloc.MatchesTotal,
			Riposte:         bloc.Riposte,
			Appui:           bloc.Appui,
		})
	}
	return out
}
