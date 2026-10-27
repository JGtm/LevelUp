/**
 * pickText — LIRE UN LIBELLÉ PAR LANGUE (ADR 0040, D3).
 *
 * Le document de rejeu garde un libellé par langue DANS le document (gros document, mis en cache
 * côté client, artefacts cuits qu'on ne re-cuit pas) : un libellé d'arme est
 * `{ en: '…', fr: '…', fx: '…', … }`, une propriété par langue de `config/locales.toml`, au même
 * niveau que les autres champs. Ce module est le SEUL endroit qui choisit une langue dans un tel
 * objet : le reste du code appelle `pickText(libellé, locale)` et ne nomme jamais `.fr` / `.en`
 * (garde-rail `replay-label-access.guard.test.ts`).
 *
 * REPLI (D3 : noms de jeu) : la langue demandée, sinon `FALLBACK_LOCALE`, sinon la chaîne vide.
 * Jamais une traduction inventée, jamais « l'autre langue » au hasard de l'ordre des clés.
 */
import { FALLBACK_LOCALE, KNOWN_LOCALES, type Locale } from './locale'

/** Un libellé par langue : toute propriété dont le nom est une langue de la liste est lue, le reste ignoré. */
export type LocaleTexts = Readonly<Partial<Record<Locale, string | null | undefined>>>

/**
 * Le texte de la langue demandée ; à défaut, celui de `FALLBACK_LOCALE` ; à défaut, `''`.
 * Un libellé absent (`undefined` / `null`) rend `''` : l'appelant décide quoi montrer à sa place.
 */
export function pickText(text: LocaleTexts | null | undefined, locale: Locale): string {
  return text?.[locale] || text?.[FALLBACK_LOCALE] || ''
}

/**
 * Les textes de toutes les langues renseignées, dans l'ordre de la liste. Pour les lectures qui
 * cherchent un radical de nom dans n'importe quelle langue (reconnaître « transloc » dans
 * « Quantum Translocator » comme dans sa traduction), jamais pour afficher.
 */
export function allTexts(text: LocaleTexts | null | undefined): string[] {
  if (!text) return []
  return KNOWN_LOCALES.flatMap((l) => {
    const v = text[l]
    return v ? [v] : []
  })
}

/**
 * Projette un objet qui porte des propriétés de langue (un libellé du document, une zone nommée)
 * sur ses seuls textes par langue : les autres propriétés (`img`, `x`, `family`…) n'y entrent pas.
 * Sert à ranger les textes d'une zone dans un champ dédié plutôt que de garder tout l'objet.
 */
export function localeTextsOf(source: object | null | undefined): Partial<Record<Locale, string>> {
  const out: Partial<Record<Locale, string>> = {}
  if (!source) return out
  const record = source as Record<string, unknown>
  for (const l of KNOWN_LOCALES) {
    const v = record[l]
    if (typeof v === 'string') out[l] = v
  }
  return out
}
