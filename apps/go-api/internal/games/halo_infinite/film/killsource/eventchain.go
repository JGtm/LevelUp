package killsource

import "levelup/go-api/internal/analysis/filmsource"

// eventchain.go — LA LISTE D EVENEMENTS EN TETE DE PAQUET.
//
// Le paquet de replication commence par une LISTE D EVENEMENTS, puis vient la boucle de records
// ECS que `walk.go` deroule. Ce fichier ne sert QU A la liste d evenements, et il n existe que
// pour une raison : le KILL-EVENT (code 85) porte un champ que le dead-state n a pas —
// L ASSISTANT.
//
// GRAMMAIRE D UN EVENEMENT (desassemblage FUN_14076a1c4 + FUN_14080a9d4, RE_LOG 7ter.25) :
//
//	[1 bit de continuation]   0 => fin de la liste
//	[R(7) code]               code >= 123 => le dispatcher renvoie une erreur
//	3 x { [1 bit de porte] ; si 1 -> champ de presence(cfgIdx[code][i]) }
//	[corps : vtable+0x68 du descripteur du code]
//
// LE PIEGE QUI A COUTE LE CHANTIER, et il est consigne parce qu il est CONTRE-INTUITIF : le bit
// de drapeau du champ de presence N EST PAS lu systematiquement. Dans FUN_1406d3140 le
// court-circuit `(param_3 == 1) && ReadBit(...)` n evalue le bit QUE si `param_3 == 1`, et
// `param_3` est le cfgIdx — une propriete du TYPE d evenement rendue par `vtable+0x58`, qui ne
// lit AUCUN bit — pas l indice de boucle. Le modele << drapeau systematique >> est le cas
// PARTICULIER cfg==1 : d ou 100 % sur le code 1 et 6 % sur le code 7. Grammaire validee a 98.3 %
// sur 3 963 paires consecutives, et **56/56 = 100 % sur le code 85** (RE_LOG 7ter.25 (2)).
//
// CE FICHIER NE DECIDE RIEN. Il rend des positions et des champs ; c est `assist.go` qui
// confronte au kill-feed, et le kill-feed reste le seul juge.

// presRange : colonne `range` de la table de quantification du moteur (9 lignes + terminateur).
// Elle vaut ZERO dans l image PE et n est remplie qu au demarrage : seule une lecture memoire la
// donne (capture `tools/ce/config_qhit_1451f98c0.bin`, RE_LOG 7ter.2).
var presRange = [10]uint32{7679, 7679, 256, 256, 512, 256, 512, 8191, 8191, 0}

// presSpecialRow : la ligne SPECIALE, prise par le court-circuit quand cfgIdx == 1 et que le bit
// de drapeau vaut 1.
const presSpecialRow = 4

// qbitsOf : largeur de quantification d un range (FUN_1406d310c). Zero quand le range est nul —
// et dans ce cas la lecture de la valeur est entierement SAUTEE, pas lue a zero bit.
func qbitsOf(r uint32) int {
	if r == 0 {
		return 0
	}
	m := 31
	for r>>uint(m) == 0 {
		m--
	}
	if r&((1<<uint(m))-1) != 0 {
		return m + 1
	}
	return m
}

