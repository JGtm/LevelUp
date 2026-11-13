/**
 * useReplayTerrainFlags — LES INTERRUPTEURS DES CALQUES DU TERRAIN du rejeu : poses d'équipement,
 * socles et armes au sol, objectifs du mode, objets portés (drapeau, couronne, crâne, bombe) et
 * véhicules. Chacun est une préférence persistée (`replayPreferences.ts`) lue au montage.
 *
 * EXTRAIT DE `useReplaySettings.ts` sans changer un comportement : la clé, le défaut et la
 * bascule de chaque calque sont ceux d'avant. `useReplaySettings` en reprend l'objet tel quel.
 */
import { usePersistedFlag } from './usePersistedFlag'

const SHOW_PLACEMENTS_KEY = 'replay-show-placements'
const SHOW_UNNAMED_PLACEMENTS_KEY = 'replay-show-unnamed-placements'
const SHOW_DROPPED_PLACEMENTS_KEY = 'replay-show-dropped-placements'
const SHOW_WEAPON_PADS_KEY = 'replay-show-weapon-pads'
const SHOW_GROUND_WEAPONS_KEY = 'replay-show-ground-weapons'
const SHOW_GROUND_WEAPONS_SPECIAL_ONLY_KEY = 'replay-show-ground-weapons-special-only'
const SHOW_GROUND_WEAPONS_STARTING_KEY = 'replay-show-ground-weapons-starting'
const SHOW_MODE_OBJECTIVES_KEY = 'replay-show-mode-objectives'
const SHOW_FLAG_CARRIES_KEY = 'replay-show-flag-carries'
const SHOW_VIP_CROWN_KEY = 'replay-show-vip-crown'
const SHOW_SKULL_CARRIER_KEY = 'replay-show-skull-carrier'
const SHOW_BOMB_CARRIER_KEY = 'replay-show-bomb-carrier'
const SHOW_VEHICLES_KEY = 'replay-show-vehicles'

/**
 * LES DEUX BASCULES DES POSES D'ÉQUIPEMENT, et leurs défauts OPPOSÉS eux aussi (décision
 * utilisateur du 18/08) :
 *
 *  - les POSES sont ALLUMÉES : le mur et le capteur sont des objets du terrain qui changent
 *    la lecture d'un échange, au même titre qu'une zone de capture ;
 *  - les OBJETS NON IDENTIFIÉS sont ÉTEINTS, et ce n'est pas de la pudeur : sur un film de
 *    Big Team Battle, AUCUNE pose n'est nommée aujourd'hui (le seuil de nommage du 18/08
 *    laisse la palette « famille A » sans nom) — les allumer y poserait des centaines de
 *    points neutres. La bascule sert exactement à cela : voir ce que la mesure a trouvé sans
 *    savoir le nommer, quand on le demande.
 *
 * Ni l'une ni l'autre n'est un demi-livrable (CLAUDE.md n°11) : les deux rendus sont livrés
 * et complets, l'interrupteur est un RÉGLAGE D'AFFICHAGE offert au lecteur.
 */
const SHOW_PLACEMENTS_DEFAULT = true
const SHOW_UNNAMED_PLACEMENTS_DEFAULT = false

/**
 * LES OBJETS DE PUISSANCE LÂCHÉS SONT ALLUMÉS PAR DÉFAUT (décision utilisateur du 2026-08-18 :
 * « hors Fiesta c'est tactiquement interessant »). Un surbouclier ou un capteur au sol est
 * RAMASSABLE : savoir qu'il est là change la lecture de l'échange suivant, exactement comme
 * savoir qu'une arme est encore sur son socle.
 *
 * LA BASCULE COMMANDE DANS TOUS LES MODES depuis le 2026-08-20. Une garde de mode l'annulait
 * en Fiesta au nom de la même décision du 18/08 ; elle masquait en fait 26 lâchers réels sur
 * le témoin Fiesta `000d5950`, et l'utilisateur veut les voir. Le tiroir affiche donc la
 * commande dès que le film porte de quoi la commander — même règle que le bouton Zones.
 *
 * Ce n'est pas un demi-livrable (CLAUDE.md n°11) : le calque est complet, l'interrupteur est un
 * RÉGLAGE D'AFFICHAGE offert au lecteur.
 */
