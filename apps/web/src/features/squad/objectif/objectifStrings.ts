/**
 * objectifStrings.ts — les textes de la section « Objectif » de l'onglet Contributions
 * (lot L3 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26). Titres, aides ⓘ (trois phrases
 * au plus) et libellés : ceux de la maquette C3EW (`.ai/V7.5/MAQUETTE_TRI_CARTES_DEPLACEES_
 * 2026-09-26.html`, section « Proposition (Contributions) : quatre cartes »).
 *
 * Fichier à part (précédent : squadRiposteStrings.ts) : `i18n.ts` de la feature dépasse déjà
 * le seuil de taille. Parité FR / EN garantie par le typage `Record<Locale, …>`.
 */
import type { Locale } from '@/lib/i18n/locale'

import type { ObjectiveRole } from '../formes/model/objectives'

/** Une note en deux temps : le début en gras, la suite en clair (encadré de la maquette). */
export interface ObjectifNote {
  lead: string
  rest: string
}

export interface ObjectifText {
  sectionTitle: string
  roles: Record<ObjectiveRole, string>
  ourSide: string
  opponent: string
  parity: string
  matchesFmt: (n: number) => string
  pctFmt: (v: number, digits?: number) => string
  durationFmt: (seconds: number) => string
  /** Libellé du résultat d'un match (clé canonique win / loss / tie / dnf). */
  outcome: Record<'win' | 'loss' | 'tie' | 'dnf', string>
  balance: {
    title: string
    info: string
    segmentTip: (side: string, column: string, value: string, pct: string) => string
  }
  fil: {
    title: string
    info: string
    winLoss: string
    ariaLabel: string
    belowMinimum: (n: number) => ObjectifNote
    pointTip: (v: {
      match: string
      context: string
      role: string
      value: string
      lobby: string
      pct: string
      cumulative: string
    }) => string
    bandTip: (match: string, context: string, dominance: string | null) => string
    /** « Drapeau, défaite 1–3 » ; sans résultat connu : le mode seul. */
    contextFmt: (mode: string, outcome: string | null, score: string | null) => string
  }
  sheets: {
    title: string
    info: string
    dominantRole: string
    rest: string
    lineTip: (player: string, family: string, column: string, value: string) => string
  }
  evenings: {
    title: string
    info: string
    median: string
    winsLosses: string
    tonight: string
    ariaLabel: string
    outOfFmt: (wins: number, matches: number) => string
    familyAbbr: Record<string, string>
    /** « B : Bases » — une entrée de la clé des abréviations. */
    abbrItem: (abbr: string, name: string) => string
    pointTip: (role: string, evening: string, value: string, median: string | null) => string
    bandTip: (evening: string, wins: number, matches: number, mix: string) => string
    eveningOf: (date: string) => string
    belowMinimum: (n: number, below: number, withObjective: number) => ObjectifNote
    noHistory: (take: string, defend: string, hold: string, wins: number, matches: number) => ObjectifNote
    history: (count: number, abbrLegend: string) => ObjectifNote
  }
}

function frPct(v: number, digits = 0): string {
  return `${v.toFixed(digits).replace('.', ',')} %`
}

function enPct(v: number, digits = 0): string {
  return `${v.toFixed(digits)}%`
}

/** « 2 min 05 » ; sous la minute : « 45 s ». Mêmes mots en français et en anglais. */
function duration(seconds: number): string {
  const s = Math.max(0, Math.round(seconds))
  const m = Math.floor(s / 60)
  return m > 0 ? `${m} min ${String(s % 60).padStart(2, '0')}` : `${s} s`
}

