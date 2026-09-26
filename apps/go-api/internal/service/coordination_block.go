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
// ─── UNE LECTURE PAR REQUÊTE, MÊME QUAND LES SCOPES SONT PLUSIEURS ───────────────────
//
// La page Sessions pose la question à TROIS scopes qui se recouvrent : la session affichée,
// la session comparée, et la période de référence qui les contient d'ordinaire. Trois
// lectures relisaient les matchs de la session affichée dans la référence. Depuis le lot
// L5a du plan perf (2026-09-23, décision D5a.3 : le journal des morts chargé UNE fois par
// requête), `lectureCoordination` lit un ensemble de matchs, se COMPLÈTE des seuls matchs
// qui lui manquent, et chaque bloc se découpe dedans — par le même coordination.Restreindre
// que la maille soirée.
//
// LE DÉCOUPAGE EST EXACT : l'univers, les équipes, le journal et les appuis sont tous des
// lectures PAR MATCH — aucun prédicat ne relie deux matchs — donc la tranche d'une lecture
// large est la lecture de la tranche. La référence reste lue SEULEMENT si elle sert (cf.
// session_page_coordination.go) : la lecture se complète, elle n'anticipe pas.
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
	"sort"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability/timing"
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

// buildCoordinationBlock assemble le bloc d'UN scope, lu pour lui seul. Best-effort de bout
// en bout : un lecteur absent ou en échec rend un bloc INDISPONIBLE avec sa raison machine —
// jamais un bloc de zéros, et jamais l'échec de la page.
//
// nil (champ omis) dans un seul cas : le scope est vide. Il n'y a alors rien à dire, et un
// bloc « indisponible » sur une page sans match serait un message de panne sur un état
// normal.
func buildCoordinationBlock(ctx context.Context, q coordinationQuery) *domain.CoordinationBlock {
	if len(q.MatchIDs) == 0 {
		return nil
	}
	lecture := lireCoordination(ctx, q, q.MatchIDs)
	return lecture.bloc(ctx, q.MatchIDs, q.TeamSize, q.Soirees)
}

// lectureCoordination — le journal des morts et les appuis d'un ENSEMBLE de matchs, lus une
// fois et découpés pour chaque scope qui en a besoin (cf. l'en-tête du fichier).
type lectureCoordination struct {
	q coordinationQuery
	// echec : vide quand la lecture a eu lieu, sinon la raison machine que portera tout bloc
	// découpé dedans (lecteur absent ou capability fermée, journal en échec).
	echec string
	// lus : les matchs DEMANDÉS déjà couverts — un complément ne relit que ce qui manque.
	lus map[string]struct{}
	// entree : l'univers (matchs, mesure, équipes), le journal et les appuis, SANS effectif de
	// camp : celui-ci dépend du scope découpé (cf. bloc).
	entree domain.CoordinationEntree
}

// lireCoordination lit le journal des morts et les appuis de `ids`.
func lireCoordination(ctx context.Context, q coordinationQuery, ids []string) *lectureCoordination {
	l := &lectureCoordination{
		q:      q,
		lus:    make(map[string]struct{}, len(ids)),
		entree: domain.CoordinationEntree{MoiXUID: q.PlayerXUID, Equipes: domain.EquipesParMatch{}},
	}
	if q.Tactical == nil || q.PlayerXUID == "" || !games.JournalDesMortsFiable(q.Caps) {
		l.echec = domain.CoordinationUnsupported
		return l
	}
	l.completer(ctx, ids)
	return l
}

// completer ajoute à la lecture les matchs de `ids` qu'elle ne couvre pas encore : chaque
// match n'est lu qu'une fois par requête. Une lecture en échec le reste (les blocs déjà
// découpés, eux, ne changent pas).
func (l *lectureCoordination) completer(ctx context.Context, ids []string) {
	if l.echec != "" {
		return
	}
	manquants := l.manquants(ids)
	if len(manquants) == 0 {
		return
	}
	stop := timing.FromContext(ctx).Section("kill_events")
	lecture, err := l.q.Tactical.KillEvents(ctx, domain.TacticalQuery{
		PlayerXUID: l.q.PlayerXUID,
		Matchs:     domain.RestreindreAux(manquants),
	})
	stop()
	if err != nil {
		slog.ErrorContext(ctx, "coordination: journal des morts en echec", "err", err,
			"match_count", len(manquants))
		l.echec = domain.CoordinationLoadFailed
		return
	}
	l.ajouter(lecture, l.q.chargerAppuis(ctx, manquants), manquants)
}

