// Package service — session_page_coordination.go : l'attachement du bloc « Coordination »
// à la page détail de session (lot S du plan AJSUP, décisions D22-1 / D22-6).
//
// TROIS SCOPES, UN SEUL PRODUCTEUR (service/coordination_block.go) :
//
//	la SESSION AFFICHÉE    → resp.Coordination ;
//	la SESSION COMPARÉE    → resp.CompareCoordination, miroir exact d'Usage/CompareUsage
//	                         et de RangeProfiles/CompareRangeProfiles — les deux colonnes
//	                         du drawer parlent des mêmes formes, avec leurs propres
//	                         données, et la rangée partagée D16 n'a plus de placeholder ;
//	la PÉRIODE DE RÉFÉRENCE → le repère d'HABITUEL des deux jauges qui n'ont pas de parité
//	                         (« je suis couvert », « on me prépare »).
//
// # LA PÉRIODE DE RÉFÉRENCE EST CELLE DE LA PAGE, PAS UNE SECONDE NOTION
//
// Ce sont les matchs du FILTRE de la page (`filtered` dans GetPage), toutes sessions
// confondues — exactement le scope sur lequel la frise des Séries temporelles pose son
// propre habituel. Inventer ici une fenêtre « 30 derniers jours » aurait donné deux
// habituels au même produit, libres de diverger au premier réglage.
//
// Le corollaire : quand l'utilisateur a resserré le filtre sur UNE session, la référence
// se réduit au scope mesuré. Le repère tomberait alors sur la valeur elle-même — il est
// OMIS (tautologie), et la lecture de référence n'a même pas lieu.
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/legacymatch"
	"levelup/go-api/internal/port"
)

// sessionBlocksScope — LES TROIS SCOPES de la colonne de session, et le contexte
// d'affichage que le bloc d'usage consomme. Une struct plutôt que sept paramètres
// adjacents (seuil CLAUDE.md n°5).
type sessionBlocksScope struct {
	Matches []legacymatch.StatsMatchRow
	// CompareMatches vide = drawer fermé : aucun bloc comparé n'est produit.
	CompareMatches []legacymatch.StatsMatchRow
	// ReferenceMatches : les matchs du filtre de la page (toutes sessions confondues).
	ReferenceMatches []legacymatch.StatsMatchRow
	MatchContext     string
	Locale           string
}

// WithSessionCoordination injecte les deux lecteurs du bloc « Coordination » et les
// capabilities du titre. Le bloc partage son producteur avec la page Series temporelles :
// deux constructeurs auraient donne deux definitions de « morts de mon camp ».
func (s *SessionPageService) WithSessionCoordination(
	tactical port.TacticalRepository, appuis port.CoordinationRepository, caps games.CapabilityMap,
) *SessionPageService {
	s.coordTactical = tactical
	s.coordAppuis = appuis
	s.coordCaps = caps
	return s
}

// attachSessionCoordination attache le bloc de la session affichée, celui de la session
// comparée le cas échéant, puis pose le repère d'habituel sur les deux.
//
// LES EFFECTIFS DE CAMP viennent du bloc d'usage, qui vient de les calculer (réserve R1) :
// bloc d'usage indisponible ⇒ table vide ⇒ le bloc de coordination n'a pas de parité, et
// le dit. Il ne la réinvente jamais.
//
// UNE SEULE LECTURE DU JOURNAL POUR LES TROIS SCOPES (lot L5a du plan perf, 2026-09-23) :
// la session affichée et la session comparée sont lues ENSEMBLE, puis découpées ; la
// référence, si elle sert, ne complète la lecture que des matchs qui lui manquent (cf.
// l'en-tête de coordination_block.go). Aucun match n'est lu deux fois par requête.
func (s *SessionPageService) attachSessionCoordination(
	ctx context.Context, resp *domain.SessionPageResponse, sc sessionBlocksScope,
	teamSize, compareTeamSize map[string]int,
) {
	courant := matchIDsFromStatsRows(sc.Matches)
	compare := matchIDsFromStatsRows(sc.CompareMatches)
	deuxSessions := append(append(make([]string, 0, len(courant)+len(compare)), courant...), compare...)
	lecture := lireCoordination(ctx, s.lecteursDeCoordination(), deuxSessions)
	resp.Coordination = lecture.bloc(ctx, courant, teamSize, nil)
	if len(compare) > 0 {
		resp.CompareCoordination = lecture.bloc(ctx, compare, compareTeamSize, nil)
	}
	s.attachCoordinationHabituel(ctx, resp, sc, lecture)
}