const SHOW_DROPPED_PLACEMENTS_DEFAULT = true

/**
 * LES SOCLES D'ARME SONT ALLUMÉS PAR DÉFAUT (décision utilisateur du 18/08, W4 : « les infos
 * sont intéressantes à avoir »). Ce sont des lieux du terrain, pas des événements : savoir que
 * le fusil de précision est encore sur son socle change la lecture d'un échange, exactement
 * comme une zone de capture. Le film qui n'en porte aucun n'affiche ni calque ni bascule.
 */
const SHOW_WEAPON_PADS_DEFAULT = true

/**
 * LES ARMES AU SOL SONT ALLUMÉES PAR DÉFAUT (schéma 27). Même raisonnement que les objets de
 * puissance lâchés, décidé le 2026-08-18 : une arme ramassable par terre change la lecture de
 * l'échange suivant, au même titre qu'une arme encore sur son socle. Le film qui n'en porte
 * aucune n'affiche ni calque ni bascule.
 *
 * Ce n'est pas un demi-livrable (CLAUDE.md n°11) : le calque est complet, l'interrupteur est un
 * RÉGLAGE D'AFFICHAGE offert au lecteur — une carte de Grande Bataille reste dense.
 */
const SHOW_GROUND_WEAPONS_DEFAULT = true

/**
 * LE FILTRE « ARMES SPÉCIALES SEULEMENT » EST ÉTEINT PAR DÉFAUT. C'est un RESSERREMENT du calque des armes au
 * sol, pas son remplacement : le comportement d'avant ce lot (tout afficher) reste le défaut,
 * et l'utilisateur choisit d'affiner sur les matchs denses (204 objets/film en moyenne, dont
 * 82 % sans intérêt tactique).
 */
const SHOW_GROUND_WEAPONS_SPECIAL_ONLY_DEFAULT = false

/**
 * LES ARMES DE DÉPART LÂCHÉES SONT MASQUÉES PAR DÉFAUT (décision utilisateur du 2026-10-15) :
 * tout le monde en porte les mêmes, elles n'apportent rien. L'option les réaffiche ; les armes
 * posées par la carte, elles, restent toujours visibles (`model/startingWeaponsOnGround.ts`).
 */
const SHOW_GROUND_WEAPONS_STARTING_DEFAULT = false

/**
 * LES DRAPEAUX SONT ALLUMÉS PAR DÉFAUT. C'est l'ENJEU du match en capture de drapeau : savoir
 * où est le drapeau, qui le porte et depuis quand est la lecture même du mode — un rejeu de CTF
 * qui s'ouvrirait sans lui montrerait huit points qui courent sans raison. Le film qui n'en
 * porte aucun (tout mode qui n'est pas de la capture) n'affiche ni calque ni bascule.
 *
 * Ce n'est pas un demi-livrable (CLAUDE.md n°11) : le calque est complet, l'interrupteur est un
 * RÉGLAGE D'AFFICHAGE offert au lecteur — un BTB de capture reste dense.
 */
const SHOW_FLAG_CARRIES_DEFAULT = true
const SHOW_VIP_CROWN_DEFAULT = true
const SHOW_SKULL_CARRIER_DEFAULT = true
const SHOW_BOMB_CARRIER_DEFAULT = true

/**
 * LES VÉHICULES SONT ALLUMÉS PAR DÉFAUT (schéma 39, lot du 2026-09-02) : c'est l'objectif même
 * du chantier (« rejouer un match ... avec véhicules visibles »), au même titre que les socles
 * ou les drapeaux — un objet du terrain qui change la lecture du match, pas un effet de style.
 * Le film qui n'en porte aucun (`coverage.vehicles`) n'affiche ni calque ni bascule.
 *
 * Ce n'est pas un demi-livrable (CLAUDE.md n°11) : le calque est complet, l'interrupteur est un
 * RÉGLAGE D'AFFICHAGE offert au lecteur — une carte de Grande Bataille reste dense.
 */
const SHOW_VEHICLES_DEFAULT = true

