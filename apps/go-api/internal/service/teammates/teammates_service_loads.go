// Package teammates — teammates_service_loads.go : LES LECTURES PARTAGÉES D'UNE REQUÊTE
// (lot perf L2, 2026-09-23).
//
// # LE DÉFAUT SUPPRIMÉ
//
// Chaque bloc de la page Escouade lisait lui-même ce dont il avait besoin : les événements
// d'impact (Q32) étaient lus QUATRE fois sur les mêmes matchs — matrice d'impact, profil
// d'intensité, séries de performance, premier frag —, à 1,7-2,4 s la lecture sur la copie de
// production tant qu'elle joignait v_gamertag_lookup (C1/C4 de l'état des lieux).
//
// # LA MÉCANIQUE
//
// GetPage travaille sur une COPIE du service (pourLaRequete) dont les lecteurs partagés sont
// enveloppés par une mémoire propre à la requête : le code des blocs ne change pas, leurs
// appels identiques retombent sur la même lecture. La lecture elle-même est faite AVANT les
// blocs, sous sa propre section de durée (sections = feuilles, cf. lot L1) : les sections des
// blocs ne mesurent plus que leur calcul.
//
// La mémoire vit le temps d'une requête et n'est jamais partagée entre deux : une lecture de
// la requête suivante voit la base du moment.
package teammates

import (
	"context"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"sync"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/squadagg"
)

// lecturesDeLaPage : les lectures qu'au moins deux blocs de la page consomment, faites une
// fois par requête. Sûre en concurrence (un bloc parallélisé la verrait entière).
type lecturesDeLaPage struct {
	mu     sync.Mutex
	repo   port.SquadRepository   // le lecteur réel (jamais l'enveloppe)
	loader squadagg.SquadV2Loader // idem pour l'historique des membres
	// slug, principal : le titre et le joueur de la page (clés du préchargement).
	slug, principal string
	// impacts : Q32 par GROUPES de match_id (clé canonique, cf. cleDesGroupes). Les blocs
	// passent tous les matchs uniques de la population escouade, dans des ordres
	// différents : les ensembles, pas les listes, identifient la lecture.
	impacts map[string]lectureImpacts
	// membres : LoadFor par (titre, gamertag), filtres nuls — la seule forme que la page
	// appelle ; tout autre filtre passe tel quel au chargeur réel.
	membres map[string]lectureMembre
}

// lectureMembre : l'historique canonique d'UN membre, erreur comprise. Les blocs le lisent
// sans le modifier (vérifié : filtres et agrégats rendent de nouvelles tranches).
type lectureMembre struct {
	rows []canonical.PlayerMatchRow
	err  error
}

// lectureImpacts : le résultat d'UNE lecture Q32, erreur comprise — chaque bloc
// consommateur reçoit la même et la journalise comme avant. groupes : la clé (cleDEnsemble)
// de chaque groupe lu.
type lectureImpacts struct {
	rows    []domain.ImpactEventRow
	err     error
	groupes map[string]bool
}

// pourLaRequete rend une copie du service dont les lecteurs partagés passent par la mémoire
// de CETTE requête, et cette mémoire. Un lecteur non câblé (nil) le reste : les blocs gardent
// leur dégradation d'origine.
func (s *TeammatesService) pourLaRequete() (*TeammatesService, *lecturesDeLaPage) {
	l := &lecturesDeLaPage{
		repo: s.repo, loader: s.squadLoader, slug: s.titleSlug, principal: s.gamertag,
		impacts: map[string]lectureImpacts{}, membres: map[string]lectureMembre{},
	}
	cp := *s
	if s.repo != nil {
		cp.repo = repoDeLaPage{SquadRepository: s.repo, lectures: l}
	}
	if s.squadLoader != nil {
		cp.squadLoader = loaderDeLaPage{SquadV2Loader: s.squadLoader, lectures: l}
	}
	return &cp, l
}

