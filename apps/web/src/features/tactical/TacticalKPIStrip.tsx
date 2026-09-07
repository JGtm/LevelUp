/**
 * TacticalKPIStrip.tsx — Bande de KPI pour l'onglet Tactique.
 *
 * Affiche : matchs retenus, couverture, morts en isolement, échange.
 */
import { useMemo } from 'react'
import { KPIStrip, type KPICardData } from '@/components/layout/KPIStrip'
import { getTacticalText, type TacticalLocale } from './i18n'

export interface TacticalKPIStripProps {
  locale: TacticalLocale
  matchsRetained: number
  matchsFiltered: number
  coverage: number // percentage 0-100
  tradeRate: number // percentage 0-100
  isolatedDeathRate: number // percentage 0-100
  source: 'base' | 'artefact'
  isLoading?: boolean
}

export function TacticalKPIStrip({
  locale,
  matchsRetained,
  matchsFiltered,
  coverage,
  tradeRate,
  isolatedDeathRate,
  source,
  isLoading = false,
}: TacticalKPIStripProps) {
  const text = getTacticalText(locale)
  const sourceLabel = useMemo(() => {
    return source === 'base'
      ? locale === 'fr'
        ? 'base partagée'
        : 'shared database'
      : locale === 'fr'
        ? 'artefacts cuits'
        : 'cooked artifacts'
  }, [source, locale])

  const cards = useMemo((): KPICardData[] => {
    return [
      {
        id: 'matches-retained',
        label: text.matchesRetained,
        primary: String(matchsRetained),
        secondary: `sur ${matchsFiltered} filtrés`,
      },
      {
        id: 'coverage',
        label: text.coverage,
        primary: `${Math.round(coverage)}%`,
        secondary: sourceLabel,
      },
      {
        id: 'isolated-deaths',
        label: text.isolatedDeaths,
        primary: `${Math.round(isolatedDeathRate)}%`,
        secondary: locale === 'fr' ? 'équipier > 25 m' : 'teammate > 25 m',
      },
      {
        id: 'trade',
        label: text.tradeAfterDeath,
        primary: `${Math.round(tradeRate)}%`,
        secondary: locale === 'fr' ? 'sous 5 s' : 'within 5 s',
      },
    ]
  }, [matchsRetained, matchsFiltered, coverage, tradeRate, isolatedDeathRate, sourceLabel, locale, text])

  if (isLoading) {
    const skeletonCards: KPICardData[] = Array.from({ length: 4 }).map((_, i) => ({
      id: `skeleton-${i}`,
      label: '',
      primary: '—',
    }))
    return <KPIStrip cards={skeletonCards} className="opacity-50" />
  }

  return <KPIStrip cards={cards} />
}
