import { useAppShellStore } from '@/stores/appShellStore'
import type { Locale } from '@/lib/i18n/locale'

/**
 * useLangParam — locale ACTIVE, pour renseigner le param `lang` des liens
 * title-scoped `/{-$lang}/t/…` afin d'ÉMETTRE le segment de langue par défaut (I10).
 *
 * Contexte : le segment `{-$lang}` est OPTIONNEL (D-1) ; sans valeur passée dans
 * `params`, TanStack l'omet au premier lien (URL `/t/…` sans langue) puis HÉRITE du
 * segment courant sur les liens suivants (langSegmentInheritance.test). Rien
 * n'émettait donc le segment par défaut : la langue restait invisible dans l'URL.
 * Ce hook — jumeau de `useTitleSlug()` — fournit la locale active pour l'injecter aux
 * points CENTRAUX d'entrée dans le sous-arbre title-scoped (index `/`, réalignement
 * Settings, backstop du layout titre). Le segment posé, il est ensuite
 * HÉRITÉ par toute la navigation interne sans toucher aux ~100 call-sites de liens.
 *
 * Source = store (`locale`), TOUJOURS une `Locale` valide (défaut `'fr'`). Le layout
 * `t/$titleSlug` garantit la convergence store↔segment (réconciliation locale←segment,
 * 5a) → cette valeur égale le segment `lang` de l'URL sur les pages title-scoped qui
 * en portent un.
 */
export function useLangParam(): Locale {
  return useAppShellStore((s) => s.locale)
}
