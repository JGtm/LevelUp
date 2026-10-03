package objectives

import "strings"

// families.go — CE QUI EST UN OBJECTIF, ET CE QUI N'EN EST PAS.
//
// # Le probleme que ce fichier ferme
//
// Les tables de `named.go` ne contiennent pas que des objectifs : `comp 2 A` = `kills` y est
// l'ANCRE D'IDENTITE du balayage (§17.2 de l'etat de l'art) et `comp 3 A` = `assists` son
// voisin de controle croise. Les deux sont nommes, dates, attribues — et publies dans
// `doc.Objectives` comme les autres. Sur `8bc6074f` ils font 119 des 218 actions du calque,
// sur `32d9a94f` 93 des 148 : un denominateur de couverture qui les compte fait lire « 100 %
// de couverture d'objectifs » sur un calque dont la majorite n'est pas un objectif (audit du
// 2026-09-10, §12-1).
//
// # La famille se lit dans le NOM, et c'est un contrat, pas une convention
//
// Toute statistique d'objectif porte sa famille en prefixe (`flag_captures`, `zone_secures`,
// `vip_selected`, `bomb_detonations`), et les familles sont exactement les `ObjectiveType*`
// d'`extract.go`. Les deux statistiques hors objectif, elles, n'ont aucun prefixe. Ce n'est
// pas une coincidence de nommage : le nom canonique vient de `match_objective_stats` et du
// binaire (`CtfStats_FlagGrabs`, `StrongholdsStats_StrongholdCaptures`), ou la famille prefixe
// toujours la statistique. [TestStatsNommeesPortentLeurFamille] le tient sur toutes les tables.
//
// # Pourquoi une liste blanche de FAMILLES et non une liste noire `kills`/`assists`
//
// Une liste noire laisserait passer, sans un mot, la prochaine statistique hors objectif que
// le balayage nommera (`deaths` est deja lu par `slotidentity.go`, hors de ces tables). Une
// famille NOUVELLE, elle, est un ajout delibere au decodeur : elle s'ajoute ici en meme temps
// que sa constante `ObjectiveType*`.

// objectiveFamilies liste les familles d'objectif, dans l'ordre des constantes d'extract.go.
var objectiveFamilies = []string{
	ObjectiveTypeFlag,
	ObjectiveTypeZone,
	ObjectiveTypeHill,
	ObjectiveTypeSkull,
	ObjectiveTypeVip,
	ObjectiveTypeBomb,
}

// IsObjectiveFamilyStat dit si une statistique nommee appartient a une FAMILLE D'OBJECTIF.
//
// Le separateur est exige : `flag_grabs` est de la famille `flag`, un nom qui commencerait par
// les memes lettres sans souligne n'en est pas.
func IsObjectiveFamilyStat(stat string) bool {
	for _, family := range objectiveFamilies {
		if strings.HasPrefix(stat, family+"_") {
			return true
		}
	}
	return false
}

// StatName rend le nom canonique de la statistique.
//
// ELLE EXISTE POUR QUE LES DEUX FORMES D'EVENEMENT DU PAQUET SE COMPTENT PAR LE MEME CODE :
// [IdentifiedEvent] embarque un [NamedEvent], donc les deux la portent, et [CountObjectiveFamily]
// n'a pas besoin d'une copie par forme (regle des deux copies du depot).
func (e NamedEvent) StatName() string { return e.Stat }

// CountObjectiveFamily compte les evenements dont la statistique appartient a une famille
// d'objectif. C'est le denominateur de `coverage.objectives` (cf. replay/objectives.go) : ce que
// le calque DES OBJECTIFS avait a poser, et rien d'autre.
func CountObjectiveFamily[E interface{ StatName() string }](evs []E) int {
	n := 0
	for _, e := range evs {
		if IsObjectiveFamilyStat(e.StatName()) {
			n++
		}
	}
	return n
}
