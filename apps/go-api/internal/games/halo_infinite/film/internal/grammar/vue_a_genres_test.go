package grammar

import "testing"

// vue_a_genres_test.go — la table des genres de message relue dans l executable (lot LN) :
// genre, nom de debogage (`vtable + 0x08`), adresse du descripteur (affectee par `FUN_140e453b4` a
// `+0x210 + genre * 8`), adresse du lecteur de charge (`vtable + 0x68`) et domaines des trois
// references (rendus de `vtable + 0x58` pour i = 0, 1, 2), ecrits ici en LITTERAUX : le test ne
// derive rien de [tableDesGenresVueA], il la confronte au releve
// (`.ai/V7.5/film_re/campagne_grammaire_2026-10-01/ln_ghidra/table_genres.tsv`).

type genreReleve struct {
	genre                               int
	nom, descripteur, lecteur, domaines string
}

var genresReleves = []genreReleve{
	{0, "damage_aftermath", "0x144724f80", "0x1407f15a4", "117"},
	{1, "damage_section_response", "0x144724f78", "0x140968368", "187"},
	{2, "restore_damage_section", "0x14473faa0", "0x142ef90a4", "187"},
	{3, "item_detonate", "0x14473fa68", "0x1408d8220", "087"},
	{4, "item_detonate_countdown", "0x14473fa70", "0x1408d8220", "087"},
	{5, "projectile_detonate", "0x144724e28", "0x1408096f8", "587"},
	{6, "projectile_impact_effect", "0x144724e10", "0x1410f03b4", "587"},
	{7, "projectile_object_impact_effect", "0x144724e08", "0x142f17474", "187"},
	{8, "biped_board_vehicle", "0x144724e20", "0x142f168c0", "237"},
	{9, "biped_pickup", "0x144724e18", "0x141037828", "287"},
	{10, "weapon_effect", "0x144724db0", "0x142f17eec", "887"},
	{11, "weapon_empty_click", "0x144724da8", "0x142f17ffc", "287"},
	{12, "biped_melee_clang", "0x14473fa78", "0x142ef8e08", "887"},
	{13, "motor_system_interruption", "0x144724ce0", "0x142ef8f44", "487"},
	{14, "PlayEffectOnObject", "0x144724c68", "0x142eebf08", "227"},
	{15, "Script", "0x144724c20", "0x14080bb4c", "887"},
	{16, "ShowDebugText", "0x144724c50", "0x142eebf1c", "887"},
	{17, "Allegiance", "0x144724c60", "0x142eebb3c", "667"},
	{18, "MusicTrigger", "0x144724d00", "0x142ef8828", "687"},
	{19, "CollectibleUnlockEvent", "0x144724d78", "0x142ef8530", "087"},
	{20, "incident", "0x144724f98", "0x142f16fa0", "117"},
	{21, "unit_zoom", "0x144724e80", "0x141168b28", "487"},
	{22, "unit_exit_vehicle", "0x144724dc8", "0x142f17b94", "117"},
	{23, "authority_ignored_predicted_position", "0x144724e88", "0x1408d8220", "087"},
	{24, "trade_weapon", "0x144724f20", "0x1408d8220", "027"},
	{25, "device_touch", "0x144724f18", "0x1408d8220", "027"},
	{26, "deviceRelease", "0x144724ee8", "0x1408d8220", "027"},
	{27, "controlToggleResponse", "0x144724ef0", "0x142f16de8", "007"},
	{28, "biped_debug_teleport", "0x144724e68", "0x142f1699c", "087"},
	{29, "prediction_determinism_msg", "0x144724a58", "0x142c469c8", "687"},
	{30, "biped_equipment_activation", "0x144724d80", "0x142f16ad8", "487"},
	{31, "equipment_teleport_request", "0x144724d20", "0x142ef8ec8", "187"},
	{32, "unit_teleported", "0x144724c98", "0x142ef9470", "187"},
	{33, "vehicle_auto_turret_choose_target", "0x144724e78", "0x1408d8220", "347"},
	{34, "PromptToBootGriefer", "0x144724ca8", "0x142ef8ab0", "887"},
	{35, "request_weapon_fire", "0x144724de0", "0x142f17500", "187"},
	{36, "action_weapon_fire", "0x144724dd8", "0x14080c1f8", "187"},
	{37, "weapon_overheat", "0x144724c90", "0x142ef94f4", "187"},
	{38, "weapon_reload", "0x144724ed0", "0x1407f0ff8", "287"},
	{39, "biped_throw_initiate", "0x144724ec8", "0x140c6a58c", "287"},
	{40, "biped_melee_initiate", "0x14473fa80", "0x140ff8d70", "207"},
	{41, "vehicle_trick", "0x144724f08", "0x142f17c84", "387"},
	{42, "biped_dodge", "0x144724e70", "0x142f169d0", "287"},
	{43, "initiate_mobility_action", "0x144724d18", "0x142ef8f04", "207"},
	{44, "weapon_pickup", "0x144724ed8", "0x142f18158", "207"},
	{45, "weapon_put_away", "0x144724eb0", "0x142f18284", "287"},
	{46, "weapon_drop", "0x144724ea8", "0x142f17d74", "487"},
	{47, "weapon_throw", "0x144724ec0", "0x142f18490", "487"},
	{48, "weapon_tether_request", "0x144724eb8", "0x142f183f0", "487"},
	{49, "vehicle_flip", "0x144724f10", "0x1408d8220", "327"},
	{50, "request_ai_mount_exit", "0x14473fa98", "0x142ef908c", "447"},
	{51, "biped_throw_release", "0x144724f00", "0x142f16cac", "287"},
	{52, "biped_melee_damage", "0x14473fa88", "0x142ef8eb8", "207"},
	{53, "unit_enter_vehicle", "0x144724ef8", "0x142f168c0", "117"},
	{54, "unit_switch_seat", "0x144724c80", "0x1408d8220", "117"},
	{55, "game_engine_request_boot_player", "0x144724e58", "0x142f16dfc", "687"},
	{56, "request_projectile_attach", "0x144724ee0", "0x142f174c4", "517"},
	{57, "biped_pickup_item_request", "0x144724e50", "0x1408d8220", "207"},
	{58, "projectile_supercombine_request", "0x144724e98", "0x142f17480", "187"},
	{59, "object_refresh", "0x144724e90", "0x1408d8220", "087"},
	{60, "RequestChangeFrameConfiguration", "0x144724d58", "0x142ef8b18", "087"},
	{61, "player_forge_action", "0x144724cc0", "0x142ef8ff8", "087"},
	{62, "player_loadout_request", "0x144724e60", "0x142f16fd4", "687"},
	{63, "biped_laser_designation", "0x144724ea0", "0x142f16c90", "287"},
	{64, "player_set_respawn_target_transform", "0x144724e38", "0x142f17348", "687"},
	{65, "player_set_orbiting_camera_target", "0x144724e30", "0x142f17178", "387"},
	{66, "PlayerEmote", "0x144724e48", "0x142f164b8", "687"},
	{67, "player_force_base_respawn", "0x144724e40", "0x142f16fb8", "687"},
	{68, "supply_request", "0x144724c58", "0x142eec01c", "687"},
	{69, "CampaignMapStateUpdate", "0x144724d40", "0x142ef7fdc", "087"},
	{70, "AIPhase", "0x144724dc0", "0x142f15cf8", "287"},
	{71, "AIRequestIdleTransitionTime", "0x144724db8", "0x142f15e70", "287"},
	{72, "AILand", "0x144724d90", "0x142f15c9c", "287"},
	{73, "AIJuke", "0x144724d88", "0x142f15ae4", "287"},
	{74, "AISetMotorProgram", "0x144724da0", "0x142f15eac", "287"},
	{75, "AIDialog", "0x144724d38", "0x140f2e634", "187"},
	{76, "Dialogue2D", "0x144724d30", "0x140f2e87c", "187"},
	{77, "DebugSendCameraPosition", "0x144724d98", "0x142f160c4", "687"},
	{78, "ai_jump", "0x144724d50", "0x142ef8df0", "487"},
	{79, "networked_ai_action", "0x144724ce8", "0x142ef8f5c", "407"},
	{80, "networked_ai_effect", "0x144724cf8", "0x142ef8f74", "887"},
	{81, "PlayerGameEvent", "0x144724f50", "0x142f164f4", "087"},
	{82, "PlayerGameEventSmall", "0x144724f48", "0x14080add8", "087"},
	{83, "TeamGameEvent", "0x144724f60", "0x142f1686c", "087"},
	{84, "TeamGameEventSmall", "0x144724f58", "0x142f16818", "087"},
	{85, "PlayerKilledEvent", "0x144724f30", "0x14104bd08", "887"},
	{86, "EngineClientEvent", "0x144724f28", "0x142f1615c", "187"},
	{87, "SaveGame", "0x144724cd0", "0x142ef8b98", "687"},
	{88, "RevertMap", "0x144724cd8", "0x142ef8b5c", "687"},
	{89, "CancelCinematic", "0x144724d60", "0x142ef80fc", "687"},
	{90, "ClientOnlyShowComplete", "0x144724d70", "0x142ef8138", "687"},
	{91, "ClientResourcesLoadComplete", "0x144724d68", "0x142ef8334", "687"},
	{92, "BetrayResponse", "0x144724d48", "0x1408d8220", "687"},
	{93, "activate_spartan_ability", "0x144724c78", "0x142ef8d94", "207"},
	{94, "CrewSetTargetObject", "0x144724df0", "0x142f15fc0", "687"},
	{95, "CrewOrderPositionAdd", "0x144724de8", "0x142f15ec0", "687"},
	{96, "NetworkedCrewEventType", "0x144724e00", "0x142f162c8", "687"},
	{97, "SaveToUGCService", "0x144724df8", "0x142f16544", "687"},
	{98, "Equipment", "0x14473fa58", "0x142eebd68", "117"},
	{99, "SelectedSpawnZoneChangedEvent", "0x144724f40", "0x142f167d4", "887"},
	{100, "PowerUpApplied", "0x144724cb0", "0x142ef8a64", "187"},
	{101, "LoadForgeObjectGroup", "0x144724dd0", "0x142f16174", "687"},
	{102, "NetworkedActionRequest", "0x144724cf0", "0x142ef8a58", "287"},
	{103, "EquipmentSpawnedObject", "0x144724c28", "0x1408d8220", "007"},
	{104, "EquipmentKnockbackPlayer", "0x144724c38", "0x14116c344", "007"},
	{105, "EquipmentObjectKnockedBack", "0x144724c30", "0x141118a00", "007"},
	{106, "ObjectCollisionDamage", "0x144724f88", "0x14112134c", "007"},
	{107, "player_forge_user_string_action", "0x144724cc8", "0x142ef9040", "087"},
	{108, "NavpointRequest", "0x144724d10", "0x142ef8a40", "687"},
	{109, "PersonalAILifceycleEffect", "0x144724a88", "0x142c61310", "007"},
	{110, "ObjectDeterministicDamageAcceleration", "0x144724f90", "0x142f163c8", "187"},
	{111, "QueueNextShow", "0x144724cb8", "0x142ef8afc", "687"},
	{112, "SetDifficultyAndSkulls", "0x144724c70", "0x142ef8d50", "687"},
	{113, "FOBClientInput", "0x144724d28", "0x142ef86d4", "687"},
	{114, "MusicMarker", "0x144724d08", "0x142ef87f0", "687"},
	{115, "synchronized_teleport", "0x144724ca0", "0x142ef9284", "107"},
	{116, "teleport_effects", "0x144724c88", "0x142ef93e0", "107"},
	{117, "EquipmentTranslocatorTeleportEffects", "0x144724c48", "0x140f04fb8", "207"},
	{118, "repair_complete", "0x14473fa90", "0x142ef9074", "187"},
	{119, "EquipmentKnockbackRequest", "0x144724c40", "0x142eebcec", "207"},
	{120, "PlayerCalloutRequest", "0x144724f38", "0x142f163e0", "887"},
	{121, "PlayerForgeableCustomAction", "0x144724f70", "0x141fd8740", "887"},
	{122, "PlayerTriggerRadialMenu", "0x144724f68", "0x141fd87d0", "887"},
}

