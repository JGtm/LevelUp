package filmdec

// player_table.go — LA TABLE DES 32 JOUEURS DU MATCH, LUE DANS `chunk_00` (lot 1.5.2 et 1.5.3).
//
// # D'OU VIENT CETTE GRAMMAIRE : DU DESASSEMBLAGE, PAS DES OCTETS
//
// La derniere instruction du corps de `chunk_00` (`FUN_1407ec560`) est une boucle :
//
//	LEA RDI,[RSI + 0xeaaf0]      ; premier enregistrement
//	LEA RSI,[RDI + 0x28a00]      ; borne de fin
//	... FUN_1407ecb08(enregistrement, writer) ; enregistrement += 0x1450
//
// `0x28A00 / 0x1450 = 32` : la table porte TRENTE-DEUX enregistrements, un par slot du match.
// `FUN_1407ecb08` en ecrit l'en-tete (85 bits puis le XUID sur 64) et delegue le reste a
// `FUN_1407edea8`, dont le desassemblage donne l'ordre d'ecriture et la largeur de CHAQUE champ
// (releve du 2026-09-12, Ghidra base `0x140000000`, lecture seule —
// `NOTE_SECTION3_SLOTS_2026-09-12.md` section A, instrument permanent
// `section3_slot_grammar_research_test.go`) :
//
//	slot+0x00/01/02  1+1+1 bits   trois booleens (1/0/0 sur un slot occupe)
//	slot+0x04        32 bits      u32
//	slot+0x08        2 bits       octet SIGNE (domaine -2..1, PAS constant : cf. le balayage)
//	slot+0x09        48 bits      jeton propre au couple (match, joueur)
//	slot+0x10        64 bits      LE XUID
//	sub+0x000        11 bits + N  masque de presence, prefixe du rang du bit haut (FUN_1424ccf94)
//	sub+0x100/0x108  12 bits + N  longueur puis N octets                  (FUN_1411b1a24)
//	sub+0x908/0x910  8 bits + M   longueur puis M mots de 32 bits          (FUN_1411b198c)
//	sub+0xc48        832 bits     104 octets bruts
//	sub+0xc14        <= 16 unites LE GAMERTAG, UTF-16 MSB d'abord, arret APRES l'unite nulle
//	sub+0xc38        128 bits     16 octets bruts
//	sub+0xcb0        32 bits      `desired-representation`                 (FUN_1407edaf4)
//	sub+0xcb8        64 bits
//	sub+0xc12/c36/c35/c10/c34/c11  10+14+6+8+7+1 bits   les six champs courts
//	sub+0xcc0        LARGEUR PAR BUILD   le bloc de personnalisation (player_table_profile.go)
//	sub+0x1400       352 bits     44 octets bruts, le second champ de nom
//	slot+0x1448      32 bits      u32 de queue
//
// # LES DEUX FERMETURES QUI DISENT QUE LA GRAMMAIRE EST COMPRISE
//
//  1. LA LONGUEUR SE PREDIT. A partir de quatre nombres LUS DANS LE FLUX (compte du masque, N,
//     M, longueur du nom), la grammaire predit la longueur TOTALE d'un enregistrement ; elle
//     doit valoir exactement l'ecart mesure jusqu'au suivant. Mesure : 38/38 sur six films
//     temoins, 2 121 ecarts sur 2 121 (96 films) une fois les slots VACANTS enjambes.
//  2. LE LECTEUR LIT 32 SLOTS, NI PLUS NI MOINS. Occupes plus vacants, sur 1 351 films du cache
//     sur 1 351 (2026-09-14) — c'est la borne `0x28A00 / 0x1450` de l'ecrivain, verifiee par la
//     lecture sur tout le corpus et sur sept builds. Une seule largeur fausse la casserait.
//
// # UN SLOT VACANT EST INVISIBLE AU BALAYAGE, ET SA LONGUEUR SE CALCULE
//
// Un enregistrement vacant porte `0/0/0` en tete (le balayage exige `1/0/0`) et un XUID nul
// (hors de la plage Xbox) : il ne coute donc pas un enregistrement, il DOUBLE l'ecart entre ses
// deux voisins. Sa longueur ne se mesure pas, elle se CALCULE — chaque terme est une largeur
// deja lue chez l'ecrivain, evaluee pour des champs tous nuls (cf. slotVacantBits). Verification
// sur les deux ecarts aberrants du corpus : surplus mesure = longueur calculee, au bit pres,
// reste 0 (`NOTE_RESIDUS_CHUNK00_2026-09-13.md` section 1).
//
// # POURQUOI LE DEPART SE CHERCHE, ET COMMENT IL SE VERIFIE
//
// Le corps de `chunk_00` porte, avant la table, des sous-serialiseurs NON DECODES
// (`FUN_140b857d8`, `FUN_1410bc140`) : la position du premier enregistrement n'est donc pas
// derivable, elle se cherche. Ce lecteur ne retient PAS un candidat sur un seuil d'ecart (le
// `40 000` bits des instruments de recherche) : le depart est le PREMIER enregistrement reel du
// balayage, recule des slots vacants qui le precedent. Une lecture est RECEVABLE si la marche
// ferme a 32 slots ET si elle VISITE TOUS les enregistrements du balayage ; entre deux lectures
// recevables, la plus COMPLETE gagne. Le second volet n est pas un luxe : la queue du tampon est
// faite de zeros, donc le predicat de vacance y passe indefiniment et « 32 slots » seul se
// satisfait de n importe quelle largeur (cf. chercherDepart).
//
// # CE LECTEUR N'A PAS DE CONSOMMATEUR (lot 1.5), et il ne pose aucun reglage : il
// est une fonction pure de (octets, profil), sans variable de paquet ni crochet (D-5, ADR 0034).

