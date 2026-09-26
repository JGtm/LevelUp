package replay

// flag_carries_handoff.go — LE PASSAGE DE MAIN EN MAIN, et l'identite du drapeau PAR L'EQUIPE.
//
// # LA REGLE, EN UNE PHRASE
//
// Un drapeau n'a qu'UN porteur a la fois : toute PRISE d'un AUTRE joueur, portant sur le MEME
// drapeau et datee a l'interieur d'un portage, en est une borne haute. Le porteur avait lache
// avant, et le calque le dessinait encore dans sa main.
//
// # SAVOIR DE QUEL DRAPEAU IL S'AGIT — SANS GEOMETRIE
//
// Le bornage precede l'attribution geometrique (`flag_assign.go` a besoin des positions, qui
// dependent de `t1`). On ne peut donc PAS lire `flagIndex` ici. Mais on n'en a pas besoin : la
// REGLE DU MODE nomme le drapeau a elle seule, et c'est l'invariant dur deja pose par
// `flag_assign.go` — EN CTF ON NE PORTE JAMAIS SON PROPRE DRAPEAU, on le RENVOIE.
//
//	un joueur de l'equipe T qui prend un drapeau prend celui qui n'appartient PAS a T ;
//	deux joueurs de la MEME equipe prennent donc le MEME drapeau ;
//	deux joueurs d'equipes OPPOSEES prennent deux drapeaux DIFFERENTS.
//
// D'ou : le passage de main en main n'existe qu'entre COEQUIPIERS. Un adversaire qui touche ce
// drapeau-la le RENVOIE (`flag_returns`), il ne le prend pas — sauf en variante DRAPEAU NEUTRE,
// ou il n'y a qu'un seul drapeau et ou tout le monde le prend. Les deux cas sont couverts par
// [flagOfCarrier] : un seul drapeau en jeu, ou le drapeau de l'equipe adverse.
//
// # CE QUE LA MESURE EN DIT, ET POURQUOI LA REGLE RESTE
//
// Sur les 11 films CTF a calque du parc de la vague 6, cette population est PETITE : le lacher
// suivi d'une reprise par un coequipier est deja date, dans l'immense majorite des cas, par la
// VIE LIBRE de l'objet (`closeByFreeLives`, lot 6.7-B1) — l'objet touche le sol entre les deux
// mains. La regle ferme ce que ce canal ne voit pas (objet dont la piste manque), et elle rend
// le calque VRAI par construction plutot que par constat : sans elle, un film ou le drapeau
// passe de main en main sans jamais repliquer sa position publierait deux porteurs simultanes.
//
// # ELLE NE PEUT QUE RACCOURCIR
//
// Comme toutes les chaines de fermeture du calque, elle ne retient qu'un instant STRICTEMENT
// interieur a `]t0, t1[` : tout instant rendu est deja plus petit que la borne en place. Le biais
// du calque garde donc son sens — trop long, jamais trop court.

// flagSingleInPlay dit que le film ne met en jeu qu'UN SEUL drapeau : la variante DRAPEAU NEUTRE
// (un socle retenu, cf. flag_neutral.go), ou une carte hors du catalogue d'objectifs (aucun
// socle, tous les portages tombent dans un drapeau unique).
func flagSingleInPlay(spawns []FlagSpawn) bool { return len(spawns) <= 1 }

// flagOfCarrier rend l'index du drapeau qu'un joueur PORTE quand il en prend un, et si ce
// drapeau est NOMME.
//
// Un seul drapeau en jeu : c'est celui-la, quelle que soit l'equipe — personne ne le possede.
// Sinon, c'est l'unique socle dont l'equipe du joueur n'est PAS proprietaire. Equipe inconnue
// (table `TeamOf` vide : CLI hors ligne, ouvrier sans faits) ou plusieurs socles adverses
// (carte a plus de deux drapeaux) : rien n'est nomme, et l'abstention SE COMPTE — sans quoi un
// film que la regle traverse en silence serait indistinguable d'un film sans passage.
func flagOfCarrier(scan FlagCarryScan, xuid string) (int, bool) {
	if flagSingleInPlay(scan.Spawns) {
		return 0, true
	}
	team, known := scan.TeamOf[xuid]
	if !known || team == TeamNeutral {
		return -1, false
	}
	found := -1
	for i := range scan.Spawns {
		if scan.Spawns[i].Team == team {
			continue
		}
		if found >= 0 {
			return -1, false // plus d'un drapeau adverse : rien ne les departage.
		}
		found = i
	}
	return found, found >= 0
}

// closeByHandoff ferme chaque portage a la PREMIERE prise d'un AUTRE joueur portant sur le MEME
// drapeau. Rend le nombre de portages ainsi fermes, et celui des portages dont l'equipe ne nomme
// aucun drapeau — ceux que la regle n'a pas pu juger.
func closeByHandoff(raws []flagCarryRaw, ops []flagOpening, scan FlagCarryScan) (unnamed int) {
	parPortage := flagIndexByTeam(raws, scan)
	// `ops` et `raws` portent les MEMES joueurs, un portage par prise : la table par xuid se
	// deduit donc de celle par portage, sans second balayage des socles.
	drapeaux := make(map[string]int, len(raws))
	for i := range raws {
		drapeaux[raws[i].xuid] = parPortage[i]
	}
	for i := range raws {
		mien := parPortage[i]
		if mien < 0 {
			unnamed++
			continue
		}
		at, found := flagFirstOtherOpening(raws[i], ops, drapeaux, mien)
		if !found {
			continue
		}
		flagCloseAt(&raws[i], at, flagCloserHandoff)
	}
	return unnamed
}

// flagFirstOtherOpening rend l'instant de la PREMIERE prise d'un AUTRE joueur du drapeau `mien`,
// STRICTEMENT a l'interieur du portage. Strictement : une prise qui tombe sur l'une des deux
// bornes ne raccourcit rien, et la regle ne doit jamais allonger.
func flagFirstOtherOpening(r flagCarryRaw, ops []flagOpening, drapeaux map[string]int,
	mien int) (int64, bool) {
	best, found := int64(0), false
	for _, o := range ops {
		if o.xuid == r.xuid || o.t0 <= r.t0 || o.t0 >= r.t1 || drapeaux[o.xuid] != mien {
			continue
		}
		if !found || o.t0 < best {
			best, found = o.t0, true
		}
	}
	return best, found
}

// flagIndexByTeam rend, par portage, l'index du drapeau que son porteur tient — -1 quand
// l'equipe ne le nomme pas. La table est calculee UNE fois : la resoudre dans la boucle
// refarait le meme balayage de socles a chaque portage.
func flagIndexByTeam(raws []flagCarryRaw, scan FlagCarryScan) []int {
	out := make([]int, len(raws))
	cache := make(map[string]int, len(raws))
	for i := range raws {
		f, vu := cache[raws[i].xuid]
		if !vu {
			var ok bool
			if f, ok = flagOfCarrier(scan, raws[i].xuid); !ok {
				f = -1
			}
			cache[raws[i].xuid] = f
		}
		out[i] = f
	}
	return out
}
