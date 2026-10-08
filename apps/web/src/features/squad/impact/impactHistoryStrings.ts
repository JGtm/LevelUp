/**
 * impactHistoryStrings.ts — les textes du graphe « Points d'impact par soirée et par rôle »
 * (Escouade › Contributions, sous la matrice d'impact) : titre, aide ⓘ, infobulle, légende et
 * libellés courts des rôles (ceux de la Vue match, plus « Voleur » de l'Escouade).
 *
 * Fichier à part (précédent : objectif/objectifStrings.ts) : `i18n.ts` de la feature dépasse
 * déjà le seuil de taille. Parité FR / EN garantie par le typage `Record<Locale, …>`.
 */
import type { Locale } from '@/lib/i18n/locale'

export interface ImpactHistoryText {
  title: string
  /** Aide ⓘ : le barème est inséré à la place de {scale}, la liste des joueurs à {players}. */
  info: (players: string, scale: string) => string
  /** Libellés courts des rôles, par clé servie. */
  roles: Record<string, string>
  /** Une entrée du barème dans l'aide : « Finisseur et Premier sang +2 ». */
  scaleJoin: (labels: string[]) => string
  /** Nombre signé : « +1,5 », « −2 », « 0 ». */
  signed: (v: number) => string
  eveningOf: (date: string) => string
  eveningAt: (date: string, hour: string) => string
  hourOf: (d: Date) => string
  matchesWins: (matches: number, wins: number) => string
  noRole: string
  net: string
  legendNet: string
  /** Entrée de légende des rôles qui partagent une nuance : « Autres rôles à −1 (…) ». */
  legendGrouped: (weight: string, labels: string[]) => string
  ariaSummary: (first: string, last: string, perPlayer: string) => string
  ariaPlayer: (player: string, nets: string) => string
}

/** Nombre signé à la française : virgule décimale, signe moins typographique. */
function signedWith(decimal: string): (v: number) => string {
  return (v) => {
    const a = Math.abs(v)
    const s = Number.isInteger(a) ? String(a) : a.toFixed(1).replace('.', decimal)
    if (v > 0) return `+${s}`
    if (v < 0) return `−${s}`
    return '0'
  }
}

const pad = (n: number) => String(n).padStart(2, '0')

/** Liste « a, b et c » / « a, b and c ». */
function joinWith(and: string): (labels: string[]) => string {
  return (labels) =>
    labels.length <= 1 ? labels.join('') : `${labels.slice(0, -1).join(', ')} ${and} ${labels[labels.length - 1]}`
}

const frJoin = joinWith('et')
const enJoin = joinWith('and')

const FR: ImpactHistoryText = {
  title: 'Points d’impact par soirée et par rôle',
  info: (players, scale) =>
    `Une barre par joueur et par soirée, dans l’ordre ${players}, repérée par la couleur du joueur au pied. ` +
    'Chaque segment vaut le nombre de fois où le rôle revient au joueur dans la soirée, multiplié par le barème du ' +
    'rôle : au-dessus de zéro les rôles qui ajoutent des points, en dessous ceux qui en retirent. La courbe d’un ' +
    'joueur relie le net de chaque soirée, chaque losange posé sur la barre du joueur. ' +
    `Barème : ${scale}. Première victime, Touriste, Kamikaze et Voleur partagent une couleur. ` +
    'Soirées où les joueurs de l’escouade sont tous dans la même équipe : celles de la période affichée, ' +
    'précédées des soirées antérieures de la composition, onze au plus.',
  roles: {
    clutch_finisher: 'Finisseur',
    first_blood: 'Premier sang',
    silent_hero: 'Héros silencieux',
    top_killer: 'Bourreau',
    last_casualty: 'Boulet',
    false_brother: 'Faux-frère',
    first_group_death: 'Première victime',
    last_group_kill: 'Touriste',
    kamikaze: 'Kamikaze',
    thief: 'Voleur',
  },
  scaleJoin: frJoin,
  signed: signedWith(','),
  eveningOf: (date) => `Soirée du ${date}`,
  eveningAt: (date, hour) => `Soirée du ${date}, ${hour}`,
  hourOf: (d) => `${d.getHours()}h${pad(d.getMinutes())}`,
  matchesWins: (matches, wins) =>
    `${matches} match${matches > 1 ? 's' : ''}, ` +
    (wins === 0 ? 'aucune victoire' : `${wins} victoire${wins > 1 ? 's' : ''}`),
  noRole: 'Aucun rôle',
  net: 'Net',
  legendNet: 'Net de la soirée :',
  legendGrouped: (weight, labels) => `Autres rôles à ${weight} (${labels.join(', ')})`,
  ariaSummary: (first, last, perPlayer) => `Points d’impact par soirée, du ${first} au ${last}. ${perPlayer}.`,
  ariaPlayer: (player, nets) => `${player} : ${nets}`,
}

const EN: ImpactHistoryText = {
  title: 'Impact points per evening and role',
  info: (players, scale) =>
    `One bar per player and evening, in the order ${players}, marked by the player colour at the foot of the bar. ` +
    'Each segment is the number of times the role went to the player that evening, multiplied by the role ' +
    'points: above zero the roles that add points, below zero the roles that remove points. The line of a player ' +
    'joins the net of each evening, each diamond set on the bar of the player. ' +
    `Points: ${scale}. First down, Late starter, Kamikaze and Thief share a colour. ` +
    'Evenings where the squad players are all on the same team: those of the displayed period, preceded by ' +
    'earlier evenings of the line-up, eleven at most.',
  roles: {
    clutch_finisher: 'Finisher',
    first_blood: 'First blood',
    silent_hero: 'Silent hero',
    top_killer: 'Top killer',
    last_casualty: 'Last casualty',
    false_brother: 'False brother',
    first_group_death: 'First down',
    last_group_kill: 'Late starter',
    kamikaze: 'Kamikaze',
    thief: 'Thief',
  },
  scaleJoin: enJoin,
  signed: signedWith('.'),
  eveningOf: (date) => `Evening of ${date}`,
  eveningAt: (date, hour) => `Evening of ${date}, ${hour}`,
  hourOf: (d) => `${pad(d.getHours())}:${pad(d.getMinutes())}`,
  matchesWins: (matches, wins) =>
    `${matches} match${matches > 1 ? 'es' : ''}, ` + (wins === 0 ? 'no win' : `${wins} win${wins > 1 ? 's' : ''}`),
  noRole: 'No role',
  net: 'Net',
  legendNet: 'Evening net:',
  legendGrouped: (weight, labels) => `Other ${weight} roles (${labels.join(', ')})`,
  ariaSummary: (first, last, perPlayer) => `Impact points per evening, from ${first} to ${last}. ${perPlayer}.`,
  ariaPlayer: (player, nets) => `${player}: ${nets}`,
}

export const IMPACT_HISTORY_TEXT: Record<Locale, ImpactHistoryText> = { fr: FR, en: EN }
