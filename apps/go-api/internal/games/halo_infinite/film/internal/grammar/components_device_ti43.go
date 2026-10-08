package grammar

import "levelup/go-api/internal/games/halo_infinite/film/types"

// components_device_ti43.go — LES COMPOSANTS `device-*` DE L ARCHETYPE `ti=43` (dispositif de
// carte : porte, ascenseur, station de soin, distributeur d armes ou de vehicules), `i18` a `i40`.
//
// # CE QUI FONDE CHAQUE LARGEUR
//
// Chaque lecteur ci-dessous est celui du jeu : la fonction a `descripteur + 0x40` du composant
// dans `HaloInfinite.exe` (HI_1_13_0), que `FUN_14076cb60` appelle pour tout composant present
// au masque. Son adresse est sur chaque `case`. Pour les composants a branches (`i21`, `i34`,
// `i35`), l ECRIVAIN (`descripteur + 0x28`) a ete relu en plus : il dit quand un champ
// conditionnel est ecrit, donc ce que le lecteur doit lire. Notes de releve :
// `.ai/V7.5/film_re/NOTE_3_6_TI43_GRAMMAIRES_A_2026-09-17.md` (`i19`..`i29`), `_C_` (`i30`..`i35`),
// `_D_` (`i36`..`i40`), et `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/T7_dispositifs_ti43_moteur.md`
// (ecrivains).
//
// # CE QUE LE LECTEUR DU JEU NE FAIT PAS, ET CE PORT NON PLUS
//
//   - Aucun de ces lecteurs ne lit le NIVEAU que le registre du film declare (trois parametres,
//     `R9` jamais lu) : le niveau n entre pas ici.
//   - Aucune largeur ne depend d un etat du processus. Les seules entrees hors du flux sont les
//     plages des references d entite de `FUN_1406d3140` (categories 0 et 1), que
//     [consume1408f0ac4] lit dans le profil du film.
//   - Les valeurs quantifiees (`FUN_1406d84b4`) ne sont pas dequantifiees : aucun consommateur ne
//     les lit, seule la largeur sert a fermer le record. La condition d `i35` se juge sur l entier
//     brut, ce que l ecrivain rend exact (voir [consumeDeviceAnimationLayerState]).
//
// # UN LECTEUR QUI ECHOUE ARRETE LE RECORD
//
// Le lecteur d `i31` rend 0 quand le compte lu depasse la capacite de l objet (8) ; la boucle de
// composants du jeu tue alors le record. Ce port rend `ported = false` : la traversee s arrete au
// composant (`EntityTrace.DesyncAt`), sans borner le compte ni deviner la suite.

// Les etiquettes de registre des composants `device-*` (bloc `ti=43` de `chunk_00`).
const (
	compDevicePosition               = "device-position-component"
	compDevicePositionAnimationName  = "device-position-animation-name-component"
	compDevicePositionAnimationCtrl  = "device-position-animation-control-component"
	compDevicePositionGroup          = "device-position-group-component"
	compDevicePower                  = "device-power-component"
	compDevicePowerGroup             = "device-power-group-component"
	compDeviceInteractionInProgress  = "device-interaction-in-progress-component"
	compDeviceInteractionHoldTime    = "device-interaction-hold-time-component"
	compDeviceActionStringOverride   = "device-control-action-string-override-component"
	compDeviceHealthStationCharges   = "device-health-station-charges-component"
	compDeviceHealthStationInUse     = "device-health-station-in-use-component"
	compDeviceExclusiveUser          = "device-exclusive-user-component"
	compDeviceInPrimaryMode          = "device-in-primary-mode-component"
	compDeviceDispenserMonitors      = "device-dispenser-monitors-changed-component"
	compDeviceDispenserStateFlags    = "device-dispenser-state-flags-component"
	compDeviceDispenserRequireLOS    = "device-dispenser-require-los-component"
	compDeviceAnimationLayerSettings = "device-animation-layer-settings-component"
	compDeviceAnimationLayerState    = "device-animation-layer-state-component"
	compDeviceDispenserState         = "device-dispenser-state-component"
	compDeviceObjectDispenserTimer   = "device-object-dispenser-timer-component"
	compDeviceTransitionVelocity     = "device-position-transition-velocity-component"
	compDeviceMachineFlags           = "device-machine-flags-component"
	compDeviceInteractionStartTime   = "device-interaction-start-time-override-component"
)

