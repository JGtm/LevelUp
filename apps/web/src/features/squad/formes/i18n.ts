/**
 * i18n.ts — LE VOCABULAIRE du bloc « formes retenues » (artefact 2ec1b8eb) :
 * intertitres, axes, familles, rôles, colonnes d'objectif et étiquettes communes
 * aux six formes. Les dix-neuf cartes (titres, notes, sous-titres) vivent dans
 * `cardsI18n.ts` — un seul fichier aurait dépassé le seuil de taille.
 *
 * FR ET EN À PARITÉ DE TYPAGE (`Record<Locale, …>`) : une clé ajoutée d'un côté
 * casse la compilation de l'autre. Le FR reprend MOT POUR MOT l'artefact — c'est
 * lui la spécification, et sa langue fait partie de ce qui a été validé.
 *
 * LES LIBELLÉS DE COLONNE D'OBJECTIF sont une DEUXIÈME copie assumée de ceux de
 * la vue match (`features/match-view/i18n.ts`, table `objectives.cols`) :
 * l'import croisé entre features est interdit par le ratchet. À la troisième
 * copie, centraliser et poser le garde-rail (règle CLAUDE.md n°6).
 */
import type { Locale } from '@/lib/i18n/locale'

import type { EquipmentAxis } from './model/access'
import type { ObjectiveRole } from './model/objectives'
import type { WeaponClass } from './model/pads'

export interface FormesBlockText {
  title: string
  /**
   * L'AIDE ⓘ du titre du bloc : ce que le bloc mesure, et ce qu'il ne mesure pas.
   *
   * TROIS PHRASES AU PLUS (décision 9 du 2026-09-19). Le constat et le lexique qui
   * vivaient ici en pavés de texte gris — cinq à huit phrases chacun — ont été résumés :
   * un lexique se lit une fois, puis n'est plus qu'un mur entre deux formes.
   */
  aide: string
}

