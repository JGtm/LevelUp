package filmdec

// vehicle_occupancy.go — LE BLOC SUPPLÉMENTAIRE DU RECORD D'IMAGE-CLÉ D'UN VÉHICULE : le
// localiser, le mesurer, et dire EXACTEMENT ce qu'il ne dit pas.
//
// D'OÙ ÇA VIENT. Le lot V5 (`.ai/V7.5/film_re/V5_ETAT_OCCUPATION_2026-09-02.md` § 6) a mesuré que
// le record d'image-clé d'un véhicule `ti=40` est PLUS LONG quand le véhicule est occupé, avec un
// écart récurrent de +89 bits sur trois films indépendants. Le lot V5B a ouvert ce bloc.
//
// CE QUI EST ÉTABLI (rapport V5B_BLOC_OCCUPATION_2026-09-02.md) :
//
//  1. LE BLOC EXISTE ET C'EST BIEN UNE INSERTION. Sur 18 blocs attestés, le modèle
//     « O = F[0:p] + BLOC(d) + F[p:] » explique 95,4 à 97,7 % des bits comparés, contre nettement
//     moins pour les deux modèles dégénérés (tout le décalage en tête / tout en queue). La taille
//     la plus fréquente est d = 89 bits (10 blocs sur 18) ; la position `p` varie de 364 à 422
//     bits depuis le début du record — le contenu qui la précède est de longueur variable.
//
//  2. LE BLOC NE NOMME PAS SON OCCUPANT. Balayage exhaustif de TOUT le dictionnaire d'encodages
//     du chantier (les 4 formes d'entité du lot V5, les références gardées de largeur 7/8/13 des
//     événements board/exit, le siège `R(6)`) à tous les décalages du bloc ± 48 bits : le
//     meilleur canal désigne le bon occupant sur 1 instance sur 18 (5,6 %) — EXACTEMENT le score
//     du témoin par permutation des occupants. Le siège semble passer à 16/18, mais le témoin
//     permuté fait 16/18 lui aussi : les 18 sièges attestés valent 0, le canal lit un zéro.
//
//  3. LE BLOC SUIT LA CINÉMATIQUE AU MOINS AUTANT QUE L'OCCUPATION. Sur 104 records `ti=40`
//     mesurables (8 films), un véhicule À L'ARRÊT porte un excès >= 89 bits dans 25,0 % des cas,
//     un véhicule EN MOUVEMENT dans 66,2 %. À mouvement contrôlé, l'occupation attestée ajoute
//     encore (80,0 % contre 58,1 %), mais la classe « sans épisode attesté » est contaminée
//     (ratio board:exit = 1:15). Le « signal d'occupation » du lot V5 est donc, pour l'essentiel,
//     un signal de MOUVEMENT.
//
// CE QUE CE FICHIER LIVRE, EN CONSÉQUENCE : une MESURE, pas un verdict. `ScanFilmVehicleOccupancy`
// rend, par image-clé et par véhicule, l'excès de longueur du record par rapport à la ligne de
// base du MÊME véhicule. `ExtraBlock` dit que cet excès atteint la taille d'un bloc — c'est un
// INDICE d'état supplémentaire, pas une affirmation d'occupation, et surtout pas une identité
// d'occupant. Un appelant qui en ferait « ce véhicule est occupé » se tromperait une fois sur
// quatre à l'arrêt et une fois sur trois en mouvement.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.

import "sort"

// VehicleKeyframeBlockBits est la taille MESURÉE du bloc supplémentaire, en bits : 89, la valeur
// la plus fréquente (10 blocs sur 18 attestés) et la seule qui se répète à l'identique d'un film
// à l'autre. Les autres tailles observées (52, 62, 89, 108, 269, 279, 348) ne sont pas des
// multiples exacts de 89 : le record ne croît PAS par quanta de 89 bits, et le dire est le
// résultat — c'est pourquoi ce seuil sert de PLANCHER (`ExcessBits >= VehicleKeyframeBlockBits`)
// et non d'unité de comptage.
const VehicleKeyframeBlockBits = 89