// Largeurs des champs, lues aux sites d appel (`MOV R9D, n`, `MOV [RSP+0x20], n`, `ADD [flux+0x2c], n`).
const (
	largeurMotBrut              = 32  // FUN_141015740, FUN_14080dec4, FUN_1406d676c(n = 0x20)
	largeurPositionAnimation    = 10  // i19 : Q(10 ; 0..10)
	largeurControleAnimation    = 256 // i20 : FUN_1406d676c(n = 0x100), quatre tours de 64
	largeurIndexDeGroupe        = 8   // i21 : FUN_1407f08f8 sous la porte de FUN_1407f08bc
	largeurPuissance            = 14  // i22, i23 : Q(14 ; 0..1)
	largeurTempsInteraction     = 8   // i25, i40 : Q(8 ; 0..60)
	largeurCharges              = 6   // i27 : +0x2c += 6 apres le drapeau
	largeurCompteMoniteurs      = 8   // i31 : +0x2c += 8
	capaciteMoniteurs           = 8   // i31 : `CMP R9D, 0x8 ; JLE`, tableau de 8 entrees a +0x5a4
	largeurEtatDistributeur     = 5   // i32 : FUN_1424ccc74
	couchesDAnimation           = 8   // i34, i35 : (0x6a8 - 0x5e8) / 0x18 et (0x6e8 - 0x6a8) / 8
	largeurReglageCouche14      = 14  // i34 : les trois Q(14) d une couche
	largeurReglageCouche10      = 10  // i34 : le Q(10 ; 0..1) d une couche
	largeurDrapeauxCouche       = 2   // i34 : FUN_142af27f8
	largeurPoidsCouche          = 10  // i35 : Q(10 ; 0..1), le poids
	largeurValeurCouche         = 14  // i35 : Q(14 ; 0..1), lue si le poids est non nul
	emplacementsDistributeur    = 2   // i36, i37 : deux entrees (0x18 / 0xc et 0x20 / 0x10)
	largeurEtatEmplacement      = 3   // i36 : `ADD [flux+0x2c], 3` inconditionnel par entree
	largeurMinuteurDistributeur = 10  // i37 : `MOV R8D, 0xa` passe a FUN_142ba78dc
	largeurVitesseTransition    = 18  // i38 : Q(18 ; 0..29,999998)
	largeurDrapeauxMachine      = 9   // i39 : `ADD [flux+0x2c], 9`
	categorieReferenceSonde     = 1   // `MOV R8D, 0x1` avant FUN_1408f0ac4 (i24, i29, i31)
	categorieReferenceSansSonde = 0   // `XOR R8D, R8D` avant FUN_1408f0ac4 (i36)
)

// consumeComposantsDispositif est le maillon des composants `device-*` (cf. l en-tete de
// `dispatch_object.go`). Son `default` passe la main au maillon suivant.
func consumeComposantsDispositif(br *Lecteur, name string) (variant uint32, dead *types.DeadState, ported bool) { //nolint:gocyclo // un case par composant du registre
	variant = noVariant
	switch name {
	case compDevicePosition: // i18 (FUN_140bef320) — R(14) + R(1)
		consumeDevicePosition(br)
	case compDevicePositionAnimationName: // i19 (FUN_1410156e4) — R(32) + Q(10)
		br.Skip(largeurMotBrut + largeurPositionAnimation)
	case compDevicePositionAnimationCtrl: // i20 (FUN_142f02d04) — R(256) brut
		br.Skip(largeurControleAnimation)
	case compDevicePositionGroup: // i21 (FUN_1407f0678 ; ecrivain FUN_142f05cd0) — R(32) + porte [R(8)]
		br.Skip(largeurMotBrut)
		consumeGateR(br, largeurIndexDeGroupe) // FUN_1407f08bc
	case compDevicePower, compDevicePowerGroup: // i22 (FUN_14100d310), i23 (FUN_14100d2d0) — Q(14)
		br.Skip(largeurPuissance)
	case compDeviceInteractionInProgress, compDeviceExclusiveUser: // i24 (FUN_141167910), i29 (FUN_14116fcb0)
		consume1408f0ac4(br, categorieReferenceSonde)
	case compDeviceInteractionHoldTime, compDeviceInteractionStartTime: // i25 (FUN_142f02c54), i40 (FUN_141fd7bc0) — Q(8)
		br.Skip(largeurTempsInteraction)
	case compDeviceActionStringOverride: // i26 (FUN_142f029e4) — 2 x FUN_14080dec4 = 2 x R(32)
		br.Skip(2 * largeurMotBrut)
	case compDeviceHealthStationCharges: // i27 (FUN_140bee524) — R(1) + R(6)
		br.Skip(1 + largeurCharges)
	case compDeviceHealthStationInUse, compDeviceInPrimaryMode, compDeviceDispenserRequireLOS:
		// i28 (FUN_142f02bec), i30 (FUN_142f02c20), i33 (FUN_142f02b7c) — un bit de l octet +0x582
		br.Skip(1)
	case compDeviceDispenserMonitors: // i31 (FUN_142f02a48) — R(8) N ; N > 8 : le lecteur du jeu rend 0
		if !consumeDeviceDispenserMonitors(br) {
			return variant, nil, false
		}
	case compDeviceDispenserStateFlags: // i32 (FUN_142f02bcc) — R(5)
		br.Skip(largeurEtatDistributeur)
	case compDeviceAnimationLayerSettings: // i34 (FUN_140f44104 -> FUN_143206e48)
		consumeDeviceAnimationLayerSettings(br)
	case compDeviceAnimationLayerState: // i35 (FUN_141076f68 -> FUN_143206f24 ; ecrivain FUN_142f0570c)
		consumeDeviceAnimationLayerState(br)
	case compDeviceDispenserState: // i36 (FUN_142f02bb0 -> FUN_143206ae0)
		consumeDeviceDispenserState(br)
	case compDeviceObjectDispenserTimer: // i37 (FUN_142f02c94 -> FUN_142ba78dc)
		consumeDeviceObjectDispenserTimer(br)
	case compDeviceTransitionVelocity: // i38 (FUN_142f02d28) — Q(18)
		br.Skip(largeurVitesseTransition)
	case compDeviceMachineFlags: // i39 (FUN_14107bb68) — R(9)
		br.Skip(largeurDrapeauxMachine)
	default:
		return consumeComposantsVehiculeTi40(br, name)
	}
	return variant, nil, true
}

