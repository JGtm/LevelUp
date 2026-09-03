package filmdec

// r7_grammaire_research_test.go — lot R7 du PLAN_PERCER_TRAME_FILM : la TABLE DE GRAMMAIRE
// des evenements de film, sourcee de l'executable (projet Ghidra HI, HaloInfinite.exe retail).
//
// POURQUOI CE FICHIER. Marcher la LISTE COMPLETE d'un paquet delta exige, pour chaque type
// rencontre, (a) les domaines de ses 3 references (largeur de l'index) et (b) la largeur en
// bits de sa charge. La tete seule ne suffit pas : c'est le verrou que R6 n'avait pas fait
// sauter. Ce fichier porte les deux tables ; `r7_marche_liste_research_test.go` s'en sert.
//
// SOURCE DES DOMAINES (extraction mecanique, ce lot). Pour chacun des 123 descripteurs
// (annexe A de GRAMMAIRE_EVENTS_FILM_2026-08-30) : objet 8 octets -> vtable -> slot +0x58 =
// le thunk `domaine(i)`. Les 123 types se ramenent a 30 thunks distincts, tous decodes sur
// OCTETS (mini-interprete des opcodes 33C0 / 85D2 / 83EA01 / 74 / 75 / EB / E9 / 0F84 /
// 0F85 / B8 / 8D42 / C3) simules a i = 0, 1, 2.
//
// PIEGE MAJEUR, ET IL A COUTE UNE PASSE : dix thunks se terminent par un `JNZ rel32` vers
// une adresse tres lointaine. R6 (par.2.1.1) l'a lu comme un CHEMIN D'ERREUR et en a conclu
// que les references 1 et 2 « n'existent pas » pour ces types (117, 104, 21, 9, 5, 6, 1, 36,
// 75, 106). C'est FAUX : c'est un BLOC FROID du decoupage chaud/froid du compilateur. En le
// SUIVANT (lecture des octets a l'arrivee), le bloc reprend `SUB EDX,1 ; JZ <retour> ;
// MOV EAX,7 ; RET` — la suite normale de la cascade. Les trois domaines existent pour TOUS
// les types.
//
// PREUVE INDEPENDANTE que le suivi de saut est le bon : le type 21 `unit_zoom` rend {4,8,7},
// exactement les domaines que le decodeur de PRODUCTION `zoom_events.go` utilise depuis sa
// validation (98 % de fermeture de slot, temoin nomme 6/6). Deux chaines sans etape commune.
//
// Recoupements avec l'anterieur, tous verts : type 0 damage_aftermath = {1,1,7}
// (NOTE_MODELE_EVENEMENTS) ; type 82 PlayerGameEventSmall = {0,8,7} (grammaire E7.3) ;
// types 30 et 48 = {4,8,7}, 42 = {2,8,7}, 43 et 93 = {2,0,7}, 100 et 32 = {1,8,7}.
// Corrections a R6 : 103 est {0,0,7} et non {7,0,7} (sans consequence : 0 et 7 font tous
// deux 13 bits), et les six types dits « refs inexistantes » ont bien trois domaines.
//
// LECTURE SEULE, aucun effet de bord, skip par defaut.

// r7DomWidth : largeur R(w) de l'index par domaine (table 0x1451f98d0, lot B1/E2 de la
// grammaire). Le domaine 1 porte en plus une SONDE R(1) : largeur 9 si sonde=1, sinon 13.
var r7DomWidth = map[int]uint{0: 13, 1: 13, 2: 8, 3: 8, 4: 9, 5: 8, 6: 9, 7: 13, 8: 13}