/**
 * LES OBJECTIFS DU MODE SONT ALLUMÉS PAR DÉFAUT (2026-09-18) : collines, bases, zones de capture,
 * socles et points de livraison sont le TERRAIN de l'enjeu — un Roi de la colline sans colline
 * montrerait des joueurs qui se battent pour rien. Le film qui n'en porte aucun (Assassin) n'affiche
 * ni calque ni bascule. Comme les drapeaux : un réglage d'affichage, pas un demi-livrable.
 */
const SHOW_MODE_OBJECTIVES_DEFAULT = true

/** Les interrupteurs de calques du terrain : un état et sa bascule par calque. */
export interface ReplayTerrainFlags {
  /** Calque des POSES d'équipement (mur, capteur). Allumé par défaut. */
  showPlacements: boolean
  togglePlacements: () => void
  /** Poses dont la nature n'est pas établie (famille `other`). ÉTEINT par défaut. */
  showUnnamedPlacements: boolean
  toggleUnnamedPlacements: () => void
  /**
   * Objets de PUISSANCE lâchés à la mort (power-ups, équipements déployables). ALLUMÉ par
   * défaut, et effectif dans TOUS les modes (cf. SHOW_DROPPED_PLACEMENTS_DEFAULT).
   */
  showDroppedPlacements: boolean
  toggleDroppedPlacements: () => void
  /** Calque des SOCLES D'ARME (schéma 11). Allumé par défaut (cf. SHOW_WEAPON_PADS_DEFAULT). */
  showWeaponPads: boolean
  toggleWeaponPads: () => void
  /**
   * Calque des ARMES AU SOL (schéma 27) — les armes abandonnées, distinctes des socles.
   * Allumé par défaut (cf. SHOW_GROUND_WEAPONS_DEFAULT).
   */
  showGroundWeapons: boolean
  toggleGroundWeapons: () => void
  /**
   * Filtre « ARMES SPÉCIALES SEULEMENT » du calque des armes au sol (lot 6.5, 2026-09-10) :
   * sniper/power/special uniquement. ÉTEINT par défaut (cf.
   * SHOW_GROUND_WEAPONS_SPECIAL_ONLY_DEFAULT).
   */
  showGroundWeaponsSpecialOnly: boolean
  toggleGroundWeaponsSpecialOnly: () => void
  /** Option « afficher aussi les armes de départ au sol » (cf. SHOW_GROUND_WEAPONS_STARTING_DEFAULT). */
  showGroundWeaponsStarting: boolean
  toggleGroundWeaponsStarting: () => void
  /**
   * Calque des OBJECTIFS DU MODE (2026-09-18) : la géométrie cuite, l'état vivant des zones et
   * les pulses d'action — trois calques pour un objet, une seule bascule (cf.
   * `SceneToggles.modeObjectives`). Allumé par défaut (cf. SHOW_MODE_OBJECTIVES_DEFAULT).
   */
  showModeObjectives: boolean
  toggleModeObjectives: () => void
  /**
   * Calque des DRAPEAUX de capture (schéma 15). Allumé par défaut : c'est l'enjeu du mode
   * (cf. SHOW_FLAG_CARRIES_DEFAULT). Un film hors capture n'en publie aucun.
   */
  showFlagCarries: boolean
  toggleFlagCarries: () => void
  /**
   * Calque de la COURONNE VIP (schéma 22). Allumé par défaut : c'est l'enjeu du mode
   * (cf. SHOW_VIP_CROWN_DEFAULT). Un film hors VIP n'en publie aucune.
   */
  showVipCrown: boolean
  toggleVipCrown: () => void
  /**
   * Calque du PORTEUR DU CRÂNE d'Oddball (schéma 23). Allumé par défaut : c'est l'enjeu du mode
   * (cf. SHOW_SKULL_CARRIER_DEFAULT). Un film hors Oddball n'en publie aucun.
   */
  showSkullCarrier: boolean
  toggleSkullCarrier: () => void
  /**
   * Calque de LA BOMBE d'Assaut (schéma 30) : portée sur son porteur, posée au sol. Allumé par
   * défaut : c'est l'enjeu du mode (cf. SHOW_BOMB_CARRIER_DEFAULT). Un film hors Assaut n'en
   * publie aucune.
   */
  showBombCarrier: boolean
  toggleBombCarrier: () => void
  /** Calque des VÉHICULES (schéma 39). Allumé par défaut (cf. SHOW_VEHICLES_DEFAULT). */
  showVehicles: boolean
  toggleVehicles: () => void
}

