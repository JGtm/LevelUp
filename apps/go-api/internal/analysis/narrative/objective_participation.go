package narrative

// objective_participation.go — index de participation aux objectifs PAR OPPORTUNITÉ
// (plan .ai/PLAN_AXE_OBJECTIFS_INDEX.md, étape 3).
//
// Remplace, pour l'axe « Objectifs » du profil de participation, l'ancienne somme
// PSA catégorie 'objective' diluée par le nombre TOTAL de matchs (Slayer inclus).
// L'index est construit sur les colonnes de `match_objective_stats_latest`
// (shared_matches_v2) : sous-score r_f par famille de mode calculé sur les SEULS
// matchs de la famille, agrégé pondéré par n_f, clampé à ObjectiveIndexClip.
//
// Pour une famille f et un scope de matchs (session, fenêtre, escouade ou 1 match) :
//
//	actions_f = Σ_col (w_col × Σ stat_col) / n_f      (cadence pondérée par match)
//	hold_f    = Σ durées_f / Σ time_played_f          (part du temps sur l'objectif)
//	r_f       = min(1.25, 0.65×actions_f/P80_actions + 0.35×hold_f/P80_hold)
//	            (extraction : pas de durée → r_f = min(1.25, actions_f/P80_actions))
//	raw       = Σ_f (n_f × r_f) / n_obj               (n_obj = Σ_f n_f)
//
// Un joueur au P80 de sa famille marque r_f = 1.0 → 80/100 après normalisation par
// ObjectiveIndexThreshold (idiome intensif P80×1.25, cf. ScorePerMinuteP80).
// n_obj == 0 → l'axe est RETIRÉ par le caller (pas de 0 trompeur).
//
// AUCUN I/O ici : les agrégats (sommes par colonne, durées, time_played) sont fournis
// par la couche repo (platform/duckdb) ou service ; la normalisation 0..100 finale
// reste ComputeParticipationProfile (réutilisée, pas dupliquée).

// ObjectiveFamily identifie une famille de modes à objectif de
// match_objective_stats. Le split zones_koth / zones_strongholds se fait par ligne
// (zone_scoring_ticks > 0 ⇒ KOTH), décision 3 du plan.
type ObjectiveFamily string

// Familles supportées (clés stables, alignées sur les blocs de la table).
const (
	FamilyCTF              ObjectiveFamily = "ctf"
	FamilyZonesKOTH        ObjectiveFamily = "zones_koth"
	FamilyZonesStrongholds ObjectiveFamily = "zones_strongholds"
	FamilyOddball          ObjectiveFamily = "oddball"
	FamilyStockpile        ObjectiveFamily = "stockpile"
	FamilyExtraction       ObjectiveFamily = "extraction"
	FamilyVIP              ObjectiveFamily = "vip"
)

// AllObjectiveFamilies retourne les 7 familles dans un ordre stable.
func AllObjectiveFamilies() []ObjectiveFamily {
	return []ObjectiveFamily{
		FamilyCTF, FamilyZonesKOTH, FamilyZonesStrongholds, FamilyOddball,
		FamilyStockpile, FamilyExtraction, FamilyVIP,
	}
}

// ObjectiveColKillsAsVIP est la seule colonne d'action à dénominateur spécifique :
// sa contribution est divisée par max(1, Σ times_selected_as_vip) au lieu de n_f
// (décision 7 du plan — kills en tant que VIP, rapportés au nombre de sélections).
// Seule colonne EXPORTÉE : son traitement particulier doit être nommable depuis les
// producteurs d'agrégats (service, repo). Les autres noms de colonnes n'ont pas à
// franchir la frontière du package — leur cohérence avec les producteurs est
// verrouillée par garde-rail (TestObjectiveIndexRepo_WeightKeysSubsetOfSchema,
// TestObjectiveIndexInputFromScoreboard_KeysMatchWeights).
const ObjectiveColKillsAsVIP = "kills_as_vip"

