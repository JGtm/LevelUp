package replay

// golden_inputs_decode_test.go — LE DECODEUR DU FIXTURE D ENTREES.
//
// Extrait de golden_inputs_test.go le 2026-09-14 (revue R1, constat R1-7). DEPLACEMENT PUR.

import (
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

func decodeGoldenInputs(blob []byte, entry filmdec.MapQuantEntry) (*goldenInputs, error) {
	g, r, lay, world, err := decodeGoldenEntete(blob, entry)
	if err != nil {
		return nil, err
	}
	g.Positions = decodePositionSection(r, lay, world)
	g.BipedCreations = decodeBipedCreations(r)
	decodeGoldenEvenements(r, g)
	g.WeaponChanges = decodeWeaponChanges(r)
	g.Pickups, g.PickupStats = decodePickups(r)
	decodeGoldenInventaire(r, g)
	decodeGoldenCanauxDelta(r, g)
	g.EquipmentChanges, g.EquipmentChangeStats = decodeEquipmentChanges(r)
	decodeGoldenCapacites(r, g)
	g.ZoomEvents = decodeZoomEvents(r)
	decodeGoldenMonde(r, g)
	g.Vehicles = decodeVehicleScan(r, lay, world)
	decodeGoldenQueue(r, g)
	if r.err != nil {
		return nil, r.err
	}
	if r.off != len(r.b) {
		return nil, fmt.Errorf("fixture d entrees : %d octet(s) non consomme(s) — format desynchronise",
			len(r.b)-r.off)
	}
	return g, nil
}

// decodeGoldenEntete relit l en-tete, verifie carte et decoupage, et rend le lecteur arme.
func decodeGoldenEntete(blob []byte, entry filmdec.MapQuantEntry) (
	*goldenInputs, *greader, filmdec.I0Layout, filmdec.Vec3Range, error,
) {
	if len(blob) < len(goldenInputsMagic) || string(blob[:len(goldenInputsMagic)]) != goldenInputsMagic {
		return nil, nil, filmdec.I0Layout{}, filmdec.Vec3Range{}, fmt.Errorf("fixture d entrees : magie absente ou version inconnue — regenerer")
	}
	r := &greader{b: blob, off: len(goldenInputsMagic)}
	g := &goldenInputs{Film: r.str()}
	g.MapModule = r.str()
	if g.MapModule != entry.Module {
		return nil, nil, filmdec.I0Layout{}, filmdec.Vec3Range{}, fmt.Errorf("%w : fixture cuit pour %q, entree de catalogue fournie %q",
			errGoldenInputsCarte, g.MapModule, entry.Module)
	}
	for a := 0; a < 3; a++ {
		g.AxisW[a] = uint(r.u())
	}
	g.LayoutDetected = r.bool8()
	g.InventoryDeltaAmmoRefused = r.bool8()
	if r.bool8() {
		v := int(r.i())
		g.FilmMajorVersion = &v
	}
	// CONTRADICTION BLOB / CATALOGUE = ERREUR TYPEE. Quand le fixture dit tenir son decoupage
	// du CATALOGUE, il doit etre celui que la regle de production tranche aujourd hui : sinon le
	// catalogue a bouge sous le fixture, et les quanta se dequantifieraient avec un autre pas.
	// LE BLOB ET LE CATALOGUE DOIVENT DIRE LA MEME CHOSE, DANS LES DEUX SENS (revue R1,
	// constat R1-2). Un fixture qui se dit « catalogue » doit porter le decoupage que la regle
	// tranche aujourd hui ; un fixture qui se dit « detecte » doit venir d une carte dont
	// l entree est INVALIDE — sinon il a ete cuit hors de la regle, et ses quanta se
	// dequantifieraient avec un autre pas sans que rien ne le dise.
	impose := filmdec.NewFilmContextForMap(nil, &entry, nil).ImposedLayout()
	switch {
	case !g.LayoutDetected && (impose == nil || impose.AxisW != g.AxisW):
		return nil, nil, filmdec.I0Layout{}, filmdec.Vec3Range{}, fmt.Errorf("%w : fixture au decoupage %v (dit du CATALOGUE), catalogue %v",
			errGoldenInputsDecoupage, g.AxisW, imposeAxisW(impose))
	case g.LayoutDetected && impose != nil:
		return nil, nil, filmdec.I0Layout{}, filmdec.Vec3Range{}, fmt.Errorf(
			"%w : fixture dit son decoupage %v AUTO-DETECTE, or le catalogue en impose un (%v)",
			errGoldenInputsDecoupage, g.AxisW, impose.AxisW)
	}
	// LE DECOUPAGE VIENT DU BLOB, LES BORNES DU CATALOGUE : le premier dit comment le film a
	// quantifie, le second ou la carte commence et finit. Melanger les deux sources est ce qui
	// rendait des coordonnees fausses sur Live Fire.
	lay, world := filmdec.I0Layout{AxisW: g.AxisW}, entry.Range()
	g.FilmClockOriginUS = r.u()
	return g, r, lay, world, nil
}

// decodeGoldenEvenements relit tirs, equipements de depart, lancers et projectiles.
func decodeGoldenEvenements(r *greader, g *goldenInputs) {
	var lastTS uint64
	n := int(r.u())
	g.Fire = make([]filmdec.FireEvent, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		var e filmdec.FireEvent
		lastTS += r.u()
		e.TimestampUS = lastTS
		e.FilmIndex = int(r.i())
		e.WeaponID = r.u()
		if e.HasAim = r.bool8(); e.HasAim {
			for a := 0; a < 3; a++ {
				e.Aim[a] = r.f32()
			}
		}
		g.Fire = append(g.Fire, e)
	}

	n = int(r.u())
	g.Loadouts = make([]filmdec.KeyframeLoadout, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		l := filmdec.KeyframeLoadout{TimestampUS: r.u(), Slot: uint32(r.u())}
		nf := int(r.u())
		for j := 0; j < nf && r.err == nil; j++ {
			l.Families = append(l.Families, uint32(r.u()))
		}
		g.Loadouts = append(g.Loadouts, l)
	}

	n = int(r.u())
	g.Grenades = make([]filmdec.GrenadeThrow, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		g.Grenades = append(g.Grenades, filmdec.GrenadeThrow{
			TimestampUS: r.u(), FilmIndex: int(r.i()), TypeID: uint32(r.u()),
		})
	}

	g.Projectiles = decodeTracks(r)

}

// decodeGoldenInventaire relit les inventaires d image-cle et leurs deltas.
func decodeGoldenInventaire(r *greader, g *goldenInputs) {
	var lastTS uint64
	n := int(r.u())
	g.Inventory = make([]KeyframeInventory, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		inv := KeyframeInventory{TimestampUS: r.u(), Slot: uint32(r.u())}
		inv.GrenadesRead = r.bool8()
		for j := 0; j < invGrenadeSlots; j++ {
			inv.Grenades[j] = uint32(r.u())
		}
		inv.SelectedGrenadeRank = int(r.i())
		inv.AbilityRank = int(r.i())
		inv.DrawnSlot = int(r.i())
		inv.AmmoCandidates = int(r.u())
		inv.AmmoRead = r.bool8()
		for j := 0; j < invGrenadeSlots; j++ {
			inv.Ammo[j] = decodeAmmo(r)
		}
		g.Inventory = append(g.Inventory, inv)
	}

	n = int(r.u())
	g.InventoryDeltas = make([]filmdec.InventoryDelta, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		d := filmdec.InventoryDelta{TimestampUS: lastTS, Slot: uint32(r.u())}
		if gn := int(r.u()); gn > 0 {
			d.Grenades = make([]uint32, 0, gn)
			for j := 0; j < gn && r.err == nil; j++ {
				d.Grenades = append(d.Grenades, uint32(r.u()))
			}
		}
		d.SelRead = r.bool8()
		d.Sel = int(r.i())
		d.Mask = uint32(r.u())
		g.InventoryDeltas = append(g.InventoryDeltas, d)
	}

}

// decodeGoldenCanauxDelta relit rangs de capacite, camouflage, grappin et translocations.
func decodeGoldenCanauxDelta(r *greader, g *goldenInputs) {
	var lastTS uint64
	n := int(r.u())
	g.AbilityRanks = make([]filmdec.AbilityRank, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		g.AbilityRanks = append(g.AbilityRanks,
			filmdec.AbilityRank{TimestampUS: lastTS, Slot: uint32(r.u()), Rank: int(r.i())})
	}

	n = int(r.u())
	g.CamoStates = make([]filmdec.CamoRead, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		g.CamoStates = append(g.CamoStates,
			filmdec.CamoRead{TimestampUS: lastTS, Slot: uint32(r.u()), Q: uint16(r.u())})
	}

	n = int(r.u())
	g.GrappleReads = make([]filmdec.GrappleRead, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		gr := filmdec.GrappleRead{TimestampUS: lastTS, Slot: uint32(r.u()), Heavy: r.bool8()}
		for a := 0; a < 3; a++ {
			gr.PosQ[a] = uint32(r.u())
		}
		g.GrappleReads = append(g.GrappleReads, gr)
	}

	n = int(r.u())
	g.Translocations = make([]filmdec.TranslocatorTeleport, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		tr := filmdec.TranslocatorTeleport{TimestampUS: lastTS, Slot: uint32(r.u())}
		tr.HasPositions = r.bool8()
		for a := 0; a < 3; a++ {
			tr.From[a] = r.f32()
		}
		for a := 0; a < 3; a++ {
			tr.To[a] = r.f32()
		}
		g.Translocations = append(g.Translocations, tr)
	}

}

// decodeGoldenCapacites relit les impulsions et les charges de capacite, stats comprises.
func decodeGoldenCapacites(r *greader, g *goldenInputs) {
	var lastTS uint64
	n := int(r.u())
	g.AbilityImpulses = make([]filmdec.AbilityImpulse, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		g.AbilityImpulses = append(g.AbilityImpulses, filmdec.AbilityImpulse{
			TimestampUS: lastTS, Slot: uint32(r.u()), Predicted: r.bool8()})
	}
	g.AbilityImpulseStats = filmdec.AbilityImpulseStats{
		Records: int(r.u()), WithI57: int(r.u()), WithI59: int(r.u()),
		Read: int(r.u()), Unread: int(r.u()), Tag1: int(r.u()), Absent: r.bool8(),
		Scanned: r.bool8(),
	}

	n = int(r.u())
	g.AbilityCharges = make([]filmdec.AbilityCharge, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		g.AbilityCharges = append(g.AbilityCharges, filmdec.AbilityCharge{
			TimestampUS: lastTS, Slot: uint32(r.u()),
			Emplacement: int(r.u()), Charges: int(r.u()), Low: int(r.u())})
	}
	g.AbilityChargeStats = filmdec.AbilityChargeStats{
		Records: int(r.u()), WithI56: int(r.u()),
		Read: int(r.u()), Unread: int(r.u()), Armed: int(r.u()),
		Absent: r.bool8(), Scanned: r.bool8(),
	}

}

// decodeGoldenMonde relit les poses d equipement et les deux voies de socles.
func decodeGoldenMonde(r *greader, g *goldenInputs) {
	var lastTS uint64
	n := int(r.u())
	g.Placements = make([]filmdec.EquipmentPlacement, 0, n)
	lastTS = 0
	for k := 0; k < n && r.err == nil; k++ {
		lastTS += r.u()
		p := filmdec.EquipmentPlacement{T0US: lastTS, T1US: r.u()}
		p.Life = filmdec.EquipmentLifeKey{Slot: uint32(r.u()), Gen: uint32(r.u())}
		p.X, p.Y, p.Z = r.f32(), r.f32(), r.f32()
		p.GlobalID, p.Points = uint32(r.u()), int(r.u())
		g.Placements = append(g.Placements, p)
	}
	g.PlacementStats = filmdec.EquipmentPlacementStats{ByID: map[uint32]int{}}
	g.PlacementStats.Calibration.Widths = filmdec.MPPWidths{Lead: int(r.i()), Index: int(r.i())}
	g.PlacementStats.Calibration.Agree = int(r.i())
	g.PlacementStats.Lives = int(r.u())
	g.PlacementStats.Anchors = int(r.u())
	g.PlacementStats.Accepted = int(r.u())
	g.PlacementStats.Confirmed = int(r.u())
	g.PlacementStats.Placements = len(g.Placements)

	g.Pads.Weapons = decodeWorldObjectScan(r)
	g.Pads.Powerups = decodeWorldObjectScan(r)

}

// decodeGoldenQueue relit les morts et la table des index de joueur.
func decodeGoldenQueue(r *greader, g *goldenInputs) {
	n := int(r.u())
	g.Deaths = make([]Death, 0, n)
	for k := 0; k < n && r.err == nil; k++ {
		g.Deaths = append(g.Deaths, Death{XUID: r.u(), Gamertag: r.str(), TimeMS: r.i()})
	}

	g.PlayerIndices = PlayerIndexTable{ByXUID: map[uint64]int{}}
	g.PlayerIndices.Readings = int(r.u())
	g.PlayerIndices.Disagreements = int(r.u())
	n = int(r.u())
	for k := 0; k < n && r.err == nil; k++ {
		x := r.u()
		g.PlayerIndices.ByXUID[x] = int(r.i())
	}
}
