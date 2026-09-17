import { outcomeScale, narrativeScale } from '@/lib/accessibility/scales'
import { tokenCssVar } from '@/lib/accessibility'
import type { Locale } from '@/lib/i18n/locale'

export interface MatchCardOutcomeStyle {
  scoreColor: string
  panelBackground: string
  panelBorder: string
}

export interface MatchNarrativeBadgeMeta {
  label: string
  color: string
  textColor: string
}

function hexToRgba(cssVar: string, alpha: number): string {
  // En runtime, le composant aura la vraie couleur via CSS var.
  // Pour panelBackground/panelBorder on construit une valeur color-mix compatible.
  return `color-mix(in srgb, ${cssVar} ${Math.round(alpha * 100)}%, transparent)`
}

const DEFAULT_OUTCOME_STYLE: MatchCardOutcomeStyle = {
  scoreColor: tokenCssVar('divergent-neutral'),
  panelBackground: 'rgba(158, 158, 158, 0.12)', // color-allow: 2026-09-06 (ronde 2, N3) — GRIS NEUTRE du panneau d'une carte de match SANS issue connue (le defaut, avant que l'issue ne teinte la carte) : une valeur de repli, pas une couleur qui dit quelque chose ; dette PREEXISTANTE au lot v2 D
  panelBorder: 'rgba(158, 158, 158, 0.28)', // color-allow: 2026-09-06 (ronde 2, N3) — meme gris neutre, sur la bordure du meme panneau de repli ; dette PREEXISTANTE au lot v2 D
}

export function getMatchCardOutcomeStyle(tone: string | null | undefined): MatchCardOutcomeStyle {
  const key = tone ?? 'dnf'
  const token = outcomeScale(key === 'tie' ? 'draw' : key)
  if (!token) return DEFAULT_OUTCOME_STYLE

  const color = tokenCssVar(token)
  return {
    scoreColor: color,
    panelBackground: hexToRgba(color, 0.14),
    panelBorder: hexToRgba(color, 0.34),
  }
}

/** Types narratifs reconnus par la tuile de match (type ferme : la parite FR/EN
 *  des libelles de badge est verifiee a la compilation). */
export type NarrativeType =
  | 'dominant'
  | 'humiliation'
  | 'remontada'
  | 'debacle'
  | 'contre_remontada'
  | 'sabordage'
  | 'abnegation'

/** Libelles de badge, en majuscules (le badge est rendu en `uppercase`).
 *  Vocabulaire EN aligne sur le manifeste `lib/i18n/generated/match_view.ts`
 *  (cles `narrative.dominance.*`). */
const NARRATIVE_LABELS: Record<Locale, Record<NarrativeType, string>> = {
  fr: {
    dominant:         'DOMINATION',
    humiliation:      'HUMILIATION',
    remontada:        'REMONTADA',
    debacle:          'DÉBÂCLE',
    contre_remontada: 'CONTRE-REMONTADA',
    sabordage:        'SABORDAGE',
    abnegation:       'ABNÉGATION',
  },
  en: {
    dominant:         'DOMINATION',
    humiliation:      'HUMILIATION',
    remontada:        'COMEBACK',
    debacle:          'COLLAPSE',
    contre_remontada: 'COUNTER-COMEBACK',
    sabordage:        'SCUTTLED',
    abnegation:       'SELFLESS',
  },
}

function isNarrativeType(type: string): type is NarrativeType {
  return type in NARRATIVE_LABELS.fr
}

export function getMatchNarrativeBadgeMeta(
  type: string | null | undefined,
  locale: Locale = 'fr',
): MatchNarrativeBadgeMeta | null {
  if (!type) return null
  const token = narrativeScale(type)
  if (!token) return null

  return {
    label:     isNarrativeType(type) ? NARRATIVE_LABELS[locale][type] : type,
    color:     tokenCssVar(token),
    textColor: tokenCssVar(`${token}-text` as Parameters<typeof tokenCssVar>[0]),
  }
}
