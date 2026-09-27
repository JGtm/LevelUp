package archlint

// film_tri_total_test.go — LE CLIQUET DES TRIS NON TOTAUX DU DECODEUR (lot J10.1, 2026-09-27,
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, DT-9 amende par la decision utilisateur du
// 2026-09-27).
//
// # POURQUOI
//
// `sort.Slice` n'est pas stable, et `sort.Sort` non plus : a cle egale, le rang de deux elements
// est celui que le tri leur donne — stable PAR ACCIDENT sous treize elements (tri par insertion),
// tire par pdqsort au-dela. Quand l'entree est batie en iterant une MAP, ce rang change d'une
// execution a l'autre ; quand elle vient du film, il change des qu'un amont bouge. Dans un
// decodeur dont la sortie est publiee, persistee et comparee a l'octet, un ordre qui ne tient qu'au
// tri est un defaut, pas un detail.
//
// DT-9 fixe la forme : `slices.SortStableFunc` / `slices.SortFunc` a comparateur TOTAL — une chaine
// `cmp.Or` qui finit sur une cle unique, ou, quand aucun champ n'est unique, un tri STABLE sur une
// entree deja dans l'ordre du film (la cle unique est alors ce rang, et le commentaire du tri le
// dit).
//
// # LA DECISION DU 2026-09-27 : CORRECTION CIBLEE + CLIQUET
//
// La mesure d'entree a trouve 231 appels `sort.Slice*` / `sort.Sort` de production dans le
// perimetre (144 `Slice`, 87 `SliceStable`, 149 fichiers ; l'audit en annoncait 43). Le lot J10.1 a
// converti les 42 qui DECIDAIENT D'UNE SORTIE sans cle prouvee unique (releve au rapport du lot, un
// test d'ex aequo par tri corrige). Les 189 restants sont listes NOMMEMENT ci-dessous.
//
// # LA REGLE, EN DEUX LIGNES
//
//	appel hors table     interdit : un NOUVEL appel est rouge (TestTriTotalAucunNouvelAppel)
//	entree de la table   son compte ne peut que BAISSER : une entree dont le compte reel est plus
//	                     bas, ou qui n'existe plus, est rouge (TestTriTotalTableNeFaitQueBaisser)
//
// La cle est « fichier:fonction englobante » (methode : `(*T).Nom`), jamais un numero de ligne :
// une ligne bouge a chaque lot, une fonction non. Un appel hors de toute fonction (initialisation
// de variable de paquet) a pour fonction `(niveau paquet)`.
//
// # PERIMETRE
//
// La PRODUCTION de `film/**` (hors `film/research/` et hors fichiers `//go:build research`),
// `replaybuild`, `filmproc` et `sync/killcollector`. Les tests en sont exclus : un tri de test ne
// publie rien. `sort.Stable` est compte avec `sort.Sort` (meme forme, zero appel le 2026-09-27).

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// racinesTriTotal : le perimetre du cliquet, relatif a apps/go-api.
var racinesTriTotal = []string{
	"internal/games/halo_infinite/film",
	"internal/replaybuild",
	"internal/filmproc",
	"internal/sync/killcollector",
}

// exclusTriTotal : les sous-arbres exclus (instruments de recherche).
var exclusTriTotal = []string{"internal/games/halo_infinite/film/research"}

// plancherFichiersTriTotal : CONTRE UN BALAYAGE MUET. 588 fichiers de production mesures le
// 2026-09-27 ; nettement moins veut dire que l'arborescence a bouge et que le cliquet ne garde plus
// rien.
const plancherFichiersTriTotal = 500

// fonctionsDeTriNonTotal : les fonctions du paquet `sort` que le cliquet compte.
var fonctionsDeTriNonTotal = map[string]bool{"Slice": true, "SliceStable": true, "Sort": true, "Stable": true}

