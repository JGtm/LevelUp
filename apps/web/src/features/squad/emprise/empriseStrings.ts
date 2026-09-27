/**
 * empriseStrings.ts — les textes de l'onglet « Emprise » de l'Escouade (lot L5 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) : titres (§3 du plan), aides ⓘ (trois phrases au
 * plus) et libellés de la maquette `.ai/V7.5/MAQUETTE_ONGLET_TACTIQUE_ESCOUADE_2026-09-26.html`.
 * Fichier à part (précédent : `objectif/objectifStrings.ts`) : `i18n.ts` de la feature dépasse
 * déjà le seuil de taille. Parité FR / EN garantie par le typage `Record<Locale, …>`.
 */
import type { OutcomeValue } from '@/components/charts/outcomeSequence'
import type { Locale } from '@/lib/i18n/locale'

/**
 * Les noms d'une ressource, selon l'endroit où elle s'écrit : `label` (« Bonus »), sous-libellé
 * de la piste du bilan (« prises · camouflage, surbouclier ») et de la synthèse de la grille
 * (« prises »), mot du pied de fiche (« armes spéciales »).
 */
export interface ResourceText {
  label: string
  pisteSub: string
  gridSub: string
  footer: string
  /** Cases vides : synthèse (« Aucun bonus sur cette carte. »), objet (« : pas sur cette carte. », après son nom). */
  absent: string
  itemAbsent: string
  /** Sous-libellés de « Frags obtenus avec… » (« frags pendant l’effet ») et du « Rendement… » (« frags par prise »). */
  productionSub: string
  yieldSub: string
}

/** Une exposition (barre fine) : son nom (« temps d’effet ») et le format de sa valeur (« 2 min 39 »). */
export interface ExposureText { name: string; fmt: (v: number) => string }

