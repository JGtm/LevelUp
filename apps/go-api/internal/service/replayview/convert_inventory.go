package replayview

// convert_inventory.go — inventaire, capacite d'armure, equipement, ramassages et
// changements d'arme. Jumeau de `domain/replaydoc/inventory.go`.

import (
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/replaydoc"
)

func toInventory(v replay.Inventory) replaydoc.Inventory {
	return replaydoc.Inventory{
		T:     v.T,
		Slot:  v.Slot,
		G:     v.G,
		Gs:    v.Gs,
		D:     v.D,
		Am:    sliceOf(v.Am, toAmmoSlot),
		Cand:  v.Cand,
		Empty: v.Empty,
	}
}

func toAmmoSlot(v replay.AmmoSlot) replaydoc.AmmoSlot {
	return replaydoc.AmmoSlot{
		Mag:   v.Mag,
		Res:   v.Res,
		Gauge: v.Gauge,
	}
}

func toAbilityRead(v replay.AbilityRead) replaydoc.AbilityRead {
	return replaydoc.AbilityRead{
		T:    v.T,
		Slot: v.Slot,
		R:    v.R,
		Src:  v.Src,
	}
}

func toGrenadeRead(v replay.GrenadeRead) replaydoc.GrenadeRead {
	return replaydoc.GrenadeRead{
		T:    v.T,
		Slot: v.Slot,
		G:    v.G,
		Gs:   v.Gs,
		Src:  v.Src,
	}
}

func toAbilityImpulse(v replay.AbilityImpulse) replaydoc.AbilityImpulse {
	return replaydoc.AbilityImpulse{
		T:      v.T,
		Slot:   v.Slot,
		Family: v.Family,
	}
}

func toAbilityCharge(v replay.AbilityCharge) replaydoc.AbilityCharge {
	return replaydoc.AbilityCharge{
		T:       v.T,
		Slot:    v.Slot,
		Family:  v.Family,
		Charges: v.Charges,
	}
}

func toWeaponChange(v replay.WeaponChange) replaydoc.WeaponChange {
	return replaydoc.WeaponChange{
		T:    v.T,
		Slot: v.Slot,
		Kind: replaydoc.WeaponChangeKind(v.Kind),
		W:    v.W,
		From: v.From,
	}
}

func toEquipmentChange(v replay.EquipmentChange) replaydoc.EquipmentChange {
	return replaydoc.EquipmentChange{
		T:         v.T,
		Slot:      v.Slot,
		Kind:      replaydoc.EquipmentChangeKind(v.Kind),
		R:         v.R,
		From:      v.From,
		Recovered: v.Recovered,
		Gap:       v.Gap,
	}
}

func toEquipmentEpisode(v replay.EquipmentEpisode) replaydoc.EquipmentEpisode {
	return replaydoc.EquipmentEpisode{
		Slot:    v.Slot,
		Fam:     v.Fam,
		T0:      v.T0,
		T1:      v.T1,
		EndRead: v.EndRead,
		K:       v.K,
		A:       v.A,
	}
}

func toEquipmentPlacement(v replay.EquipmentPlacement) replaydoc.EquipmentPlacement {
	return replaydoc.EquipmentPlacement{
		T0:       v.T0,
		T1:       v.T1,
		X:        v.X,
		Y:        v.Y,
		Z:        v.Z,
		Family:   v.Family,
		ID:       v.ID,
		Owner:    v.Owner,
		H:        v.H,
		Origin:   v.Origin,
		Until:    v.Until,
		UntilMax: v.UntilMax,
		End:      v.End,
	}
}

func toGrappleLine(v replay.GrappleLine) replaydoc.GrappleLine {
	return replaydoc.GrappleLine{
		Slot: v.Slot,
		T0:   v.T0,
		T1:   v.T1,
		AX:   v.AX,
		AY:   v.AY,
		AZ:   v.AZ,
	}
}

func toTranslocation(v replay.Translocation) replaydoc.Translocation {
	return replaydoc.Translocation{
		T:    v.T,
		Slot: v.Slot,
		FX:   v.FX,
		FY:   v.FY,
		FZ:   v.FZ,
		TX:   v.TX,
		TY:   v.TY,
		TZ:   v.TZ,
	}
}

func toPickup(v replay.Pickup) replaydoc.Pickup {
	return replaydoc.Pickup{
		T:      v.T,
		Slot:   v.Slot,
		XUID:   v.XUID,
		W:      v.W,
		Family: v.Family,
		Kind:   replaydoc.PickupKind(v.Kind),
		Class:  v.Class,
		Origin: v.Origin,
	}
}