// keyframeBitAt rend le bit à la position `pos` du payload (MSB d'abord, comme kfReadBits).
func keyframeBitAt(pay []byte, pos int) bool { return kfReadBits(pay, pos, 1) == 1 }

// VehicleKeyframeState est l'état MESURÉ d'un véhicule à une image-clé.
type VehicleKeyframeState struct {
	// Slot identifie le véhicule ; TimestampUS, Chunk et PacketIndex localisent l'image-clé.
	Slot               int
	TimestampUS        uint64
	Chunk, PacketIndex int
	// LengthBits est l'emprise du record (KeyframeRecordSpan.LengthBits).
	LengthBits int
	// SlotGap est l'écart de slot au record suivant TROUVÉ. Voir KeyframeRecordSpan.SlotGap :
	// une emprise dont le voisin a été sauté couvre PLUSIEURS records, donc sa longueur ne veut
	// rien dire. `Measurable` en découle.
	SlotGap int
	// Measurable vaut vrai quand SlotGap == 1. Les états non mesurables sont RENDUS quand même
	// (l'appelant a le droit de savoir que le véhicule était là), mais BaselineBits, ExcessBits
	// et ExtraBlock valent zéro/faux sur eux.
	Measurable bool
	// BaselineBits est la plus courte emprise mesurable observée POUR CE VÉHICULE sur tout le
	// film — la ligne de base propre au châssis (un Wraith et une Mongoose n'ont pas le même
	// nombre de composants, comparer entre véhicules n'a aucun sens).
	BaselineBits int
	// ExcessBits = LengthBits - BaselineBits, donc toujours >= 0 sur un état mesurable.
	ExcessBits int
	// ExtraBlock : ExcessBits atteint VehicleKeyframeBlockBits. INDICE d'un état supplémentaire
	// sérialisé — cf. l'en-tête du fichier pour ce que cela ne prouve PAS.
	ExtraBlock bool
}

// VehicleKeyframeStates calcule les états véhicule à partir d'emprises de records déjà relevées.
// PUR (aucune I/O) — c'est la moitié testable de `ScanFilmVehicleOccupancy`.
//
// La ligne de base est établie PAR VÉHICULE sur les seules emprises mesurables ; un véhicule dont
// AUCUNE emprise n'est mesurable n'a pas de ligne de base et tous ses états restent à zéro.
func VehicleKeyframeStates(spans []KeyframeRecordSpan) []VehicleKeyframeState {
	base := map[int]int{}
	for _, s := range spans {
		if s.TI != VehicleTypeIndex || s.SlotGap != 1 {
			continue
		}
		if b, ok := base[s.Slot]; !ok || s.LengthBits < b {
			base[s.Slot] = s.LengthBits
		}
	}
	out := make([]VehicleKeyframeState, 0, len(spans))
	for _, s := range spans {
		if s.TI != VehicleTypeIndex {
			continue
		}
		st := VehicleKeyframeState{
			Slot: s.Slot, TimestampUS: s.TimestampUS, Chunk: s.Chunk, PacketIndex: s.PacketIndex,
			LengthBits: s.LengthBits, SlotGap: s.SlotGap, Measurable: s.SlotGap == 1,
		}
		if b, ok := base[s.Slot]; ok && st.Measurable {
			st.BaselineBits = b
			st.ExcessBits = s.LengthBits - b
			st.ExtraBlock = st.ExcessBits >= VehicleKeyframeBlockBits
		}
		out = append(out, st)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TimestampUS != out[j].TimestampUS {
			return out[i].TimestampUS < out[j].TimestampUS
		}
		return out[i].Slot < out[j].Slot
	})
	return out
}

