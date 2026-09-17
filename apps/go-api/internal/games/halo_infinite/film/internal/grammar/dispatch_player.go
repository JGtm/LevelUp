package grammar

import "levelup/go-api/internal/games/halo_infinite/film/types"

// dispatch_player.go — TROISIEME, QUATRIEME ET CINQUIEME MAILLONS : le joueur et son monde.
//
// Deplacement pur depuis `traverse.go` au lot 2.7 ; chaine et exemption de longueur
// documentees en tete de `dispatch_object.go`.

// consumePlayerAndSceneComponent porte le debut des composants de JOUEUR (ti=5 i0 a i9), plus
// les arms de scene, de statborg, de physique et de nuee portes dans les memes lots.
func consumePlayerAndSceneComponent(br *Lecteur, name string, typeIndex uint32, level uint32) (variant uint32, dead *types.DeadState, ported bool) { //nolint:gocyclo,funlen // dette gelee
	variant = noVariant
	switch name {
	case "player-waypoint-component": // ti=5 i0 (FUN_1410665dc) — R(3)
		br.ReadBits(3)
		return variant, nil, true
	case "player-respawn-timer-component": // ti=5 i1 (FUN_140f3fb8c) — R(1)+R(10)+R(10) = 21 bits
		// Grammaire dans decodePlayerRespawnTimer (vitality.go) — copie unique.
		_ = decodePlayerRespawnTimer(br)
		return variant, nil, true
	case compPlayerSoftKillTimer: // ti=5 i2 (FUN_140d580a8->FUN_140d580d0) — publie
		consumePlayerSoftKillTimer(br)
		return variant, nil, true
	case compPlayerTargetTracking: // ti=5 i3 (FUN_142f044f0) — publie
		consumePlayerTargetTracking(br)
		return variant, nil, true
	case "player-unsafe-respawn-timer-component": // ti=5 i4 (FUN_142f0452c) — R(10)
		br.ReadBits(10)
		return variant, nil, true
	case "player-respawn-safety-component": // ti=5 i5 (FUN_1410e3f30->FUN_140ebf854) — R(5)
		br.ReadBits(5)
		return variant, nil, true
	case compPlayerDesiredRespawnPlayer: // ti=5 i6 (FUN_1410f7330) — publie
		consumePlayerDesiredRespawnPlayer(br)
		return variant, nil, true
	// --- lot workflow filmdec-port-component-desers (2026-06-30, RE Ghidra parallele) ---
	// HIGH-confidence : largeurs fixes verifiees bit-exact.
	case "crew-orders-off-flags-component": // ti=14 i2 (FUN_142ed428c) — R(8)
		br.ReadBits(8)
		return variant, nil, true
	case "statborg-entry-index-and-type-component": // ti=6 i57 (FUN_1410be614) — R(32)+R(8)
		br.ReadBits(32)
		br.ReadBits(8)
		return variant, nil, true
	case "physics-state-component": // ti=22 i0 (FUN_142ed6c20) — R(32)+R(1)[si1:R(32)]
		br.ReadBits(32)
		if br.ReadBit() {
			br.ReadBits(32)
		}
		return variant, nil, true
	case "flock-emitting-component": // ti=21 i0 (FUN_142ed46c8) — R(1)
		br.ReadBit()
		return variant, nil, true
	case "music-variables-component": // ti=17 i0 (FUN_142ed66bc) — 32x{R(1); si bit==0: R(2)+R(32)+R(32)}
		for i := 0; i < 32; i++ {
			if !br.ReadBit() {
				br.ReadBits(2)
				br.ReadBits(32)
				br.ReadBits(32)
			}
		}
		return variant, nil, true
	case compGameEngineCurrentState: // ti=0 i2 (FUN_14116d1d0) — publie
		consumeGameEngineCurrentState(br)
		return variant, nil, true
	case "game-engine-game-finished-component": // ti=2 i3 (FUN_142f035e0) — R(1)
		br.ReadBit()
		return variant, nil, true
	case compManagedObjectPropName: // ti=13 i0 (FUN_142ed69d8) — R(32), sonde
		br.obs.publishProbe(typeIndex, ProbeManagedObjectPropertyName, br.ReadBits(32))
		return variant, nil, true
	case "managed-navpoint-sub-type-component": // ti=12 i0 (FUN_1410e0cac) — R(32)
		br.ReadBits(32)
		return variant, nil, true
	case "player-early-respawn-requested-component": // ti=5 i8 (FUN_142f04034) — R(1)
		br.ReadBit()
		return variant, nil, true
	case "player-engine-loadout-index-component": // ti=5 i9 (FUN_142f04054) — R(3)
		br.ReadBits(3)
		return variant, nil, true
	default:
		return consumeCrewFlockAndMusicComponent(br, name, typeIndex, level)
	}
}

