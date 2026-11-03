/**
 * Gabarits des dictionnaires de feature des Réglages : remplace les `{nom}` d'un texte par leurs
 * valeurs (un nom absent des valeurs reste tel quel).
 */
export function fill(template: string, vars: Record<string, string | number>): string {
  return template.replace(/\{(\w+)\}/g, (whole, name: string) => (name in vars ? String(vars[name]) : whole))
}
