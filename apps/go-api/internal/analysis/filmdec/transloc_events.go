package filmdec

// transloc_events.go — LA TÉLÉPORTATION DU TRANSLOCATEUR, LUE DANS LES ÉVÉNEMENTS DU FILM :
// QUAND, QUI, ET D'OÙ À OÙ.
//
// CE QUE C'EST. Chaque usage du translocateur émet un événement type 117
// `EquipmentTranslocatorTeleportEffects` (nom sourcé de l'exécutable, lecteur 0x140f04fb8)
// dans la liste d'événements en tête d'un paquet delta — le même canal que `unit_zoom`
// (zoom_events.go), qui est le patron de ce fichier. L'événement porte la référence du
// bipède qui saute, PUIS LES DEUX POSITIONS DU VA-ET-VIENT, et il précède la discontinuité de
// position de 5 à 80 ms.
//
// LA MESURE QUI FONDE LE FICHIER (rapport R1 §4, 2026-09-03, 5 films) : précision 18/18
// (chaque tête 117 correspond à une discontinuité vérifiée de 3,24 à 25,36 m du slot
// désigné), rappel 8/8 sur les téléportations connues — y compris un saut de 3,24 m
// invisible à tout seuil spatial. AUCUNE heuristique : l'événement EST la mesure.
//
// LA FORME DU RECORD (mesurée sur pièces, R1 §4.2) : premier octet 0xFA = bit de
// configuration (1) + bit de présence (1) + les 6 bits hauts du type ; le bit de poids
// faible du type est le premier bit du 2e octet (117 = 1110101). Puis ref0 = l'unité qui se
// téléporte : porte 1 bit, index 8 bits (domaine 2), génération 2 bits — l'index plus 512
// donne le SLOT du bipède, la même fermeture que le zoom (base mesurée, 18/18 slots
// plausibles). Les refs 1 et 2 sont absentes (portes à 0) sur toutes les observations.
//
// LA CHARGE, LUE DANS L'EXE ET VALIDÉE 18/18 (rapport R6 §1, décompilation FUN_140f04fb8 +
// FUN_14080d69c + FUN_14076e524 + FUN_140cc5128 ; écrivain FUN_142eec354 symétrique) :
//
//	[R(1) g0 ; si g0 : R(32) mot]   identifiant d'effet — mesuré constant 0xA1344FC2
//	position A : [R(1) porte ; si porte==0 : R(wr) index de région] R(bx) R(by) R(bz)
//	position B : idem
//
// A est le DÉPART du saut, B son ARRIVÉE — dans cet ordre sur les 18 événements des 5 films,
// à 0,00-0,26 m des discontinuités de piste correspondantes.
//
// DEUX PIÈGES, SOURCÉS ET RESPECTÉS ICI :
//
//  1. LA PORTE DE RÉGION EST INVERSÉE. Le bloc « lire l'index de région et prendre les bornes
//     de CETTE région » s'exécute quand le bit vaut ZÉRO ; le bit à UN sélectionne les bornes
//     par défaut du moteur (±20000, 22 bits par axe, DAT_143b8c6b8). C'est l'inverse de la
//     lecture naïve, et la validation film confirme ce sens.
//  2. LES BORNES SONT CELLES DU CATALOGUE (`map_quant_bounds.json`, MapQuantEntry), JAMAIS le
//     champ `bounds` d'un artefact de rejeu, qui n'est qu'un cadrage d'affichage. Déquantifier
//     avec ce dernier fut l'erreur qui a fait manquer les positions à la sonde de R1.
//
// PAS DE BORNES -> PAS DE POSITION (règle map_bounds.go). Une carte absente du catalogue, une
// entrée sans largeurs d'axe, une région autre que celle que le catalogue décrit, une charge
// non conforme : la téléportation sort SANS positions, et l'appelant compte le cas. Jamais une
// position devinée — l'instant et le slot, eux, restent lus.
//
// RÉSERVE HONNÊTE, la même que zoom_events.go : seuls les événements EN TÊTE de paquet sont
// lus. Trois `spent` du corpus n'ont aucune tête 117 de leur slot (expiration sans usage, ou
// événement hors tête de liste — non départagé, R1 §4.3). Lire la liste entière est le
// chantier PLAN_PERCER_TRAME_FILM.

import "sort"