// ErrPlayerTableNotFound : aucun depart ne ferme la table a 32 slots.
//
// C'est la forme que prend, ici, un `chunk_00` tronque dans le corps : la marche par la
// grammaire sort du tampon avant le 32e slot. Erreur typee, jamais de panique ni de table
// partielle rendue en silence.
const ErrPlayerTableNotFound = chunk00Error(
	"filmdec: aucune table de 32 slots dans le corps de chunk_00")

const (
	// playerTableSlots : la borne de l'ecrivain, `0x28A00 / 0x1450`.
	playerTableSlots = 32
	// slotBooleensTete : les trois booleens de tete d'un slot OCCUPE, lus comme un champ de
	// 3 bits MSB d'abord : `1/0/0` = 4.
	slotBooleensTete = 4
	// slotHeaderBits : 1+1+1+32+2+48, l'en-tete qui precede le XUID (FUN_1407ecb08).
	slotHeaderBits = 85
	// slotTokenBit : l'octet-bit du jeton de 48 bits dans l'en-tete (3 + 32 + 2).
	slotTokenBit = 37
	slotXUIDBits = 64
	// Largeurs du sous-enregistrement, lues chez l'ecrivain (cf. l'en-tete de fichier).
	slotMaskPrefixBits   = 11
	slotMaskMaxBits      = 2048
	slotListNBits        = 12
	slotListNMax         = 2048
	slotListMBits        = 8
	slotListMMax         = 192
	slotBloc104Bits      = 832
	slotGamertagMaxUnits = 16
	slotBloc16Bits       = 128
	slotReprBits         = 32
	slotQ64Bits          = 64
	slotCourtsBits       = 10 + 14 + 6 + 8 + 7 + 1
	slotBloc44Bits       = 352
	slotQueueU32Bits     = 32
	// slotFixeHorsPerso : tout ce que l'enregistrement ecrit de largeur FIXE, hors le bloc de
	// personnalisation (dont la largeur depend du build) et hors les parties de longueur
	// variable (masque, deux listes, gamertag).
	slotFixeHorsPerso = slotHeaderBits + slotXUIDBits + slotMaskPrefixBits + slotListNBits +
		slotListMBits + slotBloc104Bits + slotBloc16Bits + slotReprBits + slotQ64Bits +
		slotCourtsBits + slotBloc44Bits + slotQueueU32Bits
	// slotVacantHorsPerso : la longueur d'un enregistrement ENTIEREMENT A ZERO, hors le bloc de
	// personnalisation. Elle est CALCULEE terme a terme, pas ajustee sur une mesure : en-tete 85,
	// XUID nul 64, prefixe de masque 11 et le masque reduit a UN bit (rang du bit haut = 0),
	// N nul 12, M nul 8, bloc de 104 octets 832, la chaine reduite a son NUL 16, bloc de 16
	// octets 128, `desired-representation` 32, le champ de 64 bits 64, les six champs courts 46,
	// puis le bloc de queue 352 et le u32 final 32. Somme = 1 683.
	slotVacantHorsPerso = slotHeaderBits + slotXUIDBits + slotMaskPrefixBits + 1 +
		slotListNBits + slotListMBits + slotBloc104Bits + 16 + slotBloc16Bits + slotReprBits +
		slotQ64Bits + slotCourtsBits + slotBloc44Bits + slotQueueU32Bits
	// slotXuidLo / slotXuidHi : la plage des XUID Xbox Live. Bornes de la recherche d'origine,
	// ecrites avant la mesure ; elles ont servi a fabriquer les leurres du controle negatif
	// (240 leurres de meme forme, 0 touche).
	slotXuidLo = uint64(0x0009000000000000)
	slotXuidHi = uint64(0x000A000000000000)
	// slotNomMinImprimable : un champ de nom rend un texte d'au moins trois caracteres ASCII
	// imprimables, sans quoi la position est un PARASITE du balayage (un motif d'en-tete
	// fortuit a l'interieur d'un vrai enregistrement).
	slotNomMinImprimable = 3
)

