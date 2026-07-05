/**
 * CareerEncountersSection — adversaires et coéquipiers fréquents.
 */
import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import type { CareerEncounter } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { formatMessage } from '@/lib/i18n/format'
import { careerManifest, type CareerManifestKey } from '@/lib/i18n/generated/career'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import { useCareerEncounters } from './queries'

interface Props {
  playerSlug: string
  preview: CareerEncounter[] | null | undefined
}

function WinRate({ wins, total }: { wins: number; total: number }) {
  const pct = total > 0 ? Math.round((wins / total) * 100) : 0
  return (
    <span
      className={
        pct >= 55
          ? 'text-success font-medium'
          : pct <= 40
          ? 'text-destructive font-medium'
          : 'text-foreground'
      }
    >
      {pct} %
    </span>
  )
}

function EncounterTable({ items }: { items: CareerEncounter[] }) {
  // Phase D plan multi-titres : libellés métier issus du backend TOML avec
  // fallback gracieux sur les valeurs FR locales si MULTI_TITLE_API_ENABLED off.
  const { data: fieldMappings } = useFieldMappings()
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CareerManifestKey) => formatMessage(careerManifest, key, locale)
  const labelOf = (key: string): string =>
    fieldMappings?.fields[key]?.label ?? key
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border text-xs font-medium text-muted-foreground">
            <th className="pb-2 text-left">{t('career.encounters.col_player')}</th>
            <th className="pb-2 text-right">{labelOf('total_matches_played')}</th>
            <th className="pb-2 text-right">{labelOf('win_rate')}</th>
            <th className="pb-2 text-right">{t('career.encounters.col_wins')}</th>
            <th className="pb-2 text-right">{t('career.encounters.col_losses')}</th>
            <th className="pb-2 text-right">{t('career.encounters.col_last_encounter')}</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {items.map((enc) => (
            <tr key={enc.encounter_key} className="hover:bg-muted">
              <td className="py-1.5 font-medium text-foreground">{enc.opponent_gamertag}</td>
              <td className="py-1.5 text-right text-muted-foreground">{enc.count_matches}</td>
              <td className="py-1.5 text-right">
                <WinRate wins={enc.wins} total={enc.count_matches} />
              </td>
              <td className="py-1.5 text-right text-success">{enc.wins}</td>
              <td className="py-1.5 text-right text-destructive">{enc.losses}</td>
              <td className="py-1.5 text-right text-xs text-muted-foreground">
                {enc.last_seen_at
                  ? new Date(enc.last_seen_at).toLocaleDateString(intlLocale(locale))
                  : '—'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function CareerEncountersSection({ playerSlug, preview }: Props) {
  const [showAll, setShowAll] = useState(false)
  const { data, isLoading } = useCareerEncounters(playerSlug, showAll)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CareerManifestKey) => formatMessage(careerManifest, key, locale)

  const items = showAll && data ? data.items : (preview ?? [])

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">{t('career.encounters.section_title')}</CardTitle>
          <Badge variant="secondary">{items.length} joueur{items.length !== 1 ? 's' : ''}</Badge>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="flex justify-center py-6">
            <Spinner size="sm" label={t('career.encounters.loading')} />
          </div>
        ) : items.length === 0 ? (
          <p className="py-6 text-center text-sm text-muted-foreground">{t('career.encounters.empty')}</p>
        ) : (
          <>
            <EncounterTable items={items} />
            {!showAll && (preview?.length ?? 0) > 0 && (
              <div className="mt-4 flex justify-center">
                <Button variant="outline" size="sm" onClick={() => setShowAll(true)}>
                  {t('career.encounters.see_all')}
                </Button>
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}
