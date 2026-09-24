package replayview

// convert_coverage.go — couvertures des calques generaux. Jumeau de
// `domain/replaydoc/coverage.go`.

import (
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

func toCoverage(v replay.Coverage) replaydoc.Coverage {
	return replaydoc.Coverage{
		Shots:             toLayerCoverage(v.Shots),
		Grenades:          toLayerCoverage(v.Grenades),
		Objectives:        toLayerCoverage(v.Objectives),
		Tracks:            ptrOf(v.Tracks, toTrackCoverage),
		Teams:             ptrOf(v.Teams, toTeamCoverage),
		Seats:             ptrOf(v.Seats, toSeatCoverage),
		Projectiles:       ptrOf(v.Projectiles, toProjectileCoverage),
		Equipment:         ptrOf(v.Equipment, toEquipmentCoverage),
		Stances:           ptrOf(v.Stances, toStanceCoverage),
		Grapple:           ptrOf(v.Grapple, toGrappleCoverage),
		Placements:        ptrOf(v.Placements, toEquipmentPlacementCoverage),
		GroundWeapons:     ptrOf(v.GroundWeapons, toGroundWeaponCoverage),
		Score:             ptrOf(v.Score, toScoreCoverage),
		FlagCarries:       ptrOf(v.FlagCarries, toFlagCarriesCoverage),
		VipCrown:          ptrOf(v.VipCrown, toVipCrownCoverage),
		SkullCarries:      ptrOf(v.SkullCarries, toSkullCarriesCoverage),
		BombCarries:       ptrOf(v.BombCarries, toBombCarriesCoverage),
		BombArmings:       ptrOf(v.BombArmings, toBombArmingsCoverage),
		WeaponChanges:     ptrOf(v.WeaponChanges, toWeaponChangeCoverage),
		Pickups:           ptrOf(v.Pickups, toPickupCoverage),
		PadDating:         ptrOf(v.PadDating, toPadDatingStats),
		EquipmentChanges:  ptrOf(v.EquipmentChanges, toEquipmentChangeCoverage),
		Translocations:    ptrOf(v.Translocations, toTranslocationCoverage),
		AbilityImpulses:   ptrOf(v.AbilityImpulses, toAbilityImpulseCoverage),
		AbilityCharges:    ptrOf(v.AbilityCharges, toAbilityChargeCoverage),
		GroundWeaponItems: ptrOf(v.GroundWeaponItems, toGroundWeaponItemsCoverage),
		Vehicles:          ptrOf(v.Vehicles, toVehicleCoverage),
		ObjectiveObjects:  ptrOf(v.ObjectiveObjects, toObjectiveObjectsCoverage),
		Inventory:         ptrOf(v.Inventory, toInventoryCoverage),
		FilmMajorVersion:  v.FilmMajorVersion,
		GrenadeReads:      ptrOf(v.GrenadeReads, toGrenadeReadCoverage),
		Abilities:         ptrOf(v.Abilities, toAbilityCoverage),
		Zones:             ptrOf(v.Zones, toZonesCoverage),
		OriginResolved:    v.OriginResolved,
		T0Film:            ptrOf(v.T0Film, toT0FilmCoverage),
		Verdict:           v.Verdict,
		Bridge:            toBridgeHealth(v.Bridge),
		Fallbacks:         toFallbackHits(v.Fallbacks),
		Decoder:           ptrOf(v.Decoder, toDecoderCoverage),
		DeathsPaths:       ptrOf(v.DeathsPaths, toDeathsPathsCoverage),
	}
}

// toFallbackHits projette les replis declenches. Nil reste nil : `omitempty` fait alors
// disparaitre le champ, et un document sans repli ne porte pas de liste vide.
func toFallbackHits(v []replay.FallbackHit) []replaydoc.FallbackHit {
	if len(v) == 0 {
		return nil
	}
	out := make([]replaydoc.FallbackHit, 0, len(v))
	for _, h := range v {
		out = append(out, replaydoc.FallbackHit{Name: h.Name, Hits: h.Hits})
	}
	return out
}

func toLayerCoverage(v replay.LayerCoverage) replaydoc.LayerCoverage {
	return replaydoc.LayerCoverage{
		Available:       v.Available,
		Attached:        v.Attached,
		NoSlot:          v.NoSlot,
		Ambiguous:       v.Ambiguous,
		OutOfWindow:     v.OutOfWindow,
		Unpublished:     v.Unpublished,
		RefusedByRoster: v.RefusedByRoster,
	}
}

func toTrackCoverage(v replay.TrackCoverage) replaydoc.TrackCoverage {
	return replaydoc.TrackCoverage{
		Published:        v.Published,
		PublishedPoints:  v.PublishedPoints,
		RefusedMinPoints: v.RefusedMinPoints,
		RefusedPoints:    v.RefusedPoints,
		MinPoints:        v.MinPoints,
		Gaps:             v.Gaps,
		GapMS:            v.GapMS,
	}
}

func toTeamCoverage(v replay.TeamCoverage) replaydoc.TeamCoverage {
	return replaydoc.TeamCoverage{
		Read:          v.Read,
		Refusal:       v.Refusal,
		Records:       v.Records,
		Rejected:      v.Rejected,
		Divergences:   v.Divergences,
		Film:          v.Film,
		NoTeam:        v.NoTeam,
		Unread:        v.Unread,
		Accord:        v.Accord,
		Contradiction: v.Contradiction,
		Silence:       v.Silence,
		Tracks:        v.Tracks,
		TracksNamed:   v.TracksNamed,

		TracksSlotAmbiguous: v.TracksSlotAmbiguous,
	}
}

// toSeatCoverage : la couverture des places (lot 1.9.14 ; lot M2.3, schema 69).
func toSeatCoverage(v replay.SeatCoverage) replaydoc.SeatCoverage {
	return replaydoc.SeatCoverage{
		Entrees:         v.Entrees,
		Sieges:          v.Sieges,
		Lus:             v.Lus,
		PlacesTirs:      v.PlacesTirs,
		Apparies:        v.Apparies,
		PlacesOuvertes:  v.PlacesOuvertes,
		SansPlace:       v.SansPlace,
		ReprisesEcrites: v.ReprisesEcrites,
		Arrivants:       v.Arrivants,
		PresencesCloses: v.PresencesCloses,
		SansPresence:    v.SansPresence,
		OccupantsMax:    v.OccupantsMax,
		Capacite:        v.Capacite,
		Depassements:    v.Depassements,
		PlacesEnTrop:    v.PlacesEnTrop,
		SansEquipe:      v.SansEquipe,

		IdentitesHorsRoster: v.IdentitesHorsRoster,
		BotsSuccesseurs:     v.BotsSuccesseurs,
		PresencesParLesVies: v.PresencesParLesVies,

		RelaisBornes:      v.RelaisBornes,
		Chevauchements:    v.Chevauchements,
		TirsContestes:     v.TirsContestes,
		TirsIndexTronque:  v.TirsIndexTronque,
		Presences:         v.Presences,
		EntitesNonLiees:   v.EntitesNonLiees,
		EntitesContestees: v.EntitesContestees,
		TrousDEntite:      v.TrousDEntite,
		SansTableDuFilm:   v.SansTableDuFilm,
	}
}

func toProjectileCoverage(v replay.ProjectileCoverage) replaydoc.ProjectileCoverage {
	return replaydoc.ProjectileCoverage{
		Tracks:    v.Tracks,
		Published: v.Published,
		Truncated: v.Truncated,
	}
}

func toBridgeHealth(v replay.BridgeHealth) replaydoc.BridgeHealth {
	return replaydoc.BridgeHealth{
		Slots:              v.Slots,
		FromReading:        v.FromReading,
		LivesNamed:         v.LivesNamed,
		LivesTotal:         v.LivesTotal,
		IndexReadings:      v.IndexReadings,
		IndexDisagreements: v.IndexDisagreements,
		SlotCollisions:     v.SlotCollisions,
		Concordant:         v.Concordant,
		Discordant:         v.Discordant,
		BridgeNamedLives:   v.BridgeNamedLives,

		DirectByCreation:           v.DirectByCreation,
		DirectByCreationPropagated: v.DirectByCreationPropagated,
		BodiesWithCreation:         v.BodiesWithCreation,
		NamedByPreviousLife:        v.NamedByPreviousLife,
		NamedByNextLife:            v.NamedByNextLife,
		NamedBySlotBridge:          v.NamedBySlotBridge,
		UnnamedLives:               v.UnnamedLives,
		UnnamedLivesContested:      v.UnnamedLivesContested,
		DeathOffsetMatched:         v.DeathOffsetMatched,
		DeathOffsetRunnerUp:        v.DeathOffsetRunnerUp,
		DeathOffsetMs:              v.DeathOffsetMs,
		ClosedByShot:               v.ClosedByShot,
		ClosedByRespawn:            v.ClosedByRespawn,
		ClosedContested:            v.ClosedContested,
		ClosedRefused:              v.ClosedRefused,
	}
}

func toT0FilmCoverage(v replay.T0FilmCoverage) replaydoc.T0FilmCoverage {
	return replaydoc.T0FilmCoverage{
		Detected: v.Detected,
		Reason:   v.Reason,
		Tracks:   v.Tracks,
		Moving:   v.Moving,
		Burst:    v.Burst,
		MarginMs: v.MarginMs,
	}
}

func toPadDatingStats(v replay.PadDatingStats) replaydoc.PadDatingStats {
	return replaydoc.PadDatingStats{
		Occupations:        v.Occupations,
		Dated:              v.Dated,
		Named:              v.Named,
		Ambiguous:          v.Ambiguous,
		Uncovered:          v.Uncovered,
		PowerupOccupations: v.PowerupOccupations,
	}
}

func toInventoryCoverage(v replay.InventoryCoverage) replaydoc.InventoryCoverage {
	return replaydoc.InventoryCoverage{
		Decoded:             v.Decoded,
		DroppedBeforeOrigin: v.DroppedBeforeOrigin,
		Unpublished:         v.Unpublished,
		Published:           v.Published,
	}
}

func toGrenadeReadCoverage(v replay.GrenadeReadCoverage) replaydoc.GrenadeReadCoverage {
	return replaydoc.GrenadeReadCoverage{
		FromKeyframe: v.FromKeyframe,
		FromDelta:    v.FromDelta,
		Unpublished:  v.Unpublished,
		AmmoRefused:  v.AmmoRefused,
	}
}

func toAbilityCoverage(v replay.AbilityCoverage) replaydoc.AbilityCoverage {
	return replaydoc.AbilityCoverage{
		Reads:       v.Reads,
		ScanNoise:   v.ScanNoise,
		Unpublished: v.Unpublished,
		Published:   v.Published,
	}
}

func toEquipmentCoverage(v replay.EquipmentCoverage) replaydoc.EquipmentCoverage {
	return replaydoc.EquipmentCoverage{
		TracksTotal:        v.TracksTotal,
		CamoLives:          v.CamoLives,
		CamoEpisodes:       v.CamoEpisodes,
		OvershieldLives:    v.OvershieldLives,
		OvershieldEpisodes: v.OvershieldEpisodes,
		KillsRead:          v.KillsRead,
	}
}

// toStanceCoverage convertit la couverture des ETATS DE MOUVEMENT (schema 66). `ByKind` est
// RECOPIEE : partager la map ferait du document servi une vue sur celle du document stocke.
func toStanceCoverage(v replay.StanceCoverage) replaydoc.StanceCoverage {
	out := replaydoc.StanceCoverage{
		Scanned:               v.Scanned,
		Absent:                v.Absent,
		Records:               v.Records,
		Desyncs:               v.Desyncs,
		Reads:                 v.Reads,
		Intervals:             v.Intervals,
		JumpEpisodes:          v.JumpEpisodes,
		JumpsDerived:          v.JumpsDerived,
		Lives:                 v.Lives,
		TracksTotal:           v.TracksTotal,
		Dropped:               v.Dropped,
		EventPacketsUnlocated: v.EventPacketsUnlocated,
		MapWidths:             v.MapWidths,
	}
	if len(v.ByKind) > 0 {
		out.ByKind = make(map[string]int, len(v.ByKind))
		for k, n := range v.ByKind {
			out.ByKind[k] = n
		}
	}
	return out
}

func toGrappleCoverage(v replay.GrappleCoverage) replaydoc.GrappleCoverage {
	return replaydoc.GrappleCoverage{
		LightReads:    v.LightReads,
		HeavyReads:    v.HeavyReads,
		Pulls:         v.Pulls,
		PullLives:     v.PullLives,
		UnpairedFires: v.UnpairedFires,
		BrokenBodies:  v.BrokenBodies,
	}
}

func toScoreCoverage(v replay.ScoreCoverage) replaydoc.ScoreCoverage {
	return replaydoc.ScoreCoverage{
		TeamIdentity:              v.TeamIdentity,
		Rounds:                    v.Rounds,
		ModeSupported:             v.ModeSupported,
		Truncated:                 v.Truncated,
		Oracle:                    v.Oracle,
		Points:                    v.Points,
		RoundsWritten:             v.RoundsWritten,
		RoundsContradicted:        v.RoundsContradicted,
		RoundsContradictedRecords: v.RoundsContradictedRecords,
		RoundsDecreed:             v.RoundsDecreed,
	}
}

func toWeaponChangeCoverage(v replay.WeaponChangeCoverage) replaydoc.WeaponChangeCoverage {
	return replaydoc.WeaponChangeCoverage{
		Decoded:      v.Decoded,
		Published:    v.Published,
		Restated:     v.Restated,
		BeforeOrigin: v.BeforeOrigin,
		Taken:        v.Taken,
		Dropped:      v.Dropped,
		Swapped:      v.Swapped,
	}
}

func toTranslocationCoverage(v replay.TranslocationCoverage) replaydoc.TranslocationCoverage {
	return replaydoc.TranslocationCoverage{
		Events:       v.Events,
		Published:    v.Published,
		BeforeOrigin: v.BeforeOrigin,
		Unpublished:  v.Unpublished,
		Positioned:   v.Positioned,
	}
}

func toAbilityImpulseCoverage(v replay.AbilityImpulseCoverage) replaydoc.AbilityImpulseCoverage {
	return replaydoc.AbilityImpulseCoverage{
		Reads:           v.Reads,
		Episodes:        v.Episodes,
		Published:       v.Published,
		BeforeOrigin:    v.BeforeOrigin,
		Unpublished:     v.Unpublished,
		NoIdentity:      v.NoIdentity,
		OtherFamily:     v.OtherFamily,
		NoResolver:      v.NoResolver,
		ComponentAbsent: v.ComponentAbsent,
		Scan:            ptrOf(v.Scan, toAbilityImpulseScanCoverage),
	}
}

func toAbilityChargeCoverage(v replay.AbilityChargeCoverage) replaydoc.AbilityChargeCoverage {
	return replaydoc.AbilityChargeCoverage{
		Reads:           v.Reads,
		Published:       v.Published,
		BeforeOrigin:    v.BeforeOrigin,
		Unpublished:     v.Unpublished,
		NoIdentity:      v.NoIdentity,
		OtherFamily:     v.OtherFamily,
		NoResolver:      v.NoResolver,
		ComponentAbsent: v.ComponentAbsent,
	}
}

func toEquipmentChangeCoverage(v replay.EquipmentChangeCoverage) replaydoc.EquipmentChangeCoverage {
	return replaydoc.EquipmentChangeCoverage{
		Decoded:           v.Decoded,
		Published:         v.Published,
		Taken:             v.Taken,
		Spent:             v.Spent,
		Spawned:           v.Spawned,
		BeforeOrigin:      v.BeforeOrigin,
		Lives:             v.Lives,
		MissedEstimate:    v.MissedEstimate,
		CounterJumps:      v.CounterJumps,
		LivesFirstOffSpec: v.LivesFirstOffSpec,
		Repeats:           v.Repeats,
		Recovered:         v.Recovered,
	}
}

// toDecoderCoverage projette le bloc `coverage.decoder` : les quatre revisions de calque, la cle
// du profil, et la classification de l empreinte du registre.
//
// LA SEULE TRADUCTION, ET ELLE EST PLATE : cinq chaines et un sous-bloc optionnel. La parite
// champ par champ est tenue par `parity_test.go` (D-7) — un champ ajoute d un cote sans l autre
// y rougit.
func toDecoderCoverage(v replay.DecoderCoverage) replaydoc.DecoderCoverage {
	return replaydoc.DecoderCoverage{
		SourceRev:  v.SourceRev,
		ProfileRev: v.ProfileRev,
		GrammarRev: v.GrammarRev,
		FactsRev:   v.FactsRev,
		Build:      v.Build,
		Registry:   ptrOf(v.Registry, toRegistryCoverage),
	}
}

func toRegistryCoverage(v replay.RegistryCoverage) replaydoc.RegistryCoverage {
	return replaydoc.RegistryCoverage{
		Fingerprint: v.Fingerprint,
		Status:      v.Status,
		Blocks:      v.Blocks,
		NamedSlots:  v.NamedSlots,
	}
}

func toAbilityImpulseScanCoverage(v replay.AbilityImpulseScanCoverage) replaydoc.AbilityImpulseScanCoverage {
	return replaydoc.AbilityImpulseScanCoverage{
		Records: v.Records,
		WithI57: v.WithI57,
		WithI59: v.WithI59,
		Read:    v.Read,
		Unread:  v.Unread,
		Tag1:    v.Tag1,
	}
}

// toDeathsPathsCoverage projette le bloc `coverage.deathsPaths` : les trois denominateurs de
// CHACUNE des deux voies de lecture des morts. Plate, comme `toDecoderCoverage`, et sous la meme
// parite champ par champ (`parity_test.go`).
func toDeathsPathsCoverage(v replay.DeathsPathsCoverage) replaydoc.DeathsPathsCoverage {
	return replaydoc.DeathsPathsCoverage{
		Walk: toDeathsPathTally(v.Walk),
		Scan: toDeathsPathTally(v.Scan),
	}
}

func toDeathsPathTally(v replay.DeathsPathTally) replaydoc.DeathsPathTally {
	return replaydoc.DeathsPathTally{
		Population: v.Population,
		Matched:    v.Matched,
		Published:  v.Published,
	}
}
