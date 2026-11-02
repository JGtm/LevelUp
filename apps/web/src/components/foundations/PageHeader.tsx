/**
 * PageHeader — en-tête de page UNIQUE des sections de l'application (site hébergé et application
 * de bureau, même rendu).
 *
 * Composition, de haut en bas : titre (h1, UN par page), chapeau (une phrase), puis la ligne
 * d'onglets de la section.
 *
 * Aucun saut de page : la zone titre + chapeau a une hauteur MINIMALE fixe, et la ligne d'onglets
 * (hauteur fixe, trait de séparation) est RÉSERVÉE même quand la section n'a pas d'onglets
 * (Médias, Explorer : ligne vide). Deux sections successives posent donc leur premier bloc de
 * contenu à la même ordonnée.
 *
 * Quand la section a des onglets, la ligne est collante (`top-0`, au-dessus du contenu qui défile) :
 * un bandeau collant posé sous elle se décale de sa hauteur (`STICKY_BELOW_TABS_CLASS`). Elle reste
 * sous la barre de progression, les tiroirs et les toasts (`z-30` contre `z-50`). La ligne est un
 * frère de la zone titre, pas son enfant : collante dans l'en-tête seul, elle partirait avec lui.
 *
 * L'accueil n'utilise pas ce composant : sa bannière d'identité (joueur, rang, indicateurs) porte
 * déjà le titre de la page, un second en-tête la doublerait.
 */
import { useMemo } from 'react'
import { Link, useParams, useRouterState } from '@tanstack/react-router'
import { hasCapabilityIn, useTitleCapabilities } from '@/lib/capabilities/capabilities'
import { playerRelativePath, useTitleSlug } from '@/lib/title-routing'
import type { PageHeaderTab } from './pageHeaderLogic'
import { activeTabKey, PAGE_HEADER_TABS_HEIGHT_CLASS, PAGE_HEADER_TITLE_MIN_HEIGHT_CLASS } from './pageHeaderLogic'
import { edgeFadeMask, useTabStripScroll } from './useTabStripScroll'

export interface PageHeaderProps {
  /** Titre de la page : libellé de la section. */
  title: string
  /** Chapeau : une phrase factuelle qui dit ce que la section montre. */
  lead: string
  /** Onglets de la section, rendus sous le titre. */
  tabs?: readonly PageHeaderTab[]
  /** Nom accessible de la barre d'onglets. */
  tabsAriaLabel?: string
}

function TabStrip({ tabs, ariaLabel }: { tabs: readonly PageHeaderTab[]; ariaLabel?: string }) {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug?: string }
  const titleSlug = useTitleSlug()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const activeKey = activeTabKey(playerRelativePath(pathname), tabs)
  const { ref, edges, onScroll } = useTabStripScroll(activeKey ?? '')
  return (
    <nav
      ref={ref}
      aria-label={ariaLabel}
      className={`relative flex ${PAGE_HEADER_TABS_HEIGHT_CLASS} items-stretch overflow-x-auto border-b border-border [scrollbar-width:none] [&::-webkit-scrollbar]:hidden`}
      style={edgeFadeMask(edges)}
      onScroll={onScroll}
    >
      {tabs.map((tab) => {
        const active = tab.key === activeKey
        return (
          <Link
            key={tab.key}
            to={tab.to}
            params={{ titleSlug, playerSlug: playerSlug ?? '' }}
            // Correspondance exacte : par défaut le routeur marque actif tout préfixe de chemin et
            // pose `aria-current` sur l'onglet racine d'une sous-page ; l'onglet actif est décidé par
            // `activeTabKey`, un seul.
            activeOptions={{ exact: true }}
            aria-current={active ? 'page' : undefined}
            className={[
              'flex shrink-0 items-center whitespace-nowrap border-b-2 px-4 text-sm font-medium transition-colors',
              active
                ? 'border-primary text-foreground'
                : 'border-transparent text-muted-foreground hover:border-border hover:text-foreground',
            ].join(' ')}
          >
            {tab.label}
          </Link>
        )
      })}
    </nav>
  )
}

export function PageHeader({ title, lead, tabs, tabsAriaLabel }: PageHeaderProps) {
  const titleCaps = useTitleCapabilities()
  const visibleTabs = useMemo(
    () => (tabs ?? []).filter((tab) => tab.capability == null || hasCapabilityIn(titleCaps, tab.capability)),
    [tabs, titleCaps],
  )
  return (
    <>
      <header className="px-6 pt-6">
        <div className={`${PAGE_HEADER_TITLE_MIN_HEIGHT_CLASS} min-w-0`}>
          <h1 className="text-xl font-bold text-foreground">{title}</h1>
          <p className="text-[13px] leading-5 text-muted-foreground">{lead}</p>
        </div>
      </header>
      {visibleTabs.length > 0 ? (
        <div className="sticky top-0 z-30 bg-background px-6">
          <TabStrip tabs={visibleTabs} ariaLabel={tabsAriaLabel} />
        </div>
      ) : (
        <div className="px-6" aria-hidden="true">
          <div className={`${PAGE_HEADER_TABS_HEIGHT_CLASS} border-b border-border`} />
        </div>
      )}
    </>
  )
}
