/**
 * empriseStrings.ts — les textes de l'onglet « Emprise » de l'Escouade (lot L5 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) : titres (§3 du plan), aides ⓘ (trois phrases au
 * plus) et libellés de la maquette `.ai/V7.5/MAQUETTE_ONGLET_TACTIQUE_ESCOUADE_2026-09-26.html`.
 * Fichier à part (précédent : `objectif/objectifStrings.ts`) : `i18n.ts` de la feature dépasse
 * déjà le seuil de taille. Parité FR / EN garantie par le typage `Record<Locale, …>`.
 */
import type { Locale } from '@/lib/i18n/locale'
import { buildVehicleText, type VehicleText } from './vehicleStrings'

/**
 * Les noms d'une ressource, selon l'endroit où elle s'écrit : `label` (« Bonus » : légendes,
 * fiches), titres de ligne qui se lisent seuls (« Prises de bonus », « Frags avec arme
 * spéciale » — jamais le nom suivi d'un sous-libellé qui le complète), mot du pied de fiche
 * (« armes spéciales »).
 */
export interface ResourceText {
  label: string
  /** Piste du bilan : titre (« Prises de bonus ») et précision facultative (« camouflage, surbouclier »). */
  pisteTitle: string
  pisteSub?: string
  /** Synthèse de la grille match par match (« Prises de bonus »). */
  gridTitle: string
  footer: string
  /** Cases vides : synthèse (« Aucun bonus sur cette carte. »), objet (« : pas sur cette carte. », après son nom). */
  absent: string
  itemAbsent: string
  /** Ligne de « Frags par ressource » (« Frags avec arme spéciale »). */
  productionTitle: string
  /** Sous-libellé du « Rendement… » (« frags par prise »). */
  yieldSub: string
}

/** Une exposition (barre fine) : son nom (« temps d’effet ») et le format de sa valeur (« 2 min 39 »). */
export interface ExposureText { name: string; fmt: (v: number) => string }

type BaseEmpriseText = Omit<EmpriseText, 'vehicles'> // avant l'ajout des véhicules (vehicleStrings.ts)

export interface EmpriseText {
  vehicles: VehicleText // les mots propres aux véhicules ; leur entrée de ressource est posée par withVehicles
  sections: { bilan: string; roles: string; carte: string; prendre: string; habitude: string }
  resources: Record<string, ResourceText>
  ourSide: string
  opponent: string
  parity: string
  /** « 60 % », « 44,2 % » (une décimale au plus) ; `pctIntFmt` : « 17 % » (entier). */
  pctFmt: (v: number) => string
  pctIntFmt: (v: number) => string
  control: {
    title: string
    info: string
    ariaLabel: string
    segmentTip: (side: string, resource: string, sub: string | undefined, value: number, total: number, pct: string) => string
  }
  fil: {
    title: string
    info: string
    winLoss: string
    dominance: string
    pointTip: (v: { match: string; outcome: string | null; resource: string; us: number; them: number; pct: string; cumUs: number; cumTotal: number; cumPct: string }) => string
    endTip: (resource: string, cumUs: number, cumTotal: number, pct: string) => string
    bandTip: (match: string, result: string | null, dominance: string | null) => string
  }
  sheets: {
    title: string
    info: string
    dominant: string
    legendTaken: string
    legendLost: string
    lossesTitle: string
    lossesFmt: (lost: number, taken: number) => string
    lineTip: (player: string, item: string, n: number, camp: number, lost: string | null) => string
    /** « 1 perdue : gardé sans l’activer ». */
    lostFmt: (kept: number, dropped: number) => string | null
  }
  production: {
    title: string
    info: string
    ariaLabel: string
    thinLegend: string
    exposure: Record<string, ExposureText>
    segmentTip: (side: string, title: string, value: number, total: number, pct: string) => string
    thinTip: (side: string, name: string, value: string, pct: string) => string
    exposureLine: (name: string, value: string, pct: string) => string
  }
  yield: {
    title: string
    info: string
    ariaLabel: string
    more: string
    less: string
    /** Graduations de l'axe (−50 % à +50 %), écart signé (« +14 % »), rendements bruts (« 3,0 contre 2,7 »). */
    axis: [string, string, string, string, string]
    gapFmt: (gap: number) => string
    rawFmt: (us: number, them: number) => string
    tip: (resource: string, sub: string, us: number, them: number, gap: string) => string
  }
  habit: {
    title: string
    info: string
    tonight: string
    pointTip: (resource: string, evening: string, value: string, median: string | null) => string
    eveningOf: (date: string) => string
    medianTip: (resource: string, value: string) => string
    /** Légende et infobulle d'une soirée hors comparaison (aucun mode de ce soir filmé). */
    notComparable: string
    notComparableTip: (families: string) => string
    /** Le bloc placeholder : aucune soirée, ce soir compris, n'a de part. */
    empty: { title: string; description: string }
  }
  grid: {
    title: string
    info: string
    more: string
    less: string
    nothing: string
    noFilm: string
    noFilmCell: string
    noFilmTip: string
    noTeamCell: string
    noTeamTip: string
    untieredCell: string
    untieredTip: string
    unestablishedTip: string
    killsName: string
    killsAbsent: string
    racks: string
    /**
     * Le nombre de TYPES d'armes de râtelier listés sous la ligne repliable (« 16 types d'armes »),
     * jamais un nombre de prises. Source unique : la Vue match le reprend.
     */
    rackTypes: (n: number) => string
    matchHead: (time: string, map: string, mode: string, result: string | null) => string
    cellTip: (name: string, us: number, them: number, pct: string) => string
    whoFmt: (list: string) => string
    restLower: string
    padsFmt: (emptied: number, attributed: number) => string
    dominanceTip: (label: string) => string
  }
}

