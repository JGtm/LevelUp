package filmdec

// vehicle_creation.go — LE RECORD DE CREATION d'un VEHICULE (ti=40).
//
// CE QUE CETTE VOIE OUVRE. Le record de CREATION porte son `R(6) typeIndex` (il DIT `ti=40`) et,
// dans son default-state, le bloc `object-multiplayer-properties` dont le mot de 32 bits
// inconditionnel `MPPWord32` est l'IDENTITE du chassis (hypothese tag `vehi`, cadrage § 3.1). Ce
// mot est lu AVANT toute position. Le default-state est porte : consumeDefaultStateTI40
// (default_state_ti40.go).
//
// LA MARCHE EST CELLE DE L'EQUIPEMENT / DE L'ARME AU SOL, parametree par l'archetype
// (`equipCreationWalk`) : meme en-tete NEW, meme porte, meme masque. Aucune seconde copie de la
// marche n'est ecrite (regle des 2 copies).
//
// LE GATE DE SELECTIVITE EST i0 EN PRECISION-DYNAMIQUE. i0 de `ti=40` est
// `object-position-dynamic-precision-component` (porte 5 bits, grammaire biped) — PAS objet du
// monde (porte 3 bits). On passe donc au walk le decodeur i0 dyn.-prec. (decodeBipedI0Pos),
// construit sur le decoupage lu DANS le film (DetectI0Layout) et les primitives livrees par V1a
// (offline_biped.go : la porte biped, DequantBipedAxis, saturatedQuantum). Ce gate valide le cadre
// du record : un default-state de mauvaise largeur fait tomber i0 hors de la porte, et le record
// est rejete. L'INSTRUMENT DE MESURE (vehicle_creation_test.go) durcit ce gate en n'acceptant que
// les i0 qui coincident avec une position REELLE de vehicule (nuage decode par
// ScanFilmBipedPositionsForBand) — c'est l'oracle qui a valide ti=42, transpose en dyn.-prec.
//
// POURQUOI ti=40 N'EST PAS DANS defaultStateDeserByTI. Le default-state de ti=40 a une feuille
// config-dependante (le quaternion, dossier RE § 3 FINDING A) : par la regle de
// default_state_arch.go:30-32 il n'est pas inscrit dans la table. On passe donc `deser` ICI
// explicitement, ce que `equipCreationWalk` permet deja.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requete.

import "fmt"

// decodeBipedI0Pos VALIDE et decode le composant i0 en PRECISION-DYNAMIQUE (grammaire biped, porte
// 5 bits) a l'offset bit `at`. C'est l'equivalent de decodeWorldObjectPos (porte 3 bits) pour les
// deux archetypes qui portent cette forme (biped ti=35, vehicule ti=40). Rejette : porte non nulle
// (spine+useDefault), region inattendue, ou axe sature (quantum de garde). Chaque brique est celle
// du decodeur biped de V1a (matchBipedHeader, saturatedQuantum, DequantBipedAxis).
func decodeBipedI0Pos(pay []byte, at int, lay I0Layout, wr *Vec3Range) ([3]float32, bool) {
	const preGate = i0SpineBits + i0UseDefaultBits // 4 bits nuls = chemin absolu
	total := len(pay) * 8
	if at < 0 || at+lay.TotalBits() > total {
		return [3]float32{}, false
	}
	if readBitsAt(pay, at, preGate) != 0 {
		return [3]float32{}, false
	}
	if readBitsAt(pay, at+preGate, lay.GateBits-preGate) != lay.Region {
		return [3]float32{}, false
	}
	var q [3]uint32
	for ax := 0; ax < 3; ax++ {
		q[ax] = readBitsAt(pay, at+lay.AxisOffset(ax), int(lay.AxisW[ax]))
	}
	if saturatedQuantum(q, lay) {
		return [3]float32{}, false
	}
	var v [3]float32
	for ax := 0; ax < 3; ax++ {
		v[ax] = DequantBipedAxis(q[ax], ax, lay, *wr)
	}
	return v, true
}

