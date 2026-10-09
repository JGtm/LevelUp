package grammar

import "levelup/go-api/internal/games/halo_infinite/film/types"

// components_moteur_de_partie.go — LA FIN DE L'ARCHETYPE DU MOTEUR DE PARTIE (`ti=0`, `ti=1`,
// `ti=2`, composants `i11` a `i17`).
//
// Chaque lecteur est celui du jeu (HaloInfinite.exe HI_1_13_0, Ghidra, lecture seule) : nom ->
// accesseur de nom -> descripteur -> lecteur a `descripteur + 0x40` ; l ecrivain est a
// `descripteur + 0x28`. Les deux ont ete relus pour chaque composant :
//
//	composant                                   descripteur  lecteur        ecrivain
//	game-engine-soft-ceilings                   143d0f560    FUN_14116d1ac  0x142f0688c
//	game-engine-disabled-kill-volume-flags      143d0f510    FUN_142f03498  FUN_142f06530
//	GameEngineComposerLetterboxComponent        143d0f740    FUN_142f0328c  FUN_142f06308
//	managed-engine-timers                       143d08ca0    FUN_1407ee7b8  FUN_142edad74
//	scenario-intro                              143d08c50    FUN_1410d9004  FUN_142edd05c
//	matchflow-isplaying-flags                   143d08bb0    FUN_141101038  0x142edbef4
//
// LE NIVEAU. La boucle de composants du jeu (FUN_14076cb60) passe au lecteur, en rejeu de film, le
// niveau que le registre DU FILM declare (FUN_1428e1b50, puis le thunk FUN_14076ce9c le pose dans
// R9D). Seul le lecteur de `GameEngineComposerLetterboxComponent` le lit (`CMP R9D, 2`) ; ce port
// recoit le meme niveau, celui du registre du film (`arch.Level(i)`), et prend la meme branche.
// Aucune branche ne depend de la version du jeu.

// Etiquettes de registre des composants portes ici.
const (
	compGameEngineSoftCeilings               = "game-engine-soft-ceilings-component"
	compGameEngineDisabledKillVolumes        = "game-engine-disabled-kill-volume-flags-component"
	compGameEngineComposerLetterbox          = "GameEngineComposerLetterboxComponent"
	compManagedEngineTimers                  = "managed-engine-timers-component"
	compScenarioIntro                        = "scenario-intro-component"
	compMatchflowIsPlayingFlags              = "matchflow-isplaying-flags-component"
	niveauLetterboxLong               uint32 = 2 // `CMP R9D, 2 ; JC` : en dessous, la forme courte
)

// Largeurs lues dans les lecteurs du jeu.
const (
	largeurPlafondsDeMoteur       = 128 // FUN_1406d676c(.., n = 0x80)
	largeurCompteVolumes     uint = 13  // `+0x2c += 0xd`
	largeurLetterboxReel     uint = 16  // FUN_1406d84b4, [RSP+0x20] = 0x10
	largeurLetterboxIndex    uint = 7   // FUN_142efd284 : `+0x2c += 7`
	largeurLetterboxMot      uint = 16  // `+0x2c += 0x10`
	largeurLetterboxCourte        = 64  // FUN_1406d676c(.., n = 0x40), forme courte seulement
	entreesLetterbox              = 4   // deux boucles de 4 (etat + 0x56c..0x57b, + 0x57c..)
	largeurMasqueMinuteurs   uint = 64  // FUN_1406d6bac(flux, 0x40)
	fentesMinuteurs               = 64  // `CMP EDI, 0x40`
	largeurEtiquetteFente    uint = 2   // FUN_1407ee87c : `+0x2c += 2`
	largeurMinuteurFente     uint = 16  // `MOV [RSP+0x28], 0x10`, le n passe aux lecteurs de fente
	largeurScenarioIntro     uint = 7   // FUN_1410d9088 : `+0x2c += 7`, rend la valeur - 1
	largeurDrapeauxMatchflow uint = 8   // FUN_141101038 : `+0x2c += 8`
)

// Les etiquettes d une fente du bassin (octet `fente + 0x10`, FUN_1407ee87c / FUN_142ecf980).
const (
	fenteEteinte      = 0 // rien d autre n est ecrit
	fenteQuatreChamps = 1 // FUN_142ba78dc (trois reels et la queue), borne DAT_143cd86b4 = 3600,0
	// 2 et 3 : FUN_1424cd048 = FUN_140d580d0, borne DAT_143cd8a84 = 36000,0
)