const frPct = (v: number): string => `${(Math.round(v * 10) / 10).toLocaleString('fr-FR', { maximumFractionDigits: 1 })} %`
const enPct = (v: number): string => `${(Math.round(v * 10) / 10).toLocaleString('en-GB', { maximumFractionDigits: 1 })}%`

/** « 2 min 39 » — un temps d'effet (millisecondes), à la seconde ; « 45 s » sous la minute. */
function duration(ms: number): string {
  const sec = Math.round(ms / 1000)
  const m = Math.floor(sec / 60)
  return m === 0 ? `${sec} s` : `${m} min ${String(sec % 60).padStart(2, '0')}`
}

/** Une décimale, séparateur de la langue (« 3,0 », « 3.0 »). */
function dec1(v: number, locale: string): string {
  return v.toLocaleString(locale, { minimumFractionDigits: 1, maximumFractionDigits: 1 })
}

/** « +14 % » / « −8 % » / « 0 % » : l'écart relatif en points entiers, signe typographique. */
function signedPct(gap: number, sep: string): string {
  const n = Math.round(Math.abs(gap) * 100)
  if (n === 0) return `0${sep}%`
  return `${gap > 0 ? '+' : '−'}${n}${sep}%`
}

const FR: BaseEmpriseText = {
  sections: {
    bilan: 'Ressources',
    roles: 'Prises par joueur',
    carte: 'Par match',
    // Pas « Rendement » seul : c'est le libellé du champ `offensive_conversion` (fields.toml), une
    // autre grandeur — lint-no-hardcoded-fields.
    prendre: 'Rendement des ressources',
    habitude: 'Soirées précédentes',
  },
  resources: {
    powerup: {
      label: 'Bonus',
      pisteTitle: 'Prises de bonus',
      pisteSub: 'camouflage, surbouclier',
      gridTitle: 'Prises de bonus',
      footer: 'bonus',
      absent: 'Aucun bonus sur cette carte.',
      itemAbsent: ' : pas sur cette carte.',
      productionTitle: 'Frags pendant l’effet d’un bonus',
      yieldSub: 'frags par minute d’effet',
    },
    power_weapon: {
      label: 'Armes spéciales',
      pisteTitle: 'Prises d’armes spéciales',
      gridTitle: 'Prises d’armes spéciales',
      footer: 'armes spéciales',
      absent: 'Aucune arme spéciale prise.',
      itemAbsent: ' : aucune prise sur cette carte.',
      productionTitle: 'Frags avec arme spéciale',
      yieldSub: 'frags par prise',
    },
    rack: {
      label: 'Armes de râtelier',
      pisteTitle: 'Prises d’armes de râtelier',
      gridTitle: 'Prises d’armes de râtelier',
      footer: 'armes de râtelier',
      absent: 'Aucune arme de râtelier prise.',
      itemAbsent: ' : aucune prise sur cette carte.',
      productionTitle: 'Frags avec arme de râtelier',
      yieldSub: 'frags par prise',
    },
  },
  ourSide: 'Équipe',
  opponent: 'Adversaire',
  parity: '50 % : autant que l’adversaire',
  pctFmt: frPct,
  pctIntFmt: (v) => `${Math.round(v)} %`,
  control: {
    title: 'Contrôle des ressources',
    info:
      'Prises de chaque ressource par l’équipe et par l’adversaire, en comptes, sur les matchs filmés de la ' +
      'soirée ; trait orange : 50 %. Les bonus sans ramasseur connu ne comptent dans aucune équipe.',
    ariaLabel: 'Part de l’équipe dans les prises de chaque ressource, face à l’adversaire',
    segmentTip: (side, resource, sub, value, tot, pct) => `${side} · ${resource}${sub ? ` (${sub})` : ''}\n${value} sur ${tot} (${pct})`,
  },
  fil: {
    title: 'Contrôle des ressources, cumul par match',
    info:
      'Part de l’équipe dans les prises de chaque ressource, cumulée match après match sur la soirée. Points : ' +
      'part de chaque match, taille selon le volume ; un match sans la ressource n’a pas de point.',
    winLoss: 'Victoire, défaite',
    dominance: 'Drapeau de dominance',
    pointTip: (v) =>
      `${v.match}${v.outcome ? ` (${v.outcome})` : ''}\n${v.resource} : équipe ${v.us}, adversaire ${v.them} (${v.pct})\n` +
      `Cumul : ${v.cumUs} sur ${v.cumTotal} (${v.cumPct})`,
    endTip: (resource, cumUs, cumTotal, pct) => `${resource}\nCumul de la soirée : ${cumUs} sur ${cumTotal} (${pct})`,
    bandTip: (match, result, dominance) => `${match}${result ? `\n${result}` : ''}${dominance ? ` · ${dominance}` : ''}`,
  },
  sheets: {
    title: 'Prises par joueur',
    info:
      'Prises de chaque bonus, arme spéciale et véhicule par joueur de l’équipe sur la soirée, une pastille par ' +
      'prise. Pastille vide : bonus perdu (gardé sans être activé, ou lâché à la mort).',
    dominant: 'Ressource dominante',
    legendTaken: 'Prise',
    legendLost: 'Bonus pris puis perdu',
    lossesTitle: 'Bonus perdus',
    lossesFmt: (lost, taken) => `${lost} sur ${taken}`,
    lineTip: (player, item, n, camp, lost) =>
      `${player} · ${item}\n${n} des ${camp} prises de l’équipe${lost ? `\n${lost}` : ''}`,
    lostFmt: (kept, dropped) => {
      const n = kept + dropped
      if (n <= 0) return null
      const lead = `${n} perdue${n > 1 ? 's' : ''} : `
      if (dropped === 0) return `${lead}${kept > 1 ? 'gardés' : 'gardé'} sans l’activer`
      if (kept === 0) return `${lead}${dropped > 1 ? 'lâchés' : 'lâché'} en mourant`
      return `${lead}${kept} gardé${kept > 1 ? 's' : ''} sans l’activer, ${dropped} lâché${dropped > 1 ? 's' : ''} en mourant`
    },
  },
  production: {
    title: 'Frags par ressource',
    info:
      'Barre épaisse : part de l’équipe dans les frags obtenus avec chaque ressource ; barre fine : part de ' +
      'l’équipe dans l’exposition (temps d’effet d’un bonus, prises d’une arme spéciale, temps à bord d’un ' +
      'véhicule). Périmètre : les matchs de la soirée où l’exposition est mesurée.',
    ariaLabel: 'Part de l’équipe dans les frags obtenus avec chaque ressource, et dans l’exposition',
    thinLegend: 'Barre fine : temps d’effet, prises ou temps à bord',
    exposure: {
      effect_ms: { name: 'temps d’effet', fmt: duration },
      pickups: { name: 'prises sur les socles', fmt: (v) => `${v} prise${v > 1 ? 's' : ''}` },
    },
    segmentTip: (side, title, value, tot, pct) => `${side} · ${title}\n${value} sur ${tot} (${pct})`,
    thinTip: (side, name, value, pct) => `${side} · ${name}\n${value} (${pct})`,
    exposureLine: (name, value, pct) => `${name} : ${value} · ${pct}`,
  },
  yield: {
    title: 'Rendement par ressource',
    info:
      'Écart relatif entre les frags de l’équipe et ceux de l’adversaire pour la même exposition : par minute ' +
      'd’effet d’un bonus, par prise d’arme spéciale, par minute à bord d’un véhicule. Zéro : autant ; ' +
      'rendements bruts sous la valeur.',
    ariaLabel: 'Rendement de l’équipe face à celui de l’adversaire, par ressource',
    more: 'Équipe plus productive',
    less: 'Moins',
    axis: ['−50 %', '−25', 'autant', '+25', '+50 %'],
    gapFmt: (gap) => signedPct(gap, ' '),
    rawFmt: (us, them) => `${dec1(us, 'fr-FR')} contre ${dec1(them, 'fr-FR')}`,
    tip: (resource, sub, us, them, gap) => `${resource} · ${sub}\nÉquipe ${dec1(us, 'fr-FR')}, adversaire ${dec1(them, 'fr-FR')} : ${gap}`,
  },
  habit: {
    title: 'Contrôle des ressources, par soirée',
    info:
      'Part de l’équipe dans les prises de chaque ressource, par soirée de la composition, la plus récente à ' +
      'droite. Point gris : soirée sans aucun des modes de ce soir, hors médiane ; pointillé fin : médiane ' +
      'des soirées précédentes jouées dans les modes de ce soir.',
    tonight: 'ce soir',
    pointTip: (resource, evening, value, med) =>
      `${resource}\n${evening} : ${value}${med ? `\nMédiane des soirées précédentes : ${med}` : ''}`,
    eveningOf: (date) => `Soirée du ${date}`,
    medianTip: (resource, value) => `${resource}\nMédiane des soirées précédentes : ${value}`,
    notComparable: 'Autres modes que ce soir',
    notComparableTip: (families) => `Autres modes que ce soir${families ? ` (${families})` : ''} : hors médiane`,
    empty: { title: 'Aucune prise mesurée', description: 'Aucune soirée de la composition n’a de prise lue au film.' },
  },
  grid: {
    title: 'Contrôle des ressources, par match',
    info:
      'Une colonne par match de la soirée, avec son résultat ; couleur : écart entre les prises de l’équipe et ' +
      'celles de l’adversaire, saturée à trente points.',
    more: 'Plus que l’adversaire',
    less: 'Moins',
    nothing: 'Rien à prendre',
    noFilm: 'Sans film',
    noFilmCell: 'sans film',
    noFilmTip: 'Film non décodé : rien à lire pour cette ligne.',
    noTeamCell: 'équipe inconnue',
    noTeamTip: 'Équipe inconnue sur ce match (chacun pour soi, ou équipe absente de la feuille de match) : rien ne se partage entre les deux équipes.',
    untieredCell: 'non classé',
    untieredTip: 'Niveaux de socle non mesurés sur ce match : armes spéciales et armes de râtelier ne se séparent pas.',
    unestablishedTip: 'Carte absente de la référence des socles : armes spéciales et armes de râtelier ne se séparent pas.',
    killsName: 'Frags avec arme spéciale',
    killsAbsent: 'Aucun frag à l’arme spéciale.',
    racks: 'Armes de râtelier',
    rackTypes: (n) => `${n} ${n > 1 ? 'types d’armes' : 'type d’arme'}`,
    matchHead: (time, map, mode, result) =>
      `${[time, map].filter(Boolean).join(' · ')}${mode || result ? ` (${[mode, result].filter(Boolean).join(', ')})` : ''}`,
    cellTip: (name, us, them, pct) => `${name} : équipe ${us}, adversaire ${them} (${pct})`,
    whoFmt: (list) => `Équipe : ${list}`,
    restLower: 'reste de l’équipe',
    padsFmt: (emptied, attributed) => `${emptied} socles vidés, ${attributed} prises attribuées`,
    dominanceTip: (label) => `${label}\nDrapeau de dominance du match`,
  },
}