// evCfgIdx : le cfgIdx de chacun des 3 emplacements de presence, par code d evenement. Source :
// la table du registre FUN_140e453b4 (123 codes) plus quatre fournisseurs desassembles. La valeur
// -1 marque un fournisseur NON desassemble : l emplacement n est decodable que s il est ABSENT.
// C est une reserve honnete, pas un trou masque — la chaine s arrete au lieu d extrapoler.
var evCfgIdx = map[int][3]int{
	0: {1, 1, 7}, 1: {1, 8, 7}, 2: {1, 8, 7}, 3: {0, 8, 7},
	4: {0, 8, 7}, 5: {5, 8, 7}, 6: {5, 8, 7}, 7: {1, 8, 7},
	8: {2, 3, 7}, 9: {2, 8, 7}, 10: {8, 8, 7}, 11: {2, 8, 7},
	12: {8, 8, 7}, 13: {4, 8, 7}, 14: {-1, -1, -1}, 15: {8, 8, 7},
	16: {8, 8, 7}, 17: {-1, -1, -1}, 18: {6, 8, 7}, 19: {0, 8, 7},
	20: {1, 1, 7}, 21: {4, 8, 7}, 22: {1, 1, 7}, 23: {0, 8, 7},
	24: {0, 2, 7}, 25: {0, 2, 7}, 26: {0, 2, 7}, 27: {0, 0, 7},
	28: {0, 8, 7}, 29: {6, 8, 7}, 30: {4, 8, 7}, 31: {1, 8, 7},
	32: {1, 8, 7}, 33: {3, 4, 7}, 34: {8, 8, 7}, 35: {1, 8, 7},
	36: {1, 8, 7}, 37: {1, 8, 7}, 38: {2, 8, 7}, 39: {2, 8, 7},
	40: {2, 0, 7}, 41: {3, 8, 7}, 42: {2, 8, 7}, 43: {2, 0, 7},
	44: {2, 0, 7}, 45: {2, 8, 7}, 46: {4, 8, 7}, 47: {4, 8, 7},
	48: {4, 8, 7}, 49: {3, 2, 7}, 50: {-1, -1, -1}, 51: {2, 8, 7},
	52: {2, 0, 7}, 53: {1, 1, 7}, 54: {1, 1, 7}, 55: {6, 8, 7},
	56: {5, 1, 7}, 57: {2, 0, 7}, 58: {1, 8, 7}, 59: {0, 8, 7},
	60: {0, 8, 7}, 61: {0, 8, 7}, 62: {6, 8, 7}, 63: {2, 8, 7},
	64: {6, 8, 7}, 65: {3, 8, 7}, 66: {6, 8, 7}, 67: {6, 8, 7},
	68: {6, 8, 7}, 69: {0, 8, 7}, 70: {2, 8, 7}, 71: {2, 8, 7},
	72: {2, 8, 7}, 73: {2, 8, 7}, 74: {2, 8, 7}, 75: {1, 8, 7},
	76: {1, 8, 7}, 77: {6, 8, 7}, 78: {4, 8, 7}, 79: {-1, -1, -1},
	80: {8, 8, 7}, 81: {0, 8, 7}, 82: {0, 8, 7}, 83: {0, 8, 7},
	84: {0, 8, 7}, 85: {8, 8, 7}, 86: {1, 8, 7}, 87: {6, 8, 7},
	88: {6, 8, 7}, 89: {6, 8, 7}, 90: {6, 8, 7}, 91: {6, 8, 7},
	92: {6, 8, 7}, 93: {2, 0, 7}, 94: {6, 8, 7}, 95: {6, 8, 7},
	96: {6, 8, 7}, 97: {6, 8, 7}, 98: {1, 1, 7}, 99: {8, 8, 7},
	100: {1, 8, 7}, 101: {6, 8, 7}, 102: {2, 8, 7}, 103: {0, 0, 7},
	104: {0, -1, -1}, 105: {0, 0, 7}, 106: {0, -1, -1}, 107: {0, 8, 7},
	108: {6, 8, 7}, 109: {0, 0, 7}, 110: {1, 8, 7}, 111: {6, 8, 7},
	112: {6, 8, 7}, 113: {6, 8, 7}, 114: {6, 8, 7}, 115: {-1, -1, -1},
	116: {-1, -1, -1}, 117: {-1, -1, -1}, 118: {1, 8, 7}, 119: {2, 0, 7},
	120: {8, 8, 7}, 121: {8, 8, 7}, 122: {8, 8, 7},
}

// evStub : les 13 codes dont le deserialiseur est un `return 1` — corps de ZERO bit.
var evStub = map[int]bool{3: true, 4: true, 23: true, 24: true, 25: true, 26: true, 33: true,
	49: true, 54: true, 57: true, 59: true, 92: true, 103: true}