// consumeDeviceDispenserMonitors porte `FUN_142f02a48` : `N = R(8)`, puis N references d entite
// de categorie 1 (`FUN_1408f0ac4`). Rend faux, apres les 8 bits du compte, quand `N > 8` : le
// lecteur du jeu rend alors 0 sans lire plus (`142f02b1e CMP R9D, 0x8 ; JLE`, `XOR AL, AL`).
func consumeDeviceDispenserMonitors(br *Lecteur) bool {
	n := br.ReadBits(largeurCompteMoniteurs)
	if n > capaciteMoniteurs {
		return false
	}
	for range n {
		consume1408f0ac4(br, categorieReferenceSonde)
	}
	return true
}

// consumeDeviceAnimationLayerSettings porte `FUN_140f44104` : une porte A, puis, si elle vaut 1,
// huit couches `FUN_143206e48` = porte G (`FUN_143206d34`) puis, si G, `R(32)` + `Q(14)` +
// `Q(14)` + `Q(10)` + `Q(14)` + `R(2)` (87 bits par couche ouverte avec sa porte).
func consumeDeviceAnimationLayerSettings(br *Lecteur) {
	if !br.ReadBit() {
		return
	}
	for range couchesDAnimation {
		if br.ReadBit() {
			br.Skip(largeurMotBrut + 3*largeurReglageCouche14 + largeurReglageCouche10 + largeurDrapeauxCouche)
		}
	}
}

// consumeDeviceAnimationLayerState porte `FUN_141076f68` : une porte A, puis, si elle vaut 1, huit
// entrees = porte G puis, si G, `FUN_143206f24` : un poids `Q(10 ; 0..1)` et, SI LE POIDS EST
// NON NUL, une valeur `Q(14 ; 0..1)`.
//
// La condition du jeu est `poids > 0,0f` sur la valeur dequantifiee (`COMISS`, `JBE`). Le
// dequantificateur (`FUN_1406d84b4`, `b7 = 1`) rend exactement 0 pour le code 0 et une valeur
// strictement positive pour tout autre code ; l ecrivain (`FUN_1432070d0`) n ecrit le code 0
// que pour un poids `<= 0` et ecrit alors la valeur seulement si le poids brut est `> 0`. Les
// deux cotes coincident donc sur `code != 0`, que ce port teste sur l entier.
func consumeDeviceAnimationLayerState(br *Lecteur) {
	if !br.ReadBit() {
		return
	}
	for range couchesDAnimation {
		if br.ReadBit() && br.ReadBits(largeurPoidsCouche) != 0 {
			br.Skip(largeurValeurCouche)
		}
	}
}

// consumeDeviceDispenserState porte `FUN_143206ae0` : deux entrees = porte EXTERNE G, puis si G
// une reference d entite de CATEGORIE 0 (`FUN_1408f0ac4`, sans bit de sonde), puis un etat `R(3)`
// lu dans tous les cas.
func consumeDeviceDispenserState(br *Lecteur) {
	for range emplacementsDistributeur {
		if br.ReadBit() {
			consume1408f0ac4(br, categorieReferenceSansSonde)
		}
		br.Skip(largeurEtatEmplacement)
	}
}

// consumeDeviceObjectDispenserTimer porte `FUN_142f02c94` : deux entrees = porte G puis, si G,
// `FUN_142ba78dc(n = 10)` ([lireMinuteur142ba78dc]).
func consumeDeviceObjectDispenserTimer(br *Lecteur) {
	for range emplacementsDistributeur {
		if br.ReadBit() {
			lireMinuteur142ba78dc(br, largeurMinuteurDistributeur)
		}
	}
}
