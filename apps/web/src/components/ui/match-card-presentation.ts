import { outcomeScale, narrativeScale } from '@/lib/accessibility/scales'
import { tokenCssVar } from '@/lib/accessibility'

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
  panelBackground: 'rgba(158, 158, 158, 0.12)',
  panelBorder: 'rgba(158, 158, 158, 0.28)',
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

const NARRATIVE_LABELS: Record<string, string> = {
  dominant:         'DOMINATION',
  humiliation:      'HUMILIATION',
  remontada:        'REMONTADA',
  debacle:          'DÉBÂCLE',
  contre_remontada: 'CONTRE-REMONTADA',
}

export function getMatchNarrativeBadgeMeta(type: string | null | undefined): MatchNarrativeBadgeMeta | null {
  if (!type) return null
  const token = narrativeScale(type)
  if (!token) return null

  return {
    label:     NARRATIVE_LABELS[type] ?? type,
    color:     tokenCssVar(token),
    textColor: tokenCssVar(`${token}-text` as Parameters<typeof tokenCssVar>[0]),
  }
}