// consumeCrewFlockAndMusicComponent porte les composants d'EQUIPAGE (ti=14), de NUEE (ti=21),
// de MUSIQUE (ti=17) et d'EFFET (ti=18) — la tranche ou se concentrent les vec3 quantifies a
// la largeur du registre (6 + niveau) — plus les premiers arms du moteur de partie (ti=0).
func consumeCrewFlockAndMusicComponent(br *Lecteur, name string, typeIndex uint32, level uint32) (variant uint32, dead *types.DeadState, ported bool) { //nolint:gocyclo,funlen // dette gelee
	variant = noVariant
	switch name {
	case "biped-emp-timer-component": // ti=35 i51 (FUN_142f02830) — R(8) (timer quant 0..10s)
		br.obs.publishEmpTimer(br.ReadBits(8)) // cf. emp_timer.go — largeur inchangée, valeur publiée
		return variant, nil, true
	case compSplashMessageDynamic: // ti=47 i1 (FUN_140daebd0) — R(24), sonde
		br.obs.publishProbe(typeIndex, ProbeSplashDynamic, br.ReadBits(24))
		return variant, nil, true
	// LOW-confidence : largeur quantifiee runtime data-dependent sur la branche gate==1.
	// On porte le cas commun (gate==0) et on desync PROPREMENT sur la branche data-dependent
	// (jamais de bit-guess = pas de corruption silencieuse en aval).
	case "crew-order-component": // ti=14 i0 (FUN_142ed4274) — R(3)+R(1)gate1[si1: vec3 quant 6+level]
		br.ReadBits(3)
		if br.ReadBit() { // gate1 == présence du vecteur
			consumeQuantVec3(br, quantAxisWidth(uint(level))) // PISTE 1 : largeur = 6+niveau(registre)
		}
		return variant, nil, true
	case "tacmap-poiiconoffset": // ti=30 i1 — vec3 quant pur (6+level)
		consumeQuantVec3(br, quantAxisWidth(uint(level)))
		return variant, nil, true
	case "tacmap-poiicon": // ti=30 i0 (FUN_142ed8418) — bloc + vec3 quant(6+level) au milieu
		br.ReadBits(32) // icon-id
		br.ReadBits(32) // icon-missionid
		br.ReadBit()
		br.ReadBits(3)
		br.ReadBits(32) // icon-text
		br.ReadBits(32) // icon-bitmapbg
		br.ReadBits(9)
		br.ReadBits(9)
		consumeQuantVec3(br, quantAxisWidth(uint(level))) // vec3
		br.ReadBits(32)                                   // string-id
		br.ReadBit()
		br.ReadBits(8)
		br.ReadBits(8)
		br.ReadBits(8)
		br.ReadBits(8)
		return variant, nil, true
	case "flock-destination-component": // ti=21 i2-i11 — R(1)flag + vec3 quant(6+level) + R(2) si rsp>1
		br.ReadBit()
		consumeQuantVec3(br, quantAxisWidth(uint(level)))
		if paramForComponent(br, name) > 1 {
			br.ReadBits(2)
		}
		return variant, nil, true
	case "crew-marked-objects-component": // ti=14 i1 (FUN_142ed421c) — R(1); gate==1 -> R(W)+R(2) W runtime
		if br.ReadBit() {
			return variant, nil, false
		}
		return variant, nil, true
	case "nav-cutscene-flag-component": // ti=15 i0 (FUN_142ed69b8) — R(1); present==1 -> gros bloc vec3 runtime
		if br.ReadBit() {
			return variant, nil, false
		}
		return variant, nil, true
	case "player-desired-respawn-seat-component": // ti=5 i7 (FUN_142f03f34) — R(1)[si1:R(W)+R(2) runtime]+R(7)
		if br.ReadBit() {
			return variant, nil, false
		}
		br.ReadBits(7)
		return variant, nil, true
	case "effect-state-data-component": // ti=18 i0 (FUN_142ed438c) — gates R(1)/R(32) ; markers R(W) runtime -> desync
		if br.ReadBit() {
			br.ReadBits(32) // tag-ref
		}
		if br.ReadBit() {
			return variant, nil, false // markerA -> R(W)+R(2) runtime
		}
		br.ReadBits(32) // marker-name
		if br.ReadBit() {
			return variant, nil, false // 2e marker -> R(W) runtime
		}
		return variant, nil, true
	// --- lot 2 workflow filmdec-port-component-desers (2026-06-30) ---
	// HIGH-confidence : largeurs fixes verifiees bit-exact.
	case "tacmap-poiisgoldenpath": // ti=30 i2 (FUN_142ed4800) — R(1)+R(1)
		br.ReadBit()
		br.ReadBit()
		return variant, nil, true
	case compPlayerEngineLoadout: // ti=5 i11 (FUN_141044428) — 8xR(8)=64, publie
		consumePlayerEngineLoadout(br)
		return variant, nil, true
	case "player-fade-properties-component": // ti=5 i13 (FUN_141020bac) — 6xR(12)=72
		for i := 0; i < 6; i++ {
			br.ReadBits(12)
		}
		return variant, nil, true
	case compPlayerLivesRemaining: // ti=5 i14 (FUN_141055734) — R(7), publie
		consumePlayerLivesRemaining(br)
		return variant, nil, true
	case "music-state-component": // ti=17 i1 (FUN_142ed5ef0) — R1+R32+R33+R33+gate1{R1+R16+R16}+gate2{6xR16}
		br.ReadBit()
		br.ReadBits(32)
		br.ReadBits(32)
		br.ReadBit()
		br.ReadBits(32)
		br.ReadBit()
		if br.ReadBit() {
			br.ReadBit()
			br.ReadBits(16)
			br.ReadBits(16)
		}
		if br.ReadBit() {
			br.ReadBits(16)
			br.ReadBits(16)
			br.ReadBits(16)
			br.ReadBits(16)
			br.ReadBits(16)
			br.ReadBits(16)
		}
		return variant, nil, true
	case "flock-destroying-component": // ti=21 i1 (FUN_142ed468c) — R(1)
		br.ReadBit()
		return variant, nil, true
	case "flock-forced-respawn-component": // ti=21 i12 (FUN_142ed4740) — R(1)
		br.ReadBit()
		return variant, nil, true
	case "flock-relevancy-component": // ti=21 i13 (FUN_14110d2fc) — R(8) (float quant 0..1)
		br.ReadBits(8)
		return variant, nil, true
	case "game-engine-round-timer-component": // ti=0 i5 (FUN_1407ee790) — R(16)+R(16)+R(5)
		// Grammaire dans decodeGameEngineRoundTimer (vitality.go) — copie unique.
		_ = decodeGameEngineRoundTimer(br)
		return variant, nil, true
	case compGameEngineSuddenDeath: // ti=0 i6 (FUN_14116d3a4) — R(16)+R(16)+R(5), publie
		consumeGameEngineSuddenDeath(br)
		return variant, nil, true
	case "game-engine-screen-sequence-component": // ti=0 i9 (FUN_14101d118) — R(4)+R(8)
		br.ReadBits(4)
		br.ReadBits(8)
		return variant, nil, true
	default:
		return consumePlayerTailAndGameEngineComponent(br, name, typeIndex, level)
	}
}

