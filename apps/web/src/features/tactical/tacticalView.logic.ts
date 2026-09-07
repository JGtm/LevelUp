/**
 * tacticalView.logic.ts — Logique pure pour l'onglet Tactique.
 *
 * Contient :
 * - Sélection de question
 * - Génération de titre
 * - État du message de traitement
 * - Validation des paramètres
 */

export type QuestionType = 'morts' | 'kills' | 'gagne' | 'temps' | 'routes' | 'isole'
export type WhoType = 'moi' | 'escouade' | 'adv'
export type SpawnType = 'tous' | string // 'tous' ou ID de grappes

/**
 * Métadonnées d'une question tactique.
 */
export interface QuestionDef {
  id: QuestionType
  labelFr: string
  labelEn: string
  unitFr: string
  unitEn: string
  ramp: 'heat' | 'div' | 'none' // 'heat' = chaleur, 'div' = divergent, 'none' = routes
  source: 'base' | 'artefact'
}

export const QUESTION_DEFS: Record<QuestionType, QuestionDef> = {
  morts: {
    id: 'morts',
    labelFr: 'Où je meurs',
    labelEn: 'Where I Die',
    unitFr: 'morts par match',
    unitEn: 'deaths per match',
    ramp: 'heat',
    source: 'base',
  },
  kills: {
    id: 'kills',
    labelFr: 'Où je tue',
    labelEn: 'Where I Kill',
    unitFr: 'kills par match',
    unitEn: 'kills per match',
    ramp: 'heat',
    source: 'base',
  },
  gagne: {
    id: 'gagne',
    labelFr: 'Où je gagne',
    labelEn: 'Where I Win',
    unitFr: 'écart V moins D',
    unitEn: 'win delta',
    ramp: 'div',
    source: 'base',
  },
  temps: {
    id: 'temps',
    labelFr: 'Où je passe mon temps',
    labelEn: 'Where I Spend My Time',
    unitFr: 's par match',
    unitEn: 's per match',
    ramp: 'heat',
    source: 'artefact',
  },
  routes: {
    id: 'routes',
    labelFr: 'Mes routes de spawn',
    labelEn: 'My Spawn Routes',
    unitFr: '',
    unitEn: '',
    ramp: 'none',
    source: 'artefact',
  },
  isole: {
    id: 'isole',
    labelFr: 'Où je meurs isolé',
    labelEn: 'Where I Die Isolated',
    unitFr: 'morts par match',
    unitEn: 'deaths per match',
    ramp: 'heat',
    source: 'artefact',
  },
}

/**
 * getQuestionDef — récupère la définition d'une question.
 */
export function getQuestionDef(question: QuestionType): QuestionDef {
  return QUESTION_DEFS[question]
}

/**
 * getQuestionLabel — retourne le libellé d'une question dans la locale donnée.
 */
export function getQuestionLabel(question: QuestionType, locale: 'fr' | 'en'): string {
  const def = QUESTION_DEFS[question]
  return locale === 'fr' ? def.labelFr : def.labelEn
}

/**
 * getQuestionUnit — retourne l'unité d'une question dans la locale donnée.
 */
export function getQuestionUnit(question: QuestionType, locale: 'fr' | 'en'): string {
  const def = QUESTION_DEFS[question]
  return locale === 'fr' ? def.unitFr : def.unitEn
}

/**
 * generateAnalysisTitle — génère le titre de la section d'analyse.
 *
 * Format : "Plan de <mapName> — <questionLabel>"
 */
export function generateAnalysisTitle(mapName: string, question: QuestionType, locale: 'fr' | 'en'): string {
  const questionLabel = getQuestionLabel(question, locale)
  if (locale === 'fr') {
    return `Plan de ${mapName} — ${questionLabel}`
  }
  return `${mapName} Plan — ${questionLabel}`
}

/**
 * Statut du message de traitement/attente.
 */
export type ProcessingStatus = 'idle' | 'processing' | 'unavailable' | 'empty' | 'error'

/**
 * generateProcessingMessage — génère le message d'état basé sur les compteurs.
 *
 * Règles :
 * - Si matchs_en_attente > 0 → "Traitement en cours: N matchs"
 * - Si matchs_non_cuisables > 0 → "Données non disponibles pour N matchs"
 * - Sinon → idle
 */
export function generateProcessingMessage(
  matchsEnAttente: number,
  matchsNonCuisables: number,
  locale: 'fr' | 'en',
): { status: ProcessingStatus; message: string } {
  if (matchsEnAttente > 0) {
    const label = locale === 'fr' ? 'Traitement en cours : ' : 'Processing: '
    const matchsWord = locale === 'fr' ? 'matchs' : 'matches'
    return {
      status: 'processing',
      message: `${label}${matchsEnAttente} ${matchsWord}`,
    }
  }

  if (matchsNonCuisables > 0) {
    const label = locale === 'fr' ? 'Données non disponibles pour ' : 'Data unavailable for '
    const matchsWord = locale === 'fr' ? 'matchs' : 'matches'
    return {
      status: 'unavailable',
      message: `${label}${matchsNonCuisables} ${matchsWord}`,
    }
  }

  return {
    status: 'idle',
    message: '',
  }
}

/**
 * validateQuestion — valide qu'une question est reconnue.
 */
export function validateQuestion(question: string): question is QuestionType {
  return question in QUESTION_DEFS
}

/**
 * validateWho — valide qu'un mode "qui" est reconnu.
 */
export function validateWho(who: string): who is WhoType {
  return ['moi', 'escouade', 'adv'].includes(who)
}

/**
 * isRoutesQuestion — retourne true si la question est "routes".
 */
export function isRoutesQuestion(question: QuestionType): boolean {
  return question === 'routes'
}

/**
 * isHeatQuestion — retourne true si la question affiche une heatmap.
 */
export function isHeatQuestion(question: QuestionType): boolean {
  const def = QUESTION_DEFS[question]
  return def.ramp !== 'none'
}

/**
 * isDivergentQuestion — retourne true si la question utilise une échelle divergente.
 */
export function isDivergentQuestion(question: QuestionType): boolean {
  const def = QUESTION_DEFS[question]
  return def.ramp === 'div'
}

/**
 * getSourceLabel — retourne le libellé de la source de données dans la locale donnée.
 */
export function getSourceLabel(question: QuestionType, locale: 'fr' | 'en'): string {
  const def = QUESTION_DEFS[question]
  if (def.source === 'base') {
    return locale === 'fr' ? 'base partagée' : 'shared database'
  }
  return locale === 'fr' ? 'artefacts de rejeu cuits' : 'cooked replay artifacts'
}