// PlayerSlotShorts : les champs COURTS d'un enregistrement de slot.
//
// Ils sont publies parce qu'ils sont lus, et parce qu'un champ mesure CONSTANT qui se met a
// varier est le premier signe qu'une grammaire a bouge. Mesure du 2026-09-12 sur les 44
// enregistrements de six films : huit d'entre eux sont constants sur tout le corpus (`Deux`=0,
// `F10`=183, `F14`=0, `F6`=-1, `F8`=0, `F7`=0, `Repr`=0, `Tete`=0) ; le neuvieme, `F1`, prend
// deux valeurs et ce sont exactement les enregistrements a listes vides contre ceux a listes
// pleines — un drapeau de presence des listes. AUCUN ne porte l'equipe : c'est mesure, et ferme
// par la negative (aucun ne partage un roster en deux moities egales, sur 0/6 films).
type PlayerSlotShorts struct {
	Tete uint32 // slot+0x04, 32 bits
	Deux uint32 // slot+0x08, 2 bits (octet signe : domaine -2..1)
	Repr uint32 // sub+0xcb0, 32 bits, etiquete `desired-representation`
	Q64  uint64 // sub+0xcb8, 64 bits
	F10  uint32 // sub+0xc12, 10 bits
	F14  uint32 // sub+0xc36, 14 bits
	F6   int    // sub+0xc35, 6 bits, ecrit VALEUR+1 : rendu signe (-1 sur un slot occupe)
	F8   uint32 // sub+0xc10, 8 bits
	F7   uint32 // sub+0xc34, 7 bits
	F1   uint32 // sub+0xc11 & 1, 1 bit
}

// PlayerSlot : un slot OCCUPE de la table.
type PlayerSlot struct {
	// FilmIndex : le RANG du slot dans la table de 32, VACANTS COMPRIS. C'est l'index du
	// tableau que l'ecrivain parcourt (`enregistrement += 0x1450`), donc l'index du slot.
	//
	// CE QUE LA MESURE DIT, ET CE QU'ELLE NE DIT PAS. L'ordre des enregistrements EST le
	// `player_index` de production : `filmIndex - rang` est CONSTANT sur 76 films sur 76, et la
	// coincidence est totale sur 72 (2026-09-12, oracle = le `roster[].filmIndex` des documents
	// de rejeu). Mais aucun de ces 76 films ne porte de slot vacant INTERCALE, donc l'oracle ne
	// separe pas « rang absolu » de « index parmi les occupes » : les deux definitions y
	// coincident. Les 5 films du cache a vacant intercale (`07f6af1b`, `0d1dddfb`, `1c5c10cc`,
	// `b1bcbe24`, `c744aa29`, mesures le 2026-09-14) n'ont AUCUN document de rejeu — la question
	// n'est donc pas tranchable sur ce corpus, et le rapport porte `InterleavedVacant` pour que
	// le consommateur sache quand les deux lectures divergent.
	FilmIndex int
	XUID      uint64
	Gamertag  string
	// SessionToken : le champ de 48 bits de `slot+0x09`. Propre au couple (match, joueur) :
	// cinq valeurs differentes a forte entropie pour un meme XUID sur cinq films. Lecture la
	// plus economique : un jeton de session. NON PROUVE.
	SessionToken uint64
	// Bit / TotalBits : la position et la longueur de l'enregistrement dans le flux, pour qu'un
	// rapport ou un instrument puisse revenir dessus sans re-chercher.
	Bit       int
	TotalBits int
	Shorts    PlayerSlotShorts
}

