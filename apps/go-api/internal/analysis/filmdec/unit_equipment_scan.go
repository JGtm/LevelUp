package filmdec

// unit_equipment_scan.go — LE BALAYAGE d'i26 `unit-equipment-component` : la liste de
// références que le porteur émet sur son équipement.
//
// CE QUE LA MESURE DU 2026-08-30 A ÉTABLI (equipment_i26_research_test.go), et qui fonde
// l'export : à 70,2 % des prises d'équipement (i48), la liste i26 du même slot gagne une
// entrée NOUVELLE dans la seconde (témoin décalé de 30 s : 0 %). Chaque entrée est un
// optionnel `porte(1) + valeur(13) + queue(2)` — les largeurs d'un slot d'entité et d'une
// génération — et les valeurs tombent dans la zone des objets du monde (0 % dans la bande
// bipède). C'est le canal côté PORTEUR qui référence l'objet, celui que la proximité ne sait
// pas donner (l'équipement tombe en tas avec les grenades, mesure D).
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmsource"
)

// UnitEquipmentEmission est UNE lecture d'i26 rattachée à son record bipède.
type UnitEquipmentEmission struct {
	// Slot est le slot du bipède émetteur — une VIE, pas un joueur.
	Slot uint32
	// TimestampUS est l'horodatage du paquet — même horloge que BipedPosition.
	TimestampUS uint64
	// Read est la lecture publiée par le déserialiseur (en-tête + liste).
	Read UnitEquipmentRead
}

// ScanFilmUnitEquipment décode toutes les émissions d'i26 des paquets delta du film de dir.
//
// UN SEUL DÉCODAGE filmdec À LA FOIS PAR PROCESS : ce balayage installe `unitEquipmentHook`,
// un global de paquet. L'appelant doit détenir LockProcessDecode ; le hook est restauré à la
// sortie, y compris en cas d'erreur.
//
// ScanFilmUnitEquipment est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanUnitEquipment].
func ScanFilmUnitEquipment(dir string) ([]UnitEquipmentEmission, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, err
	}
	return ScanUnitEquipment(film)
}

// ScanUnitEquipment décode les émissions d'i26 d'un film DEJA CHARGE.
func ScanUnitEquipment(film *filmsource.Film) ([]UnitEquipmentEmission, error) {
	chunks := FilmChunkNumbers(film)
	if len(chunks) == 0 {
		return nil, ErrNoFilmChunk
	}
	slots := bipedSlotBand(film, chunks)
	if len(slots) == 0 {
		return nil, fmt.Errorf("aucun slot biped (ti=%d) dans les keyframes du film", BipedTypeIndex)
	}
	lay, _, err := DetectI0LayoutOf(film)
	if err != nil {
		return nil, fmt.Errorf("découpage i0 illisible : %w", err)
	}
	arch, err := bipedArchetype(film)
	if err != nil {
		return nil, err
	}
	idx26 := -1
	for id := 0; id < archetypeBlockSlots; id++ {
		if arch.component(id) == "unit-equipment-component" {
			idx26 = id
			break
		}
	}
	if idx26 < 0 {
		return nil, fmt.Errorf("aucun unit-equipment-component dans l'archétype biped du film")
	}

	var last struct {
		read UnitEquipmentRead
		got  bool
	}
	prev := unitEquipmentHook
	SetUnitEquipmentHook(func(r UnitEquipmentRead) { last.read, last.got = r, true })
	defer SetUnitEquipmentHook(prev)

	var out []UnitEquipmentEmission
	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + lay.TotalBits()
	for _, c := range chunks {
		data, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			for p := 0; p+minRecord <= total; {
				i0, slot, idx, ok := matchBipedHeader(pay, p, total, slots, true, lay)
				if !ok {
					p++
					continue
				}
				if maskHas(idx, idx26) {
					last.got = false
					if walkRecordTo(pay, i0, total, idx, lay, arch, idx26) && last.got {
						out = append(out, UnitEquipmentEmission{
							Slot: slot, TimestampUS: pk.TimestampUS, Read: last.read,
						})
					}
					last.got = false
				}
				p = i0 + lay.TotalBits()
			}
		}
	}
	return out, nil
}
