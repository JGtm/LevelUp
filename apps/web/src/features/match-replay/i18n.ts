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
  /**
   * L'ASSISTANCE EST UN PICTOGRAMME, PAS UN MOT (décision utilisateur du 16/08) : le fil
   * n'écrit plus « assisté par », il pose une marque puis le nom. Ce libellé est ce que la
   * marque DIT (infobulle + lecteur d'écran) — elle ne se lit pas toute seule.
   */
  killFeedAssistMark: string
  /** Part de participation de l'assistant, telle que la ligne l'écrit : « - 37 % ». */
  killFeedAssistShare: (pct: number) => string
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
  /** Filtre des sons par catégorie (tiroir de réglages, phase 2, décision du 16/08). */
  soundCategoriesTitle: string
  soundCategory: Record<'weapon' | 'grenade' | 'melee' | 'equipment', string>
  /** Le tiroir de réglages (décision utilisateur du 16/08) : bouton et panneau partagent
   *  le même intitulé — ouvrir dit ce qu'on va trouver derrière. */
  settingsButton: string
  settingsClose: string
  /** Calques que le lecteur peut éteindre. */
  layers: string
  layerAim: string
  layerAimHint: string
  /** Zones nommées (callouts officiels) : calque + libellé de la fiche. */
  layerZones: string
  layerZonesHint: string
  /** Noms des joueurs sous leur marqueur (calque éteignable — un BTB à 24 joueurs). */
  layerNames: string
  layerNamesHint: string
  zoneLabel: string
  /**
   * EFFETS D'ÉVÉNEMENT, réglables séparément (décision utilisateur du 16/08) : les éclairs
   * de bouche de TOUS les tirs, et le trait tueur -> victime des éliminations. Le premier
   * porte une RÉSERVE DE MESURE affichée en clair (le film n'enregistre un tir que
   * lorsqu'un dégât est appliqué) : elle ne vit pas dans un commentaire, elle est à l'écran.
   */
  effects: string
  layerShotFx: string
  layerShotFxHint: string
  layerShotFxCoverage: string
  layerKillFx: string
  layerKillFxHint: string
  /**
   * Carte de chaleur : le calque, ce qu'il mesure, et sa légende. JAMAIS « heatmap » à
   * l'écran (règle FR sans anglicismes) — « carte de chaleur » partout.
   */
  layerHeatmap: string
  layerHeatmapHint: string
  heatmapReading: string
  heatmapMode: Record<'presence' | 'kills', string>
  heatmapModeHint: Record<'presence' | 'kills', string>
  /** Extrémités de la légende : elles nomment la QUANTITÉ, le titre dit de quoi il s'agit. */
  heatLegendLow: string
  heatLegendHigh: string
  heatLegendHint: string
  /** Fiches joueur : ce qui est lu, et ce qui ne l'est pas. */
  rosterEmpty: string
  teamUnknown: string
  /** Libellé d'équipe (cascade `lib/halo/teamLabel.ts`, mêmes textes que la Match View). */
  teamLabelFmt: (name: string) => string
  teamNumberedFmt: (n: number) => string
  unknownPlayer: string
  /** Marques d'identité devant un nom (fiches, fil) : le joueur de la page, un ami. */
  markMe: string
  markFriend: string
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
  /**
   * Capacité absente de la table du titre : la fiche pose un GLYPHE NEUTRE (pas un
   * caractère, décision utilisateur du 16/08) et dit en infobulle ce qu'on ne sait pas —
   * le RANG lu reste écrit, parce qu'il est la seule chose de vraie à cet endroit.
   */
  abilityUnidentified: (rank: number) => string
  abilityAge: string
  abilityAhead: string
  /**
   * État ACTIF d'un équipement, par FAMILLE mesurée (jamais un libellé libre) : le
   * camouflage rend la fiche vitreuse, le surbouclier l'encadre d'or (cahier des
   * charges Notion 21.1, rendu par `ReplayTeams.tsx`). La clé est l'identifiant
   * stable publié par le document (`fam`) — une famille inconnue n'a pas de libellé et
   * ne reçoit aucun effet.
   */
  equipmentActive: Record<'camo' | 'overshield', string>
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
      "Grenades : l'explosion se joue à la dernière position connue du projectile — le film n'enregistre aucune détonation, la mise en scène est assumée à cet endroit.",
    killFeedTitle: 'Éliminations',
    killFeedEmpty: 'Rien à cet instant du match.',
    killFeedUnknownWeapon: 'Arme non identifiée',
    killFeedNoAssistHint:
      'Mort sans assistant — MESURÉ : la mort porte son événement dans le film, et il ne déclare personne.',
    killFeedAssistHint:
      'Assistant lu dans le film, avec sa part de dégâts quand elle est mesurée. Les parts ne sont pas bornées à 100 %.',
    killFeedKillerShare: (pct) => `tueur ${pct} %`,
    killFeedAssistMark: 'Assistance',
    killFeedAssistShare: (pct) => `- ${pct} %`,
    killFeedDeathLabel: 'mort',
    killFeedDeathHint:
      'Mort sans tueur crédité (suicide, chute ou sortie), lue dans les trajectoires du film.',
    killFeedDeathKind: {
      environment: 'Chute ou sortie de zone',
      suicide: 'Tué par sa propre arme',
    },
    sound: 'Son',
    soundHint:
      "Sons d'armes sur les éliminations, les lancers de grenade et les activations d'équipement, coupés à la seconde. Une arme sans son enregistré reste muette. Coupé par défaut.",
    soundVolume: 'Volume des sons',
    soundFastHint:
      'À cette vitesse de lecture les sons se chevaucheraient : ils reviennent à 2× ou moins.',
    soundCategoriesTitle: 'Sons par catégorie',
    soundCategory: {
      weapon: 'Armes',
      grenade: 'Grenades',
      melee: 'Mêlée',
      equipment: 'Équipements',
    },
    settingsButton: 'Réglages',
    settingsClose: 'Fermer les réglages',
    layers: 'Calques',
    layerAim: 'Visée',
    layerAimHint:
      "Cône de regard : la direction où le joueur regarde, décodée du même enregistrement que la position. Le jeu ne la retransmet que lorsqu'elle change : une mesure ancienne pâlit au lieu de disparaître, et rien n'est dessiné au-delà de cinq secondes.",
    layerZones: 'Zones',
    layerZonesHint:
      'Zones nommées officielles de la carte, extraites du jeu. Les grandes zones pavent le terrain ; les contours pointillés sont des étages imbriqués.',
    layerNames: 'Noms',
    layerNamesHint: 'Le nom de chaque joueur sous son marqueur.',
    zoneLabel: 'Zone de la carte',
    effects: 'Effets',
    layerShotFx: 'Effets de tirs',
    layerShotFxHint:
      'Éclair de bouche sur chaque tir décodé, dans la teinte de la décharge (cinétique, plasma, énergie).',
    layerShotFxCoverage:
      "La couverture des tirs peut ne pas être totale : le film n'enregistre un tir que lorsqu'un dégât est appliqué.",
    layerKillFx: 'Effets de mort',
    layerKillFxHint:
      "Trait orienté du tueur vers la victime, à l'instant de l'élimination. Éteint par défaut.",
    layerHeatmap: 'Carte de chaleur',
    layerHeatmapHint:
      "Où le match s'est joué, sur tout le match. Une cellule jamais atteinte reste vide : « froid » veut dire peu fréquenté, l'absence de couleur veut dire jamais vu.",
    heatmapReading: 'Ce que la chaleur mesure',
    heatmapMode: {
      presence: 'Présence',
      kills: 'Éliminations',
    },
    heatmapModeHint: {
      presence: 'Temps passé par les joueurs, lu dans les trajectoires du film.',
      kills: "Morts comptées à l'endroit où la victime est tombée, pas d'où le tir partait.",
    },
    heatLegendLow: 'rare',
    heatLegendHigh: 'fréquent',
    heatLegendHint:
      "Échelle étalonnée sur les lieux fréquentés (médiane au bas, 95e centile en haut) : au-delà, la couleur sature. Un seul point extrême ne peut donc pas écraser le reste de la carte.",
    rosterEmpty:
      "Aucune vie du film n'a pu être rattachée à un joueur : le rejeu reste anonyme.",
    teamUnknown: 'Sans équipe',
    teamLabelFmt: (name) => `Équipe ${name}`,
    teamNumberedFmt: (n) => `Équipe ${n}`,
    unknownPlayer: 'Joueur inconnu',
    markMe: 'Moi',
    markFriend: 'Ami',
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
    respawnIn: 'Réapparition dans',
    respawnUnknown: 'Réapparition ?',
    inventoryAge: 'Inventaire lu il y a',
    inventoryAhead: 'Inventaire de la première image-clé de cette vie, lue dans',
    grenadeSelected: 'Type équipé : le seul porté, donc celui qui partira au prochain lancer.',
    grenadeSelectedRead:
      'Type équipé, LU dans le film (sélecteur de grenade de l’image-clé) : celui qui partira au prochain lancer.',
    grenadeSelUnknown: 'sél. ?',
    abilityUnidentified: (rank) => `capacité non identifiée (rang ${rank})`,
    abilityAge: 'Capacité lue il y a',
    abilityAhead: 'Capacité lue dans',
    equipmentActive: {
      camo: 'Camouflage actif — le joueur est invisible à l’écran de jeu',
      overshield: 'Surbouclier actif',
    },
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
      "Grenades: the explosion plays at the projectile's last known position — the film records no detonation, the staging is deliberate.",
    killFeedTitle: 'Kills',
    killFeedEmpty: 'Nothing at this point of the match.',
    killFeedUnknownWeapon: 'Unidentified weapon',
    killFeedNoAssistHint:
      'Death without an assist — MEASURED: the death carries its event in the film, and it names no one.',
    killFeedAssistHint:
      'Assist read from the film, with its damage share when measured. Shares are not capped at 100%.',
    killFeedKillerShare: (pct) => `killer ${pct}%`,
    killFeedAssistMark: 'Assist',
    killFeedAssistShare: (pct) => `- ${pct}%`,
    killFeedDeathLabel: 'died',
    killFeedDeathHint:
      'Death with no credited killer (suicide, fall or leaving), read from the film trails.',
    killFeedDeathKind: {
      environment: 'Fall or out of bounds',
      suicide: 'Killed by their own weapon',
    },
    sound: 'Sound',
    soundHint:
      'Weapon sounds on kills, grenade throws and equipment activations, cut at one second. A weapon with no recorded sound stays silent. Off by default.',
    soundVolume: 'Sound volume',
    soundFastHint:
      'At this playback speed the sounds would overlap: they come back at 2× or below.',
    soundCategoriesTitle: 'Sounds by category',
    soundCategory: {
      weapon: 'Weapons',
      grenade: 'Grenades',
      melee: 'Melee',
      equipment: 'Equipment',
    },
    settingsButton: 'Settings',
    settingsClose: 'Close settings',
    layers: 'Layers',
    layerAim: 'Aim',
    layerAimHint:
      'Look cone: the direction the player is looking at, decoded from the same record as the position. The game only retransmits it when it changes: an older reading fades instead of vanishing, and nothing is drawn beyond five seconds.',
    layerZones: 'Zones',
    layerZonesHint:
      'Official named map zones, extracted from the game. Large zones tile the terrain; dashed outlines are nested floors.',
    layerNames: 'Names',
    layerNamesHint: "Each player's name under their marker.",
    zoneLabel: 'Map zone',
    effects: 'Effects',
    layerShotFx: 'Shot effects',
    layerShotFxHint:
      'Muzzle flash on every decoded shot, in the tint of the discharge (kinetic, plasma, energy).',
    layerShotFxCoverage:
      'Shot coverage may not be complete: the film only records a shot when damage is applied.',
    layerKillFx: 'Kill effects',
    layerKillFxHint:
      'Line drawn from killer to victim at the moment of the kill. Off by default.',
    layerHeatmap: 'Heat map',
    layerHeatmapHint:
      'Where the match was played, over the whole match. A cell never reached stays empty: "cold" means seldom visited, no colour at all means never seen.',
    heatmapReading: 'What the heat measures',
    heatmapMode: {
      presence: 'Presence',
      kills: 'Kills',
    },
    heatmapModeHint: {
      presence: 'Time spent by players, read from the film trails.',
      kills: 'Deaths counted where the victim fell, not where the shot came from.',
    },
    heatLegendLow: 'rare',
    heatLegendHigh: 'frequent',
    heatLegendHint:
      'Scale calibrated on the visited places (median at the bottom, 95th percentile at the top): beyond that, the colour saturates. A single extreme spot therefore cannot flatten the rest of the map.',
    rosterEmpty: 'No life from the film could be attached to a player: the replay stays anonymous.',
    teamUnknown: 'No team',
    teamLabelFmt: (name) => `Team ${name}`,
    teamNumberedFmt: (n) => `Team ${n}`,
    unknownPlayer: 'Unknown player',
    markMe: 'Me',
    markFriend: 'Friend',
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
    respawnIn: 'Respawn in',
    respawnUnknown: 'Respawn ?',
    inventoryAge: 'Inventory read',
    inventoryAhead: 'Inventory from the first keyframe of this life, read in',
    grenadeSelected: 'Equipped type: the only one carried, so the one the next throw will use.',
    grenadeSelectedRead:
      'Equipped type, READ from the film (keyframe grenade selector): the one the next throw will use.',
    grenadeSelUnknown: 'sel. ?',
    abilityUnidentified: (rank) => `unidentified ability (rank ${rank})`,
    abilityAge: 'Ability read',
    abilityAhead: 'Ability read in',
    equipmentActive: {
      camo: 'Active camo — the player is invisible on the game screen',
      overshield: 'Overshield active',
    },
    ammoFullLabel: 'Ammo full',
    gaugeLabel: 'charge left',
    respawnBarLabel: 'progress since death',
    ammoDrawnHint:
      'Slot DRAWN according to the record selector: the same reading that puts this weapon first in the row.',
    drawnUnknown: 'drawn ?',
  },
}