export interface FormesText {
  /** Le NOM ACCESSIBLE de la section. Plus de titre a l'ecran depuis la decision D5 du
   *  2026-09-21 : trois intertitres se suivaient, le premier ne nommait rien de lisible. */
  sectionTitle: string
  /**
   * CE QUE DIT UN BLOC SANS DONNEE (decision D8 du 2026-09-21). Un bloc vide reste
   * affiche et NOMME SA CAUSE : sans cause, un ecran vide se lit comme une panne. La
   * cause se deduit des comptes du bloc ; `generic` est le repli quand aucune ne se
   * deduit.
   */
  empty: {
    noFilm: string
    noEquipment: string
    noPads: string
    padsUnnamedOnly: string
    generic: string
  }
  blocks: { equipment: FormesBlockText; weapons: FormesBlockText; objectives: FormesBlockText }
  axes: Record<EquipmentAxis, string>
  weaponClasses: Record<WeaponClass, string>
  allWeapons: string
  allWeaponsHint: string
  roles: Record<ObjectiveRole, string>
  roleHints: Record<ObjectiveRole, string>
  columns: Record<string, string>
  /** Les colonnes propres au bloc « contrôle des armes spéciales ». */
  padsColumns: {
    /** Le nom de la grandeur « prises de socle » dans une forme. */
    pads: string
    rate: string
    mine: string
    occupations: string
    rateLegend: string
    rateAxis: string
    namedPickupsFmt: (n: number) => string
  }
  /** Étiquettes communes aux six formes. */
  common: {
    notMeasured: string
    noMeasure: string
    inMyTeam: string
    inLobby: string
    myShare: string
    spread: string
    spreadByMatch: string
    parity: string
    lineParity: string
    parityFmt: (pct: string) => string
    aboveFmt: (above: number, total: number) => string
    pointsFmt: (signed: string) => string
    morePlus: string
    moreThanOpponent: string
    lessThanLobby: string
    lessThanOpponent: string
    less: string
    outOfFmt: (value: string, total: string) => string
    enemyTeam: string
    enemyTeamCounted: string
    teamRest: string
    noFilm: string
    noMeasureOnAxis: string
    aboveBelowByMatch: string
    unknownWeapon: string
    occupationsShortFmt: (n: number) => string
    /** Le dénominateur d'une piste dont l'escouade est le tout. */
    inSquadFmt: (n: number) => string
    /**
     * Le pied d'une forme par match ALIMENTÉE PAR LE FILM : ce qu'elle montre,
     * ce qu'elle a laissé de côté faute de place, ce qu'elle a écarté faute de
     * film. Une forme qui se borne sans le dire ment sur sa portée.
     */
    foldMeasuredFmt: (shown: number, hidden: number, unmeasured: number) => string
    /** Le pied d'une forme bornée en nombre seulement (les matchs d'un mode). */
    foldListFmt: (shown: number, hidden: number) => string
    matchesShortFmt: (n: number) => string
    gesturesAxis: string
    gesturesPerMatchAxis: string
    /** L'axe du bâton d'étendue — une seule échelle, pas une par colonne. */
    spreadAxis: string
    /** Le préfixe de la moyenne écrite à droite du bâton. */
    meanPrefix: string
    lowestToHighest: string
    meanPerMatch: string
    shareAxis: string
    lobbyShareAxis: string
    squadShareAxis: string
    gapAxis: string
    padShareAxis: string
    matchesAxis: string
    /**
     * L'en-tête d'une colonne de LÂCHERS par famille (D9, 2026-09-21). Forme NEUTRE
     * (« Lâchés : Surbouclier »), et c'est délibéré : les noms de famille du manifeste
     * ont les deux genres (« Balise de translocation », « Capteur »), un participe
     * accordé au masculin s'y tromperait une fois sur deux.
     */
    droppedFamilyFmt: (family: string) => string
    totalFmt: (n: string) => string
    valueTipFmt: (row: string, column: string, value: string) => string
    /** La règle des prises nettes, en toutes lettres, sous la valeur. */
    juggleFoldedFmt: (seconds: string) => string
    matchTipFmt: (match: string, row: string, value: string, gap: string) => string
    gapTipFmt: (row: string, value: string, parity: string, gap: string) => string
    shareTipFmt: (row: string, side: string, value: string, detail: string) => string
    spreadTipFmt: (row: string, side: string, from: string, to: string) => string
    segmentTipFmt: (name: string, row: string, value: string, total: string) => string
    rangeTipFmt: (row: string, from: string, to: string) => string
    meanTipFmt: (row: string, value: string) => string
  }
}

/**
 * Un COMPTE de matchs, avec son séparateur de milliers : « 1 043 matchs », pas
 * « 1043 matchs ». La typographie d'un millier fait partie du français.
 */
const frCount = (n: number): string => n.toLocaleString('fr-FR')
const enCount = (n: number): string => n.toLocaleString('en-GB')

const FR_COLUMNS: Record<string, string> = {
  flag_captures: 'Drapeaux capturés',
  // Prises NETTES : le compteur officiel du jeu compte chaque ramassage, donc
  // aussi le jonglage (lancer le drapeau devant soi pour courir plus vite, puis
  // le reprendre). Cette grandeur-ci replie ces allers-retours.
  flag_grabs_net: 'Prises nettes',
  flag_capture_assists: 'Aides à la capture',
  flag_steals: 'Drapeaux volés',
  flag_returners_killed: 'Rapatrieurs abattus',
  flag_returns: 'Retours',
  flag_secures: 'Drapeaux sécurisés',
  flag_carriers_killed: 'Porteurs abattus',
  time_as_flag_carrier_seconds: 'Temps de portage',
  zone_captures: 'Zones capturées',
  zone_secures: 'Zones sécurisées',
  zone_offensive_kills: 'Frags offensifs de zone',
  zone_defensive_kills: 'Frags défensifs de zone',
  time_in_zones_seconds: 'Temps en zone',
  skull_grabs: 'Crânes récupérés',
  skull_carriers_killed: 'Porteurs de crâne abattus',
  time_as_skull_carrier_seconds: 'Temps de portage',
  power_seeds_deposited: 'Graines déposées',
  power_seeds_stolen: 'Graines volées',
  power_seed_carriers_killed: 'Porteurs de graine abattus',
  time_as_power_seed_carrier_seconds: 'Temps de portage',
  successful_extractions: 'Extractions réussies',
  extraction_initiations_completed: 'Amorçages menés à terme',
  extraction_conversions_completed: 'Conversions réussies',
  extraction_conversions_denied: 'Conversions refusées',
  vip_kills: 'VIP adverses abattus',
  vip_assists: 'Aides sur VIP',
  kills_as_vip: 'Frags en étant VIP',
  time_as_vip_seconds: 'Temps en VIP',
}