// PlayerTableReport : ce que la lecture a vu, et ce que le CONTROLE en dit.
type PlayerTableReport struct {
	// Build / PersoBytes / ProfileDeltaBits : ce que le PROFIL dit (player_table_profile.go).
	Build            string
	PersoBytes       int
	ProfileDeltaBits int
	// FilmDeltaBits : le calibrage LU SUR LE FILM — la transposition modale des ecarts mesures,
	// exprimee dans la meme unite que `ProfileDeltaBits`. C'EST UN CONTROLE, JAMAIS UNE SOURCE
	// (D-3, ADR 0034) : aucune lecture n'en depend. `CalibrationAgrees` dit s'il tombe sur la
	// valeur du profil ; `FilmDeltaGaps` dit sur combien d'ecarts il s'appuie.
	FilmDeltaBits     int
	FilmDeltaGaps     int
	CalibrationAgrees bool
	// GapsAgree / GapsVacant / GapsContradict : la ventilation des ecarts mesurables entre
	// candidats du balayage. `Vacant` = l'ecart depasse la prediction d'un nombre ENTIER de
	// slots vacants, chacun verifie par le predicat grammatical a sa position calculee.
	// `Hidden` = l intervalle porte un enregistrement que la MARCHE a lu et que le balayage ne
	// pouvait pas voir (jeton nul, XUID hors plage). `Contradict` = rien de tout cela : la
	// grammaire ne ferme pas la, et cela se compte.
	GapsAgree      int
	GapsVacant     int
	GapsHidden     int
	GapsContradict int
	// Occupied / Vacant : la table lue. Leur somme vaut 32 des que l'erreur est nulle.
	Occupied int
	Vacant   int
	// HeadVacant : les slots vacants qui PRECEDENT le premier occupe. Mesure du 2026-09-14 :
	// 0 sur les 1 351 films du cache — le slot 0 est occupe partout.
	HeadVacant int
	// InterleavedVacant : un slot vacant tombe AVANT le dernier occupe. C'est le seul cas ou
	// « rang absolu » et « index parmi les occupes » divergent (cf. PlayerSlot.FilmIndex).
	InterleavedVacant bool
	// FirstRecordBit : la position du slot 0 dans le flux.
	FirstRecordBit int
	// CandidatesScanned : les en-tetes que le balayage a trouves, PARASITES COMPRIS.
	// CandidatesReal : ceux qui rendent un nom imprimable. `CandidatesReal - Occupied` compte les
	// PARASITES IMPRIMABLES — un motif d en-tete fortuit a l interieur d un vrai enregistrement
	// dont le champ de nom rend quand meme du texte. Mesure du 2026-09-14 sur les 1 351 films du
	// cache : rarissime, et un seul film en porte un (`d4ddf054`, deux positions separees de
	// 10 170 bits, moins que le plus court enregistrement mesure).
	CandidatesScanned int
	CandidatesReal    int
}

// ReadPlayerTable lit les 32 slots de la table des joueurs d'un `chunk_00` DEJA DECOMPRESSE,
// avec l'identite deja lue par [ReadFilmIdentity] (elle porte le build, donc le profil, et le
// premier bit du corps).
//
// Les slots VACANTS sont ECARTES de la tranche rendue — ils n'ont ni XUID ni nom — mais ils
// comptent dans le rang (`PlayerSlot.FilmIndex`) et dans le rapport.
//
// DEUX BORNES, ET ELLES NE SONT PAS INTERCHANGEABLES. Le BALAYAGE s'arrete au dernier octet
// ECRIT : au-dela il n'y a que des zeros, donc aucun en-tete de slot occupe a trouver. Les
// LECTURES, elles, vont jusqu'au bout du TAMPON — parce qu'un enregistrement VACANT est ecrit
// entierement a zero et tombe donc, en queue de table, APRES le dernier octet non nul. Borner
// les lectures sur la zone ecrite ferait echouer la marche sur les slots vacants de queue,
// c'est-a-dire sur 24 des 32 slots d'une partie d'arene (constate a l'ecriture de ce lot).
//
// Erreurs typees : [ErrUnknownBuild] enveloppe avec le nom du build (D-4 : le film est mis de
// cote, JAMAIS lu au profil du build voisin ; publier [UnknownBuildExpvarPairs] au meme
// endroit), [ErrChunk00Truncated], [ErrPlayerTableNotFound].
func ReadPlayerTable(chunk0 []byte, ident FilmIdentity) ([]PlayerSlot, PlayerTableReport, error) {
	rep := PlayerTableReport{Build: ident.Build}
	octets, connu := personnalisationOctets(ident.Build)
	if !connu {
		return nil, rep, erreurBuildInconnu(ident.Build)
	}
	rep.PersoBytes, rep.ProfileDeltaBits = octets, persoDeltaBits(octets)
	persoBits := octets * 8
	finEcrit, finTampon := (dernierOctetNonNul(chunk0)+1)*8, len(chunk0)*8
	if ident.BodyBit <= 0 || finEcrit <= ident.BodyBit {
		return nil, rep, ErrChunk00Truncated
	}
	candidats := balayerCandidats(chunk0, ident.BodyBit, finEcrit)
	rep.CandidatesScanned = len(candidats)
	dt, ok := chercherDepart(chunk0, candidats, finTampon, persoBits)
	rep.CandidatesReal = dt.essais
	if !ok {
		return nil, rep, ErrPlayerTableNotFound
	}
	remplirRapport(&rep, dt)
	controlerCalibrage(chunk0, candidats, finTampon, persoBits, dt.slots, &rep)
	return dt.slots, rep, nil
}