// evFixed : longueurs de CORPS mesurees contre l oracle du dispatcher sous la grammaire corrigee.
// Les codes a un seul echantillon (12, 21, 34, 40) sont plausibles mais NON confirmes
// statistiquement — c est ecrit ici plutot que dissimule.
var evFixed = map[int]int{5: 111, 6: 93, 7: 118, 9: 36, 12: 94, 21: 2, 34: 59, 38: 10, 40: 78,
	75: 54, 76: 104}

// curseurEv : LE CURSEUR MEFIANT DE LA CHAINE D EVENEMENTS.
//
// Il ne lit AUCUN octet lui-meme (lot 2.4.1, ADR 0034 D-2) : il porte le lecteur de bits
// canonique de la couche source ([filmsource.Bits]) et lui ajoute la seule chose que le moteur
// n a pas — LE REFUS DE LIRE AU-DELA DU PAQUET. Le drapeau est indispensable : le lecteur du
// moteur bourre a zero, et sans ce refus une chaine desynchronisee << lit >> des evenements
// parfaitement valides apres la fin du paquet.
//
// LE DRAPEAU EST ICI, ET PAS DANS LE LECTEUR (arbitrage V15 (3)). La mefiance appartient au
// MARCHEUR de chaine, pas au moteur : le lecteur canonique garde la semantique du jeu
// (bourrage a zero), et le refus se teste par [filmsource.Bits.Remaining] AVANT chaque lecture.
// Aucune valeur lue ne change — `bp+n > len(pl)*8` et `Remaining() < n` sont la meme condition,
// et l equivalence bit a bit est prouvee par `equivalence_lecteur_test.go`.
//
// Jusqu au lot 2.4.1 ce type s appelait `evReader` et portait son propre `pl []byte` / `bp int`
// avec sa propre boucle bit a bit (`bitsWide`) : c etait le DEUXIEME des sept lecteurs de bits
// du depot.
type curseurEv struct {
	b    *filmsource.Bits
	over bool
}

// nouveauCurseurEv ouvre un curseur sur `pl`, positionne au bit `bp`.
func nouveauCurseurEv(pl []byte, bp int) *curseurEv {
	b := filmsource.NewBits(pl)
	b.SetBitPos(bp)
	return &curseurEv{b: b}
}

// pos : la position de lecture, en bits.
func (r *curseurEv) pos() int { return r.b.BitPos() }

// aller : repositionne le curseur a une position ABSOLUE deja calculee (la fin des champs d un
// kill-event, par exemple). Ne leve pas le drapeau : la position vient d une lecture qui a
// deja fait ses bornes.
func (r *curseurEv) aller(p int) { r.b.SetBitPos(p) }

// octets : les octets du paquet parcouru.
func (r *curseurEv) octets() []byte { return r.b.Octets() }

// epuise : plus aucun bit a lire.
func (r *curseurEv) epuise() bool { return r.b.Remaining() <= 0 }

func (r *curseurEv) rd(n int) uint64 {
	if n <= 0 {
		return 0
	}
	if r.b.Remaining() < n {
		r.over = true
		return 0
	}
	return r.b.ReadBits(uint(n))
}

func (r *curseurEv) g1() int { return int(r.rd(1)) }

// skip : avance de `n` bits sans construire de valeur. Necessaire au-dela de 64 bits, ou `rd`
// n aurait plus de sens.
func (r *curseurEv) skip(n int) {
	if n <= 0 {
		return
	}
	if r.b.Remaining() < n {
		r.over = true
		return
	}
	r.b.Skip(n)
}

// estAncreDeKillEvent : la position `x` ouvre-t-elle un kill-event ? Bit de continuation a 1
// juste avant, code 85 sur les sept bits suivants. C est le GENERATEUR de candidats de
// [killEventsIn] — la seule lecture que le balayage fait a CHAQUE bit du paquet, donc la seule
// qui doive rester sans allocation : elle passe par les primitives de position de la couche
// source, pas par un curseur construit par position essayee.
func estAncreDeKillEvent(pl []byte, x int) bool {
	return filmsource.BitAt(pl, x-1) == 1 &&
		int(filmsource.BitsAt(pl, x, 7)) == killEventCode
}

