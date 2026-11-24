/**
 * viewLabel — LE NOM ET LA PHRASE de la vue active, dans la langue de la page : celui d'un préréglage vient du
 * dictionnaire, celui d'une vue enregistrée est le nom que le lecteur lui a donné. Partagé par la ligne d'état du
 * panneau et le bouton « Vue » de la carte.
 */
import type { SettingsText } from '../i18n/settings.i18n'
import { isPresetId } from './viewPresets'
import type { ViewsController } from './useViewsController'

export interface ActiveViewLabel {
  name: string
  /** La phrase qui dit ce que la vue montre (préréglage) ou ce qu'elle est (vue enregistrée). */
  hint: string
  modified: boolean
}

export function activeViewLabel(t: SettingsText, views: Pick<ViewsController, 'state' | 'activeUser' | 'changed'>): ActiveViewLabel {
  const modified = views.changed.length > 0
  if (views.activeUser) return { name: views.activeUser.name, hint: t.myViewState, modified }
  const id = views.state.activeId
  return isPresetId(id) ? { name: t.presets[id].label, hint: t.presets[id].hint, modified } : { name: '', hint: '', modified }
}
