/**
 * viewPanelModel — CE QUE LE PANNEAU « VUE » REÇOIT, en un seul objet : le contexte du match (ce qui est proposé),
 * la valeur de chaque réglage et le geste qui l'écrit, les vues, la commande d'étage, et les filtres du fil. Tout
 * le reste du panneau (rubriques, lignes, accès rapide, compte « N sur M », « modifié ») se déduit de la table des
 * réglages et de cet objet : un réglage de plus ne change pas ce type.
 */
import type { ReplayRelief } from '../hooks/useReplayRelief'
import type { ReplayLocale } from '../i18n/i18n'
import type { FeedFilterOption } from '../model/feedFilterLogic'
import type { SettingsContext } from './settingsAvailability'
import type { SettingId, SettingValue, SettingValues } from './settingsTable'
import type { ViewsController } from './useViewsController'

/** La commande d'étage : l'état de la coupe, la coupe retenue et les gestes qui la changent. */
export type ReplayFloorControls = ReplayRelief['floor']

export interface ReplayViewPanelModel {
  locale: ReplayLocale
  ctx: SettingsContext
  values: SettingValues
  set: (id: SettingId, value: SettingValue) => void
  views: ViewsController
  floor: ReplayFloorControls
  /** La part du relief tirée du maillage, pour l'infobulle du fond Relief ; `null` quand elle est inconnue. */
  meshPercent: number | null
  /** Les filtres du fil que ce match propose, avec leur nombre de lignes. */
  feedOptions: readonly FeedFilterOption[]
}
