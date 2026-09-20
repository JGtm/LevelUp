package killsource

// roster.go — LE ROSTER ETENDU : les humains du kill-feed, PUIS les bots de BOT_METADATA.
//
// Le dead-state identifie un participant par son INDICE ABSOLU sur 5 bits (espace 0..31), pas
// par un gamertag. Il faut donc une bijection indice -> joueur, et un roster a mettre en face.
//
// CE QUI MANQUAIT AVANT N ETAIT PAS UN DECODAGE MAIS UN GATE : l outil de RE figeait
// `indice <= 7`, et l indice 8 — le bot — etait CORRECTEMENT DECODE PUIS JETE A L AFFICHAGE.
// BOT_METADATA donne le slot, donc la borne se DERIVE du film au lieu d etre codee en dur.
//
// REGLE D EPINGLAGE, structurelle et qui ne regarde AUCUN resultat : un bot dont le slot tombe
// AU-DELA de l espace des humains est ajoute au roster et son indice est EPINGLE (il sort de
// l espace de recherche de la bijection) ; un bot dont le slot tombe DANS cet espace contredit
// le modele et n est PAS epingle — il est signale, pas dissimule.
//
// POURQUOI EPINGLER PLUTOT QUE LAISSER LIBRE : un bot n apparait JAMAIS au kill-feed. Laisser
// son indice libre offrirait au solveur une case qui score zero, et lui permettrait d y ranger
// un HUMAIN — ce qui deplacerait toutes les autres attributions.

import (
	"fmt"
	"sort"
	"strings"
)

// Roster : les noms retenus et la bijection indice -> joueur, publies pour que le consommateur
// puisse verifier a qui les indices ont ete attribues.
type Roster struct {
	// Names : les joueurs, humains du kill-feed d abord, puis les bots, puis les sieges que la
	// TABLE DU FILM ajoute (des joueurs que le feed ne nomme pas : ils n ont ni tue ni sont
	// morts), puis le remplissage.
	Names []string
	// Humans : combien des `Names` viennent du kill-feed.
	Humans int
	// Bots : les bots declares par le film, avec leur slot.
	Bots []BotEntry
	// IndexToName : bijection resolue. `IndexToName[i]` est le joueur porte par l indice i.
	IndexToName []string
	// UnpinnedBots : bots dont le slot contredit l espace des humains. Non vide = anomalie a
	// regarder, pas a ignorer.
	UnpinnedBots []BotEntry
	// BotsSuccedes : bots declares sur un slot DEJA tenu par un autre bot — une SUCCESSION dans
	// le meme siege, que le film ecrit et que le roster ne peut pas representer deux fois. Le
	// dernier declare nomme l indice ; les precedents sont comptes ici, jamais laisses en noms
	// libres (cf. [roster.pinBots]).
	BotsSuccedes int
	// IndexSource : la PROVENANCE de chaque indice, meme longueur qu `IndexToName`. Un artefact
	// doit pouvoir dire quelle part de lui vient d une LECTURE et quelle part d un REPLI
	// (doctrine D14 c du chantier, D-10 d ADR 0034).
	IndexSource []IndexOrigin
	// FilmTable : ce que la table des joueurs du film a donne, et ce que le kill-feed en dit.
	FilmTable FilmTablePinning
}

// IndexOrigin nomme d ou vient le joueur porte par un indice. Liste FERMEE.
type IndexOrigin string

const (
	// OriginFilmTable : la table des joueurs de `chunk_00` — LA LECTURE, le lien direct.
	OriginFilmTable IndexOrigin = "table_film"
	// OriginBotMeta : BOT_METADATA a epingle ce slot.
	OriginBotMeta IndexOrigin = "bot_meta"
	// OriginXUIDMotif : les 5 bits qui precedent le motif du xuid dans un chunk de replication
	// (`index_motif.go`) — LA LECTURE qui voit les REMPLACANTS, que la table de `chunk_00`
	// ignore parce qu elle est ecrite a l ouverture du film.
	OriginXUIDMotif IndexOrigin = "motif_xuid"
	// OriginInference : la bijection inferee depuis les votes du kill-feed — LE REPLI.
	OriginInference IndexOrigin = "inference"
	// OriginNone : aucun nom, ni lu ni infere (trou de remplissage).
	OriginNone IndexOrigin = "silence"
)