// consumeMoteurDePartie est un maillon de la chaine de dispatch (cf. l en-tete de
// `dispatch_object.go`) : les composants `i11` a `i17` du moteur de partie, et `ti=45 i0` (le
// deroulement de partie, `components_matchflow_ti45.go`).
func consumeMoteurDePartie(br *Lecteur, name string, level uint32) (variant uint32, dead *types.DeadState, ported bool) {
	variant = noVariant
	switch name {
	case compGameEngineSoftCeilings: // ti=0/1/2 i11 (FUN_14116d1ac) — R(128)
		br.Skip(largeurPlafondsDeMoteur)
	case compGameEngineDisabledKillVolumes: // ti=0/1/2 i13 (FUN_142f03498) — R(13) puis n x R(1)
		consumeGameEngineDisabledKillVolumes(br)
	case compGameEngineComposerLetterbox: // ti=0/1/2 i14 (FUN_142f0328c) — deux formes, selon le niveau
		consumeGameEngineComposerLetterbox(br, level)
	case compManagedEngineTimers: // ti=0/2 i15 (FUN_1407ee7b8) — R(64) puis les fentes presentes
		consumeManagedEngineTimers(br)
	case compScenarioIntro: // ti=0/2 i16 (FUN_1410d9004) — R(7) + R(1)
		br.ReadBits(largeurScenarioIntro)
		br.ReadBit()
	case compMatchflowIsPlayingFlags: // ti=0/2 i17 (FUN_141101038) — R(8)
		br.ReadBits(largeurDrapeauxMatchflow)
	case compMatchflowSequenceData: // ti=45 i0 (FUN_14101cdd8) — R(4) + 4 x R(32)
		consumeMatchflowSequenceData(br)
	case compMatchflowFocusData: // ti=45 i1 (FUN_141167744) — R(6) + R(4)
		consumeMatchflowFocusData(br)
	default:
		return consumeComposantsVueBM4b(br, name)
	}
	return variant, nil, true
}

// consumeGameEngineDisabledKillVolumes porte FUN_142f03498 : un compte R(13), puis un bit par
// volume (champ de bits a `etat + 0x164`). L ecrivain FUN_142f06530 ecrit le compte global puis
// autant de bits (FUN_1406d49c4).
func consumeGameEngineDisabledKillVolumes(br *Lecteur) {
	n := br.ReadBits(largeurCompteVolumes)
	br.Skip(int(n))
}

// consumeGameEngineComposerLetterbox porte FUN_142f0328c.
//
//	tronc commun   R(1) -> etat+0x564 ; reel R(16) sur [0, DAT_143cd8374] -> etat+0x568 ;
//	               4 x FUN_142efd284 (R(1) ; si 0 : R(7)) -> etat+0x56c..0x57b
//	niveau >= 2    4 x [ R(1) ; si 1 : R(16) ] -> etat+0x57c.. (l ecrivain FUN_142f06308 n ecrit
//	               le mot que s il ne vaut pas 0xffff)
//	niveau < 2     FUN_1406d676c(.., n = 0x40) = R(64)
func consumeGameEngineComposerLetterbox(br *Lecteur, level uint32) {
	br.ReadBit()
	br.ReadBits(largeurLetterboxReel)
	for range entreesLetterbox {
		if !br.ReadBit() {
			br.ReadBits(largeurLetterboxIndex)
		}
	}
	if level < niveauLetterboxLong {
		br.Skip(largeurLetterboxCourte)
		return
	}
	for range entreesLetterbox {
		if br.ReadBit() {
			br.ReadBits(largeurLetterboxMot)
		}
	}
}

// consumeManagedEngineTimers porte FUN_1407ee7b8 : le masque R(64) des 64 fentes du bassin, puis,
// pour chaque bit k pose (k croissant), le lecteur de fente FUN_1407ee87c : etiquette R(2) ;
// 0 -> rien ; 1 -> FUN_142ba78dc (n = 16) ; 2 ou 3 -> FUN_140d580d0 (n = 16). L ecrivain
// FUN_142edad74 ecrit le meme masque puis, par fente presente, FUN_142ecf980 (etiquette sur 2 bits,
// puis FUN_142ba7c74 ou FUN_142b6f75c).
func consumeManagedEngineTimers(br *Lecteur) {
	masque := br.ReadBits(largeurMasqueMinuteurs)
	for k := range fentesMinuteurs {
		if masque&(uint64(1)<<uint(k)) == 0 {
			continue
		}
		switch br.ReadBits(largeurEtiquetteFente) {
		case fenteEteinte:
		case fenteQuatreChamps:
			lireMinuteur142ba78dc(br, largeurMinuteurFente)
		default:
			lireMinuteur140d580d0(br, largeurMinuteurFente)
		}
	}
}