const FR: ObjectifText = {
  sectionTitle: 'Objectif',
  roles: { take: 'Prendre', defend: 'Défendre', hold: 'Tenir' },
  ourSide: 'Équipe',
  opponent: 'Adversaire',
  parity: '50 % : autant que l’adversaire',
  matchesFmt: (n) => (n > 1 ? `${n} matchs` : `${n} match`),
  pctFmt: frPct,
  durationFmt: duration,
  outcome: { win: 'victoire', loss: 'défaite', tie: 'égalité', dnf: 'abandon' },
  balance: {
    title: 'Rapport de force',
    info:
      'Actions de l’objectif de l’équipe face à celles de l’adversaire, par famille de mode ; trait orange : 50 %.',
    segmentTip: (side, column, value, pct) => `${side} · ${column} : ${value} (${pct})`,
  },
  fil: {
    title: 'Rapport de force au fil de la session',
    info:
      'Part de l’équipe dans les actions de l’objectif du lobby, par rôle, cumulée depuis le premier match à ' +
      'objectif de la soirée, chaque match pesant pareil. Points : part de chaque match, taille selon le volume.',
    winLoss: 'Victoire, défaite',
    ariaLabel: 'Part de l’équipe dans l’objectif, cumulée au fil de la soirée, par rôle',
    belowMinimum: (n) => ({
      lead: `${n > 1 ? `${n} matchs` : `${n} match`} à objectif ce soir`,
      rest: ' : sous le minimum de trois, la carte se masque.',
    }),
    pointTip: (v) =>
      `${v.match} (${v.context})\n${v.role} : ${v.value} sur ${v.lobby} (${v.pct})\nCumul : ${v.cumulative}`,
    bandTip: (match, context, dominance) =>
      `${match}\n${context}${dominance ? `\n${dominance}` : ''}`,
    contextFmt: (mode, outcome, score) =>
      outcome ? `${mode}, ${outcome}${score ? ` ${score}` : ''}` : mode,
  },
  sheets: {
    title: 'Répartition de l’objectif dans l’escouade',
    info:
      'Actions de l’objectif de chaque joueur et du reste de l’équipe, rôle par rôle, dans le même ordre sur ' +
      'chaque fiche ; un zéro reste affiché, atténué.',
    dominantRole: 'Rôle dominant',
    rest: 'Reste de l’équipe',
    lineTip: (player, family, column, value) => `${player} · ${family}\n${column} : ${value}`,
  },
  evenings: {
    title: 'Rapport de force, soirée après soirée',
    info:
      'Part de l’équipe dans les actions de l’objectif du lobby, par rôle, pour chaque soirée d’au moins trois ' +
      'matchs à objectif, ce soir à droite ; pointillé fin : médiane des soirées précédentes.',
    median: 'Médiane des soirées précédentes',
    winsLosses: 'Matchs gagnés, perdus',
    tonight: 'ce soir',
    ariaLabel: 'Part de l’équipe dans l’objectif par rôle, soirée après soirée',
    outOfFmt: (wins, matches) => `${wins} sur ${matches}`,
    familyAbbr: {
      ctf: 'D',
      zones_strongholds: 'B',
      zones_koth: 'RC',
      oddball: 'C',
      stockpile: 'R',
      extraction: 'E',
      vip: 'V',
    },
    abbrItem: (abbr, name) => `${abbr} : ${name}`,
    pointTip: (role, evening, value, median) =>
      `${role} · ${evening}\nPart de l’équipe : ${value}${median ? ` (médiane des précédentes ${median})` : ''}`,
    bandTip: (evening, wins, matches, mix) =>
      `${evening}\n${wins} victoire${wins > 1 ? 's' : ''} sur ${matches} matchs à objectif (${mix})`,
    eveningOf: (date) => `soirée du ${date}`,
    belowMinimum: (n, below, withObjective) => ({
      lead: `${n > 1 ? `${n} matchs` : `${n} match`} à objectif ce soir`,
      rest:
        ' : sous le minimum de trois, la soirée n’a pas de point et la carte se masque.' +
        (withObjective > 0
          ? ` C’est fréquent : ${below} soirées de la composition sur ${withObjective} avec de ` +
            'l’objectif n’en ont qu’un ou deux.'
          : ''),
    }),
    noHistory: (take, defend, hold, wins, matches) => ({
      lead: 'Première soirée à objectif de la composition',
      rest:
        ` (trois matchs ou plus) : pas d’historique. Ce soir-là : Prendre ${take}, Défendre ` +
        `${defend}, Tenir ${hold}, ${wins} victoire${wins > 1 ? 's' : ''} sur ${matches}.`,
    }),
    history: (count, abbrLegend) => ({
      lead: '',
      rest:
        (count > 1
          ? `Les ${count} soirées précédentes d’au moins trois matchs à objectif`
          : 'La soirée précédente d’au moins trois matchs à objectif') +
        `, quels que soient leurs modes (${abbrLegend}). Chaque match pèse pareil dans la part ` +
        'd’une soirée, comme dans « Rapport de force au fil de la session ».',
    }),
  },
}

