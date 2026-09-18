package replay

// filmfacts_codec.go — LES SOUS-CODECS PARTAGES ENTRE SECTIONS : positions, pistes, objets du
// monde, images-cles, munitions.
//
// Extrait de golden_inputs_test.go le 2026-09-14 (revue R1, constat R1-7), passe en PRODUCTION
// le 2026-09-17 (lot 4.1.1-a). DEPLACEMENTS PURS. Le FLUX d octets lui-meme (`gwriter`,
// `greader`, varints, flottants, centimetre entier) vit dans `filmfacts_flux.go`.

import (
	"fmt"
	"math"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// encodePositionSection / decodePositionSection serialisent UNE suite de positions de bipede AVEC
// sa table de slots.
//
// UN SEUL CODEC POUR LES DEUX SUITES (lot 1.0, 2026-09-14) : les positions de BIPEDE et celles
// des VEHICULES (`VehicleScan.Positions`, lues a la meme grammaire sur la bande `ti=40`) ont la
// meme forme. Un second codec aurait diverge du premier au premier champ ajoute — c est
// exactement ce qui est arrive a la sequence de balayages elle-meme.
//
// LA TABLE DES SLOTS EST DANS LA SECTION, et non dans l en-tete du blob : un slot tient sur
// 13 bits mais un film n en emploie qu une centaine, et l indirection ramene 2 octets a 1 sur
// chaque position. Chaque suite a la sienne — celle des vehicules n est pas celle des bipedes.
func encodePositionSection(w *gwriter, pos []grammar.BipedPosition) {
	slotIdx := map[uint32]int{}
	var slots []uint32
	for _, p := range pos {
		if _, ok := slotIdx[p.Slot]; !ok {
			slotIdx[p.Slot] = len(slots)
			slots = append(slots, p.Slot)
		}
	}
	w.u(uint64(len(slots)))
	for _, s := range slots {
		w.u(uint64(s))
	}
	w.u(uint64(len(pos)))
	var lastTS uint64
	lastXYZ := map[uint32][3]int64{}
	for _, p := range pos {
		w.u(p.TimestampUS - lastTS) // horodatages non decroissants dans l ordre du film
		lastTS = p.TimestampUS
		w.u(uint64(slotIdx[p.Slot]))
		var fl byte
		if p.HasWorld {
			fl |= gpHasWorld
		}
		if p.HasYaw {
			fl |= gpHasYaw
		}
		if p.HasBody {
			fl |= gpHasBody
		}
		if p.HasShield {
			fl |= gpHasShield
		}
		w.byte8(fl)
		if p.HasWorld {
			// LES QUANTA DU FILM, PAS LES FLOTTANTS DERIVES (lot 0.D.3 bis). `X/Y/Z` sont
			// le resultat de `DequantBipedAxis(Q[ax], ax, layout, bornes)` : porter `Q` et
			// redequantifier a la relecture par LE MEME chemin rend la coordonnee a
			// l identique, sans coder un flottant. Et un quantum est un ENTIER qui bouge
			// peu d une position a la suivante : le delta signe tient sur un a deux octets,
			// la ou les bits d un float32 n en tenaient aucun.
			cur := [3]int64{int64(p.Q[0]), int64(p.Q[1]), int64(p.Q[2])}
			prev := lastXYZ[p.Slot]
			for a := 0; a < 3; a++ {
				w.i(cur[a] - prev[a])
			}
			lastXYZ[p.Slot] = cur
		}
		if p.HasYaw {
			// LES DEUX ANGLES D I21, ensemble : le cap et l elevation viennent du MEME
			// composant et partagent leur validite. En serialiser un seul rendrait un
			// fixture ou toutes les visees sont a plat.
			w.u(uint64(p.YawRaw))
			w.u(uint64(p.PitchRaw))
		}
		if p.HasBody {
			w.f32(p.Body.Health)
		}
		if p.HasShield {
			w.f32(p.Shield.Shield)
			w.byte8(p.Shield.Q) // le QUANTUM : la regle du surbouclier (q > 64) le lit, pas la valeur clampee
		}
	}
}

func decodePositionSection(r *greader, lay profile.I0Layout, world profile.Vec3Range) []grammar.BipedPosition {
	nSlots := r.compte(1)
	slots := make([]uint32, 0, nSlots)
	for k := 0; k < nSlots && r.err == nil; k++ {
		slots = append(slots, uint32(r.u()))
	}
	n := r.compte(3) // horodatage + index de slot + drapeaux, au minimum
	out := make([]grammar.BipedPosition, 0, n)
	var lastTS uint64
	lastXYZ := map[uint32][3]int64{}
	for k := 0; k < n && r.err == nil; k++ {
		var p grammar.BipedPosition
		lastTS += r.u()
		p.TimestampUS = lastTS
		si := int(r.u())
		if si >= len(slots) {
			r.err = fmt.Errorf("index de slot %d hors table (%d)", si, len(slots))
			return out
		}
		p.Slot = slots[si]
		fl := r.byte8()
		if fl&gpHasWorld != 0 {
			p.HasWorld = true
			prev := lastXYZ[p.Slot]
			var cur [3]int64
			for a := 0; a < 3; a++ {
				cur[a] = prev[a] + r.i()
			}
			lastXYZ[p.Slot] = cur
			p.Q = [3]uint32{uint32(cur[0]), uint32(cur[1]), uint32(cur[2])}
			p.X = grammar.DequantBipedAxis(p.Q[0], 0, lay, world)
			p.Y = grammar.DequantBipedAxis(p.Q[1], 1, lay, world)
			p.Z = grammar.DequantBipedAxis(p.Q[2], 2, lay, world)
		}
		if fl&gpHasYaw != 0 {
			p.HasYaw = true
			p.YawRaw = uint32(r.u())
			p.PitchRaw = uint32(r.u())
		}
		if fl&gpHasBody != 0 {
			p.HasBody = true
			p.Body.Health = r.f32()
		}
		if fl&gpHasShield != 0 {
			p.HasShield = true
			p.Shield.Shield = r.f32()
			p.Shield.Q = r.byte8()
		}
		out = append(out, p)
	}
	return out
}

// precisionDePiste dit COMMENT une suite de points se serialise, et la reponse n est pas la meme
// pour toutes les pistes du decodeur. C EST UNE MESURE, PAS UN GOUT (2026-09-18, lot 4.1.3) :
//
//	precisionCentimetre  la coordonnee est PUBLIEE arrondie au centieme (`round2`), donc le
//	                     centimetre entier est exactement la precision que la sortie porte, et
//	                     coder un float32 couterait 12 octets par point pour rien.
//	                     PROUVE pour les PROJECTILES : `projectiles.go:110` publie
//	                     `round2(p.X)`, `round2(p.Y)`.
//	precisionExacte      la coordonnee est PUBLIEE TELLE QUELLE. L arrondi est alors une PERTE.
//	                     PROUVE pour les PISTES D OBJETS DU MONDE : `o.Pos` sort des pistes
//	                     (`gwPickupResolve`) et `document_ground_weapon_items.go:217` publie
//	                     `X: o.Pos[0]` SANS `round2` — `ground_weapon_pads.go` n arrondit que
//	                     ses statistiques de cadence.
//
// # CE QUE L ABSENCE DE CE PARAMETRE A COUTE
//
// `encodeTracks` arrondissait TOUT au centimetre, sur la foi d un commentaire de `cmScale` qui
// affirmait « toute coordonnee publiee par l assemblage passe par round2 ». La mesure du
// 2026-09-18 (gate S8, diff structurel de deux artefacts du meme film) l a CONTREDIT :
// `groundWeapons[0]/x` valait `24.479221` au decodage et `24.48` au rejeu, sur les 10 films.
// Un artefact rejoue perdait donc ses positions d armes au sol a la decimale — silencieusement.
type precisionDePiste bool

const (
	// precisionCentimetre : centimetre entier, delta-varint. Pour ce que `round2` publie.
	precisionCentimetre precisionDePiste = false
	// precisionExacte : float32 tel quel. Pour ce qui est publie brut.
	precisionExacte precisionDePiste = true
)

func encodeTracks(w *gwriter, tracks []types.ProjectileTrack, prec precisionDePiste) {
	w.u(uint64(len(tracks)))
	for _, tr := range tracks {
		w.u(uint64(tr.Slot))
		w.u(uint64(tr.Gen))
		w.u(uint64(len(tr.Pts)))
		var pts uint64
		var prev [3]int64
		for _, s := range tr.Pts {
			w.u(s.TimestampUS - pts)
			pts = s.TimestampUS
			if prec == precisionExacte {
				w.f32(s.X)
				w.f32(s.Y)
				w.f32(s.Z)
			} else {
				cur := [3]int64{cmOf(s.X), cmOf(s.Y), cmOf(s.Z)}
				for a := 0; a < 3; a++ {
					w.i(cur[a] - prev[a])
				}
				prev = cur
			}
			w.bool8(s.AtRest)
			w.u(uint64(s.Chunk))
		}
	}
}

func decodeTracks(r *greader, prec precisionDePiste) []types.ProjectileTrack {
	n := r.compte(3)
	out := make([]types.ProjectileTrack, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		tr := types.ProjectileTrack{Slot: uint32(r.u()), Gen: uint32(r.u())}
		np := r.compte(2)
		var ts uint64
		var prev [3]int64
		for j := 0; j < np && r.err == nil; j++ {
			ts += r.u()
			var x, y, z float32
			if prec == precisionExacte {
				x, y, z = r.f32(), r.f32(), r.f32()
			} else {
				var cur [3]int64
				for a := 0; a < 3; a++ {
					cur[a] = prev[a] + r.i()
				}
				prev = cur
				x, y, z = fromCM(cur[0]), fromCM(cur[1]), fromCM(cur[2])
			}
			tr.Pts = append(tr.Pts, types.ProjectileSample{
				TimestampUS: ts, X: x, Y: y, Z: z,
				AtRest: r.bool8(), Chunk: int(r.u()),
			})
		}
		out = append(out, tr)
	}
	return out
}

// encodeWorldObjectScan / decodeWorldObjectScan serialisent ce que le film rend sur UN archetype
// d objet du monde : les records de CREATION (position i0, instant, identite MPP), le
// RECENSEMENT des images-cles qui borne les disparitions, et les pistes de position qui disent
// si l objet a bouge.
//
// UN SEUL CODEC POUR LES DEUX VOIES (armes `ti=42`, power-ups `ti=37`) : elles ont la meme
// forme, et un second codec aurait diverge du premier au premier champ ajoute.
//
// LA BANDE DE SLOTS N EST PAS SERIALISEE, et c est deliberé : l assemblage ne la lit pas (elle
// sert au seul balayage, qui a deja eu lieu). Le fixture porte ce que l assemblage CONSOMME,
// pas ce que le decodage a traverse.
func encodeWorldObjectScan(w *gwriter, s WorldObjectScan) {
	w.bool8(s.Scanned)
	encodeCreations(w, s.Creations)
	encodeCreationStats(w, s.Stats)
	encodeKeyframes(w, s.Keyframes)
	encodeTracks(w, s.Tracks, precisionExacte)
}

func decodeWorldObjectScan(r *greader) WorldObjectScan {
	s := WorldObjectScan{Scanned: r.bool8()}
	s.Creations = decodeCreations(r)
	s.Stats = decodeCreationStats(r)
	s.Keyframes = decodeKeyframes(r)
	s.Tracks = decodeTracks(r, precisionExacte)
	return s
}

// encodeCreations / decodeCreations serialisent les records de CREATION d un archetype d objet du
// monde : position i0, instant, identite MPP.
//
// SORTI DE `encodeWorldObjectScan` AU LOT 1.0 : les VEHICULES (`VehicleScan.Creations`) portent
// exactement les memes records, et un second codec aurait diverge du premier.
//
// L IDENTITE TIENT DANS `MPPWord32`, ET ELLE SEULE : c est le mot inconditionnel du bloc MPP —
// le GlobalID du tag `eqip` pour ti=37, l identite du chassis pour ti=40 (cf.
// filmdec/vehicle_creation.go). Les trois autres champs du bloc ne sont lus par aucun assemblage.
// encodeCreations / decodeCreations : UN RECORD DE CREATION, ENTIER.
//
// # CE QUE LA VERSION PRECEDENTE PERDAIT, ET CE QUE CA A COUTE (2026-09-18, lot 4.1.3)
//
// Elle portait SEPT champs sur vingt : l instant, la vie (slot, gen), la position, et UN SEUL des
// quatre mots MPP. Tombaient donc `HasAmmo` / `Ammo` (les MUNITIONS de l arme au sol, lues au
// composant i20), `HasRef` / `Ref`, `HasID` / `AbilityID`, `Mask` et ses trois temoins,
// `BitPos`, `Chunk`, `PacketIndex`, `DefaultStateBits`, `AfterBit`, et trois mots MPP sur quatre.
//
// LA CONSEQUENCE ETAIT MESUREE ET PUBLIEE : `document_ground_weapon_items.go:224` lit
// `o.HasAmmo` pour publier `groundWeapons[].ammo` et compter `coverage.groundWeaponItems.ammoRead`.
// Un artefact rejoue depuis les faits sortait donc SANS munitions d arme au sol, et son compteur
// de couverture avec — sur les dix films du gate S8, sans une ligne pour le dire.
//
// LA REGLE EST DESORMAIS SIMPLE, ET C EST UN TEST QUI LA TIENT : ce codec porte TOUT le record.
// `TestCodecCouvreFilmInputs` remplit chaque champ de chaque structure imbriquee (et non plus le
// premier seulement, le trou par lequel ce defaut est passe) : un champ ajoute a
// `types.EquipmentCreation` et non porte ici le fait rougir.
func encodeCreations(w *gwriter, creations []types.EquipmentCreation) {
	w.u(uint64(len(creations)))
	var lastTS uint64
	for _, c := range creations {
		w.u(c.TimestampUS - lastTS) // les creations sortent du balayage dans l ordre du film
		lastTS = c.TimestampUS
		w.u(uint64(c.Slot))
		w.u(uint64(c.Gen))
		w.i(int64(c.Chunk))
		w.i(int64(c.PacketIndex))
		w.i(int64(c.BitPos))
		w.bool8(c.HasRef)
		w.u(uint64(c.Ref))
		w.bool8(c.HasID)
		w.u(uint64(c.AbilityID))
		for i := 0; i < types.MPPFieldCount; i++ {
			w.bool8(c.MPPPresent[i])
			w.u(c.MPPVal[i])
		}
		w.f32(c.X)
		w.f32(c.Y)
		w.f32(c.Z)
		w.u(uint64(len(c.Mask)))
		for _, m := range c.Mask {
			w.i(int64(m))
		}
		w.bool8(c.MaskFull)
		w.bool8(c.MaskHasI0)
		w.i(int64(c.DefaultStateBits))
		w.bool8(c.HasAmmo)
		w.u(uint64(c.Ammo.Mag))
		w.u(uint64(c.Ammo.Res))
		w.i(int64(c.AfterBit))
	}
}

func decodeCreations(r *greader) []types.EquipmentCreation {
	n := r.compte(20) // vingt champs, un octet au minimum chacun
	out := make([]types.EquipmentCreation, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		c := types.EquipmentCreation{TimestampUS: lastTS}
		c.Slot, c.Gen = uint32(r.u()), uint32(r.u())
		c.Chunk, c.PacketIndex, c.BitPos = int(r.i()), int(r.i()), int(r.i())
		c.HasRef, c.Ref = r.bool8(), uint32(r.u())
		c.HasID, c.AbilityID = r.bool8(), uint32(r.u())
		for i := 0; i < types.MPPFieldCount; i++ {
			c.MPPPresent[i] = r.bool8()
			c.MPPVal[i] = r.u()
		}
		c.X, c.Y, c.Z = r.f32(), r.f32(), r.f32()
		if nm := int(r.u()); nm > 0 {
			c.Mask = make([]int, 0, nm)
			for j := 0; j < nm && r.err == nil; j++ {
				c.Mask = append(c.Mask, int(r.i()))
			}
		}
		c.MaskFull, c.MaskHasI0 = r.bool8(), r.bool8()
		c.DefaultStateBits = int(r.i())
		c.HasAmmo = r.bool8()
		c.Ammo.Mag, c.Ammo.Res = uint32(r.u()), uint32(r.u())
		c.AfterBit = int(r.i())
		out = append(out, c)
	}
	return out
}

// encodeCreationStats / decodeCreationStats : LES DENOMINATEURS DU BALAYAGE, ENTIERS.
//
// La version precedente n en portait que TROIS (`Slots`, `Anchors`, `Accepted`), inlines dans
// `encodeWorldObjectScan`. Les huit autres tombaient — dont `WithAmmo`, le denominateur meme de
// la lecture des munitions. Une couverture qui perd son denominateur ne se juge plus.
func encodeCreationStats(w *gwriter, st types.EquipmentCreationStats) {
	for _, v := range []int{
		st.Slots, st.Anchors, st.Overflow, st.MaskBad, st.PosBad, st.Accepted,
		st.MaskSparse, st.MaskFull, st.NoI0, st.WithRef, st.WithID, st.WithAmmo,
	} {
		w.i(int64(v))
	}
}

func decodeCreationStats(r *greader) types.EquipmentCreationStats {
	var st types.EquipmentCreationStats
	for _, p := range []*int{
		&st.Slots, &st.Anchors, &st.Overflow, &st.MaskBad, &st.PosBad, &st.Accepted,
		&st.MaskSparse, &st.MaskFull, &st.NoI0, &st.WithRef, &st.WithID, &st.WithAmmo,
	} {
		*p = int(r.i())
	}
	return st
}

func encodeKeyframes(w *gwriter, kf grammar.WorldObjectKeyframes) {
	w.u(uint64(len(kf.TimesUS)))
	var lastTS uint64
	for _, t := range kf.TimesUS {
		w.u(t - lastTS)
		lastTS = t
	}
	// L ORDRE DES CLES EST RENDU TOTAL : une map Go s itere au hasard, et un fixture dont les
	// octets changent a chaque regeneration n est plus un fixture.
	keys := make([]types.EquipmentLifeKey, 0, len(kf.SeenUS))
	for k := range kf.SeenUS {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Slot != keys[j].Slot {
			return keys[i].Slot < keys[j].Slot
		}
		return keys[i].Gen < keys[j].Gen
	})
	w.u(uint64(len(keys)))
	for _, k := range keys {
		w.u(uint64(k.Slot))
		w.u(uint64(k.Gen))
		seen := kf.SeenUS[k]
		w.u(uint64(len(seen)))
		lastTS = 0
		for _, t := range seen {
			w.u(t - lastTS)
			lastTS = t
		}
	}
}