// evPresence : la boucle de presence, 3 emplacements FIXES. Rend faux quand un emplacement
// PRESENT porte un cfgIdx non resolu : la longueur est alors inconnue, et une longueur inconnue
// arrete la chaine.
func evPresence(r *curseurEv, code int) bool {
	cfg := evCfgIdx[code]
	for i := 0; i < 3; i++ {
		if r.g1() == 0 {
			continue
		}
		c := cfg[i]
		if c < 0 || c >= len(presRange) {
			return false
		}
		if c == 1 && r.g1() == 1 {
			c = presSpecialRow
		}
		if w := qbitsOf(presRange[c]); w > 0 {
			r.rd(w)
		}
		r.rd(2) // tag, toujours lu
	}
	return !r.over
}

// killEventFields : les champs du kill-event (code 85), struct 0x28 de FUN_14104bd08.
//
// LA QUEUE N EST PAS LUE ICI, ET C EST UNE MESURE, PAS UN CHOIX : le champ arme/categorie existe
// dans l executable mais est ABSENT des films (0/133 au vrai curseur, sur les cinq films testes,
// RE_LOG 7ter.25 (4)). C est precisement pourquoi la source du degat se lit dans le dead-state.
//
// LES DEUX BLOCS R32 SONT DES PARTS DE DEGATS EN POURCENTAGE ENTIER — le premier pour le TUEUR,
// le second pour l ASSISTANT. QUATRE JAMBES CONVERGENTES, chacune mesuree :
//
//	(1) ADJACENCE ET ORDRE dans le modele de recap du jeu : `KillerPercentageDamageDone` a
//	    +0x228 est immediatement suivi de `AssistantPercentageDamageDone` a +0x22c — le meme
//	    couple, dans le meme ordre, que les deux blocs de la grammaire.
//	(2) TYPE ENTIER confirme au desassemblage : tag 1 pour ces deux champs, contre 2 pour les
//	    flottants voisins du meme modele.
//	(3) SOMME == 99 sur 22 367 des 31 204 kills ASSISTES de 892 films. La deuxieme valeur la plus
//	    frequente est 89 (1.78 %) et RIEN ne tombe a 100 : la signature du double arrondi vers le
//	    bas de deux parts complementaires.
//	(4) COLLISION avec une capture Cheat Engine du 2026-06-10 sur la constante ARBITRAIRE 149 :
//	    presente sur 3 films de 886, absente de 983 000 lectures a position aleatoire.
//
// RESERVE, ET ELLE SE PORTE PARTOUT OU CES DEUX CHAMPS SONT CITES : le CHEMIN DE DONNEES entre le
// kill-event du film et ces deux champs du modele de recap N EST PAS DEMONTRE — l ecrivain de
// +0x228 est un setter GENERIQUE partage par neuf modeles d UI. Quatre jambes convergentes, PAS
// une chaine d appels prouvee.
type killEventFields struct {
	killer, victim, assist int
	// killerPct : part de degats du TUEUR, pourcentage ENTIER. AUCUN PLAFOND A 100 : des valeurs
	// jusqu a 228 sont mesurees sur des kill-events attaches a de vraies morts nommees (1.7 % des
	// attaches). L hypothese naturelle est le degat EXCEDENTAIRE — une roquette qui retire plus
	// que la vie restante — et elle n est PAS etablie ; ce qui l est, c est qu une valeur > 100
	// est une donnee reelle et non une lecture ratee.
	killerPct uint32
	// assistPct : part de degats de l ASSISTANT. NE VEUT RIEN DIRE quand `assist` vaut -1 : le
	// bloc porte alors une CONSTANTE PAR FILM (149, 70, 20, 197 selon le film). Voir
	// [decodeCtx.fillAssist], qui est le seul endroit ou cette regle se decide.
	assistPct uint32
	flag      int // le R1 entre `killerPct` et `assist` — semantique NON etablie
	end       int // fin des champs obligatoires, -1 en cas de debordement
}

