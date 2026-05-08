// Package sync — refdata_personal_scores.go : table de référence pour les
// PersonalScores Halo Infinite.
//
// Portage de src/data/domain/_refdata_personal_scores.py (Python LevelUp).
// Source : Halo Waypoint API — chaque NameId est un FNV1a32 stable du nom EN
// du score (KILLED_PLAYER, FLAG_CAPTURED, etc).
//
// Trois maps fournies :
//   - psaTechnicalIDs : NameId numérique → identifiant snake_case stocké
//     dans personal_score_awards.award_name (ex: 1024030246 → "killed_player").
//   - psaPoints       : valeur de points par occurrence (utilisé en fallback
//     quand TotalPersonalScoreAwarded est absent du JSON).
//   - psaCategory     : catégorie ("kill", "assist", "objective", "vehicle",
//     "penalty", "other") stockée dans personal_score_awards.award_category.
package sync

// IDs canoniques (FNV1a32 du nom EN — stables côté API Halo).
const (
	psaKilledPlayer           uint64 = 1024030246
	psaBetrayedPlayer         uint64 = 911992497
	psaSelfDestruction        uint64 = 249491819
	psaEliminatedPlayer       uint64 = 2408971842
	psaRevivedPlayer          uint64 = 3428202435
	psaReviveDenied           uint64 = 2130209372
	psaKillAssist             uint64 = 638246808
	psaMarkAssist             uint64 = 152718958
	psaSensorAssist           uint64 = 1267013266
	psaEMPAssist              uint64 = 221060588
	psaDriverAssist           uint64 = 963594075
	psaFlagCaptured           uint64 = 601966503
	psaFlagStolen             uint64 = 3002710045
	psaFlagReturned           uint64 = 22113181
	psaFlagTaken              uint64 = 2387185397
	psaFlagCaptureAssist      uint64 = 555570945
	psaRunnerStopped          uint64 = 316828380
	psaBallControl            uint64 = 454168309
	psaBallTaken              uint64 = 204144695
	psaCarrierStopped         uint64 = 746397417
	psaHillControl            uint64 = 340198991
	psaHillScored             uint64 = 1032565232
	psaZoneCaptured50         uint64 = 3507884073
	psaZoneCaptured75         uint64 = 4026987576
	psaZoneCaptured100        uint64 = 757037588
	psaZoneSecured            uint64 = 709346128
	psaPowerSeedSecured       uint64 = 2188620691
	psaPowerSeedStolen        uint64 = 3996338664
	psaCarrierKilled          uint64 = 4128329646
	psaStockpileScored        uint64 = 2801241965
	psaExtractionInitiated    uint64 = 1825517751
	psaExtractionConverted    uint64 = 1117301492
	psaExtractionCompleted    uint64 = 4130011565
	psaExtractionDenied       uint64 = 1552628741
	psaConversionDenied       uint64 = 4247243561
	psaCollectedBonusXP       uint64 = 522435689
	psaHackedTerminal         uint64 = 665081740
	psaDestroyedBanshee       uint64 = 597066859
	psaDestroyedChopper       uint64 = 3472794399
	psaDestroyedFalcon        uint64 = 395875864
	psaDestroyedGhost         uint64 = 4254982885
	psaDestroyedGungoose      uint64 = 2107631925
	psaDestroyedMongoose      uint64 = 1416267372
	psaDestroyedPhantom       uint64 = 2742351765
	psaDestroyedRazorback     uint64 = 1661163286
	psaDestroyedRocketWarthog uint64 = 2008690931
	psaDestroyedScorpion      uint64 = 3454330054
	psaDestroyedWarthog       uint64 = 3107879375
	psaDestroyedWasp          uint64 = 2106274556
	psaDestroyedWraith        uint64 = 3243589708
	psaHijackedBanshee        uint64 = 3150095814
	psaHijackedChopper        uint64 = 1059880024
	psaHijackedFalcon         uint64 = 586857799
	psaHijackedGhost          uint64 = 1614285349
	psaHijackedGungoose       uint64 = 4186766732
	psaHijackedMongoose       uint64 = 2191528998
	psaHijackedRazorback      uint64 = 2848565291
	psaHijackedRocketWarthog  uint64 = 4294405210
	psaHijackedWarthog        uint64 = 1834653062
	psaHijackedWasp           uint64 = 674964649
	psaCustom                 uint64 = 4294967295
)