func decodeKeyframes(r *greader) grammar.WorldObjectKeyframes {
	var kf grammar.WorldObjectKeyframes
	n := int(r.u())
	kf.TimesUS = make([]uint64, 0, n)
	var lastTS uint64
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		kf.TimesUS = append(kf.TimesUS, lastTS)
	}
	n = int(r.u())
	kf.SeenUS = make(map[types.EquipmentLifeKey][]uint64, n)
	for k := 0; k < n && r.err == nil; k++ {
		key := types.EquipmentLifeKey{Slot: uint32(r.u()), Gen: uint32(r.u())}
		np := r.compte(2)
		seen := make([]uint64, 0, np)
		lastTS = 0
		for j := 0; j < np && r.err == nil; j++ {
			lastTS += r.u()
			seen = append(seen, lastTS)
		}
		kf.SeenUS[key] = seen
	}
	return kf
}

// encodeAmmo serialise un emplacement de munitions. LES TROIS CAS SONT DISTINCTS (chargeur,
// jauge, rien) : un drapeau par pointeur, jamais un zero qui vaudrait absence.
func encodeAmmo(w *gwriter, a SlotAmmo) {
	w.bool8(a.Mag != nil)
	if a.Mag != nil {
		w.u(uint64(*a.Mag))
	}
	w.bool8(a.Res != nil)
	if a.Res != nil {
		w.u(uint64(*a.Res))
	}
	w.bool8(a.Gauge != nil)
	if a.Gauge != nil {
		w.b = ajouterPoidsFaibleDAbord(w.b, math.Float64bits(*a.Gauge), 8)
	}
	w.u(uint64(a.Overheat))
	w.u(uint64(a.Flags))
}

func decodeAmmo(r *greader) SlotAmmo {
	var a SlotAmmo
	if r.bool8() {
		v := uint32(r.u())
		a.Mag = &v
	}
	if r.bool8() {
		v := uint32(r.u())
		a.Res = &v
	}
	if r.bool8() {
		if r.off+8 > len(r.b) {
			r.err = fmt.Errorf("jauge tronquee a l offset %d", r.off)
			return a
		}
		v := math.Float64frombits(lirePoidsFaibleDAbord(r.b[r.off:], 8))
		r.off += 8
		a.Gauge = &v
	}
	a.Overheat = uint32(r.u())
	a.Flags = uint32(r.u())
	return a
}
