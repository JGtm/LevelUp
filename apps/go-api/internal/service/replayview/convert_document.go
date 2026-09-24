package replayview

// convert_document.go — la racine du document et les types de trajectoire, tir, grenade,
// projectile et libelle. Jumeau de `domain/replaydoc/document.go`.

import (
	"maps"

	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

func toReplayDocument(v replay.ReplayDocument) replaydoc.ReplayDocument {
	return replaydoc.ReplayDocument{
		SchemaVersion:       v.SchemaVersion,
		MatchID:             v.MatchID,
		TitleSlug:           v.TitleSlug,
		FrameCount:          v.FrameCount,
		Bounds:              toBounds(v.Bounds),
		Tracks:              sliceOf(v.Tracks, toTrack),
		FrameIntervalMS:     v.FrameIntervalMS,
		DurationMS:          v.DurationMS,
		OriginMs:            v.OriginMs,
		T0FilmMs:            v.T0FilmMs,
		Geometry:            sliceOf(v.Geometry, toMapObject),
		GeometryBounds:      ptrOf(v.GeometryBounds, toBounds),
		Structure:           sliceOf(v.Structure, toSurface),
		StructureBounds:     ptrOf(v.StructureBounds, toBounds),
		Shots:               sliceOf(v.Shots, toShot),
		Loadouts:            sliceOf(v.Loadouts, toLoadout),
		Inventory:           sliceOf(v.Inventory, toInventory),
		GrenadeLabels:       sliceOf(v.GrenadeLabels, toLabel),
		Abilities:           sliceOf(v.Abilities, toAbilityRead),
		GrenadeReads:        sliceOf(v.GrenadeReads, toGrenadeRead),
		AbilityLabels:       mapOf(v.AbilityLabels, toLabel),
		EquipmentEpisodes:   sliceOf(v.EquipmentEpisodes, toEquipmentEpisode),
		Stances:             sliceOf(v.Stances, toStance),
		GrappleLines:        sliceOf(v.GrappleLines, toGrappleLine),
		EquipmentPlacements: sliceOf(v.EquipmentPlacements, toEquipmentPlacement),
		WeaponChanges:       sliceOf(v.WeaponChanges, toWeaponChange),
		Pickups:             sliceOf(v.Pickups, toPickup),
		EquipmentChanges:    sliceOf(v.EquipmentChanges, toEquipmentChange),
		Translocations:      sliceOf(v.Translocations, toTranslocation),
		AbilityImpulses:     sliceOf(v.AbilityImpulses, toAbilityImpulse),
		AbilityCharges:      sliceOf(v.AbilityCharges, toAbilityCharge),
		GroundWeapons:       sliceOf(v.GroundWeapons, toGroundWeapon),
		Vehicles:            sliceOf(v.Vehicles, toVehicleTrack),
		VehicleLabels:       mapOf(v.VehicleLabels, toVehicleLabel),
		VehicleWeapons:      mapOf(v.VehicleWeapons, toVehicleWeapon),
		VehicleScenery:      ptrOf(v.VehicleScenery, toVehicleScenery),
		VehicleCycles:       sliceOf(v.VehicleCycles, toVehicleCycle),
		WeaponPads:          sliceOf(v.WeaponPads, toWeaponPad),
		PadPickups:          sliceOf(v.PadPickups, toPadPickup),
		Grenades:            sliceOf(v.Grenades, toGrenade),
		Projectiles:         sliceOf(v.Projectiles, toProjectile),
		WeaponLabels:        mapOf(v.WeaponLabels, toWeaponLabel),
		KillEffects:         v.KillEffects,
		NeutralDeaths:       sliceOf(v.NeutralDeaths, toNeutralDeath),
		Roster:              sliceOf(v.Roster, toRosterEntry),
		MapObjectives:       ptrOf(v.MapObjectives, toMapObjectives),
		MapWeaponPads:       ptrOf(v.MapWeaponPads, toMapWeaponPads),
		WeaponTiers:         ptrOf(v.WeaponTiers, toWeaponTiersInfo),
		Objectives:          sliceOf(v.Objectives, toObjectiveAction),
		ScoreTimeline:       ptrOf(v.ScoreTimeline, toScoreTimeline),
		FlagCarries:         sliceOf(v.FlagCarries, toFlagCarry),
		FlagReturnZone:      ptrOf(v.FlagReturnZone, toFlagReturnZone),
		ObjectiveObjects:    sliceOf(v.ObjectiveObjects, toObjectiveObjectLife),
		ZoneStates:          sliceOf(v.ZoneStates, toZoneState),
		VipCrown:            sliceOf(v.VipCrown, toVipPeriod),
		BombArmings:         sliceOf(v.BombArmings, toBombArming),
		SkullCarries:        sliceOf(v.SkullCarries, toSkullCarry),
		BombCarries:         sliceOf(v.BombCarries, toBombCarry),
		BombStats:           ptrOf(v.BombStats, toBombMatchStats),
		BombEvents:          sliceOf(v.BombEvents, toBombEvent),
		Coverage:            ptrOf(v.Coverage, toCoverage),
		Identity:            ptrOf(v.Identity, toIdentitySection),
		// `layers` VOYAGE TEL QUEL : une table de chaines courtes, dont les cles sont les balises
		// JSON des calques et les valeurs des revisions de couche. Rien a traduire — et surtout pas
		// a completer : une entree absente est une reponse (« ce calque n a pas ete produit »).
		Layers: maps.Clone(v.Layers),
	}
}

func toBounds(v replay.Bounds) replaydoc.Bounds {
	return replaydoc.Bounds{
		MinX: v.MinX,
		MinY: v.MinY,
		MaxX: v.MaxX,
		MaxY: v.MaxY,
		MinZ: v.MinZ,
		MaxZ: v.MaxZ,
	}
}

func toTrack(v replay.Track) replaydoc.Track {
	return replaydoc.Track{
		Slot:       v.Slot,
		Team:       v.Team,
		Name:       v.Name,
		XUID:       v.XUID,
		Bot:        v.Bot,
		Points:     sliceOf(v.Points, toPoint),
		StartFrame: v.StartFrame,
		EndFrame:   v.EndFrame,
	}
}

func toPoint(v replay.Point) replaydoc.Point {
	return replaydoc.Point{
		T:  v.T,
		X:  v.X,
		Y:  v.Y,
		Z:  v.Z,
		G:  v.G,
		H:  v.H,
		P:  v.P,
		Sh: v.Sh,
		Hp: v.Hp,
		S:  v.S,
	}
}

func toRosterEntry(v replay.RosterEntry) replaydoc.RosterEntry {
	return replaydoc.RosterEntry{
		XUID:       v.XUID,
		FilmIndex:  v.FilmIndex,
		Name:       v.Name,
		Team:       v.Team,
		Bot:        v.Bot,
		Bid:        v.Bid,
		Seat:       v.Seat,
		SeatSource: v.SeatSource,
		Presence:   presenceServie(v),
	}
}

// presenceServie recopie les intervalles de presence d une entree de roster (lot M2.3, schema 69).
//
// LA BOUCLE NE NOMME PAS LE TYPE STOCKE, et c est voulu : la surface de `film/replay` citee hors de
// `film/` est un ratchet date (`archlint/film_facade_surface_test.go`), qu un type de transport
// recopie champ pour champ n a pas a faire monter. Nil reste nil, comme `sliceOf`.
func presenceServie(v replay.RosterEntry) []replaydoc.PresenceInterval {
	if v.Presence == nil {
		return nil
	}
	out := make([]replaydoc.PresenceInterval, 0, len(v.Presence))
	for _, p := range v.Presence {
		out = append(out, replaydoc.PresenceInterval{From: p.From, To: p.To, ToMax: p.ToMax})
	}
	return out
}

func toShot(v replay.Shot) replaydoc.Shot {
	return replaydoc.Shot{
		T:       v.T,
		Slot:    v.Slot,
		X:       v.X,
		Y:       v.Y,
		H:       v.H,
		Weapon:  v.Weapon,
		Vehicle: v.Vehicle,
	}
}

func toLoadout(v replay.Loadout) replaydoc.Loadout {
	return replaydoc.Loadout{
		T:    v.T,
		Slot: v.Slot,
		W:    v.W,
		Src:  v.Src,
		K:    v.K,
	}
}

func toGrenade(v replay.Grenade) replaydoc.Grenade {
	return replaydoc.Grenade{
		T:    v.T,
		Slot: v.Slot,
		Idx:  v.Idx,
		X:    v.X,
		Y:    v.Y,
		Rank: v.Rank,
		Src:  v.Src,
		Proj: v.Proj,
	}
}

func toProjectile(v replay.Projectile) replaydoc.Projectile {
	return replaydoc.Projectile{
		T0:   v.T0,
		P:    v.P,
		Rest: v.Rest,
	}
}

func toNeutralDeath(v replay.NeutralDeath) replaydoc.NeutralDeath {
	return replaydoc.NeutralDeath{
		XUID:   v.XUID,
		FeedMs: v.FeedMs,
		Kind:   v.Kind,
		Img:    v.Img,
		Tinted: v.Tinted,
	}
}

func toLabel(v replay.Label) replaydoc.Label {
	return replaydoc.Label{
		En:     v.En,
		Fr:     v.Fr,
		Img:    v.Img,
		Tinted: v.Tinted,
		Family: v.Family,
	}
}

func toWeaponLabel(v replay.WeaponLabel) replaydoc.WeaponLabel {
	return replaydoc.WeaponLabel{
		En:     v.En,
		Fr:     v.Fr,
		Fx:     v.Fx,
		Key:    v.Key,
		Role:   v.Role,
		Tint:   v.Tint,
		Img:    v.Img,
		Tinted: v.Tinted,
	}
}

func toVehicleLabel(v replay.VehicleLabel) replaydoc.VehicleLabel {
	return replaydoc.VehicleLabel{
		Img:    v.Img,
		Tinted: v.Tinted,
		Kind:   v.Kind,
		En:     v.En,
		Fr:     v.Fr,
	}
}
