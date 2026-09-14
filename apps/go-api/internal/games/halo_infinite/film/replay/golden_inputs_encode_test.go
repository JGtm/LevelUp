package replay

// golden_inputs_encode_test.go — L ENCODEUR DU FIXTURE D ENTREES.
//
// Extrait de golden_inputs_test.go le 2026-09-14 (revue R1, constat R1-7 : un fichier de
// 1 560 lignes, deux fonctions de ~290). DEPLACEMENT PUR : aucune ligne de logique changee,
// le decoupage suit les SECTIONS du blob, et chaque section a desormais sa fonction.

import "sort"

const (
	gpHasWorld  byte = 1 << 0
	gpHasYaw    byte = 1 << 1
	gpHasBody   byte = 1 << 2
	gpHasShield byte = 1 << 3
)

// encodeGoldenInputs serialise les entrees. Format decrit en tete de fichier.

func encodeGoldenInputs(g *goldenInputs) []byte {
	w := &gwriter{b: []byte(goldenInputsMagic)}
	encodeGoldenEntete(w, g)
	encodePositionSection(w, g.Positions)
	encodeBipedCreations(w, g.BipedCreations)
	encodeGoldenEvenements(w, g)
	encodeWeaponChanges(w, g.WeaponChanges)
	encodePickups(w, g.Pickups, g.PickupStats)
	encodeGoldenInventaire(w, g)
	encodeGoldenCanauxDelta(w, g)
	encodeEquipmentChanges(w, g.EquipmentChanges, g.EquipmentChangeStats)
	encodeGoldenCapacites(w, g)
	encodeZoomEvents(w, g.ZoomEvents)
	encodeGoldenMonde(w, g)
	encodeVehicleScan(w, g.Vehicles)
	encodeGoldenQueue(w, g)
	return w.b
}

// encodeGoldenEntete ecrit l en-tete du blob.
func encodeGoldenEntete(w *gwriter, g *goldenInputs) {
	w.str(g.Film)
	// LE MODULE DE LA CARTE OUVRE LE BLOB (lot 0.D.3 bis). Les positions y sont des QUANTA :
	// sans l entree de catalogue qui les a produites, elles ne se dequantifient pas — et avec
	// la MAUVAISE, elles se dequantifient en coordonnees FAUSSES, pas approximatives
	// (cf. DequantBipedAxis). Le module est donc ecrit ici et VERIFIE a la relecture.
	w.str(g.MapModule)
	for a := 0; a < 3; a++ {
		w.u(uint64(g.AxisW[a]))
	}
	w.bool8(g.LayoutDetected)
	// LES STATS DE BALAYAGE QUE L ASSEMBLAGE CONSOMME (revue R1, constat R1-1). Elles ne sont
	// pas des donnees mais des VERDICTS du decodeur, et le document les publie : sans elles le
	// golden figeait une constante (« canal munitions refuse : false », version de film absente)
	// et la fidelite etait aveugle DES DEUX COTES.
	w.bool8(g.InventoryDeltaAmmoRefused)
	w.bool8(g.FilmMajorVersion != nil)
	if g.FilmMajorVersion != nil {
		w.i(int64(*g.FilmMajorVersion))
	}
	w.u(g.FilmClockOriginUS)
}

// encodeGoldenEvenements ecrit tirs, equipements de depart, lancers et projectiles.
func encodeGoldenEvenements(w *gwriter, g *goldenInputs) {
	var lastTS uint64
	w.u(uint64(len(g.Fire)))
	lastTS = 0
	for _, e := range g.Fire {
		w.u(e.TimestampUS - lastTS)
		lastTS = e.TimestampUS
		w.i(int64(e.FilmIndex))
		w.u(e.WeaponID)
		w.bool8(e.HasAim)
		if e.HasAim {
			for a := 0; a < 3; a++ {
				w.f32(e.Aim[a])
			}
		}
	}

	w.u(uint64(len(g.Loadouts)))
	for _, l := range g.Loadouts {
		w.u(l.TimestampUS)
		w.u(uint64(l.Slot))
		w.u(uint64(len(l.Families)))
		for _, f := range l.Families {
			w.u(uint64(f))
		}
	}

	w.u(uint64(len(g.Grenades)))
	for _, t := range g.Grenades {
		w.u(t.TimestampUS)
		w.i(int64(t.FilmIndex))
		w.u(uint64(t.TypeID))
	}

	encodeTracks(w, g.Projectiles)
}

