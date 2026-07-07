/**
 * CareerEncountersSection — adversaires et coéquipiers fréquents.
 * V8b (2026-07-07) — consomme EncounterDTO (shape réelle de /pages/career/encounters :
 * { teammates, enemies, total }). Le champ `encounters_preview` de /pages/career
 * n'existait pas côté contrat Go — la section est désormais alimentée par son
 * endpoint dédié (avg_kda + compteurs coéquipier/adversaire).
 */
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import type { EncounterDTO } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { formatMessage } from '@/lib/i18n/format'
import { careerManifest, type CareerManifestKey } from '@/lib/i18n/generated/career'
import { useAppShellStore } from '@/stores/appShellStore'
import { useCareerEncounters } from './queries'

interface Props {
  playerSlug: string
}

function EncounterTable({ items }: { items: EncounterDTO[] }) {
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
            <th className="pb-2 text-right">{t('career.encounters.col_as_teammate')}</th>
            <th className="pb-2 text-right">{t('career.encounters.col_as_enemy')}</th>
            <th className="pb-2 text-right">{labelOf('kda')}</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {items.map((enc) => (
            <tr key={enc.xuid} className="hover:bg-muted">
              <td className="py-1.5 font-medium text-foreground">{enc.gamertag}</td>
              <td className="py-1.5 text-right text-muted-foreground">{enc.match_count}</td>
              <td className="py-1.5 text-right text-success">{enc.as_teammate}</td>
              <td className="py-1.5 text-right text-destructive">{enc.as_enemy}</td>
              <td className="py-1.5 text-right text-foreground">
                {enc.avg_kda != null ? enc.avg_kda.toFixed(1) : '—'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function CareerEncountersSection({ playerSlug }: Props) {
  const { data, isLoading } = useCareerEncounters(playerSlug)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CareerManifestKey) => formatMessage(careerManifest, key, locale)

  const teammates = data?.teammates ?? []
  const enemies = data?.enemies ?? []
  const items = [...teammates, ...enemies]

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">{t('career.encounters.section_title')}</CardTitle>
          <Badge variant="secondary">
            {items.length} {t('career.encounters.players_suffix')}
          </Badge>
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
          <EncounterTable items={items} />
        )}
      </CardContent>
    </Card>
  )
}
