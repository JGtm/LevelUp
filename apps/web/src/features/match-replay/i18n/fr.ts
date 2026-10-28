/** Textes du rejeu 2D en français (contrat : `i18nContract.ts`). */
import type { ReplayText } from './i18nContract'

export const FR_TEXT = {
  title: 'Rejeu 2D',
  back: 'Fiche du match',
  play: 'Lecture',
  pause: 'Pause',
  restart: 'Recommencer',
  loading: 'Chargement du rejeu…',
  empty: 'Aucun rejeu 2D disponible pour ce match.',
  loadErrorTitle: 'Le rejeu n’a pas pu être chargé',
  loadErrorDescription: 'Le serveur était occupé ou injoignable : réessayez dans quelques secondes.',
  loadErrorRetry: 'Réessayer',
  schemaBadgeUpToDateFmt: (schemaVersion) => `Schéma ${schemaVersion} · à jour`,
  schemaBadgeStaleFmt: (schemaVersion, latestSchemaVersion) =>
    `Schéma ${schemaVersion} · à recuire (dernier : ${latestSchemaVersion})`,
  schemaBadgeStaleNoTargetFmt: (schemaVersion) => `Schéma ${schemaVersion} · à recuire`,
  schemaBadgeStalePublicationFmt: (schemaVersion, latestSchemaVersion) =>
    `Schéma ${schemaVersion} · à recuire (dernier : ${latestSchemaVersion}) — seule la publication a changé`,
  schemaBadgeUnknownFmt: (schemaVersion) => `Schéma ${schemaVersion}`,
  schemaBadgeInvalidFmt: (schemaVersion, detail) =>
    `Schéma ${schemaVersion} · contrat non respecté (${detail})`,
  contractUnknownKeysFmt: (keys) => `clé(s) inconnue(s) : ${keys}`,
  contractInvalidFieldFmt: (path, detail) => `${path} : ${detail}`,
  contractMalformed: 'document non conforme au contrat',
  offscreenMarkerFmt: (name, meters) => `${name} · ${Math.round(meters)} m`,
  offscreenGroupMarkerFmt: (count, meters) => `${count} joueurs · ${Math.round(meters)} m`,
  openAtNoOriginTitle: 'Instant non calé sur cet artefact',
  openAtNoOriginDescription:
    'Le lien demandait un instant précis, mais l’origine du film n’est pas établie pour ce rejeu : l’instant ne peut pas être atteint exactement. Recuisson nécessaire — le rejeu s’ouvre au début.',
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
    'Mort sans assistant : la mort porte son événement dans le film, et il ne déclare personne.',
  killFeedAssistHint:
    'Assistant lu dans le film, avec sa part de dégâts. Les parts ne sont pas bornées à 100 %.',
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
  fullscreenEnter: 'Plein écran',
  fullscreenExit: 'Quitter le plein écran',
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
  zoomReset: 'Revoir toute la carte (0)',
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
    "Un tir s'affiche lorsqu'il applique un dégât.",
  layerKillFx: 'Effets de mort',
  layerKillFxHint:
    "Trait orienté du tueur vers la victime, à l'instant de l'élimination. Allumé par défaut.",
  layerPlacements: 'Équipements posés',
  layerPlacementsHint:
    "Les objets qu'un joueur a DÉPLOYÉS en cours de vie : mur de protection, capteur de menaces, traqueur de menaces, écran occultant, balise du translocateur quantique, champ de réparation. La BALISE est le point de retour que pose le translocateur quantique — ce n'est pas le marquage d'un ennemi, que le jeu appelle « ping ». Le champ de réparation porte une croix qui respire : elle dit que l'objet soigne, elle ne compte rien — le film ne publie aucune cadence de soin, et son cercle pointillé garde la réserve sur sa portée. Ce que la mesure classe autrement ne se dessine pas — près de neuf poses sur dix sont en réalité l'équipement et les grenades qu'un joueur lâche en mourant, et ce n'est pas un usage. Le capteur, le traqueur, l'écran occultant et le champ de réparation se dessinent là où l'objet s'est posé, lu dans son vol ; un objet dont le film ne donne pas le lieu de repos ne se dessine pas. Le film écrit la suppression du capteur et du champ de réparation : ils restent affichés de leur atterrissage jusqu'à cette suppression (le capteur se tient à ses 15 s officielles quand le film n'en donne aucune). Il ne date rien d'autre : le mur se tient à sa durée officielle — une dizaine de secondes — et les autres poses restent affichées jusqu'à la fin du rejeu. L'arc du mur est orienté par le regard du poseur ; quand la pose ne porte pas ce cap (un peu plus d'une fois sur huit), il suit la dernière direction de déplacement du poseur, et à défaut sa dernière visée connue — un arc déduit se trace alors en pointillé. Le capteur balaie sa portée toutes les 1,8 s — chiffres officiels du jeu, le film n'en porte aucun — et marque brièvement les adversaires du poseur qui s'y trouvent au passage de l'onde ; le traqueur, lui, n'émet qu'une seule impulsion.",
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
  placementDroppedLabel: 'Objet lâché au sol',
  placementDroppedOwnerFmt: (name) => `Lâché par ${name}`,
  placementDroppedAtFmt: (clock) => `Au sol depuis ${clock}`,
  layerWeaponPads: "Emplacements d'arme",
  layerWeaponPadsHint:
    "Les endroits où une arme réapparaît au fil du match : l'arme y est dessinée en grand quand elle change une partie (fusil de précision, épée, marteau, roquettes, empaleur, crémateur, surbouclier, camouflage), en petit sinon. Socle au sol et râtelier mural partagent le même dessin. Après un ramassage, l'emplacement s'atténue pendant une vingtaine de secondes. Un compte à rebours accompagne la réapparition de l'arme.",
  padEquipmentFamily: {
    powerup_overshield: 'Surbouclier',
    powerup_camo: 'Camouflage actif',
  },
  layerModeObjectives: 'Objectifs du mode',
  layerModeObjectivesHint:
    "Zones et points d'objectif du mode joué (collines, bases, zones de capture, socles et points de livraison), aux couleurs des camps. Les anneaux autour d'un point et les contours autour d'une zone indiquent l'étage, comme pour les joueurs : plus il y en a, plus c'est haut.",
  layerFlagCarries: 'Drapeaux',
  layerFlagCarriesHint:
    "La vie des drapeaux de capture, lue dans le film : porté (le drapeau suit son porteur image par image), au sol à sa dernière position, ou à sa base. La base garde un drapeau atténué tant que le sien est ailleurs. Un portage qui court jusqu'à la fin du film s'affiche atténué lui aussi.",
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
  flagReturnFmt: (progress) => `Retour ${Math.round(progress * 100)} %`,
  flagOpenNote:
    "Ce portage court jusqu'à la fin du film.",
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
  layerGroundWeaponsSpecialOnly: 'Armes spéciales seulement',
  layerGroundWeaponsSpecialOnlyHint:
    "N'affiche que les armes de puissance, de précision longue et hors gabarit (fusil de précision, épée, marteau, roquettes, empaleur, crémateur, aiguiseur, faisceau sentinelle...) : ce qu'un socle distribue et qu'un adversaire a intérêt à ramasser. Une arme dont le rôle n'est pas connu reste visible bascule éteinte, et disparaît bascule allumée — un rôle manquant ne s'affirme jamais spécial.",
  layerGroundWeaponsStarting: 'Afficher aussi les armes de départ au sol',
  layerGroundWeaponsStartingHint:
    "Par défaut, une arme lâchée qui est une arme de départ du match (celle que tout le monde porte à l'apparition) n'est pas dessinée. Les armes posées par la carte restent toujours visibles.",
  groundWeaponSpawned: 'apparue',
  groundWeaponDroppedByFmt: (name) => `lâchée par ${name}`,
  groundWeaponDropperUnknown: 'lâchée par un joueur non nommé',
  groundWeaponPickedByFmt: (name) => `reprise par ${name}`,
  groundWeaponPickerUnknown: 'reprise par un joueur non nommé',
  groundWeaponAmmoExactFmt: (mag, res) => `${mag} au chargeur, ${res} en réserve`,
  groundWeaponAmmoFmt: (mag, age) =>
    `≈ ${mag} munitions au chargeur, lues ${age} avant le lâcher`,
  groundWeaponAmmoResFmt: (mag, res, age) =>
    `≈ ${mag} au chargeur et ${res} en réserve, lues ${age} avant le lâcher`,
  layerVehicles: 'Véhicules',
  layerVehiclesHint:
    "Les véhicules du match, lus dans le film : leur naissance, leur trajectoire orientée dans le sens du déplacement (le dernier cap connu à l'arrêt), et qui est à bord. Un véhicule sans occupant connu reste neutre ; un châssis que la table ne nomme pas encore se dessine en petit repère neutre, jamais avec le véhicule d'un voisin. LA DISPARITION DU SPRITE N'EST PAS UNE DESTRUCTION : le film ne date aucune destruction — le véhicule reste dessiné jusqu'à la PREMIÈRE preuve mesurée de son absence (fenêtre d'environ 20 s après la dernière preuve de présence), puis s'efface — rien de plus.",
  padCountdownFmt: (seconds) => `${Math.ceil(seconds)} s`,
  padRespawnMeasuredFmt: (seconds) => `Réapparition dans ${Math.ceil(seconds)} s`,
  padRespawnExpectedFmt: (seconds) => `Réapparition dans ≈ ${Math.ceil(seconds)} s`,
  vehicleCycleTitle: 'Emplacement de véhicule',
  vehicleCycleFmt: (medianS, p10S, p90S) =>
    `Cycle ≈ ${Math.round(medianS)} s (${Math.round(p10S)} à ${Math.round(p90S)})`,
  vehicleCycleGapsFmt: (gaps) => `Sur ${gaps} cycle${gaps > 1 ? 's' : ''}`,
  vehicleCycleOccupied: 'Véhicule présent',
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
  seatVacant: 'Place libre',
  seatVacantHint: "Personne ne tient cette place à cet instant : elle attend son prochain occupant.",
  seatNotSpawned: 'Pas encore apparu',
  seatNotSpawnedHint: "Ce joueur tient sa place, mais il n'est pas encore apparu sur la carte.",
  bridgeDiag: (named: number, total: number, collisions: number) =>
    `Pont du film : ${named}/${total} vies nommées, ${collisions} collision(s) de slot.`,
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
  roundDotAllyFmt: (index) => `Manche ${index} : gagnée par l'équipe alliée`,
  roundDotEnemyFmt: (index) => `Manche ${index} : gagnée par l'équipe adverse`,
  roundDotPendingFmt: (index) => `Manche ${index} : à jouer`,
  roundOverFmt: (index) => `Manche ${index} terminée`,
  bombArmedFmt: (remaining) => `Bombe armée — ${remaining}`,
  victoryPanelLabel: 'Fin du match',
  victoryScoreLabel: 'Score final',
  trackTeammates: 'Coéquipiers',
  trackDominance: 'Dominance',
  viewpointLabel: 'Joueur suivi',
  viewpointNoData: 'Aucune donnée de match pour ce joueur',
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
  loadoutAge: 'Armes lues il y a',
  loadoutBirth: 'Dotation de naissance',
  loadoutAhead:
    'Armes de la première image-clé de cette vie, lue dans',
  weaponSecondaryHint: 'secondaire (arme rangée à la dernière lecture)',
  weaponStowedFmt: (name) => `Arme rangée : ${name}`,
  playerScoreLiveFmt: (score) => `Score personnel à l'instant lu : ${score}`,
  ammoHintFmt: (ammo) =>
    ammo.kind === 'full'
      ? 'Munitions pleines'
      : ammo.kind === 'charge'
        ? `Charge restante : ${ammo.pct} %`
        : ammo.res === undefined
          ? `Munitions : ${ammo.mag}`
          : `Munitions : ${ammo.mag} / ${ammo.res}`,
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
  translocationPose: 'Faille du translocateur quantique posée',
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
    viewByPlayer: "Usage d'équipements, par joueur",
    infoByPlayer:
      "Objets pris par chaque joueur du match, une pile par issue : servi (activé pour un bonus, posé pour le mur, charge consommée pour le reste), gardé sans servir, lâché en mourant ; le grappin compte ses tractions. Chaque colonne a sa propre échelle.",
    legendUsed: 'Servi',
    legendKept: 'Gardé sans servir',
    legendDropped: 'Lâché',
    legendGrapple: 'Tractions de grappin',
    gridTipFmt: (player, column, value) => `${player} — ${column} : ${value}`,
    groupGrapple: 'Grappin',
    outcomeUsedFmt: (count) => `Servi : ${count}`,
    outcomeKeptFmt: (count) => `Gardé sans servir : ${count}`,
    outcomeDroppedFmt: (count) => `Lâché : ${count}`,
    outcomeTotalTakenFmt: (count) => `${count} objet${count > 1 ? 's' : ''} pris`,
    killBadgeFmt: {
      camo: (kills) => `${kills} frags sous camouflage`,
      overshield: (kills) => `${kills} frags sous surbouclier`,
    },
    killBadgeHint:
      "Le meilleur épisode du match pour cette famille. Même réserve que les états actifs : la source de l'épisode n'est pas distinguée (ramassage ou capacité déclenchée), ses bornes sont à la précision de la retransmission près, et le camo seul reste sous le seuil de mesure en lecture large.",
  },
  ammoFullLabel: 'Munitions pleines',
  gaugeLabel: 'charge restante',
  exportVideo: 'Exporter la vidéo',
  exportHint:
    "L'export recalcule le film aussi vite que la machine le permet : le fichier ne se paie pas en temps de match.",
  exportDialogTitle: 'Exporter le rejeu en vidéo',
  exportFrom: 'Début',
  exportTo: 'Fin',
  exportWithSound: 'Inclure le son',
  exportFormat: 'Format',
  exportFormatOptionFmt: (name, width, height) => `${name} (${width} × ${height})`,
  exportFraming: 'Cadrage',
  exportFramingWhole: 'Carte entière',
  exportFramingCurrentFmt: (zoom) => `Cadrage actuel (${String(zoom).replace('.', ',')}x)`,
  exportStart: 'Exporter',
  exportCancel: 'Annuler',
  exportClose: 'Fermer',
  exportRunningHint: 'Le terrain défile dans le cadre de la vidéo pendant le calcul : le rejeu revient à sa position à la fin.',
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
} satisfies ReplayText