// Noms de colonnes de match_objective_stats consommés par les deux tables
// ci-dessous (poids d'action, colonnes de durée). Constantes et non littéraux :
// plusieurs colonnes sont partagées par deux familles (bloc zones : KOTH et
// Strongholds portent les mêmes 4 actions), et ces tables sont la SOURCE UNIQUE
// d'où la couche repo génère ses SUM — un nom y est un contrat, pas une chaîne
// d'agrément. Valeurs figées : elles doivent rester égales aux noms de colonnes
// physiques de la table.
const (
	// Bloc CTF.
	objectiveColFlagCaptures        = "flag_captures"
	objectiveColFlagCaptureAssists  = "flag_capture_assists"
	objectiveColFlagSteals          = "flag_steals"
	objectiveColFlagSecures         = "flag_secures"
	objectiveColFlagReturns         = "flag_returns"
	objectiveColFlagCarriersKilled  = "flag_carriers_killed"
	objectiveColFlagReturnersKilled = "flag_returners_killed"
	objectiveColTimeAsFlagCarrier   = "time_as_flag_carrier_seconds"

	// Bloc zones (KOTH + Strongholds).
	objectiveColZoneCaptures       = "zone_captures"
	objectiveColZoneSecures        = "zone_secures"
	objectiveColZoneOffensiveKills = "zone_offensive_kills"
	objectiveColZoneDefensiveKills = "zone_defensive_kills"
	objectiveColZoneScoringTicks   = "zone_scoring_ticks"
	objectiveColTimeInZones        = "time_in_zones_seconds"

	// Bloc oddball.
	objectiveColSkullGrabs          = "skull_grabs"
	objectiveColSkullCarriersKilled = "skull_carriers_killed"
	objectiveColSkullScoringTicks   = "skull_scoring_ticks"
	objectiveColTimeAsSkullCarrier  = "time_as_skull_carrier_seconds"

	// Bloc stockpile.
	objectiveColPowerSeedsDeposited     = "power_seeds_deposited"
	objectiveColPowerSeedsStolen        = "power_seeds_stolen"
	objectiveColPowerSeedCarriersKilled = "power_seed_carriers_killed"
	objectiveColTimeAsPowerSeedCarrier  = "time_as_power_seed_carrier_seconds"
	objectiveColTimeAsPowerSeedDriver   = "time_as_power_seed_driver_seconds"

	// Bloc extraction (aucune durée — décision 8).
	objectiveColExtractionConversionsCompleted = "extraction_conversions_completed"
	objectiveColSuccessfulExtractions          = "successful_extractions"
	objectiveColExtractionInitiationsCompleted = "extraction_initiations_completed"
	objectiveColExtractionConversionsDenied    = "extraction_conversions_denied"

	// Bloc VIP (kills_as_vip : cf. ObjectiveColKillsAsVIP ci-dessus).
	objectiveColVipKills   = "vip_kills"
	objectiveColVipAssists = "vip_assists"
	objectiveColTimeAsVip  = "time_as_vip_seconds"
)

// ObjectiveFamilyActionWeights : poids w_col par colonne d'action, par famille.
// Clés = noms de colonnes de match_objective_stats (source unique consommée par la
// couche repo pour construire ses SUM — ne pas créer de liste de colonnes parallèle).
// Poids repris 1:1 de config/titles/halo_infinite/mappings/awards.toml quand un
// équivalent existe ; colonnes sans équivalent : décision 7 du plan. Colonnes
// écartées (signal nul ou nature inadaptée) : kills_as_flag_carrier,
// kills_as_flag_returner, kills_as_skull_carrier, kills_as_power_seed_carrier,
// longest_time_as_*, max_killing_spree_as_vip, times_selected_as_vip (dénominateur),
// extraction_initiations_denied, flag_grabs.
var ObjectiveFamilyActionWeights = map[ObjectiveFamily]map[string]float64{
	FamilyCTF: {
		objectiveColFlagCaptures:        5.0,
		objectiveColFlagCaptureAssists:  2.0,
		objectiveColFlagSteals:          2.0,
		objectiveColFlagSecures:         1.0,
		objectiveColFlagReturns:         1.5,
		objectiveColFlagCarriersKilled:  1.5,
		objectiveColFlagReturnersKilled: 1.5,
	},
	FamilyZonesKOTH: {
		objectiveColZoneCaptures:       3.0,
		objectiveColZoneSecures:        2.0,
		objectiveColZoneOffensiveKills: 1.0,
		objectiveColZoneDefensiveKills: 1.0,
		objectiveColZoneScoringTicks:   0.5, // KOTH seulement (0 structurel en Strongholds)
	},
	FamilyZonesStrongholds: {
		objectiveColZoneCaptures:       3.0,
		objectiveColZoneSecures:        2.0,
		objectiveColZoneOffensiveKills: 1.0,
		objectiveColZoneDefensiveKills: 1.0,
	},
	FamilyOddball: {
		objectiveColSkullGrabs:          1.5,
		objectiveColSkullCarriersKilled: 1.5,
		objectiveColSkullScoringTicks:   0.5,
	},
	FamilyStockpile: {
		objectiveColPowerSeedsDeposited:     3.0,
		objectiveColPowerSeedsStolen:        2.0,
		objectiveColPowerSeedCarriersKilled: 1.5,
	},
	FamilyExtraction: {
		objectiveColExtractionConversionsCompleted: 3.0,
		objectiveColSuccessfulExtractions:          3.0,
		objectiveColExtractionInitiationsCompleted: 1.5,
		objectiveColExtractionConversionsDenied:    2.0,
	},
	FamilyVIP: {
		objectiveColVipKills:   3.0,
		objectiveColVipAssists: 1.0,
		ObjectiveColKillsAsVIP: 2.0, // ÷ max(1, Σ times_selected_as_vip), pas ÷ n_f
	},
}

