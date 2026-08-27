/**
 * i18nContract.ts — LE CONTRAT DE TEXTE du rejeu 2D : un champ par string affichée, et la
 * raison d'être de chacune en commentaire.
 *
 * EXTRAIT DE `i18n.ts` LE 2026-08-18 (lot R2-V) : le fichier portait le contrat ET les deux
 * tables de traduction, et il avait franchi le seuil de taille du dépôt (CLAUDE.md n°5, 505
 * lignes). La découpe tombe sur une ligne qui se dit en une phrase — d'un côté ce que l'UI
 * PROMET d'afficher, de l'autre ce que chaque langue en DIT.
 *
 * LA PARITÉ FR/EN RESTE TENUE PAR LE TYPAGE : `Record<ReplayLocale, ReplayText>` dans
 * `i18n.ts` refuse toute langue à laquelle il manque un champ.
 */
import type { PadEquipmentFamilyKey } from './weaponPadFamilies'

/**
 * LE TABLEAU DES USAGES D'ÉQUIPEMENT de la page match (onglet Chronologie). Il compte, sur tout
 * le match, ce que le rejeu ne montre qu'image par image.
 *
 * TROIS RÉSERVES SONT PORTÉES PAR CES TEXTES, et aucune ne doit se perdre :
 *
 *  1. `groupActiveHint` — un épisode de camouflage ou de surbouclier est un ÉTAT MESURÉ, pas
 *     un geste : le film dit que l'effet court, il ne dit PAS d'où il vient (bonus ramassé au
 *     socle, ou capacité déclenchée). Compter ces épisodes comme des « utilisations d'objet »
 *     serait affirmer une origine que rien n'établit.
 *  2. `powerupPadsHint` — les vidages de socle de bonus sont ANONYMES PAR MESURE
 *     (`padPickups[].xuid` vaut `null` partout) : la ligne reste au niveau du MATCH, et aucun
 *     libellé ne doit laisser croire qu'on connaît le ramasseur.
 *  3. `notMeasured` — répulseur et propulseur n'ont AUCUN canal d'activation dans le film. Pas
 *     de colonne vide (elle se lirait « zéro utilisation ») : une phrase qui le dit.
 *
 * LES NOMS DE FAMILLE NE SONT PAS ICI, et c'est voulu : ils vivent déjà dans `placementFamily`
 * (règles de rendu) et `padEquipmentFamily` (socles de bonus). Une troisième table de noms
 * divergerait au premier ajout du manifeste.
 */
export interface EquipmentUsageText {
  title: string
  colPlayer: string
  teamTotal: string
  /** Tractions de grappin : la seule ACTIVATION de capacité que le film mesure et attribue. */
  groupGrapple: string
  groupGrappleHint: string
  /** États actifs mesurés (camouflage, surbouclier) : nombre d'épisodes et durée cumulée. */
  groupActive: string
  groupActiveHint: string
  /**
   * LE NOM COURT des deux familles d'état, et pourquoi il ne se prend nulle part ailleurs.
   *
   * `equipmentActive` porte bien ces deux familles, mais ce sont des PHRASES d'infobulle de
   * fiche (« Camouflage actif — le joueur est invisible à l'écran de jeu ») : illisibles en tête
   * de colonne. `padEquipmentFamily` porte bien deux noms courts, mais ce sont ceux des SOCLES
   * de bonus — les employer ici nommerait l'ÉTAT par la source que `groupActiveHint` dit
   * justement ne pas être établie. D'où deux libellés propres, et le typage tient la parité.
   */
  activeFamily: Record<'camo' | 'overshield', string>
  activeCount: string
  activeDuration: string
  groupDeployed: string
  groupDeployedHint: string
  groupDropped: string
  groupDroppedHint: string
  groupGrenades: string
  groupGrenadesHint: string
  /** Repli quand le catalogue du titre ne nomme pas ce rang de grenade (le rang reste vrai). */
  grenadeRankFmt: (rank: number) => string
  /** La ligne ANONYME, au niveau du match — jamais rattachée à un joueur. */
  powerupPads: string
  powerupPadsHint: string
  powerupPadsDenomFmt: (pads: number) => string
  /** Dénominateurs repris de `doc.coverage` : « N épisodes » ne se juge pas sans eux. */
  coverageActiveFmt: (lives: number) => string
  coverageGrappleFmt: (pulls: number, lives: number) => string
  /** Gestes mesurés dont le film ne nomme pas l'auteur : comptés hors tableau, jamais versés. */
  unattributedFmt: (count: number) => string
  notMeasured: string
}

