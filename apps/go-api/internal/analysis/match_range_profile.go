// Package analysis — match_range_profile.go : LA PORTÉE PAR (MATCH, JOUEUR), RAPPORTÉE À LA
// MÉDIANE DU LOBBY (plan .ai/V7.5/PLAN_AJUSTEMENTS_SUPP_PRE_V75_2026-09-21.md, lot N2).
//
// # PUR, COMME TOUT CE PAQUET
//
// L'entrée est une tranche de frags DÉJÀ MESURÉS (le repo a fait la jointure et la
// classification) ; la sortie est le DTO publié. Aucune base, aucun HTTP, aucune mutation de
// l'entrée.
//
// # LES DEUX PIÈGES QUE CE FICHIER EXISTE POUR ÉVITER
//
//	médiane du lobby   elle se calcule sur LES FRAGS du match, jamais comme la moyenne des
//	                   médianes par joueur — qui pèserait autant un joueur à trois frags
//	                   qu'un joueur à trente (cf. WeaponRangeSideTotals, même doctrine).
//	joueur sans frag   il est ABSENT du match, jamais publié à zéro : un zéro se lirait
//	                   « il fragge au corps à corps », ce qui est le contraire de « on n'a
//	                   rien mesuré de lui ».
//
// # AUCUN SEUIL DE PUBLICATION ICI
//
// `WeaponRangeMinMeasured` protège un bâton p10->p90, dont les bornes retombent sur les
// extrêmes sous son effectif. Ici on publie une MÉDIANE et son effectif : le client décide
// de creuser le point sous 5 frags (D22-5) ou de le tracer plein. Poser le seuil ici lui
// retirerait la donnée dont il a besoin pour le dire.
package analysis

import (
	"sort"
	"time"

	"levelup/go-api/internal/domain"
)

// MatchRangeMatch nomme UN match du scope et fixe l'ordre de sortie.
type MatchRangeMatch struct {
	MatchID  string
	PlayedAt time.Time
	MapName  string
}

// MatchRangeInput est l'entrée de [MatchRangeProfiles] — struct plutôt que quatre arguments,
// et surtout parce que les trois champs se lisent ensemble ou pas du tout.
type MatchRangeInput struct {
	// Kills sont les frags mesurés de TOUS les joueurs des matchs du scope. Seul le côté
	// tueur est retenu : un frag lu côté victime est le MÊME frag, et le compter deux fois
	// déplacerait la médiane du lobby.
	Kills []MeasuredKill
	// Matches est le scope, DANS L'ORDRE DE SORTIE (le plus ancien d'abord côté Escouade).
	// Un match sans aucun frag mesuré n'est pas rendu.
	Matches []MatchRangeMatch
	// Publish nomme les joueurs à PUBLIER (xuid -> gamertag, gamertag éventuellement vide).
	// VIDE = aucun joueur publié : les profils ne portent alors que leur référentiel de
	// lobby. C'est un cas légitime (un scope dont aucun joueur suivi n'est nommable), pas
	// une invitation à tout publier — publier le lobby nominatif n'a été demandé nulle part.
	Publish map[string]string
	// PublishOrder fixe l'ordre des joueurs dans chaque profil (le joueur principal en
	// tête, côté Escouade). Un xuid absent de Publish est ignoré ; un xuid de Publish
	// absent d'ici est publié après, par ordre de xuid — jamais silencieusement perdu.
	PublishOrder []string
}

// matchRangeMesures accumule les distances au sol d'un lot de frags.
type matchRangeMesures struct {
	dist []float64
}

func (m *matchRangeMesures) ajouter(k MeasuredKill) {
	m.dist = append(m.dist, k.DistanceM)
}

// mediane trie et rend la médiane des distances du lot.
func (m *matchRangeMesures) mediane() float64 {
	sort.Float64s(m.dist)
	return percentileLinear(m.dist, 50)
}

// MatchRangeProfiles calcule un profil de portée par match du scope.
func MatchRangeProfiles(in MatchRangeInput) []domain.MatchRangeProfile {
	parMatch := make(map[string]*matchRangeMesures, len(in.Matches))
	parJoueur := make(map[string]map[string]*matchRangeMesures, len(in.Matches))
	for _, k := range in.Kills {
		if k.Side != SideKiller || k.MatchID == "" || k.KillerXUID == "" {
			continue
		}
		if _, ok := parMatch[k.MatchID]; !ok {
			parMatch[k.MatchID] = &matchRangeMesures{}
			parJoueur[k.MatchID] = make(map[string]*matchRangeMesures, 8)
		}
		parMatch[k.MatchID].ajouter(k)
		j, ok := parJoueur[k.MatchID][k.KillerXUID]
		if !ok {
			j = &matchRangeMesures{}
			parJoueur[k.MatchID][k.KillerXUID] = j
		}
		j.ajouter(k)
	}

	ordre := matchRangePublishOrder(in)
	out := make([]domain.MatchRangeProfile, 0, len(in.Matches))
	for _, m := range in.Matches {
		lot := parMatch[m.MatchID]
		if lot == nil || len(lot.dist) == 0 {
			// Aucun frag mesuré : pas de référentiel, donc aucun écart n'a de sens.
			continue
		}
		lobby := lot.mediane()
		out = append(out, domain.MatchRangeProfile{
			MatchID:       m.MatchID,
			PlayedAt:      m.PlayedAt,
			MapName:       m.MapName,
			Players:       matchRangePlayers(parJoueur[m.MatchID], ordre, in.Publish, lobby),
			LobbyMedianM:  lobby,
			LobbyMeasured: len(lot.dist),
		})
	}
	return out
}

// matchRangePublishOrder rend les xuid à publier, dans l'ordre demandé puis, pour ceux que
// `PublishOrder` a oubliés, par ordre de xuid — une sortie déterministe vaut mieux qu'un
// parcours de map, qui ferait varier l'ordre des séries d'une requête à l'autre.
func matchRangePublishOrder(in MatchRangeInput) []string {
	ordre := make([]string, 0, len(in.Publish))
	vus := make(map[string]bool, len(in.Publish))
	for _, xuid := range in.PublishOrder {
		if _, ok := in.Publish[xuid]; !ok || vus[xuid] {
			continue
		}
		vus[xuid] = true
		ordre = append(ordre, xuid)
	}
	reste := make([]string, 0, len(in.Publish))
	for xuid := range in.Publish {
		if !vus[xuid] {
			reste = append(reste, xuid)
		}
	}
	sort.Strings(reste)
	return append(ordre, reste...)
}

// matchRangePlayers projette les joueurs publiables d'UN match. Un joueur sans frag mesuré
// sur ce match n'y a pas de ligne.
func matchRangePlayers(
	parJoueur map[string]*matchRangeMesures, ordre []string, publish map[string]string,
	lobbyMedianM float64,
) []domain.MatchRangePlayer {
	out := make([]domain.MatchRangePlayer, 0, len(ordre))
	for _, xuid := range ordre {
		lot := parJoueur[xuid]
		if lot == nil || len(lot.dist) == 0 {
			continue
		}
		med := lot.mediane()
		out = append(out, domain.MatchRangePlayer{
			XUID:        xuid,
			Gamertag:    publish[xuid],
			MedianM:     med,
			LobbyDeltaM: med - lobbyMedianM,
			Measured:    len(lot.dist),
		})
	}
	return out
}