// vehicleI0Layout rend le decoupage d'i0 lu DANS le film (DetectI0Layout) : les largeurs d'axe sont
// une constante de CARTE, jamais d'archetype, et le gate dyn.-prec. en depend.
func vehicleI0Layout(dir string) (I0Layout, error) {
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		return I0Layout{}, fmt.Errorf("decoupage i0 illisible dans %s : %w", dir, err)
	}
	return lay, nil
}

// runCreationWalk balaye les paquets delta du film et decode les records de creation avec le walk
// donne (deja construit : archetype, deser, gate de position). Le point de passage unique — la
// fonction publique et l'instrument de mesure y injectent leur propre walk.
//
// UN SEUL DECODAGE filmdec A LA FOIS PAR PROCESS : ce balayage installe les sondes de creation et
// du bloc MPP (globaux de paquet), restaurees a la sortie.
func runCreationWalk(dir string, w equipCreationWalk) ([]EquipmentCreation, EquipmentCreationStats, error) {
	st := EquipmentCreationStats{Slots: len(w.band)}
	if w.wr == nil {
		return nil, st, fmt.Errorf("bornes absentes : sans elles le decodeur ne rend que des quanta")
	}
	n := CountFilmChunks(dir)
	if n == 0 {
		return nil, st, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	defer installCreationHooks(w.cur)()
	var out []EquipmentCreation
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			out = append(out, w.scanPayload(pk.Payload(data), &st, pk, c)...)
		}
	}
	return out, st, nil
}

// ScanFilmVehicleCreations decode les records de creation des VEHICULES du film de dir, sur la bande
// de slots de `ti=40` lue dans les images-cles, avec le default-state porte (consumeDefaultStateTI40)
// et le gate i0 dyn.-prec.
func ScanFilmVehicleCreations(
	dir string, wr *Vec3Range,
) ([]EquipmentCreation, EquipmentCreationStats, error) {
	var st EquipmentCreationStats
	n := CountFilmChunks(dir)
	if n == 0 {
		return nil, st, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	band := worldObjectSlotBand(dir, n, VehicleTypeIndex)
	if len(band) == 0 {
		return nil, st, fmt.Errorf("aucun slot d'archetype ti=%d dans les keyframes de %s",
			VehicleTypeIndex, dir)
	}
	return ScanFilmVehicleCreationsForBand(dir, wr, band)
}

// ScanFilmVehicleCreationsForBand balaye une BANDE DE SLOTS donnee avec le gate i0 dyn.-prec.
// (porte 5 bits + rejet des quanta satures). Sans oracle de nuage, ce gate garde un plancher de
// faux positifs — l'instrument de mesure durcit le gate par le nuage des positions reelles.
func ScanFilmVehicleCreationsForBand(
	dir string, wr *Vec3Range, band map[uint32]bool,
) ([]EquipmentCreation, EquipmentCreationStats, error) {
	lay, err := vehicleI0Layout(dir)
	if err != nil {
		return nil, EquipmentCreationStats{}, err
	}
	arch, err := vehicleArchetype(dir)
	if err != nil {
		return nil, EquipmentCreationStats{}, err
	}
	var cur equipCreationRead
	w := equipCreationWalk{
		comps: len(arch.Components), wr: wr, band: band, cur: &cur,
		ti: VehicleTypeIndex, deser: consumeDefaultStateTI40,
		posDecode: func(pay []byte, at int) ([3]float32, bool) {
			return decodeBipedI0Pos(pay, at, lay, wr)
		},
		posBits: lay.TotalBits(),
	}
	return runCreationWalk(dir, w)
}

// vehicleArchetype rend l'archetype `ti=40` du registre du film (chunk_00).
func vehicleArchetype(dir string) (Archetype, error) {
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		return Archetype{}, fmt.Errorf("chunk_00 (registre) illisible dans %s : %w", dir, err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		return Archetype{}, fmt.Errorf("registre illisible dans %s : %w", dir, err)
	}
	arch, ok := reg.Archetype(VehicleTypeIndex)
	if !ok {
		return Archetype{}, fmt.Errorf("archetype vehicule %d absent du registre de %s",
			VehicleTypeIndex, dir)
	}
	return arch, nil
}
