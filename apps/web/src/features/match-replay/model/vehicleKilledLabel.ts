/**
 * vehicleKilledLabel.ts — LA PARENTHÈSE « TUÉS AVEC LE VÉHICULE » d'une ligne de destruction.
 *
 * ELLE NE PARLE QUE DES TUÉS (`vehicles[].destruction.killed`, lus dans le film) : un seul joueur
 * nommé, son nom, « (Cobra01 tué avec le véhicule) » ; deux ou davantage, leur nombre, « (3 tués
 * avec le véhicule) » — la ligne reste sur une rangée, et une liste de noms ne s'y tiendrait pas.
 * Aucun tué nommé : aucune parenthèse, même quand le document nomme un occupant. Le siège ne se dit
 * pas. La catégorie de pluriel est celle de la LANGUE (`Intl.PluralRules`), jamais un test sur 1.
 * Aucun possessif : la parenthèse ne dit pas à qui appartient le véhicule.
 */
import { intlLocale } from '@/lib/formatters/intlLocale'
import { fillTemplate } from '@/lib/i18n/template'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'

/**
 * @param names les noms affichables des joueurs morts avec le véhicule, dans l'ordre de la source ;
 * chaîne vide quand il n'y en a aucun.
 */
export function vehicleKilledLabel(names: readonly string[], locale: ReplayLocale): string {
  const t = REPLAY_TEXT[locale]
  if (names.length === 0) return ''
  if (names.length === 1) return t.feedVehicleKilledWho(names[0])
  const one = new Intl.PluralRules(intlLocale(locale)).select(names.length) === 'one'
  return fillTemplate(one ? t.feedVehicleKilledOne : t.feedVehicleKilledOther, { n: names.length })
}
