/**
 * i18n — Dictionnaire de l'aide (glossaire), un fichier par langue (`fr.ts`, `en.ts`) pour un
 * type déclaré une fois (`contract.ts`). Ajouter une langue : un fichier, une ligne ici.
 */
import type { Locale } from '@/lib/i18n/locale'
import type { HelpText } from './contract'
import { EN_TEXT } from './en'
import { FR_TEXT } from './fr'

export type { GlossaryEntry, GlossarySection, HelpTab, HelpText } from './contract'

/**
 * Baseline PV-pour-tuer par défaut (Halo Infinite : 90 vie + 135 bouclier).
 * Utilisé comme repli quand le titre courant n'expose pas son barème. Doit rester
 * aligné avec `games.DefaultEffectiveHpToKill` côté backend.
 */
export const DEFAULT_EFFECTIVE_HP_TO_KILL = 225

/**
 * Jeton remplacé par le barème PV-pour-tuer du titre courant dans le copy combat
 * (rendement / résistance). Toutes les valeurs réelles sont à 3 chiffres (115–225),
 * ce qui préserve l'alignement des tableaux ASCII après substitution.
 */
const HP_TOKEN = '{{HP}}'

const TEXT: Record<Locale, HelpText> = { fr: FR_TEXT, en: EN_TEXT }

/**
 * Injecte le barème PV-pour-tuer du titre courant dans le copy combat. Le copy
 * source porte le jeton `{{HP}}` partout où la constante est title-aware (rendement,
 * résistance, et leurs déclinaisons escouade) ; on le remplace par la valeur résolue
 * côté backend (`TitleSummary.effective_hp_to_kill`). Les repères élite mondiale
 * (0,90 / 1,65) restent calibrés Halo Infinite (recalibration par titre différée) et
 * ne sont pas tokenisés.
 */
function withDamageBaseline(text: HelpText, effectiveHpToKill: number): HelpText {
  const hp = String(Math.round(effectiveHpToKill))
  const sub = (s: string): string => s.split(HP_TOKEN).join(hp)
  return {
    ...text,
    glossary: {
      ...text.glossary,
      sections: text.glossary.sections.map((section) => ({
        ...section,
        entries: section.entries.map((entry) => ({
          ...entry,
          definition: sub(entry.definition),
          formula: entry.formula != null ? sub(entry.formula) : entry.formula,
          example: entry.example != null ? sub(entry.example) : entry.example,
        })),
      })),
    },
  }
}

/**
 * Retourne le glossaire localisé. `effectiveHpToKill` (PV-pour-tuer du titre courant,
 * résolu depuis le bootstrap) rend le copy combat title-aware ; à défaut, repli sur
 * le barème Halo Infinite.
 */
export function getHelpText(
  locale: Locale,
  effectiveHpToKill: number = DEFAULT_EFFECTIVE_HP_TO_KILL,
): HelpText {
  return withDamageBaseline(TEXT[locale], effectiveHpToKill)
}
