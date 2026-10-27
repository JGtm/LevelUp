/** Textes de l'onglet « Emprise » en français (contrat : `contract.ts`). */
import { intlLocale } from '@/lib/formatters/intlLocale'
import { dec1, duration, signedPct } from './format'
import type { BaseEmpriseText } from './contract'

const frPct = (v: number): string => `${(Math.round(v * 10) / 10).toLocaleString(intlLocale('fr'), { maximumFractionDigits: 1 })} %`

export const FR_TEXT = {
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
      'soirée ; trait orange : 50 %.',
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
      'véhicule), sur la soirée.',
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
    rawFmt: (us, them) => `${dec1(us, intlLocale('fr'))} contre ${dec1(them, intlLocale('fr'))}`,
    tip: (resource, sub, us, them, gap) => `${resource} · ${sub}\nÉquipe ${dec1(us, intlLocale('fr'))}, adversaire ${dec1(them, intlLocale('fr'))} : ${gap}`,
  },
  habit: {
    title: 'Contrôle des ressources, par soirée',
    info:
      'Part de l’équipe dans les prises de chaque ressource : un groupe de barres par soirée de la composition, ' +
      'la plus récente à droite. Pointillé fin : médiane des soirées précédentes jouées dans les modes de ce ' +
      'soir ; une soirée sans aucun de ces modes n’entre pas dans la médiane.',
    tonight: 'ce soir',
    pointTip: (resource, evening, value, med) =>
      `${resource}\n${evening} : ${value}${med ? `\nMédiane des soirées précédentes : ${med}` : ''}`,
    eveningOf: (date) => `Soirée du ${date}`,
    medianTip: (resource, value) => `${resource}\nMédiane des soirées précédentes : ${value}`,
    notComparableTip: (families) => `Autres modes que ce soir${families ? ` (${families})` : ''} : hors médiane`,
    empty: { title: 'Aucune prise', description: 'Aucune prise de ressource sur les matchs filmés des soirées de la composition.' },
  },
  grid: {
    title: 'Contrôle des ressources, par match',
    info:
      'Une colonne par match de la soirée, avec son résultat ; couleur : écart entre les prises de l’équipe et ' +
      'celles de l’adversaire, saturée à trente points.',
    more: 'Plus que l’adversaire',
    less: 'Moins',
    nothing: 'Rien à prendre',
    killsName: 'Frags avec arme spéciale',
    killsAbsent: 'Aucun frag à l’arme spéciale.',
    rackTypes: (n) => `${n} ${n > 1 ? 'types d’armes' : 'type d’arme'}`,
    matchHead: (time, map, mode, result) =>
      `${[time, map].filter(Boolean).join(' · ')}${mode || result ? ` (${[mode, result].filter(Boolean).join(', ')})` : ''}`,
    cellTip: (name, us, them, pct) => `${name} : équipe ${us}, adversaire ${them} (${pct})`,
    whoFmt: (list) => `Équipe : ${list}`,
    restLower: 'reste de l’équipe',
    padsFmt: (emptied, attributed) => `${emptied} socles vidés, ${attributed} prises attribuées`,
    dominanceTip: (label) => `${label}\nDrapeau de dominance du match`,
  },
} satisfies BaseEmpriseText