// r7Domains : les 3 domaines de references par type, derives des 30 thunks +0x58.
// Le commentaire donne le thunk d'origine (adresse virtuelle, exe retail).
var r7Domains = map[int][3]int{
	// 0x14080a018 {1,1,7}
	0: {1, 1, 7}, 20: {1, 1, 7}, 22: {1, 1, 7}, 53: {1, 1, 7}, 54: {1, 1, 7}, 98: {1, 1, 7},
	// 0x14080a048 {1,8,7} (bloc froid suivi)
	1: {1, 8, 7}, 36: {1, 8, 7}, 75: {1, 8, 7},
	// 0x140ebfee8 {1,8,7}
	2: {1, 8, 7}, 7: {1, 8, 7}, 31: {1, 8, 7}, 32: {1, 8, 7}, 35: {1, 8, 7}, 37: {1, 8, 7},
	58: {1, 8, 7}, 76: {1, 8, 7}, 86: {1, 8, 7}, 100: {1, 8, 7}, 110: {1, 8, 7}, 118: {1, 8, 7},
	// 0x142ef7f6c {0,8,7}
	3: {0, 8, 7}, 4: {0, 8, 7}, 19: {0, 8, 7}, 23: {0, 8, 7}, 28: {0, 8, 7}, 59: {0, 8, 7},
	60: {0, 8, 7}, 61: {0, 8, 7}, 69: {0, 8, 7}, 81: {0, 8, 7}, 82: {0, 8, 7}, 83: {0, 8, 7},
	84: {0, 8, 7}, 107: {0, 8, 7},
	// 0x1408096ec {5,8,7} (bloc froid suivi)
	5: {5, 8, 7}, 6: {5, 8, 7},
	// 0x142f1556c {2,3,7} · 0x1410f92bc {2,8,7} · 0x14116f6f0 {4,8,7} (blocs froids suivis)
	8: {2, 3, 7}, 9: {2, 8, 7}, 21: {4, 8, 7},
	// 0x141fd8440 {8,8,7}
	10: {8, 8, 7}, 12: {8, 8, 7}, 15: {8, 8, 7}, 16: {8, 8, 7}, 34: {8, 8, 7}, 80: {8, 8, 7},
	85: {8, 8, 7}, 99: {8, 8, 7}, 120: {8, 8, 7}, 121: {8, 8, 7}, 122: {8, 8, 7},
	// 0x140ebff6c {2,8,7}
	11: {2, 8, 7}, 42: {2, 8, 7}, 45: {2, 8, 7}, 51: {2, 8, 7}, 63: {2, 8, 7}, 70: {2, 8, 7},
	71: {2, 8, 7}, 72: {2, 8, 7}, 73: {2, 8, 7}, 74: {2, 8, 7}, 102: {2, 8, 7},
	// 0x140ebff50 {4,8,7}
	13: {4, 8, 7}, 30: {4, 8, 7}, 46: {4, 8, 7}, 47: {4, 8, 7}, 48: {4, 8, 7}, 78: {4, 8, 7},
	// 0x142eeb430 {2,2,7} · 0x142eeb41c {6,6,7}
	14: {2, 2, 7}, 17: {6, 6, 7},
	// 0x142c46770 {6,8,7}
	18: {6, 8, 7}, 29: {6, 8, 7}, 55: {6, 8, 7}, 62: {6, 8, 7}, 64: {6, 8, 7}, 66: {6, 8, 7},
	67: {6, 8, 7}, 68: {6, 8, 7}, 77: {6, 8, 7}, 87: {6, 8, 7}, 88: {6, 8, 7}, 89: {6, 8, 7},
	90: {6, 8, 7}, 91: {6, 8, 7}, 92: {6, 8, 7}, 94: {6, 8, 7}, 95: {6, 8, 7}, 96: {6, 8, 7},
	97: {6, 8, 7}, 101: {6, 8, 7}, 108: {6, 8, 7}, 111: {6, 8, 7}, 112: {6, 8, 7},
	113: {6, 8, 7}, 114: {6, 8, 7},
	// 0x142f15588 {0,2,7} · 0x142c612fc {0,0,7} · 0x142f155d8 {3,4,7}
	24: {0, 2, 7}, 25: {0, 2, 7}, 26: {0, 2, 7}, 27: {0, 0, 7}, 109: {0, 0, 7}, 33: {3, 4, 7},
	// 0x140ebff04 {2,8,7}
	38: {2, 8, 7}, 39: {2, 8, 7},
	// 0x140ebff38 {2,0,7} · 0x140ebff20 {2,0,7}
	40: {2, 0, 7}, 57: {2, 0, 7}, 93: {2, 0, 7}, 119: {2, 0, 7},
	43: {2, 0, 7}, 44: {2, 0, 7}, 52: {2, 0, 7},
	// 0x142f155a0 {3,8,7} · 0x142f155f4 {3,2,7} · 0x142ef7f9c {4,4,7} · 0x142f155bc {5,1,7}
	41: {3, 8, 7}, 65: {3, 8, 7}, 49: {3, 2, 7}, 50: {4, 4, 7}, 56: {5, 1, 7},
	// 0x142ef7f84 {4,0,7} · 0x14116cf48 {0,0,7}
	79: {4, 0, 7}, 103: {0, 0, 7}, 105: {0, 0, 7},
	// 0x141173cb0 / 0x141173ca4 / 0x142ef7fb0 / 0x141166e50 (blocs froids suivis)
	104: {0, 0, 7}, 106: {0, 0, 7}, 115: {1, 0, 7}, 116: {1, 0, 7}, 117: {2, 0, 7},
}