const EN_COLUMNS: Record<string, string> = {
  flag_captures: 'Flags captured',
  flag_grabs_net: 'Net grabs',
  flag_capture_assists: 'Capture assists',
  flag_steals: 'Flags stolen',
  flag_returners_killed: 'Returners killed',
  flag_returns: 'Returns',
  flag_secures: 'Flags secured',
  flag_carriers_killed: 'Carriers killed',
  time_as_flag_carrier_seconds: 'Carrier time',
  zone_captures: 'Zones captured',
  zone_secures: 'Zones secured',
  zone_offensive_kills: 'Zone offensive kills',
  zone_defensive_kills: 'Zone defensive kills',
  time_in_zones_seconds: 'Zone time',
  skull_grabs: 'Skulls grabbed',
  skull_carriers_killed: 'Skull carriers killed',
  time_as_skull_carrier_seconds: 'Carrier time',
  power_seeds_deposited: 'Seeds deposited',
  power_seeds_stolen: 'Seeds stolen',
  power_seed_carriers_killed: 'Seed carriers killed',
  time_as_power_seed_carrier_seconds: 'Carrier time',
  successful_extractions: 'Successful extractions',
  extraction_initiations_completed: 'Initiations completed',
  extraction_conversions_completed: 'Conversions completed',
  extraction_conversions_denied: 'Conversions denied',
  vip_kills: 'Enemy VIPs killed',
  vip_assists: 'VIP assists',
  kills_as_vip: 'Kills as VIP',
  time_as_vip_seconds: 'VIP time',
}

