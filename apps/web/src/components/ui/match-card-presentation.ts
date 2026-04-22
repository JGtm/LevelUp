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

const DEFAULT_OUTCOME_STYLE: MatchCardOutcomeStyle = {
  scoreColor: '#9E9E9E',
  panelBackground: 'rgba(158, 158, 158, 0.12)',
  panelBorder: 'rgba(158, 158, 158, 0.28)',
}

const OUTCOME_STYLES: Record<string, MatchCardOutcomeStyle> = {
  win: {
    scoreColor: '#4CAF50',
    panelBackground: 'rgba(76, 175, 80, 0.14)',
    panelBorder: 'rgba(76, 175, 80, 0.34)',
  },
  loss: {
    scoreColor: '#F44336',
    panelBackground: 'rgba(244, 67, 54, 0.14)',
    panelBorder: 'rgba(244, 67, 54, 0.34)',
  },
  tie: {
    scoreColor: '#9E9E9E',
    panelBackground: 'rgba(158, 158, 158, 0.14)',
    panelBorder: 'rgba(158, 158, 158, 0.3)',
  },
  dnf: {
    scoreColor: '#9E9E9E',
    panelBackground: 'rgba(158, 158, 158, 0.14)',
    panelBorder: 'rgba(158, 158, 158, 0.3)',
  },
}

const NARRATIVE_BADGE_META: Record<string, MatchNarrativeBadgeMeta> = {
  dominant: { label: 'DOMINATION', color: '#00DC82', textColor: '#052e16' },
  humiliation: { label: 'HUMILIATION', color: '#8B5CF6', textColor: '#f8fafc' },
  remontada: { label: 'REMONTADA', color: '#0072B2', textColor: '#f8fafc' },
  debacle: { label: 'DÉBÂCLE', color: '#D55E00', textColor: '#fff7ed' },
  contre_remontada: { label: 'CONTRE-REMONTADA', color: '#33D6FF', textColor: '#082f49' },
}

export function getMatchCardOutcomeStyle(tone: string | null | undefined): MatchCardOutcomeStyle {
  if (!tone) {
    return DEFAULT_OUTCOME_STYLE
  }
  return OUTCOME_STYLES[tone] ?? DEFAULT_OUTCOME_STYLE
}

export function getMatchNarrativeBadgeMeta(type: string | null | undefined): MatchNarrativeBadgeMeta | null {
  if (!type) {
    return null
  }
  return NARRATIVE_BADGE_META[type] ?? null
}
