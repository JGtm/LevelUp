package filmdec

// components_player.go — LES COMPOSANTS DE L'ENTITE JOUEUR (ti=5), QUI PUBLIENT.
//
// MEME CONTRAT QUE `components_game_engine.go` : les onze deserialiseurs ci-dessous
// consommaient deja EXACTEMENT ces bits, en ligne dans le `switch` de `consumeByName`, et
// jetaient leurs valeurs. Ils sont ici, nommes, et ils publient. Aucune largeur, aucun ordre
// de lecture ne bouge.
//
// POURQUOI ti=5 MERITE SON PROPRE FICHIER. C'est l'entite du JOUEUR au sens du moteur, et
// c'est le gisement le plus dense du film hors bipede : le chargement de depart (i11), la
// position de reapparition desiree (i12), les vies restantes (i14), le dernier traitre (i15),
// la direction de visee de contole (i17), la presence en partie (i18/i19) et un bloc de
// proprietes modifiables (i20). Le lot P du plan d'exploitation le mesure ; ici, on ne fait
// que le rendre lisible.
//
// CE QUI RESTE SUR LA COUCHE DE CAPTURE, ET NE DOIT PAS ETRE DUPLIQUE ICI : ti=5 i1
// (`player-respawn-timer-component`) est rendu par `consumeByNameCapturing` sous forme typee
// (`RespawnTimer`). Le publier une seconde fois ici serait la troisieme copie.

import "fmt"

// Etiquettes de registre des composants de ti=5 publies ici.
const (
	compPlayerSoftKillTimer         = "player-soft-kill-timer-component"
	compPlayerTargetTracking        = "player-target-tracking-detection-component"
	compPlayerDesiredRespawnPlayer  = "player-desired-respawn-player-component"
	compPlayerEngineLoadout         = "player-engine-loadout-component"
	compPlayerDesiredRespawnLoc     = "player-desired-respawn-location-component"
	compPlayerLivesRemaining        = "player-lives-remaining-component"
	compPlayerLastBetrayer          = "player-last-betrayer-component"
	compPlayerControlAiming         = "player-control-aiming-component"
	compPlayerActiveInGame          = "player-active-in-game-component"
	compPlayerPendingJoinInProgress = "player-pending-join-in-progress-spawn-component"
	compPlayerMalleableProperties   = "player-malleable-properties-simulation-component"
)

// PlayerStateField designe l'un des onze champs publies. Enumeration STABLE et non index de
// registre, pour la raison ecrite dans `components_game_engine.go` (un index est un numero de
// build, et le lot 0 a mesure deux decoupages differents sur le corpus).
type PlayerStateField int

// Les onze champs, dans l'ordre des index du registre de reference, et leur compte.
const (
	PlayerSoftKill               PlayerStateField = iota // i2  : R(5)+R(5)+R(5)
	PlayerTargetTracking                                 // i3  : R(1)+R(1)
	PlayerDesiredRespawnPlayer                           // i6  : R(16)
	PlayerLoadout                                        // i11 : 8 x R(8)
	PlayerDesiredRespawnLocation                         // i12 : R(1) porte [si 1 -> vec3 quant + R(19)]
	PlayerLives                                          // i14 : R(7)
	PlayerLastBetrayer                                   // i15 : R(6)
	PlayerControlAiming                                  // i17 : R(19) direction cubemap
	PlayerActiveInGame                                   // i18 : R(1)
	PlayerPendingJoinInProgress                          // i19 : R(1)
	PlayerMalleableProperties                            // i20 : 3xR(1) + 6x[R(1) si 1 -> R(12)] + 9xR(1)
	PlayerStateFieldCount        = 11
)

// String rend l'etiquette de registre du champ.
func (f PlayerStateField) String() string {
	switch f {
	case PlayerSoftKill:
		return compPlayerSoftKillTimer
	case PlayerTargetTracking:
		return compPlayerTargetTracking
	case PlayerDesiredRespawnPlayer:
		return compPlayerDesiredRespawnPlayer
	case PlayerLoadout:
		return compPlayerEngineLoadout
	case PlayerDesiredRespawnLocation:
		return compPlayerDesiredRespawnLoc
	case PlayerLives:
		return compPlayerLivesRemaining
	case PlayerLastBetrayer:
		return compPlayerLastBetrayer
	case PlayerControlAiming:
		return compPlayerControlAiming
	case PlayerActiveInGame:
		return compPlayerActiveInGame
	case PlayerPendingJoinInProgress:
		return compPlayerPendingJoinInProgress
	case PlayerMalleableProperties:
		return compPlayerMalleableProperties
	}
	return fmt.Sprintf("champ inconnu (%d)", int(f))
}

// playerStateHook, si non nil, recoit CHAQUE lecture d'un des onze composants. Meme contrat de
// `values` / `present` que `gameEngineHook` (cf. son commentaire). Global de paquet : l'appelant
// detient `LockProcessDecode`.
var playerStateHook func(f PlayerStateField, values []uint64, present bool)

// SetPlayerStateHook installe (ou retire, avec nil) la sonde des composants de ti=5.
func SetPlayerStateHook(h func(f PlayerStateField, values []uint64, present bool)) {
	playerStateHook = h
}

func publishPlayerState(f PlayerStateField, present bool, values ...uint64) {
	if playerStateHook != nil {
		playerStateHook(f, values, present)
	}
}

// consumePlayerSoftKillTimer porte ti=5 i2 (FUN_140d580a8 -> FUN_140d580d0) : 3 x R(5).
func consumePlayerSoftKillTimer(br *BitReader) {
	a := br.ReadBits(5)
	b := br.ReadBits(5)
	c := br.ReadBits(5)
	publishPlayerState(PlayerSoftKill, true, a, b, c)
}