export const FORMES_TEXT: Record<Locale, FormesText> = {
  fr: {
    sectionTitle: 'Les formes retenues',
    empty: {
      noFilm:
        'Aucun match de cette sélection n’a de film décodé — ces formes se lisent dans le ' +
        'film, elles n’ont donc rien à montrer ici.',
      noEquipment:
        'Aucun usage d’équipement dans les modes sélectionnés : les matchs sont mesurés, ' +
        'mais personne n’y a posé de mur, de camouflage, de surbouclier ni de grappin.',
      noPads:
        'Aucun socle d’arme spéciale dans les modes sélectionnés — ces modes n’en portent ' +
        'pas, ou aucun n’a été occupé.',
      padsUnnamedOnly:
        'Des socles ont été occupés, mais l’événement natif ne nomme aucun ramasseur : sans ' +
        'nom, une prise ne peut être versée à aucun camp.',
      generic: 'Données manquantes sur cette sélection.',
    },
    blocks: {
      equipment: {
        title: "Usages d'équipements",
        aide:
          "Les usages d'équipement lus dans le film : camouflage, surbouclier, mur de protection, grappin et objets lâchés au sol. Les grenades n'en sont pas et restent hors du bloc. Attention à l'échelle : ce sont de petits volumes, et une part y bouge de vingt points pour un usage de plus.",
      },
      weapons: {
        title: 'Contrôle des armes spéciales',
        aide:
          "Un socle est l'emplacement fixe où une arme de puissance réapparaît ; une prise de socle est un ramassage lu dans l'événement natif du film, daté et nominatif. Le dénominateur n'est pas le nombre de socles mais le nombre de prises NOMMÉES. Les occupations sans ramasseur connu n'entrent donc dans aucun camp.",
      },
      objectives: {
        title: 'Objectifs',
        aide:
          "Les grandeurs de chaque rôle d'objectif viennent de la table du titre, jamais d'un composant. Les durées (temps en zone, temps de portage) sont en secondes : elles se comparent à leur propre parité, jamais aux comptes d'actions.",
      },
    },
    axes: {
      camo: 'Camouflages',
      wall: 'Murs de protection',
      overshield: 'Surboucliers',
      grapple: 'Tractions de grappin',
      dropped: 'Objets lâchés au sol',
    },
    weaponClasses: {
      heavy: 'Armes lourdes',
      precision: 'Armes de précision',
      other: 'Autres socles',
    },
    allWeapons: 'Toutes armes',
    allWeaponsHint: 'toutes prises nommées',
    roles: { take: 'Prendre', defend: 'Défendre', hold: 'Tenir' },
    roleHints: {
      take: 'zones capturées · drapeaux saisis',
      defend: 'zones sécurisées · retours',
      hold: 'temps en zone · temps de portage',
    },
    columns: FR_COLUMNS,
    padsColumns: {
      pads: 'Prises de socle',
      rate: 'Taux de rafle',
      mine: 'Mes prises',
      occupations: 'Occupations du socle',
      rateLegend: 'Prises rapportées aux occupations du socle',
      rateAxis: 'une échelle par colonne — le taux est une part des occupations',
      namedPickupsFmt: (n) => `${n} prises nommées`,
    },
    common: {
      notMeasured: 'non mesuré',
      noMeasure: 'aucune mesure',
      inMyTeam: 'dans mon équipe',
      inLobby: 'dans le lobby',
      myShare: 'Ma part',
      spread: 'Étendue',
      spreadByMatch: 'Étendue match par match',
      parity: 'Parité',
      lineParity: 'Parité de la ligne',
      parityFmt: (pct) => `Parité : ${pct}`,
      aboveFmt: (above, total) => `${above}/${total} au-dessus`,
      pointsFmt: (signed) => `${signed} pts`,
      morePlus: 'Plus que la moyenne du lobby',
      moreThanOpponent: "Plus que l'équipe adverse",
      lessThanLobby: 'Moins que la moyenne du lobby',
      lessThanOpponent: 'Moins',
      less: 'Moins',
      outOfFmt: (value, total) => `${value} sur ${total}`,
      enemyTeam: 'équipe adverse',
      enemyTeamCounted: 'Équipe adverse (comptée, jamais nommée)',
      teamRest: 'Coéquipier hors escouade',
      noFilm: 'Match sans film décodé',
      noMeasureOnAxis: 'Aucune mesure sur cet axe',
      aboveBelowByMatch: 'Au-dessus / en dessous par match',
      unknownWeapon: 'arme non cataloguée',
      occupationsShortFmt: (n) => `${n} occ.`,
      inSquadFmt: (n) => `${n} dans l'escouade`,
      foldMeasuredFmt: (shown, hidden, unmeasured) =>
        `Affichés : les ${frCount(shown)} derniers matchs à film décodé` +
        (hidden > 0 ? ` · ${frCount(hidden)} autres matchs mesurés ne sont pas affichés` : '') +
        (unmeasured > 0
          ? ` · ${frCount(unmeasured)} matchs sans film décodé sont hors de cette forme`
          : '') +
        '.',
      foldListFmt: (shown, hidden) =>
        `Affichés : les ${frCount(shown)} derniers matchs de ce mode` +
        (hidden > 0 ? ` · ${frCount(hidden)} autres ne sont pas affichés` : '') +
        '.',
      matchesShortFmt: (n) => (n > 1 ? `${n} matchs` : `${n} match`),
      gesturesAxis: 'usages — une échelle par colonne',
      gesturesPerMatchAxis: 'usages par match — une échelle par colonne',
      spreadAxis: 'usages par match',
      meanPrefix: 'moy',
      lowestToHighest: 'Du match le plus faible au plus fort',
      meanPerMatch: 'Moyenne par match',
      shareAxis: 'part, en pourcentage du dénominateur de la ligne',
      lobbyShareAxis: 'part du lobby',
      squadShareAxis:
        "part de l'escouade — les coéquipiers hors escouade et l'adversaire ne comptent pas ici",
      gapAxis: 'écart à la parité, en points de pourcentage',
      padShareAxis: 'part des occupations du socle — une échelle par joueur',
      matchesAxis: 'les matchs de la période, dans l’ordre',
      droppedFamilyFmt: (family) => `Lâchés : ${family}`,
      totalFmt: (n) => `${n} au total`,
      valueTipFmt: (row, column, value) => `${row} — ${column} : ${value}`,
      juggleFoldedFmt: (seconds) =>
        `jonglage replié (fenêtre ${seconds} s) : une reprise du même drapeau par le même ` +
        `joueur dans ce délai compte pour une seule prise`,
      matchTipFmt: (match, row, value, gap) => `${match} — ${row} : ${value} (${gap} pts)`,
      gapTipFmt: (row, value, parity, gap) =>
        `${row} : ${value} · parité ${parity} · écart ${gap} points`,
      shareTipFmt: (row, side, value, detail) => `${row} ${side} : ${value} (${detail})`,
      spreadTipFmt: (row, side, from, to) =>
        `${row} ${side} — étendue par match : de ${from} à ${to}`,
      segmentTipFmt: (name, row, value, total) => `${name} — ${row} : ${value} sur ${total}`,
      rangeTipFmt: (row, from, to) => `${row} — du match le plus faible ${from} au plus fort ${to}`,
      meanTipFmt: (row, value) => `${row} — moyenne par match : ${value}`,
    },
  },
  en: {
    sectionTitle: 'The retained forms',
    empty: {
      noFilm:
        'No match in this selection has a decoded film — these forms are read from the film, ' +
        'so they have nothing to show here.',
      noEquipment:
        'No equipment usage in the selected modes: the matches are measured, but nobody ' +
        'deployed a wall, a camouflage, an overshield or a grapple there.',
      noPads:
        'No power weapon pad in the selected modes — either these modes carry none, or none ' +
        'was occupied.',
      padsUnnamedOnly:
        'Pads were occupied, but the native event names no picker: without a name, a pickup ' +
        'belongs to no side.',
      generic: 'Data missing for this selection.',
    },
    blocks: {
      equipment: {
        title: 'Equipment usage',
        aide:
          "Equipment actions read from the film: active camouflage, overshield, drop wall, grappleshot and objects dropped on the ground. Grenades are not equipment and stay out of this block. Mind the scale: these are small volumes, and a share moves twenty points for one extra action.",
      },
      weapons: {
        title: 'Power weapon control',
        aide:
          'A pad is the fixed spot where a power weapon respawns; a pad pickup is a pickup on that spot, read from the film native event, dated and named. The denominator is not the number of pads but the number of NAMED pickups. Occupations without a known picker therefore belong to no side.',
      },
      objectives: {
        title: 'Objectives',
        aide:
          'The measures of each objective role come from the title table, never from a component. Durations (zone time, carrier time) are in seconds: they compare to their own parity, never to action counts.',
      },
    },
    axes: {
      camo: 'Camouflages',
      wall: 'Drop walls',
      overshield: 'Overshields',
      grapple: 'Grapple pulls',
      dropped: 'Objects dropped',
    },
    weaponClasses: {
      heavy: 'Heavy weapons',
      precision: 'Precision weapons',
      other: 'Other pads',
    },
    allWeapons: 'All weapons',
    allWeaponsHint: 'all named pickups',
    roles: { take: 'Take', defend: 'Defend', hold: 'Hold' },
    roleHints: {
      take: 'zones captured · flags grabbed',
      defend: 'zones secured · returns',
      hold: 'zone time · carrier time',
    },
    columns: EN_COLUMNS,
    padsColumns: {
      pads: 'Pad pickups',
      rate: 'Take rate',
      mine: 'My pickups',
      occupations: 'Pad occupations',
      rateLegend: 'Pickups against pad occupations',
      rateAxis: 'one scale per column — the rate is a share of occupations',
      namedPickupsFmt: (n) => `${n} named pickups`,
    },
    common: {
      notMeasured: 'not measured',
      noMeasure: 'no measure',
      inMyTeam: 'in my team',
      inLobby: 'in the lobby',
      myShare: 'My share',
      spread: 'Spread',
      spreadByMatch: 'Spread match by match',
      parity: 'Parity',
      lineParity: 'Parity of the row',
      parityFmt: (pct) => `Parity: ${pct}`,
      aboveFmt: (above, total) => `${above}/${total} above`,
      pointsFmt: (signed) => `${signed} pts`,
      morePlus: 'More than the lobby average',
      moreThanOpponent: 'More than the other team',
      lessThanLobby: 'Less than the lobby average',
      lessThanOpponent: 'Less',
      less: 'Less',
      outOfFmt: (value, total) => `${value} out of ${total}`,
      enemyTeam: 'other team',
      enemyTeamCounted: 'Other team (counted, never named)',
      teamRest: 'Teammate outside the squad',
      noFilm: 'Match without a decoded film',
      noMeasureOnAxis: 'No measure on this axis',
      aboveBelowByMatch: 'Above / below, match by match',
      unknownWeapon: 'uncatalogued weapon',
      occupationsShortFmt: (n) => `${n} occ.`,
      inSquadFmt: (n) => `${n} in the squad`,
      foldMeasuredFmt: (shown, hidden, unmeasured) =>
        `Shown: the last ${enCount(shown)} matches with a decoded film` +
        (hidden > 0 ? ` · ${enCount(hidden)} other measured matches are not shown` : '') +
        (unmeasured > 0
          ? ` · ${enCount(unmeasured)} matches without a decoded film are outside this form`
          : '') +
        '.',
      foldListFmt: (shown, hidden) =>
        `Shown: the last ${enCount(shown)} matches of this mode` +
        (hidden > 0 ? ` · ${enCount(hidden)} others are not shown` : '') +
        '.',
      matchesShortFmt: (n) => (n > 1 ? `${n} matches` : `${n} match`),
      gesturesAxis: 'actions — one scale per column',
      gesturesPerMatchAxis: 'actions per match — one scale per column',
      spreadAxis: 'actions per match',
      meanPrefix: 'avg',
      lowestToHighest: 'From the weakest match to the strongest',
      meanPerMatch: 'Average per match',
      shareAxis: 'share, in percent of the denominator of the row',
      lobbyShareAxis: 'share of the lobby',
      squadShareAxis:
        'share of the squad — teammates outside the squad and the other team are not counted here',
      gapAxis: 'gap to parity, in percentage points',
      padShareAxis: 'share of pad occupations — one scale per player',
      matchesAxis: 'the matches of the period, in order',
      droppedFamilyFmt: (family) => `Dropped: ${family}`,
      totalFmt: (n) => `${n} in total`,
      valueTipFmt: (row, column, value) => `${row} — ${column}: ${value}`,
      juggleFoldedFmt: (seconds) =>
        `juggling folded (${seconds}s window): the same player re-grabbing the same flag ` +
        `within that delay counts as a single grab`,
      matchTipFmt: (match, row, value, gap) => `${match} — ${row}: ${value} (${gap} pts)`,
      gapTipFmt: (row, value, parity, gap) => `${row}: ${value} · parity ${parity} · gap ${gap} points`,
      shareTipFmt: (row, side, value, detail) => `${row} ${side}: ${value} (${detail})`,
      spreadTipFmt: (row, side, from, to) => `${row} ${side} — spread by match: from ${from} to ${to}`,
      segmentTipFmt: (name, row, value, total) => `${name} — ${row}: ${value} out of ${total}`,
      rangeTipFmt: (row, from, to) => `${row} — from the lowest match ${from} to the highest ${to}`,
      meanTipFmt: (row, value) => `${row} — average per match: ${value}`,
    },
  },
}
