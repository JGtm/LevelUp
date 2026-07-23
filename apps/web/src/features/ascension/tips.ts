/**
 * buildAscensionTips — Source des tips de JEU pour le bandeau TipsTicker.
 *
 * Les tips sont des conseils de jeu (« comment mieux jouer »), PAS des
 * définitions de l'app : ils proviennent du manifeste `coachingTipsManifest`
 * (catégories Combat / Impact / Objectif / Score / Support / Survie). Chaque
 * tip est rendu sous la forme « Catégorie : conseil » par le TipsTicker.
 *
 * Les définitions de concepts de l'app (Série, Palier, Levier LUSR…) restent
 * accessibles via le glossaire `/help?tab=glossary` — elles ne sont plus
 * mélangées au ticker, qui est désormais 100 % axé jeu.
 *
 * L'ordre est shuffleé à chaque appel et borné à `MAX_TIPS` pour varier
 * l'expérience entre les pages sans saturer le cycle du ticker.
 */
import type { Tip } from '@/components/ui/tips-ticker'
import { coachingTipsManifest } from '@/lib/i18n/generated/coaching_tips'
import type { Locale } from '@/lib/i18n/locale'

const TIPS_MANIFEST = coachingTipsManifest as Record<string, { fr: string; en: string }>

// Clés qui sont de vrais conseils affichables : on exclut `.title` et
// `.related_signals` (méta-données de catégorie, pas des tips).
const TIP_KEY_RE =
  /^coaching_tips\.([a-z]+)\.(ingame|routine|settings|strategic|tactical)\.\d+$/

// Borne le nombre de tips renvoyés par appel (~70 au catalogue) : limite la
// longueur du cycle du ticker et de la liste statique en reduced-motion.
const MAX_TIPS = 14

export function buildAscensionTips(locale: Locale): Tip[] {
  const lang: Locale = locale === 'en' ? 'en' : 'fr'
  const tips: Tip[] = []
  for (const [key, value] of Object.entries(TIPS_MANIFEST)) {
    const match = key.match(TIP_KEY_RE)
    if (!match) continue
    const category = match[1]
    const titleEntry = TIPS_MANIFEST[`coaching_tips.${category}.title`]
    tips.push({
      id: key,
      term: titleEntry ? titleEntry[lang] : category,
      shortDef: normalizeWhitespace(value[lang]),
    })
  }
  return shuffleArray(tips).slice(0, MAX_TIPS)
}

function normalizeWhitespace(text: string): string {
  return text.replace(/\s+/g, ' ').trim()
}

function shuffleArray<T>(arr: T[]): T[] {
  const copy = [...arr]
  for (let i = copy.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[copy[i], copy[j]] = [copy[j], copy[i]]
  }
  return copy
}
