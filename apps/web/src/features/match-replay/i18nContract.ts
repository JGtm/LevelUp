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

export interface ReplayText {
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
  /**
   * TRAÎNÉE derrière chaque marqueur (V1, retour utilisateur du 2026-08-18 : « avoir la
   * traînée en option »). ALLUMÉE par défaut : c'est elle qui dit le SENS d'un déplacement,
   * et le verdict du 16/08 sur le marqueur la comptait dans le « Parfait ».
   */
  layerTrail: string
  layerTrailHint: string
  zoneLabel: string
  /**
   * EFFETS D'ÉVÉNEMENT, réglables séparément (décision utilisateur du 16/08) : les éclairs
   * de bouche de TOUS les tirs, et le trait tueur -> victime des éliminations. Le premier
   * porte une RÉSERVE DE MESURE affichée en clair (le film n'enregistre un tir que
   * lorsqu'un dégât est appliqué) : elle ne vit pas dans un commentaire, elle est à l'écran.
   */
  effects: string
  /** Titre de la section FICHES du tiroir (la colonne d etat, pas le canvas). */
  cards: string
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
  /**
   * FICHES COMPACTES (B2/R2-7) : la bascule et ce qu'elle change. L'infobulle DIT ce qu'on
   * perd — les munitions des armes qui ne sont pas en main — parce qu'un réglage qui retire
   * de l'information doit l'annoncer.
   */
  cardsCompact: string
  cardsCompactHint: string
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
  /** Compte à rebours COMPACT, celui de la carte (« 12 s »). */
  padCountdownFmt: (seconds: number) => string
  /** Compte à rebours de l'infobulle, en toutes lettres. */
  padRespawnFmt: (seconds: number) => string
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
   * LE SCORE À L'INSTANT LU (schéma 12) — celui de l'équipe en tête de colonne, celui du
   * joueur sur sa fiche. Ce n'est PAS le score final : il tique avec la lecture.
   */
  scoreLive: string
  playerScoreLive: string
  /** Frags / morts / assistances : deux grandeurs qui ne se confondent pas. */
  countersLive: string
  countersMatch: string
  /**
   * MANCHE : les modes qui en ont (Oddball) remettent le compteur à zéro à chaque manche.
   * L'en-tête affiche le TOTAL du match, et rappelle en second la manche en cours.
   */
  roundShortFmt: (index: number) => string
  roundLabelFmt: (index: number, count: number, value: number) => string
  /**
   * LE BANDEAU DE SCORE au-dessus du terrain : deux barres de camp encadrant l'horloge de
   * lecture. Les deux camps s'y nomment par leur RAPPORT au joueur de la page (allié /
   * adverse) et non par leur nom d'équipe, qui est déjà en tête des colonnes de fiches —
   * le bandeau dit un affrontement, la colonne dit une identité.
   *
   * `roundNumberFmt` est le rang seul, affiché sous l'horloge ; `roundOfCountFmt` le
   * situe dans le match pour l'infobulle. Aucun des deux ne porte de valeur : entre deux
   * barres, un nombre sans camp ne se rattacherait à personne (`roundLabelFmt`, lui, sert
   * une colonne, où le camp est acquis).
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
