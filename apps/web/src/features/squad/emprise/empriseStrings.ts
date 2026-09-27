/**
 * empriseStrings.ts — les textes de l'onglet « Emprise » de l'Escouade (lot L5 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26). Titres (tableau §3 du plan), aides ⓘ (trois
 * phrases au plus) et libellés : ceux de la maquette
 * `.ai/V7.5/MAQUETTE_ONGLET_TACTIQUE_ESCOUADE_2026-09-26.html` (bloc « Proposition »).
 *
 * Fichier à part (précédent : `objectif/objectifStrings.ts`) : `i18n.ts` de la feature dépasse
 * déjà le seuil de taille. Parité FR / EN garantie par le typage `Record<Locale, …>`.
 */
import type { OutcomeValue } from '@/components/charts/outcomeSequence'
import type { Locale } from '@/lib/i18n/locale'

/** Les noms d'une ressource, selon l'endroit où elle s'écrit. */
export interface ResourceText {
  /** « Bonus », « Armes spéciales ». */
  label: string
  /** Sous-libellé de la piste du bilan (« prises · camouflage, surbouclier »). */
  pisteSub: string
  /** Sous-libellé de la ligne de synthèse de la grille (« prises »). */
  gridSub: string
  /** Le mot du pied de fiche (« bonus », « armes spéciales »). */
  footer: string
  /** Case vide de la ligne de synthèse (« Aucun bonus sur cette carte. »). */
  absent: string
  /** Case vide d'une ligne d'objet (« : pas sur cette carte. »), après le nom de l'objet. */
  itemAbsent: string
}

export interface EmpriseText {
  sections: { bilan: string; roles: string; carte: string }
  resources: Record<string, ResourceText>
  ourSide: string
  opponent: string
  parity: string
  /** « 60 % », « 44,2 % » (une décimale au plus). */
  pctFmt: (v: number) => string
  /** « 17 % » (entier). */
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
  grid: {
    title: string
    info: string
    more: string
    less: string
    nothing: string
    noFilm: string
    noFilmCell: string
    noFilmTip: string
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

function frPct(v: number): string {
  return `${(Math.round(v * 10) / 10).toLocaleString('fr-FR', { maximumFractionDigits: 1 })} %`
}

function enPct(v: number): string {
  return `${(Math.round(v * 10) / 10).toLocaleString('en-GB', { maximumFractionDigits: 1 })}%`
}

const FR: EmpriseText = {
  sections: { bilan: 'Bilan de la soirée', roles: 'Rôles dans l’escouade', carte: 'Carte par carte' },
  resources: {
    powerup: {
      label: 'Bonus',
      pisteSub: 'prises · camouflage, surbouclier',
      gridSub: 'prises',
      footer: 'bonus',
      absent: 'Aucun bonus sur cette carte.',
      itemAbsent: ' : pas sur cette carte.',
    },
    power_weapon: {
      label: 'Armes spéciales',
      pisteSub: 'prises sur les socles',
      gridSub: 'prises sur les socles',
      footer: 'armes spéciales',
      absent: 'Aucune arme spéciale prise.',
      itemAbsent: ' : aucune prise sur cette carte.',
    },
    rack: {
      label: 'Armes de râtelier',
      pisteSub: 'prises',
      gridSub: 'prises',
      footer: 'armes de râtelier',
      absent: 'Aucune arme de râtelier prise.',
      itemAbsent: ' : aucune prise sur cette carte.',
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
  sections: { bilan: 'Session summary', roles: 'Roles within the squad', carte: 'Map by map' },
  resources: {
    powerup: {
      label: 'Power-ups',
      pisteSub: 'pickups · camo, overshield',
      gridSub: 'pickups',
      footer: 'power-ups',
      absent: 'No power-up on this map.',
      itemAbsent: ': not on this map.',
    },
    power_weapon: {
      label: 'Power weapons',
      pisteSub: 'pickups from the pads',
      gridSub: 'pickups from the pads',
      footer: 'power weapons',
      absent: 'No power weapon picked up.',
      itemAbsent: ': not picked up on this map.',
    },
    rack: {
      label: 'Rack weapons',
      pisteSub: 'pickups',
      gridSub: 'pickups',
      footer: 'rack weapons',
      absent: 'No rack weapon picked up.',
      itemAbsent: ': not picked up on this map.',
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
