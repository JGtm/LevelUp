/**
 * ECharts compose en anglais, quelle que soit la langue de l'interface, le texte d'accessibilité d'un
 * graphe dont l'option `aria` est active. Un graphe qui ne fournit pas lui-même `aria.label.description`
 * (texte localisé par l'appelant) n'expose donc aucun libellé plutôt qu'un libellé anglais.
 */
export function sansTexteAriaAnglais<T extends object | undefined>(option: T): T {
  const aria = (option as { aria?: { label?: { description?: string } } } | undefined)?.aria
  if (!aria || aria.label?.description) return option
  return { ...option, aria: { ...aria, label: { ...aria.label, enabled: false } } } as T
}