// consumePlayerTailAndGameEngineComponent porte la QUEUE des composants de joueur (ti=5 i10 a
// i26), le JOUEUR GERE (ti=9) et la fin du MOTEUR DE PARTIE (ti=0), plus les derniers arms de
// tacmap et d'equipement portes dans les memes lots.
func consumePlayerTailAndGameEngineComponent(br *Lecteur, name string, typeIndex uint32, level uint32) (variant uint32, dead *types.DeadState, ported bool) { //nolint:gocyclo,funlen // dette gelee
	variant = noVariant
	switch name {
	// LOW safe-prefix : cas dominant (gate==0 = absent) avance ; gate==1 (largeur runtime) desync propre.
	case "player-primary-respawn-object-component": // ti=5 i10 (FUN_142f043e8) — R(1)[si1:R(W)+R(2) runtime]
		if br.ReadBit() {
			return variant, nil, false
		}
		return variant, nil, true
	case compPlayerDesiredRespawnLoc: // ti=5 i12 — R(1)[si1: vec3 quant(6+level) + R(19)], publie
		consumePlayerDesiredRespawnLocation(br, level)
		return variant, nil, true
	// --- lot 3 workflow filmdec-port-component-desers (2026-06-30) ---
	case compPlayerLastBetrayer: // ti=5 i15 (FUN_142f04158) — R(6), publie
		consumePlayerLastBetrayer(br)
		return variant, nil, true
	case "player-vehicle-entrance-ban-component": // ti=5 i16 (FUN_142f04610) — R(1)
		br.ReadBit()
		return variant, nil, true
	case compPlayerControlAiming: // ti=5 i17 (FUN_142f03ea4) — R(19) direction cubemap, publie
		consumePlayerControlAiming(br)
		return variant, nil, true
	case compPlayerActiveInGame: // ti=5 i18 (FUN_1411615d8) — R(1), publie
		consumePlayerActiveInGame(br)
		return variant, nil, true
	case compPlayerPendingJoinInProgress: // ti=5 i19 (FUN_1411615b8) — R(1), publie
		consumePlayerPendingJoinInProgress(br)
		return variant, nil, true
	case compPlayerMalleableProperties: // ti=5 i20 (FUN_1407f0518) — publie, bits de porte compris
		consumePlayerMalleableProperties(br)
		return variant, nil, true
	case "player-representation-component": // ti=5 i21 (FUN_14111ec64) — R(32)
		br.ReadBits(32)
		return variant, nil, true
	case "player-power-frame-points-component": // ti=5 i23 (FUN_142f04240) — R(16)+R(16)
		br.ReadBits(16)
		br.ReadBits(16)
		return variant, nil, true
	case "player-supply-lines-currency-simulation-component": // ti=5 i25 (FUN_142f04408) — R(16)
		br.ReadBits(16)
		return variant, nil, true
	case "player-allowed-to-quit-component": // ti=5 i26 (FUN_142f04138) — R(1)
		br.ReadBit()
		return variant, nil, true
	case compGameEngineGracePeriod: // ti=0 i7 (FUN_141165d24) — R(16)+R(16)+R(5), publie
		consumeGameEngineGracePeriod(br)
		return variant, nil, true
	case compGameEngineRoundConditionFlags: // ti=0 i8 (FUN_141132dc0) — R(10), publie
		consumeGameEngineRoundConditionFlags(br)
		return variant, nil, true
	case "game-engine-alliance-component": // ti=0 i10 (FUN_140a24968) — R(32) mask + popcount*(R(32)+R(32))
		mask := uint32(br.ReadBits(32))
		for i := 0; i < 32; i++ {
			if (mask>>uint(i))&1 != 0 {
				br.ReadBits(32)
				br.ReadBits(32)
			}
		}
		return variant, nil, true
	case compGameEngineCurrentRound: // ti=0 i4 (FUN_14116fc70) — R(1)[si0:R(5)], publie
		consumeGameEngineCurrentRound(br)
		return variant, nil, true
	case "tacmap-backmenu-openoverride": // ti=34 i13 (FUN_142ed3d64)
		consumeTacmapBackmenuOpenoverride(br)
		return variant, nil, true
	case "tacmap-queuedreplaymission": // ti=34 i9 (FUN_1407f24f8)
		consumeTacmapQueuedReplayMission(br)
		return variant, nil, true
	case "tacmap-cooptetherarea": // ti=34 i11 (FUN_142ed4198)
		consumeTacmapCoopTetherArea(br)
		return variant, nil, true
	case "tacmap-displayasset": // ti=33 i0 (FUN_142ed433c)
		consumeTacmapDisplayAsset(br)
		return variant, nil, true
	case "equipment-tracked-object-handles-stack-component": // ti=37 i28 (FUN_140f72dec)
		consumeEquipmentTrackedStack2(br)
		return variant, nil, true
	case "equipment-command-tick-component": // ti=37 i29 (FUN_140e0a564, table ECS live)
		consumeEquipmentCommandTick(br)
		return variant, nil, true
	case "equipment-has-infinite-uses-component": // ti=37 i30 (FUN_142ed4640, table ECS live) — R(1)
		br.ReadBit()
		return variant, nil, true
	case "track-frame-component": // ti=16 i5 (FUN_142ed740c)
		consumeTrackFrameComponent(br)
		return variant, nil, true
	case "statborg-round-outcomes-component": // ti=6 i56 (FUN_142ed71a4) — 32×R(2)
		consumeStatborgRoundOutcomes(br)
		return variant, nil, true
	case compSplashMessageStatic: // ti=47 i0 (FUN_141085d50) — sonde sur le R(24) inconditionnel
		br.obs.publishProbe(typeIndex, ProbeSplashStatic, consumeManagedSplashMessage(br))
		return variant, nil, true
	default:
		return consumeManagedPlayerComponent(br, name, typeIndex, level)
	}
}

