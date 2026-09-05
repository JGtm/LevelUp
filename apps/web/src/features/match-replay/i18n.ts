/**
 * i18n de la feature match-replay (rejeu 2D vue du dessus). Strings UI FR + EN,
 * parité par typage Record<ReplayLocale, ReplayText>.
 *
 * LE CONTRAT (un champ par string, et sa justification) VIT DANS `i18nContract.ts` : ce
 * fichier-ci ne porte plus que les deux TABLES. La découpe date du 2026-08-18 (lot R2-V),
 * quand le fichier réuni a franchi le seuil de taille du dépôt.
 */
import type { Locale } from '@/lib/i18n/locale'

import type { ReplayText } from './i18nContract'

/** Alias local du type central : la feature ne redéclare pas l'union des langues. */
export type ReplayLocale = Locale

export const REPLAY_TEXT: Record<ReplayLocale, ReplayText> = {
  fr: {
    title: 'Rejeu 2D',
    back: 'Fiche du match',
    play: 'Lecture',
    pause: 'Pause',
    restart: 'Recommencer',
    loading: 'Chargement du rejeu…',
    empty: 'Aucun rejeu 2D disponible pour ce match.',
    speed: 'Vitesse',
    time: 'Temps de match',
    skipBackFmt: (seconds) => `Reculer de ${seconds} s`,
    skipForwardFmt: (seconds) => `Avancer de ${seconds} s`,
    speedNormal: 'normal',
    speedMuted: 'son coupé',
    keySpace: 'Espace',
    killFeedTitle: 'Éliminations',
    killFeedEmpty: 'Rien à cet instant du match.',
    killFeedUnknownWeapon: 'Arme non identifiée',
    killFeedNoAssistHint:
      'Mort sans assistant — MESURÉ : la mort porte son événement dans le film, et il ne déclare personne.',
    killFeedAssistHint:
      'Assistant lu dans le film, avec sa part de dégâts quand elle est mesurée. Les parts ne sont pas bornées à 100 %.',
    killFeedKillerShare: (pct) => `tueur ${pct} %`,
    killFeedAssistMark: 'Assistance',
    killFeedAssistShare: (pct) => `${pct} %`,
    killFeedDeathLabel: 'mort',
    killFeedDeathHint:
      'Mort sans tueur crédité (suicide, chute ou sortie), lue dans les trajectoires du film.',
    killFeedDeathKind: {
      environment: 'Chute ou sortie de zone',
      suicide: 'Tué par sa propre arme',
    },
    presenceJoined: 'a rejoint la partie',
    presenceLeft: 'a quitté la partie',
    presenceJoinedHint:
      "Horodatage de participation de l'API du match : le joueur a rejoint en cours de partie.",
    presenceLeftHint:
      "Horodatage de participation de l'API du match : le joueur a quitté avant la fin.",
    presenceJoinedDerived: 'entre en partie',
    presenceLeftDerived: 'ne reviendra plus',
    presenceJoinedDerivedHint:
      "Première apparition bien après le coup d'envoi, dérivée des vies du film (participation API absente sur ce match).",
    presenceLeftDerivedHint:
      "Dernière vie du film bien avant la fin : un départ — ou une élimination définitive sur un mode à manches, le film ne les distingue pas (participation API absente sur ce match).",
    sound: 'Son',
    soundHint:
      "Sons d'armes sur les éliminations, les lancers de grenade et les activations d'équipement, coupés à la seconde. Une arme sans son enregistré reste muette. Coupé par défaut.",
    soundVolume: 'Volume des sons',
    soundVolumeMutedHint:
      'Son coupé : le volume est à zéro. Rallumer le son rend le niveau réglé précédemment.',
    soundFastHint:
      'À cette vitesse de lecture les sons se chevaucheraient : ils reviennent à 2× ou moins.',
    soundCategoriesTitle: 'Sons par catégorie',
    soundCategory: {
      weapon: 'Armes',
      grenade: 'Grenades',
      melee: 'Mêlée',
      equipment: 'Équipements',
    },
    captureImage: "Capturer l'image",
    recordVideo: 'Enregistrer la vidéo',
    stopRecording: "Arrêter l'enregistrement",
    recordHint:
      "L'enregistrement filme le rejeu tel qu'il défile : changer de vitesse ou déplacer le curseur se voit dans le fichier. Démarrer relance la lecture si elle est en pause ; mettre en pause ou laisser le film finir arrête l'enregistrement et télécharge le clip.",
    // « REC » et « Arrêter » : la convention des lecteurs, identique dans les deux langues
    // pour la première — c'est le mot qu'on cherche du regard sur un bouton d'enregistrement.
    settingsButton: 'Réglages',
    settingsClose: 'Fermer les réglages',
    autoPlay: 'Lecture automatique',
    autoPlayHint:
      "Allumé, le rejeu démarre tout seul à l'ouverture de la page. Éteint — le réglage par défaut — il s'ouvre en pause au coup d'envoi et attend le bouton Lecture. Le choix est retenu d'un match à l'autre ; il ne met ni en lecture ni en pause le rejeu déjà ouvert.",
    layers: 'Calques',
    layerAim: 'Visée',
    layerAimHint:
      "Cône de regard : la direction où le joueur regarde, décodée du même enregistrement que la position. Le jeu ne la retransmet que lorsqu'elle change : une mesure ancienne pâlit au lieu de disparaître, et rien n'est dessiné au-delà de cinq secondes. Un trait court se pose à la POINTE du cône quand la visée n'est pas à plat : vers l'extérieur si le joueur lève la tête, vers l'intérieur s'il pique. Le cône raccourcit dans les deux cas — sa longueur seule ne les distinguerait pas.",
    zoomGroup: 'Cadrage de la carte',
    zoomIn: 'Grossir (+ ou molette)',
    zoomOut: 'Réduire (− ou molette)',
    zoomReset: 'Revoir toute la carte',
    zoomLevelFmt: (z: number) => `${String(z).replace('.', ',')}x`,
    panUp: 'Déplacer vers le haut (Maj + flèche haut)',
    panDown: 'Déplacer vers le bas (Maj + flèche bas)',
    panLeft: 'Déplacer vers la gauche (Maj + flèche gauche)',
    panRight: 'Déplacer vers la droite (Maj + flèche droite)',
    layerGroupPlayers: 'Joueurs',
    layerGroupTerrain: 'Terrain',
    layerGroupObjectives: 'Objectifs',
    layerZones: 'Zones',
    layerZonesHint:
      'Zones nommées officielles de la carte, extraites du jeu. Les grandes zones pavent le terrain ; les contours pointillés sont des étages imbriqués.',
    layerTrail: 'Traînée',
    layerTrailHint:
      "Les sept dernières secondes parcourues, derrière chaque marqueur. L'opacité monte vers la tête : la trace la plus visible est celle de l'instant, et c'est ce qui donne le sens du déplacement.",
    effects: 'Effets',
    layerShotFx: 'Effets de tirs',
    layerShotFxHint:
      'Éclair de bouche sur chaque tir décodé, dans la teinte de la décharge (cinétique, plasma, énergie).',
    layerShotFxCoverage:
      "La couverture des tirs peut ne pas être totale : le film n'enregistre un tir que lorsqu'un dégât est appliqué.",
    layerKillFx: 'Effets de mort',
    layerKillFxHint:
      "Trait orienté du tueur vers la victime, à l'instant de l'élimination. Allumé par défaut.",
    layerPlacements: 'Équipements posés',
    layerPlacementsHint:
      "Les objets qu'un joueur a DÉPLOYÉS en cours de vie : mur de protection, capteur de menaces, traqueur de menaces, balise du translocateur quantique, champ de réparation. La BALISE est le point de retour que pose le translocateur quantique — ce n'est pas le marquage d'un ennemi, que le jeu appelle « ping ». Le champ de réparation porte une croix qui respire : elle dit que l'objet soigne, elle ne compte rien — le film ne publie aucune cadence de soin, et son cercle pointillé garde la réserve sur sa portée. Ce que la mesure classe autrement ne se dessine pas — près de neuf poses sur dix sont en réalité l'équipement et les grenades qu'un joueur lâche en mourant, et ce n'est pas un geste. Le film ne dit pas quand un équipement disparaît : chaque famille se tient donc à sa durée officielle quand le jeu en publie une — 15 s pour le capteur, une dizaine de secondes pour le mur — et les autres poses restent affichées jusqu'à la fin du rejeu. L'arc du mur est orienté par le regard du poseur ; quand la pose ne porte pas ce cap (un peu plus d'une fois sur huit), il suit la dernière direction de déplacement du poseur, et à défaut sa dernière visée connue — un arc déduit se trace alors en pointillé. Le capteur balaie sa portée toutes les 1,8 s — chiffres officiels du jeu, le film n'en porte aucun — et marque brièvement les adversaires du poseur qui s'y trouvent au passage de l'onde ; le traqueur, lui, n'émet qu'une seule impulsion.",
    layerPlacementsDropped: 'Objets lâchés au sol',
    layerPlacementsDroppedHint:
      "Les objets de PUISSANCE qu'un joueur laisse au sol en mourant : power-ups (surbouclier, camouflage) et équipements déployables (mur de protection, capteur, traqueur, balise, champ de réparation). Ils sont RAMASSABLES — savoir qu'ils traînent change la lecture de l'échange suivant. Un anneau pointillé et atténué, jamais la forme de l'objet actif : au sol, l'objet n'exerce ni portée ni effet. Les grenades et les capacités lâchées restent hors carte : elles représentent près de neuf poses sur dix et ne diraient rien du terrain. Allumé par défaut.",
    layerPlacementsUnnamed: 'Objets non identifiés',
    layerPlacementsUnnamedHint:
      "Les objets d'équipement que la mesure situe sans pouvoir les nommer : un point neutre, sans forme empruntée aux familles nommées. Ceux-là s'affichent quelle que soit leur origine — on cherche justement à voir ce qu'on ne sait pas nommer. Éteint par défaut.",
    placementFamily: {
      wall: 'Mur de protection',
      sensor: 'Capteur de menaces',
      rift: 'Faille du translocateur quantique',
      shroud: 'Écran occultant',
      seeker: 'Traqueur de menaces',
      field: 'Champ de réparation',
    },
    placementUnnamedLabel: "Objet d'équipement non identifié",
    placementOwnerFmt: (name) => `Posé par ${name}`,
    placementOwnerUnknown: 'Poseur non mesuré',
    placementDroppedLabel: 'Objet lâché au sol',
    placementDroppedOwnerFmt: (name) => `Lâché par ${name}`,
    placementDroppedAtFmt: (clock) => `Au sol depuis ${clock}`,
    layerWeaponPads: "Emplacements d'arme",
    layerWeaponPadsHint:
      "Les endroits où une arme réapparaît au fil du match, mesurés sur ce match : l'arme y est dessinée en grand quand elle change une partie (fusil de précision, épée, marteau, roquettes, empaleur, crémateur, surbouclier, camouflage), en petit sinon. Socle au sol ou râtelier mural : la mesure ne porte qu'une position, elle ne les distingue pas. Le film ne date pas l'instant du ramassage — l'emplacement reste donc INCERTAIN pendant l'intervalle des relevés, environ vingt secondes, plutôt que de s'éteindre à un instant inventé. Un compte à rebours n'apparaît que là où le délai de réapparition a pu être établi ; ailleurs, aucun chiffre. Qui a pris l'arme n'est jamais affiché : la mesure n'atteint pas le niveau de certitude exigé.",
    padEquipmentFamily: {
      powerup_overshield: 'Surbouclier',
      powerup_camo: 'Camouflage actif',
    },
    layerFlagCarries: 'Drapeaux',
    layerFlagCarriesHint:
      "La vie des drapeaux de capture, lue dans le film : porté (le drapeau suit son porteur image par image), au sol à la dernière position mesurée, ou à sa base. La base garde un drapeau atténué tant que le sien est ailleurs. Un portage dont RIEN ne date la fin s'affiche atténué lui aussi : son intervalle court jusqu'à la fin du film, c'est une borne haute et non une mesure.",
    flagSide: {
      ally: 'Drapeau allié',
      enemy: 'Drapeau adverse',
      unknown: 'Drapeau',
    },
    flagState: {
      carried: 'Porté',
      carried_open: 'Porté, fin non datée',
      dropped: 'Au sol',
      home: 'À la base',
    },
    flagCarrierUnknown: 'Porteur non nommé',
    flagSinceFmt: (seconds) => `Depuis ${Math.round(seconds)} s`,
    flagOpenNote:
      "Rien ne date la fin de ce portage : l'intervalle court jusqu'à la fin du film — c'est une borne haute, pas une durée mesurée.",
    layerVipCrown: 'VIP',
    layerVipCrownHint:
      "La couronne du VIP courant, lue dans le film : chaque désignation ouvre une période de port, fermée par la mort du VIP ou la désignation suivante. La couronne suit son porteur image par image. Un port dont RIEN ne date la fin s'affiche atténué : son intervalle court jusqu'à la fin du film — c'est une borne haute, pas une mesure.",
    layerSkullCarrier: 'Porteur du crâne',
    layerSkullCarrierHint:
      "Qui porte le crâne d'Oddball, lu dans le film : le porteur est le joueur dont les tics de score de mode montent, un train de tics étant une période de portage. Le crâne suit son porteur image par image. Un portage dont RIEN ne date la fin s'affiche atténué : son intervalle court jusqu'à la fin du film — c'est une borne haute, pas une mesure. Le crâne posé au sol, lui, reste dessiné par le calque des objets d'objectif.",
    layerBombCarrier: 'Bombe',
    layerBombCarrierHint:
      "La bombe d'Assaut, lue dans le film : une prise ouvre une période de portage, fermée par le lâcher — souvent la pose elle-même — ou par la mort du porteur. Portée, la bombe suit son porteur image par image ; lâchée, elle reste dessinée au dernier point de son lâcheur jusqu'à la reprise ou l'explosion. Un portage dont RIEN ne date la fin s'affiche atténué : son intervalle court jusqu'à la fin du film — c'est une borne haute, pas une mesure.",
    layerGroundWeapons: 'Armes au sol',
    layerGroundWeaponsHint:
      "Les armes abandonnées au sol, lues dans le film : l'arme d'un mort, celle qu'on laisse en ramassant autre chose, ou celle qu'un râtelier a éjectée. Chacune est dessinée là où elle s'est arrêtée, dès qu'elle y tombe. Les armes SUR LEUR EMPLACEMENT ne sont pas ici : elles appartiennent au calque des emplacements d'arme. LA FIN est celle que le film montre, jamais une durée de table : un ramassage daté l'arrête exactement ; sinon l'arme est dessinée pleine tant qu'un relevé la voit encore, puis S'ESTOMPE jusqu'au premier relevé qui ne la voit plus — la disparition a eu lieu quelque part entre les deux, et rien n'est dessiné au-delà de cette borne.",
    padCountdownFmt: (seconds) => `${Math.ceil(seconds)} s`,
    padRespawnMeasuredFmt: (seconds) => `Réapparition dans ${Math.ceil(seconds)} s`,
    padRespawnExpectedFmt: (seconds) => `Réapparition dans ≈ ${Math.ceil(seconds)} s`,
    layerHeatmap: 'Carte de chaleur',
    layerHeatmapHint:
      "Où le match s'est joué, sur tout le match. Une cellule jamais atteinte reste vide : « froid » veut dire peu fréquenté, l'absence de couleur veut dire jamais vu.",
    heatmapReading: 'Mesure',
    heatmapMode: {
      presence: 'Présence',
      kills: 'Éliminations',
    },
    heatmapModeHint: {
      presence: 'Temps passé par les joueurs, lu dans les trajectoires du film.',
      kills: "Morts comptées à l'endroit où la victime est tombée, pas d'où le tir partait.",
    },
    heatmapSpanTitle: 'Durée',
    heatmapSpan: {
      match: 'Partie',
      live: 'Progressif',
    },
    heatmapSpanHint: {
      match:
        "Le match entier, d'un bout à l'autre, quelle que soit l'image affichée : les zones chaudes se lisent d'un coup d'œil, comme une analyse d'après-match.",
      live:
        "Seulement ce qui a été joué jusqu'ici : la carte se remplit au fil de la lecture, et revenir en arrière la ramène à ce qu'elle était. Recalculée toutes les deux secondes de match.",
    },
    markerColorsTitle: 'Couleur des points',
    markerColorsMode: {
      team: 'Par équipe',
      player: 'Par joueur',
    },
    markerColorsHint: {
      team: 'Chaque point porte la couleur de son camp (allié / adverse) — la lecture par défaut.',
      player:
        "Une couleur stable et distincte par joueur, pour en suivre un dans la mêlée. Le camp reste dit par les fiches, le fil et le bandeau.",
    },
    heatLegendLow: 'rare',
    heatLegendHigh: 'fréquent',
    heatLegendHint:
      "Échelle étalonnée sur les lieux fréquentés (médiane au bas, 95e centile en haut) : au-delà, la couleur sature. Un seul point extrême ne peut donc pas écraser le reste de la carte.",
    rosterEmpty:
      "Aucune vie du film n'a pu être rattachée à un joueur : le rejeu reste anonyme.",
    bridgeDiag: (named: number, total: number, collisions: number) =>
      `Pont du film : ${named}/${total} vies nommées, ${collisions} collision(s) de slot.`,
    teamUnknown: 'Sans équipe',
    teamLabelFmt: (name) => `Équipe ${name}`,
    teamNumberedFmt: (n) => `Équipe ${n}`,
    playerScoreLive: "Score personnel à l'instant lu",
    countersLive: "Frags / morts / assistances à l'instant lu",
    countersMatch: 'Frags / morts / assistances du match',
    fdaTooltipFmt: (counters, fda) => `${counters} — FDA ${fda}`,
    scoreBannerLabel: "Score des équipes à l'instant lu",
    scoreBannerAlly: 'Équipe alliée',
    scoreBannerEnemy: 'Équipe adverse',
    scoreBannerClock: 'Position de lecture',
    hillHoldAlly: 'Garde de la colline — équipe alliée',
    hillHoldEnemy: 'Garde de la colline — équipe adverse',
    roundNumberFmt: (index) => `Manche ${index}`,
    roundOfCountFmt: (index, count) => `Manche ${index} sur ${count}`,
    roundDotsLabel: 'Manches gagnées',
    roundsTallyFmt: (ally, enemy) => `${ally} - ${enemy}`,
    roundsTallyLabel: 'Manches gagnées : alliée - adverse',
    roundDotAllyFmt: (index) => `Manche ${index} : gagnée par l'équipe alliée`,
    roundDotEnemyFmt: (index) => `Manche ${index} : gagnée par l'équipe adverse`,
    roundDotPendingFmt: (index) => `Manche ${index} : à jouer`,
    roundOverFmt: (index) => `Manche ${index} terminée`,
    bombArmedFmt: (remaining) => `Bombe armée — ${remaining}`,
    victoryPanelLabel: 'Fin du match',
    victoryScoreLabel: 'Score final',
    trackYou: 'Toi',
    trackAllies: 'Alliés',
    trackDominance: 'Dominance',
    dominanceOfFmt: (team) => `${team} mène aux frags`,
    dominanceTied: 'Égalité aux frags',
    trackScore: 'Score',
    scoreOfFmt: (team) => `${team} mène au score`,
    scoreTied: 'Égalité au score',
    mediaTrack: 'Médias',
    mediaEmpty: 'Aucun média sur ce match',
    mediaOpen: 'Ouvrir le média',
    mediaClose: 'Fermer',
    mediaPausedHint: 'Rejeu en pause',
    mediaHlsUnsupported: 'Ce navigateur ne sait pas lire ce clip.',
    mediaHlsError: 'La lecture du clip a échoué.',
    tracksCollapse: 'Replier les pistes',
    tracksExpand: 'Déplier les pistes',
    unknownPlayer: 'Joueur inconnu',
    markMe: 'Moi',
    healthLabel: 'Santé',
    shieldLabel: 'Bouclier',
    abilityLabel: "Capacité d'armure équipée",
    loadoutUnread: 'armes non lues sur cette vie',
    loadoutAge: 'Armes lues il y a',
    loadoutAhead:
      'Armes de la première image-clé de cette vie, lue dans',
    weaponSecondaryHint: 'secondaire (arme rangée à la dernière lecture)',
    grenadeThrown: 'Grenade lancée',
    eliminatedLabel: 'Éliminé',
    respawnIn: 'Réapparition dans',
    goneLabel: 'Hors film',
    goneValue: 'ne revient plus',
    inventoryAge: 'Inventaire lu il y a',
    inventoryAhead: 'Inventaire de la première image-clé de cette vie, lue dans',
    grenadeSelected: 'Type équipé : le seul porté, donc celui qui partira au prochain lancer.',
    grenadeSelectedRead:
      'Type équipé, LU dans le film (sélecteur de grenade de l’image-clé) : celui qui partira au prochain lancer.',
    grenadeSelUnknown: 'sél. ?',
    grenadeAge: 'Grenades lues il y a',
    grenadeAhead: 'Grenades lues dans',
    abilityUnidentified: (rank) => `capacité non identifiée (rang ${rank})`,
    abilityAge: 'Capacité lue il y a',
    abilityAhead: 'Capacité lue dans',
    abilityChargesFull: 'plein',
    abilityChargesFullHint:
      'Charges pleines — le film ne transmet une lecture qu’après le premier usage.',
    abilityChargesCount: (n) => `Charges restantes : ${n}`,
    abilityChargesAge: 'Charges lues il y a',
    equipmentActive: {
      camo: 'Camouflage actif — le joueur est invisible à l’écran de jeu',
      overshield: 'Surbouclier actif',
    },
    translocationFlash: 'Passage par translocateur quantique',
    zonePresence: {
      field: 'Dans un champ de réparation',
      shroud: 'Dans un écran occultant',
      sensor: 'Détecté — dans la zone d’un capteur de menaces adverse',
    },
    objectiveCarry: {
      flag: 'Porte le drapeau',
      skull: 'Porte le crâne',
      vip: 'Est le VIP',
      hill: 'Tient la colline',
      zone: 'Vient de prendre une base',
      bomb: 'Porte la bombe — ou vient de la faire sauter',
    },
    equipmentUsage: {
      title: "Usages d'équipement",
      viewByPlayer: 'Nombre de gestes par joueur',
      viewTeamShare: "Part de chaque équipe, geste par geste",
      gridTipFmt: (player, column, value) => `${player} — ${column} : ${value}`,
      shareTipFmt: (team, family, count, total) =>
        `${team} — ${family} : ${count} sur ${total}`,
      groupGrapple: 'Grappin',
      groupGrappleHint:
        "Tractions de grappin lues dans le film — la seule activation de capacité que la mesure sait attribuer à un joueur. Un tir sans accroche n'est pas une traction : il est compté à part et n'entre pas dans cette colonne.",
      groupActive: 'États actifs',
      groupActiveHint:
        "Épisodes de camouflage et de surbouclier. Le film mesure que l'effet COURT ; il ne dit pas d'où il vient — un bonus ramassé au socle et une capacité déclenchée produisent le même épisode, et la source n'est pas distinguée. Le nombre et la durée cumulée se lisent ensemble : six épisodes d'une seconde et un épisode de six secondes ne racontent pas la même partie. Les frags sous effet actif se lisent à la précision de la retransmission près (les bornes de l'épisode) ; le camo seul reste sous le seuil de mesure en lecture large (26,2 % des épisodes avec au moins un frag).",
      activeFamily: { camo: 'Camouflage', overshield: 'Surbouclier' },
      activeCount: 'épisodes',
      activeDuration: 'durée',
      activeKillsFamily: { camo: 'Frags sous camo', overshield: 'Frags sous surbouclier' },
      groupDeployed: 'Déploiements',
      groupDeployedHint:
        "Les objets qu'un joueur a réellement DÉPLOYÉS en cours de vie, par famille. Un mur déployé publie deux poses (l'appareil et ses panneaux) et n'en compte qu'une. Les lancers de grenade ont leur propre colonne ; le grappin, le propulseur et le répulseur agissent sur leur porteur et ne posent rien sur le terrain.",
      groupDropped: 'Objets lâchés',
      groupDroppedHint:
        "Les objets de puissance laissés au sol en mourant — bonus et équipements déployables. Ils restent ramassables : savoir qui en sème change la lecture des échanges suivants. Les grenades et les capacités lâchées ne sont pas comptées — près de neuf poses sur dix, et elles ne disent rien du terrain.",
      groupGrenades: 'Grenades lancées',
      groupGrenadesHint:
        "Les lancers lus dans le film, par type. L'auteur du lancer est écrit dans le film — ce n'est pas une déduction de proximité.",
      grenadeRankFmt: (rank) => `Rang ${rank}`,
      powerupPads: 'Socles de bonus de puissance vidés',
      powerupPadsHint:
        "Combien de fois un socle de bonus s'est vidé pendant le match. Ce compte ne descend sur AUCUN joueur, et ce n'est pas un oubli : un socle de bonus s'identifie par un nom, pas par un identifiant d'objet, donc aucun ramassage du film ne peut lui être rattaché. Un même socle peut se vider plusieurs fois — le bonus réapparaît.",
      powerupPadsDenomFmt: (pads) =>
        `sur ${pads} socle${pads > 1 ? 's' : ''} mesuré${pads > 1 ? 's' : ''}`,
      coverageActiveFmt: (lives) => `États actifs mesurés sur ${lives} vies publiées.`,
      coverageGrappleFmt: (pulls, lives) =>
        `${pulls} traction${pulls > 1 ? 's' : ''} de grappin lue${pulls > 1 ? 's' : ''}, réparties sur ${lives} vie${lives > 1 ? 's' : ''}.`,
      unattributedFmt: (count) =>
        `${count} geste${count > 1 ? 's' : ''} mesuré${count > 1 ? 's' : ''} sans propriétaire (vie sans joueur, ou poseur non mesuré) : hors des deux vues.`,
      notMeasured:
        "Le répulseur n'apparaît pas : neuf canaux du film ont été fouillés, aucun ne date son activation. Une colonne vide se lirait « zéro utilisation ». Le propulseur, lui, a désormais son canal d'usage mesuré — validé contre un relevé Theater — et ses poussées se voient sur la carte du rejeu, pas dans ce tableau : le geste dure une demi-seconde.",
      killBadgeFmt: {
        camo: (kills) => `${kills} frags sous camouflage`,
        overshield: (kills) => `${kills} frags sous surbouclier`,
      },
      killBadgeHint:
        "Le meilleur épisode du match pour cette famille. Même réserve que les états actifs : la source de l'épisode n'est pas distinguée (ramassage ou capacité déclenchée), ses bornes sont à la précision de la retransmission près, et le camo seul reste sous le seuil de mesure en lecture large.",
    },
    padControl: {
      title: 'Contrôle des armes spéciales',
      titleHint:
        "Les armes de socle prises pendant le match, et par qui. Chaque prise vient de l'événement de ramassage écrit dans le film : il est daté à la milliseconde et porte son ramasseur. Une occupation de socle qu'aucun ramassage ne couvre — ou que plusieurs couvrent — n'est comptée pour personne : on ne devine pas un ramasseur, on s'abstient et on le dit sous le graphe.",
      axisPickups: 'nombre de prises',
      barTipFmt: (player, team, weapon, count) =>
        `${player} (${team}) — ${weapon} : ${count} prise${count > 1 ? 's' : ''}`,
      unnamedFmt: (count) => `+ ${count} sans nom`,
      attributedFmt: (attributed, occupations) =>
        `${attributed} prise${attributed > 1 ? 's' : ''} attribuée${attributed > 1 ? 's' : ''} sur ${occupations} occupation${occupations > 1 ? 's' : ''} de socle mesurée${occupations > 1 ? 's' : ''}.`,
      missingFmt: (missing) =>
        `${missing} occupation${missing > 1 ? 's' : ''} hors tableau :`,
      gapFmt: {
        ambiguous: (n) =>
          `${n} ambiguë${n > 1 ? 's' : ''} (plusieurs ramassages de la même arme dans la fenêtre)`,
        uncovered: (n) => `${n} sans ramassage correspondant dans le film`,
        unnamed: (n) => `${n} datée${n > 1 ? 's' : ''} sans ramasseur nommé`,
        powerup: (n) =>
          `${n} sur socle de bonus (jamais rattachable : un bonus s'identifie par un nom, pas par une famille d'arme)`,
        unjoined: (n) => `${n} au nom d'un joueur que le film n'a pas vu vivre`,
      },
    },
    collapsedColumnsShowFmt: (count) => `Voir plus (${count})`,
    collapsedColumnsHide: 'Replier',
    collapsedColumnsHint:
      'Les colonnes les moins décisives sont repliées. Rien n’est retiré : les totaux, les dénominateurs et les notes de mesure les comptent toujours.',
    ammoFullLabel: 'Munitions pleines',
    gaugeLabel: 'charge restante',
    exportVideo: 'Exporter la vidéo',
    exportHint:
      "L'export recalcule le film aussi vite que la machine le permet : le fichier ne se paie pas en temps de match.",
    exportDialogTitle: 'Exporter le rejeu en vidéo',
    exportFrom: 'Début',
    exportTo: 'Fin',
    exportWithSound: 'Inclure le son',
    exportStart: 'Exporter',
    exportCancel: 'Annuler',
    exportClose: 'Fermer',
    exportRunningHint: 'Le terrain défile pendant le calcul : le rejeu revient à sa position à la fin.',
    exportProgressFmt: (done, total) => `Image ${done} / ${total}`,
    exportLengthFmt: (clock) => `Plage exportée : ${clock}`,
    exportFailed: "L'export a échoué. Rien n'a été téléchargé.",
    exportPreparing: 'Préparation du son et des images…',
    exportEtaFmt: (clock) => `environ ${clock} restantes`,
    exportDoneFmt: (filename) => `Fichier déposé : ${filename}`,
    exportMutedFallback: "Clip MUET : ce navigateur a refusé la piste sonore.",
    exportTrackMix: 'Mixage complet',
    exportTrackSfx: 'Bruitages',
    exportTrackVoice: 'Voix',
    exportTrackMusic: 'Musique',
    ammoDrawnHint:
      'Emplacement DÉGAINÉ selon le sélecteur du record : la même lecture qui place cette arme en tête de rangée.',
    drawnUnknown: 'dégainée ?',
    inventoryDeadLabel: 'Mort',
    inventoryDeadHint:
      'Lecture vide, et le fil des éliminations donne le joueur pour mort — lue il y a',
    inventoryEmptyLabel: 'Inventaire indisponible',
    inventoryEmptyHint:
      'Lecture d’image-clé sans grenade ni munition, que le fil des éliminations n’explique pas — lue il y a',
    inventoryFallbackHint: 'l’équipement affiché est la dernière lecture pleine, lue il y a',
    inventoryNoPriorHint: 'aucune lecture d’inventaire avant cet instant',
  },
  en: {
    title: '2D replay',
    back: 'Match details',
    play: 'Play',
    pause: 'Pause',
    restart: 'Restart',
    loading: 'Loading replay…',
    empty: 'No 2D replay available for this match.',
    speed: 'Speed',
    time: 'Match time',
    skipBackFmt: (seconds) => `Back ${seconds} s`,
    skipForwardFmt: (seconds) => `Forward ${seconds} s`,
    speedNormal: 'normal',
    speedMuted: 'sound off',
    keySpace: 'Space',
    killFeedTitle: 'Kills',
    killFeedEmpty: 'Nothing at this point of the match.',
    killFeedUnknownWeapon: 'Unidentified weapon',
    killFeedNoAssistHint:
      'Death without an assist — MEASURED: the death carries its event in the film, and it names no one.',
    killFeedAssistHint:
      'Assist read from the film, with its damage share when measured. Shares are not capped at 100%.',
    killFeedKillerShare: (pct) => `killer ${pct}%`,
    killFeedAssistMark: 'Assist',
    killFeedAssistShare: (pct) => `${pct}%`,
    killFeedDeathLabel: 'died',
    killFeedDeathHint:
      'Death with no credited killer (suicide, fall or leaving), read from the film trails.',
    killFeedDeathKind: {
      environment: 'Fall or out of bounds',
      suicide: 'Killed by their own weapon',
    },
    presenceJoined: 'joined the game',
    presenceLeft: 'left the game',
    presenceJoinedHint:
      'Match API participation timestamp: this player joined in progress.',
    presenceLeftHint:
      'Match API participation timestamp: this player left before the end.',
    presenceJoinedDerived: 'joins the game',
    presenceLeftDerived: 'will not return',
    presenceJoinedDerivedHint:
      'First appearance well after kickoff, derived from the film lives (no API participation on this match).',
    presenceLeftDerivedHint:
      'Last film life well before the end: a leaver — or a final elimination in a round-based mode; the film does not tell them apart (no API participation on this match).',
    sound: 'Sound',
    soundHint:
      'Weapon sounds on kills, grenade throws and equipment activations, cut at one second. A weapon with no recorded sound stays silent. Off by default.',
    soundVolume: 'Sound volume',
    soundVolumeMutedHint:
      'Sound is off: the volume sits at zero. Turning the sound back on restores the level you had set.',
    soundFastHint:
      'At this playback speed the sounds would overlap: they come back at 2× or below.',
    soundCategoriesTitle: 'Sounds by category',
    soundCategory: {
      weapon: 'Weapons',
      grenade: 'Grenades',
      melee: 'Melee',
      equipment: 'Equipment',
    },
    captureImage: 'Capture image',
    recordVideo: 'Record video',
    stopRecording: 'Stop recording',
    recordHint:
      'Recording films the replay as it plays: changing speed or moving the cursor shows up in the file. Starting resumes playback if it is paused; pausing, or letting the film end, stops the recording and downloads the clip.',
    settingsButton: 'Settings',
    settingsClose: 'Close settings',
    autoPlay: 'Auto-play',
    autoPlayHint:
      'Turned on, the replay starts on its own when the page opens. Turned off — the default — it opens paused at kickoff and waits for the Play button. The choice is kept from one match to the next; it neither plays nor pauses a replay that is already open.',
    layers: 'Layers',
    layerAim: 'Aim',
    layerAimHint:
      'Look cone: the direction the player is looking at, decoded from the same record as the position. The game only retransmits it when it changes: an older reading fades instead of vanishing, and nothing is drawn beyond five seconds. A short tick sits at the TIP of the cone when the aim is not level: outwards when the player looks up, inwards when they look down. The cone shortens either way — its length alone could not tell them apart.',
    zoomGroup: 'Map framing',
    zoomIn: 'Zoom in (+ or wheel)',
    zoomOut: 'Zoom out (− or wheel)',
    zoomReset: 'Show the whole map',
    zoomLevelFmt: (z: number) => `${z}x`,
    panUp: 'Pan up (Shift + up arrow)',
    panDown: 'Pan down (Shift + down arrow)',
    panLeft: 'Pan left (Shift + left arrow)',
    panRight: 'Pan right (Shift + right arrow)',
    layerGroupPlayers: 'Players',
    layerGroupTerrain: 'Terrain',
    layerGroupObjectives: 'Objectives',
    layerZones: 'Zones',
    layerZonesHint:
      'Official named map zones, extracted from the game. Large zones tile the terrain; dashed outlines are nested floors.',
    layerTrail: 'Trail',
    layerTrailHint:
      'The last seven seconds travelled, behind every marker. Opacity rises towards the head: the most visible trace is always the current one, and that is what gives the direction of travel.',
    effects: 'Effects',
    layerShotFx: 'Shot effects',
    layerShotFxHint:
      'Muzzle flash on every decoded shot, in the tint of the discharge (kinetic, plasma, energy).',
    layerShotFxCoverage:
      'Shot coverage may not be complete: the film only records a shot when damage is applied.',
    layerKillFx: 'Kill effects',
    layerKillFxHint:
      'Line drawn from killer to victim at the moment of the kill. On by default.',
    layerPlacements: 'Deployed equipment',
    layerPlacementsHint:
      'The objects a player actually DEPLOYED while alive: drop wall, threat sensor, threat seeker, quantum translocator beacon, repair field. The BEACON is the return point dropped by the quantum translocator — not the enemy marking the game calls a “ping”. The repair field carries a breathing cross: it says the object heals, it counts nothing — the film publishes no healing cadence, and its dashed circle keeps the reservation about its radius. Anything the measurement classes otherwise stays off the map — nearly nine placements out of ten are in fact the equipment and grenades a player drops on death, and that is not a gesture. The film never says when a piece of equipment disappears: each family therefore keeps to its official duration wherever the game publishes one — 15 s for the sensor, about ten seconds for the drop wall — and the other placements stay on screen until the end of the replay. The wall arc is oriented by where the deployer was looking; when the placement carries no such heading (a little more than once in eight), it follows the deployer’s last direction of travel, and failing that their last known aim — a deduced arc is then drawn dashed. The sensor sweeps its radius every 1.8 s — official game figures, the film carries none — and briefly marks the deployer’s opponents caught by the wave; the seeker emits a single pulse.',
    layerPlacementsDropped: 'Objects dropped on the ground',
    layerPlacementsDroppedHint:
      'The POWER objects a player leaves on the ground when they die: power-ups (overshield, camouflage) and deployable equipment (drop wall, sensor, seeker, beacon, repair field). They can be PICKED UP — knowing they are lying around changes how the next fight reads. A dashed, dimmed ring, never the shape of the active object: on the ground it exerts no radius and no effect. Dropped grenades and abilities stay off the map: they are nearly nine placements out of ten and would say nothing about the terrain. On by default.',
    layerPlacementsUnnamed: 'Unidentified objects',
    layerPlacementsUnnamedHint:
      'Equipment objects the measurement locates without being able to name them: a neutral dot, borrowing no shape from the named families. These show whatever their origin — the point is precisely to see what cannot be named. Off by default.',
    placementFamily: {
      wall: 'Drop wall',
      sensor: 'Threat sensor',
      rift: 'Quantum translocator rift',
      shroud: 'Shroud screen',
      seeker: 'Threat seeker',
      field: 'Repair field',
    },
    placementUnnamedLabel: 'Unidentified equipment object',
    placementOwnerFmt: (name) => `Deployed by ${name}`,
    placementOwnerUnknown: 'Deployer not measured',
    placementDroppedLabel: 'Object dropped on the ground',
    placementDroppedOwnerFmt: (name) => `Dropped by ${name}`,
    placementDroppedAtFmt: (clock) => `On the ground since ${clock}`,
    layerWeaponPads: 'Weapon spots',
    layerWeaponPadsHint:
      'The spots where a weapon reappears during the match, measured on this match: the weapon is drawn large when it changes a game (sniper, sword, hammer, rockets, skewer, cindershot, overshield, camo), small otherwise. Floor pad or wall rack: the measurement only carries a position, it does not tell them apart. The film never dates the moment of pickup — the spot therefore stays UNCERTAIN for the sampling interval, about twenty seconds, rather than going dark at an invented instant. A countdown only appears where the respawn delay could be established; nowhere else. Who took the weapon is never shown: the measurement falls short of the required certainty.',
    padEquipmentFamily: {
      powerup_overshield: 'Overshield',
      powerup_camo: 'Active camouflage',
    },
    layerFlagCarries: 'Flags',
    layerFlagCarriesHint:
      'The life of capture flags, read from the film: carried (the flag follows its carrier frame by frame), on the ground at the last measured position, or at its base. A base keeps a faded flag for as long as its own is elsewhere. A carry whose end NOTHING dates is faded too: its interval runs to the end of the film, an upper bound rather than a measurement.',
    flagSide: {
      ally: 'Allied flag',
      enemy: 'Enemy flag',
      unknown: 'Flag',
    },
    flagState: {
      carried: 'Carried',
      carried_open: 'Carried, end undated',
      dropped: 'On the ground',
      home: 'At base',
    },
    flagCarrierUnknown: 'Carrier not named',
    flagSinceFmt: (seconds) => `For ${Math.round(seconds)} s`,
    flagOpenNote:
      'Nothing dates the end of this carry: the interval runs to the end of the film — an upper bound, not a measured duration.',
    layerVipCrown: 'VIP',
    layerVipCrownHint:
      'The crown of the current VIP, read from the film: each selection opens a wearing period, closed by the VIP’s death or the next selection. The crown follows its bearer frame by frame. A wearing whose end NOTHING dates is faded: its interval runs to the end of the film — an upper bound, not a measurement.',
    layerSkullCarrier: 'Skull carrier',
    layerSkullCarrierHint:
      'Who carries the Oddball skull, read from the film: the carrier is the player whose mode-score ticks rise, a run of ticks being one carry period. The skull follows its carrier frame by frame. A carry whose end NOTHING dates is faded: its interval runs to the end of the film — an upper bound, not a measurement. The skull dropped on the ground is still drawn by the objective-objects layer.',
    layerBombCarrier: 'Bomb',
    layerBombCarrierHint:
      'The Assault bomb, read from the film: a pickup opens a carry period, closed by the drop — often the plant itself — or by the carrier’s death. Carried, the bomb follows its carrier frame by frame; dropped, it stays drawn at its dropper’s last point until the next pickup or the detonation. A carry whose end NOTHING dates is faded: its interval runs to the end of the film — an upper bound, not a measurement.',
    layerGroundWeapons: 'Weapons on the ground',
    layerGroundWeaponsHint:
      'Weapons abandoned on the ground, read from the film: a weapon dropped by a dead player, left behind while picking up another, or ejected from a rack. Each one is drawn where it came to rest, from the moment it lands. Weapons ON THEIR SPOT are not here — they belong to the weapon-spots layer. The END is what the film shows, never a duration from a table: a dated pickup ends the display exactly; otherwise the weapon is drawn solid for as long as a key frame still records it, then FADES to the first key frame that no longer does — the disappearance happened somewhere in between, and nothing is drawn past that bound.',
    padCountdownFmt: (seconds) => `${Math.ceil(seconds)} s`,
    padRespawnMeasuredFmt: (seconds) => `Respawn in ${Math.ceil(seconds)} s`,
    padRespawnExpectedFmt: (seconds) => `Respawn in ≈ ${Math.ceil(seconds)} s`,
    layerHeatmap: 'Heat map',
    layerHeatmapHint:
      'Where the match was played, over the whole match. A cell never reached stays empty: "cold" means seldom visited, no colour at all means never seen.',
    heatmapReading: 'Measure',
    heatmapMode: {
      presence: 'Presence',
      kills: 'Kills',
    },
    heatmapModeHint: {
      presence: 'Time spent by players, read from the film trails.',
      kills: 'Deaths counted where the victim fell, not where the shot came from.',
    },
    heatmapSpanTitle: 'Duration',
    heatmapSpan: {
      match: 'Match',
      live: 'Progressive',
    },
    heatmapSpanHint: {
      match:
        'The entire match, end to end, whatever frame is showing: hot zones read at a glance, like a post-match analysis.',
      live:
        'Only what has been played so far: the map fills in as the replay runs, and stepping back returns it to what it was. Recomputed every two seconds of match time.',
    },
    markerColorsTitle: 'Marker colors',
    markerColorsMode: {
      team: 'By team',
      player: 'Per player',
    },
    markerColorsHint: {
      team: 'Every marker carries its side colour (allied / enemy) — the default reading.',
      player:
        'A stable, distinct colour per player, to follow one through the fight. The side is still told by the cards, the feed and the banner.',
    },
    heatLegendLow: 'rare',
    heatLegendHigh: 'frequent',
    heatLegendHint:
      'Scale calibrated on the visited places (median at the bottom, 95th percentile at the top): beyond that, the colour saturates. A single extreme spot therefore cannot flatten the rest of the map.',
    rosterEmpty: 'No life from the film could be attached to a player: the replay stays anonymous.',
    bridgeDiag: (named: number, total: number, collisions: number) =>
      `Film bridge: ${named}/${total} lives named, ${collisions} slot collision(s).`,
    teamUnknown: 'No team',
    teamLabelFmt: (name) => `Team ${name}`,
    teamNumberedFmt: (n) => `Team ${n}`,
    playerScoreLive: 'Personal score at the moment being played',
    countersLive: 'Kills / deaths / assists at the moment being played',
    countersMatch: 'Kills / deaths / assists for the whole match',
    fdaTooltipFmt: (counters, fda) => `${counters} — KDA ${fda}`,
    scoreBannerLabel: 'Team scores at the moment being played',
    scoreBannerAlly: 'Allied team',
    scoreBannerEnemy: 'Enemy team',
    scoreBannerClock: 'Playback position',
    hillHoldAlly: 'Hill hold — allied team',
    hillHoldEnemy: 'Hill hold — enemy team',
    roundNumberFmt: (index) => `Round ${index}`,
    roundOfCountFmt: (index, count) => `Round ${index} of ${count}`,
    roundDotsLabel: 'Rounds won',
    roundsTallyFmt: (ally, enemy) => `${ally} - ${enemy}`,
    roundsTallyLabel: 'Rounds won: ally - enemy',
    roundDotAllyFmt: (index) => `Round ${index}: won by the allied team`,
    roundDotEnemyFmt: (index) => `Round ${index}: won by the enemy team`,
    roundDotPendingFmt: (index) => `Round ${index}: to play`,
    roundOverFmt: (index) => `Round ${index} over`,
    bombArmedFmt: (remaining) => `Bomb armed — ${remaining}`,
    victoryPanelLabel: 'End of match',
    victoryScoreLabel: 'Final score',
    trackYou: 'You',
    trackAllies: 'Allies',
    trackDominance: 'Dominance',
    dominanceOfFmt: (team) => `${team} leads on kills`,
    dominanceTied: 'Tied on kills',
    trackScore: 'Score',
    scoreOfFmt: (team) => `${team} leads on score`,
    scoreTied: 'Tied on score',
    mediaTrack: 'Media',
    mediaEmpty: 'No media on this match',
    mediaOpen: 'Open media',
    mediaClose: 'Close',
    mediaPausedHint: 'Replay paused',
    mediaHlsUnsupported: 'This browser cannot play this clip.',
    mediaHlsError: 'Clip playback failed.',
    tracksCollapse: 'Collapse tracks',
    tracksExpand: 'Expand tracks',
    unknownPlayer: 'Unknown player',
    markMe: 'Me',
    healthLabel: 'Health',
    shieldLabel: 'Shield',
    abilityLabel: 'Equipped armor ability',
    loadoutUnread: 'weapons not read on this life',
    loadoutAge: 'Weapons read',
    loadoutAhead: 'Weapons from the first keyframe of this life, read in',
    weaponSecondaryHint: 'secondary (weapon holstered at the last reading)',
    grenadeThrown: 'Grenade thrown',
    eliminatedLabel: 'Eliminated',
    respawnIn: 'Respawn in',
    goneLabel: 'Out of film',
    goneValue: 'does not return',
    inventoryAge: 'Inventory read',
    inventoryAhead: 'Inventory from the first keyframe of this life, read in',
    grenadeSelected: 'Equipped type: the only one carried, so the one the next throw will use.',
    grenadeSelectedRead:
      'Equipped type, READ from the film (keyframe grenade selector): the one the next throw will use.',
    grenadeSelUnknown: 'sel. ?',
    grenadeAge: 'Grenades read',
    grenadeAhead: 'Grenades read in',
    abilityUnidentified: (rank) => `unidentified ability (rank ${rank})`,
    abilityAge: 'Ability read',
    abilityAhead: 'Ability read in',
    abilityChargesFull: 'full',
    abilityChargesFullHint:
      'Charges full — the film only transmits a reading after the first use.',
    abilityChargesCount: (n) => `Charges left: ${n}`,
    abilityChargesAge: 'Charges read',
    equipmentActive: {
      camo: 'Active camo — the player is invisible on the game screen',
      overshield: 'Overshield active',
    },
    translocationFlash: 'Quantum translocator jump',
    zonePresence: {
      field: 'Inside a repair field',
      shroud: 'Inside a shroud screen',
      sensor: 'Detected — inside an enemy threat sensor zone',
    },
    objectiveCarry: {
      flag: 'Carrying the flag',
      skull: 'Carrying the oddball',
      vip: 'Is the VIP',
      hill: 'Holding the hill',
      zone: 'Just captured a stronghold',
      bomb: 'Carrying the bomb — or just detonated it',
    },
    equipmentUsage: {
      title: 'Equipment usage',
      viewByPlayer: 'Gesture count by player',
      viewTeamShare: "Each team's share, gesture by gesture",
      gridTipFmt: (player, column, value) => `${player} — ${column}: ${value}`,
      shareTipFmt: (team, family, count, total) => `${team} — ${family}: ${count} of ${total}`,
      groupGrapple: 'Grappleshot',
      groupGrappleHint:
        'Grapple pulls read from the film — the only ability activation the measurement can attribute to a player. A shot with no anchor is not a pull: it is counted separately and never enters this column.',
      groupActive: 'Active states',
      groupActiveHint:
        "Camo and overshield episodes. The film measures that the effect IS RUNNING; it never says where it came from — a power-up picked up from a pad and a triggered ability produce the same episode, and the source is not told apart. Count and cumulative duration read together: six one-second episodes and one six-second episode are not the same game. Kills under active effect read at the precision of the broadcast (the episode's bounds); camo alone stays under the measurement threshold in broad reading (26.2% of episodes with at least one kill).",
      activeFamily: { camo: 'Camo', overshield: 'Overshield' },
      activeCount: 'episodes',
      activeDuration: 'duration',
      activeKillsFamily: { camo: 'Kills under camo', overshield: 'Kills under overshield' },
      groupDeployed: 'Deployments',
      groupDeployedHint:
        'The objects a player actually DEPLOYED while alive, by family. A deployed drop wall publishes two placements (the device and its panels) and counts as one. Grenade throws have their own column; the grappleshot, thruster and repulsor act on their carrier and put nothing on the ground.',
      groupDropped: 'Dropped objects',
      groupDroppedHint:
        'The power objects left on the ground on death — power-ups and deployable equipment. They remain pickable: knowing who scatters them changes how the next fights read. Dropped grenades and abilities are not counted — nearly nine placements out of ten, and they say nothing about the terrain.',
      groupGrenades: 'Grenades thrown',
      groupGrenadesHint:
        'Throws read from the film, by type. The thrower is written in the film — not inferred from proximity.',
      grenadeRankFmt: (rank) => `Rank ${rank}`,
      powerupPads: 'Power-up pads emptied',
      powerupPadsHint:
        'How many times a power-up pad went empty during the match. This count is attached to NO player, and that is not an oversight: a power-up pad is identified by a name, not by an object id, so no pickup in the film can be tied to it. One pad can empty several times — the power-up respawns.',
      powerupPadsDenomFmt: (pads) => `across ${pads} measured pad${pads > 1 ? 's' : ''}`,
      coverageActiveFmt: (lives) => `Active states measured over ${lives} published lives.`,
      coverageGrappleFmt: (pulls, lives) =>
        `${pulls} grapple pull${pulls > 1 ? 's' : ''} read, spread over ${lives} ${lives > 1 ? 'lives' : 'life'}.`,
      unattributedFmt: (count) =>
        `${count} measured gesture${count > 1 ? 's' : ''} with no owner (life with no player, or unmeasured deployer): outside both views.`,
      notMeasured:
        'The repulsor is absent: nine channels of the film were searched, none dates its activation. An empty column would read as "zero uses". The thruster now has its own measured usage channel — validated against a Theater reading — and its bursts show on the replay map, not in this table: the gesture lasts half a second.',
      killBadgeFmt: {
        camo: (kills) => `${kills} kills under camo`,
        overshield: (kills) => `${kills} kills under overshield`,
      },
      killBadgeHint:
        "The best episode of the match for this family. Same reserve as active states: the episode's source is not told apart (picked up or triggered), its bounds are at broadcast precision, and camo alone stays under the measurement threshold in broad reading.",
    },
    padControl: {
      title: 'Power weapon control',
      titleHint:
        'The pad weapons picked up during the match, and by whom. Every pickup comes from the pickup event written in the film: it is timed to the millisecond and carries its picker. A pad occupancy no pickup covers — or that several cover — is counted for nobody: a picker is never guessed, the measurement abstains and says so below the chart.',
      axisPickups: 'pickup count',
      barTipFmt: (player, team, weapon, count) =>
        `${player} (${team}) — ${weapon}: ${count} pickup${count > 1 ? 's' : ''}`,
      unnamedFmt: (count) => `+ ${count} unnamed`,
      attributedFmt: (attributed, occupations) =>
        `${attributed} pickup${attributed > 1 ? 's' : ''} attributed out of ${occupations} measured pad occupanc${occupations > 1 ? 'ies' : 'y'}.`,
      missingFmt: (missing) =>
        `${missing} occupanc${missing > 1 ? 'ies' : 'y'} outside the table:`,
      gapFmt: {
        ambiguous: (n) =>
          `${n} ambiguous (several pickups of the same weapon inside the window)`,
        uncovered: (n) => `${n} with no matching pickup in the film`,
        unnamed: (n) => `${n} timed with no named picker`,
        powerup: (n) =>
          `${n} on a power-up pad (never attachable: a power-up is identified by a name, not by a weapon family)`,
        unjoined: (n) => `${n} named for a player the film never saw alive`,
      },
    },
    collapsedColumnsShowFmt: (count) => `Show more (${count})`,
    collapsedColumnsHide: 'Collapse',
    collapsedColumnsHint:
      'The least game-changing columns are folded away. Nothing is removed: totals, denominators and measurement notes still count them.',
    ammoFullLabel: 'Ammo full',
    gaugeLabel: 'charge left',
    exportVideo: 'Export video',
    exportHint:
      'The export recomputes the film as fast as the machine allows: the file does not cost you a match length.',
    exportDialogTitle: 'Export replay as video',
    exportFrom: 'Start',
    exportTo: 'End',
    exportWithSound: 'Include sound',
    exportStart: 'Export',
    exportCancel: 'Cancel',
    exportClose: 'Close',
    exportRunningHint: 'The field scrolls while it computes: the replay returns to its position at the end.',
    exportProgressFmt: (done, total) => `Frame ${done} / ${total}`,
    exportLengthFmt: (clock) => `Exported range: ${clock}`,
    exportFailed: 'The export failed. Nothing was downloaded.',
    exportPreparing: 'Preparing sound and frames…',
    exportEtaFmt: (clock) => `about ${clock} remaining`,
    exportDoneFmt: (filename) => `File saved: ${filename}`,
    exportMutedFallback: 'MUTED clip: this browser refused the audio track.',
    exportTrackMix: 'Full mix',
    exportTrackSfx: 'Sound effects',
    exportTrackVoice: 'Voice',
    exportTrackMusic: 'Music',
    ammoDrawnHint:
      'Slot DRAWN according to the record selector: the same reading that puts this weapon first in the row.',
    drawnUnknown: 'drawn ?',
    inventoryDeadLabel: 'Dead',
    inventoryDeadHint:
      'Empty reading, and the player is dead according to the kill feed — taken',
    inventoryEmptyLabel: 'Inventory unavailable',
    inventoryEmptyHint:
      'Keyframe reading with no grenade and no ammo, which the kill feed does not explain — taken',
    inventoryFallbackHint: 'the gear shown is the last full reading, taken',
    inventoryNoPriorHint: 'no inventory reading before this moment',
  },
}