// ObjectiveFamilyHoldColumns : colonnes de durée (secondes) du terme hold_f par
// famille (décision 8). Extraction : aucune durée → pas de terme hold.
// VIP : time_as_vip_seconds est normalisé par sélection (÷ max(1, Σ selections))
// dans ComputeObjectiveIndex, pas ici.
var ObjectiveFamilyHoldColumns = map[ObjectiveFamily][]string{
	FamilyCTF:              {objectiveColTimeAsFlagCarrier},
	FamilyZonesKOTH:        {objectiveColTimeInZones},
	FamilyZonesStrongholds: {objectiveColTimeInZones},
	FamilyOddball:          {objectiveColTimeAsSkullCarrier},
	FamilyStockpile:        {objectiveColTimeAsPowerSeedCarrier, objectiveColTimeAsPowerSeedDriver},
	FamilyExtraction:       {},
	FamilyVIP:              {objectiveColTimeAsVip},
}

// ObjectiveIndexClip est le plafond du sous-score r_f et de l'index brut : +25 %
// au-dessus du repère P80 sature (idiome ScorePerMinuteP80 × 1.25).
const ObjectiveIndexClip = 1.25

// ObjectiveIndexThreshold est le seuil à passer à ParticipationThresholds.Objective
// quand l'axe est alimenté par ComputeObjectiveIndex : un joueur au P80 marque
// 80/100, le clip 1.25 marque 100.
const ObjectiveIndexThreshold = ObjectiveIndexClip

// Pondérations relatives des deux termes du sous-score r_f (familles avec durée).
const (
	objectiveActionsTermWeight = 0.65
	objectiveHoldTermWeight    = 0.35
)

// objectiveFamilyCalibration porte les repères P80 d'une famille. holdP80 == 0 ⇒
// pas de terme hold (extraction) : r_f = actions_f / actionsP80.
type objectiveFamilyCalibration struct {
	actionsP80 float64
	holdP80    float64
}

// objectiveCalibrations : repères P80 par famille, mesurés localement le 2026-07-26
// (diag_q sur match_objective_stats_latest ⋈ match_participants, population
// match-joueur, time_played_seconds > 30, hors bots xuid LIKE 'bid(%').
// actions = Σ w_col × col par ligne ; hold = durée_f / time_played_seconds par ligne.
//
//	ctf               n=3419 (234 matchs)  actions p50=3    p80=11   p90=16.5
//	                                        hold    p50=.0016 p80=.0434 p90=.0807
//	zones_strongholds n=1639 (136 matchs)  actions p50=15   p80=29   p90=36
//	                                        hold    p50=.1075 p80=.1809 p90=.2194
//	zones_koth        n=361  (47 matchs)   actions p50=35.5 p80=51   p90=60.5
//	                                        hold    p50=.1348 p80=.2178 p90=.2591
//
// PROVISIONAL (re-mesurer dès que la famille atteint ≥ 30 matchs distincts en base
// locale après reset MBitObjectiveStats + cmd/backfill_objective_stats — backfill
// impossible le 2026-07-26 : shared DB tenue par le serveur dev, cf. plan étape 1) :
//
//	oddball    n=57 (7 matchs, sous le seuil) : valeurs MESURÉES retenues telles
//	           quelles — actions p50=21.5 p80=35 p90=43.1 ; hold p50=.0614 p80=.1187
//	           p90=.1651.
//	stockpile  aucune ligne : analogie porteur-objet (cadence CTF × porteurs
//	           multiples ; hold entre oddball et strongholds).
//	extraction aucune ligne : cadence estimée (2-3 conversions/extractions + 1-2
//	           initiations par bon match) ; pas de hold (décision 8).
//	vip        aucune ligne : vip_kills ~ zone_captures en fréquence ; temps VIP
//	           par sélection ~ hold oddball.
var objectiveCalibrations = map[ObjectiveFamily]objectiveFamilyCalibration{
	FamilyCTF:              {actionsP80: 11.0, holdP80: 0.0434},
	FamilyZonesStrongholds: {actionsP80: 29.0, holdP80: 0.1809},
	FamilyZonesKOTH:        {actionsP80: 51.0, holdP80: 0.2178},
	FamilyOddball:          {actionsP80: 35.0, holdP80: 0.1187}, // provisional
	FamilyStockpile:        {actionsP80: 20.0, holdP80: 0.10},   // provisional
	FamilyExtraction:       {actionsP80: 12.0, holdP80: 0},      // provisional, sans hold
	FamilyVIP:              {actionsP80: 18.0, holdP80: 0.12},   // provisional
}