// trisNonTotauxToleresAu20260927 — LA TABLE DATEE (2026-09-27, lot J10.1). Elle ne peut que
// DIMINUER. Conversion du reste : J12 (modernisation neutre, DT-10).
//
// 173 entrees, 189 appels : les 87 `sort.SliceStable` (deterministes sur une entree deterministe,
// mais sans comparateur total ecrit) et les `sort.Slice` dont la cle est prouvee unique, qui
// trient des valeurs, qui alimentent un calcul insensible a l'ordre ou qui vivent hors production
// (verdicts au rapport du lot J10.1).
var trisNonTotauxToleresAu20260927 = map[string]int{
	"internal/games/halo_infinite/film/damagetag/damagetag.go:Labels":                                                    1,
	"internal/games/halo_infinite/film/damagetag/damagetag.go:parseIDs":                                                  1,
	"internal/games/halo_infinite/film/internal/facts/fallback/compteur.go:(*Compteur).Rapport":                          1,
	"internal/games/halo_infinite/film/internal/facts/fallback/repli.go:Table":                                           1,
	"internal/games/halo_infinite/film/internal/facts/killsource/botmeta.go:clesTriees":                                  1,
	"internal/games/halo_infinite/film/internal/facts/killsource/botmeta.go:firstCopyOnly":                               1,
	"internal/games/halo_infinite/film/internal/facts/killsource/calibrate.go:motDePoigneeRetenu":                        1,
	"internal/games/halo_infinite/film/internal/facts/killsource/calibrate.go:oracleLargeurAxe":                          1,
	"internal/games/halo_infinite/film/internal/facts/killsource/feed.go:buildFeed":                                      1,
	"internal/games/halo_infinite/film/internal/facts/killsource/health.go:(*decodeCtx).relaxedProbe":                    1,
	"internal/games/halo_infinite/film/internal/facts/killsource/health.go:(*decodeCtx).walkOutOfCatalogue":              1,
	"internal/games/halo_infinite/film/internal/facts/killsource/hybrid.go:(*pass).kills":                                1,
	"internal/games/halo_infinite/film/internal/facts/killsource/hybrid.go:(*pass).runUnclaimed":                         1,
	"internal/games/halo_infinite/film/internal/facts/killsource/index_motif.go:xuidsNommesParLeFilm":                    1,
	"internal/games/halo_infinite/film/internal/facts/killsource/world.go:keyframeRecs":                                  1,
	"internal/games/halo_infinite/film/internal/facts/objectives/awards.go:LabelPersonalScore":                           2,
	"internal/games/halo_infinite/film/internal/facts/objectives/awards.go:decompose":                                    1,
	"internal/games/halo_infinite/film/internal/facts/objectives/awards.go:unitIndex":                                    1,
	"internal/games/halo_infinite/film/internal/facts/objectives/extract.go:collectCaptureBursts":                        1,
	"internal/games/halo_infinite/film/internal/facts/objectives/extract.go:finalize":                                    1,
	"internal/games/halo_infinite/film/internal/facts/objectives/named.go:sortNamedEvents":                               1,
	"internal/games/halo_infinite/film/internal/facts/objectives/named_series.go:cumulateRounds":                         1,
	"internal/games/halo_infinite/film/internal/facts/objectives/named_series.go:sortedSlotKeys":                         1,
	"internal/games/halo_infinite/film/internal/facts/objectives/round_bounds.go:(RoundBounds).KeptSegments":             1,
	"internal/games/halo_infinite/film/internal/facts/objectives/round_bounds.go:chainedRounds":                          1,
	"internal/games/halo_infinite/film/internal/facts/objectives/score.go:SeriesByRound":                                 1,
	"internal/games/halo_infinite/film/internal/facts/objectives/slotidentity.go:sortIdentifiedEvents":                   1,
	"internal/games/halo_infinite/film/internal/facts/objectives/slotidentity_rounds.go:roundStartsOf":                   1,
	"internal/games/halo_infinite/film/internal/facts/objectives/statborg.go:modeScoreRunsByRound":                       1,
	"internal/games/halo_infinite/film/internal/facts/objectives/statborg.go:sortRecords":                                1,
	"internal/games/halo_infinite/film/internal/grammar/biped_creation.go:motAlternatifModal":                            1,
	"internal/games/halo_infinite/film/internal/grammar/equipment_creation_width.go:EquipmentLifeSpans":                  1,
	"internal/games/halo_infinite/film/internal/grammar/equipment_placements.go:confirmPlacements":                       1,
	"internal/games/halo_infinite/film/internal/grammar/equipment_recovery.go:acceptEquipRecovery":                       1,
	"internal/games/halo_infinite/film/internal/grammar/equipment_recovery.go:buildEquipRecoveryWindows":                 1,
	"internal/games/halo_infinite/film/internal/grammar/keyframe_closure.go:keyframeBornesDe":                            1,
	"internal/games/halo_infinite/film/internal/grammar/keyframe_entity_queue.go:MeasureKeyframeAnchors":                 1,
	"internal/games/halo_infinite/film/internal/grammar/keyframe_ground_weapons.go:WorldObjectPositionsForBand":          1,
	"internal/games/halo_infinite/film/internal/grammar/keyframe_loadout.go:familiesByRecordRecs":                        1,
	"internal/games/halo_infinite/film/internal/grammar/keyframe_record_spans.go:KeyframeRecordSpans":                    1,
	"internal/games/halo_infinite/film/internal/grammar/movement_states.go:sortMovementStates":                           1,
	"internal/games/halo_infinite/film/internal/grammar/movement_states_jump.go:(*movementStateScanner).deriverLesSauts": 1,
	"internal/games/halo_infinite/film/internal/grammar/movement_states_jump.go:vitessesOrdonnees":                       1,
	"internal/games/halo_infinite/film/internal/grammar/navpoint_radial_rises.go:NavpointContiguousRises":                1,
	"internal/games/halo_infinite/film/internal/grammar/navpoint_radial_segments.go:NavpointSegments":                    1,
	"internal/games/halo_infinite/film/internal/grammar/object_deaths.go:dedupObjectDeaths":                              1,
	"internal/games/halo_infinite/film/internal/grammar/object_deaths.go:marchPacketsOf":                                 1,
	"internal/games/halo_infinite/film/internal/grammar/object_deaths_calibrate.go:calibrateFrameConfig":                 1,
	"internal/games/halo_infinite/film/internal/grammar/object_deaths_march.go:newMarchTimeline":                         1,
	"internal/games/halo_infinite/film/internal/grammar/offline_biped_band.go:filledSlotMap":                             1,
	"internal/games/halo_infinite/film/internal/grammar/offline_filters.go:TeleportExemptionsOf":                         1,
	"internal/games/halo_infinite/film/internal/grammar/player_entities.go:(*accumulateurDEntites).doutesPublies":        1,
	"internal/games/halo_infinite/film/internal/grammar/player_entities.go:(*accumulateurDEntites).publier":              1,
	"internal/games/halo_infinite/film/internal/grammar/positions/team.go:assignTeamsBestEffort":                         1,
	"internal/games/halo_infinite/film/internal/grammar/projectiles.go:ScanWorldObjectsForBand":                          2,
	"internal/games/halo_infinite/film/internal/grammar/tir_continu.go:sortContinuousFire":                               1,
	"internal/games/halo_infinite/film/internal/grammar/vehicle_occupancy.go:VehicleKeyframeStates":                      1,
	"internal/games/halo_infinite/film/internal/grammar/vehicle_occupancy_march.go:dedupOccupancy":                       1,
	"internal/games/halo_infinite/film/internal/grammar/weapon_hits.go:PairWeaponHits":                                   1,
	"internal/games/halo_infinite/film/internal/grammar/weaponscan/scanner.go:ScanFormulaANS":                            1,
	"internal/games/halo_infinite/film/internal/grammar/world_object_census.go:ScanWorldObjectKeyframes":                 2,
	"internal/games/halo_infinite/film/internal/source/source.go:newDirSource":                                           1,
	"internal/games/halo_infinite/film/killicon/killicon.go:ResolvedTags":                                                1,
	"internal/games/halo_infinite/film/replay/abilities.go:buildAbilityReads":                                            1,
	"internal/games/halo_infinite/film/replay/bomb_armings.go:bombReadsBySlot":                                           1,
	"internal/games/halo_infinite/film/replay/bomb_arms.go:bombArmCandidates":                                            1,
	"internal/games/halo_infinite/film/replay/bomb_arms.go:bombArmsByXUID":                                               1,
	"internal/games/halo_infinite/film/replay/bomb_stats.go:bombPlayerRows":                                              1,
	"internal/games/halo_infinite/film/replay/bomb_stats.go:sortedBombEvents":                                            1,
	"internal/games/halo_infinite/film/replay/build.go:(*assemblage).ouvrir":                                             1,
	"internal/games/halo_infinite/film/replay/build_objective_objects.go:buildObjectiveObjects":                          1,
	"internal/games/halo_infinite/film/replay/closures.go:sortedClaimSlots":                                              1,
	"internal/games/halo_infinite/film/replay/closures_respawn.go:respawnWindow":                                         1,
	"internal/games/halo_infinite/film/replay/closures_respawn.go:sortedVictims":                                         1,
	"internal/games/halo_infinite/film/replay/death_context.go:mortsParVictime":                                          1,
	"internal/games/halo_infinite/film/replay/document_ability_impulses.go:foldAbilityImpulses":                          2,
	"internal/games/halo_infinite/film/replay/document_birth_loadouts.go:fenetresParSlot":                                1,
	"internal/games/halo_infinite/film/replay/document_birth_loadouts.go:mergeLoadouts":                                  1,
	"internal/games/halo_infinite/film/replay/document_ground_weapon_items.go:gwItemLinkPickups":                         1,
	"internal/games/halo_infinite/film/replay/document_stances.go:sortStances":                                           1,
	"internal/games/halo_infinite/film/replay/document_stances.go:stancesDesCles":                                        2,
	"internal/games/halo_infinite/film/replay/document_weapon_changes.go:spawnSetFrom":                                   2,
	"internal/games/halo_infinite/film/replay/equipment_episodes.go:buildCamoEpisodes":                                   2,
	"internal/games/halo_infinite/film/replay/equipment_episodes.go:buildEquipmentEpisodes":                              1,
	"internal/games/halo_infinite/film/replay/equipment_placement_ends.go:placementEnds":                                 1,
	"internal/games/halo_infinite/film/replay/equipment_placements.go:buildEquipmentPlacements":                          1,
	"internal/games/halo_infinite/film/replay/filmfacts_codec.go:encodeKeyframes":                                        1,
	"internal/games/halo_infinite/film/replay/filmfacts_encode.go:encodeQueue":                                           1,
	"internal/games/halo_infinite/film/replay/filmfacts_statsdepose.go:encodeCouplesTriesParID":                          1,
	"internal/games/halo_infinite/film/replay/filmfacts_statsdepose.go:encodeCouplesTriesParLargeurs":                    1,
	"internal/games/halo_infinite/film/replay/fire_bursts.go:buildFireBursts":                                            1,
	"internal/games/halo_infinite/film/replay/fire_bursts.go:dotationsParSlot":                                           1,
	"internal/games/halo_infinite/film/replay/fire_bursts.go:prisesParSlot":                                              1,
	"internal/games/halo_infinite/film/replay/flag_assign.go:flagGroundTimeline":                                         1,
	"internal/games/halo_infinite/film/replay/flag_carrier_tracks.go:tracksByXUID":                                       1,
	"internal/games/halo_infinite/film/replay/flag_carries.go:deathTimesByXUID":                                          1,
	"internal/games/halo_infinite/film/replay/flag_carries.go:flagOpenings":                                              1,
	"internal/games/halo_infinite/film/replay/flag_carries.go:sortFlagOpenings":                                          1,
	"internal/games/halo_infinite/film/replay/flag_carries.go:timesByRoundSlot":                                          1,
	"internal/games/halo_infinite/film/replay/flag_carries_lives.go:flagReturnTimes":                                     1,
	"internal/games/halo_infinite/film/replay/flag_objects.go:flagFreeLives":                                             1,
	"internal/games/halo_infinite/film/replay/flag_return_gauge.go:bindFlagGauges":                                       1,
	"internal/games/halo_infinite/film/replay/flag_return_gauge.go:flagGaugeSlotsOf":                                     2,
	"internal/games/halo_infinite/film/replay/geometry.go:sortFloats":                                                    1,
	"internal/games/halo_infinite/film/replay/grapple_lines.go:buildGrappleLines":                                        3,
	"internal/games/halo_infinite/film/replay/grenade_reads.go:buildGrenadeReads":                                        1,
	"internal/games/halo_infinite/film/replay/grenades.go:projectileBirths":                                              1,
	"internal/games/halo_infinite/film/replay/ground_weapon_objects.go:padObjects":                                       1,
	"internal/games/halo_infinite/film/replay/ground_weapon_rules.go:gwPadsClusterAssign":                                2,
	"internal/games/halo_infinite/film/replay/ground_weapon_rules.go:gwPadsSortClusters":                                 1,
	"internal/games/halo_infinite/film/replay/held_object_carry.go:BuildHeldObjectCarry":                                 1,
	"internal/games/halo_infinite/film/replay/identity_registry_corps.go:(occupantsDesSlots).slotsDe":                    1,
	"internal/games/halo_infinite/film/replay/identity_registry_creation.go:corpsParSlot":                                1,
	"internal/games/halo_infinite/film/replay/identity_registry_exclusion.go:occupationMaximale":                         2,
	"internal/games/halo_infinite/film/replay/inventory.go:buildInventory":                                               1,
	"internal/games/halo_infinite/film/replay/inventory_dead_readings.go:deathTimesByVictimMS":                           1,
	"internal/games/halo_infinite/film/replay/killpos.go:siegesTries":                                                    1,
	"internal/games/halo_infinite/film/replay/lives_death_offset.go:apparierMortsEtVies":                                 1,
	"internal/games/halo_infinite/film/replay/lives_death_offset.go:voteDeathOffsets":                                    1,
	"internal/games/halo_infinite/film/replay/lives_decoupe.go:buildLifeSpans":                                           1,
	"internal/games/halo_infinite/film/replay/lives_decoupe.go:creationsParSlot":                                         1,
	"internal/games/halo_infinite/film/replay/loadouts.go:buildLoadouts":                                                 1,
	"internal/games/halo_infinite/film/replay/mapvar/points_apparition.go:SpawnPointTypeIDs":                             1,
	"internal/games/halo_infinite/film/replay/mapvar/points_apparition.go:SpawnPoints":                                   1,
	"internal/games/halo_infinite/film/replay/mapvar/socles.go:PadSpots":                                                 1,
	"internal/games/halo_infinite/film/replay/objectives.go:buildObjectiveActions":                                       1,
	"internal/games/halo_infinite/film/replay/objectives_catalog.go:(MapObjectivesEntry).PointsOfRole":                   1,
	"internal/games/halo_infinite/film/replay/objectives_catalog.go:sortZonesSpatially":                                  1,
	"internal/games/halo_infinite/film/replay/occupants_presence.go:fusionnerLesPresences":                               1,
	"internal/games/halo_infinite/film/replay/player_index.go:rosterOf":                                                  1,
	"internal/games/halo_infinite/film/replay/positions_porte.go:axesDesPositions":                                       3,
	"internal/games/halo_infinite/film/replay/positions_porte_vehicules.go:ecarterVehiculesHorsEmprise":                  1,
	"internal/games/halo_infinite/film/replay/projectiles.go:buildProjectiles":                                           1,
	"internal/games/halo_infinite/film/replay/roster_bots_successeurs.go:admettreLesBotsSuccesseurs":                     1,
	"internal/games/halo_infinite/film/replay/score_timeline_players.go:buildPlayerScoresFlat":                           1,
	"internal/games/halo_infinite/film/replay/sieges.go:ordreDesArrivants":                                               1,
	"internal/games/halo_infinite/film/replay/sieges_places.go:(*poseDesPlaces).bornerAuSuccesseur":                      1,
	"internal/games/halo_infinite/film/replay/sieges_tirs.go:balayerLesBornes":                                           1,
	"internal/games/halo_infinite/film/replay/t0_film.go:t0FilmDepartures":                                               1,
	"internal/games/halo_infinite/film/replay/t0_film.go:t0FilmSteps":                                                    1,
	"internal/games/halo_infinite/film/replay/tracks_publication.go:bornesDesVies":                                       1,
	"internal/games/halo_infinite/film/replay/usage_summary.go:(*usageTallies).rows":                                     1,
	"internal/games/halo_infinite/film/replay/usage_summary_owners.go:usageSlotOwners":                                   2,
	"internal/games/halo_infinite/film/replay/vehicle_cycles.go:buildVehicleCycles":                                      1,
	"internal/games/halo_infinite/film/replay/vehicle_cycles.go:clusterVehicleSpawns":                                    1,
	"internal/games/halo_infinite/film/replay/vehicle_cycles.go:vehicleCycleGaps":                                        1,
	"internal/games/halo_infinite/film/replay/vehicle_relays.go:mergeVehicleRelay":                                       1,
	"internal/games/halo_infinite/film/replay/vehicle_rides.go:vehicleEventsByOccupant":                                  1,
	"internal/games/halo_infinite/film/replay/vehicle_rides.go:vehicleGaps":                                              2,
	"internal/games/halo_infinite/film/replay/vehicle_rides.go:vehicleNearestWithin":                                     1,
	"internal/games/halo_infinite/film/replay/vehicle_rides_aim.go:vehicleAimBySlot":                                     1,
	"internal/games/halo_infinite/film/replay/vehicle_rides_build.go:(*vehicleRideBuild).trier":                          1,
	"internal/games/halo_infinite/film/replay/vehicle_rides_events.go:vehicleEventEpisodes":                              1,
	"internal/games/halo_infinite/film/replay/vehicle_rides_film.go:(vehicleFilmRides).trier":                            1,
	"internal/games/halo_infinite/film/replay/vehicle_rides_film.go:vehicleOccupancyBySlot":                              1,
	"internal/games/halo_infinite/film/replay/vehicle_shots.go:attachVehicleShots":                                       1,
	"internal/games/halo_infinite/film/replay/vehicle_shots.go:vehicleShotCandidates":                                    1,
	"internal/games/halo_infinite/film/replay/vehicle_tracks.go:sortVehicleTracks":                                       1,
	"internal/games/halo_infinite/film/replay/vehicle_tracks.go:vehiclePositionsBySlot":                                  1,
	"internal/games/halo_infinite/film/replay/vehicle_turrets.go:moveTurretRides":                                        1,
	"internal/games/halo_infinite/film/replay/vip_crown.go:vipSelectionOpenings":                                         1,
	"internal/games/halo_infinite/film/replay/zone_attribution.go:samplesByXUID":                                         1,
	"internal/games/halo_infinite/film/replay/zone_states.go:sortedZoneSlots":                                            1,
	"internal/games/halo_infinite/film/replay/zone_states.go:zoneRampsOf":                                                1,
	"internal/games/halo_infinite/film/replay/zone_states.go:zoneSeriesOf":                                               1,
	"internal/games/halo_infinite/film/replay/zone_states_gauge.go:zoneGaugeRampsOf":                                     1,
	"internal/games/halo_infinite/film/replay/zone_states_gauge.go:zoneGaugeSeriesOf":                                    1,
	"internal/games/halo_infinite/film/replay/zone_states_hill.go:buildRampHills":                                        1,
	"internal/games/halo_infinite/film/replay/zone_states_hill_owners.go:hillStatesOf":                                   1,
	"internal/games/halo_infinite/film/replay/zone_states_lettres.go:zoneLetterRanks":                                    1,
	"internal/games/halo_infinite/film/replay/zone_states_owner.go:electZoneOwners":                                      1,
	"internal/games/halo_infinite/film/replay/zoom_state.go:finsDeVieParSlot":                                            1,
	"internal/sync/killcollector/shots.go:sortedWeaponIDs":                                                               1,
}

