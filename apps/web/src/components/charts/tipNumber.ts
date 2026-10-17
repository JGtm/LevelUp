/**
 * tipNumber.ts — nombres d'infobulle ECharts, dans la langue de l'application (source unique).
 *
 * Feuille de dépendances : `_utils.ts` (qui pose le formateur par défaut des infobulles) et
 * `axisTooltip.ts` en dépendent, jamais l'inverse.
 */
import { intlLocale } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'

/** Signe moins typographique, le même que `formatSignedFixed`. */
const MINUS = '−'

/** Décimales au plus d'un nombre affiché sans précision demandée (`tipNumberAuto`). */
export const TIP_AUTO_MAX_DECIMALS = 2

const formats = new Map<string, Intl.NumberFormat>()

function numberFormat(locale: string, min: number, max: number, signed: boolean): Intl.NumberFormat {
  const key = `${locale}|${min}|${max}|${signed ? 1 : 0}`
  let nf = formats.get(key)
  if (!nf) {
    nf = new Intl.NumberFormat(locale, {
      minimumFractionDigits: min,
      maximumFractionDigits: max,
      ...(signed ? { signDisplay: 'exceptZero' as const } : {}),
    })
    formats.set(key, nf)
  }
  return nf
}

function format(value: number, min: number, max: number, signed: boolean): string {
  if (!Number.isFinite(value)) return '—'
  const text = numberFormat(intlLocale(useAppShellStore.getState().locale), min, max, signed).format(value)
  return text.replace(/^-/, MINUS)
}

/**
 * Nombre d'infobulle dans la langue de l'application (« 1 234,5 » / « 1,234.5 »), `decimals`
 * décimales fixes. `signed` : « +3,0 » / « −1,2 » (signe moins typographique), zéro sans signe.
 * Non fini : tiret demi-cadratin.
 */
export function tipNumber(value: number, decimals = 0, signed = false): string {
  return format(value, decimals, decimals, signed)
}

/**
 * Nombre d'infobulle sans précision demandée : entier sans décimale, sinon au plus
 * `TIP_AUTO_MAX_DECIMALS` décimales, jamais « 1.3333333333333333 ». Filet du rendu par défaut
 * d'ECharts (`getTooltipBase`) ; un graphe qui connaît sa précision passe par `tipNumber`.
 */
export function tipNumberAuto(value: number): string {
  return format(value, 0, TIP_AUTO_MAX_DECIMALS, false)
}