export interface EmpriseText {
  sections: { bilan: string; roles: string; carte: string; prendre: string; habitude: string }
  resources: Record<string, ResourceText>
  ourSide: string
  opponent: string
  parity: string
  /** « 60 % », « 44,2 % » (une décimale au plus) ; `pctIntFmt` : « 17 % » (entier). */
  pctFmt: (v: number) => string
  pctIntFmt: (v: number) => string
  outcome: Record<OutcomeValue, string>
  outcomeLower: Record<OutcomeValue, string>
  control: {
    title: string
    info: string
    ariaLabel: string
    segmentTip: (side: string, resource: string, sub: string, value: number, total: number, pct: string) => string
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
    rest: string
    legendTaken: string
    legendLost: string
    legendRest: string
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
    segmentTip: (side: string, sub: string, value: number, total: number, pct: string) => string
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
    noHistory: (list: string) => { lead: string; rest: string }
    shareItem: (resource: string, pct: string) => string
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
    killsSub: string
    killsName: string
    killsAbsent: string
    racks: string
    racksCount: (n: number) => string
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

const FR: EmpriseText = {
  sections: {
    bilan: 'Bilan de la soirée',
    roles: 'Rôles dans l’escouade',
    carte: 'Carte par carte',
    prendre: 'Prendre, et s’en servir',
    habitude: 'Par rapport à d’habitude',
  },
  resources: {
    powerup: {
      label: 'Bonus',
      pisteSub: 'prises · camouflage, surbouclier',
      gridSub: 'prises',
      footer: 'bonus',
      absent: 'Aucun bonus sur cette carte.',
      itemAbsent: ' : pas sur cette carte.',
      productionSub: 'frags pendant l’effet',
      yieldSub: 'frags par minute d’effet',
    },
    power_weapon: {
      label: 'Armes spéciales',
      pisteSub: 'prises sur les socles',
      gridSub: 'prises sur les socles',
      footer: 'armes spéciales',
      absent: 'Aucune arme spéciale prise.',
      itemAbsent: ' : aucune prise sur cette carte.',
      productionSub: 'frags obtenus avec',
      yieldSub: 'frags par prise',
    },
    rack: {
      label: 'Armes de râtelier',
      pisteSub: 'prises',
      gridSub: 'prises',
      footer: 'armes de râtelier',
      absent: 'Aucune arme de râtelier prise.',
      itemAbsent: ' : aucune prise sur cette carte.',
      productionSub: 'frags obtenus avec',
      yieldSub: 'frags par prise',
    },
  },
  ourSide: 'Notre camp',
  opponent: 'Adversaire',
  parity: '50 % : autant que l’adversaire',
  pctFmt: frPct,
  pctIntFmt: (v) => `${Math.round(v)} %`,
  outcome: { win: 'Victoire', loss: 'Défaite', tie: 'Égalité', dnf: 'Abandon' },
  outcomeLower: { win: 'victoire', loss: 'défaite', tie: 'égalité', dnf: 'abandon' },
  control: {
    title: 'Contrôle des ressources',
    info:
      'La part de chaque ressource prise par notre camp face à l’adversaire, sur les matchs de ' +
      'la soirée. Les nombres sont des comptes ; le trait orange marque 50 %, autant que ' +
      'l’adversaire. Les bonus sans ramasseur connu ne comptent dans aucun camp.',
    ariaLabel: 'Notre part des prises de chaque ressource, face à l’adversaire',
    segmentTip: (side, resource, sub, value, tot, pct) => `${side} · ${resource} (${sub})\n${value} sur ${tot} (${pct})`,
  },
  fil: {
    title: 'Contrôle des ressources au fil de la session',
    info:
      'Notre part des prises de chaque ressource, cumulée depuis le premier match de la ' +
      'soirée. Les petits points sont la part de chaque match, leur taille son volume. Un match ' +
      'sans la ressource laisse la courbe filer jusqu’au suivant.',
    winLoss: 'Victoire, défaite',
    dominance: 'Drapeau de dominance',
    pointTip: (v) =>
      `${v.match}${v.outcome ? ` (${v.outcome})` : ''}\n${v.resource} : ${v.us} pour nous, ${v.them} pour eux (${v.pct})\n` +
      `Cumul : ${v.cumUs} sur ${v.cumTotal} (${v.cumPct})`,
    endTip: (resource, cumUs, cumTotal, pct) => `${resource}\nCumul de la soirée : ${cumUs} sur ${cumTotal} (${pct})`,
    bandTip: (match, result, dominance) => `${match}${result ? `\n${result}` : ''}${dominance ? ` · ${dominance}` : ''}`,
  },
  sheets: {
    title: 'Répartition des prises dans l’escouade',
    info:
      'Qui, dans notre camp, a pris chaque bonus et chaque arme spéciale de la soirée, une ' +
      'pastille par prise. Pour un bonus, une pastille vide est une prise perdue : gardée sans ' +
      'être activée, ou lâchée en mourant. L’usage d’une arme spéciale n’est pas mesuré par ' +
      'prise : ses pastilles sont toutes pleines.',
    dominant: 'Ressource dominante',
    rest: 'Reste du camp',
    legendTaken: 'Prise',
    legendLost: 'Bonus pris puis perdu',
    legendRest: 'Reste du camp',
    lossesTitle: 'Bonus perdus',
    lossesFmt: (lost, taken) => `${lost} sur ${taken}`,
    lineTip: (player, item, n, camp, lost) =>
      `${player} · ${item}\n${n} des ${camp} prises de notre camp${lost ? `\n${lost}` : ''}`,
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
    title: 'Frags obtenus avec les ressources',
    info:
      'La barre épaisse partage les frags obtenus grâce à la ressource, la barre fine ce qui les a ' +
      'permis (temps d’effet d’un bonus, prises d’une arme spéciale), toutes deux sur les matchs où ' +
      'ce qui les a permis est mesuré. Si la coupure de la barre épaisse est à gauche de celle de la ' +
      'fine, on a moins produit qu’on n’a eu. Les frags de toute la soirée se lisent match par match.',
    ariaLabel: 'Notre part des frags obtenus avec chaque ressource, et de ce qui les a permis',
    thinLegend: 'Barre fine : temps d’effet ou prises',
    exposure: {
      effect_ms: { name: 'temps d’effet', fmt: duration },
      pickups: { name: 'prises sur les socles', fmt: (v) => `${v} prise${v > 1 ? 's' : ''}` },
    },
    segmentTip: (side, sub, value, tot, pct) => `${side} · ${sub}\n${value} sur ${tot} (${pct})`,
    thinTip: (side, name, value, pct) => `${side} · ${name}\n${value} (${pct})`,
    exposureLine: (name, value, pct) => `${name} : ${value} · ${pct}`,
  },
  yield: {
    title: 'Rendement face à l’adversaire',
    info:
      'Combien notre camp produit de plus ou de moins que l’adversaire pour la même exposition : ' +
      'par minute d’effet d’un bonus, par prise d’arme spéciale. Zéro veut dire autant que lui. ' +
      'Les deux rendements bruts sont écrits sous la valeur.',
    ariaLabel: 'Notre rendement face à celui de l’adversaire, par ressource',
    more: 'Plus productifs que l’adversaire',
    less: 'Moins',
    axis: ['−50 %', '−25', 'autant', '+25', '+50 %'],
    gapFmt: (gap) => signedPct(gap, ' '),
    rawFmt: (us, them) => `${dec1(us, 'fr-FR')} contre ${dec1(them, 'fr-FR')}`,
    tip: (resource, sub, us, them, gap) => `${resource} · ${sub}\nNous ${dec1(us, 'fr-FR')}, eux ${dec1(them, 'fr-FR')} : ${gap}`,
  },
  habit: {
    title: 'Contrôle des ressources, soirée après soirée',
    info:
      'Notre part des prises sur chaque soirée comparable de la composition (mêmes familles de ' +
      'mode que ce soir), la dernière à droite. Le trait fin pointillé de chaque couleur est la ' +
      'médiane des soirées précédentes.',
    tonight: 'ce soir',
    pointTip: (resource, evening, value, med) =>
      `${resource}\n${evening} : ${value}${med ? `\nMédiane des soirées précédentes : ${med}` : ''}`,
    eveningOf: (date) => `Soirée du ${date}`,
    medianTip: (resource, value) => `${resource}\nMédiane des soirées précédentes : ${value}`,
    noHistory: (list) => ({ lead: 'Aucune soirée précédente comparable. ', rest: `Ce soir, notre part des prises : ${list}.` }),
    shareItem: (resource, pct) => `${resource.toLowerCase()} ${pct}`,
  },
  grid: {
    title: 'Contrôle des ressources, match par match',
    info:
      'Une colonne par match, dans l’ordre de la soirée, avec son résultat. La couleur dit si ' +
      'notre camp a pris plus ou moins que l’adversaire, et sature à trente points d’écart. Le ' +
      'survol d’une case détaille les armes.',
    more: 'Plus que l’adversaire',
    less: 'Moins',
    nothing: 'Rien à prendre',
    noFilm: 'Sans film',
    noFilmCell: 'sans film',
    noFilmTip: 'Film non décodé : rien à lire pour cette ligne.',
    noTeamCell: 'camp inconnu',
    noTeamTip: 'Notre camp est inconnu sur ce match (chacun pour soi, ou camp absent de la feuille de match) : rien ne se partage entre les deux camps.',
    untieredCell: 'non classé',
    untieredTip: 'Niveaux de socle non mesurés sur ce match : armes spéciales et armes de râtelier ne se séparent pas.',
    unestablishedTip: 'Carte absente de la référence des socles : armes spéciales et armes de râtelier ne se séparent pas.',
    killsSub: 'frags obtenus avec',
    killsName: 'Frags aux armes spéciales',
    killsAbsent: 'Aucun frag à l’arme spéciale.',
    racks: 'Armes de râtelier',
    racksCount: (n) => `(${n}, prises)`,
    matchHead: (time, map, mode, result) =>
      `${[time, map].filter(Boolean).join(' · ')}${mode || result ? ` (${[mode, result].filter(Boolean).join(', ')})` : ''}`,
    cellTip: (name, us, them, pct) => `${name} : ${us} pour nous, ${them} pour eux (${pct})`,
    whoFmt: (list) => `Chez nous : ${list}`,
    restLower: 'reste du camp',
    padsFmt: (emptied, attributed) => `${emptied} socles vidés, ${attributed} prises attribuées`,
    dominanceTip: (label) => `${label}\nDrapeau de dominance du match`,
  },
}

const EN: EmpriseText = {
  sections: {
    bilan: 'Session summary',
    roles: 'Roles within the squad',
    carte: 'Map by map',
    prendre: 'Taking, and using',
    habitude: 'Compared with usual',
  },
  resources: {
    powerup: {
      label: 'Power-ups',
      pisteSub: 'pickups · camo, overshield',
      gridSub: 'pickups',
      footer: 'power-ups',
      absent: 'No power-up on this map.',
      itemAbsent: ': not on this map.',
      productionSub: 'kills during the effect',
      yieldSub: 'kills per minute of effect',
    },
    power_weapon: {
      label: 'Power weapons',
      pisteSub: 'pickups from the pads',
      gridSub: 'pickups from the pads',
      footer: 'power weapons',
      absent: 'No power weapon picked up.',
      itemAbsent: ': not picked up on this map.',
      productionSub: 'kills with them',
      yieldSub: 'kills per pickup',
    },
    rack: {
      label: 'Rack weapons',
      pisteSub: 'pickups',
      gridSub: 'pickups',
      footer: 'rack weapons',
      absent: 'No rack weapon picked up.',
      itemAbsent: ': not picked up on this map.',
      productionSub: 'kills with them',
      yieldSub: 'kills per pickup',
    },
  },
  ourSide: 'Our side',
  opponent: 'Opponent',
  parity: '50%: as much as the opponent',
  pctFmt: enPct,
  pctIntFmt: (v) => `${Math.round(v)}%`,
  outcome: { win: 'Win', loss: 'Loss', tie: 'Tie', dnf: 'Left' },
  outcomeLower: { win: 'win', loss: 'loss', tie: 'tie', dnf: 'left' },
  control: {
    title: 'Resource control',
    info:
      'The share of each resource our side picked up against the opponent, over the session’s ' +
      'matches. The numbers are counts; the orange line marks 50%, as much as the opponent. ' +
      'Power-ups with no known picker count for neither side.',
    ariaLabel: 'Our share of each resource’s pickups, against the opponent',
    segmentTip: (side, resource, sub, value, tot, pct) => `${side} · ${resource} (${sub})\n${value} of ${tot} (${pct})`,
  },
  fil: {
    title: 'Resource control over the session',
    info:
      'Our share of each resource’s pickups, cumulated from the session’s first match. The ' +
      'small dots are each match’s share, their size its volume. A match without the resource ' +
      'lets the line run on to the next one.',
    winLoss: 'Win, loss',
    dominance: 'Dominance flag',
    pointTip: (v) =>
      `${v.match}${v.outcome ? ` (${v.outcome})` : ''}\n${v.resource}: ${v.us} for us, ${v.them} for them (${v.pct})\n` +
      `Cumulated: ${v.cumUs} of ${v.cumTotal} (${v.cumPct})`,
    endTip: (resource, cumUs, cumTotal, pct) => `${resource}\nSession total: ${cumUs} of ${cumTotal} (${pct})`,
    bandTip: (match, result, dominance) => `${match}${result ? `\n${result}` : ''}${dominance ? ` · ${dominance}` : ''}`,
  },
  sheets: {
    title: 'Pickups within the squad',
    info:
      'Who on our side picked up each power-up and each power weapon of the session, one dot ' +
      'per pickup. For a power-up, a hollow dot is a lost pickup: held without being ' +
      'activated, or dropped on death. Power weapon use isn’t measured per pickup: its dots are ' +
      'all filled.',
    dominant: 'Main resource',
    rest: 'Rest of the side',
    legendTaken: 'Pickup',
    legendLost: 'Power-up picked up, then lost',
    legendRest: 'Rest of the side',
    lossesTitle: 'Lost power-ups',
    lossesFmt: (lost, taken) => `${lost} of ${taken}`,
    lineTip: (player, item, n, camp, lost) =>
      `${player} · ${item}\n${n} of our side’s ${camp} pickups${lost ? `\n${lost}` : ''}`,
    lostFmt: (kept, dropped) => {
      const n = kept + dropped
      if (n <= 0) return null
      if (dropped === 0) return `${n} lost: held without activating`
      if (kept === 0) return `${n} lost: dropped on death`
      return `${n} lost: ${kept} held without activating, ${dropped} dropped on death`
    },
  },
  production: {
    title: 'Kills with resources',
    info:
      'The thick bar splits the kills the resource brought, the thin bar what made them possible ' +
      '(effect time for a power-up, pickups for a power weapon), both over the matches where the ' +
      'latter is measured. If the thick bar’s split sits left of the thin one’s, we produced less ' +
      'than we had. Kills over the whole session read match by match.',
    ariaLabel: 'Our share of the kills made with each resource, and of what made them possible',
    thinLegend: 'Thin bar: effect time or pickups',
    exposure: {
      effect_ms: { name: 'effect time', fmt: duration },
      pickups: { name: 'pickups from the pads', fmt: (v) => `${v} pickup${v > 1 ? 's' : ''}` },
    },
    segmentTip: (side, sub, value, tot, pct) => `${side} · ${sub}\n${value} of ${tot} (${pct})`,
    thinTip: (side, name, value, pct) => `${side} · ${name}\n${value} (${pct})`,
    exposureLine: (name, value, pct) => `${name}: ${value} · ${pct}`,
  },
  yield: {
    title: 'Efficiency against the opponent',
    info:
      'How much more or less our side produces than the opponent for the same exposure: per ' +
      'minute of power-up effect, per power weapon pickup. Zero means as much as them. Both raw ' +
      'rates are written next to the value.',
    ariaLabel: 'Our efficiency against the opponent’s, per resource',
    more: 'More productive than the opponent',
    less: 'Less',
    axis: ['−50%', '−25', 'even', '+25', '+50%'],
    gapFmt: (gap) => signedPct(gap, ''),
    rawFmt: (us, them) => `${dec1(us, 'en-GB')} vs ${dec1(them, 'en-GB')}`,
    tip: (resource, sub, us, them, gap) => `${resource} · ${sub}\nUs ${dec1(us, 'en-GB')}, them ${dec1(them, 'en-GB')}: ${gap}`,
  },
  habit: {
    title: 'Resource control, session by session',
    info:
      'Our share of the pickups on each comparable session of the line-up (same mode families as ' +
      'tonight), the latest on the right. Each colour’s thin dotted line is the median of the ' +
      'previous sessions.',
    tonight: 'tonight',
    pointTip: (resource, evening, value, med) =>
      `${resource}\n${evening}: ${value}${med ? `\nMedian of the previous sessions: ${med}` : ''}`,
    eveningOf: (date) => `Session of ${date}`,
    medianTip: (resource, value) => `${resource}\nMedian of the previous sessions: ${value}`,
    noHistory: (list) => ({ lead: 'No comparable previous session. ', rest: `Tonight, our share of the pickups: ${list}.` }),
    shareItem: (resource, pct) => `${resource.toLowerCase()} ${pct}`,
  },
  grid: {
    title: 'Resource control, match by match',
    info:
      'One column per match, in session order, with its result. The colour says whether our ' +
      'side picked up more or less than the opponent, and saturates at a thirty-point gap. ' +
      'Hover a cell for the weapons.',
    more: 'More than the opponent',
    less: 'Less',
    nothing: 'Nothing to pick up',
    noFilm: 'No film',
    noFilmCell: 'no film',
    noFilmTip: 'Film not decoded: nothing to read for this row.',
    noTeamCell: 'side unknown',
    noTeamTip: 'Our side is unknown in this match (free-for-all, or side missing from the match sheet): nothing splits between the two sides.',
    untieredCell: 'unsorted',
    untieredTip: 'Pad levels not measured for this match: power weapons and rack weapons can’t be told apart.',
    unestablishedTip: 'Map missing from the pad reference: power weapons and rack weapons can’t be told apart.',
    killsSub: 'kills with them',
    killsName: 'Power weapon kills',
    killsAbsent: 'No power weapon kill.',
    racks: 'Rack weapons',
    racksCount: (n) => `(${n}, pickups)`,
    matchHead: (time, map, mode, result) =>
      `${[time, map].filter(Boolean).join(' · ')}${mode || result ? ` (${[mode, result].filter(Boolean).join(', ')})` : ''}`,
    cellTip: (name, us, them, pct) => `${name}: ${us} for us, ${them} for them (${pct})`,
    whoFmt: (list) => `On our side: ${list}`,
    restLower: 'rest of the side',
    padsFmt: (emptied, attributed) => `${emptied} pads emptied, ${attributed} attributed pickups`,
    dominanceTip: (label) => `${label}\nThe match’s dominance flag`,
  },
}

export const EMPRISE_TEXT: Record<Locale, EmpriseText> = { fr: FR, en: EN }
