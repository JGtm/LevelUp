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
