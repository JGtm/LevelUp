/**
 * i18n de la feature match-replay (rejeu 2D vue du dessus). Strings UI FR + EN,
 * parité par typage Record<ReplayLocale, ReplayText>.
 */
import type { Locale } from '@/lib/i18n/locale'

/** Alias local du type central : la feature ne redéclare pas l'union des langues. */
export type ReplayLocale = Locale

interface ReplayText {
  title: string
  back: string
  play: string
  pause: string
  restart: string
  loading: string
  empty: string
  note: string
  livesSuffix: string
  aliveSuffix: string
  speed: string
  time: string
  floor: string
  floorAll: string
  floorLow: string
  floorMid: string
  floorHigh: string
  propsSuffix: string
  /** Fond de carte figé : présent, ou remplacé par le sol reconstruit. */
  mapBackgroundNote: string
  mapBackgroundFallback: string
  /** Kill feed synchronisé sur l'horloge du rejeu. */
  killFeedTitle: string
  killFeedEmpty: string
  killFeedUnknownWeapon: string
  killFeedTotal: (n: number) => string
  /** Calques que le lecteur peut éteindre. */
  layers: string
  layerAim: string
  layerShield: string
  layerAimHint: string
  layerShieldHint: string
  /** Fiches joueur : ce qui est lu, et ce qui ne l'est pas. */
  rosterEmpty: string
  teamUnknown: string
  unknownPlayer: string
  shieldUnread: string
  healthUnread: string
  healthLabel: string
  shieldLabel: string
  abilityLabel: string
  loadoutUnread: string
  loadoutAge: string
  weaponInHand: string
  weaponInHandHint: string
  weaponSecondaryHint: string
  weaponsHolstered: string
  weaponsHolsteredShort: string
  weaponSwap: string
  weaponSwapHint: string
  respawnIn: string
  respawnUnknown: string
  respawnUnknownHint: string
  /** Ligne d'inventaire : grenades, capacité, munitions. */
  inventoryAge: string
  grenadeSelected: string
  grenadeSelectedRead: string
  grenadeSelUnknown: string
  grenadeSelUnknownHint: string
  abilityUnknown: string
  abilityUnknownHint: string
  ammoNone: string
  ammoNoneHint: string
  ammoSlotHint: string
  ammoDrawnHint: string
  drawnUnknown: string
  drawnUnknownHint: string
  gaugeLabel: string
  respawnBarLabel: string
  /** Bandeau de couverture : ce que chaque calque a rattaché sur ce qui existait. */
  coverageTitle: string
  coverageHint: string
  layerShots: string
  layerGrenades: string
  causeNoSlot: string
  causeAmbiguous: string
  causeOutOfWindow: string
  causeUnpublished: string
  bridgeRead: string
  bridgeNotRead: string
  coverageLeak: string
}