// FilmTablePinning : ce que la table du film a epingle, et ce que le kill-feed en dit.
//
// LES TROIS DERNIERS CHAMPS SONT UN CONTROLE, JAMAIS UNE DECISION (D14 b) : la table du film
// n est pas corrigee par les votes du kill-feed, elle est CONFRONTEE a eux. Une contradiction se
// compte et se lit ; elle ne se resout pas en silence.
type FilmTablePinning struct {
	// Refusal : la cause nommee quand la table n a pas ete lue. Vide = lue.
	Refusal FilmTableRefusal
	// Build : le build lu en clair (renseigne meme sur un refus pour build inconnu).
	Build string
	// Seats : les sieges OCCUPES et NOMMES que la table rend.
	Seats int
	// Pinned : les indices dont le joueur vient de la table — la part LUE de la bijection.
	Pinned int
	// AddedNames : les noms que la table ajoute au roster parce que le kill-feed ne les porte
	// pas. Mesure du 2026-09-14 : 6 sur 30 films, tous des joueurs qui n ont ni tue ni sont morts.
	AddedNames int
	// BotConflict : sieges refuses parce que BOT_METADATA epingle deja cet indice. Deux lectures
	// du film qui se contredisent : on garde la plus ancienne et la plus eprouvee (le bot), et on
	// COMPTE. Mesure du 2026-09-14 : 0 sur 30 films.
	BotConflict int
	// DuplicateName : sieges refuses parce que le meme nom est deja epingle a un autre indice.
	DuplicateName int
	// OutOfRange : sieges dont l indice sort de l espace des 5 bits (0..31). Impossible par
	// construction de la table de 32 ; compte pour que l impossible se voie s il arrive.
	OutOfRange int
	// Inferred : les indices laisses a l inference — LE REPLI, compte.
	Inferred int
	// FreeNames : les NOMS que l inference a encore a placer sur ces indices. Il n est PAS la
	// meme quantite que `Inferred` et il ne l a plus jamais ete depuis le lot 1.8 : la table du
	// film ajoute au roster les joueurs que le kill-feed ne nomme pas, donc il peut rester plus
	// de noms libres que d indices libres (cf. [roster.freeSlots]). Sans ce compte, « un seul
	// indice a inferer » se lisait a tort « une seule affectation possible » — alors que deux
	// noms pour un indice se tranchent par les votes, et qu un nom sans kill ni mort n en porte
	// aucun : le choix etait ARBITRAIRE (revue de jalon M1, lentille L4).
	FreeNames int
	// MotifPinned / MotifAgree / MotifContradict / MotifDuplicate : LE LIEN PAR LE MOTIF DU XUID
	// (lot 5.2b.1, `index_motif.go`), ventile comme la table l est au-dessus. `MotifPinned` est
	// ce qu il AJOUTE — les indices que ni BOT_METADATA ni la table de `chunk_00` n epinglent,
	// c est-a-dire les REMPLACANTS ; `MotifAgree` / `MotifContradict` sont le CONTROLE sur les
	// indices deja epingles (meme nom / autre nom), et une contradiction ne tranche rien : la
	// table de `chunk_00` garde la main, elle est la plus eprouvee. `MotifDuplicate` : le nom lu
	// est deja epingle ailleurs.
	MotifPinned, MotifAgree, MotifContradict, MotifDuplicate int
	// MotifReadings / MotifDisagreements / MotifAbsent : le COUT de cette lecture. `Readings` =
	// chunks de replication qui ont livre au moins un index ; `Disagreements` = xuids lus a deux
	// index differents (non publies) ; `Absent` = xuids dont le motif ne figure dans aucun chunk.
	MotifReadings, MotifDisagreements, MotifAbsent int
	// Agree / Contradict / Silent : le CONTROLE des indices epingles par les votes du kill-feed.
	// `Agree` = les votes designent le meme joueur ; `Contradict` = ils en designent un autre,
	// strictement plus vote ; `Silent` = aucun vote sur cet indice (le joueur n a ni tue ni est
	// mort dans la fenetre d appariement).
	Agree, Contradict, Silent int
}

