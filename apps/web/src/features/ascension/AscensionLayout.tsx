/**
 * AscensionLayout — layout parent à 4 onglets de la section Ascension.
 *
 * Architecture (parcours utilisateur : qui je suis → ce que je vise →
 * comment progresser → ce que j'ai accompli) :
 *   /ascension               → tab "Profil" (index)
 *   /ascension/objectifs     → tab "Objectifs" (couche Prestige)
 *   /ascension/coaching      → tab "Entraînement"
 *   /ascension/realisations  → tab "Réalisations"
 *
 * Le layout fournit le header (H1 + sous-titre), le bandeau TipsTicker
 * partagé, et la barre d'onglets. Le contenu de chaque tab est rendu
 * via <Outlet />.
 *
 * Refonte Ascension UX (2026-07) : restructuration 3 → 4 onglets (DEC-3).
 */
import { useMemo } from 'react'
import { Link, Outlet, useMatchRoute, useParams } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { TipsTicker } from '@/components/ui/tips-ticker'
import { buildAscensionTips } from './tips'
import { getAscensionText } from './i18n'

function LightbulbIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="12"
      height="12"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M9 18h6" />
      <path d="M10 22h4" />
      <path d="M12 2a7 7 0 0 0-4 12.7c.7.6 1 1.5 1 2.3h6c0-.8.3-1.7 1-2.3A7 7 0 0 0 12 2z" />
    </svg>
  )
}

export function AscensionLayout() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = useAppShellStore((s) => s.locale)
  const t = getAscensionText(locale)
  const matchRoute = useMatchRoute()

  const tips = useMemo(() => buildAscensionTips(locale === 'en' ? 'en' : 'fr'), [locale])

  const profileRoute = '/players/$playerSlug/ascension' as const
  const objectivesRoute = '/players/$playerSlug/ascension/objectifs' as const
  const coachingRoute = '/players/$playerSlug/ascension/coaching' as const
  const realisationsRoute = '/players/$playerSlug/ascension/realisations' as const
  const isObjectives = !!matchRoute({ to: objectivesRoute })
  const isCoaching = !!matchRoute({ to: coachingRoute })
  const isRealisations = !!matchRoute({ to: realisationsRoute })
  const isProfile = !isObjectives && !isCoaching && !isRealisations

  return (
    <main className="container mx-auto max-w-6xl space-y-6 px-4 py-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold">{t.pageTitle}</h1>
        <p className="text-sm text-muted-foreground">{t.pageSubtitle}</p>
      </header>

      <TipsTicker
        tips={tips}
        ariaLabel={t.tipsTickerAriaLabel}
        leadingIcon={<LightbulbIcon />}
      />

      <nav
        role="tablist"
        aria-label={t.tabsAriaLabel}
        className="flex border-b border-border"
      >
        <Link
          to={profileRoute}
          params={{ playerSlug }}
          role="tab"
          aria-selected={isProfile}
          className={tabClass(isProfile)}
        >
          {t.tabProfile}
        </Link>
        <Link
          to={objectivesRoute}
          params={{ playerSlug }}
          role="tab"
          aria-selected={isObjectives}
          className={tabClass(isObjectives)}
        >
          {t.tabObjectives}
        </Link>
        <Link
          to={coachingRoute}
          params={{ playerSlug }}
          role="tab"
          aria-selected={isCoaching}
          className={tabClass(isCoaching)}
        >
          {t.tabCoaching}
        </Link>
        <Link
          to={realisationsRoute}
          params={{ playerSlug }}
          role="tab"
          aria-selected={isRealisations}
          className={tabClass(isRealisations)}
        >
          {t.tabRealisations}
        </Link>
      </nav>

      <Outlet />
    </main>
  )
}

function tabClass(active: boolean): string {
  return [
    'border-b-2 px-4 py-2 text-sm font-medium transition-colors',
    active
      ? 'border-primary text-foreground'
      : 'border-transparent text-muted-foreground hover:text-foreground',
  ].join(' ')
}
