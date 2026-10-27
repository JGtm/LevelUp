package ops

// seed_citation_data.go — table de donnees: mapping medaille -> citation.
// Littéral de données (exempt de la regle 80L/fonction : data table).

// defaultCitationMappings — 110 règles citations (88 portées de l'ancien
// scripts/populate_citation_mappings.py, + 10 citations d'objectif v7.2.1,
// + 8 « Artilleur de » véhicule le 2026-09-10, + 4 citations Halo 5 le 2026-10-16).
//
// Catégories : Mode de jeu, Multijoueur, Spartan Companies, Véhicule, Arme,
// Ennemi (PVE) + composites.
//
// Source de vérité unique (le script Python a été supprimé avec la migration) :
// toute modification impose un re-seed + un backfill citations (docs/COMMENDATIONS.md).
// Les noms et descriptions ne sont PAS ici : ils sont une donnée par langue
// (citation_texts.toml, clé = Norm).
//
// littéral `return []CitationMapping{...}` : funlen mord sur les lignes, pas les
// statements). maintidx n'est pas activé dans .golangci.yml au 2026-07 (absent de
// linters.enable et du set "standard") — la directive est donc inerte aujourd'hui ;
// conservée pour documenter l'intention si le linter est un jour activé, plutôt
// que de retirer sans certitude et risquer un futur gate rouge silencieux.
//
//nolint:funlen,maintidx // Données de seed — préfère lisibilité linéaire (un seul
func defaultCitationMappings() []CitationMapping {
	// Phase 6.5 (2026-04-28, commit a1d25325) a renommé les sous-dossiers vers
	// les slugs canoniques longs. Tout chemin émis ici doit utiliser ces slugs,
	// sinon le seed UPSERT écrase image_path avec un chemin 404.
	const wpH5 = "static/commendations/halo_5_guardians/"
	const wpHI = "static/commendations/halo_infinite/"
	return []CitationMapping{
		// ── PVP — Mode de jeu (10) ────────────────────────────────────
		// v7.2 (décision utilisateur) — BASCULE DES 4 CITATIONS D'OBJECTIF vers la
		// source objective_stat (colonnes match_objective_stats_latest), plus fiable
		// que les awards personal_score : charge→zone_captures, got_you→flag_returns,
		// stakeholder→zone_secures, flag_carrier_hunter→flag_carriers_killed. La
		// progression est recalculée depuis l'historique au backfill citations
		// (--citations-recompute-all) : re-seed puis rejeu requis (local + prod).
		{Norm: citationNormCharge, MappingType: mappingTypeObjectiveStat,
			StatName: "zone_captures", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_À_la_charge.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets10_20_30_50_100},
		{Norm: "forced_annexation", MappingType: mappingTypeCustom,
			CustomFunction: "compute_annexion_forcee", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Annexion_forcée.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets3_6_9_15_30},
		{Norm: "assistant", MappingType: mappingTypeStat,
			StatName: "assists", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Assistant.png",
			Category:    citationCatModeJeu,
			TierTargets: "25,50,75,125,250"},
		// I7 (2026-07-24) — DÉCISION UTILISATEUR : flag_defender DÉSACTIVÉE (Enabled=false).
		// Aucun award d'ingestion (refdata_personal_scores.go) ne mesure la défense de SON
		// PROPRE drapeau de façon non ambiguë : "carrier_killed" (seul candidat direct) veut
		// dire « tuer le porte-drapeau ENNEMI » = flag_carrier_hunter, pas la défense. La
		// citation reste listée (inventaire + parité EN) mais le moteur l'ignore
		// (loadFullCitationMappings : WHERE enabled IS NOT FALSE). Candidat futur si un award
		// dédié apparaît : "carrier_stopped" (objective — stopper le porteur ennemi de VOTRE
		// drapeau). Non enfant d'un composite → aucun composite rendu inatteignable.
		{Norm: "flag_defender", MappingType: mappingTypeAward,
			AwardName: "carrier_killed", AwardCategory: awardCategoryObjective, Enabled: false,
			ImagePath:   wpH5 + "H5G_citation_Défenseur_du_drapeau.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets10_20_30_50_100},
		{Norm: citationNormGotYou, MappingType: mappingTypeObjectiveStat,
			StatName: "flag_returns", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Je_te_tiens_!.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets10_20_30_50_100},
		{Norm: citationNormStakeholder, MappingType: mappingTypeObjectiveStat,
			StatName: "zone_secures", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Partie_prenante.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets10_20_30_50_100},
		{Norm: citationNormFlagCarrierHunter, MappingType: mappingTypeObjectiveStat,
			StatName: "flag_carriers_killed", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Sus_au_porteur_du_drapeau.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets10_20_30_50_100},
		{Norm: "flag_victory", MappingType: mappingTypeCustom,
			CustomFunction: "compute_wins_ctf", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Victoire_au_drapeau.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets5_10_15_25_50},
		{Norm: "slayer_victory", MappingType: mappingTypeCustom,
			CustomFunction: "compute_wins_slayer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Victoire_en_Assassin.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets5_10_15_25_50},
		{Norm: "strongholds_victory", MappingType: mappingTypeCustom,
			CustomFunction: "compute_wins_strongholds", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Victoire_en_Bases.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets5_10_15_25_50},

		{Norm: "tick_tick_boom", MappingType: mappingTypeCustom,
			CustomFunction: "compute_wins_assault", Enabled: true,
			ImagePath:   wpHI + "HI_citation_Tick_Tick_Boom.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets5_10_15_25_50},
		{Norm: "wonderful_toys", MappingType: mappingTypeCustom,
			CustomFunction: "compute_wonderful_toys", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Jouets_éducatifs.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets5_10_20_35_60},

		// ── PVP — Mode de jeu · objectifs CTF/Zones/Oddball (10, v7.2.1) ──────
		// DÉCISION UTILISATEUR (2026-07-25) : sur les 19 colonnes de
		// match_objective_stats restées libres après la bascule v7.2, 10 sont retenues
		// et 9 écartées (« elles n'en valent pas la peine »). Ne pas re-proposer les 9.
		// Paliers calibrés sur les données réelles — cf. le bloc de constantes
		// tierTargets d'objectif (seed.go) : NE PAS les réaligner sur les paliers
		// génériques, quatre de ces citations deviendraient inatteignables.
		// Ordre du bloc : CTF, puis Zones, puis Oddball — celui d'objectiveStatColumns
		// (internal/sync/citations.go), et non l'ordre alphabétique du bloc précédent.
		//
		// VISUELS (2026-09-10) — le blocage « SVG bouche-trous » du plan V721-03 est LEVÉ :
		// les deux SVG générés en interne ont été supprimés. « Capture du drapeau » sert
		// désormais le PNG définitif fourni par l'utilisateur (redimensionné à 100x100,
		// la taille de tous les autres visuels de citation) ; « Vol du drapeau » n'a plus
		// de visuel du tout car la citation est désactivée (voir son entrée).
		// 2026-09-19 : le PNG de « Capture du drapeau » portait un damier de transparence
		// CUIT dans les pixels (fond blanc/gris à l'écran) — fond retiré, alpha réel.
		// « Crâne intouchable » pointait par erreur un visuel d'ARME Halo 5 (Éradicateur) ;
		// Halo 5 n'a aucune citation Oddball : silhouette de crâne fournie par l'utilisateur,
		// recadrée 100x100 avec une bordure noire de 2,5 px (HI_citation_Crane_intouchable.png).
		{Norm: citationNormFlagCaptures, MappingType: mappingTypeObjectiveStat,
			StatName: "flag_captures", Enabled: true,
			ImagePath:   wpHI + "HI_citation_Capture_du_drapeau.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets10_25_50_75_125},
		{Norm: citationNormFlagSecures, MappingType: mappingTypeObjectiveStat,
			StatName: "flag_secures", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Défenseur_du_drapeau.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets50_100_200_350_600},
		// DÉCISION UTILISATEUR (2026-09-10) : « Vol du drapeau » DÉSACTIVÉE (Enabled=false).
		// La citation reste listée (inventaire + parité EN) mais le moteur l'ignore
		// (loadFullCitationMappings : WHERE enabled IS NOT FALSE). ImagePath vidé et SVG
		// bouche-trou supprimé du dépôt : aucun visuel n'a jamais été produit pour elle et
		// une citation désactivée n'est jamais rendue (TestCitationEnabled_HasImagePath
		// n'exige un visuel que sur les citations actives). Non enfant d'un composite →
		// aucun composite rendu inatteignable. La colonne `flag_steals` reste utilisée
		// ailleurs (KPI, radar de match, séries temporelles) : rien à retirer côté stats.
		{Norm: citationNormFlagSteals, MappingType: mappingTypeObjectiveStat,
			StatName: "flag_steals", Enabled: false,
			Category:    citationCatModeJeu,
			TierTargets: tierTargets25_50_100_175_300},
		// DÉCISION UTILISATEUR (2026-09-19) : « Chasse au rapatrieur » DÉSACTIVÉE
		// (Enabled=false) — citation maison (nom et définition inventés le 2026-07-25, seule
		// la colonne est native), jugée source de confusion et sans intérêt à suivre. Même
		// patron que « Vol du drapeau » : listée (inventaire + parité EN), ignorée par le
		// moteur, ImagePath vidé (le visuel H5 « Not so fast » redevient libre), non enfant
		// d'un composite. La colonne `flag_returners_killed` reste utilisée ailleurs (rôles
		// d'objectif, radar de match) : seule la citation est retirée.
		{Norm: citationNormReturnerTakedown, MappingType: mappingTypeObjectiveStat,
			StatName: "flag_returners_killed", Enabled: false,
			Category:    citationCatModeJeu,
			TierTargets: tierTargets5_10_20_35_60},
		{Norm: citationNormUnstoppableCarrier, MappingType: mappingTypeObjectiveStat,
			StatName: "kills_as_flag_carrier", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Imparable.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets1_2_3_5_10},
		{Norm: citationNormAggressiveReturn, MappingType: mappingTypeObjectiveStat,
			StatName: "kills_as_flag_returner", Enabled: true,
			ImagePath:   wpHI + "HI_citation_Rapatriement_agressif.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets5_10_20_30_50},
		{Norm: citationNormZoneDefense, MappingType: mappingTypeObjectiveStat,
			StatName: "zone_defensive_kills", Enabled: true,
			ImagePath:   wpHI + "HI_citation_Defense_de_zone.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets25_50_100_200_350},
		{Norm: citationNormUntouchableCarrier, MappingType: mappingTypeObjectiveStat,
			StatName: "kills_as_skull_carrier", Enabled: true,
			ImagePath:   wpHI + "HI_citation_Crane_intouchable.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets1_2_3_5_10},
		{Norm: citationNormSkullCarrierTakedown, MappingType: mappingTypeObjectiveStat,
			StatName: "skull_carriers_killed", Enabled: true,
			ImagePath:   wpHI + "HI_citation_Chasse_au_porteur.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets2_5_10_20_40},
		{Norm: citationNormSkullGrabs, MappingType: mappingTypeObjectiveStat,
			StatName: "skull_grabs", Enabled: true,
			ImagePath:   wpHI + "HI_citation_Prise_du_crane.png",
			Category:    citationCatModeJeu,
			TierTargets: tierTargets2_5_10_20_40},

		// ── PVP — Véhicule (2) + Grenade (2) ──────────────────────────
		{Norm: "splatter", MappingType: mappingTypeMedal, MedalID: 221693153, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Écrasement.png",
			Category:    citationCatVehicule,
			TierTargets: tierTargets10_20_30_50_100, Subcategory: citationSubGeneral},
		// BUG C (I7, 2026-07-24) : `driver` et `road_trip` pointaient tous deux la
		// médaille 3169118333 (« Violence routière »/vehicle-kill). Le nom EN de
		// cette citation est « Wheelman » (cf. citation_texts.toml) = médaille de
		// pilote-assist (categorie "assists" dans medal_category_table.go:120),
		// medal_id 2926348688 → remap ici. Distinct de 3169118333 (vehicles), ce
		// qui lève la collision côté `driver`.
		{Norm: "driver", MappingType: mappingTypeMedal, MedalID: 2926348688, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Pilote.png",
			Category:    citationCatVehicule,
			TierTargets: tierTargets10_20_30_50_100, Subcategory: citationSubGeneral},

		// ── PVP — Artilleur de véhicule (6) ───────────────────────────
		// AJOUT 2026-09-10 (demande utilisateur). Ces six citations comptent les frags
		// TIRÉS depuis l'armement d'un châssis — ce que le catalogue 343i ne récompense
		// nulle part : `splatter`/`road_trip`/`lawnmower` comptent l'ÉCRASEMENT, `driver`
		// la médaille de pilote-assist, les `*_destroyer` la DESTRUCTION d'un engin.
		//
		// La mesure ne coûte rien à produire : `weapon_stat` est alimenté depuis la
		// SOURCE DE DÉGÂT DU FILM (sync/citations_weapons_source.go, bascule du
		// 2026-09-01), qui nomme le châssis par sa racine de banque sonore. Les clés
		// hinf_ghost..hinf_wraith sont déjà au registre et déjà verrouillées hors
		// arsenal (games/weapons/off_arsenal_guard_test.go) : aucune n'émet de record
		// de dégât 0xd2, donc aucune n'est visible de la voie `weapon_kills` — le
		// non-double-comptage est une propriété constatée, pas une règle appliquée.
		//
		// SIX CHÂSSIS ET PAS PLUS, et pourquoi les autres manquent :
		//   - Warthog et Warthog gauss : leur artilleur tire depuis une TOURELLE, qui
		//     porte sa propre racine (`tur_un_machinegun`, `tur_un_gausscannon`). Le
		//     film nomme l'ARME, jamais le PORTEUR — une citation « Artilleur de
		//     Warthog » compterait donc aussi les tourelles fixes. Écartée comme
		//     libellé faux (décision utilisateur du 2026-09-10). La lever demande de
		//     croiser avec l'occupation du véhicule (`biped_board_vehicle`), soit un
		//     chantier de décodage, pas un branchement.
		//   - Gungoose, Mantis, Phaéton : aucune racine de banque mesurée.
		//   - Chopper, Phantom, Pélican et les tourelles : mesurables, mais aucune
		//     image « Artilleur de » en stock — à rouvrir le jour où l'image existe.
		//
		// StatName = nom canonique EN du registre (weapon_names.toml), pas la clé :
		// c'est ce que loadWeaponKillsFromSource dépose dans ctx.Stats.
		{Norm: "ghost_gunner", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Ghost", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Artilleur_de_ghost.png",
			Category:    citationCatVehicule,
			TierTargets: tierTargets5_10_15_25_50, Subcategory: citationSubGeneral},
		{Norm: "banshee_gunner", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Banshee", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Artilleur_de_banshee.png",
			Category:    citationCatVehicule,
			TierTargets: tierTargets5_10_15_25_50, Subcategory: citationSubGeneral},
		// « Apparition » est le libellé FR arrêté pour le Wraith (weapon_names.toml,
		// décision D14) — l'image H5 et la citation `wraith_destroyer` le disent déjà.
		{Norm: "wraith_gunner", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Wraith", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Artilleur_de_l'apparition.png",
			Category:    citationCatVehicule,
			TierTargets: tierTargets5_10_15_25_50, Subcategory: citationSubGeneral},
		{Norm: "scorpion_gunner", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Scorpion", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Artilleur_de_scorpion.png",
			Category:    citationCatVehicule,
			TierTargets: tierTargets5_10_15_25_50, Subcategory: citationSubGeneral},
		{Norm: "wasp_gunner", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Wasp", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Artilleur_de_wasp.png",
			Category:    citationCatVehicule,
			TierTargets: tierTargets5_10_15_25_50, Subcategory: citationSubGeneral},
		{Norm: "rockethog_gunner", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Rockethog", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Artilleur_de_warthog_lance-roquettes.png",
			Category:    citationCatVehicule,
			TierTargets: tierTargets5_10_15_25_50, Subcategory: citationSubGeneral},
		// AJOUT 2026-09-10, second temps. Ces deux-là manquaient au premier passage parce que
		// leur source n'était pas atteignable : le Warthog tire depuis une TOURELLE dont la
		// racine de banque est partagée avec la tourelle fixe de carte, et le Gungoose n'a
		// AUCUNE banque sonore. Le lot kill feed `PORTEUR` (fusionné le même jour) a posé un
		// genre de règle indexé sur le tag `vehi` du porteur, qui lève les deux cas d'un coup
		// et pose `hinf_warthog` / `hinf_gungoose` au registre. Les deux images « Artilleur
		// de » étaient déjà au dépôt depuis le catalogue Halo 5.
		{Norm: "warthog_gunner", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Warthog", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Artilleur_de_warthog.png",
			Category:    citationCatVehicule,
			TierTargets: tierTargets5_10_15_25_50, Subcategory: citationSubGeneral},
		{Norm: "gungoose_gunner", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Gungoose", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Artilleur_de_gungoose.png",
			Category:    citationCatVehicule,
			TierTargets: tierTargets5_10_15_25_50, Subcategory: citationSubGeneral},

		{Norm: "frag_grenade", MappingType: mappingTypeMedal, MedalID: 2648272972, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Grenade_à_fragmentation.png",
			Category:    citationCatArme,
			TierTargets: tierTargets5_10_15_25_50, Subcategory: "Grenade"},
		{Norm: "plasma_grenade", MappingType: mappingTypeMedal, MedalID: 3655682764, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Grenade_à_plasma.png",
			Category:    citationCatArme,
			TierTargets: "2,4,6,10,20", Subcategory: "Grenade"},

		// ── PVP — Multijoueur (11) ────────────────────────────────────
		// DÉCISION UTILISATEUR (2026-10-26) : « Bulldozer » passe de Mode de jeu à Multijoueur.
		{Norm: "bulldozer", MappingType: mappingTypeCustom,
			CustomFunction: "compute_bulldozer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Bulldozer.png",
			Category:    citationCatMultijoueur,
			TierTargets: tierTargets3_6_9_15_30},
		{Norm: "assassin", MappingType: mappingTypeMedal, MedalID: 548533137, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Assassin.png",
			Category:    citationCatMultijoueur,
			TierTargets: tierTargets5_10_15_25_50},
		{Norm: "first_strike", MappingType: mappingTypeCustom,
			CustomFunction: "compute_first_strike", Enabled: true,
			ImagePath:   wpHI + "HI_citation_Premier_coup.png",
			Category:    citationCatMultijoueur,
			TierTargets: tierTargets10_25_50_100_250},
		{Norm: "almost_doesnt_count", MappingType: mappingTypeStat,
			StatName: "shieldless_kills", Enabled: true,
			ImagePath:   wpHI + "HI_citation_Presque_ca_ne_compte_pas.png",
			Category:    citationCatMultijoueur,
			TierTargets: tierTargets50_100_150_250_500},
		{Norm: "spartan_carnage", MappingType: mappingTypeMedal,
			MedalIDs:    "2780740615,4261842076,418532952,1486797009,710323196,1720896992,2567026752,2875941471",
			Enabled:     true,
			ImagePath:   wpH5 + "H5G_citation_Carnage_de_Spartans.png",
			Category:    citationCatMultijoueur,
			TierTargets: tierTargets3_6_9_15_30},
		{Norm: "close_combat", MappingType: mappingTypeStat,
			StatName: "melee_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Combat_rapproché.png",
			Category:    citationCatMultijoueur,
			TierTargets: tierTargets10_20_30_50_100},
		{Norm: "opportunist", MappingType: mappingTypeMedal,
			MedalIDs:    "622331684,2063152177,4261842076,2137071619,1486797009,1430343434,2242633421",
			Enabled:     true,
			ImagePath:   wpH5 + "H5G_citation_Combattant_opportuniste.png",
			Category:    citationCatMultijoueur,
			TierTargets: tierTargets10_20_30_50_100},
		{Norm: "multikill", MappingType: mappingTypeMedal, MedalID: 622331684, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Multifrag.png",
			Category:    citationCatMultijoueur,
			TierTargets: tierTargets3_6_9_15_30},
		{Norm: "melee_fighter", MappingType: mappingTypeStat,
			StatName: "melee_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Pugilat.png",
			Category:    citationCatMultijoueur,
			TierTargets: tierTargets5_10_15_25_50},
		{Norm: "headshot", MappingType: mappingTypeStat,
			StatName: "headshot_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Tir_à_la_tête.png",
			Category:    citationCatMultijoueur,
			TierTargets: tierTargets10_20_30_50_100},
		{Norm: "spartan_killer", MappingType: mappingTypeStat,
			StatName: "kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Tueur_de_Spartans.png",
			Category:    citationCatMultijoueur,
			TierTargets: "20,40,60,100,200"},
		{Norm: "eagle_eye", MappingType: mappingTypeMedal, MedalID: 1512363953, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Œil_de_lynx.png",
			Category:    citationCatMultijoueur,
			TierTargets: tierTargets10_20_30_50_100},
		{Norm: "avenger", MappingType: mappingTypeMedal, MedalID: 9000000001, Enabled: true,
			ImagePath:   wpH5 + "Assassin_commendation.png",
			Category:    citationCatMultijoueur,
			TierTargets: "5,15,30,55,105"},

		// ── PVP — Spartan Companies (15) ──────────────────────────────
		{Norm: "flag_em_down", MappingType: mappingTypeCustom,
			CustomFunction: "compute_flag_em_down", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Flag_'em_down.png",
			Category:    citationCatSpartanCompanies,
			TierTargets: "1000,2000,3000,4800,9700"},
		{Norm: "grand_theft", MappingType: mappingTypeCustom,
			CustomFunction: "compute_hijack", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Grand_Theft.png",
			Category:    citationCatSpartanCompanies,
			TierTargets: "200,400,600,960,1940"},
		{Norm: "helping_hand", MappingType: mappingTypeStat,
			StatName: "assists", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Helping_Hand.png",
			Category:    citationCatSpartanCompanies,
			TierTargets: "20000,40000,60000,96000,194400"},
		{Norm: "im_just_perfect", MappingType: mappingTypeMedal, MedalID: 1512363953, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_I'm_just_perfect.png",
			Category:    citationCatSpartanCompanies,
			TierTargets: "2000,4000,6000,9600,19400"},
		{Norm: "lawnmower", MappingType: mappingTypeMedal, MedalID: 221693153, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Lawnmower.png",
			Category:    citationCatSpartanCompanies,
			TierTargets: "500,1000,1500,2400,4900"},
		{Norm: "look_ma_no_pin", MappingType: mappingTypeStat,
			StatName: "grenade_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Look_ma_no_pin.png",
			Category:    citationCatSpartanCompanies,
			TierTargets: "4000,8000,12000,19200,38900"},
		{Norm: "lucky", MappingType: mappingTypeMedal,
			MedalIDs: "3905838030,3091261182", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Lucky.png",
			Category:    citationCatSpartanCompanies,
			TierTargets: "400,800,1200,1920,3880"},
		{Norm: "no_hard_feelings", MappingType: mappingTypeStat,
			StatName: "kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_No_Hard_Feelings.png",
			Category:    citationCatSpartanCompanies,
			TierTargets: "50000,100000,150000,240000,486000"},
		{Norm: "positive_contribution", MappingType: mappingTypeCustom,
			CustomFunction: "compute_bulldozer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Positive_contribution.png",
			Category:    citationCatSpartanCompanies,
			TierTargets: "300,600,900,1440,2960"},
		{Norm: "power_play", MappingType: mappingTypeStat,
			StatName: "power_weapon_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Power_play.png",
			Category:    citationCatSpartanCompanies,
			TierTargets: "10000,20000,30000,48000,97200"},
		// I7 (2026-07-24) — DÉCISION UTILISATEUR : road_trip remappée sur la médaille
		// d'ÉCRASEMENT (Splatter) 221693153, présente au catalogue Infinite — faisceau
		// concordant : medal_category_table.go la classe "vehicles" ; la citation `splatter`
		// (Display « Écrasement », EN « Splatter ») pointe déjà cette médaille ; asset_labels.toml
		// mappe "Splatter"="Écrasement". L'ancien medal_id 3169118333 (« Violence routière »)
		// n'est plus référencé par aucune citation. Duplication de médaille intentionnelle
		// (comme splatter/lawnmower) : road_trip conserve son palier Spartan Company élevé
		// (grind long), splatter/lawnmower gardent les leurs. Non enfant d'un composite
		// (vehicle_mastery cible `splatter`, pas `road_trip`) → aucun composite impacté.
		{Norm: "road_trip", MappingType: mappingTypeMedal, MedalID: 221693153, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Road_Trip.png",
			Category:    citationCatSpartanCompanies,
			TierTargets: "3000,6000,9000,14400,29200"},
		{Norm: "sting_like_a_bee", MappingType: mappingTypeStat,
			StatName: "melee_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Sting_like_a_bee.png",
			Category:    citationCatSpartanCompanies,
			TierTargets: "5000,10000,15000,24000,48600"},
		{Norm: "the_reaper", MappingType: mappingTypeMedal, MedalID: 2625820422, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_The_Reaper.png",
			Category:    citationCatSpartanCompanies,
			TierTargets: "500,1000,1500,2400,4850"},
		{Norm: "too_fast_for_you", MappingType: mappingTypeMedal, MedalID: 2123530881, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Too_fast_for_you.png",
			Category:    citationCatSpartanCompanies,
			TierTargets: "2000,4000,6000,9600,19400"},
		{Norm: "vandalism", MappingType: mappingTypeCustom,
			CustomFunction: "compute_vandalism", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Vandalism.png",
			Category:    citationCatSpartanCompanies,
			TierTargets: "1200,2400,3600,5760,11640"},

		// ── PVP — Véhicules destructeurs (7) ─────────────────────────
		{Norm: "wraith_destroyer", MappingType: mappingTypeCustom,
			CustomFunction: "compute_wraith_destroyer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_d'apparitions.png",
			Category:    citationCatVehicule,
			TierTargets: tierTargets3_6_9_15_30, Subcategory: citationSubCovenant},
		{Norm: "banshee_destroyer", MappingType: mappingTypeAward,
			AwardName: "destroyed_banshee", AwardCategory: awardCategoryVehicle, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_de_banshees.png",
			Category:    citationCatVehicule,
			TierTargets: tierTargets3_6_9_15_30, Subcategory: citationSubCovenant},
		{Norm: "ghost_destroyer", MappingType: mappingTypeAward,
			AwardName: "destroyed_ghost", AwardCategory: awardCategoryVehicle, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_de_ghosts.png",
			Category:    citationCatVehicule,
			TierTargets: tierTargets5_10_15_25_50, Subcategory: citationSubCovenant},
		{Norm: "mongoose_destroyer", MappingType: mappingTypeCustom,
			CustomFunction: "compute_mongoose_destroyer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_de_mongooses.png",
			Category:    citationCatVehicule,
			TierTargets: tierTargets5_10_15_25_50, Subcategory: citationSubUNSC},
		{Norm: "scorpion_destroyer", MappingType: mappingTypeAward,
			AwardName: "destroyed_scorpion", AwardCategory: awardCategoryVehicle, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_de_scorpions.png",
			Category:    citationCatVehicule,
			TierTargets: "1,3,5,7,10", Subcategory: citationSubUNSC},
		{Norm: "warthog_destroyer", MappingType: mappingTypeCustom,
			CustomFunction: "compute_warthog_destroyer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_de_warthogs.png",
			Category:    citationCatVehicule,
			TierTargets: tierTargets5_10_15_25_50, Subcategory: citationSubUNSC},
		{Norm: "wasp_destroyer", MappingType: mappingTypeAward,
			AwardName: "destroyed_wasp", AwardCategory: awardCategoryVehicle, Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Destructeur_de_wasps.png",
			Category:    citationCatVehicule,
			TierTargets: tierTargets3_6_9_15_30, Subcategory: citationSubUNSC},

		// ── PVE — Firefight (10, dont 4 désactivées) ──────────────────
		{Norm: "grunt_slayer", MappingType: mappingTypePVEStat,
			StatName: "grunt_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Tueur_de_Grognards.png",
			Category:    citationCatEnnemi,
			TierTargets: tierTargets10_20_30_50_100, Subcategory: citationSubCovenant},
		{Norm: "elite_slayer", MappingType: mappingTypePVEStat,
			StatName: "elite_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Tueur_d'Élites.png",
			Category:    citationCatEnnemi,
			TierTargets: tierTargets10_20_30_50_100, Subcategory: citationSubCovenant},
		{Norm: "jackal_slayer", MappingType: mappingTypePVEStat,
			StatName: "jackal_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Tueur_de_Rapaces.png",
			Category:    citationCatEnnemi,
			TierTargets: tierTargets5_10_15_25_50, Subcategory: citationSubCovenant},
		{Norm: "hunter_slayer", MappingType: mappingTypePVEStat,
			StatName: "hunter_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Tueur_de_Chasseurs.png",
			Category:    citationCatEnnemi,
			TierTargets: "2,4,6,10,20", Subcategory: citationSubCovenant},
		{Norm: "sentinel_slayer", MappingType: mappingTypePVEStat,
			StatName: "sentinel_kills", Enabled: false,
			ImagePath:   wpH5 + "H5G_citation_Tueur_de_sentinelles.png",
			Category:    citationCatEnnemi,
			TierTargets: tierTargets5_10_15_25_50, Subcategory: citationSubBanished},
		{Norm: "like_a_boss", MappingType: mappingTypePVEStat,
			StatName: "boss_kills", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Like_a_boss.png",
			Category:    citationCatSpartanCompanies,
			TierTargets: "250,500,750,1200,2400"},
		// I7 suite (2026-07-24) — DÉCISION UTILISATEUR : « Éliminations Firefight » compte
		// les VICTOIRES en Baptême du feu (la description « Gagner des parties en Baptême du
		// feu » était déjà correcte ; l'ancien stat_name total_enemy_kills comptait des kills
		// = faux). Bascule sur la fonction custom compute_wins_firefight (déjà implémentée et
		// enregistrée dans citations_custom.go, jusqu'ici NON liée à une citation → dead code
		// résorbé) : renvoie 1 par match Firefight gagné (outcome == OutcomeWin), 0 sinon —
		// même patron que flag_victory / slayer_victory / strongholds_victory, d'où le même
		// palier de victoires tierTargets5_10_15_25_50 (recalibré depuis l'échelle kills x00).
		// Reste thématiquement Firefight (Category inchangée). Non enfant d'un composite → la
		// sémantique de covenant_destroyer et des autres composites reste inchangée.
		{Norm: "player_vs_everything", MappingType: mappingTypeCustom,
			CustomFunction: "compute_wins_firefight", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Player_vs_Everything.png",
			Category:    citationCatSpartanCompanies,
			TierTargets: tierTargets5_10_15_25_50},
		{Norm: "brute_slayer", MappingType: mappingTypePVEStat,
			StatName:    "brute_kills",
			Enabled:     false,
			Category:    citationCatEnnemi,
			TierTargets: "10,20,30,50,100", Subcategory: citationSubBanished},
		{Norm: "skimmer_slayer", MappingType: mappingTypePVEStat,
			StatName:    "skimmer_kills",
			Enabled:     false,
			Category:    citationCatEnnemi,
			TierTargets: "10,20,30,50,100", Subcategory: citationSubBanished},
		{Norm: "marine_slayer", MappingType: mappingTypePVEStat,
			StatName: "marine_kills", Enabled: false,
			ImagePath:   wpH5 + "H5G_citation_Tueur_de_répliques_de_Marines.png",
			Category:    citationCatEnnemi,
			TierTargets: "20,40,60,100,200", Subcategory: citationSubBanished},

		// ── Armes — UNSC (10) ─────────────────────────────────────────
		{Norm: "br75_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:BR75", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_BR75.png",
			Category:    citationCatArme,
			TierTargets: "25,50,100,200,500", Subcategory: citationSubUNSC},
		{Norm: "ma40_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:MA40 AR", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_MA40.png",
			Category:    citationCatArme,
			TierTargets: "25,50,100,200,500", Subcategory: citationSubUNSC},
		{Norm: "sidekick_mastery", MappingType: mappingTypeWeaponStat,
			// CORRECTION 2026-09-10 : `Mk51` -> `Mk50`. La citation demandait un nom qui
			// n'existe nulle part — le registre et weapon_names.toml disent tous deux
			// « Mk50 Sidekick », et le Display de cette ligne le disait déjà. Un chiffre,
			// et la citation comptait zéro depuis sa création, sans le moindre signal.
			StatName: "weapon_kills:Mk50 Sidekick", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Sidekick.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubUNSC},
		{Norm: "commando_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:VK78 Commando", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Commando.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubUNSC},
		{Norm: "sniper_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:S7 Sniper", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Sniper-S7.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_20_40_80_200, Subcategory: citationSubUNSC},
		{Norm: "spnkr_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:M41 SPNKr", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_SPNKR.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_20_40_80_200, Subcategory: citationSubUNSC},
		{Norm: "bulldog_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:CQS48 Bulldog", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Bulldog.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubUNSC},
		{Norm: "bandit_mastery", MappingType: mappingTypeWeaponStat,
			// CORRECTION 2026-09-10 : « Bandit Evo » est le libellé FR (weapon_names.toml
			// `fr`), pas l'identité EN canonique. Le moteur compare au nom canonique —
			// « M392 Bandit » — donc la citation comptait zéro depuis sa création. Piège
			// général du type `weapon_stat` : le StatName porte un nom EN, jamais un
			// libellé affiché. Garde-rail : TestWeaponStatCitations_ResolvableName.
			StatName: "weapon_kills:M392 Bandit", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Bandit.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubUNSC},
		{Norm: "hydra_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:MLRS-2 Hydra", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Hydra.png",
			Category:    citationCatArme,
			TierTargets: "5,10,20,40,100", Subcategory: citationSubUNSC},
		{Norm: "mutilator_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Mutilator", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Mutilator.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubUNSC},

		// ── Armes — Paria (9) ─────────────────────────────────────────
		{Norm: "stalker_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Stalker Rifle", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Stalker.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubParia},
		{Norm: "needler_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Needler", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Needler.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubParia},
		{Norm: "energy_sword_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Energy Sword", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Épée_à_énergie.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_20_40_80_200, Subcategory: citationSubParia},
		{Norm: "mangler_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Mangler", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Mangler.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubParia},
		{Norm: "skewer_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Skewer", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Skewer.png",
			Category:    citationCatArme,
			TierTargets: "5,10,20,40,100", Subcategory: citationSubParia},
		{Norm: "gravity_hammer_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Gravity Hammer", Enabled: true,
			ImagePath:   wpH5 + "H5G_citation_Marteau_antigrav.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_20_40_80_200, Subcategory: citationSubParia},
		{Norm: "pulse_carbine_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Pulse Carbine", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Carabine.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubParia},
		{Norm: "ravager_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Ravager", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Ravager.png",
			Category:    citationCatArme,
			TierTargets: "5,10,20,40,100", Subcategory: citationSubParia},
		{Norm: "plasma_pistol_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Plasma Pistol", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Plasma.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubParia},

		// ── Armes — Forerunner (5) ────────────────────────────────────
		{Norm: "heatwave_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Heatwave", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Heatwave.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubForerunner},
		{Norm: "cindershot_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Cindershot", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Cremator.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_20_40_80_200, Subcategory: citationSubForerunner},
		{Norm: "sentinel_beam_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Sentinel Beam", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Sentinel.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubForerunner},
		{Norm: "disruptor_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Disruptor", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Disruptor.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubForerunner},
		{Norm: "shock_rifle_mastery", MappingType: mappingTypeWeaponStat,
			StatName: "weapon_kills:Shock Rifle", Enabled: true,
			ImagePath:   wpHI + "HI_Commendations_Shock.png",
			Category:    citationCatArme,
			TierTargets: tierTargets10_25_50_100_250, Subcategory: citationSubForerunner},

		// ── Composites (7) ────────────────────────────────────────────
		{Norm: "covenant_destroyer", MappingType: mappingTypeComposite,
			CompositeChildren: `["grunt_slayer","elite_slayer","jackal_slayer","hunter_slayer","like_a_boss","brute_slayer","skimmer_slayer"]`,
			Enabled:           true,
			ImagePath:         wpH5 + "H5G_maîtrise_Destructeur_de_Covenants.png",
			Category:          citationCatEnnemi,
			Subcategory:       citationSubCovenant},
		{Norm: "grenade_mastery", MappingType: mappingTypeComposite,
			CompositeChildren: `["frag_grenade","plasma_grenade"]`,
			Enabled:           true,
			ImagePath:         wpH5 + "H5G_Maîtrise_des_grenades.png",
			Category:          citationCatArme,
			Subcategory:       "Grenade"},
		{Norm: "vehicle_mastery", MappingType: mappingTypeComposite,
			// Neuf enfants d origine + les HUIT « Artilleur de » du 2026-09-10 (six au premier
			// passage, Warthog et Gungoose une fois leur source atteignable). Le max du
			// composite passe donc de 9 à 17 : la progression déjà écrite reste valide
			// (elle compte des enfants masterisés, pas un total brut), mais le palier
			// final recule pour tout le monde. Un re-seed impose le backfill
			// `--citations-recompute-all` (docs/COMMENDATIONS.md), sans quoi les six
			// nouveaux enfants restent à zéro sur l'historique.
			CompositeChildren: `["splatter","driver","wraith_destroyer","banshee_destroyer","ghost_destroyer","mongoose_destroyer","scorpion_destroyer","warthog_destroyer","wasp_destroyer","ghost_gunner","banshee_gunner","wraith_gunner","scorpion_gunner","wasp_gunner","rockethog_gunner","warthog_gunner","gungoose_gunner"]`,
			Enabled:           true,
			ImagePath:         wpH5 + "Vehicle_Mastery.png",
			Category:          citationCatVehicule,
			Subcategory:       citationSubGeneral},
		{Norm: "human_weapons_mastery", MappingType: mappingTypeComposite,
			CompositeChildren: `["br75_mastery","ma40_mastery","sidekick_mastery","commando_mastery","sniper_mastery","spnkr_mastery","bulldog_mastery","bandit_mastery","hydra_mastery","mutilator_mastery"]`,
			Enabled:           true,
			ImagePath:         wpH5 + "H5G_Maîtrise_en_armes_UNSC.png",
			Category:          citationCatArme,
			Subcategory:       citationSubUNSC},
		{Norm: "paria_weapons_mastery", MappingType: mappingTypeComposite,
			CompositeChildren: `["stalker_mastery","needler_mastery","energy_sword_mastery","mangler_mastery","skewer_mastery","gravity_hammer_mastery","pulse_carbine_mastery","ravager_mastery","plasma_pistol_mastery"]`,
			Enabled:           true,
			ImagePath:         wpHI + "HI_Maîtrise_en_armes_lourdes_Parias.png",
			Category:          citationCatArme,
			Subcategory:       citationSubParia},
		{Norm: "forerunner_weapons_mastery", MappingType: mappingTypeComposite,
			CompositeChildren: `["heatwave_mastery","cindershot_mastery","sentinel_beam_mastery","disruptor_mastery","shock_rifle_mastery"]`,
			Enabled:           true,
			ImagePath:         wpH5 + "H5G_Maîtrise_en_armes_lourdes_forerunners.png",
			Category:          citationCatArme,
			Subcategory:       citationSubForerunner},
		{Norm: "all_weapons_mastery", MappingType: mappingTypeComposite,
			CompositeChildren: `["human_weapons_mastery","paria_weapons_mastery","forerunner_weapons_mastery","grenade_mastery"]`,
			Enabled:           true,
			ImagePath:         wpH5 + "H5G_Maîtrise_en_armement.png",
			Category:          citationCatArme,
			Subcategory:       citationSubGeneral},
	}
}