// AffectationUnique dit si l inference n avait QU UNE SEULE affectation possible a rendre.
//
// C EST LA QUESTION QUE `Inferred <= 1` CROYAIT POSER, ET QU IL NE POSAIT PAS. Un indice libre
// pour DEUX noms libres se tranche par les votes du kill-feed, et le cout d un nom qui n a ni
// tue ni ete tue vaut zero contre tous les indices : le hongrois rend alors un nom pris au
// hasard du departage ([permLess]), [refine] ne peut rien echanger (il lui faudrait deux indices
// libres) et [bijectionMargin] rend structurellement zero (sa double boucle ne tourne pas sur
// une seule case libre). La porte de publication ligne par ligne reposait donc sur ce seul
// booleen, et publiait la source du degat, le credit, l assistant et les deux parts de degats
// sur un occupant TIRE AU SORT.
//
// LES TROIS REGIMES, ET POURQUOI LE PREMIER RESTE VRAI. Aucun indice libre : la bijection est
// entierement LUE, il n y a rien a choisir — les noms libres qui restent ne portent aucun
// indice, ce qui est exact (cf. [hungarianStart]). Un indice libre pour au plus un nom libre :
// l affectation est forcee. Au-dela : au moins un choix, donc la marge de bijection reprend son
// office.
func (t FilmTablePinning) AffectationUnique() bool {
	switch t.Inferred {
	case 0:
		return true
	case 1:
		return t.FreeNames <= 1
	default:
		return false
	}
}

// BotEntry : un bot tel que le film le declare.
type BotEntry struct {
	Slot  int
	BotID int
	Name  string
}

// roster : l etat interne. `pin` associe un indice absolu a une position de `names`.
type roster struct {
	names    []string
	pin      map[int]int
	nPlay    int // borne du gate des indices, derivee du film
	nHumans  int
	bots     botMeta
	unpinned []bot
	perm     []int
	// seatPin : les indices epingles par la TABLE DU FILM (sous-ensemble de `pin` ; le reste de
	// `pin` vient de BOT_METADATA). Sert a nommer la provenance de chaque indice.
	seatPin map[int]bool
	// botsSuccedes : bots declares sur un slot DEJA epingle par un autre bot — une succession
	// dans le meme siege. Compte et publie ([Roster.BotsSuccedes]) : deux bots sur un slot est un
	// fait du film, pas une anomalie, mais il doit se voir.
	botsSuccedes int
	// motifPin : les indices epingles par le MOTIF DU XUID. Meme role que `seatPin` pour la
	// provenance — les trois ensembles sont disjoints par construction (chacun refuse un indice
	// deja epingle).
	motifPin map[int]bool
	// table : ce que l epinglage par la table a produit, compteurs de controle compris.
	table FilmTablePinning
}

// BotSuffix : marqueur ajoute au nom d un bot. Il doit rester visible : un consommateur ne doit
// jamais confondre un bot avec un joueur, et le kill-feed ne les distingue pas pour lui.
const BotSuffix = " [bot]"

// buildRoster : roster etendu + epinglage des slots de bot + EPINGLAGE DES SIEGES DU FILM +
// borne des indices.
//
// L ORDRE DES TROIS EPINGLAGES EST LE RESULTAT, et il n est pas negociable :
//
//  1. BOT_METADATA d abord — c est la lecture la plus ancienne et la plus eprouvee
//     (RE_LOG 7ter.62), et un bot n apparait JAMAIS au kill-feed : lui laisser un indice libre
//     offrirait au solveur une case qui score zero, ou il rangerait un humain.
//  2. LA TABLE DU FILM ensuite — la LECTURE du lien `index <-> gamertag` (lot 1.5). Elle
//     n ecrase jamais un slot de bot : un indice revendique par les deux est une contradiction
//     entre deux lectures, comptee (`BotConflict`), jamais tranchee en silence.
//  3. LE MOTIF DU XUID ensuite (lot 5.2b.1, `index_motif.go`) — la LECTURE des chunks de
//     replication, la seule qui voie les REMPLACANTS. Elle passe APRES la table parce qu elle
//     est la moins eprouvee des deux et qu elle ne doit rien ecraser : sur un indice deja
//     epingle elle CONTROLE (`MotifAgree` / `MotifContradict`), sur un indice libre elle
//     EPINGLE (`MotifPinned`).
//  4. L INFERENCE en dernier, sur ce qui reste (bijection.go) — le REPLI, compte (`Inferred`).
func buildRoster(kf *killFeed, bm botMeta, useBots bool, t FilmTable, m indexParMotif) *roster {
	r := &roster{
		names:    append([]string(nil), kf.names...),
		pin:      map[int]int{},
		nPlay:    len(kf.names),
		nHumans:  len(kf.names),
		bots:     bm,
		seatPin:  map[int]bool{},
		motifPin: map[int]bool{},
	}
	if useBots {
		r.pinBots(bm)
	}
	r.pinFilmSeats(t)
	r.pinMotifSeats(m)
	// Les indices libres qui ne recoivent aucun nom (trous entre le dernier humain et un slot
	// de bot eleve) recoivent un nom de remplissage : le probleme d affectation doit avoir AU
	// MOINS autant de noms libres que d indices libres, sinon le hongrois n est pas defini.
	for len(r.names) < r.nPlay {
		r.names = append(r.names, fmt.Sprintf("?%d", len(r.names)))
	}
	return r
}

