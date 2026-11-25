/**
 * chatterTable.ts — LA TABLE DES RÉPLIQUES des Spartans, telle que le jeu l'a versée (`cmd/chatter-build`), et ce qu'on en lit.
 *
 * LE FICHIER (`static/sounds/halo_infinite/repliques/repliques.json`) : pour chaque identifiant de son de réplique parlée,
 * son texte dans chaque langue de `config/locales.toml` et la liste des langues dont la voix est versée
 * (`static/sounds/halo_infinite/repliques/<langue>/<son>.ogg`). Le film dit QUEL son est joué par qui et à quel instant ; il
 * ne dit ni le texte ni la variante audio : le texte et la voix d'un son viennent de la même variante, versée par la commande.
 *
 * LA LANGUE DES RÉPLIQUES N'EST PAS UN RÉGLAGE. Texte ET voix suivent la langue de l'interface par la source unique des langues :
 * la langue de l'interface si la table la porte, sinon `FALLBACK_LOCALE` (l'anglais). La voix est celle de la langue du TEXTE :
 * une voix qui manque dans cette langue n'est pas remplacée par celle d'une autre, la réplique reste affichée sans voix (et
 * aucune icône de voix ne la promet).
 *
 * LOGIQUE PURE : aucun React, aucun réseau. Une entrée mal formée est écartée, jamais devinée.
 */
import { FALLBACK_LOCALE, KNOWN_LOCALES, type Locale } from '@/lib/i18n/locale'
import type { LocaleTexts } from '@/lib/i18n/pickText'

/** La forme du fichier versé que ce module sait lire. */
export const CHATTER_TABLE_VERSION = 1

/** Une réplique parlée : son texte par langue et les langues dont la voix est versée. */
export interface ChatterEntry {
  texte: LocaleTexts
  audio: readonly Locale[]
}

/** La table : identifiant du son (hexadécimal, huit chiffres, comme `chatter[].sound`) -> réplique. */
export type ChatterTable = ReadonlyMap<string, ChatterEntry>

/** Une table sans réplique : le film n'en a pas de lisible, ou le fichier n'est pas arrivé. */
export const EMPTY_CHATTER_TABLE: ChatterTable = new Map()

function isKnown(value: unknown): value is Locale {
  return typeof value === 'string' && (KNOWN_LOCALES as readonly string[]).includes(value)
}

/** Lit le fichier versé. Une version inconnue ou un document étranger rend une table vide ; une entrée sans texte est écartée. */
export function parseChatterTable(raw: unknown): ChatterTable {
  if (typeof raw !== 'object' || raw === null) return EMPTY_CHATTER_TABLE
  const doc = raw as { version?: unknown; sons?: unknown }
  if (doc.version !== CHATTER_TABLE_VERSION || typeof doc.sons !== 'object' || doc.sons === null) return EMPTY_CHATTER_TABLE
  const out = new Map<string, ChatterEntry>()
  for (const [id, value] of Object.entries(doc.sons as Record<string, unknown>)) {
    if (typeof value !== 'object' || value === null) continue
    const { texte, audio } = value as { texte?: unknown; audio?: unknown }
    if (typeof texte !== 'object' || texte === null) continue
    const texts: Partial<Record<Locale, string>> = {}
    for (const l of KNOWN_LOCALES) {
      const v = (texte as Record<string, unknown>)[l]
      if (typeof v === 'string' && v !== '') texts[l] = v
    }
    if (Object.keys(texts).length === 0) continue
    out.set(id, { texte: texts, audio: Array.isArray(audio) ? audio.filter(isKnown) : [] })
  }
  return out
}

/**
 * La langue dans laquelle une réplique se lit et s'entend : celle de l'interface si la table la porte, sinon l'anglais ;
 * `null` quand ni l'une ni l'autre n'a de texte (la réplique n'est alors pas affichable).
 */
export function chatterLanguage(entry: ChatterEntry, locale: Locale): Locale | null {
  if (entry.texte[locale]) return locale
  if (entry.texte[FALLBACK_LOCALE]) return FALLBACK_LOCALE
  return null
}

/** Le dossier des voix sous `static/sounds/halo_infinite/`, et le nom du fichier de table à côté. */
const CHATTER_DIR = 'repliques'

/** Le stem du fichier de voix d'un son dans une langue (`soundUrlOf` en fait l'URL). */
export function chatterVoiceStem(sound: string, language: Locale): string {
  return `${CHATTER_DIR}/${language}/${sound}`
}

/** Le stem du fichier de la table (extension `.json` posée par l'appelant). */
export const CHATTER_TABLE_STEM = `${CHATTER_DIR}/repliques`