// manquants rend les matchs de `ids` que la lecture ne couvre pas encore, sans doublon,
// dans l'ordre reçu.
func (l *lectureCoordination) manquants(ids []string) []string {
	out := make([]string, 0, len(ids))
	vus := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, deja := l.lus[id]; deja {
			continue
		}
		if _, deja := vus[id]; deja {
			continue
		}
		vus[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// ajouter verse une lecture dans l'entrée.
//
// L'UNIVERS FAIT FOI SUR LES MATCHS, pas la liste demandée : un match_id que le lecteur n'a
// pas retenu (hors périmètre du joueur) n'a ni équipes ni drapeau de mesure, et le compter
// au total gonflerait le dénominateur d'une couverture avec des matchs qui n'existent pas
// pour cette lecture.
//
// UN COMPLÉMENT REVERSE SES MATCHS DANS L'ORDRE DU LECTEUR (identifiant croissant, celui de
// son ORDER BY) : l'entrée complétée se lit dans le même ordre qu'une lecture d'un seul
// tenant. Les événements et les appuis, eux, n'ont pas d'ordre à tenir — l'analyse les range
// par match et par instant (coordination.Echanges).
func (l *lectureCoordination) ajouter(
	lecture domain.TacticalKillEvents, appuis []domain.CoordinationAppuiRow, lus []string,
) {
	complement := len(l.entree.Matchs) > 0
	for _, m := range lecture.Univers.Matchs {
		l.entree.Matchs = append(l.entree.Matchs, domain.CoordinationMatch{MatchID: m.MatchID, Mesure: m.Mesure})
	}
	if complement {
		sort.SliceStable(l.entree.Matchs, func(i, j int) bool {
			return l.entree.Matchs[i].MatchID < l.entree.Matchs[j].MatchID
		})
	}
	for matchID, equipes := range lecture.Univers.Equipes {
		l.entree.Equipes[matchID] = equipes
	}
	l.entree.Kills = append(l.entree.Kills, lecture.Events...)
	l.entree.Appuis = append(l.entree.Appuis, appuis...)
	for _, id := range lus {
		l.lus[id] = struct{}{}
	}
}

// bloc assemble le bloc du scope `ids`, découpé dans la lecture, avec l'effectif de camp de
// CE scope. nil sur un scope vide (cf. buildCoordinationBlock).
func (l *lectureCoordination) bloc(
	ctx context.Context, ids []string, teamSize map[string]int, soirees []coordinationSoiree,
) *domain.CoordinationBlock {
	if len(ids) == 0 {
		return nil
	}
	if l.echec != "" {
		return &domain.CoordinationBlock{
			UnavailableReason: l.echec,
			FenetreMs:         coordination.FenetreEchangeMs,
			MatchesTotal:      len(ids),
		}
	}
	entree := l.entree
	if !l.couvreExactement(ids) {
		entree = coordination.Restreindre(l.entree, ids)
	}
	entree.Matchs = avecEffectifs(entree.Matchs, teamSize)

	stop := timing.FromContext(ctx).Section("bloc")
	bloc := coordination.Bloc(entree)
	if len(soirees) > 0 {
		// La frise temporelle ne peint pas de cases par match : ses bâtons sont les
		// soirées, et publier les deux ferait voyager une série que rien ne lit.
		bloc.PerMatch = nil
		bloc.Sessions = soireesDuScope(entree, soirees)
	}
	stop()
	slog.InfoContext(ctx, "coordination_bloc",
		"player", l.q.PlayerXUID, "matchs", bloc.MatchesTotal, "matchs_mesures", bloc.MatchesMeasured,
		"morts_de_camp", bloc.Riposte.TeamDeaths, "morts_ripostees", bloc.Riposte.TeamDeathsAvenged,
		"mes_frags_mesures", bloc.Appui.OnMePrepare.N, "soirees", len(bloc.Sessions))
	return &bloc
}

// couvreExactement dit si `ids` désigne exactement les matchs lus : le découpage serait
// alors une copie à l'identique, et il est épargné (cas de la lecture d'un seul scope).
func (l *lectureCoordination) couvreExactement(ids []string) bool {
	vus := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := l.lus[id]; !ok {
			return false
		}
		vus[id] = struct{}{}
	}
	return len(vus) == len(l.lus)
}

// avecEffectifs rend une COPIE des matchs, chacun avec l'effectif de camp du scope. Une
// copie, jamais une écriture en place : la lecture sert d'autres scopes, aux effectifs
// propres.
func avecEffectifs(matchs []domain.CoordinationMatch, teamSize map[string]int) []domain.CoordinationMatch {
	out := make([]domain.CoordinationMatch, 0, len(matchs))
	for _, m := range matchs {
		m.TeamSize = nil
		if n, ok := teamSize[m.MatchID]; ok && n > 0 {
			size := n
			m.TeamSize = &size
		}
		out = append(out, m)
	}
	return out
}

// chargerAppuis lit les appuis de `ids`. Lecteur absent ou en échec ⇒ aucune ligne, loggé :
// le versant appui a alors des dénominateurs vides (que la couverture publie), et le
// versant riposte reste servi. Dégrader UN sujet vaut mieux que retirer le bloc entier.
func (q coordinationQuery) chargerAppuis(ctx context.Context, ids []string) []domain.CoordinationAppuiRow {
	defer timing.FromContext(ctx).Section("appuis")()
	if q.Appuis == nil {
		slog.DebugContext(ctx, "coordination: aucun lecteur d'appuis cable",
			"player", q.PlayerXUID)
		return nil
	}
	rows, err := q.Appuis.LoadAppuis(ctx, ids)
	if err != nil {
		slog.ErrorContext(ctx, "coordination: appuis en echec", "err", err,
			"match_count", len(ids))
		return nil
	}
	return rows
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