// ObjectiveFamilyStats agrège les entrées d'une famille sur le scope étudié.
type ObjectiveFamilyStats struct {
	// Matches = n_f : nombre de lignes match-joueur de la famille dans le scope.
	Matches int
	// ColumnSums : Σ par colonne d'action (clés ⊆ ObjectiveFamilyActionWeights[f] —
	// les colonnes inconnues de la table de poids sont ignorées).
	ColumnSums map[string]float64
	// HoldSeconds : Σ des colonnes ObjectiveFamilyHoldColumns[f] (secondes).
	HoldSeconds float64
	// TimePlayedSeconds : Σ time_played_seconds des matchs de la famille.
	TimePlayedSeconds float64
	// TimesSelectedAsVIP : Σ times_selected_as_vip (famille vip uniquement) —
	// dénominateur de kills_as_vip et de time_as_vip_seconds.
	TimesSelectedAsVIP float64
}

// ObjectiveIndexInput : agrégats par famille pour un scope donné.
type ObjectiveIndexInput map[ObjectiveFamily]ObjectiveFamilyStats

// ComputeObjectiveIndex calcule l'index brut (raw ∈ [0, ObjectiveIndexClip]) et
// n_obj, le nombre de matchs à objectif du scope. n_obj == 0 ⇒ (0, 0) : le caller
// RETIRE l'axe Objectif (patron dropUncomputableRadarAxes / skipSurvivalAxis).
// Le raw se normalise en 0..100 via ComputeParticipationProfile avec
// Thresholds.Objective = ObjectiveIndexThreshold.
func ComputeObjectiveIndex(in ObjectiveIndexInput) (float64, int) {
	var weighted float64
	var nObj int
	for _, fam := range AllObjectiveFamilies() {
		st, ok := in[fam]
		if !ok || st.Matches <= 0 {
			continue
		}
		r := objectiveFamilySubScore(fam, st)
		weighted += float64(st.Matches) * r
		nObj += st.Matches
	}
	if nObj == 0 {
		return 0, 0
	}
	return weighted / float64(nObj), nObj
}

// objectiveFamilySubScore calcule r_f ∈ [0, ObjectiveIndexClip] pour une famille.
func objectiveFamilySubScore(fam ObjectiveFamily, st ObjectiveFamilyStats) float64 {
	cal := objectiveCalibrations[fam]
	if cal.actionsP80 <= 0 {
		return 0
	}
	n := float64(st.Matches)

	// actions_f : cadence pondérée par match ; kills_as_vip rapporté aux sélections.
	var actions float64
	for col, w := range ObjectiveFamilyActionWeights[fam] {
		v := st.ColumnSums[col]
		if v <= 0 {
			continue
		}
		if col == ObjectiveColKillsAsVIP {
			actions += w * v / max(1, st.TimesSelectedAsVIP)
			continue
		}
		actions += w * v / n
	}

	r := actions / cal.actionsP80
	if cal.holdP80 > 0 {
		hold := objectiveFamilyHold(fam, st)
		r = objectiveActionsTermWeight*actions/cal.actionsP80 +
			objectiveHoldTermWeight*hold/cal.holdP80
	}
	if r > ObjectiveIndexClip {
		r = ObjectiveIndexClip
	}
	if r < 0 {
		r = 0
	}
	return r
}

// objectiveFamilyHold calcule hold_f ∈ [0, ~1] : part du temps de jeu passée sur
// l'objectif. VIP : la durée est d'abord ramenée par sélection (décision 8), puis
// rapportée à la durée moyenne d'un match du scope.
func objectiveFamilyHold(fam ObjectiveFamily, st ObjectiveFamilyStats) float64 {
	if st.TimePlayedSeconds <= 0 {
		return 0
	}
	if fam == FamilyVIP {
		avgMatchSeconds := st.TimePlayedSeconds / float64(st.Matches)
		if avgMatchSeconds <= 0 {
			return 0
		}
		perSelection := st.HoldSeconds / max(1, st.TimesSelectedAsVIP)
		return perSelection / avgMatchSeconds
	}
	return st.HoldSeconds / st.TimePlayedSeconds
}
