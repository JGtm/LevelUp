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
	// Agree / Contradict / Silent : le CONTROLE des indices epingles par les votes du kill-feed.
	// `Agree` = les votes designent le meme joueur ; `Contradict` = ils en designent un autre,
	// strictement plus vote ; `Silent` = aucun vote sur cet indice (le joueur n a ni tue ni est
	// mort dans la fenetre d appariement).
	Agree, Contradict, Silent int
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
//  3. L INFERENCE en dernier, sur ce qui reste (bijection.go) — le REPLI, compte (`Inferred`).
func buildRoster(kf *killFeed, bm botMeta, useBots bool, t FilmTable) *roster {
	r := &roster{
		names:   append([]string(nil), kf.names...),
		pin:     map[int]int{},
		nPlay:   len(kf.names),
		nHumans: len(kf.names),
		bots:    bm,
		seatPin: map[int]bool{},
	}
	if useBots {
		r.pinBots(bm)
	}
	r.pinFilmSeats(t)
	// Les indices libres qui ne recoivent aucun nom (trous entre le dernier humain et un slot
	// de bot eleve) recoivent un nom de remplissage : le probleme d affectation doit avoir AU
	// MOINS autant de noms libres que d indices libres, sinon le hongrois n est pas defini.
	for len(r.names) < r.nPlay {
		r.names = append(r.names, fmt.Sprintf("?%d", len(r.names)))
	}
	return r
}

// pinBots : l epinglage des slots de bot, inchange depuis 2026-08 (cf. l en-tete du fichier).
func (r *roster) pinBots(bm botMeta) {
	for _, b := range bm.Bots {
		if b.Slot < r.nHumans || b.Slot >= 32 {
			r.unpinned = append(r.unpinned, b)
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

// originOf : la provenance du joueur porte par un indice.
func (r *roster) originOf(i int) IndexOrigin {
	switch {
	case r.seatPin[i]:
		return OriginFilmTable
	case r.isBotIndex(i):
		return OriginBotMeta
	case r.nameOf(i) == "?" || strings.HasPrefix(r.nameOf(i), "?"):
		return OriginNone
	default:
		return OriginInference
	}
}

// freeSlots : indices NON epingles, et positions de noms NON epinglees. Les deux listes ont la
// meme longueur par construction.
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

// isBotIndex : l indice est-il epingle sur un BOT ?
//
// `pin` PORTE DEUX EPINGLAGES DEPUIS LE LOT 1.8 — BOT_METADATA et la table du film — et ce
// predicat ne doit reconnaitre que le premier. Le lire comme « present dans pin » faisait passer
// pour bots les huit joueurs d un film entierement lu : la mini-bobine tombait de 10 lignes
// publiees a 2, les morts etant reclassees en « mort de bot », population jamais publiee. Le
// symptome a ete mesure avant d etre corrige (permutation IDENTIQUE, dix lignes devenues deux) —
// c est ce qui a nomme la cause.
func (r *roster) isBotIndex(i int) bool {
	if r.seatPin[i] {
		return false
	}
	_, ok := r.pin[i]
	return ok
}

// public : la vue exportee.
func (r *roster) public() Roster {
	out := Roster{Names: append([]string(nil), r.names...), Humans: r.nHumans}
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