export const REPLAY_TEXT: Record<ReplayLocale, ReplayText> = {
  fr: {
    title: 'Rejeu 2D',
    back: 'Retour au match',
    play: 'Lecture',
    pause: 'Pause',
    restart: 'Recommencer',
    loading: 'Chargement du rejeu…',
    empty: 'Aucun rejeu 2D disponible pour ce match.',
    note: 'Trajectoires décodées du film (vue du dessus). Une trace = une vie : un joueur qui réapparaît repart sur une nouvelle trace, sans attribution de joueur. La lecture « 1× » suit le temps réel du match.',
    livesSuffix: 'vies',
    aliveSuffix: 'en jeu',
    speed: 'Vitesse',
    time: 'Temps de match',
    floor: 'Étage',
    floorAll: 'Tous',
    floorLow: 'Bas',
    floorMid: 'Milieu',
    floorHigh: 'Haut',
    propsSuffix: 'objets de décor',
    mapBackgroundNote:
      'Fond : la carte du jeu, reconstruite hors ligne et calée au mètre sur le repère du rejeu.',
    mapBackgroundFallback:
      "Fond : sol reconstruit — cette carte n'a pas encore d'image figée.",
    killFeedTitle: 'Éliminations',
    killFeedEmpty: 'Rien à cet instant du match.',
    killFeedUnknownWeapon: 'Arme non identifiée',
    killFeedTotal: (n) => `${n} sur le match`,
    layers: 'Calques',
    layerAim: 'Visée',
    layerShield: 'Bouclier',
    layerAimHint:
      "Direction du regard, décodée du même enregistrement que la position. Le jeu ne la retransmet que lorsqu'elle change : une mesure ancienne pâlit au lieu de disparaître, et rien n'est dessiné au-delà de cinq secondes.",
    layerShieldHint:
      "Bouclier lu dans le même enregistrement que la position. Une piste vide soulignée d'un trait est un bouclier BRISÉ, pas une absence de mesure.",
    rosterEmpty:
      "Aucune vie du film n'a pu être rattachée à un joueur : le rejeu reste anonyme.",
    teamUnknown: 'Sans équipe',
    unknownPlayer: 'Joueur inconnu',
    shieldUnread: 'bouclier non transmis',
    healthUnread: 'santé non transmise',
    healthLabel: 'Santé',
    shieldLabel: 'Bouclier',
    abilityLabel: "Capacité d'armure équipée",
    loadoutUnread: 'armes non lues sur cette vie',
    loadoutAge: 'Armes lues il y a',
    weaponInHand: 'en main',
    weaponInHandHint:
      "Arme dégainée à la dernière image-clé lue (~20 s d'écart entre deux) : un changement de main entre deux lectures est invisible.",
    weaponSecondaryHint: 'secondaire (arme rangée à la dernière lecture)',
    weaponsHolstered: 'Rien en main à la dernière lecture — armes rangées',
    weaponsHolsteredShort: 'rangées',
    weaponSwap: 'échange',
    weaponSwapHint:
      'Dotation changée entre les deux dernières images-clés (~20 s) — état de référence, pas un suivi continu :',
    respawnIn: 'retour dans',
    respawnUnknown: 'retour ?',
    respawnUnknownHint:
      "Le film ne porte aucune vie suivante pour ce joueur : le délai n'est pas lisible, et il n'est pas deviné. Le cas se produit en fin de partie, que le film ne clôt par aucun événement.",
    inventoryAge: 'Inventaire lu il y a',
    grenadeSelected: 'Type équipé : le seul porté, donc celui qui partira au prochain lancer.',
    grenadeSelectedRead:
      'Type équipé, LU dans le film (sélecteur de grenade de l’image-clé) : celui qui partira au prochain lancer.',
    grenadeSelUnknown: 'sél. ?',
    grenadeSelUnknownHint:
      'Plusieurs types portés et sélecteur non lu sur ce record : le type équipé est INDÉTERMINÉ, il n’est pas deviné.',
    abilityUnknown: 'capacité inconnue',
    abilityUnknownHint:
      "La table des capacités est partielle : 4 index observés pour 11 capacités dans le jeu. Un index hors table garde son numéro plutôt que d'emprunter le nom d'une capacité voisine.",
    ammoNone: 'aucune',
    ammoNoneHint:
      "Le film n'ecrit rien pour cet emplacement. Pour une arme a charge, cela veut dire PLEIN : le flux est differentiel, et le plein est la valeur par defaut, donc il n'est jamais transmis.",
    gaugeLabel: 'charge restante',
    respawnBarLabel: 'avancement depuis la mort',
    ammoSlotHint:
      "Munitions de l'emplacement portant ce numéro dans l'enregistrement, dans l'ordre des armes ci-dessus. Sur les 198 appariements dont la lecture est unique, 197 concordent avec l'arme de même rang. Une lecture à plusieurs candidats, elle, n'est pas fiable : elle est signalée à part.",
    ammoDrawnHint:
      'Emplacement DÉGAINÉ selon le sélecteur du record : la même lecture qui place cette arme en tête de rangée.',
    drawnUnknown: 'dégainée ?',
    drawnUnknownHint:
      'Le sélecteur d’emplacement n’a pas été lu sur ce record : on ne sait pas quelle arme est en main.',
    coverageTitle: 'Ce qui est rattaché',
    coverageHint:
      "Le dénominateur est ce que LE FILM porte, pas ce que la partie a compté : le film n'enregistre que les tirs qui infligent un dégât.",
    layerShots: 'Tirs',
    layerGrenades: 'Lancers de grenade',
    causeNoSlot: 'aucune trace connue pour ce joueur à cet instant',
    causeAmbiguous: 'plusieurs traces possibles',
    causeOutOfWindow: 'aucune position assez proche dans le temps',
    causeUnpublished: 'trace trop courte pour être publiée',
    bridgeRead: 'Rattachement lu dans le film',
    bridgeNotRead: 'Rattachement incomplet — une partie des événements n’est pas publiée',
    coverageLeak: 'Comptage incohérent : des événements manquent à l’appel',
  },
  en: {
    title: '2D replay',
    back: 'Back to match',
    play: 'Play',
    pause: 'Pause',
    restart: 'Restart',
    loading: 'Loading replay…',
    empty: 'No 2D replay available for this match.',
    note: 'Trajectories decoded from the film (top-down). One trail = one life: a respawning player starts a new trail, with no per-player attribution. Playback at 1x follows the real match time.',
    livesSuffix: 'lives',
    aliveSuffix: 'on the map',
    speed: 'Speed',
    time: 'Match time',
    floor: 'Floor',
    floorAll: 'All',
    floorLow: 'Lower',
    floorMid: 'Mid',
    floorHigh: 'Upper',
    propsSuffix: 'map props',
    mapBackgroundNote: 'Background: the in-game map, rebuilt offline and aligned to the metre with the replay frame of reference.',
    mapBackgroundFallback: 'Background: reconstructed floor — this map has no baked image yet.',
    killFeedTitle: 'Kills',
    killFeedEmpty: 'Nothing at this point of the match.',
    killFeedUnknownWeapon: 'Unidentified weapon',
    killFeedTotal: (n) => `${n} in the match`,
    layers: 'Layers',
    layerAim: 'Aim',
    layerShield: 'Shield',
    layerAimHint:
      'Look direction, decoded from the same record as the position. The game only retransmits it when it changes: an older reading fades instead of vanishing, and nothing is drawn beyond five seconds.',
    layerShieldHint:
      'Shield read from the same record as the position. An empty bar with a line under it is a BROKEN shield, not a missing measurement.',
    rosterEmpty: 'No life from the film could be attached to a player: the replay stays anonymous.',
    teamUnknown: 'No team',
    unknownPlayer: 'Unknown player',
    shieldUnread: 'shield not transmitted',
    healthUnread: 'health not transmitted',
    healthLabel: 'Health',
    shieldLabel: 'Shield',
    abilityLabel: 'Equipped armor ability',
    loadoutUnread: 'weapons not read on this life',
    loadoutAge: 'Weapons read',
    weaponInHand: 'in hand',
    weaponInHandHint:
      'Weapon drawn at the last keyframe read (~20 s between two): a hand change between two readings is invisible.',
    weaponSecondaryHint: 'secondary (weapon holstered at the last reading)',
    weaponsHolstered: 'Nothing drawn at the last reading — weapons holstered',
    weaponsHolsteredShort: 'holstered',
    weaponSwap: 'swap',
    weaponSwapHint:
      'Loadout changed between the two last keyframes (~20 s) — a reference state, not continuous tracking:',
    respawnIn: 'back in',
    respawnUnknown: 'back ?',
    respawnUnknownHint:
      'The film carries no next life for this player: the delay is not readable, and it is not guessed. This happens at the end of the match, which the film closes with no event.',
    inventoryAge: 'Inventory read',
    grenadeSelected: 'Equipped type: the only one carried, so the one the next throw will use.',
    grenadeSelectedRead:
      'Equipped type, READ from the film (keyframe grenade selector): the one the next throw will use.',
    grenadeSelUnknown: 'sel. ?',
    grenadeSelUnknownHint:
      'Several types carried and no selector read on this record: the equipped type is UNDETERMINED, it is not guessed.',
    abilityUnknown: 'unknown ability',
    abilityUnknownHint:
      'The ability table is partial: 4 indices observed for 11 abilities in the game. An index outside the table keeps its number rather than borrowing a neighbouring ability name.',
    ammoNone: 'none',
    ammoNoneHint:
      'The film writes nothing for this slot. For a charge weapon that means FULL: the stream is differential, and full is the default value, so it is never transmitted.',
    gaugeLabel: 'charge left',
    respawnBarLabel: 'progress since death',
    ammoSlotHint:
      'Ammo for the record slot bearing this number, in the order of the weapons above. Across the 198 pairings whose reading is unique, 197 agree with the weapon of the same rank. A reading with several candidates is not reliable and is flagged separately.',
    ammoDrawnHint:
      'Slot DRAWN according to the record selector: the same reading that puts this weapon first in the row.',
    drawnUnknown: 'drawn ?',
    drawnUnknownHint:
      'The slot selector was not read on this record: which weapon is in hand is unknown.',
    coverageTitle: 'What is attached',
    coverageHint:
      'The denominator is what THE FILM carries, not what the match counted: the film only records shots that deal damage.',
    layerShots: 'Shots',
    layerGrenades: 'Grenade throws',
    causeNoSlot: 'no known trail for that player at that moment',
    causeAmbiguous: 'several possible trails',
    causeOutOfWindow: 'no position close enough in time',
    causeUnpublished: 'trail too short to be published',
    bridgeRead: 'Attachment read from the film',
    bridgeNotRead: 'Incomplete attachment — some events are not published',
    coverageLeak: 'Inconsistent count: events are unaccounted for',
  },
}
