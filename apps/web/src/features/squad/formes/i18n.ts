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
  /** Le paragraphe de constat du bloc (gabarit alimenté par les mesures). */
  constat: string
  /** L'encadré de lexique : ce que le décodeur prend en charge, les réserves. */
  lexique: string
}

export interface FormesText {
  sectionTitle: string
  /** Le chapeau de la section : ce que les deux lentilles répondent. */
  intro: string
  /** Les quatre repères du bandeau (ensemble, lobbies, parité, modes). */
  header: {
    scope: string
    scopeMeasuredFmt: (measured: number, total: number) => string
    lobbies: string
    lobbiesPlacesFmt: (players: number, squad: number) => string
    parity: string
    parityHint: string
    modes: string
    modesHint: string
    matchesFmt: (n: number) => string
    familiesFmt: (n: number) => string
  }
  contexts: { solo: string; squad: string }
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
    totalFmt: (n: string) => string
    valueTipFmt: (row: string, column: string, value: string) => string
    matchTipFmt: (match: string, row: string, value: string, gap: string) => string
    gapTipFmt: (row: string, value: string, parity: string, gap: string) => string
    shareTipFmt: (row: string, side: string, value: string, detail: string) => string
    spreadTipFmt: (row: string, side: string, from: string, to: string) => string
    segmentTipFmt: (name: string, row: string, value: string, total: string) => string
    rangeTipFmt: (row: string, from: string, to: string) => string
    meanTipFmt: (row: string, value: string) => string
    notMeasuredTipFmt: (row: string) => string
  }
}