// lecteursDeCoordination assemble les lecteurs du producteur. Un seul point de montage :
// les trois scopes lisent la même chose, par les mêmes lecteurs — et désormais dans la même
// lecture.
func (s *SessionPageService) lecteursDeCoordination() coordinationQuery {
	return coordinationQuery{
		Tactical:   s.coordTactical,
		Appuis:     s.coordAppuis,
		Caps:       s.coordCaps,
		PlayerXUID: s.usageXUID,
	}
}

// attachCoordinationHabituel pose `riposte.habituel_pct` et `appui.habituel_pct` sur les
// blocs servis.
//
// LA LECTURE DE RÉFÉRENCE N'A LIEU QUE SI ELLE SERT : si les deux blocs sont tautologiques
// (ou absents), on rend la main sans ouvrir le journal des morts une seconde fois. Quand elle
// sert, elle COMPLÈTE la lecture des deux sessions : seuls les matchs de la référence qui n'y
// sont pas encore sont lus.
//
// AUCUN EFFECTIF DE CAMP n'est passé à la référence, et c'est voulu : les deux grandeurs
// d'habituel (« je suis couvert », « on me prépare ») ne se rapportent à aucune parité,
// donc le producteur n'a pas besoin de `TeamSize` pour les calculer.
func (s *SessionPageService) attachCoordinationHabituel(
	ctx context.Context, resp *domain.SessionPageResponse, sc sessionBlocksScope,
	lecture *lectureCoordination,
) {
	refIDs := matchIDsFromStatsRows(sc.ReferenceMatches)
	if len(refIDs) == 0 {
		return
	}
	courant := cibleDHabituel(resp.Coordination, refIDs, sc.Matches)
	compare := cibleDHabituel(resp.CompareCoordination, refIDs, sc.CompareMatches)
	if courant == nil && compare == nil {
		return
	}
	lecture.completer(ctx, refIDs)
	ref := lecture.bloc(ctx, refIDs, nil, nil)
	if ref == nil || !ref.Available {
		slog.DebugContext(ctx, "coordination: aucun habituel sur la periode de reference",
			"matchs_reference", len(refIDs))
		return
	}
	couvert := tauxOuRien(ref.Riposte.JeSuisCouvert)
	prepare := tauxOuRien(ref.Appui.OnMePrepare)
	for _, bloc := range []*domain.CoordinationBlock{courant, compare} {
		if bloc == nil {
			continue
		}
		bloc.Riposte.HabituelPct = couvert
		bloc.Appui.HabituelPct = prepare
	}
}

// cibleDHabituel rend le bloc s'il PEUT porter un repère, nil sinon : bloc absent ou
// indisponible (rien à repérer), ou référence tautologique (elle se réduit au scope
// mesuré, et le repère tomberait sur la valeur).
func cibleDHabituel(
	bloc *domain.CoordinationBlock, refIDs []string, scope []legacymatch.StatsMatchRow,
) *domain.CoordinationBlock {
	if bloc == nil || !bloc.Available {
		return nil
	}
	if memeScope(refIDs, matchIDsFromStatsRows(scope)) {
		return nil
	}
	return bloc
}

// memeScope dit si deux listes de matchs désignent le MÊME ensemble. Les deux viennent du
// même filtrage (aucun doublon), la comparaison des tailles et l'appartenance suffisent.
func memeScope(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	index := make(map[string]struct{}, len(a))
	for _, id := range a {
		index[id] = struct{}{}
	}
	for _, id := range b {
		if _, ok := index[id]; !ok {
			return false
		}
	}
	return true
}

// tauxOuRien convertit une couverture en pourcentage de repère. Dénominateur nul = grandeur
// NON MESURÉE sur la référence : pas de repère, jamais un 0 % qui se lirait « habituellement
// jamais couvert ».
func tauxOuRien(c domain.Couverture) *float64 {
	if c.N <= 0 {
		return nil
	}
	pct := c.Taux * 100
	return &pct
}
