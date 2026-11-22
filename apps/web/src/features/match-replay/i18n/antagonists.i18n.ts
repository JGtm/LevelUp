/**
 * antagonists.i18n.ts — LES TEXTES DES ANTAGONISTES du rejeu : le contrat (`AntagonistsText`) et ses
 * deux langues, en un dictionnaire typé par langue (`Record<Locale, AntagonistsText>`).
 *
 * UN DICTIONNAIRE À PART, branché directement par les surfaces qui le lisent : `fr.ts` et `en.ts`
 * sont au plafond de taille du dépôt, et ces textes forment un tout (la bande des titres de la
 * frise, puis la section de la fiche). La parité des langues est tenue par le typage : une langue
 * ajoutée à `config/locales.toml` sans son entrée ici ne compile pas.
 *
 * LE VOCABULAIRE (D-1) : N est le joueur regardé, X le tenant du titre, Y l'ancien tenant, k le
 * compte de X, j celui de Y. « Némésis » est l'adversaire qui a éliminé N le plus de fois,
 * « Souffre-douleur » celui que N a le plus éliminé ; en anglais « Nemesis » et « Bully target »,
 * comme les cartes de la page du match.
 */
import type { Locale } from '@/lib/i18n/locale'

/** Les phrases d'un rôle sur la frise : première prise, puis dépassement d'un ancien tenant. */
interface BandRoleText {
  /** Le titre de l'infobulle : le nom du titre et son tenant. */
  titleFmt: (holder: string) => string
  /** Le premier adversaire à compter : personne ne tenait le titre. */
  firstFmt: (viewpoint: string) => string
  /** Le dépassement : X (k fois) passe devant Y (j fois). */
  overtakeFmt: (viewpoint: string, count: number, previous: string, previousCount: number) => string
}

/** Un titre dans la fiche : son libellé, sa définition (infobulle de l'icône) et son compte. */
interface FicheRoleText {
  label: string
  /** La définition du titre, N étant le joueur de la fiche ; elle dit aussi ce qui arrive à égalité. */
  hintFmt: (subject: string) => string
  /** Le compte du tenant : « k éliminations » (némésis), « éliminé k fois » (souffre-douleur). */
  countFmt: (count: number) => string
}

export interface AntagonistsText {
  /** La bande des titres de la frise : une infobulle par prise de titre. */
  band: {
    nemesis: BandRoleText
    victim: BandRoleText
  }
  /** La section « Antagonistes » de la fiche d'un joueur (D-2). */
  fiche: {
    title: string
    nemesis: FicheRoleText
    victim: FicheRoleText
    /** Ce qui sépare le nom du tenant, son compte et la mention d'égalité (« X, k éliminations, à égalité avec Y »). */
    partSeparator: string
    /** « à égalité avec » : suivi des noms des autres adversaires au même compte. */
    tiedWith: string
    /** La liste de noms : « Y, Z et W » — le séparateur courant, puis le dernier. */
    listSeparator: string
    listLast: string
    /** L'état vide, dans un cadre pointillé. */
    empty: string
  }
}

const FR_ANTAGONISTS = {
  band: {
    nemesis: {
      titleFmt: (holder) => `Némésis : ${holder}`,
      firstFmt: (viewpoint) => `Premier adversaire à éliminer ${viewpoint}.`,
      overtakeFmt: (viewpoint, count, previous, previousCount) =>
        `A éliminé ${viewpoint} ${count} fois et passe devant ${previous} (${previousCount}).`,
    },
    victim: {
      titleFmt: (holder) => `Souffre-douleur : ${holder}`,
      firstFmt: (viewpoint) => `Premier adversaire éliminé par ${viewpoint}.`,
      overtakeFmt: (viewpoint, count, previous, previousCount) =>
        `Éliminé ${count} fois par ${viewpoint}, passe devant ${previous} (${previousCount}).`,
    },
  },
  fiche: {
    title: 'Antagonistes',
    nemesis: {
      label: 'Némésis',
      hintFmt: (subject) =>
        `L'adversaire qui a éliminé ${subject} le plus de fois jusqu'à cet instant. En cas d'égalité, le titre reste à celui qui l'avait.`,
      countFmt: (count) => (count === 1 ? '1 élimination' : `${count} éliminations`),
    },
    victim: {
      label: 'Souffre-douleur',
      hintFmt: (subject) =>
        `L'adversaire que ${subject} a éliminé le plus de fois jusqu'à cet instant. En cas d'égalité, le titre reste à celui qui l'avait.`,
      countFmt: (count) => `éliminé ${count} fois`,
    },
    partSeparator: ', ',
    tiedWith: 'à égalité avec',
    listSeparator: ', ',
    listLast: ' et ',
    empty: 'Aucun antagoniste à cet instant.',
  },
} satisfies AntagonistsText

const EN_ANTAGONISTS = {
  band: {
    nemesis: {
      titleFmt: (holder) => `Nemesis: ${holder}`,
      firstFmt: (viewpoint) => `First opponent to eliminate ${viewpoint}.`,
      overtakeFmt: (viewpoint, count, previous, previousCount) =>
        `Eliminated ${viewpoint} ${count} times and moves ahead of ${previous} (${previousCount}).`,
    },
    victim: {
      titleFmt: (holder) => `Bully target: ${holder}`,
      firstFmt: (viewpoint) => `First opponent eliminated by ${viewpoint}.`,
      overtakeFmt: (viewpoint, count, previous, previousCount) =>
        `Eliminated ${count} times by ${viewpoint}, moves ahead of ${previous} (${previousCount}).`,
    },
  },
  fiche: {
    title: 'Antagonists',
    nemesis: {
      label: 'Nemesis',
      hintFmt: (subject) =>
        `The opponent who has eliminated ${subject} the most times so far. If there is a tie, the title stays with whoever held it.`,
      countFmt: (count) => (count === 1 ? '1 elimination' : `${count} eliminations`),
    },
    victim: {
      label: 'Bully target',
      hintFmt: (subject) =>
        `The opponent ${subject} has eliminated the most times so far. If there is a tie, the title stays with whoever held it.`,
      countFmt: (count) => (count === 1 ? 'eliminated 1 time' : `eliminated ${count} times`),
    },
    partSeparator: ', ',
    tiedWith: 'tied with',
    listSeparator: ', ',
    listLast: ' and ',
    empty: 'No antagonist at this moment.',
  },
} satisfies AntagonistsText

export const ANTAGONISTS_TEXT: Record<Locale, AntagonistsText> = {
  fr: FR_ANTAGONISTS,
  en: EN_ANTAGONISTS,
}