func TestTriTotalAucunNouvelAppel(t *testing.T) {
	vus := balayerTrisNonTotaux(t)
	for _, cle := range clesDeTriTriees(vus) {
		n, tolere := vus[cle], trisNonTotauxToleresAu20260927[cle]
		if n <= tolere {
			continue
		}
		t.Errorf("%s : %d appel(s) sort.Slice/SliceStable/Sort/Stable, %d tolere(s) au 2026-09-27.\n"+
			"Un NOUVEL appel est interdit (DT-9) : slices.SortStableFunc / slices.SortFunc a "+
			"comparateur TOTAL — une chaine cmp.Or qui finit sur une cle unique. Ajouter une entree "+
			"a trisNonTotauxToleresAu20260927 N'EST PAS une reponse : la table ne fait que baisser.",
			cle, n, tolere)
	}
}

func TestTriTotalTableNeFaitQueBaisser(t *testing.T) {
	vus := balayerTrisNonTotaux(t)
	for _, cle := range clesDeTriTriees(trisNonTotauxToleresAu20260927) {
		tolere, n := trisNonTotauxToleresAu20260927[cle], vus[cle]
		switch {
		case n == 0:
			t.Errorf("%s est dans la table mais ne porte plus aucun appel (converti, renomme ou "+
				"deplace) : RETIRER l'entree — une entree perimee finit par autoriser n'importe quoi.", cle)
		case n < tolere:
			t.Errorf("%s : %d appel(s) pour %d tolere(s) — BAISSER l'entree a %d (cliquet).", cle, n, tolere, n)
		}
	}
}