// precharger fait, AVANT les blocs, les lectures qu'ils partagent (D2.2, D2.3) :
//   - l'historique de chaque membre — les coéquipiers sélectionnés dès qu'il y en a (le
//     bandeau les lit toujours), le joueur principal quand la population escouade existe
//     (radar et séries de performance, seuls à le relire par LoadFor) ;
//   - les événements d'impact, en une lecture, de la population escouade et de chaque soirée
//     précédente des points d'impact (impactGroupes, cf. groupesDImpact) : les blocs qui ne
//     lisent que la population en reçoivent la part, sans seconde lecture (impactsDe).
//
// Exactement les lectures que les blocs faisaient au moins une fois : aucune de plus.
func (l *lecturesDeLaPage) precharger(ctx context.Context, selected []string, impactGroupes [][]string) {
	if len(selected) > 0 {
		membres := selected
		if len(impactGroupes) > 0 {
			membres = append([]string{l.principal}, selected...)
		}
		l.prechargerMembres(ctx, membres)
	}
	if ctx.Err() != nil {
		return // requête annulée (D2.7) : GetPage rend l'erreur
	}
	l.prechargerImpacts(ctx, impactGroupes)
}

// loaderDeLaPage : le chargeur d'historiques de la requête. Seul LoadFor est partagé ; tout le
// reste passe tel quel au chargeur réel.
type loaderDeLaPage struct {
	squadagg.SquadV2Loader
	lectures *lecturesDeLaPage
}

// LoadFor sert l'historique d'un membre, lu une fois par requête (D2.3). Des filtres non nuls
// — que la page n'emploie pas — passent au chargeur réel sans mémoire.
func (d loaderDeLaPage) LoadFor(
	ctx context.Context, slug, gamertag string, filters port.PlayerMatchFilters,
) ([]canonical.PlayerMatchRow, error) {
	if !reflect.ValueOf(filters).IsZero() {
		return d.SquadV2Loader.LoadFor(ctx, slug, gamertag, filters)
	}
	return d.lectures.membre(ctx, slug, gamertag)
}

// membre rend l'historique d'un membre, lu au premier appel puis servi de mémoire.
func (l *lecturesDeLaPage) membre(ctx context.Context, slug, gamertag string) ([]canonical.PlayerMatchRow, error) {
	cle := slug + "\x00" + gamertag
	l.mu.Lock()
	defer l.mu.Unlock()
	lu, ok := l.membres[cle]
	if !ok {
		rows, err := l.loader.LoadFor(ctx, slug, gamertag, port.PlayerMatchFilters{})
		lu = lectureMembre{rows: rows, err: err}
		l.membres[cle] = lu
	}
	return lu.rows, lu.err
}

// prechargerMembres lit l'historique de chaque membre UNE fois (D2.3), sous la section
// `squad_members`. Une erreur est servie telle quelle à chaque bloc qui relit ce membre, et il
// la journalise comme avant ; ici elle n'est que tracée.
func (l *lecturesDeLaPage) prechargerMembres(ctx context.Context, gamertags []string) {
	if l.loader == nil || len(gamertags) == 0 {
		return
	}
	defer timing.FromContext(ctx).Section("squad_members")()
	for _, gt := range gamertags {
		if ctx.Err() != nil {
			return // requête annulée (D2.7)
		}
		if _, err := l.membre(ctx, l.slug, gt); err != nil {
			slog.DebugContext(ctx, "teammates_squad_member_load_failed", "gamertag", gt, "err", err)
		}
	}
}

// repoDeLaPage : le lecteur Escouade de la requête. Seule la lecture des événements d'impact
// (Q32) est partagée ; tout le reste passe tel quel au lecteur réel.
type repoDeLaPage struct {
	port.SquadRepository
	lectures *lecturesDeLaPage
}

// LoadImpactEvents sert la lecture Q32 de la requête pour cet ensemble de matchs (un groupe).
func (r repoDeLaPage) LoadImpactEvents(ctx context.Context, matchIDs []string) ([]domain.ImpactEventRow, error) {
	return r.lectures.impactsDe(ctx, [][]string{matchIDs})
}

// LoadImpactEventsParGroupes sert la lecture Q32 de la requête pour ces groupes.
func (r repoDeLaPage) LoadImpactEventsParGroupes(ctx context.Context, groupes [][]string) ([]domain.ImpactEventRow, error) {
	return r.lectures.impactsDe(ctx, groupes)
}