// encodeGoldenInventaire ecrit les inventaires d image-cle et leurs deltas.
func encodeGoldenInventaire(w *gwriter, g *goldenInputs) {
	var lastTS uint64
	w.u(uint64(len(g.Inventory)))
	for _, inv := range g.Inventory {
		w.u(inv.TimestampUS)
		w.u(uint64(inv.Slot))
		w.bool8(inv.GrenadesRead)
		for _, c := range inv.Grenades {
			w.u(uint64(c))
		}
		w.i(int64(inv.SelectedGrenadeRank))
		w.i(int64(inv.AbilityRank))
		w.i(int64(inv.DrawnSlot))
		w.u(uint64(inv.AmmoCandidates))
		w.bool8(inv.AmmoRead)
		for _, a := range inv.Ammo {
			encodeAmmo(w, a)
		}
	}

	w.u(uint64(len(g.InventoryDeltas)))
	lastTS = 0
	for _, d := range g.InventoryDeltas {
		w.u(d.TimestampUS - lastTS) // horodatages non decroissants dans l ordre du film
		lastTS = d.TimestampUS
		w.u(uint64(d.Slot))
		w.u(uint64(len(d.Grenades)))
		for _, c := range d.Grenades {
			w.u(uint64(c))
		}
		w.bool8(d.SelRead)
		w.i(int64(d.Sel))
		w.u(uint64(d.Mask))
	}

}

// encodeGoldenCanauxDelta ecrit rangs de capacite, camouflage, grappin et translocations.
func encodeGoldenCanauxDelta(w *gwriter, g *goldenInputs) {
	var lastTS uint64
	w.u(uint64(len(g.AbilityRanks)))
	lastTS = 0
	for _, a := range g.AbilityRanks {
		w.u(a.TimestampUS - lastTS) // horodatages non decroissants dans l ordre du film
		lastTS = a.TimestampUS
		w.u(uint64(a.Slot))
		w.i(int64(a.Rank))
	}

	w.u(uint64(len(g.CamoStates)))
	lastTS = 0
	for _, cr := range g.CamoStates {
		w.u(cr.TimestampUS - lastTS) // horodatages non decroissants dans l ordre du film
		lastTS = cr.TimestampUS
		w.u(uint64(cr.Slot))
		w.u(uint64(cr.Q))
	}

	w.u(uint64(len(g.GrappleReads)))
	lastTS = 0
	for _, gr := range g.GrappleReads {
		w.u(gr.TimestampUS - lastTS) // horodatages non decroissants dans l ordre du film
		lastTS = gr.TimestampUS
		w.u(uint64(gr.Slot))
		w.bool8(gr.Heavy)
		for a := 0; a < 3; a++ {
			w.u(uint64(gr.PosQ[a]))
		}
	}

	w.u(uint64(len(g.Translocations)))
	lastTS = 0
	for _, tr := range g.Translocations {
		w.u(tr.TimestampUS - lastTS) // le scan rend les evenements tries par instant
		lastTS = tr.TimestampUS
		w.u(uint64(tr.Slot))
		// LE VA-ET-VIENT VOYAGE AVEC SON TEMOIN (v12) : sans lui, un saut sans position
		// serait indistinguable d un saut vers l origine du monde.
		w.bool8(tr.HasPositions)
		for a := 0; a < 3; a++ {
			w.f32(tr.From[a])
		}
		for a := 0; a < 3; a++ {
			w.f32(tr.To[a])
		}
	}

	// LES IMPULSIONS DE CAPACITE (v13) : le scan les rend TRIEES par instant, d ou le delta.
	// Les STATS suivent la liste — c est le temoin `Absent` qui distingue « ce film ne
	// transmet pas le composant » de « personne ne s en est servi ».
}