// r7Noms : les 123 noms de l'annexe A, pour lire les mesures sans table externe.
var r7Noms = map[int]string{
	0: "damage_aftermath", 1: "damage_section_response", 2: "restore_damage_section",
	3: "item_detonate", 4: "item_detonate_countdown", 5: "projectile_detonate",
	6: "projectile_impact_effect", 7: "projectile_object_impact_effect", 8: "biped_board_vehicle",
	9: "biped_pickup", 10: "weapon_effect", 11: "weapon_empty_click", 12: "biped_melee_clang",
	13: "motor_system_interruption", 14: "PlayEffectOnObject", 15: "Script", 16: "ShowDebugText",
	17: "Allegiance", 18: "MusicTrigger", 19: "CollectibleUnlockEvent", 20: "incident",
	21: "unit_zoom", 22: "unit_exit_vehicle", 23: "authority_ignored_predicted_position",
	24: "trade_weapon", 25: "device_touch", 26: "deviceRelease", 27: "controlToggleResponse",
	28: "biped_debug_teleport", 29: "prediction_determinism_msg", 30: "biped_equipment_activation",
	31: "equipment_teleport_request", 32: "unit_teleported", 33: "vehicle_auto_turret_choose_target",
	34: "PromptToBootGriefer", 35: "request_weapon_fire", 36: "action_weapon_fire",
	37: "weapon_overheat", 38: "weapon_reload", 39: "biped_throw_initiate",
	40: "biped_melee_initiate", 41: "vehicle_trick", 42: "biped_dodge",
	43: "initiate_mobility_action", 44: "weapon_pickup", 45: "weapon_put_away", 46: "weapon_drop",
	47: "weapon_throw", 48: "weapon_tether_request", 49: "vehicle_flip", 50: "request_ai_mount_exit",
	51: "biped_throw_release", 52: "biped_melee_damage", 53: "unit_enter_vehicle",
	54: "unit_switch_seat", 55: "game_engine_request_boot_player", 56: "request_projectile_attach",
	57: "biped_pickup_item_request", 58: "projectile_supercombine_request", 59: "object_refresh",
	60: "RequestChangeFrameConfiguration", 61: "player_forge_action", 62: "player_loadout_request",
	63: "biped_laser_designation", 64: "player_set_respawn_target_transform",
	65: "player_set_orbiting_camera_target", 66: "PlayerEmote", 67: "player_force_base_respawn",
	68: "supply_request", 69: "CampaignMapStateUpdate", 70: "AIPhase",
	71: "AIRequestIdleTransitionTime", 72: "AILand", 73: "AIJuke", 74: "AISetMotorProgram",
	75: "AIDialog", 76: "Dialogue2D", 77: "DebugSendCameraPosition", 78: "ai_jump",
	79: "networked_ai_action", 80: "networked_ai_effect", 81: "PlayerGameEvent",
	82: "PlayerGameEventSmall", 83: "TeamGameEvent", 84: "TeamGameEventSmall",
	85: "PlayerKilledEvent", 86: "EngineClientEvent", 87: "SaveGame", 88: "RevertMap",
	89: "CancelCinematic", 90: "ClientOnlyShowComplete", 91: "ClientResourcesLoadComplete",
	92: "BetrayResponse", 93: "activate_spartan_ability", 94: "CrewSetTargetObject",
	95: "CrewOrderPositionAdd", 96: "NetworkedCrewEventType", 97: "SaveToUGCService",
	98: "Equipment", 99: "SelectedSpawnZoneChangedEvent", 100: "PowerUpApplied",
	101: "LoadForgeObjectGroup", 102: "NetworkedActionRequest", 103: "EquipmentSpawnedObject",
	104: "EquipmentKnockbackPlayer", 105: "EquipmentObjectKnockedBack", 106: "ObjectCollisionDamage",
	107: "player_forge_user_string_action", 108: "NavpointRequest", 109: "PersonalAILifceycleEffect",
	110: "ObjectDeterministicDamageAcceleration", 111: "QueueNextShow", 112: "SetDifficultyAndSkulls",
	113: "FOBClientInput", 114: "MusicMarker", 115: "synchronized_teleport", 116: "teleport_effects",
	117: "EquipmentTranslocatorTeleportEffects", 118: "repair_complete",
	119: "EquipmentKnockbackRequest", 120: "PlayerCalloutRequest", 121: "PlayerForgeableCustomAction",
	122: "PlayerTriggerRadialMenu",
}

// r7Ref lit UNE reference gardee du domaine dom. Rend (index, presente, decodable).
func r7Ref(br *BitReader, dom int) (uint64, bool, bool) {
	if !br.ReadBit() {
		return 0, false, true
	}
	w := r7DomWidth[dom]
	if dom == 1 && br.ReadBit() { // sonde du domaine 1
		w = 9
	}
	idx := br.ReadBits(w)
	br.Skip(2) // generation
	return idx, true, true
}

// r7RefsSkip consomme les 3 references d'en-tete du type. Rend l'index de ref0 (utile pour
// le pont slot) et false si le cadrage est refute par une porte impossible.
func r7RefsSkip(br *BitReader, typ int) (uint64, bool, bool) {
	doms, ok := r7Domains[typ]
	if !ok {
		return 0, false, false
	}
	var idx0 uint64
	var has0 bool
	for i := 0; i < 3; i++ {
		idx, has, dec := r7Ref(br, doms[i])
		if !dec {
			return 0, false, false
		}
		if i == 0 {
			idx0, has0 = idx, has
		}
	}
	return idx0, has0, true
}