export interface ReplayText {
  title: string
  back: string
  play: string
  pause: string
  restart: string
  loading: string
  empty: string
  speed: string
  time: string
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
  /**
   * LE CURSEUR DE VOLUME QUAND LE SON EST COUPÉ (demande utilisateur du 2026-08-25 : « couper
   * le son ne doit plus faire disparaître la barre de volume »). Le curseur RESTE, à zéro et
   * inerte ; ce texte est ce qui l'empêche de se lire comme un réglage cassé — il dit l'état
   * (zéro) ET ce que fera le retour du son (le niveau réglé revient, il n'est pas perdu).
   */
  soundVolumeMutedHint: string
  /** Le son est activé mais tu par la vitesse de lecture — le dire, pas le cacher. */
  soundFastHint: string
  /** Filtre des sons par catégorie (tiroir de réglages, phase 2, décision du 16/08). */
  soundCategoriesTitle: string
  soundCategory: Record<'weapon' | 'grenade' | 'melee' | 'equipment' | 'objective', string>
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
  /**
   * TRAÎNÉE derrière chaque marqueur (V1, retour utilisateur du 2026-08-18 : « avoir la
   * traînée en option »). ALLUMÉE par défaut : c'est elle qui dit le SENS d'un déplacement,
   * et le verdict du 16/08 sur le marqueur la comptait dans le « Parfait ».
   */
  layerTrail: string
  layerTrailHint: string
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
   * POSES D'ÉQUIPEMENT (schéma 10) : le calque, sa bascule d'objets non identifiés, et les
   * NOMS des familles que le calque sait dessiner. Les clés de `placementFamily` sont les
   * RÈGLES DE RENDU (`PlacementKind`), pas les familles du document : deux familles qui
   * partageraient un tracé partageraient son libellé, et une famille que le calque ne dessine
   * pas n'a besoin d'aucun nom — elle n'a pas d'infobulle.
   *
   * Les noms français des trois familles ajoutées le 2026-08-18 viennent du manifeste du titre
   * (`replay_labels.toml`, `[ability_palettes.ranks]`), la même source que le fil et les
   * fiches : « traqueur de menaces » (rang 12), « champ de réparation » (rang 23),
   * « translocateur quantique » (rang 11) dont la BALISE est l'objet posé.
   *
   * LA RÈGLE `unnamed` N'Y FIGURE PAS : c'est le défaut du serveur, dessiné en point neutre et
   * nommé par `placementUnnamedLabel`. Le garde-rail `placementFamily.guard.test.ts` tient
   * cette liste alignée sur `PLACEMENT_RENDER` (calque) et sur `equipmentFamilies` (Go).
   */
  layerPlacements: string
  layerPlacementsHint: string
  /**
   * OBJETS DE PUISSANCE LÂCHÉS (décision produit du 2026-08-18). La commande apparaît dès que
   * le film en porte, dans TOUS les modes — la restriction « hors Fiesta » a été levée le
   * 2026-08-20. Son libellé n'a donc toujours pas à parler de Fiesta, et pour une raison
   * désormais plus simple : le mode n'entre plus dans la règle.
   */
  layerPlacementsDropped: string
  layerPlacementsDroppedHint: string
  layerPlacementsUnnamed: string
  layerPlacementsUnnamedHint: string
  placementFamily: Record<'wall' | 'sensor' | 'beacon' | 'seeker' | 'field', string>
  /** Ce que dit le point neutre d'un objet dont la nature n'est pas établie. */
  placementUnnamedLabel: string
  /** Ligne « posé par <joueur> » de l'infobulle ; le poseur est une MESURE (proximité). */
  placementOwnerFmt: (name: string) => string
  /** Poseur non mesuré (aucun bipède contemporain à moins de 3 m) : le dire, pas le taire. */
  placementOwnerUnknown: string
  /**
   * L'INFOBULLE D'UN OBJET LÂCHÉ : ce qu'il est, qui l'a lâché, et à quel instant du rejeu.
   *
   * `placementDroppedOwnerFmt` remplace « posé par » : le geste n'est pas le même, et écrire
   * « posé » sur un objet tombé d'un cadavre serait faux. `placementDroppedAtFmt` reçoit un
   * chronomètre déjà formaté (`m:ss`) — un INSTANT et jamais une durée écoulée.
   */
  placementDroppedLabel: string
  placementDroppedOwnerFmt: (name: string) => string
  placementDroppedAtFmt: (clock: string) => string
  /**
   * SOCLES D'ARME (schéma 11) : le calque, ses trois états, son compte à rebours et son cycle.
   *
   * LES TROIS ÉTATS NE SE VALENT PAS, et les libellés doivent le dire : « Disponible » est une
   * présence PROUVÉE, « Pris » une absence PROUVÉE, « Incertain » l'intervalle de ~20 s entre
   * les deux, que le film ne date pas. Aucun de ces libellés ne nomme un joueur : le ramasseur
   * n'est pas publié (`padPickups[].xuid` vaut `null` partout), et aucune ligne d'écran ne doit
   * laisser croire qu'on le connaît.
   *
   * NI MÉDIANE NI ÉCARTS À L'ÉCRAN (verdict du 2026-08-18) : le cycle mesuré ne sert plus qu'à
   * DATER le compte à rebours, il ne s'affiche plus. `padCycleFmt` a donc été retiré avec sa
   * ligne d'infobulle — du texte qu'aucun écran ne rend est du code mort (CLAUDE.md n°7).
   */
  layerWeaponPads: string
  layerWeaponPadsHint: string
  /**
   * LES DRAPEAUX DE CTF (schéma 15) : le calque et son infobulle. Titre au PLURIEL — une partie
   * de capture en porte deux, et c'est leur opposition qui se lit.
   *
   * LE CAMP SE DIT « allié » / « adverse », jamais par une couleur ni un nom d'équipe : la page
   * ne connaît que le point de vue du joueur (accessibilité du rejeu), et un film dont la ligne
   * « moi » manque n'a AUCUN camp allié — d'où la troisième clé, qui ne nomme que l'objet.
   *
   * L'ÉTAT « porté, fin non datée » PORTE SA RÉSERVE DANS SON LIBELLÉ, et `flagOpenNote` la
   * développe : l'intervalle court jusqu'à la fin du film, c'est une BORNE HAUTE. Une icône
   * atténuée seule se lirait comme un effet de style.
   */
  layerFlagCarries: string
  layerFlagCarriesHint: string
  flagSide: Record<'ally' | 'enemy' | 'unknown', string>
  flagState: Record<'carried' | 'carried_open' | 'dropped' | 'home', string>
  /** Le porteur que le tableau de bord ne nomme pas — jamais un identifiant brut à l'écran. */
  flagCarrierUnknown: string
  /** Depuis combien de temps l'état courant dure (infobulle). */
  flagSinceFmt: (seconds: number) => string
  /** La réserve de `carried_open`, en toutes lettres. */
  flagOpenNote: string
  padState: Record<'full' | 'uncertain' | 'empty', string>
  /**
   * LE NOM D'UN SOCLE QUI NE PORTE PAS UNE ARME (schéma 17). Les clés sont les familles
   * d'équipement publiées par le document (`weaponPads[].weapon`), énumérées une par une dans
   * `weaponPadFamilies.ts` — d'où le typage : une famille ajoutée là-bas sans libellé ici ne
   * compile pas, dans AUCUNE des deux langues.
   *
   * POURQUOI CES LIBELLÉS SONT LOCAUX ET NON SERVIS PAR LE DOCUMENT : le manifeste du titre
   * (`replay_labels.toml`) ne porte, sur ses `[[equipment_objects]]`, que l'identité (famille,
   * provenance, nature) — aucun libellé bilingue, aucune icône — et aucun canal du document ne
   * transporte de libellé d'équipement au client (il n'existe pas de `equipmentLabels`). Les
   * noms des familles DESSINÉES vivent déjà ici, dans `placementFamily` : ceux-ci les
   * rejoignent, avec les mêmes mots que le jeu.
   */
  padEquipmentFamily: Record<PadEquipmentFamilyKey, string>
  /** Ce que la donnée ne distingue pas : socle au sol ou râtelier mural (position seule). */
  padPlacementNote: string
  /**
   * La même réserve pour un socle NON-ARME : la question du râtelier mural n'y a pas de sens
   * (un power-up n'est jamais accroché à un mur), mais la position reste une mesure de CE
   * match — l'infobulle ne doit pas laisser croire à un catalogue de carte.
   */
  padPlacementNotePowerUp: string
  /** Compte à rebours COMPACT, celui de la carte (« 12 s ») — il ne dit pas sa source. */
  padCountdownFmt: (seconds: number) => string
  /**
   * LES DEUX COMPTES À REBOURS DE L'INFOBULLE, et leur différence est la seule chose qui compte
   * (D3, 2026-08-27) : `Measured` vise la prochaine apparition VUE dans le film — le rejeu
   * connaît la suite, le chiffre est EXACT et n'a pas à porter de « ≈ » ; `Expected` vise ce que
   * le CYCLE prédit, pour le dernier trou qu'aucune apparition ne ferme, et garde sa réserve.
   *
   * DEUX CLÉS ET NON UN DRAPEAU dans une seule phrase : les deux langues n'insèrent pas la
   * réserve au même endroit, et une phrase à trous se serait figée sur l'ordre du français.
   */
  padRespawnMeasuredFmt: (seconds: number) => string
  padRespawnExpectedFmt: (seconds: number) => string
  /**
   * Carte de chaleur : le calque, ce qu'il mesure, et sa légende. JAMAIS « heatmap » à
   * l'écran (règle FR sans anglicismes) — « carte de chaleur » partout.
   */
  layerHeatmap: string
  layerHeatmapHint: string
  heatmapReading: string
  heatmapMode: Record<'presence' | 'kills', string>
  heatmapModeHint: Record<'presence' | 'kills', string>
  /**
   * PORTÉE DE TEMPS de la carte de chaleur (V2, 2026-08-18) : toute la partie — la lecture
   * d'analyse d'après-match, et le défaut — ou jusqu'à l'image courante, la carte qui se
   * remplit en même temps que le rejeu.
   */
  heatmapSpanTitle: string
  heatmapSpan: Record<'match' | 'live', string>
  heatmapSpanHint: Record<'match' | 'live', string>
  /**
   * COULEUR DES POINTS des joueurs (option du tiroir, 2026-08-24) : par équipe — le défaut,
   * la couleur dit le camp (D1) — ou distincte par joueur, pour suivre quelqu'un dans la
   * mêlée. Les hints disent ce que chaque lecture garde et ce qu'elle déplace.
   */
  markerColorsTitle: string
  markerColorsMode: Record<'team' | 'player', string>
  markerColorsHint: Record<'team' | 'player', string>
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
  /**
   * LE SCORE PERSONNEL À L'INSTANT LU (schéma 12) — celui du joueur sur sa fiche. Ce n'est
   * PAS le score final : il tique avec la lecture. Le score d'ÉQUIPE, lui, ne vit plus
   * qu'au bandeau (les chiffres en tête de colonne sont partis le 2026-08-24).
   */
  playerScoreLive: string
  /** Frags / morts / assistances : deux grandeurs qui ne se confondent pas. */
  countersLive: string
  countersMatch: string
  /**
   * LE BANDEAU DE SCORE au-dessus du terrain : deux barres de camp encadrant l'horloge de
   * lecture. Les deux camps s'y nomment par leur RAPPORT au joueur de la page (allié /
   * adverse) et non par leur nom d'équipe, qui est déjà en tête des colonnes de fiches —
   * le bandeau dit un affrontement, la colonne dit une identité.
   *
   * `roundNumberFmt` est le rang seul, affiché sous l'horloge ; `roundOfCountFmt` le
   * situe dans le match pour l'infobulle. Aucun des deux ne porte de valeur : entre deux
   * barres, un nombre sans camp ne se rattacherait à personne.
   */
  scoreBannerLabel: string
  scoreBannerAlly: string
  scoreBannerEnemy: string
  scoreBannerClock: string
  roundNumberFmt: (index: number) => string
  roundOfCountFmt: (index: number, count: number) => string
  /** RETOURNEMENT : l'instant où le match change de meneur (marque sur la frise). */
  leadChange: string
  leadChangeAtFmt: (time: string, team: string) => string
  unknownPlayer: string
  /**
   * Marques d'identité devant un nom. Le glyphe « moi » ne se DESSINE plus nulle part
   * (demande utilisateur du 2026-08-24) : `markMe` ne sert plus qu'au libellé lecteur
   * d'écran du fil (retirer un dessin ne doit pas retirer une information). La marque
   * « ami » garde son glyphe.
   */
  markMe: string
  markFriend: string
  healthLabel: string
  shieldLabel: string
  abilityLabel: string
  loadoutUnread: string
  loadoutAge: string
  loadoutAhead: string
  weaponSecondaryHint: string
  /** Badge de lancer sur la fiche (le `.gic` du POC) : l'auteur est écrit dans le film. */
  grenadeThrown: string
  respawnIn: string
  respawnUnknown: string
  /** Ligne d'inventaire : grenades, capacité, munitions. */
  inventoryAge: string
  inventoryAhead: string
  grenadeSelected: string
  grenadeSelectedRead: string
  grenadeSelUnknown: string
  /**
   * ÂGE PROPRE À LA BOÎTE DE GRENADES (schéma 20) : l'axe `grenadeReads` ne tombe pas sur les
   * images-clés de l'inventaire, la boîte ne peut donc pas emprunter l'âge de la rangée. Un âge
   * négatif est une lecture À VENIR et se dit comme telle — jamais déguisée en passé.
   */
  grenadeAge: string
  grenadeAhead: string
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
  /**
   * LE TABLEAU DES USAGES D'ÉQUIPEMENT (page match, onglet Chronologie). Bloc à part parce que
   * ces textes ne servent PAS le rejeu lui-même : ils servent son BILAN, une autre surface.
   */
  equipmentUsage: EquipmentUsageText
  /** Pictogramme « munitions pleines » (emplacement jamais écrit) : décision produit 4. */
  ammoFullLabel: string
  ammoDrawnHint: string
  drawnUnknown: string
  /**
   * Lecture d'inventaire VIDE (schéma 19, `inventory[].empty`). Le LIBELLÉ se lit à l'écran,
   * l'INDICE porte l'explication et se termine par l'âge de la lecture VIDE — la sienne, pas
   * celle de l'équipement affiché à côté : les deux lectures ne datent pas du même instant, et
   * les confondre ferait passer un état de vingt secondes pour frais.
   *
   * DEUX ÉTATS ET PAS UN : `dead` est corroboré par le fil des éliminations, `empty` ne l'est
   * pas. Les fondre sous un seul mot ferait affirmer une mort qu'aucune pièce n'établit.
   *
   * LA SECONDE MOITIÉ DE L'INFOBULLE DIT CE QUE L'ÉCRAN MONTRE À CÔTÉ, et elle a DEUX formes
   * parce qu'il y a deux situations. `inventoryFallbackHint` (suivi de l'âge de la lecture
   * PLEINE) quand une lecture pleine antérieure est effectivement substituée ;
   * `inventoryNoPriorHint` quand il n'en existe AUCUNE — la fiche n'affiche alors aucun
   * équipement, et promettre « la dernière lecture pleine » serait faux.
   */
  inventoryDeadLabel: string
  inventoryDeadHint: string
  inventoryEmptyLabel: string
  inventoryEmptyHint: string
  inventoryFallbackHint: string
  inventoryNoPriorHint: string
  gaugeLabel: string
}
