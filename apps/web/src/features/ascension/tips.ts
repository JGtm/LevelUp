// cross-feature-allow: la feature ascension expose des tips qui pointent vers
// les ancres du glossaire help (source unique des définitions).
/**
 * buildAscensionTips — Source des tips pour le bandeau TipsTicker.
 *
 * Filtre le glossaire help pour récupérer les concepts pertinents à la
 * section Ascension, les convertit en `Tip[]` avec ancre stable vers
 * `/help?tab=glossary#glossary-entry-<slug>`. L'ordre est shuffleé à chaque
 * appel pour varier l'expérience entre les pages.
 */
import type { Tip } from '@/components/ui/tips-ticker'
import { getHelpText, type HelpLocale, type GlossaryEntry } from '@/features/help/i18n'
import { buildGlossaryEntryAnchor } from '@/features/help/GlossaryTab'

// Identifiant du bloc à filtrer : ce string n'est PAS un label métier
// (le titre de section FR et EN sont identiques par choix éditorial).
const ASCENSION_SECTION_TITLE = 'Ascension & Progression'

const SHORT_DEF_MAX_LEN = 180

export function buildAscensionTips(locale: HelpLocale): Tip[] {
  const sections = getHelpText(locale).glossary.sections
  const ascensionSection = sections.find((s) => s.title === ASCENSION_SECTION_TITLE)
  if (!ascensionSection) return []
  const tips: Tip[] = ascensionSection.entries.map(entryToTip)
  return shuffleArray(tips)
}

function entryToTip(entry: GlossaryEntry): Tip {
  return {
    id: entry.term,
    term: entry.term,
    shortDef: truncateOneLine(entry.definition, SHORT_DEF_MAX_LEN),
    href: `/help?tab=glossary#${buildGlossaryEntryAnchor(entry.term)}`,
  }
}

function truncateOneLine(text: string, maxLen: number): string {
  const oneLine = text.replace(/\n+/g, ' ').replace(/\s+/g, ' ').trim()
  if (oneLine.length <= maxLen) return oneLine
  return oneLine.slice(0, maxLen).trimEnd() + '…'
}

function shuffleArray<T>(arr: T[]): T[] {
  const copy = [...arr]
  for (let i = copy.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[copy[i], copy[j]] = [copy[j], copy[i]]
  }
  return copy
}
