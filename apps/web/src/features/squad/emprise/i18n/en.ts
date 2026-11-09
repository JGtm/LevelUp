/** Textes de l'onglet « Emprise » en anglais (contrat : `contract.ts`). */
import { intlLocale } from '@/lib/formatters/intlLocale'
import { dec1, duration, signedPct } from './format'
import type { BaseEmpriseText } from './contract'

const enPct = (v: number): string => `${(Math.round(v * 10) / 10).toLocaleString(intlLocale('en'), { maximumFractionDigits: 1 })}%`

export const EN_TEXT = {
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
      'matches; orange line: 50%.',
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
      'exposure (effect time for a power-up, pickups for a power weapon, time aboard a vehicle), over the session.',
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
    rawFmt: (us, them) => `${dec1(us, intlLocale('en'))} vs ${dec1(them, intlLocale('en'))}`,
    tip: (resource, sub, us, them, gap) => `${resource} · ${sub}\nTeam ${dec1(us, intlLocale('en'))}, opponent ${dec1(them, intlLocale('en'))}: ${gap}`,
  },
  habit: {
    title: 'Resource control, by session',
    info:
      'The team’s share of each resource’s pickups: one group of bars per session of the line-up, the latest ' +
      'on the right. Thin dotted line: median of the previous sessions played in tonight’s modes; a session ' +
      'with none of those modes is left out of the median.',
    tonight: 'tonight',
    pointTip: (resource, evening, value, med) =>
      `${resource}\n${evening}: ${value}${med ? `\nMedian of the previous sessions: ${med}` : ''}`,
    eveningOf: (date) => `Session of ${date}`,
    medianTip: (resource, value) => `${resource}\nMedian of the previous sessions: ${value}`,
    notComparableTip: (families) => `Other modes than tonight${families ? ` (${families})` : ''}: left out of the median`,
    empty: { title: 'No pickups', description: 'No resource pickup in the filmed matches of the line-up’s sessions.' },
  },
  grid: {
    title: 'Resource control, by match',
    info:
      'One column per match of the session, with its result; colour: gap between the team’s pickups and ' +
      'the opponent’s, saturated at thirty points.',
    more: 'More than the opponent',
    less: 'Less',
    nothing: 'Nothing to pick up',
    killsName: 'Kills with power weapons',
    killsAbsent: 'No power weapon kill.',
    rackTypes: (n) => `${n} ${n > 1 ? 'weapon types' : 'weapon type'}`,
    matchHead: (time, map, mode, result) =>
      `${[time, map].filter(Boolean).join(' · ')}${mode || result ? ` (${[mode, result].filter(Boolean).join(', ')})` : ''}`,
    cellTip: (name, us, them, pct) => `${name}: team ${us}, opponent ${them} (${pct})`,
    whoFmt: (list) => `Team: ${list}`,
    restLower: 'rest of the team',
    padsFmt: (emptied, attributed) => `${emptied} pads emptied, ${attributed} attributed pickups`,
    dominanceTip: (label) => `${label}\nThe match’s dominance flag`,
  },
} satisfies BaseEmpriseText
