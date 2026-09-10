package sessionusage

// usage_overview.go — LE BLOC « SERVI OU GÂCHÉ » AU GRAIN PÉRIODE (étapes E5 et
// E6.1 du PLAN_EQUIPEMENT_GACHIS_2026-09-09) : ce que les pages Synthèse et
// Escouade publient d'un scope de matchs qui n'est PAS une session.
//
// # CE QUE CE FICHIER AJOUTE, ET CE QU'IL NE REFAIT PAS
//
// Rien du remplissage de barre n'est recalculé ici : les trois issues et les deux
// taux de référence viennent de [computeOutcomes] (usage_outcomes.go), la même
// fonction que la page Sessions, et la liste des familles du bilan vient de
// [equipmentBilanFamilies] — donc de `replay.EquipmentOutcomeFamilies`, source
// unique qui garantit que le répulseur n'a pas de ligne (décision P4).
//
// Ce fichier ne pose que ce que le grain PÉRIODE ajoute :
//   - une ligne par joueur suivi, toutes familles confondues (étape E6.1) ;
//   - les cinq comptes exclusifs de chacun des deux donuts (décisions P10/P11).
//
// # AUCUN POURCENTAGE DE PART N'EST CALCULÉ ICI (décisions P9/P10)
//
// L'axe de Solo et Escouade est en COMPTES d'objets. Le Go publie des comptes qui
// ferment (les quatre parts font le lobby) et le front en fait des arcs et deux
// sous-totaux. Les seuls pourcentages du bloc sont les trois taux d'ISSUE, qui
// répondent à « est-ce que je gâche », pas à « quelle est ma part ».
//
// # DEUX SCOPES, ET C'EST VOULU
//
// Les lignes (familles, joueurs) portent sur TOUT le scope mesuré — comme
// PlayerTotal côté session. Les deux donuts portent sur les seuls matchs à camp
// CONNU, numérateurs ET dénominateurs : c'est la règle de scope de
// [computeMetric], et c'est la seule façon d'avoir des parts qui ferment. Mêler
// les deux ferait un donut dont la somme des parts ne vaudrait pas son centre.

import (
	"sort"

	"levelup/go-api/internal/domain"
)

// OverviewInput — un scope de matchs (ordre indifférent : le bloc n'a pas de
// point par match), le joueur de la route, et ses coéquipiers suivis dans l'ordre
// d'affichage (résolus en amont par [ResolveScopeFriends]).
type OverviewInput struct {
	PlayerXUID  string
	FriendXUIDs []string
	Matches     []MatchInput
}

// ComputeUsageOverview projette le scope en bloc contractuel. Scope sans aucun
// match mesuré : bloc Available avec MatchesMeasured=0 et rien d'autre — jamais
// nil, « matchs mesurés 0/N » doit pouvoir s'afficher.
func ComputeUsageOverview(in OverviewInput) domain.EquipmentUsageBlock {
	out := domain.EquipmentUsageBlock{Available: true, MatchesTotal: len(in.Matches)}
	measured := measuredMatches(in.Matches)
	out.MatchesMeasured = len(measured)
	if len(measured) == 0 {
		return out
	}
	out.Families = overviewFamilies(in.PlayerXUID, measured)
	out.Players = overviewPlayers(in.PlayerXUID, in.FriendXUIDs, measured)
	out.EquipmentParties = overviewParties(in, measured, equipmentObjectsOf)
	out.WeaponPadParties = overviewParties(in, measured, padPickupsOf)
	return out
}

// overviewFamilies — une ligne par famille du bilan QUE LE JOUEUR DE LA ROUTE A
// TOUCHÉE, triée du plus pris au moins pris (décision P9 : l'axe est en comptes,
// les lignes se lisent de haut en bas par volume ; clé croissante à volume égal
// pour que l'ordre soit un contrat stable).
//
// LE CRITÈRE D'ENTRÉE EST CELUI DU SUJET, pas celui du lobby : [subjectBilanFamilies]
// (usage_outcomes.go), LA MÊME fonction que lit désormais [metricKeys] pour la page
// Sessions (lot 6.4 point 4 — les deux étaient divergentes avant ce lot). La
// question posée par cette page est « est-ce que JE gâche mon équipement » : une
// famille que je n'ai jamais ramassée n'a pas de barre à remplir, et une ligne à
// zéro y serait un reproche sans objet.
func overviewFamilies(playerXUID string, measured []MatchInput) []domain.EquipmentUsageFamilyLine {
	touched := subjectBilanFamilies(playerXUID, measured)
	out := make([]domain.EquipmentUsageFamilyLine, 0, len(touched))
	for _, family := range equipmentBilanFamilies {
		if !touched[family] {
			continue
		}
		out = append(out, domain.EquipmentUsageFamilyLine{
			FamilyKey: family, SessionUsageOutcomes: computeOutcomes(playerXUID, []string{family}, measured),
		})
	}
	sort.SliceStable(out, func(a, b int) bool {
		ta := out[a].Used + out[a].Kept + out[a].Dropped
		tb := out[b].Used + out[b].Kept + out[b].Dropped
		if ta != tb {
			return ta > tb
		}
		return out[a].FamilyKey < out[b].FamilyKey
	})
	return out
}