// departTable : ce qu'une marche fermee a rendu.
type departTable struct {
	slots       []PlayerSlot
	vacants     int
	teteVacants int
	essais      int
	premierBit  int
}

// remplirRapport reporte la marche dans le rapport.
func remplirRapport(rep *PlayerTableReport, dt departTable) {
	rep.Occupied, rep.Vacant, rep.HeadVacant = len(dt.slots), dt.vacants, dt.teteVacants
	rep.FirstRecordBit = dt.premierBit
	// Sans vacant INTERCALE, le dernier occupe a pour rang `vacants de tete + occupes - 1` :
	// tout ce qui reste de vacant est alors en QUEUE de table. Un rang superieur signe un
	// vacant pris entre deux occupes — le seul cas ou « rang absolu » et « index parmi les
	// occupes » divergent.
	if n := len(dt.slots); n > 0 {
		rep.InterleavedVacant = dt.slots[n-1].FilmIndex != dt.teteVacants+n-1
	}
}

// chercherDepart trouve le slot 0 de la table et la lit.
//
// # CE QU'UNE LECTURE DOIT SATISFAIRE POUR ETRE RECEVABLE
//
// Un depart candidat est un enregistrement du balayage, recule d'un nombre de slots VACANTS que
// le balayage ne peut pas voir (tete `0/0/0`, XUID nul) : sans ce recul, un roster dont le
// slot 0 est vacant ferait commencer la table trop loin et le rang de chaque joueur serait faux
// d'autant. Le recul s'arrete des que le predicat grammatical de vacance ne passe plus.
//
// La lecture qui en sort est RECEVABLE si la marche ferme a 32 slots ET si elle VISITE TOUS les
// enregistrements du balayage. Le second volet n'est pas un luxe : la queue du tampon est faite
// de zeros, donc le predicat de vacance y passe INDEFINIMENT, et « 32 slots » seul se satisfait
// de n'importe quelle largeur de bloc — mesure a l'ecriture de ce lot, quatre largeurs fausses
// sur cinq bobines. Partir du dernier enregistrement ne visite pas les precedents ; une largeur
// fausse deraille des le deuxieme slot et n'en visite plus aucun.
//
// CE QUE LE SECOND VOLET N'EXIGE PAS, ET IL A FALLU LE MESURER POUR LE SAVOIR : que la marche ne
// visite QUE des enregistrements du balayage. Le balayage exige un jeton de 48 bits non nul et un
// XUID de la plage Xbox ; un enregistrement qui sort de ces bornes lui est INVISIBLE, alors que
// la marche l'atteint par la grammaire et le lit sans peine. Sur `d4ddf054` (HI_1_13_0), le
// balayage rend 7 enregistrements la ou la marche en lit 8 — et c'est la marche qui a raison :
// la trame du meme film porte 8 entites `ti=9`, c'est-a-dire 8 joueurs. Exiger l'egalite des deux
// suites faisait tomber le film entier ; l'inclusion dans ce sens-la le lit juste.
//
// # ENTRE DEUX LECTURES RECEVABLES, LA PLUS COMPLETE GAGNE
//
// Ce n'est pas un seuil, c'est la seule regle qui se justifie : une lecture qui explique HUIT
// enregistrements explique le film mieux qu'une qui n'en explique qu'UN.
func chercherDepart(d []byte, candidats []int, finBit, persoBits int) (departTable, bool) {
	reels := candidatsReels(d, candidats, finBit, persoBits)
	dt := departTable{essais: len(reels)}
	if len(reels) == 0 {
		return dt, false
	}
	vide := slotVacantBits(persoBits)
	trouve := false
	for tete := 0; tete < playerTableSlots && reels[0]-tete*vide >= 0; tete++ {
		depart := reels[0] - tete*vide
		slots, vac, ferme := marcherTable(d, depart, finBit, persoBits)
		if ferme && visiteTousLesReels(slots, reels) && len(slots) > len(dt.slots) {
			dt.slots, dt.vacants, dt.teteVacants, dt.premierBit = slots, vac, tete, depart
			trouve = true
		}
		if !slotVacant(d, depart-vide, finBit) {
			break
		}
	}
	return dt, trouve
}

