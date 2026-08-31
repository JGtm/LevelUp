package replay

// flag_neutral.go — LE DRAPEAU NEUTRE : reconnaitre la variante, et publier UN drapeau au lieu
// de deux.
//
// # DEUX MODES, UNE SEULE CARTE
//
// Chaque carte de CTF declare TROIS socles `flag_spawn` : un par equipe, plus un NEUTRE au
// centre. La variante ordinaire n'utilise que les deux premiers (chaque camp defend le sien) ; la
// variante « drapeau neutre » n'utilise QUE le troisieme — un seul drapeau, que les deux camps se
// disputent et rapportent chez l'adversaire.
//
// Jusqu'ici l'appelant ecartait le socle neutre en amont, pour une raison valable : le retenir sur
// une partie ordinaire ferait un troisieme drapeau immobile pour l'eternite. Mais la consequence
// etait qu'une partie a drapeau neutre publiait DEUX drapeaux qui n'existaient pas, et repartissait
// entre eux les portages d'un objet unique.
//
// # LE FILM TRANCHE, ET IL LE FAIT PAR L'OBJET
//
// Le mode n'est PAS dans le film (le constructeur du rejeu est hors ligne et ne connait pas
// `game_variant_name`). Mais l'OBJET drapeau naît ou il renaît, et il ne renaît que CHEZ LUI :
//
//	VARIANTE ORDINAIRE   les naissances se collent aux socles D'EQUIPE, jamais au centre
//	                     (mesure du 2026-08-18 : 41 / 16 / 18 naissances a 0,0 m d'un socle
//	                     d'equipe sur les trois films du corpus)
//	DRAPEAU NEUTRE       elles se collent au socle NEUTRE, qui est le seul point de retour
//
// Compter les deux, et prendre le camp majoritaire, est donc une lecture directe — pas une
// inference sur le nom de la variante.
//
// # LE SEUIL EST ECRIT AVANT LA MESURE, ET IL EST PRUDENT
//
// Il faut [flagNeutralMinBirths] naissances au socle neutre ET strictement plus qu'aux socles
// d'equipe. Le mode ordinaire est le defaut : une naissance egaree au centre ne bascule rien, et
// un film muet (aucune naissance lue) garde le comportement d'avant. On se trompe donc dans le
// sens qui ne casse rien.
//
// # CE QUE LE MODE NEUTRE CHANGE EN AVAL, ET CE QU'IL NE CHANGE PAS
//
// Il ne change QUE le jeu de socles retenu. Tout le reste suit : un seul socle donne un seul
// drapeau, d'equipe [TeamNeutral] ; tous les portages lui reviennent ; son `home` est le centre ;
// et sa RENTREE se date exactement comme celle des autres (naissance de l'objet a son socle).
// LA ZONE DE RETOUR, elle, disparait d'elle-meme : elle appartient au camp PROPRIETAIRE, et un
// drapeau neutre n'en a pas — le client ne trouve aucun defenseur a compter, la jauge se reduit
// a la minuterie. C'est exactement la regle du jeu : un drapeau neutre ne se renvoie pas, il
// revient tout seul.

// flagNeutralMinBirths — le nombre MINIMAL de naissances au socle neutre pour basculer. Trois :
// une partie a drapeau neutre en produit une par reprise du drapeau (chaque retour le remet au
// centre), donc plusieurs par manche ; une naissance isolee est du bruit de mesure.
const flagNeutralMinBirths = 3

// flagSpawnChoice porte le verdict de variante et ce qui le fonde.
type flagSpawnChoice struct {
	// Spawns sont les socles RETENUS : les deux socles d'equipe, ou le seul socle neutre.
	Spawns []FlagSpawn
	// Neutral dit que la partie a ete reconnue « drapeau neutre ».
	Neutral bool
	// NeutralBirths / TeamBirths sont les deux comptes qui fondent le verdict.
	NeutralBirths, TeamBirths int
}

// flagChooseSpawns rend les socles a retenir pour ce film, et le verdict qui l'explique.
//
// APPELANT SANS SOCLE NEUTRE : rien ne change — le compte neutre reste a zero, le verdict est
// faux, et les socles reviennent tels quels. Un appelant plus ancien (qui filtrait deja le
// neutre) continue donc de fonctionner a l'identique.
func flagChooseSpawns(scan FlagCarryScan) flagSpawnChoice {
	out := flagSpawnChoice{}
	var neutres, equipes []FlagSpawn
	for _, s := range scan.Spawns {
		if s.Team == TeamNeutral {
			neutres = append(neutres, s)
			continue
		}
		equipes = append(equipes, s)
	}
	out.NeutralBirths = flagBirthsNear(scan.Free, neutres)
	out.TeamBirths = flagBirthsNear(scan.Free, equipes)
	if len(neutres) > 0 && out.NeutralBirths >= flagNeutralMinBirths && out.NeutralBirths > out.TeamBirths {
		out.Neutral, out.Spawns = true, neutres
		return out
	}
	out.Spawns = equipes
	return out
}

// flagBirthsNear compte les vies libres qui NAISSENT a portee de l'un des socles donnes.
//
// LA DISTANCE N'EST PAS UN SEUIL NEUF : c'est `originDropMaxDist`, celle de la regle du lacher et
// de la rentree (flag_objects.go). Une naissance compte pour UN socle au plus.
func flagBirthsNear(lives []flagFreeLife, spawns []FlagSpawn) int {
	if len(spawns) == 0 {
		return 0
	}
	n := 0
	for _, l := range lives {
		x, y := l.First()
		if _, ok := flagSpawnAt(spawns, x, y); ok {
			n++
		}
	}
	return n
}