// overviewPlayers — une ligne par sujet, TOUTES FAMILLES CONFONDUES : le joueur de
// la route en tête, puis les coéquipiers suivis dans l'ordre reçu (étape E6.1).
//
// Les deux taux de référence de CHAQUE ligne excluent SON sujet (décision P7) :
// comparer un coéquipier à une moyenne qui le contient amortirait son signal
// exactement comme celui du joueur de la route.
func overviewPlayers(
	playerXUID string, friendXUIDs []string, measured []MatchInput,
) []domain.EquipmentUsagePlayerLine {
	subjects := make([]string, 0, 1+len(friendXUIDs))
	subjects = append(subjects, playerXUID)
	subjects = append(subjects, friendXUIDs...)
	out := make([]domain.EquipmentUsagePlayerLine, 0, len(subjects))
	for _, xuid := range subjects {
		out = append(out, domain.EquipmentUsagePlayerLine{
			XUID:                 xuid,
			SessionUsageOutcomes: computeOutcomes(xuid, equipmentBilanFamilies, measured),
			PadPickups:           float64(padPickupsTotal(xuid, measured)),
		})
	}
	return out
}

// padPickupsTotal — les prises de socle d'ARME d'un sujet sur tout le scope
// mesuré (grandeur sans issue au grain session : le canal `shots` qui dirait
// « a tiré » n'est pas persisté — cf. domain.EquipmentUsagePlayerLine).
func padPickupsTotal(xuid string, measured []MatchInput) int {
	n := 0
	for i := range measured {
		for j := range measured[i].Players {
			if p := &measured[i].Players[j]; p.XUID == xuid {
				n += p.PadPickups
			}
		}
	}
	return n
}

// overviewValue — la grandeur qu'un donut compte sur une ligne joueur.
type overviewValue func(p *PlayerRow) int

// equipmentObjectsOf — les OBJETS d'équipement d'une ligne, toutes familles du
// bilan confondues (utilisé + gardé + lâché), jamais les prises : c'est la même
// unité que la longueur des barres, sans quoi le donut et le graphe compteraient
// deux choses différentes sous le même mot.
func equipmentObjectsOf(p *PlayerRow) int {
	return outcomeCountsOf(p, equipmentBilanFamilies).total()
}

// padPickupsOf — les prises de socle d'ARME d'une ligne.
func padPickupsOf(p *PlayerRow) int { return p.PadPickups }

// overviewParties — les cinq comptes d'un donut (décisions P10/P11), sur les seuls
// matchs mesurés à camp CONNU. nil quand ce sous-ensemble est vide : sans camp,
// « le reste de mon équipe » et « eux » n'existent pas, et un donut à zéro serait
// une affirmation.
//
// LE CAMP PRIME SUR L'AMITIÉ dans le classement d'une ligne : un ami suivi qui se
// trouve dans le camp d'en face sur un match donné compte du côté « eux » pour ce
// match-là. Sans cette priorité, « eux » cesserait d'être « le lobby moins mon
// camp » et les quatre parts ne feraient plus le tout.
func overviewParties(
	in OverviewInput, measured []MatchInput, value overviewValue,
) *domain.EquipmentUsageParties {
	friends := make(map[string]bool, len(in.FriendXUIDs))
	for _, x := range in.FriendXUIDs {
		friends[x] = true
	}
	var out domain.EquipmentUsageParties
	byFriend := map[string]float64{}
	teamKnown := false
	for i := range measured {
		mi := &measured[i]
		if mi.PlayerTeam == nil {
			continue
		}
		teamKnown = true
		for j := range mi.Players {
			p := &mi.Players[j]
			v := float64(value(p))
			out.LobbyTotal += v
			// Un participant sans camp connu tombe du côté « eux » : on sait qu'il
			// n'est pas dans mon camp, on ne sait rien de plus (même lecture que
			// computeOutcomes).
			teamID, known := mi.TeamOf[p.XUID]
			sameTeam := known && teamID == *mi.PlayerTeam
			switch {
			case p.XUID == in.PlayerXUID:
				out.Player += v
			case sameTeam && friends[p.XUID]:
				out.Friends += v
				byFriend[p.XUID] += v
			case sameTeam:
				out.RestOfTeam += v
			default:
				out.Opponents += v
			}
		}
	}
	if !teamKnown {
		return nil
	}
	for _, x := range in.FriendXUIDs {
		out.ByFriend = append(out.ByFriend, domain.EquipmentUsageFriendCount{XUID: x, Value: byFriend[x]})
	}
	return &out
}