// balayerTrisNonTotaux rend, par « fichier:fonction », le nombre d'appels comptes.
func balayerTrisNonTotaux(t *testing.T) map[string]int {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(ici))) // .../apps/go-api
	out := map[string]int{}
	fichiers := 0
	for _, racine := range racinesTriTotal {
		base := filepath.Join(goAPIRoot, filepath.FromSlash(racine))
		err := filepath.WalkDir(base, func(chemin string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(goAPIRoot, chemin)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				if chemin != base && repertoireExcluDuTriTotal(d.Name(), rel) {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(chemin, ".go") || strings.HasSuffix(chemin, "_test.go") {
				return nil
			}
			compte, err := compterTrisDuFichier(chemin, rel, out)
			fichiers += compte
			return err
		})
		if err != nil {
			t.Fatalf("balayage de %s : %v", base, err)
		}
	}
	if fichiers < plancherFichiersTriTotal {
		t.Fatalf("balayage muet : %d fichiers de production vus dans %v, plancher %d.",
			fichiers, racinesTriTotal, plancherFichiersTriTotal)
	}
	return out
}

// repertoireExcluDuTriTotal : repertoires caches, `_x`, `testdata` et sous-arbres de recherche.
func repertoireExcluDuTriTotal(nom, rel string) bool {
	if strings.HasPrefix(nom, ".") || strings.HasPrefix(nom, "_") || nom == "testdata" {
		return true
	}
	for _, ex := range exclusTriTotal {
		if rel == ex {
			return true
		}
	}
	return false
}

