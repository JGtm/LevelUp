/**
 * SectionHeader — en-tête de page des sections L1, posé UNE fois par le layout joueur au-dessus du
 * contenu : titre (libellé de la section), chapeau, onglets de la section.
 *
 * Les sections viennent de `L1_SECTIONS` (filtrées par capacité de titre par `useVisibleNavSections`),
 * source unique que partagent aussi la barre haute, le tiroir mobile et la barre latérale. Rien à
 * rendre pour :
 * - l'accueil : sa bannière d'identité porte déjà le titre de la page ;
 * - une section dont le layout rend lui-même son en-tête (`headerInLayout`, onglets qui dépendent de
 *   ses données) ;
 * - une page hors section (détail d'un match, notifications...).
 */
import { useRouterState } from '@tanstack/react-router'
import { PageHeader } from '@/components/foundations/PageHeader'
import type { PageHeaderTab } from '@/components/foundations/pageHeaderLogic'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { useAppShellStore } from '@/stores/appShellStore'
import { sectionWithHeader } from './sectionHeaderLogic'
import { useVisibleNavSections } from './useVisibleNavSections'

export function SectionHeader() {
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const sections = useVisibleNavSections()
  const locale = useAppShellStore((s) => s.locale)
  const section = sectionWithHeader(sections, pathname)
  if (!section?.leadKey) return null
  const t = (key: CommonManifestKey, values?: Record<string, string>) =>
    formatMessage(commonManifest, key, locale, values)
  const title = t(section.labelKey)
  const tabs: PageHeaderTab[] | undefined = section.tabs?.map((tab) => ({
    key: tab.key,
    label: t(tab.labelKey),
    to: tab.path,
    capability: tab.capability,
  }))
  return (
    <PageHeader
      title={title}
      lead={t(section.leadKey)}
      tabs={tabs}
      tabsAriaLabel={t('common.nav.section_tabs_aria', { section: title })}
    />
  )
}