// pinBots : l epinglage des slots de bot (cf. l en-tete du fichier).
//
// # PLUSIEURS BOTS SUR UN MEME SLOT : UNE SUCCESSION, PAS DEUX JOUEURS (lot 5.2b.1, 2026-09-20)
//
// BOT_METADATA declare parfois DEUX OU TROIS bots sur le MEME slot — une succession dans le
// temps, le meme siege repris par un autre bot. `b1ad85eb` en porte trois sur le slot 8
// (`343 Hundy`, `343 PardonMy`, `343 Brew Dog`), et le document de rejeu publie bien trois
// entrees de roster au meme `filmIndex`.
//
// CE QUE LA VERSION D AVANT EN FAISAIT, ET LE DEFAUT QUE CELA CREAIT : chaque bot ajoutait un
// nom a `names` et ECRASAIT `pin[slot]`, donc le DERNIER nommait le slot et les precedents
// restaient dans `names` SANS AUCUN INDICE. Ils devenaient des NOMS LIBRES — c est-a-dire de la
// matiere a inference : le hongrois pouvait poser un nom de bot sur n importe quel indice libre,
// et `FreeNames` les comptait, ce qui rendait `AffectationUnique` faux. Tant que tous les indices
// etaient epingles cela ne coutait rien ; des qu un indice se libere — exactement ce que
// l epinglage des remplacants par le motif du xuid produit — deux noms de bot fantomes
// suffisaient a fermer la publication ligne par ligne de tout le match.
//
// CE QU ELLE FAIT MAINTENANT : le slot garde le MEME vainqueur (le dernier declare, valeur
// inchangee sur tout le parc), mais la succession REMPLACE le nom en place au lieu d en ajouter
// un second. Un slot, un indice, un nom — et zero nom libre fabrique.
func (r *roster) pinBots(bm botMeta) {
	for _, b := range bm.Bots {
		if b.Slot < r.nHumans || b.Slot >= 32 {
			r.unpinned = append(r.unpinned, b)
			continue
		}
		if pos, deja := r.pin[b.Slot]; deja {
			r.names[pos] = b.Name + BotSuffix
			r.botsSuccedes++
			continue
		}
		r.names = append(r.names, b.Name+BotSuffix)
		r.pin[b.Slot] = len(r.names) - 1
		if b.Slot+1 > r.nPlay {
			r.nPlay = b.Slot + 1
		}
	}
}

