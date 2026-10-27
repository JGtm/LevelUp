/** Formats communs aux langues des textes de l'onglet « Emprise ». */

/** « 2 min 39 » — un temps d'effet (millisecondes), à la seconde ; « 45 s » sous la minute. */
export function duration(ms: number): string {
  const sec = Math.round(ms / 1000)
  const m = Math.floor(sec / 60)
  return m === 0 ? `${sec} s` : `${m} min ${String(sec % 60).padStart(2, '0')}`
}

/** Une décimale, séparateur de la langue (« 3,0 », « 3.0 »). */
export function dec1(v: number, locale: string): string {
  return v.toLocaleString(locale, { minimumFractionDigits: 1, maximumFractionDigits: 1 })
}

/** « +14 % » / « −8 % » / « 0 % » : l'écart relatif en points entiers, signe typographique. */
export function signedPct(gap: number, sep: string): string {
  const n = Math.round(Math.abs(gap) * 100)
  if (n === 0) return `0${sep}%`
  return `${gap > 0 ? '+' : '−'}${n}${sep}%`
}