// consumeManagedPlayerComponent porte l ARCHETYPE `managed-player` (ti=9) EN ENTIER — le profil
// du joueur tel que le moteur le replique.
//
// # POURQUOI UN MAILLON A LUI (lot 3.6.a, 2026-09-17)
//
// Les dix composants de `ti=9` etaient EPARPILLES sur trois maillons de la chaine — `i0` dans le
// premier, `i1` a `i3` dans le deuxieme, `i5` a `i9` dans le troisieme — par pur hasard d ordre
// de portage. Les trois maillons etaient AU PLAFOND du ratchet de longueur
// (`archlint/film_function_length_test.go`, table datee du 2026-09-16) : porter `i4` n avait donc
// litteralement pas de place, et le ratchet dit lui-meme quoi faire — « sortir autant de lignes
// ailleurs dans la fonction, ou l extraire ».
//
// L EXTRACTION EST CELLE-CI, ET ELLE EST SANS EFFET SUR LES BITS. Un `switch` sur un nom de
// composant, dont les branches sont DISJOINTES, redistribue sur des maillons chaines par leur
// `default` : chaque nom tombe exactement sur la meme branche qu avant. Les trois maillons
// d origine perdent leurs `case` `ti=9` et rien d autre. Le garde-rail G1
// (`ecs_table_guard_test.go`) confronte la chaine entiere a `ecs_table.tsv` et rougirait sur un
// `case` perdu en route.
//
// Les grammaires elles-memes vivent dans `components_managed_player.go` des qu elles font plus
// d une lecture — c est la convention du paquet.
func consumeManagedPlayerComponent(br *Lecteur, name string, typeIndex uint32, level uint32) (variant uint32, dead *types.DeadState, ported bool) { //nolint:gocyclo // un case par composant du registre
	variant = noVariant
	switch name {
	case "managed-player-team-designator-component": // ti=9 i0 (FUN_140f581e8) — R(4)
		br.ReadBits(4)
		return variant, nil, true
	case "managed-player-color-override-component": // ti=9 i1 (FUN_142ed5b54) — 8xR(8)=64 (2 couleurs RGBA quant)
		for i := 0; i < 8; i++ {
			br.ReadBits(8)
		}
		return variant, nil, true
	case "managed-player-flags-component": // ti=9 i2 (FUN_142ed5bac) — R(4)
		br.ReadBits(4)
		return variant, nil, true
	case "managed-player-back-button-scoreboard-flair-component": // ti=9 i3 (FUN_142ed5af4) — R(32)
		br.ReadBits(32)
		return variant, nil, true
	case compManagedPlayerForgeWeather: // ti=9 i4 (FUN_142ed5bc8) — R(32)+R(32), lot 3.6.a
		consumeManagedPlayerForgeWeatherOverrides(br)
		return variant, nil, true
	case "managed-player-active-mission-name-component": // ti=9 i5 (FUN_142ed5ab0) — R(32)+R(32)
		br.ReadBits(32)
		br.ReadBits(32)
		return variant, nil, true
	case "managed-player-show-active-mission-name-in-hud-component": // ti=9 i6 (FUN_142ed5d68) — R(1)
		br.ReadBit()
		return variant, nil, true
	case "managed-player-campaign-progress-component": // ti=9 i7 (FUN_142ed5b18) — R(8)
		br.ReadBits(8)
		return variant, nil, true
	case "managed-player-current-season-component": // ti=9 i8 (FUN_142ed5b88) — R(32)
		br.ReadBits(32)
		return variant, nil, true
	case compManagedPlayerInputPrompt: // ti=9 i9 (FUN_141fcf160) — portes + sac texte, lot 3.6.a
		consumeManagedPlayerInputPrompt(br)
		return variant, nil, true
	default:
		return consumeCaptureAndBipedComponent(br, name, typeIndex, level)
	}
}
