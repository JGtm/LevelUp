/**
 * vehicleAboardLabel.ts — LA PARENTHÈSE « À BORD » d'une ligne de destruction de véhicule.
 *
 * UN SEUL JOUEUR CONNU À BORD : son nom, « (Cobra01 à bord) ». DEUX OU DAVANTAGE : leur nombre,
 * « (3 à bord) » — la ligne reste sur une rangée, et une liste de noms ne s'y tiendrait pas. Le
 * nombre est celui des joueurs NOMMÉS par le film (l'occupant et les tués) ; il ne dit pas combien
 * de places étaient prises. La catégorie de pluriel est celle de la LANGUE (`Intl.PluralRules`),
 * jamais un test sur 1. Aucun possessif : la parenthèse ne dit pas à qui appartient le véhicule.
 */
import { intlLocale } from '@/lib/formatters/intlLocale'
import { fillTemplate } from '@/lib/i18n/template'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'

/**
 * @param names les noms affichables des joueurs connus à bord, dans l'ordre de la source ;
 * chaîne vide quand il n'y en a aucun.
 */
export function vehicleAboardLabel(names: readonly string[], locale: ReplayLocale): string {
  const t = REPLAY_TEXT[locale]
  if (names.length === 0) return ''
  if (names.length === 1) return t.feedVehicleOccupant(names[0])
  const one = new Intl.PluralRules(intlLocale(locale)).select(names.length) === 'one'
  return fillTemplate(one ? t.feedVehicleAboardOne : t.feedVehicleAboardOther, { n: names.length })
}
