package filmdec

// event_types_catalogue_test.go — LE CATALOGUE DES TYPES D'EVENEMENT, TEL QUE L'EXE LE NOMME.
//
// PROVENANCE ET STATUT. Ces noms ne sont PAS mesures dans le film : ils viennent de la table
// statique des descripteurs de `HaloInfinite.exe` (base `0x144724A90`, 8 octets par type),
// relevee mecaniquement `table[t] -> vtable+0x08 -> LEA RAX,[rip+X] -> chaine` et consignee en
// annexe de `.ai/V7.5/film_re/NOTE_ENVELOPPE_EVENTS_2026-08-30.md`. Les types 0..49 ne suivent
// pas ce format et n'ont pas ete resolus.
//
// POURQUOI ILS VIVENT ICI, DANS UN FICHIER DE TEST. Ce sont des DONNEES DE RECHERCHE : elles
// servent aux instruments qui cherchent ces memes noms DANS le film (lot D1) et qui nomment les
// types observes (lot D3). Aucune production ne s'en sert, et une table de 78 etiquettes en
// production serait du code mort. Le jour ou un decodeur nomme les types, la table remontera
// avec son lecteur.
//
// AVERTISSEMENT PORTE PAR LA NOTE ET A NE PAS PERDRE : le dispatcher borne le type a `< 0x7b`
// (123) alors que la table statique compte 128 entrees ; les types 123..127 y portent pourtant
// des descripteurs nommes valides. Cette divergence n'est pas expliquee.

// eventTypeNames est le catalogue des types 50..127 lu dans l'exe.
var eventTypeNames = map[int]string{
	50: "Script", 51: "EquipmentSpawnedObject", 52: "EquipmentObjectKnockedBack",
	53: "EquipmentKnockbackPlayer", 54: "EquipmentKnockbackRequest",
	55: "EquipmentTranslocatorTeleportEffects", 56: "ShowDebugText", 57: "supply_request",
	58: "Allegiance", 59: "PlayEffectOnObject", 60: "SetDifficultyAndSkulls",
	61: "activate_spartan_ability", 62: "unit_switch_seat", 63: "teleport_effects",
	64: "weapon_overheat", 65: "unit_teleported", 66: "synchronized_teleport",
	67: "PromptToBootGriefer", 68: "PowerUpApplied", 69: "QueueNextShow",
	70: "player_forge_action", 71: "player_forge_user_string_action", 72: "SaveGame",
	73: "RevertMap", 74: "motor_system_interruption", 75: "networked_ai_action",
	76: "NetworkedActionRequest", 77: "networked_ai_effect", 78: "MusicTrigger",
	79: "MusicMarker", 80: "NavpointRequest", 81: "initiate_mobility_action",
	82: "equipment_teleport_request", 83: "FOBClientInput", 84: "Dialogue2D",
	85: "AIDialog", 86: "CampaignMapStateUpdate", 87: "BetrayResponse", 88: "ai_jump",
	89: "RequestChangeFrameConfiguration", 90: "CancelCinematic",
	91: "ClientResourcesLoadComplete", 92: "ClientOnlyShowComplete",
	93: "CollectibleUnlockEvent", 94: "biped_equipment_activation", 95: "AIJuke",
	96: "AILand", 97: "DebugSendCameraPosition", 98: "AISetMotorProgram",
	99: "weapon_empty_click", 100: "weapon_effect", 101: "AIRequestIdleTransitionTime",
	102: "AIPhase", 103: "unit_exit_vehicle", 104: "LoadForgeObjectGroup",
	105: "action_weapon_fire", 106: "request_weapon_fire", 107: "CrewOrderPositionAdd",
	108: "CrewSetTargetObject", 109: "SaveToUGCService", 110: "NetworkedCrewEventType",
	111: "projectile_object_impact_effect", 112: "projectile_impact_effect",
	113: "biped_pickup", 114: "biped_board_vehicle", 115: "projectile_detonate",
	116: "player_set_orbiting_camera_target", 117: "player_set_respawn_target_transform",
	118: "player_force_base_respawn", 119: "PlayerEmote", 120: "biped_pickup_item_request",
	121: "game_engine_request_boot_player", 122: "player_loadout_request",
	123: "biped_debug_teleport", 124: "biped_dodge", 125: "vehicle_auto_turret_choose_target",
	126: "unit_zoom", 127: "authority_ignored_predicted_position",
}

// nomDuType rend le nom catalogue d'un type, ou "?" pour les types 0..49 non resolus.
func nomDuType(t int) string {
	if n, ok := eventTypeNames[t]; ok {
		return n
	}
	return "?"
}

// eventNomsCibles est la liste des noms que le lot D1 cherche DANS le film : ceux que le brief
// designe, plus les types qui structurent le dossier « visee lunette ».
var eventNomsCibles = []string{
	"action_weapon_fire", "biped_board_vehicle", "unit_zoom", "weapon_effect",
	"weapon_overheat", "projectile_detonate", "unit_exit_vehicle",
	"player_set_orbiting_camera_target", "unit_switch_seat", "weapon_empty_click",
	"DebugSendCameraPosition", "biped_pickup", "equipment_teleport_request",
}