// impactsDe rend Q32 pour des groupes de matchs, lu au premier appel puis servi de mémoire.
// Des groupes qui sont chacun un groupe d'une lecture déjà faite en reçoivent les lignes de
// leurs matchs (dans l'ordre de la lecture) au lieu d'être relus : le dépôt rend à chaque
// groupe ce que sa lecture dédiée lui aurait rendu, repli des frags reconstitués compris
// (décidé par groupe, port.SquadRepository.LoadImpactEventsParGroupes). Un groupe qui n'est
// pas un groupe lu (partie ou réunion de groupes) est relu : sa décision de repli peut
// différer. Chaque appelant reçoit sa propre copie de la tranche : un bloc qui la
// réordonnerait ne changerait pas ce que voient les autres.
func (l *lecturesDeLaPage) impactsDe(ctx context.Context, groupes [][]string) ([]domain.ImpactEventRow, error) {
	groupes = groupesNonVides(groupes)
	cle := cleDesGroupes(groupes)
	l.mu.Lock()
	defer l.mu.Unlock()
	if lu, ok := l.impacts[cle]; ok {
		return slices.Clone(lu.rows), lu.err
	}
	if large, ok := l.lectureQuiCouvre(groupes); ok {
		return lignesDesMatchs(large.rows, groupes), large.err
	}
	rows, err := l.repo.LoadImpactEventsParGroupes(ctx, groupes)
	lu := lectureImpacts{rows: rows, err: err, groupes: make(map[string]bool, len(groupes))}
	for _, g := range groupes {
		lu.groupes[cleDEnsemble(g)] = true
	}
	l.impacts[cle] = lu
	return slices.Clone(lu.rows), lu.err
}

// lectureQuiCouvre rend une lecture Q32 déjà faite dont chaque groupe demandé est un groupe.
// Appelée sous l.mu.
func (l *lecturesDeLaPage) lectureQuiCouvre(groupes [][]string) (lectureImpacts, bool) {
	for _, lu := range l.impacts {
		couvre := true
		for _, g := range groupes {
			if !lu.groupes[cleDEnsemble(g)] {
				couvre = false
				break
			}
		}
		if couvre {
			return lu, true
		}
	}
	return lectureImpacts{}, false
}

// lignesDesMatchs — les lignes des matchs des groupes, dans l'ordre de la lecture (copie).
func lignesDesMatchs(rows []domain.ImpactEventRow, groupes [][]string) []domain.ImpactEventRow {
	garde := map[string]bool{}
	for _, g := range groupes {
		for _, id := range g {
			garde[id] = true
		}
	}
	out := make([]domain.ImpactEventRow, 0, len(rows))
	for _, r := range rows {
		if garde[r.MatchID] {
			out = append(out, r)
		}
	}
	return out
}

// groupesNonVides — les groupes qui ont au moins un match.
func groupesNonVides(groupes [][]string) [][]string {
	out := make([][]string, 0, len(groupes))
	for _, g := range groupes {
		if len(g) > 0 {
			out = append(out, g)
		}
	}
	return out
}

// cleDesGroupes : la clé d'une lecture par groupes — les clés des groupes, triées.
func cleDesGroupes(groupes [][]string) string {
	cles := make([]string, 0, len(groupes))
	for _, g := range groupes {
		cles = append(cles, cleDEnsemble(g))
	}
	slices.Sort(cles)
	return strings.Join(cles, "\x01")
}

// prechargerImpacts lit Q32 UNE fois pour les groupes donnés (D2.2), sous la section
// `impact_events_shared`, avant les blocs qui la consomment. Sans lecteur ou sans match : rien
// à lire, chaque bloc garde sa dégradation.
func (l *lecturesDeLaPage) prechargerImpacts(ctx context.Context, groupes [][]string) {
	groupes = groupesNonVides(groupes)
	if l.repo == nil || len(groupes) == 0 {
		return
	}
	defer timing.FromContext(ctx).Section("impact_events_shared")()
	if _, err := l.impactsDe(ctx, groupes); err != nil {
		// Chaque bloc consommateur reçoit cette erreur et la journalise lui-même en WARN.
		slog.DebugContext(ctx, "teammates_impact_events_shared_failed",
			"n_groupes", len(groupes), "err", err)
	}
}

// cleDEnsemble : la clé d'un ENSEMBLE de match_id — triés, dédoublonnés.
func cleDEnsemble(ids []string) string {
	tries := slices.Clone(ids)
	slices.Sort(tries)
	return strings.Join(slices.Compact(tries), "\x00")
}