const (
	// translocFamilyByte : le premier octet des paquets dont l'événement de tête est le
	// type 117 — config(1) + présence(1) + les 6 bits hauts de 1110101.
	translocFamilyByte = 0xFA
	// translocEventType : le numéro de type de `EquipmentTranslocatorTeleportEffects` sous
	// la numérotation du dispatcher.
	translocEventType = 117
	// translocRefWidth : largeur de l'index de ref0 (domaine 2 de la table des domaines).
	translocRefWidth = 8
	// translocGenBits : largeur du champ de génération de ref0.
	translocGenBits = 2
	// translocSlotBase : la base du domaine — même fermeture mesurée que zoomSlotBase.
	translocSlotBase = 512
	// translocEffectWordBits : largeur du mot gardé qui ouvre la charge (l'identifiant
	// d'effet, mesuré constant 0xA1344FC2 sur les 18 événements). Sa VALEUR n'est pas
	// contrôlée : c'est un identifiant de contenu, pas une signature de format — un autre
	// effet en changerait sans rien changer aux positions qui suivent.
	translocEffectWordBits = 32
	// translocDefaultBound / translocDefaultAxisBits : bornes et largeurs PAR DÉFAUT du
	// moteur (DAT_143b8c6b8 = ±20000, 22 bits par axe), la branche que la porte de région
	// ouvre quand son bit vaut 1. Jamais observée sur les 18 événements des 5 films : elle
	// est implémentée parce qu'elle EST le layout, pas parce qu'elle a été vue.
	translocDefaultBound    = 20000
	translocDefaultAxisBits = 22
	// translocMaxAxisBits : plafond de la loi du moteur sur une largeur d'axe
	// (min(26, ...), cf. MapQuantEntry.AxisWidths). translocMaxRegionBits borne de même la
	// largeur de l'index de région. Au-delà, l'entrée de catalogue est refusée plutôt que
	// lue : une largeur aberrante ferait lire n'importe quoi.
	translocMaxAxisBits   = 26
	translocMaxRegionBits = 8
)

// TranslocatorTeleport est UNE téléportation exécutée : quand, par quel bipède — et, quand la
// charge a pu être lue, d'où à où.
type TranslocatorTeleport struct {
	// TimestampUS : l'horodatage du paquet porteur — MÊME horloge que
	// BipedPosition.TimestampUS (l'horloge MOTEUR des paquets, cf. le piège documenté au
	// rapport R1 §0 : elle n'est PAS la timeline de l'artefact).
	TimestampUS uint64
	// Slot : le slot du bipède qui se téléporte, directement comparable à
	// BipedPosition.Slot.
	Slot uint32
	// From / To : le DÉPART et l'ARRIVÉE du saut en coordonnées monde (X, Y, Z), déquantifiés
	// aux bornes VRAIES de la carte. Valides SEULEMENT si HasPositions ; à lire ensemble,
	// jamais l'une sans l'autre.
	From, To [3]float32
	// HasPositions dit que la charge a été lue ET déquantifiée. Faux = l'instant et le slot
	// restent mesurés, les positions ne sont pas connues (carte hors catalogue, entrée sans
	// largeurs, région étrangère, ou charge non conforme au layout). Un appelant qui
	// lirait From/To sans ce témoin publierait l'origine du monde comme une position.
	HasPositions bool
}