// consumePlayerTargetTracking porte ti=5 i3 (FUN_142f044f0) : R(1)+R(1).
func consumePlayerTargetTracking(br *BitReader) {
	a := bit2u(br.ReadBit())
	b := bit2u(br.ReadBit())
	publishPlayerState(PlayerTargetTracking, true, a, b)
}

// consumePlayerDesiredRespawnPlayer porte ti=5 i6 (FUN_1410f7330) : R(16).
func consumePlayerDesiredRespawnPlayer(br *BitReader) {
	publishPlayerState(PlayerDesiredRespawnPlayer, true, br.ReadBits(16))
}

// consumePlayerEngineLoadout porte ti=5 i11 (FUN_141044428) : 8 x R(8) = 64 bits.
//
// LES HUIT OCTETS SONT PUBLIES BRUTS, dans l'ordre du flux. Ce qu'ils SIGNIFIENT — le
// chargement de depart en clair, ou autre chose — est la question du lot P, et la plomberie
// n'a pas a y repondre.
func consumePlayerEngineLoadout(br *BitReader) {
	v := make([]uint64, 0, 8)
	for i := 0; i < 8; i++ {
		v = append(v, br.ReadBits(8))
	}
	if playerStateHook != nil {
		playerStateHook(PlayerLoadout, v, true)
	}
}

// consumePlayerDesiredRespawnLocation porte ti=5 i12 : R(1) porte ; si 1, un vec3 quantifie
// (largeur 6 + niveau du registre) puis R(19) (FUN_14076dc04, identifiant de reapparition).
//
// LE VEC3 EST PUBLIE BRUT, ET LE NIVEAU AVEC LUI. Aucune dequantification ici : la largeur
// d'axe vaut `quantAxisWidth(level)` et le lot qui mesure en a besoin pour interpreter les
// trois quanta. `values` = [qx, qy, qz, identifiant, niveau].
//
// PORTE FERMEE (bit == 0) : `present` est faux et `values` est vide — le composant etait au
// masque, il n'a transmis aucune position. Ce n'est pas une position a l'origine.
func consumePlayerDesiredRespawnLocation(br *BitReader, level uint32) {
	if !br.ReadBit() {
		publishPlayerState(PlayerDesiredRespawnLocation, false)
		return
	}
	w := quantAxisWidth(uint(level))
	qx, qy, qz, ok := consumeQuantVec3Values(br, w)
	id := br.ReadBits(19)
	if !ok {
		// precHigh == 1 : le vecteur par defaut, zero bit de charge utile. L'identifiant a
		// bien ete lu, lui : on publie ce qui existe et on ne fabrique pas de coordonnees.
		publishPlayerState(PlayerDesiredRespawnLocation, false, id)
		return
	}
	publishPlayerState(PlayerDesiredRespawnLocation, true, qx, qy, qz, id, uint64(level))
}

// consumePlayerLivesRemaining porte ti=5 i14 (FUN_141055734) : R(7).
func consumePlayerLivesRemaining(br *BitReader) {
	publishPlayerState(PlayerLives, true, br.ReadBits(7))
}

// consumePlayerLastBetrayer porte ti=5 i15 (FUN_142f04158) : R(6).
func consumePlayerLastBetrayer(br *BitReader) {
	publishPlayerState(PlayerLastBetrayer, true, br.ReadBits(6))
}

// consumePlayerControlAiming porte ti=5 i17 (FUN_142f03ea4) : R(19), direction de visee
// encodee en cubemap. Le decodeur de cette direction EXISTE (`DecodeAimVectorChecked`,
// `aim_vector.go`) — le brancher est le travail du lot E, pas celui de la plomberie.
func consumePlayerControlAiming(br *BitReader) {
	publishPlayerState(PlayerControlAiming, true, br.ReadBits(19))
}

// consumePlayerActiveInGame porte ti=5 i18 (FUN_1411615d8) : R(1).
func consumePlayerActiveInGame(br *BitReader) {
	publishPlayerState(PlayerActiveInGame, true, bit2u(br.ReadBit()))
}

// consumePlayerPendingJoinInProgress porte ti=5 i19 (FUN_1411615b8) : R(1).
func consumePlayerPendingJoinInProgress(br *BitReader) {
	publishPlayerState(PlayerPendingJoinInProgress, true, bit2u(br.ReadBit()))
}

// consumePlayerMalleableProperties porte ti=5 i20 (FUN_1407f0518) :
// 3 x R(1), puis 6 x [R(1) porte ; si 1 -> R(12)], puis 9 x R(1).
//
// `values` PORTE LES BITS DE PORTE, et c'est la seule facon de ne rien perdre : sans eux, on
// ne saurait pas QUELLE des six proprietes a transmis sa valeur. Forme publiee, 3 + 12 + 9 =
// 24 entrees exactement : [f0 f1 f2] puis, pour chacune des six, [porte, valeur] (valeur = 0
// quand la porte est a 0), puis [g0 .. g8].
func consumePlayerMalleableProperties(br *BitReader) {
	v := make([]uint64, 0, 24)
	for i := 0; i < 3; i++ {
		v = append(v, bit2u(br.ReadBit()))
	}
	for i := 0; i < 6; i++ {
		if br.ReadBit() {
			v = append(v, 1, br.ReadBits(12))
			continue
		}
		v = append(v, 0, 0)
	}
	for i := 0; i < 9; i++ {
		v = append(v, bit2u(br.ReadBit()))
	}
	if playerStateHook != nil {
		playerStateHook(PlayerMalleableProperties, v, true)
	}
}