// readEntityRef5 : ECS_ReadEntityRefIndex5. Porte a 1 => ABSENT (-1) ; porte a 0 => 5 bits
// d indice local.
func readEntityRef5(r *curseurEv) int {
	if r.g1() != 0 {
		return -1
	}
	return int(r.rd(5))
}

// readKillEvent : decode le kill-event depuis le premier bit de son CORPS.
//
// L ORDRE DES DEUX PREMIERS CHAMPS EST **VICTIME PUIS TUEUR**, ET C EST UNE CORRECTION MESUREE.
// L outillage de retro-ingenierie portait les DEUX lectures, contradictoires, dans deux fichiers
// differents (`chainwalk.go` lisait tueur d abord, `pipeline_decode.go` victime d abord) et
// personne n avait tranche. Confrontation au kill-feed sous la bijection deja resolue, sur les
// quatre films de reference — c est sans appel :
//
//	film        ordre (tueur, victime)   ordre (victime, tueur)
//	9b191a7f          6 / 101                  79 / 101
//	000d5950         10 / 100                  83 / 100
//	fccc61cd         14 / 105                  81 / 105
//	78919882         10 / 106                  89 / 106
//
// CONSEQUENCE A PORTER PLUS LOIN QUE CE FICHIER : la mesure d assistants de RE_LOG 7ter.24ter
// rejetait les cas `assistant == champ0` en les nommant << assistant == tueur >>. Le champ 0
// etant la VICTIME, ce rejet portait en realite sur `assistant == victime`.
//
// LA GRAMMAIRE COMPLETE, telle qu elle est lue ici :
//
//	victime(E5)  tueur(E5)  [% TUEUR]  R1  assistant(E5)  [% ASSISTANT]
func readKillEvent(pl []byte, body int) killEventFields {
	r := nouveauCurseurEv(pl, body)
	var k killEventFields
	k.victim = readEntityRef5(r)
	k.killer = readEntityRef5(r)
	k.killerPct = uint32(r.rd(32))
	k.flag = r.g1()
	k.assist = readEntityRef5(r)
	k.assistPct = uint32(r.rd(32))
	k.end = r.pos()
	if r.over {
		k.end = -1
	}
	return k
}

// killEventPlausible : la grammaire MINIMALE d un kill-event — tueur et victime presents et
// distincts. Aucune heuristique de contenu : c est la chaine qui valide, pas le contenu.
func killEventPlausible(k killEventFields) bool {
	return k.end > 0 && k.killer >= 0 && k.victim >= 0 && k.killer != k.victim
}

// evStep : decode UN evenement complet a partir de son bit de continuation.
// `fin` vaut vrai quand le bit de continuation est a 0 — fin NORMALE de la liste, pas une erreur.
func evStep(r *curseurEv, gate15 bool) (fin, ok bool) {
	if r.epuise() {
		return false, false
	}
	if r.g1() == 0 {
		return true, true
	}
	code := int(r.rd(7))
	if r.over || code >= 123 {
		return false, false
	}
	if !evPresence(r, code) {
		return false, false
	}
	if !evBody(r, code, gate15) {
		return false, false
	}
	return false, !r.over
}

// evChainLen : nombre d evenements enchaines depuis `p`, borne a `maxEv`. C EST LE CRITERE DE
// LOCALISATION : une coincidence ne s enchaine pas. Seuil retenu : 3 evenements — 132/133 vrais
// gardes (rappel 99.2 %), 11/1739 faux gardes, dont 10 sont des kill-events REELS du meme paquet
// (multi-kill atteste) et 1 porte un indice de victime hors roster (RE_LOG 7ter.25 (3)).
func evChainLen(pl []byte, p int, gate15 bool, maxEv int) int {
	r := nouveauCurseurEv(pl, p)
	n := 0
	for n < maxEv {
		fin, ok := evStep(r, gate15)
		if fin || !ok {
			return n
		}
		n++
	}
	return n
}
