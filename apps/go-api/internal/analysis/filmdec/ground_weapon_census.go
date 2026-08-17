package filmdec

import "sort"

// ground_weapon_census.go — LE RECENSEMENT des ARMES AU SOL aux images-clés, et la BANDE de
// slots de leur archétype, rendus par UNE SEULE marche des images-clés.
//
// À QUOI SERT LE RECENSEMENT. Le film ne porte AUCUN événement de ramassage (mesure du
// 2026-08-12) et le record DEL n'est pas isolable (78 090 candidats pour 477 vies, correctif de
// revue des poses du 2026-08-18). La disparition d'une arme au sol se BORNE donc : tant qu'un
// record `ti=42` de la vie (slot, gen) est recensé à une image-clé, l'objet est là ; la dernière
// image-clé qui le recense et la première qui ne le recense plus encadrent sa disparition. Le
// walker durci recense 249 entités sur 250 — c'est lui qui prouve la survie, pas le balayage de
// familles, qui n'attrape que les records porteurs d'une famille connue et manquerait les objets
// muets.
//
// CE QUE LE RECENSEMENT NE FAIT PAS : dater. Les images-clés sont espacées de ~20 s (mesuré :
// médiane d'intervalle 20,00 s, p90 20,00 à 20,02 s sur huit films). Il BORNE, il ne DATE pas.
//
// POURQUOI BANDE ET RECENSEMENT SORTENT ENSEMBLE. Les deux se lisent dans la MÊME marche : les
// records `ti=42` des images-clés donnent la bande de slots (`worldObjectSlotBand`) et, à la
// paire (slot, gen) près, les instants de recensement. Les séparer coûtait une seconde lecture
// complète du film pour la même donnée — trois marches d'images-clés au lieu d'une sur le chemin
// des armes au sol (bande, recensement, puis celle de `ScanFilmWorldObjects`).
//
// HORS LIGNE par construction (I/O disque sur tout le film) — jamais depuis un chemin de requête.

// GroundWeaponKeyframes porte ce qu'une marche des images-clés dit des armes au sol.
type GroundWeaponKeyframes struct {
	// Band est la bande de slots utilisable pour l'archétype `ti=42` — même règle que
	// `GroundWeaponSlotBand` (bande comblée, privée des slots vus porter un autre archétype).
	Band map[uint32]bool
	// TimesUS porte les instants de TOUTES les images-clés du film, triés. C'est l'axe sur
	// lequel le bornage se fait : la première image-clé APRÈS le dernier recensement borne la
	// disparition.
	TimesUS []uint64
	// SeenUS porte, par vie d'objet (slot, gen), les instants des images-clés qui la RECENSENT,
	// triés. Une vie absente n'est PAS une vie inexistante : elle a pu naître et disparaître
	// entre deux images-clés (24,0 % des apparitions mesurées sur huit films).
	SeenUS map[EquipmentLifeKey][]uint64
}

// LastTimeUS rend l'instant de la DERNIÈRE image-clé du film, ou zéro s'il n'y en a aucune.
func (k GroundWeaponKeyframes) LastTimeUS() uint64 {
	if len(k.TimesUS) == 0 {
		return 0
	}
	return k.TimesUS[len(k.TimesUS)-1]
}

// ScanFilmGroundWeaponKeyframes marche les images-clés du film une seule fois et rend la bande
// de slots de l'archétype `ti=42` avec le recensement des vies qui y figurent.
//
// HORS LIGNE (I/O disque sur tout le film).
func ScanFilmGroundWeaponKeyframes(dir string) GroundWeaponKeyframes {
	out := GroundWeaponKeyframes{SeenUS: map[EquipmentLifeKey][]uint64{}}
	seen, others := map[uint32]bool{}, map[uint32]bool{}
	n := CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			out.TimesUS = append(out.TimesUS, pk.TimestampUS)
			out.censusPacket(WalkKeyframeWorld(pk.Payload(data)), pk.TimestampUS, seen, others)
		}
	}
	out.Band = slotBandExcluding(seen, others)
	sort.Slice(out.TimesUS, func(i, j int) bool { return out.TimesUS[i] < out.TimesUS[j] })
	for k := range out.SeenUS {
		v := out.SeenUS[k]
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	}
	return out
}

// censusPacket range les records d'UNE image-clé : les `ti=42` alimentent le recensement et la
// bande, les autres archétypes la liste d'exclusion (règle de `worldObjectSlotBand` — un slot vu
// porter autre chose ne peut pas être comblé dans la bande).
func (k *GroundWeaponKeyframes) censusPacket(
	recs []KeyframeRec, atUS uint64, seen, others map[uint32]bool,
) {
	for _, r := range recs {
		slot := uint32(r.Slot)
		if r.TI != GroundWeaponTypeIndex {
			others[slot] = true
			continue
		}
		seen[slot] = true
		key := EquipmentLifeKey{Slot: slot, Gen: uint32(r.Gen)}
		// UN RECORD PAR VIE ET PAR IMAGE-CLÉ : le même objet peut être répliqué deux fois
		// dans un même paquet, et compter deux fois le même instant fausserait le bornage.
		if v := k.SeenUS[key]; len(v) > 0 && v[len(v)-1] == atUS {
			continue
		}
		k.SeenUS[key] = append(k.SeenUS[key], atUS)
	}
}