// psaTechnicalIDs : NameId → identifiant snake_case (stocké dans award_name).
// Les 3 variantes ZONE_CAPTURED partagent le même technical id (zone_captured)
// pour faciliter les agrégats UI / citations.
var psaTechnicalIDs = map[uint64]string{
	psaKilledPlayer:           "killed_player",
	psaBetrayedPlayer:         "betrayed_player",
	psaSelfDestruction:        "self_destruction",
	psaEliminatedPlayer:       "eliminated_player",
	psaRevivedPlayer:          "revived_player",
	psaReviveDenied:           "revive_denied",
	psaKillAssist:             "kill_assist",
	psaMarkAssist:             "mark_assist",
	psaSensorAssist:           "sensor_assist",
	psaEMPAssist:              "emp_assist",
	psaDriverAssist:           "driver_assist",
	psaFlagCaptured:           "flag_captured",
	psaFlagStolen:             "flag_stolen",
	psaFlagReturned:           "flag_returned",
	psaFlagTaken:              "flag_taken",
	psaFlagCaptureAssist:      "flag_capture_assist",
	psaRunnerStopped:          "runner_stopped",
	psaBallControl:            "ball_control",
	psaBallTaken:              "ball_taken",
	psaCarrierStopped:         "carrier_stopped",
	psaHillControl:            "hill_control",
	psaHillScored:             "hill_scored",
	psaZoneCaptured50:         "zone_captured",
	psaZoneCaptured75:         "zone_captured",
	psaZoneCaptured100:        "zone_captured",
	psaZoneSecured:            "zone_secured",
	psaPowerSeedSecured:       "power_seed_secured",
	psaPowerSeedStolen:        "power_seed_stolen",
	psaCarrierKilled:          "carrier_killed",
	psaStockpileScored:        "stockpile_scored",
	psaExtractionInitiated:    "extraction_initiated",
	psaExtractionConverted:    "extraction_converted",
	psaExtractionCompleted:    "extraction_completed",
	psaExtractionDenied:       "extraction_denied",
	psaConversionDenied:       "conversion_denied",
	psaCollectedBonusXP:       "collected_bonus_xp",
	psaHackedTerminal:         "hacked_terminal",
	psaDestroyedBanshee:       "destroyed_banshee",
	psaDestroyedChopper:       "destroyed_chopper",
	psaDestroyedFalcon:        "destroyed_falcon",
	psaDestroyedGhost:         "destroyed_ghost",
	psaDestroyedGungoose:      "destroyed_gungoose",
	psaDestroyedMongoose:      "destroyed_mongoose",
	psaDestroyedPhantom:       "destroyed_phantom",
	psaDestroyedRazorback:     "destroyed_razorback",
	psaDestroyedRocketWarthog: "destroyed_rocket_warthog",
	psaDestroyedScorpion:      "destroyed_scorpion",
	psaDestroyedWarthog:       "destroyed_warthog",
	psaDestroyedWasp:          "destroyed_wasp",
	psaDestroyedWraith:        "destroyed_wraith",
	psaHijackedBanshee:        "hijacked_banshee",
	psaHijackedChopper:        "hijacked_chopper",
	psaHijackedFalcon:         "hijacked_falcon",
	psaHijackedGhost:          "hijacked_ghost",
	psaHijackedGungoose:       "hijacked_gungoose",
	psaHijackedMongoose:       "hijacked_mongoose",
	psaHijackedRazorback:      "hijacked_razorback",
	psaHijackedRocketWarthog:  "hijacked_rocket_warthog",
	psaHijackedWarthog:        "hijacked_warthog",
	psaHijackedWasp:           "hijacked_wasp",
	psaCustom:                 "custom",
}