// candidatsReels ecarte les positions PARASITES du balayage : celles dont le champ de nom ne
// rend pas de texte imprimable.
func candidatsReels(d []byte, candidats []int, finBit, persoBits int) []int {
	out := make([]int, 0, len(candidats))
	for _, c := range candidats {
		if e, ok := decodeSlot(d, c, finBit, persoBits); ok && gamertagImprimable(e.slot.Gamertag) {
			out = append(out, c)
		}
	}
	return out
}

// visiteTousLesReels dit si la marche est passee par CHAQUE enregistrement du balayage. Les
// enregistrements que la marche lit EN PLUS sont ceux que le balayage ne pouvait pas voir
// (jeton de 48 bits nul, XUID hors de la plage Xbox) ; le rapport les compte
// (`Occupied - CandidatesReal`).
func visiteTousLesReels(slots []PlayerSlot, reels []int) bool {
	if len(slots) == 0 || len(reels) == 0 {
		return false
	}
	vus := positionsRetenues(slots)
	for _, r := range reels {
		if !vus[r] {
			return false
		}
	}
	return true
}

// marcherTable lit 32 slots consecutifs a partir de `depart` : un vacant s'enjambe de sa
// longueur calculee, un occupe se decode et avance de sa longueur PREDITE. `ferme` dit que les
// 32 slots ont ete lus — la borne de l'ecrivain, verifiee par la lecture.
func marcherTable(d []byte, depart, finBit, persoBits int) (slots []PlayerSlot, vacants int,
	ferme bool) {
	vide := slotVacantBits(persoBits)
	p := depart
	for rang := 0; rang < playerTableSlots; rang++ {
		if slotVacant(d, p, finBit) {
			vacants++
			p += vide
			continue
		}
		e, ok := decodeSlot(d, p, finBit, persoBits)
		if !ok || !gamertagImprimable(e.slot.Gamertag) {
			return slots, vacants, false
		}
		e.slot.FilmIndex = rang
		slots = append(slots, e.slot)
		p += longueurPredite(e, persoBits)
	}
	return slots, vacants, true
}

// balayerCandidats rend les positions de bit ou un en-tete de slot OCCUPE commence : trois
// booleens `1/0/0`, u32 de `slot+0x04` nul, XUID dans la plage Xbox (bornes exclues) et jeton de
// 48 bits non nul.
//
// LE CHAMP DE 2 BITS DE `slot+0x08` N'EST PAS UN CRITERE, et c'est une correction mesuree :
// `FUN_1407ecb08` l'ecrit comme un octet SIGNE sur 2 bits, donc son domaine est -2..1, et deux
// enregistrements bien reels du cache le portent a 1. L'exiger nul rendait ces slots invisibles,
// et leur absence doublait l'ecart au voisin — ce qui faisait perdre au regroupement d'origine
// TOUTE LA TETE de la table (11 enregistrements lus au lieu de 24 sur `1c4c63c2`).
func balayerCandidats(d []byte, debutBit, finBit int) []int {
	var out []int
	br := NewBitReader(d)
	for p := debutBit; p+slotHeaderBits+slotXUIDBits <= finBit; p++ {
		br.SetBitPos(p)
		if br.ReadBits(3) != slotBooleensTete || br.ReadBits(32) != 0 {
			continue
		}
		br.SetBitPos(p + slotHeaderBits)
		if x := br.ReadBits(slotXUIDBits); x <= slotXuidLo || x >= slotXuidHi {
			continue
		}
		br.SetBitPos(p + slotTokenBit)
		if br.ReadBits(48) == 0 {
			continue
		}
		out = append(out, p)
	}
	return out
}
