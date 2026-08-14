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
  propsSuffix: string
  /** Fond de carte figé : présent, ou remplacé par le sol reconstruit. */
  mapBackgroundNote: string
  mapBackgroundFallback: string
  /**
   * Fin de vol d'une grenade : DERNIÈRE POSITION CONNUE, jamais « impact » — le film
   * n'enregistre aucune détonation (règle du plan parité, item 2.3).
   */
  grenadeRestNote: string
  /** Kill feed synchronisé sur l'horloge du rejeu. */
  killFeedTitle: string
  killFeedEmpty: string
  killFeedUnknownWeapon: string
  killFeedNoAssistHint: string
  killFeedAssistHint: string
  killFeedKillerShare: (pct: number) => string
  /** Ligne de mort NEUTRE (suicide, chute, sortie) : le mot affiché et son infobulle. */
  killFeedDeathLabel: string
  killFeedDeathHint: string
  /**
   * Le TYPE de la mort neutre, quand le film l'établit. Clés = les identifiants stables
   * publiés par le document (`kind`) — un type inconnu n'a pas de libellé et n'affiche
   * aucune icône, la ligne garde son repère neutre.
   */
  killFeedDeathKind: Record<'environment' | 'suicide', string>
  /** Sons du rejeu (lot 5 parité) : COUPÉ PAR DÉFAUT, l'utilisateur l'active. */
  sound: string
  soundHint: string
  soundVolume: string
  /** Le son est activé mais tu par la vitesse de lecture — le dire, pas le cacher. */
  soundFastHint: string
  /** Calques que le lecteur peut éteindre. */
  layers: string
  layerAim: string
  layerAimHint: string
  /** Zones nommées (callouts officiels) : calque + libellé de la fiche. */
  layerZones: string
  layerZonesHint: string
  zoneLabel: string
  /** Fiches joueur : ce qui est lu, et ce qui ne l'est pas. */
  rosterEmpty: string
  teamUnknown: string
  unknownPlayer: string
  healthLabel: string
  shieldLabel: string
  abilityLabel: string
  loadoutUnread: string
  loadoutAge: string
  loadoutAhead: string
  weaponSecondaryHint: string
  /** Pictogramme « armes rangées » (sélecteur D=2) : tooltip simple, décision produit 4. */
  holsteredLabel: string
  /** Badge de lancer sur la fiche (le `.gic` du POC) : l'auteur est écrit dans le film. */
  grenadeThrown: string
  weaponSwap: string
  respawnIn: string
  respawnUnknown: string
  /** Ligne d'inventaire : grenades, capacité, munitions. */
  inventoryAge: string
  inventoryAhead: string
  grenadeSelected: string
  grenadeSelectedRead: string
  grenadeSelUnknown: string
  abilityUnknown: string
  abilityAge: string
  abilityAhead: string
  /** Pictogramme « munitions pleines » (emplacement jamais écrit) : décision produit 4. */
  ammoFullLabel: string
  ammoDrawnHint: string
  drawnUnknown: string
  gaugeLabel: string
  respawnBarLabel: string
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
    propsSuffix: 'objets de décor',
    mapBackgroundNote:
      'Fond : la carte du jeu, reconstruite hors ligne et calée au mètre sur le repère du rejeu.',
    mapBackgroundFallback:
      "Fond : sol reconstruit — cette carte n'a pas encore d'image figée.",
    grenadeRestNote:
      "Grenades : l'effet en fin de vol marque la dernière position connue du projectile — le film n'enregistre aucune détonation.",
    killFeedTitle: 'Éliminations',
    killFeedEmpty: 'Rien à cet instant du match.',
    killFeedUnknownWeapon: 'Arme non identifiée',
    killFeedNoAssistHint:
      'Mort sans assistant — MESURÉ : la mort porte son événement dans le film, et il ne déclare personne.',
    killFeedAssistHint:
      'Assistant lu dans le film, avec sa part de dégâts quand elle est mesurée. Les parts ne sont pas bornées à 100 %.',
    killFeedKillerShare: (pct) => `tueur ${pct} %`,
    killFeedDeathLabel: 'mort',
    killFeedDeathHint:
      'Mort sans tueur crédité (suicide, chute ou sortie), lue dans les trajectoires du film.',
    killFeedDeathKind: {
      environment: 'Chute ou sortie de zone',
      suicide: 'Tué par sa propre arme',
    },
    sound: 'Son',
    soundHint:
      "Sons d'armes sur les éliminations et les lancers de grenade, coupés à la seconde. Une arme sans son enregistré reste muette. Coupé par défaut.",
    soundVolume: 'Volume des sons',
    soundFastHint:
      'À cette vitesse de lecture les sons se chevaucheraient : ils reviennent à 2× ou moins.',
    layers: 'Calques',
    layerAim: 'Visée',
    layerAimHint:
      "Direction du regard, décodée du même enregistrement que la position. Le jeu ne la retransmet que lorsqu'elle change : une mesure ancienne pâlit au lieu de disparaître, et rien n'est dessiné au-delà de cinq secondes.",
    layerZones: 'Zones',
    layerZonesHint:
      'Zones nommées officielles de la carte, extraites du jeu. Les grandes zones pavent le terrain ; les contours pointillés sont des étages imbriqués.',
    zoneLabel: 'Zone de la carte',
    rosterEmpty:
      "Aucune vie du film n'a pu être rattachée à un joueur : le rejeu reste anonyme.",
    teamUnknown: 'Sans équipe',
    unknownPlayer: 'Joueur inconnu',
    healthLabel: 'Santé',
    shieldLabel: 'Bouclier',
    abilityLabel: "Capacité d'armure équipée",
    loadoutUnread: 'armes non lues sur cette vie',
    loadoutAge: 'Armes lues il y a',
    loadoutAhead:
      'Armes de la première image-clé de cette vie, lue dans',
    weaponSecondaryHint: 'secondaire (arme rangée à la dernière lecture)',
    holsteredLabel: 'Armes rangées',
    grenadeThrown: 'Grenade lancée',
    weaponSwap: 'échange',
    respawnIn: 'retour dans',
    respawnUnknown: 'retour ?',
    inventoryAge: 'Inventaire lu il y a',
    inventoryAhead: 'Inventaire de la première image-clé de cette vie, lue dans',
    grenadeSelected: 'Type équipé : le seul porté, donc celui qui partira au prochain lancer.',
    grenadeSelectedRead:
      'Type équipé, LU dans le film (sélecteur de grenade de l’image-clé) : celui qui partira au prochain lancer.',
    grenadeSelUnknown: 'sél. ?',
    abilityUnknown: 'capacité inconnue',
    abilityAge: 'Capacité lue il y a',
    abilityAhead: 'Capacité lue dans',
    ammoFullLabel: 'Munitions pleines',
    gaugeLabel: 'charge restante',
    respawnBarLabel: 'avancement depuis la mort',
    ammoDrawnHint:
      'Emplacement DÉGAINÉ selon le sélecteur du record : la même lecture qui place cette arme en tête de rangée.',
    drawnUnknown: 'dégainée ?',
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
    propsSuffix: 'map props',
    mapBackgroundNote: 'Background: the in-game map, rebuilt offline and aligned to the metre with the replay frame of reference.',
    mapBackgroundFallback: 'Background: reconstructed floor — this map has no baked image yet.',
    grenadeRestNote:
      "Grenades: the end-of-flight effect marks the projectile's last known position — the film records no detonation.",
    killFeedTitle: 'Kills',
    killFeedEmpty: 'Nothing at this point of the match.',
    killFeedUnknownWeapon: 'Unidentified weapon',
    killFeedNoAssistHint:
      'Death without an assist — MEASURED: the death carries its event in the film, and it names no one.',
    killFeedAssistHint:
      'Assist read from the film, with its damage share when measured. Shares are not capped at 100%.',
    killFeedKillerShare: (pct) => `killer ${pct}%`,
    killFeedDeathLabel: 'died',
    killFeedDeathHint:
      'Death with no credited killer (suicide, fall or leaving), read from the film trails.',
    killFeedDeathKind: {
      environment: 'Fall or out of bounds',
      suicide: 'Killed by their own weapon',
    },
    sound: 'Sound',
    soundHint:
      'Weapon sounds on kills and grenade throws, cut at one second. A weapon with no recorded sound stays silent. Off by default.',
    soundVolume: 'Sound volume',
    soundFastHint:
      'At this playback speed the sounds would overlap: they come back at 2× or below.',
    layers: 'Layers',
    layerAim: 'Aim',
    layerAimHint:
      'Look direction, decoded from the same record as the position. The game only retransmits it when it changes: an older reading fades instead of vanishing, and nothing is drawn beyond five seconds.',
    layerZones: 'Zones',
    layerZonesHint:
      'Official named map zones, extracted from the game. Large zones tile the terrain; dashed outlines are nested floors.',
    zoneLabel: 'Map zone',
    rosterEmpty: 'No life from the film could be attached to a player: the replay stays anonymous.',
    teamUnknown: 'No team',
    unknownPlayer: 'Unknown player',
    healthLabel: 'Health',
    shieldLabel: 'Shield',
    abilityLabel: 'Equipped armor ability',
    loadoutUnread: 'weapons not read on this life',
    loadoutAge: 'Weapons read',
    loadoutAhead: 'Weapons from the first keyframe of this life, read in',
    weaponSecondaryHint: 'secondary (weapon holstered at the last reading)',
    holsteredLabel: 'Weapons holstered',
    grenadeThrown: 'Grenade thrown',
    weaponSwap: 'swap',
    respawnIn: 'back in',
    respawnUnknown: 'back ?',
    inventoryAge: 'Inventory read',
    inventoryAhead: 'Inventory from the first keyframe of this life, read in',
    grenadeSelected: 'Equipped type: the only one carried, so the one the next throw will use.',
    grenadeSelectedRead:
      'Equipped type, READ from the film (keyframe grenade selector): the one the next throw will use.',
    grenadeSelUnknown: 'sel. ?',
    abilityUnknown: 'unknown ability',
    abilityAge: 'Ability read',
    abilityAhead: 'Ability read in',
    ammoFullLabel: 'Ammo full',
    gaugeLabel: 'charge left',
    respawnBarLabel: 'progress since death',
    ammoDrawnHint:
      'Slot DRAWN according to the record selector: the same reading that puts this weapon first in the row.',
    drawnUnknown: 'drawn ?',
  },
}