const EN: BaseEmpriseText = {
  sections: {
    bilan: 'Resources',
    roles: 'Pickups by player',
    carte: 'By match',
    prendre: 'Resource efficiency',
    habitude: 'Previous sessions',
  },
  resources: {
    powerup: {
      label: 'Power-ups',
      pisteTitle: 'Power-up pickups',
      pisteSub: 'camo, overshield',
      gridTitle: 'Power-up pickups',
      footer: 'power-ups',
      absent: 'No power-up on this map.',
      itemAbsent: ': not on this map.',
      productionTitle: 'Kills during a power-up effect',
      yieldSub: 'kills per minute of effect',
    },
    power_weapon: {
      label: 'Power weapons',
      pisteTitle: 'Power weapon pickups',
      gridTitle: 'Power weapon pickups',
      footer: 'power weapons',
      absent: 'No power weapon picked up.',
      itemAbsent: ': not picked up on this map.',
      productionTitle: 'Kills with power weapons',
      yieldSub: 'kills per pickup',
    },
    rack: {
      label: 'Rack weapons',
      pisteTitle: 'Rack weapon pickups',
      gridTitle: 'Rack weapon pickups',
      footer: 'rack weapons',
      absent: 'No rack weapon picked up.',
      itemAbsent: ': not picked up on this map.',
      productionTitle: 'Kills with rack weapons',
      yieldSub: 'kills per pickup',
    },
  },
  ourSide: 'Team',
  opponent: 'Opponent',
  parity: '50%: as much as the opponent',
  pctFmt: enPct,
  pctIntFmt: (v) => `${Math.round(v)}%`,
  control: {
    title: 'Resource control',
    info:
      'Pickups of each resource by the team and by the opponent, in counts, over the session’s filmed ' +
      'matches; orange line: 50%. Power-ups with no known picker count for neither team.',
    ariaLabel: 'The team’s share of each resource’s pickups, against the opponent',
    segmentTip: (side, resource, sub, value, tot, pct) => `${side} · ${resource}${sub ? ` (${sub})` : ''}\n${value} of ${tot} (${pct})`,
  },
  fil: {
    title: 'Resource control, cumulative by match',
    info:
      'The team’s share of each resource’s pickups, cumulated match after match over the session. Dots: ' +
      'each match’s share, sized by volume; a match without the resource has no dot.',
    winLoss: 'Win, loss',
    dominance: 'Dominance flag',
    pointTip: (v) =>
      `${v.match}${v.outcome ? ` (${v.outcome})` : ''}\n${v.resource}: team ${v.us}, opponent ${v.them} (${v.pct})\n` +
      `Cumulated: ${v.cumUs} of ${v.cumTotal} (${v.cumPct})`,
    endTip: (resource, cumUs, cumTotal, pct) => `${resource}\nSession total: ${cumUs} of ${cumTotal} (${pct})`,
    bandTip: (match, result, dominance) => `${match}${result ? `\n${result}` : ''}${dominance ? ` · ${dominance}` : ''}`,
  },
  sheets: {
    title: 'Pickups by player',
    info:
      'Pickups of each power-up, power weapon and vehicle by each player of the team over the session, one ' +
      'dot per pickup. Hollow dot: a lost power-up (held without being activated, or dropped on death).',
    dominant: 'Main resource',
    legendTaken: 'Pickup',
    legendLost: 'Power-up picked up, then lost',
    lossesTitle: 'Lost power-ups',
    lossesFmt: (lost, taken) => `${lost} of ${taken}`,
    lineTip: (player, item, n, camp, lost) =>
      `${player} · ${item}\n${n} of the team’s ${camp} pickups${lost ? `\n${lost}` : ''}`,
    lostFmt: (kept, dropped) => {
      const n = kept + dropped
      if (n <= 0) return null
      if (dropped === 0) return `${n} lost: held without activating`
      if (kept === 0) return `${n} lost: dropped on death`
      return `${n} lost: ${kept} held without activating, ${dropped} dropped on death`
    },
  },
  production: {
    title: 'Kills by resource',
    info:
      'Thick bar: the team’s share of the kills made with each resource; thin bar: the team’s share of the ' +
      'exposure (effect time for a power-up, pickups for a power weapon, time aboard a vehicle). Scope: the ' +
      'session’s matches where the exposure is measured.',
    ariaLabel: 'The team’s share of the kills made with each resource, and of the exposure',
    thinLegend: 'Thin bar: effect time, pickups or time aboard',
    exposure: {
      effect_ms: { name: 'effect time', fmt: duration },
      pickups: { name: 'pickups from the pads', fmt: (v) => `${v} pickup${v > 1 ? 's' : ''}` },
    },
    segmentTip: (side, title, value, tot, pct) => `${side} · ${title}\n${value} of ${tot} (${pct})`,
    thinTip: (side, name, value, pct) => `${side} · ${name}\n${value} (${pct})`,
    exposureLine: (name, value, pct) => `${name}: ${value} · ${pct}`,
  },
  yield: {
    title: 'Efficiency by resource',
    info:
      'Relative gap between the team’s kills and the opponent’s for the same exposure: per minute of ' +
      'power-up effect, per power weapon pickup, per minute aboard a vehicle. Zero: even; raw rates next ' +
      'to the value.',
    ariaLabel: 'The team’s efficiency against the opponent’s, per resource',
    more: 'Team more productive',
    less: 'Less',
    axis: ['−50%', '−25', 'even', '+25', '+50%'],
    gapFmt: (gap) => signedPct(gap, ''),
    rawFmt: (us, them) => `${dec1(us, 'en-GB')} vs ${dec1(them, 'en-GB')}`,
    tip: (resource, sub, us, them, gap) => `${resource} · ${sub}\nTeam ${dec1(us, 'en-GB')}, opponent ${dec1(them, 'en-GB')}: ${gap}`,
  },
  habit: {
    title: 'Resource control, by session',
    info:
      'The team’s share of each resource’s pickups, per session of the line-up, the latest on the right. ' +
      'Grey dot: a session with none of tonight’s modes, left out of the median; thin dotted line: median of ' +
      'the previous sessions played in tonight’s modes.',
    tonight: 'tonight',
    pointTip: (resource, evening, value, med) =>
      `${resource}\n${evening}: ${value}${med ? `\nMedian of the previous sessions: ${med}` : ''}`,
    eveningOf: (date) => `Session of ${date}`,
    medianTip: (resource, value) => `${resource}\nMedian of the previous sessions: ${value}`,
    notComparable: 'Other modes than tonight',
    notComparableTip: (families) => `Other modes than tonight${families ? ` (${families})` : ''}: left out of the median`,
    empty: { title: 'No pickup measured', description: 'No session of the line-up has pickups read from the film.' },
  },
  grid: {
    title: 'Resource control, by match',
    info:
      'One column per match of the session, with its result; colour: gap between the team’s pickups and ' +
      'the opponent’s, saturated at thirty points.',
    more: 'More than the opponent',
    less: 'Less',
    nothing: 'Nothing to pick up',
    noFilm: 'No film',
    noFilmCell: 'no film',
    noFilmTip: 'Film not decoded: nothing to read for this row.',
    noTeamCell: 'team unknown',
    noTeamTip: 'Team unknown in this match (free-for-all, or team missing from the match sheet): nothing splits between the two teams.',
    untieredCell: 'unsorted',
    untieredTip: 'Pad levels not measured for this match: power weapons and rack weapons can’t be told apart.',
    unestablishedTip: 'Map missing from the pad reference: power weapons and rack weapons can’t be told apart.',
    killsName: 'Kills with power weapons',
    killsAbsent: 'No power weapon kill.',
    racks: 'Rack weapons',
    rackTypes: (n) => `${n} ${n > 1 ? 'weapon types' : 'weapon type'}`,
    matchHead: (time, map, mode, result) =>
      `${[time, map].filter(Boolean).join(' · ')}${mode || result ? ` (${[mode, result].filter(Boolean).join(', ')})` : ''}`,
    cellTip: (name, us, them, pct) => `${name}: team ${us}, opponent ${them} (${pct})`,
    whoFmt: (list) => `Team: ${list}`,
    restLower: 'rest of the team',
    padsFmt: (emptied, attributed) => `${emptied} pads emptied, ${attributed} attributed pickups`,
    dominanceTip: (label) => `${label}\nThe match’s dominance flag`,
  },
}

const VT = buildVehicleText(duration)
const withVehicles = (b: BaseEmpriseText, v: VehicleText): EmpriseText => ({
  ...b, vehicles: v,
  resources: { ...b.resources, vehicle: v.resource },
  production: { ...b.production, exposure: { ...b.production.exposure, aboard_ms: v.aboard } },
})
export const EMPRISE_TEXT: Record<Locale, EmpriseText> = { fr: withVehicles(FR, VT.fr), en: withVehicles(EN, VT.en) }
