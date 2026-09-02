package filmdec

// ground_weapon_creation.go — LE RECORD DE CRÉATION d'une ARME AU SOL (ti=42).
//
// CE QUE ÇA DÉBLOQUE, ET POURQUOI ÇA N'EXISTAIT PAS. La POSITION des armes au sol a été
// RÉFUTÉE le 2026-08-12 : la seule voie essayée était la bande de slots des paquets delta, et
// un record delta ne DIT PAS son archétype — la bande comblée est contaminée par les archétypes
// voisins, 62,4 % des slots s'étalaient au-delà de 20 u. Le record de CRÉATION, lui, porte son
// `R(6) typeIndex` : il dit `ti=42`, et il porte la position i0 de l'objet à sa naissance. Il
// était illisible tant que le default-state de l'archétype ne l'était pas — c'est la cause (1)
// de la réfutation, aujourd'hui levée (default_state_ti42.go, validé par oracle de position :
// 118/119 et 97/99 atterrissages exacts contre 0 pour trois déserialiseurs faux).
//
// LA MARCHE EST CELLE DE L'ÉQUIPEMENT, paramétrée par l'archétype (`equipCreationWalk`) : même
// en-tête NEW, même porte, même masque, même i0. Seuls changent le `typeIndex` exigé et le
// déserialiseur du default-state. Aucune seconde copie de la marche n'est écrite.
//
// L'IDENTITÉ VOYAGE DANS `MPPVal[MPPWord32]`, comme pour `ti=37` où ce mot est le GlobalID du
// tag `eqip`. Pour `ti=42` l'hypothèse est le tag `weap` : elle se MESURE en croisant ce mot
// avec la famille high-32 lue aux images-clés (`ScanFilmKeyframeGroundWeapons`), deux chaînes
// indépendantes — elle ne se suppose pas.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmsource"
)

// ScanFilmGroundWeaponCreations décode les records de création des ARMES AU SOL du film de dir,
// sur la bande de slots de `ti=42` lue dans les images-clés.
//
// UN SEUL DÉCODAGE filmdec À LA FOIS PAR PROCESS : ce balayage installe les sondes de création
// et du bloc MPP, qui sont des globaux de paquet. Elles sont restaurées à la sortie.
//
// ScanFilmGroundWeaponCreations est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanGroundWeaponCreations].
func ScanFilmGroundWeaponCreations(
	dir string, wr *Vec3Range,
) ([]EquipmentCreation, EquipmentCreationStats, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, EquipmentCreationStats{}, err
	}
	return ScanGroundWeaponCreations(NewFilmContext(film), wr)
}

// ScanGroundWeaponCreations décode les records de création des armes au sol d'un film DEJA CHARGE.
func ScanGroundWeaponCreations(
	fc *FilmContext, wr *Vec3Range,
) ([]EquipmentCreation, EquipmentCreationStats, error) {
	var st EquipmentCreationStats
	if len(fc.ChunkNumbers()) == 0 {
		return nil, st, ErrNoFilmChunk
	}
	band := worldObjectSlotBand(fc.Film(), GroundWeaponTypeIndex)
	if len(band) == 0 {
		return nil, st, fmt.Errorf("aucun slot d'archétype ti=%d dans les keyframes du film",
			GroundWeaponTypeIndex)
	}
	return ScanGroundWeaponCreationsForBand(fc, wr, band)
}

// ScanFilmGroundWeaponCreationsForBand balaye une BANDE DE SLOTS donnée. La bande est un
// paramètre pour que le témoin de contrôle (une bande FANTÔME de même cardinalité, faite de
// slots jamais vus porter cet archétype) passe par le MÊME code que la mesure — sans quoi le
// contrôle ne contrôlerait pas le décodeur mais une variante de lui.
//
// ScanFilmGroundWeaponCreationsForBand est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanGroundWeaponCreationsForBand].
func ScanFilmGroundWeaponCreationsForBand(
	dir string, wr *Vec3Range, band map[uint32]bool,
) ([]EquipmentCreation, EquipmentCreationStats, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, EquipmentCreationStats{}, err
	}
	return ScanGroundWeaponCreationsForBand(NewFilmContext(film), wr, band)
}

// ScanGroundWeaponCreationsForBand balaye une bande de slots donnée dans un film DEJA CHARGE.
func ScanGroundWeaponCreationsForBand(
	fc *FilmContext, wr *Vec3Range, band map[uint32]bool,
) ([]EquipmentCreation, EquipmentCreationStats, error) {
	var st EquipmentCreationStats
	if wr == nil {
		return nil, st, fmt.Errorf("bornes monde absentes : sans elles le décodeur ne rend que des quanta")
	}
	nums := fc.ChunkNumbers()
	if len(nums) == 0 {
		return nil, st, ErrNoFilmChunk
	}
	st.Slots = len(band)
	arch, err := fc.groundWeaponArchetype()
	if err != nil {
		return nil, st, err
	}

	var cur equipCreationRead
	defer installCreationHooks(&cur)()

	w := equipCreationWalk{
		comps: len(arch.Components), wr: wr, band: band, cur: &cur,
		ti: GroundWeaponTypeIndex, deser: consumeDefaultStateTI42,
	}
	var out []EquipmentCreation
	for _, c := range nums {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			out = append(out, w.scanPayload(pk.Payload(data), &st, pk, c)...)
		}
	}
	return out, st, nil
}

// groundWeaponArchetype rend l'archétype `ti=42` du registre du film (chunk_00), ANALYSE UNE
// FOIS par le contexte (lot 2, 2026-09-03).
func (c *FilmContext) groundWeaponArchetype() (Archetype, error) {
	arch, _, ok, err := c.archetype(GroundWeaponTypeIndex)
	if err != nil {
		return Archetype{}, err
	}
	if !ok {
		return Archetype{}, fmt.Errorf("archétype arme au sol %d absent du registre",
			GroundWeaponTypeIndex)
	}
	return arch, nil
}