// TestLaTableDesGenresCouvreLesCentVingtTroisGenres : une entree par genre releve, dans l ordre, une
// version native par genre, et pour chaque genre la vacuite et les trois domaines du releve.
// MUTATION : un domaine change dans [tableDesGenresVueA] -> ROUGE.
func TestLaTableDesGenresCouvreLesCentVingtTroisGenres(t *testing.T) {
	if len(tableDesGenresVueA) != GenresVueA*largeurEntreeDeGenre || len(versionsNativesDesGenres) != GenresVueA {
		t.Fatalf("table %d caracteres, versions %d", len(tableDesGenresVueA), len(versionsNativesDesGenres))
	}
	if len(genresReleves) != GenresVueA {
		t.Fatalf("%d genres releves", len(genresReleves))
	}
	for i, g := range genresReleves {
		if g.genre != i {
			t.Fatalf("releve %d porte le genre %d", i, g.genre)
		}
		vide := g.lecteur == "0x1408d8220"
		dom, v := descripteurDuGenre(i)
		if v != vide {
			t.Errorf("genre %d (%s) : vide %v, lecteur %s", i, g.nom, v, g.lecteur)
		}
		if releve := [3]int{int(g.domaines[0] - '0'), int(g.domaines[1] - '0'), int(g.domaines[2] - '0')}; dom != releve {
			t.Errorf("genre %d (%s) : domaines %v, releve %v", i, g.nom, dom, releve)
		}
	}
}
