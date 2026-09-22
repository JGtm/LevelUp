/**
 * Garde-rail (CLAUDE.md n°6) : le titre de section ne se réécrit pas à la main.
 *
 * Contexte : le gabarit `text-base font-semibold text-foreground` sur un `h3` vivait dans un
 * helper LOCAL à la page match et était recopié en toutes lettres dans cinq autres endroits
 * (deux sections de Synergies, la section Performance des Contributions, les deux titres des
 * formes retenues, le « Détail des matchs » du détail de session — celui-là en `h2`). Elles
 * ont été migrées sur `components/ui/detail-section.tsx` le 2026-09-22. Une factorisation
 * sans garde-rail re-diverge (leçon du prédicat bot, 8 → 36 copies) : ce test interdit le
 * retour du littéral qui SIGNE le gabarit.
 *
 * CE QUE CE TEST SURVEILLE, ET SEULEMENT ÇA : une balise `<h2>` ou `<h3>` dont la className
 * porte `text-base` ET `font-semibold`, dans `src/features/**`. Le gabarit passe par
 * `DetailSection` / `SectionTitle`.
 *
 * ALLOWLIST DATÉE (2026-09-22), EN RATCHET. Les fichiers listés portaient déjà ces titres au
 * moment de la factorisation et sont HORS du périmètre du lot (home, admin, palmarès,
 * explorateur, aide, média, synthèse, prestige, mentions légales, narration de la page
 * match). Ils ne sont pas tolérés « pour toujours » : le compte ne peut que DESCENDRE.
 * - un fichier absent de l'allowlist qui pose le gabarit = ÉCHEC (nouvelle copie) ;
 * - un compte qui MONTE = ÉCHEC (la dette recroît) ;
 * - un compte qui DESCEND = échec avec le message qui dit la valeur à inscrire.
 *
 * RETIRER UN FICHIER DE L'ALLOWLIST : migrer ses titres sur `SectionTitle` (ou
 * `DetailSection` si le conteneur `space-y-4` convient), puis SUPPRIMER sa ligne ici — pas
 * la passer à 0.
 */
import { describe, it, expect } from 'vitest'

// import.meta.glob (Vite) charge chaque source comme chaîne brute — même mécanique que
// section-card.guard.test.ts, pas de dépendance à node:fs.
const sources = import.meta.glob('/src/features/**/*.{ts,tsx}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

// Ouverture d'un <h2>/<h3> (attributs éventuellement sur plusieurs lignes) portant les deux
// classes du gabarit, dans l'un ou l'autre ordre.
const HEADING_TEMPLATE =
  /<h[23]\b(?![^>]*\/>)[^>]*\b(?:text-base\b[^>]*\bfont-semibold|font-semibold\b[^>]*\btext-base)\b[^>]*>/gs

/** Fichiers tolérés au 2026-09-22, avec leur nombre d'occurrences — ratchet, voir l'en-tête. */
const ALLOWLIST: Record<string, number> = {
  '/src/features/admin/data/AdminDataPage.tsx': 5,
  '/src/features/admin/management/AdminManagementPage.tsx': 4,
  '/src/features/admin/titles/AdminTitlesPage.tsx': 1,
  '/src/features/admin/titles/TitleDetailCards.tsx': 2,
  '/src/features/explorer/ExplorerCombatProfile.tsx': 2,
  '/src/features/explorer/ExplorerTargetProfileCard.tsx': 2,
  '/src/features/help/GlossaryTab.tsx': 1,
  '/src/features/help/ReleaseNotesTab.tsx': 1,
  '/src/features/home/HomeAscensionWidget.tsx': 1,
  '/src/features/home/HomeBattlePassPanel.tsx': 1,
  '/src/features/home/HomeCitationsNearCompletion.tsx': 1,
  '/src/features/home/HomePage.tsx': 5,
  '/src/features/home/HomePrestigeSection.tsx': 1,
  '/src/features/home/HomeRecentPlaylistsCard.tsx': 1,
  '/src/features/home/RecentMediaRail.tsx': 1,
  '/src/features/legal/PrivacyPage.tsx': 1,
  '/src/features/match-view/MatchNarrativeSection.tsx': 1,
  '/src/features/media/MediaAudioConfigButton.tsx': 1,
  '/src/features/media/MediaMatchPicker.tsx': 1,
  '/src/features/palmares/RelationsMomentsSection.tsx': 2,
  '/src/features/palmares/SeasonPassPage.tsx': 2,
  '/src/features/prestige/components/MomentCard.tsx': 1,
  '/src/features/synthesis/SynthesisPage.tsx': 3,
}

function countTemplates(code: string): number {
  return (code.match(HEADING_TEMPLATE) ?? []).length
}

describe('garde-rail gabarit de titre de section (detail-section source unique)', () => {
  it('le glob ne perd pas ses sources à scanner', () => {
    // Un glob qui ne matche plus rien rendrait le test vert pour de mauvaises raisons.
    expect(Object.keys(sources).length).toBeGreaterThan(300)
  })

  it('aucun fichier hors allowlist ne réécrit le gabarit de titre', () => {
    const offenders = Object.entries(sources)
      .filter(([path]) => !/\.test\.tsx?$/.test(path))
      .filter(([path]) => !(path in ALLOWLIST))
      .filter(([, code]) => countTemplates(code) > 0)
      .map(([path]) => path)
    expect(
      offenders,
      `Titre de section à migrer vers SectionTitle / DetailSection (components/ui/detail-section.tsx) : ${offenders.join(', ')}`,
    ).toEqual([])
  })

  it('les fichiers de l allowlist ne gagnent pas de nouvelles copies', () => {
    const risen = Object.entries(ALLOWLIST)
      .map(([path, allowed]) => ({ path, allowed, actual: countTemplates(sources[path] ?? '') }))
      .filter((e) => e.actual > e.allowed)
      .map((e) => `${e.path} : ${e.actual} > ${e.allowed}`)
    expect(risen, `Copies du gabarit en hausse — migrer, pas monter l allowlist : ${risen.join(', ')}`).toEqual([])
  })

  it('l allowlist ne garde pas de ligne périmée (ratchet à resserrer)', () => {
    const loosened = Object.entries(ALLOWLIST)
      .map(([path, allowed]) => ({ path, allowed, actual: countTemplates(sources[path] ?? '') }))
      .filter((e) => e.actual < e.allowed)
      .map((e) => `${e.path} : descendre a ${e.actual}${e.actual === 0 ? ' (ou mieux : supprimer la ligne)' : ''}`)
    expect(loosened, `Allowlist a resserrer : ${loosened.join(', ')}`).toEqual([])
  })
})