// encodeGoldenCapacites ecrit les impulsions et les charges de capacite, stats comprises.
func encodeGoldenCapacites(w *gwriter, g *goldenInputs) {
	var lastTS uint64
	w.u(uint64(len(g.AbilityImpulses)))
	lastTS = 0
	for _, im := range g.AbilityImpulses {
		w.u(im.TimestampUS - lastTS)
		lastTS = im.TimestampUS
		w.u(uint64(im.Slot))
		w.bool8(im.Predicted)
	}
	w.u(uint64(g.AbilityImpulseStats.Records))
	w.u(uint64(g.AbilityImpulseStats.WithI57))
	w.u(uint64(g.AbilityImpulseStats.WithI59))
	w.u(uint64(g.AbilityImpulseStats.Read))
	w.u(uint64(g.AbilityImpulseStats.Unread))
	w.u(uint64(g.AbilityImpulseStats.Tag1))
	w.bool8(g.AbilityImpulseStats.Absent)
	// `Scanned` VOYAGE AVEC LES AUTRES : sans lui, un fixture rendrait une couverture de zeros
	// indistinguable d un balayage qui n a jamais tourne (constat H1 de la revue de ronde 1).
	w.bool8(g.AbilityImpulseStats.Scanned)

	// LES CHARGES RESTANTES (v14) : le scan les rend TRIEES par instant, d ou le delta. Les
	// STATS suivent la liste, `Absent` et `Scanned` compris — memes temoins, memes raisons
	// que les impulsions ci-dessus.
	w.u(uint64(len(g.AbilityCharges)))
	lastTS = 0
	for _, ac := range g.AbilityCharges {
		w.u(ac.TimestampUS - lastTS)
		lastTS = ac.TimestampUS
		w.u(uint64(ac.Slot))
		w.u(uint64(ac.Emplacement))
		w.u(uint64(ac.Charges))
		w.u(uint64(ac.Low))
	}
	w.u(uint64(g.AbilityChargeStats.Records))
	w.u(uint64(g.AbilityChargeStats.WithI56))
	w.u(uint64(g.AbilityChargeStats.Read))
	w.u(uint64(g.AbilityChargeStats.Unread))
	w.u(uint64(g.AbilityChargeStats.Armed))
	w.bool8(g.AbilityChargeStats.Absent)
	w.bool8(g.AbilityChargeStats.Scanned)

	// Les POSES, puis la CALIBRATION qui les rend lisibles. Les deux vont ensemble : une
	// liste vide ne dit pas la meme chose selon que le film a tranche sa largeur ou non.
}

// encodeGoldenMonde ecrit les poses d equipement et les deux voies de socles.
func encodeGoldenMonde(w *gwriter, g *goldenInputs) {
	var lastTS uint64
	w.u(uint64(len(g.Placements)))
	lastTS = 0
	for _, p := range g.Placements {
		w.u(p.T0US - lastTS) // les poses sont triees par instant de creation
		lastTS = p.T0US
		w.u(p.T1US)
		w.u(uint64(p.Life.Slot))
		w.u(uint64(p.Life.Gen))
		w.f32(p.X)
		w.f32(p.Y)
		w.f32(p.Z)
		w.u(uint64(p.GlobalID))
		w.u(uint64(p.Points))
	}
	w.i(int64(g.PlacementStats.Calibration.Widths.Lead))
	w.i(int64(g.PlacementStats.Calibration.Widths.Index))
	w.i(int64(g.PlacementStats.Calibration.Agree))
	w.u(uint64(g.PlacementStats.Lives))
	w.u(uint64(g.PlacementStats.Anchors))
	w.u(uint64(g.PlacementStats.Accepted))
	w.u(uint64(g.PlacementStats.Confirmed))

	encodeWorldObjectScan(w, g.Pads.Weapons)
	encodeWorldObjectScan(w, g.Pads.Powerups)
}

// encodeGoldenQueue ecrit les morts et la table des index de joueur.
func encodeGoldenQueue(w *gwriter, g *goldenInputs) {

	w.u(uint64(len(g.Deaths)))
	for _, d := range g.Deaths {
		w.u(d.XUID)
		w.str(d.Gamertag)
		w.i(d.TimeMS)
	}

	w.u(uint64(g.PlayerIndices.Readings))
	w.u(uint64(g.PlayerIndices.Disagreements))
	xuids := make([]uint64, 0, len(g.PlayerIndices.ByXUID))
	for x := range g.PlayerIndices.ByXUID {
		xuids = append(xuids, x)
	}
	sort.Slice(xuids, func(i, j int) bool { return xuids[i] < xuids[j] })
	w.u(uint64(len(xuids)))
	for _, x := range xuids {
		w.u(x)
		w.i(int64(g.PlayerIndices.ByXUID[x]))
	}

}