// ScanFilmTranslocatorTeleports lit les téléportations du translocateur d'un film, triées
// par instant.
//
// `entry` est l'entrée de catalogue de la carte du match — celle qui porte les bornes de
// déquantification. Nil (ou inutilisable) n'est pas une erreur : les événements sortent
// datés et attribués, mais sans positions.
//
// Lecture seule, sans état global de décodage : comme le zoom, ce scanner n'appelle aucun
// déserialiseur de trame et n'a pas besoin du verrou de paquet. Les chunks illisibles sont
// une couverture moindre, pas une erreur.
func ScanFilmTranslocatorTeleports(dir string, entry *MapQuantEntry) []TranslocatorTeleport {
	var out []TranslocatorTeleport
	for c := 1; c <= CountFilmChunks(dir); c++ {
		chunk, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(chunk) {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(chunk)
			if pay[0] != translocFamilyByte {
				continue
			}
			if ev, ok := decodeTranslocHead(pay, pk.TimestampUS, entry); ok {
				out = append(out, ev)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TimestampUS < out[j].TimestampUS })
	return out
}

// decodeTranslocHead lit l'événement de tête d'un paquet de la famille et rend la
// téléportation. ok=false si la tête n'est pas un type 117 ou si l'unité n'est pas désignée
// (porte de ref0 à 0 — jamais observé, mais un slot non transmis ne se devine pas). La
// CHARGE, elle, ne conditionne rien : elle échoue en positions absentes, pas en événement
// perdu (l'instant et le slot sont déjà lus).
func decodeTranslocHead(pay []byte, tsUS uint64, entry *MapQuantEntry) (TranslocatorTeleport, bool) {
	br := NewBitReader(pay)
	br.Skip(1) // bit de configuration
	if !br.ReadBit() {
		return TranslocatorTeleport{}, false // liste vide : pas d'événement en tête
	}
	if int(br.ReadBits(7)) != translocEventType {
		return TranslocatorTeleport{}, false
	}
	if !br.ReadBit() {
		return TranslocatorTeleport{}, false // ref0 absente : pas d'unité désignée
	}
	idx := br.ReadBits(translocRefWidth)
	br.Skip(translocGenBits)
	ev := TranslocatorTeleport{TimestampUS: tsUS, Slot: uint32(idx + translocSlotBase)}
	ev.From, ev.To, ev.HasPositions = decodeTranslocJump(br, entry)
	return ev, true
}

// decodeTranslocJump lit la CHARGE de l'événement après ref0 : les portes des refs 1-2, le
// mot d'effet gardé, puis les DEUX positions quantifiées. Rend (départ, arrivée, lues).
func decodeTranslocJump(br *BitReader, entry *MapQuantEntry) ([3]float32, [3]float32, bool) {
	var none [3]float32
	// LES DEUX PORTES SE LISENT, PAS UNE. Un `||` court-circuitait la seconde (SA4000) : sans
	// conséquence ici — le chemin sort aussitôt et le lecteur de bits est abandonné — mais
	// l'intention (deux portes, lues dans l'ordre du flux) appartient au code, pas au hasard
	// d'un court-circuit.
	ref1, ref2 := br.ReadBit(), br.ReadBit()
	if ref1 || ref2 {
		// Une ref 1 ou 2 présente décale tout ce qui suit d'une largeur non sourcée pour ce
		// type (domaines {2,0,0}, R6 §1.1) : la charge n'est plus lisible. Jamais observé.
		return none, none, false
	}
	if br.ReadBit() {
		br.Skip(translocEffectWordBits)
	}
	from, ok := readTranslocVec(br, entry)
	if !ok {
		return none, none, false
	}
	to, ok := readTranslocVec(br, entry)
	if !ok {
		return none, none, false
	}
	// PAS DE SECOND CONTRÔLE DE DÉBORDEMENT ICI, et c'est délibéré : `readTranslocVec` est la
	// SEULE garde, et elle refuse déjà tout vecteur dont les bits dépassent le tampon (le
	// BitReader lit des zéros au-delà — padding de queue du moteur). Un `br.Remaining() < 0`
	// ajouté après coup serait INATTEIGNABLE, et sa présence masquerait la disparition de la
	// vraie garde (revue P1bis ronde 1, G4 : la redondance rendait les deux mutations vertes).
	return from, to, true
}

// readTranslocVec lit UNE position quantifiée de la charge et la déquantifie en coordonnées
// monde. PORTE INVERSÉE (cf. l'en-tête, piège n°1) : bit à 0 -> index de région puis bornes
// de la carte ; bit à 1 -> bornes par défaut du moteur.
func readTranslocVec(br *BitReader, entry *MapQuantEntry) ([3]float32, bool) {
	var out [3]float32
	widths := [3]uint{translocDefaultAxisBits, translocDefaultAxisBits, translocDefaultAxisBits}
	rng := Vec3Range{
		{Min: -translocDefaultBound, Max: translocDefaultBound},
		{Min: -translocDefaultBound, Max: translocDefaultBound},
		{Min: -translocDefaultBound, Max: translocDefaultBound},
	}
	if !br.ReadBit() {
		if !translocEntryUsable(entry) {
			return out, false // pas de bornes -> pas de coordonnée monde (map_bounds.go)
		}
		if uint32(br.ReadBits(entry.EffectiveRegionIndexBits())) != entry.Region {
			// Une AUTRE région : ses quanta sont exprimés dans une autre AABB, et les
			// déquantifier avec ces bornes produirait une position fausse silencieuse —
			// exactement le refus que porte I0Layout.Region sur le chemin du bipède.
			return out, false
		}
		widths, rng = entry.AxisWidths, entry.Range()
	}
	lay := I0Layout{AxisW: widths}
	for ax := 0; ax < 3; ax++ {
		out[ax] = DequantBipedAxis(uint32(br.ReadBits(widths[ax])), ax, lay, rng)
	}
	return out, br.Remaining() >= 0
}

// translocEntryUsable dit si l'entrée de catalogue permet une déquantification : bornes
// ordonnées et largeurs dans l'enveloppe de la loi du moteur. Une entrée hors enveloppe est
// REFUSÉE plutôt que lue — la largeur commande le nombre de bits consommés, une valeur
// aberrante ferait lire les bits du voisin.
func translocEntryUsable(e *MapQuantEntry) bool {
	if e == nil || e.EffectiveRegionIndexBits() > translocMaxRegionBits {
		return false
	}
	for ax := 0; ax < 3; ax++ {
		if e.AxisWidths[ax] == 0 || e.AxisWidths[ax] > translocMaxAxisBits {
			return false
		}
		if e.Max[ax] <= e.Min[ax] {
			return false
		}
	}
	return true
}