// compterTrisDuFichier ajoute a `out` les appels du fichier ; rend 1 si le fichier est compte
// (production hors tag research), 0 sinon.
func compterTrisDuFichier(chemin, rel string, out map[string]int) (int, error) {
	blob, err := os.ReadFile(chemin) //nolint:gosec // chemin derive du perimetre
	if err != nil {
		return 0, err
	}
	if estSousTagResearch(blob) {
		return 0, nil
	}
	f, err := parser.ParseFile(token.NewFileSet(), chemin, blob, 0)
	if err != nil {
		return 0, err
	}
	nomSort := nomDImportDeSort(f)
	switch nomSort {
	case "", "_":
		return 1, nil
	case ".":
		out[rel+":(import point de sort)"]++
		return 1, nil
	}
	for _, decl := range f.Decls {
		englobante := "(niveau paquet)"
		if fd, ok := decl.(*ast.FuncDecl); ok {
			englobante = nomDeFonction(fd)
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			if estAppelDeTriNonTotal(n, nomSort) {
				out[rel+":"+englobante]++
			}
			return true
		})
	}
	return 1, nil
}

// nomDImportDeSort rend le nom sous lequel le fichier importe `sort`, ou "" s'il ne l'importe pas.
func nomDImportDeSort(f *ast.File) string {
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) != "sort" {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return "sort"
	}
	return ""
}

// estAppelDeTriNonTotal dit si le noeud est un appel `sort.Slice`, `sort.SliceStable`,
// `sort.Sort` ou `sort.Stable` (sous le nom d'import du fichier).
func estAppelDeTriNonTotal(n ast.Node, nomSort string) bool {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == nomSort && fonctionsDeTriNonTotal[sel.Sel.Name]
}

func clesDeTriTriees(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