const EN: ObjectifText = {
  sectionTitle: 'Objective',
  roles: { take: 'Take', defend: 'Defend', hold: 'Hold' },
  ourSide: 'Team',
  opponent: 'Opponent',
  parity: '50%: as much as the opponent',
  matchesFmt: (n) => (n > 1 ? `${n} matches` : `${n} match`),
  pctFmt: enPct,
  durationFmt: duration,
  outcome: { win: 'win', loss: 'loss', tie: 'draw', dnf: 'left' },
  balance: {
    title: 'Balance of power',
    info: 'The team’s objective actions against the opponent’s, by mode family; orange line: 50%.',
    segmentTip: (side, column, value, pct) => `${side} · ${column}: ${value} (${pct})`,
  },
  fil: {
    title: 'Balance of power over the session',
    info:
      'The team’s share of the lobby’s objective actions, by role, accumulated since the evening’s first ' +
      'objective match, each match weighing the same. Dots: each match’s share, sized by volume.',
    winLoss: 'Win, loss',
    ariaLabel: 'The team’s share of the objective, accumulated over the evening, by role',
    belowMinimum: (n) => ({
      lead: `${n > 1 ? `${n} objective matches` : `${n} objective match`} tonight`,
      rest: ': below the minimum of three, the card is hidden.',
    }),
    pointTip: (v) =>
      `${v.match} (${v.context})\n${v.role}: ${v.value} of ${v.lobby} (${v.pct})\nRunning: ${v.cumulative}`,
    bandTip: (match, context, dominance) =>
      `${match}\n${context}${dominance ? `\n${dominance}` : ''}`,
    contextFmt: (mode, outcome, score) =>
      outcome ? `${mode}, ${outcome}${score ? ` ${score}` : ''}` : mode,
  },
  sheets: {
    title: 'Objective share within the squad',
    info:
      'Objective actions of each player and of the rest of the team, role by role, in the same order on ' +
      'every sheet; a zero stays, dimmed.',
    dominantRole: 'Dominant role',
    rest: 'Rest of the team',
    lineTip: (player, family, column, value) => `${player} · ${family}\n${column}: ${value}`,
  },
  evenings: {
    title: 'Balance of power, session by session',
    info:
      'The team’s share of the lobby’s objective actions, by role, for each evening with at least three ' +
      'objective matches, tonight on the right; thin dotted line: median of previous evenings.',
    median: 'Median of previous evenings',
    winsLosses: 'Matches won, lost',
    tonight: 'tonight',
    ariaLabel: 'The team’s share of the objective by role, evening after evening',
    outOfFmt: (wins, matches) => `${wins} of ${matches}`,
    familyAbbr: {
      ctf: 'CTF',
      zones_strongholds: 'SH',
      zones_koth: 'KH',
      oddball: 'OB',
      stockpile: 'SP',
      extraction: 'EX',
      vip: 'VIP',
    },
    abbrItem: (abbr, name) => `${abbr}: ${name}`,
    pointTip: (role, evening, value, median) =>
      `${role} · ${evening}\nTeam share: ${value}${median ? ` (median of previous ${median})` : ''}`,
    bandTip: (evening, wins, matches, mix) =>
      `${evening}\n${wins} win${wins > 1 ? 's' : ''} out of ${matches} objective matches (${mix})`,
    eveningOf: (date) => `evening of ${date}`,
    belowMinimum: (n, below, withObjective) => ({
      lead: `${n > 1 ? `${n} objective matches` : `${n} objective match`} tonight`,
      rest:
        ': below the minimum of three, the evening has no point and the card is hidden.' +
        (withObjective > 0
          ? ` It is common: ${below} of the line-up’s ${withObjective} evenings with objective ` +
            'play have only one or two.'
          : ''),
    }),
    noHistory: (take, defend, hold, wins, matches) => ({
      lead: 'First objective evening of the line-up',
      rest:
        ` (three matches or more): no history. That evening: Take ${take}, Defend ${defend}, ` +
        `Hold ${hold}, ${wins} win${wins > 1 ? 's' : ''} out of ${matches}.`,
    }),
    history: (count, abbrLegend) => ({
      lead: '',
      rest:
        (count > 1
          ? `The ${count} previous evenings with at least three objective matches`
          : 'The previous evening with at least three objective matches') +
        `, whatever their modes (${abbrLegend}). Each match weighs the same in an evening’s ` +
        'share, as in “Balance of power over the session”.',
    }),
  },
}

export const OBJECTIF_TEXT: Record<Locale, ObjectifText> = { fr: FR, en: EN }
