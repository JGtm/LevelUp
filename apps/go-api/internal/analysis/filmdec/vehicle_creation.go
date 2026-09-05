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

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmsource"
)

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

// runCreationWalk deroule les paquets DELTA du film et decode les records de creation avec le
// walk donne (deja construit : archetype, deser, gate de position, tampon de sondes).
//
// LE POINT DE PASSAGE UNIQUE des trois archetypes de creation — equipement `ti=37`, arme au sol
// `ti=42`, vehicule `ti=40`. Chacun garde ses REFUS en propre (bornes, chunks, bande, archetype) ;
// aucun ne recopie la marche. Le vehicule etant le troisieme, la boucle a ete factorisee ici
// plutot que recopiee (regle des 2 copies du depot, 2026-09-05).
func runCreationWalk(
	fc *FilmContext, w equipCreationWalk, st *EquipmentCreationStats,
) []EquipmentCreation {
	var out []EquipmentCreation
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			out = append(out, w.scanPayload(pk.Payload(data), st, pk, c)...)
		}
	}
	return out
}

// ScanFilmVehicleCreations est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanVehicleCreations].
func ScanFilmVehicleCreations(
	dir string, wr *Vec3Range,
) ([]EquipmentCreation, EquipmentCreationStats, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, EquipmentCreationStats{}, err
	}
	return ScanVehicleCreations(NewFilmContext(film), wr)
}

// ScanVehicleCreations decode les records de creation des VEHICULES d'un film DEJA CHARGE, sur la
// bande de slots de `ti=40` lue dans les images-cles, avec le default-state porte
// (consumeDefaultStateTI40) et le gate i0 dyn.-prec.
func ScanVehicleCreations(
	fc *FilmContext, wr *Vec3Range,
) ([]EquipmentCreation, EquipmentCreationStats, error) {
	var st EquipmentCreationStats
	if len(fc.ChunkNumbers()) == 0 {
		return nil, st, ErrNoFilmChunk
	}
	band := worldObjectSlotBand(fc.Film(), VehicleTypeIndex)
	if len(band) == 0 {
		return nil, st, fmt.Errorf("aucun slot d'archetype ti=%d dans les keyframes du film",
			VehicleTypeIndex)
	}
	return ScanVehicleCreationsForBand(fc, wr, band)
}

// ScanVehicleCreationsForBand balaye une BANDE DE SLOTS donnee avec le gate i0 dyn.-prec.
// (porte 5 bits + rejet des quanta satures). Sans oracle de nuage, ce gate garde un plancher de
// faux positifs — l'instrument de mesure durcit le gate par le nuage des positions reelles.
//
// LE DECOUPAGE d'i0 vient du CONTEXTE, comme pour le nuage de positions de la meme bande : le
// catalogue l'impose quand l'entree de carte est valide, l'auto-detection ne sert que de repli
// (lot 3 de PLAN_CUISSON_PERF). Deux decoupages differents entre la creation et la trajectoire
// d'un meme vehicule rendraient un gate incoherent avec le nuage qu'il est cense qualifier.
func ScanVehicleCreationsForBand(
	fc *FilmContext, wr *Vec3Range, band map[uint32]bool,
) ([]EquipmentCreation, EquipmentCreationStats, error) {
	var st EquipmentCreationStats
	if wr == nil {
		return nil, st, fmt.Errorf("bornes absentes : sans elles le decodeur ne rend que des quanta")
	}
	if len(fc.ChunkNumbers()) == 0 {
		return nil, st, ErrNoFilmChunk
	}
	st.Slots = len(band)
	lay, err := fc.I0Layout()
	if err != nil {
		return nil, st, fmt.Errorf("decoupage i0 illisible : %w", err)
	}
	arch, err := fc.vehicleArchetype()
	if err != nil {
		return nil, st, err
	}
	var cur equipCreationRead
	defer installCreationHooks(&cur)()
	w := equipCreationWalk{
		comps: len(arch.Components), wr: wr, band: band, cur: &cur,
		ti: VehicleTypeIndex, deser: consumeDefaultStateTI40,
		posDecode: func(pay []byte, at int) ([3]float32, bool) {
			return decodeBipedI0Pos(pay, at, lay, wr)
		},
		posBits: lay.TotalBits(),
	}
	return runCreationWalk(fc, w, &st), st, nil
}

// vehicleArchetype rend l'archetype `ti=40` du registre du film (chunk_00), ANALYSE UNE FOIS par
// le contexte.
func (c *FilmContext) vehicleArchetype() (Archetype, error) {
	arch, _, ok, err := c.archetype(VehicleTypeIndex)
	if err != nil {
		return Archetype{}, err
	}
	if !ok {
		return Archetype{}, fmt.Errorf("archetype vehicule %d absent du registre", VehicleTypeIndex)
	}
	return arch, nil
}