// pinFilmSeats : LA TABLE DU FILM EPINGLE SES SIEGES, dans l ordre CROISSANT des indices.
//
// L ordre est fixe parce que le resultat en depend : un nom present deux fois dans la table ne
// peut epingler qu un seul indice, et « lequel » doit etre reproductible d une execution a
// l autre. Un siege dont le nom manque au kill-feed est AJOUTE au roster — c est le gain le plus
// direct du lot 1.8 : le feed ne nomme que les joueurs qui tuent ou qui meurent.
func (r *roster) pinFilmSeats(t FilmTable) {
	r.table = FilmTablePinning{Refusal: t.Refusal, Build: t.Build, Seats: len(t.Seats)}
	if !t.Lue() {
		return
	}
	posDuNom := make(map[string]int, len(r.names))
	for i := 0; i < r.nHumans && i < len(r.names); i++ {
		if _, deja := posDuNom[r.names[i]]; !deja {
			posDuNom[r.names[i]] = i
		}
	}
	prises := make(map[int]bool, len(r.pin))
	for _, p := range r.pin {
		prises[p] = true
	}
	indices := make([]int, 0, len(t.Seats))
	for idx := range t.Seats {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	for _, idx := range indices {
		r.pinUnSiege(idx, t.Seats[idx], posDuNom, prises)
	}
}

// pinUnSiege : un siege, et les quatre raisons de le refuser — chacune comptee.
func (r *roster) pinUnSiege(idx int, nom string, posDuNom map[string]int, prises map[int]bool) {
	if idx < 0 || idx >= 32 {
		r.table.OutOfRange++
		return
	}
	if _, deja := r.pin[idx]; deja {
		r.table.BotConflict++ // seul BOT_METADATA a pu epingler avant cet appel
		return
	}
	pos, connu := posDuNom[nom]
	if connu && prises[pos] {
		r.table.DuplicateName++
		return
	}
	if !connu {
		r.names = append(r.names, nom)
		pos = len(r.names) - 1
		posDuNom[nom] = pos
		r.table.AddedNames++
	}
	r.pin[idx] = pos
	prises[pos] = true
	r.seatPin[idx] = true
	r.table.Pinned++
	if idx+1 > r.nPlay {
		r.nPlay = idx + 1
	}
}

// pinMotifSeats : LE MOTIF DU XUID EPINGLE CE QUE LA TABLE N A PAS VU — les remplacants.
//
// Il tourne MEME quand la table de `chunk_00` est refusee (film sans registre, build inconnu) :
// c est alors la seule lecture directe qui reste, et la bijection inferee n y perd rien puisque
// l inference ne travaille plus que sur les indices encore libres.
//
// LES QUATRE ISSUES SONT COMPTEES, et aucune n arbitre en silence : accord (le nom lu est celui
// deja epingle), contradiction (un AUTRE nom — l epinglage en place gagne, il vient de la
// lecture la plus eprouvee), doublon (le nom est deja epingle a un autre indice), epinglage.
func (r *roster) pinMotifSeats(m indexParMotif) {
	r.table.MotifReadings, r.table.MotifDisagreements = m.lectures, m.desaccords
	r.table.MotifAbsent = m.absents
	if len(m.nomParIndex) == 0 {
		return
	}
	posDuNom := make(map[string]int, len(r.names))
	for i := len(r.names) - 1; i >= 0; i-- {
		posDuNom[r.names[i]] = i
	}
	prises := make(map[int]bool, len(r.pin))
	for _, p := range r.pin {
		prises[p] = true
	}
	indices := make([]int, 0, len(m.nomParIndex))
	for idx := range m.nomParIndex {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	for _, idx := range indices {
		r.pinUnMotif(idx, m.nomParIndex[idx], posDuNom, prises)
	}
}

// pinUnMotif : un indice lu au motif du xuid. Decoupe de [roster.pinMotifSeats] pour rester sous
// le plafond de longueur du depot.
func (r *roster) pinUnMotif(idx int, nom string, posDuNom map[string]int, prises map[int]bool) {
	if pos, deja := r.pin[idx]; deja {
		if pos >= 0 && pos < len(r.names) && r.names[pos] == nom {
			r.table.MotifAgree++
			return
		}
		r.table.MotifContradict++
		return
	}
	pos, connu := posDuNom[nom]
	if connu && prises[pos] {
		r.table.MotifDuplicate++
		return
	}
	if !connu {
		r.names = append(r.names, nom)
		pos = len(r.names) - 1
		posDuNom[nom] = pos
	}
	r.pin[idx] = pos
	prises[pos] = true
	r.motifPin[idx] = true
	r.table.MotifPinned++
	if idx+1 > r.nPlay {
		r.nPlay = idx + 1
	}
}

// originOf : la provenance du joueur porte par un indice.
func (r *roster) originOf(i int) IndexOrigin {
	switch {
	case r.seatPin[i]:
		return OriginFilmTable
	case r.motifPin[i]:
		return OriginXUIDMotif
	case r.isBotIndex(i):
		return OriginBotMeta
	case r.nameOf(i) == "?" || strings.HasPrefix(r.nameOf(i), "?"):
		return OriginNone
	default:
		return OriginInference
	}
}

// freeSlots : indices NON epingles, et positions de noms NON epinglees.
//
// LES DEUX LISTES N ONT PAS LA MEME LONGUEUR, ET C EST LE LOT 1.8 QUI LES A DESACCORDEES. La
// table du film AJOUTE au roster les joueurs que le kill-feed ne nomme pas ([roster.pinUnSiege],
// `AddedNames`) : quand le siege ajoute porte un indice DEJA dans l espace des indices, `names`
// grandit sans que `nPlay` bouge. L ecart vaut exactement `len(names) - nPlay`, et il est
// POSITIF sur le parc (`111fa685` : 25 noms pour 24 indices ; cf. [hungarianStart], qui PADDE la
// matrice pour cette raison). Le godoc d avant affirmait l egalite « par construction » — c est
// cette affirmation qui avait fait passer [FilmTablePinning.Inferred] pour une mesure d unicite
// (revue de jalon M1, lentille L4).
func (r *roster) freeSlots() (free, freeNames []int) {
	used := map[int]bool{}
	for _, p := range r.pin {
		used[p] = true
	}
	for i := 0; i < r.nPlay; i++ {
		if _, ok := r.pin[i]; !ok {
			free = append(free, i)
		}
	}
	for p := range r.names {
		if !used[p] {
			freeNames = append(freeNames, p)
		}
	}
	return free, freeNames
}

// nameOf : nom porte par un indice absolu sous la bijection resolue, ou "?" hors roster.
func (r *roster) nameOf(i int) string {
	if i < 0 || i >= len(r.perm) || r.perm[i] < 0 || r.perm[i] >= len(r.names) {
		return "?"
	}
	return r.names[r.perm[i]]
}

// nomEpingle : le nom que la LECTURE donne a un indice — la table des joueurs du film (lots 1.5
// et 1.8) ou BOT_METADATA. Il NE DEPEND D AUCUNE BIJECTION, et c est ce qui le rend utilisable
// pour decider un couple que la bijection consommera ensuite (lot 1.9.3, `feed_couples.go`) :
// [nameOf], lui, lit `perm`, donc le resultat de l inference.
//
// Faux = cet indice n est pas epingle. Ce n est pas une erreur : c est le diagnostic « la lecture
// se tait ici », le seul qui ouvre un repli (D14 b).
func (r *roster) nomEpingle(i int) (string, bool) {
	pos, ok := r.pin[i]
	if !ok || pos < 0 || pos >= len(r.names) {
		return "", false
	}
	return r.names[pos], true
}

// isBotIndex : l indice est-il epingle sur un BOT ?
//
// `pin` PORTE DEUX EPINGLAGES DEPUIS LE LOT 1.8 — BOT_METADATA et la table du film — et ce
// predicat ne doit reconnaitre que le premier. Le lire comme « present dans pin » faisait passer
// pour bots les huit joueurs d un film entierement lu : la mini-bobine tombait de 10 lignes
// publiees a 2, les morts etant reclassees en « mort de bot », population jamais publiee. Le
// symptome a ete mesure avant d etre corrige (permutation IDENTIQUE, dix lignes devenues deux) —
// c est ce qui a nomme la cause.
func (r *roster) isBotIndex(i int) bool {
	if r.seatPin[i] || r.motifPin[i] {
		return false
	}
	_, ok := r.pin[i]
	return ok
}

// public : la vue exportee.
func (r *roster) public() Roster {
	out := Roster{Names: append([]string(nil), r.names...), Humans: r.nHumans,
		BotsSuccedes: r.botsSuccedes}
	for _, b := range r.bots.Bots {
		out.Bots = append(out.Bots, BotEntry{Slot: b.Slot, BotID: b.BotID, Name: b.Name})
	}
	for _, b := range r.unpinned {
		out.UnpinnedBots = append(out.UnpinnedBots, BotEntry{Slot: b.Slot, BotID: b.BotID, Name: b.Name})
	}
	out.IndexToName = make([]string, r.nPlay)
	out.IndexSource = make([]IndexOrigin, r.nPlay)
	for i := 0; i < r.nPlay; i++ {
		out.IndexToName[i] = r.nameOf(i)
		out.IndexSource[i] = r.originOf(i)
	}
	out.FilmTable = r.table
	return out
}