// psaPoints : NameId → points par occurrence (fallback si TotalPersonalScoreAwarded absent).
var psaPoints = map[uint64]int{
	psaKilledPlayer:           100,
	psaBetrayedPlayer:         -100,
	psaSelfDestruction:        -100,
	psaEliminatedPlayer:       200,
	psaRevivedPlayer:          100,
	psaReviveDenied:           25,
	psaKillAssist:             50,
	psaMarkAssist:             10,
	psaSensorAssist:           10,
	psaEMPAssist:              50,
	psaDriverAssist:           50,
	psaFlagCaptured:           300,
	psaFlagStolen:             25,
	psaFlagReturned:           25,
	psaFlagTaken:              10,
	psaFlagCaptureAssist:      100,
	psaRunnerStopped:          25,
	psaBallControl:            50,
	psaBallTaken:              10,
	psaCarrierStopped:         25,
	psaHillControl:            25,
	psaHillScored:             100,
	psaZoneCaptured50:         50,
	psaZoneCaptured75:         75,
	psaZoneCaptured100:        100,
	psaZoneSecured:            25,
	psaPowerSeedSecured:       100,
	psaPowerSeedStolen:        50,
	psaCarrierKilled:          10,
	psaStockpileScored:        150,
	psaExtractionInitiated:    50,
	psaExtractionConverted:    50,
	psaExtractionCompleted:    200,
	psaExtractionDenied:       25,
	psaConversionDenied:       25,
	psaCollectedBonusXP:       300,
	psaHackedTerminal:         100,
	psaDestroyedBanshee:       50,
	psaDestroyedChopper:       50,
	psaDestroyedFalcon:        75,
	psaDestroyedGhost:         50,
	psaDestroyedGungoose:      25,
	psaDestroyedMongoose:      25,
	psaDestroyedPhantom:       100,
	psaDestroyedRazorback:     50,
	psaDestroyedRocketWarthog: 50,
	psaDestroyedScorpion:      100,
	psaDestroyedWarthog:       50,
	psaDestroyedWasp:          50,
	psaDestroyedWraith:        100,
	psaHijackedBanshee:        25,
	psaHijackedChopper:        25,
	psaHijackedFalcon:         25,
	psaHijackedGhost:          25,
	psaHijackedGungoose:       25,
	psaHijackedMongoose:       25,
	psaHijackedRazorback:      25,
	psaHijackedRocketWarthog:  25,
	psaHijackedWarthog:        25,
	psaHijackedWasp:           25,
}

// psaCategory : NameId → catégorie pour personal_score_awards.award_category.
// Catégories : "kill", "assist", "objective", "vehicle", "penalty", "other".
var psaCategory = map[uint64]string{
	psaKilledPlayer:           "kill",
	psaEliminatedPlayer:       "kill",
	psaCarrierKilled:          "kill",
	psaKillAssist:             "assist",
	psaMarkAssist:             "assist",
	psaSensorAssist:           "assist",
	psaEMPAssist:              "assist",
	psaDriverAssist:           "assist",
	psaFlagCaptureAssist:      "assist",
	psaFlagCaptured:           "objective",
	psaFlagStolen:             "objective",
	psaFlagReturned:           "objective",
	psaFlagTaken:              "objective",
	psaRunnerStopped:          "objective",
	psaBallControl:            "objective",
	psaBallTaken:              "objective",
	psaCarrierStopped:         "objective",
	psaHillControl:            "objective",
	psaHillScored:             "objective",
	psaZoneCaptured50:         "objective",
	psaZoneCaptured75:         "objective",
	psaZoneCaptured100:        "objective",
	psaZoneSecured:            "objective",
	psaPowerSeedSecured:       "objective",
	psaPowerSeedStolen:        "objective",
	psaStockpileScored:        "objective",
	psaExtractionInitiated:    "objective",
	psaExtractionConverted:    "objective",
	psaExtractionCompleted:    "objective",
	psaExtractionDenied:       "objective",
	psaHackedTerminal:         "objective",
	psaDestroyedBanshee:       "vehicle",
	psaDestroyedChopper:       "vehicle",
	psaDestroyedFalcon:        "vehicle",
	psaDestroyedGhost:         "vehicle",
	psaDestroyedGungoose:      "vehicle",
	psaDestroyedMongoose:      "vehicle",
	psaDestroyedPhantom:       "vehicle",
	psaDestroyedRazorback:     "vehicle",
	psaDestroyedRocketWarthog: "vehicle",
	psaDestroyedScorpion:      "vehicle",
	psaDestroyedWarthog:       "vehicle",
	psaDestroyedWasp:          "vehicle",
	psaDestroyedWraith:        "vehicle",
	psaBetrayedPlayer:         "penalty",
	psaSelfDestruction:        "penalty",
	// "other" : revive, hijacks, conversion_denied, collected_bonus_xp, custom.
}

// categorizePSA retourne la catégorie d'un NameId (fallback "other").
func categorizePSA(nameID uint64) string {
	if c, ok := psaCategory[nameID]; ok {
		return c
	}
	return "other"
}

// technicalIDForPSA retourne l'identifiant snake_case d'un NameId. Pour les
// IDs inconnus (futurs ajouts API), retourne "" — l'extracteur skip alors.
func technicalIDForPSA(nameID uint64) string {
	return psaTechnicalIDs[nameID]
}