// ScanFilmVehicleOccupancy rend l'état mesuré de chaque véhicule à chaque image-clé du film de
// `dir`, horodaté.
//
// CE N'EST PAS UN ORACLE D'OCCUPATION — lire l'en-tête du fichier. Il rend l'EXCÈS de longueur du
// record par rapport à la ligne de base du même véhicule ; l'identité de l'occupant et son siège
// n'ont été trouvés NULLE PART et le sont d'autant moins dans ce bloc (§ 2 de l'en-tête).
//
// HORS LIGNE — jamais depuis un chemin de requête.
func ScanFilmVehicleOccupancy(dir string) []VehicleKeyframeState {
	return VehicleKeyframeStates(ScanFilmKeyframeRecordSpans(dir))
}

// KeyframeRecordBits désigne UN record d'image-clé par son payload et ses bornes en bits.
type KeyframeRecordBits struct {
	Pay              []byte
	BitStart, BitEnd int
}

func (r KeyframeRecordBits) length() int { return r.BitEnd - r.BitStart }

// KeyframeBlockInsertion est la localisation, par maximum de vraisemblance, du bloc qui sépare
// deux records du MÊME véhicule.
type KeyframeBlockInsertion struct {
	// InsertBit est la position (relative au début du record LONG) qui maximise l'accord.
	InsertBit int
	// BlockBits est la taille du bloc supposé inséré (longueur du long - longueur du court).
	BlockBits int
	// Agree / Compared : bits en accord sous le meilleur modèle, sur le nombre de bits comparés.
	// `Compared` vaut la longueur du record COURT quel que soit `InsertBit`, donc deux modèles
	// sont directement comparables.
	Agree, Compared int
	// AgreeHead / AgreeTail sont les deux modèles DÉGÉNÉRÉS, et ils sont le témoin interne :
	// tout le décalage rejeté en tête (InsertBit = 0) et tout rejeté en queue (InsertBit = fin).
	// Si l'insertion n'explique rien, `Agree` ne les dépasse pas.
	AgreeHead, AgreeTail int
	// Valid est faux quand le record « long » n'est pas plus long que le court : il n'y a alors
	// aucune insertion à chercher, et tous les autres champs valent zéro.
	Valid bool
}

// FindKeyframeBlockInsertion cherche où le bloc s'insère entre un record LONG et un record COURT
// du même véhicule, sous le modèle « long = court[0:p] + BLOC(d) + court[p:] », d = Lo - Lf.
//
// Le score d'un `p` est le nombre de bits en accord ; il se calcule par sommes préfixes, donc en
// temps linéaire en la longueur du record. PUR (aucune I/O).
func FindKeyframeBlockInsertion(long, short KeyframeRecordBits) KeyframeBlockInsertion {
	lo, lf := long.length(), short.length()
	d := lo - lf
	if d <= 0 || lf <= 0 {
		return KeyframeBlockInsertion{}
	}
	// head[i] : accords des i premiers bits, alignés sur le DÉBUT des deux records.
	head := make([]int, lf+1)
	for i := 0; i < lf; i++ {
		head[i+1] = head[i]
		if keyframeBitAt(long.Pay, long.BitStart+i) == keyframeBitAt(short.Pay, short.BitStart+i) {
			head[i+1]++
		}
	}
	// tail[i] : accords des i derniers bits, alignés sur la FIN des deux records.
	tail := make([]int, lf+1)
	for i := 0; i < lf; i++ {
		tail[i+1] = tail[i]
		if keyframeBitAt(long.Pay, long.BitEnd-1-i) == keyframeBitAt(short.Pay, short.BitEnd-1-i) {
			tail[i+1]++
		}
	}
	best, bestP := -1, 0
	for p := 0; p <= lf; p++ {
		if s := head[p] + tail[lf-p]; s > best {
			best, bestP = s, p
		}
	}
	return KeyframeBlockInsertion{
		InsertBit: bestP, BlockBits: d, Agree: best, Compared: lf,
		AgreeHead: tail[lf], AgreeTail: head[lf], Valid: true,
	}
}
