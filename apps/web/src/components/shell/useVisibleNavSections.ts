/**
 * useVisibleNavSections — sections de navigation L1 visibles pour le joueur et le titre courants.
 *
 * Source unique du filtrage, partagée par la barre haute (NavL1, son menu mobile) et par la barre
 * latérale de l'application de bureau (SidebarNav) : deux rendus d'une même liste.
 *
 * Gating multi-titre (Phase 5) : on masque les sections / onglets dont la capability n'est pas
 * déclarée par le titre courant. Fail-open (titleCaps null pendant le bootstrap → tout visible)
 * ⇒ NO-OP pour halo_infinite.
 */
import { useSettings } from '@/features/settings/queries'
import { hasCapabilityIn, useTitleCapabilities } from '@/lib/capabilities/capabilities'
import { L1_SECTIONS, type L1Section, type L1Tab } from './navL1Sections'

export function useVisibleNavSections(): L1Section[] {
  const { data: settings } = useSettings()
  const showProgression = settings?.show_progression ?? true
  const titleCaps = useTitleCapabilities()
  const sectionVisible = (s: L1Section) => s.capability == null || hasCapabilityIn(titleCaps, s.capability)
  const tabVisible = (tab: L1Tab) => tab.capability == null || hasCapabilityIn(titleCaps, tab.capability)
  return (showProgression ? L1_SECTIONS : L1_SECTIONS.filter((s) => s.key !== 'ascension'))
    .filter(sectionVisible)
    .map((s) => {
      if (!s.tabs) return s
      // Titre sans citations calculées (Halo 5) : « Citations » (moteur dérivé) →
      // « Commendations » natif — cohérent avec l'en-tête de page. Signal : capability `citations`.
      const tabs = s.tabs.filter(tabVisible).map((tab): L1Tab =>
        !hasCapabilityIn(titleCaps, 'citations') && tab.key === 'citations'
          ? {
              // Commendations natives. On conserve la clé i18n d'origine
              // (`tab_citations` = FR « Citations » / EN « Commendations », terme
              // officiel Halo) et on ne change QUE la clé de tab + le chemin.
              ...tab,
              key: 'commendations',
              path: '/{-$lang}/t/$titleSlug/players/$playerSlug/career/commendations',
            }
          : tab,
      )
      // Si l'onglet servant de landing par défaut est masqué (ex: « Classements »
      // gaté sur world.leaderboard), replier le défaut sur le 1er onglet visible.
      const defaultPath = tabs.some((tab) => tab.path === s.defaultPath)
        ? s.defaultPath
        : (tabs[0]?.path ?? s.defaultPath)
      return { ...s, tabs: tabs.length > 0 ? tabs : undefined, defaultPath }
    })
}
