package replaybuild

import (
	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/port"
)

// buildReplayOptions assemble replay.Options a partir de ce que BuildBytes a deja resolu
// (catalogue d'entrees, faits du match, entrees de catalogue, statistiques du film) — extrait
// de BuildBytes (lot restes R0, deplacement pur des champs, aucun changement de sortie).
func (b *Builder) buildReplayOptions(
	entry filmdec.MapQuantEntry, facts port.MatchFacts, cat entreesCatalogue, stats *filmStats,
) replay.Options {
	return replay.Options{
		FrameIntervalMS: b.interval,
		Geometry:        b.geometry,
		Structure:       b.structureFor(entry.Module),
		Labels:          b.labels,
		NeutralDeaths:   cat.neutral,
		Kills:           cat.kills,
		MatchKills:      cat.matchKills,
		RosterXUIDs:     rosterXUIDs(facts),
		Participants:    participantsDuTableau(facts),
		Bots:            cat.bots,
		Successions:     cat.successions,
		Objectives:      stats.objectives,
		// LE COMPTE DES ECARTES VOYAGE AVEC LES ACTIONS, ET IL EST LE DENOMINATEUR (constat C1
		// de la revue VIES-R1). Sans lui, `coverage.objectives.available` compte les seuls
		// RESCAPES du pont d'identite et `noSlot` reste structurellement a zero — le defaut
		// meme que le lot declare corriger. Mesure : `c0a82e88`, 17 actions nommees par le
		// film, 12 identifiees par le pont par manche.
		ObjectivesUnnamed: stats.objectivesUnnamed,
		// LE REFUS D EFFECTIF VOYAGE AUSSI, et il est un DENOMINATEUR de meme nature : le
		// calque se tait, et `coverage.objectives.refusedByRoster` dit combien d actions ce
		// silence coute (129 sur les trois films BTB du parc).
		ObjectivesRefused: stats.objectivesRefused,
		// LE PONT DU STATBORG VOYAGE POUR ETRE PUBLIE, PAS POUR ETRE REFAIT : le registre
		// d'identite en tire `identity.statborgSlots` avec la provenance de chaque couple.
		StatborgIdentity: stats.statborgIdentity,
		Score:            stats.score,
		Flag:             stats.flag,
		Vip:              stats.vip,
		Skull:            stats.skull,
		Bomb:             stats.bomb,
		Zone: replay.ZoneInput{Zones: cat.zones, Roles: cat.zoneRoles, TeamByXUID: teamByXUID(facts),
			Hill: isHillVariant(facts.GameVariantName)},
		MapQuant:         &entry,
		Observe:          b.observe,
		SpawnPoints:      cat.spawnPts,
		SpawnPointsState: cat.spawnPointsState,
	}
}