const FR_COLUMNS: Record<string, string> = {
  flag_captures: 'Drapeaux capturés',
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
    intro:
      "Deux lentilles cohabitent, et elles ne répondent pas à la même question. **Ce que j'ai fait** : " +
      "des comptes, match par match. **Où je me situe** : des parts, avec l'équipe d'en face pour " +
      'référence — jamais affichée, toujours comptée. Partout où une part est affichée, ses **deux ' +
      "dénominateurs** le sont aussi : mon équipe, et le lobby. Les grenades sont sorties des usages " +
      "d'équipement : ce n'en sont pas.",
    header: {
      scope: 'Ensemble',
      scopeMeasuredFmt: (measured, total) =>
        `${measured} sur ${total} avec film décodé`,
      lobbies: 'Lobbies observés',
      lobbiesPlacesFmt: (players, squad) =>
        `${players} joueurs mesurés, dont ${squad} de mon escouade`,
      parity: 'Parité',
      parityHint: 'dans mon équipe / dans le lobby',
      modes: 'Modes',
      modesHint: 'familles de mode à objectif',
      matchesFmt: (n) => (n > 1 ? `${n} matchs` : `${n} match`),
      familiesFmt: (n) => (n > 1 ? `${n} familles` : `${n} famille`),
    },
    contexts: {
      solo: 'Contexte Solo — moi dans mon équipe, et dans le lobby',
      squad: 'Contexte Escouade — mon camp contre le leur',
    },
    blocks: {
      equipment: {
        title: "Usages d'équipements",
        constat:
          '**Les grenades sont sorties du bloc** : ce ne sont pas des équipements. Restent le ' +
          'camouflage, le surbouclier, le mur de protection, le grappin et les objets lâchés au sol — ' +
          'les familles que cette session a effectivement mesurées. Attention à l’échelle : ce sont ' +
          'de **petits volumes**. Une part y bouge de vingt points pour un geste de plus, et les ' +
          'formes ci-dessous le disent toutes — étendue, compte au-dessus de la parité, numérateur ' +
          'en infobulle.',
        lexique:
          "**Ce que le décodeur prend en charge, au-delà de ce qui s'affiche ici.** Six familles " +
          'déployables sont nommées — mur de protection, capteur de menaces, faille du translocateur, ' +
          'écran occultant, traqueur de menaces, champ de réparation — plus le grappin, le camouflage ' +
          'et le surbouclier (avec durée et frags), les objets lâchés au sol et les socles de bonus ' +
          'vidés. **Les colonnes sont pilotées par la donnée** : une famille absente de l’écran est ' +
          "une famille que personne n'a utilisée sur ces matchs. **Deux exceptions** : le " +
          '**répulseur** — le film montre l’OBJET au sol mais aucun de ses neuf canaux ne date son ' +
          'ACTIVATION ; et le **propulseur**, dont l’usage est mesuré mais dure une demi-seconde — ' +
          'il se voit sur la carte du rejeu, pas dans un compteur.',
      },
      weapons: {
        title: 'Contrôle des armes spéciales',
        constat: '',
        lexique:
          '**Un socle**, c’est l’emplacement fixe d’une carte où une arme de puissance réapparaît à ' +
          'intervalle régulier. **Une prise de socle** est un ramassage sur cet emplacement, lu dans ' +
          'l’événement natif du film : daté à la milliseconde, il porte son ramasseur. C’est le ' +
          'vocabulaire déjà employé par le bloc de la vue match (colonne « Prises de socle »). **Le ' +
          'dénominateur n’est pas le nombre de socles** : c’est le nombre de prises NOMMÉES, quand ' +
          'les occupations sans ramasseur nommé n’entrent dans aucun camp. La part se lit donc « sur ' +
          'ce que l’on sait », et cette réserve reste à l’écran.',
      },
      objectives: {
        title: 'Objectifs',
        constat: '',
        lexique:
          '**La table rôle → grandeurs est un savoir du titre**, déclarée en donnée comme les rôles ' +
          'd’objectif du rejeu, jamais en dur dans un composant. C’est le coût de cette forme, et il ' +
          'est réel. Les durées (temps en zone, temps de portage) sont en secondes : elles se ' +
          'comparent à leur propre parité, jamais aux comptes d’actions.',
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
      matchesShortFmt: (n) => (n > 1 ? `${n} matchs` : `${n} match`),
      gesturesAxis: 'gestes — une échelle par colonne',
      gesturesPerMatchAxis: 'gestes par match — une échelle par colonne',
      spreadAxis: 'gestes par match',
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
      totalFmt: (n) => `${n} au total`,
      valueTipFmt: (row, column, value) => `${row} — ${column} : ${value}`,
      matchTipFmt: (match, row, value, gap) => `${match} — ${row} : ${value} (${gap} pts)`,
      gapTipFmt: (row, value, parity, gap) =>
        `${row} : ${value} · parité ${parity} · écart ${gap} points`,
      shareTipFmt: (row, side, value, detail) => `${row} ${side} : ${value} (${detail})`,
      spreadTipFmt: (row, side, from, to) =>
        `${row} ${side} — étendue par match : de ${from} à ${to}`,
      segmentTipFmt: (name, row, value, total) => `${name} — ${row} : ${value} sur ${total}`,
      rangeTipFmt: (row, from, to) => `${row} — du match le plus faible ${from} au plus fort ${to}`,
      meanTipFmt: (row, value) => `${row} — moyenne par match : ${value}`,
      notMeasuredTipFmt: (row) => `${row} — non mesuré (pas de film décodé)`,
    },
  },
  en: {
    sectionTitle: 'The retained forms',
    intro:
      'Two lenses live side by side, and they do not answer the same question. **What I did**: ' +
      'counts, match by match. **Where I stand**: shares, with the other team as the reference — ' +
      'never displayed, always counted. Wherever a share is shown, **both denominators** are shown ' +
      'too: my team, and the lobby. Grenades are out of equipment usage: they are not equipment.',
    header: {
      scope: 'Scope',
      scopeMeasuredFmt: (measured, total) => `${measured} of ${total} with a decoded film`,
      lobbies: 'Lobbies observed',
      lobbiesPlacesFmt: (players, squad) =>
        `${players} measured players, ${squad} of them in my squad`,
      parity: 'Parity',
      parityHint: 'in my team / in the lobby',
      modes: 'Modes',
      modesHint: 'objective mode families',
      matchesFmt: (n) => (n > 1 ? `${n} matches` : `${n} match`),
      familiesFmt: (n) => (n > 1 ? `${n} families` : `${n} family`),
    },
    contexts: {
      solo: 'Solo context — me in my team, and in the lobby',
      squad: 'Squad context — my side against theirs',
    },
    blocks: {
      equipment: {
        title: 'Equipment usage',
        constat:
          '**Grenades are out of this block**: they are not equipment. What remains is active ' +
          'camouflage, overshield, the drop wall, the grappleshot and objects dropped on the ' +
          'ground — the families this period actually measured. Mind the scale: these are **small ' +
          'volumes**. A share moves twenty points for one extra action, and every form below says ' +
          'so — spread, count above parity, numerator in the tooltip.',
        lexique:
          '**What the decoder covers, beyond what is shown here.** Six deployable families are ' +
          'named — drop wall, threat sensor, repulsor beacon, shroud screen, threat seeker, repair ' +
          'field — plus the grappleshot, camouflage and overshield (with duration and kills), ' +
          'objects dropped on the ground and emptied power-up pads. **Columns are driven by the ' +
          'data**: a family missing from the screen is a family nobody used in these matches. **Two ' +
          'exceptions**: the **repulsor** — the film shows the OBJECT on the ground but none of its ' +
          'nine channels dates its ACTIVATION; and the **thruster**, whose usage is measured but ' +
          'lasts half a second — it shows on the replay map, not in a counter.',
      },
      weapons: {
        title: 'Power weapon control',
        constat: '',
        lexique:
          '**A pad** is the fixed spot on a map where a power weapon respawns at a regular ' +
          'interval. **A pad pickup** is a pickup on that spot, read from the film native event: ' +
          'dated to the millisecond, it carries its picker. This is the vocabulary already used by ' +
          'the match view block (“Pad pickups” column). **The denominator is not the number of ' +
          'pads**: it is the number of NAMED pickups, while occupations without a named picker ' +
          'belong to no side. The share therefore reads “out of what we know”, and that caveat ' +
          'stays on screen.',
      },
      objectives: {
        title: 'Objectives',
        constat: '',
        lexique:
          '**The role → measures table is title knowledge**, declared as data like the replay ' +
          'objective roles, never hardcoded in a component. That is the cost of this form, and it ' +
          'is real. Durations (zone time, carrier time) are in seconds: they compare to their own ' +
          'parity, never to action counts.',
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
      totalFmt: (n) => `${n} in total`,
      valueTipFmt: (row, column, value) => `${row} — ${column}: ${value}`,
      matchTipFmt: (match, row, value, gap) => `${match} — ${row}: ${value} (${gap} pts)`,
      gapTipFmt: (row, value, parity, gap) => `${row}: ${value} · parity ${parity} · gap ${gap} points`,
      shareTipFmt: (row, side, value, detail) => `${row} ${side}: ${value} (${detail})`,
      spreadTipFmt: (row, side, from, to) => `${row} ${side} — spread by match: from ${from} to ${to}`,
      segmentTipFmt: (name, row, value, total) => `${name} — ${row}: ${value} out of ${total}`,
      rangeTipFmt: (row, from, to) => `${row} — from the lowest match ${from} to the highest ${to}`,
      meanTipFmt: (row, value) => `${row} — average per match: ${value}`,
      notMeasuredTipFmt: (row) => `${row} — not measured (no decoded film)`,
    },
  },
}