export function useReplayTerrainFlags(): ReplayTerrainFlags {
  const [showPlacements, togglePlacements] = usePersistedFlag(
    SHOW_PLACEMENTS_KEY,
    SHOW_PLACEMENTS_DEFAULT,
  )
  const [showUnnamedPlacements, toggleUnnamedPlacements] = usePersistedFlag(
    SHOW_UNNAMED_PLACEMENTS_KEY,
    SHOW_UNNAMED_PLACEMENTS_DEFAULT,
  )
  const [showDroppedPlacements, toggleDroppedPlacements] = usePersistedFlag(
    SHOW_DROPPED_PLACEMENTS_KEY,
    SHOW_DROPPED_PLACEMENTS_DEFAULT,
  )
  const [showWeaponPads, toggleWeaponPads] = usePersistedFlag(
    SHOW_WEAPON_PADS_KEY,
    SHOW_WEAPON_PADS_DEFAULT,
  )
  const [showGroundWeapons, toggleGroundWeapons] = usePersistedFlag(
    SHOW_GROUND_WEAPONS_KEY,
    SHOW_GROUND_WEAPONS_DEFAULT,
  )
  const [showGroundWeaponsSpecialOnly, toggleGroundWeaponsSpecialOnly] = usePersistedFlag(
    SHOW_GROUND_WEAPONS_SPECIAL_ONLY_KEY,
    SHOW_GROUND_WEAPONS_SPECIAL_ONLY_DEFAULT,
  )
  const [showGroundWeaponsStarting, toggleGroundWeaponsStarting] = usePersistedFlag(
    SHOW_GROUND_WEAPONS_STARTING_KEY,
    SHOW_GROUND_WEAPONS_STARTING_DEFAULT,
  )
  const [showModeObjectives, toggleModeObjectives] = usePersistedFlag(
    SHOW_MODE_OBJECTIVES_KEY,
    SHOW_MODE_OBJECTIVES_DEFAULT,
  )
  const [showFlagCarries, toggleFlagCarries] = usePersistedFlag(
    SHOW_FLAG_CARRIES_KEY,
    SHOW_FLAG_CARRIES_DEFAULT,
  )
  const [showVipCrown, toggleVipCrown] = usePersistedFlag(
    SHOW_VIP_CROWN_KEY,
    SHOW_VIP_CROWN_DEFAULT,
  )
  const [showSkullCarrier, toggleSkullCarrier] = usePersistedFlag(
    SHOW_SKULL_CARRIER_KEY,
    SHOW_SKULL_CARRIER_DEFAULT,
  )
  const [showBombCarrier, toggleBombCarrier] = usePersistedFlag(
    SHOW_BOMB_CARRIER_KEY,
    SHOW_BOMB_CARRIER_DEFAULT,
  )
  const [showVehicles, toggleVehicles] = usePersistedFlag(SHOW_VEHICLES_KEY, SHOW_VEHICLES_DEFAULT)

  return {
    showPlacements,
    togglePlacements,
    showUnnamedPlacements,
    toggleUnnamedPlacements,
    showDroppedPlacements,
    toggleDroppedPlacements,
    showWeaponPads,
    toggleWeaponPads,
    showGroundWeapons,
    toggleGroundWeapons,
    showGroundWeaponsSpecialOnly,
    toggleGroundWeaponsSpecialOnly,
    showGroundWeaponsStarting,
    toggleGroundWeaponsStarting,
    showModeObjectives,
    toggleModeObjectives,
    showFlagCarries,
    toggleFlagCarries,
    showVipCrown,
    toggleVipCrown,
    showSkullCarrier,
    toggleSkullCarrier,
    showBombCarrier,
    toggleBombCarrier,
    showVehicles,
    toggleVehicles,
  }
}
