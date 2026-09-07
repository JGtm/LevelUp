/**
 * Exporte les composants et hooks de la feature Tactique.
 */

export { TacticalAnalysisView } from './TacticalAnalysisView'
export type { TacticalAnalysisViewProps } from './TacticalAnalysisView'

export { TacticalKPIStrip } from './TacticalKPIStrip'
export type { TacticalKPIStripProps } from './TacticalKPIStrip'

export { TacticalPlanCard } from './TacticalPlanCard'
export type { TacticalPlanCardProps } from './TacticalPlanCard'

export { TacticalCellCard } from './TacticalCellCard'
export type { TacticalCellCardProps, CellSample } from './TacticalCellCard'

export { useTacticalMapsPlayed, useTacticalRaster, useTacticalBackgroundImage } from './queries'
export type { TacticalRasterResponse, TacticalMapsPlayedResponse } from './queries'

export {
  getQuestionDef,
  getQuestionLabel,
  getQuestionUnit,
  generateAnalysisTitle,
  generateProcessingMessage,
  validateQuestion,
  validateWho,
  isRoutesQuestion,
  isHeatQuestion,
  isDivergentQuestion,
  getSourceLabel,
  type QuestionType,
  type WhoType,
  type SpawnType,
  type ProcessingStatus,
  type QuestionDef,
} from './tacticalView.logic'

export { getTacticalText, TACTICAL_TEXT } from './i18n'
export type { TacticalText, TacticalLocale } from './i18n'
