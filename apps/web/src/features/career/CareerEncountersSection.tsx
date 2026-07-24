/**
 * CareerEncountersSection — adversaires et coéquipiers fréquents.
 * V8b (2026-07-07) — consomme EncounterDTO (shape réelle de /pages/career/encounters :
 * { teammates, enemies, total }). Le champ `encounters_preview` de /pages/career
 * n'existait pas côté contrat Go — la section est désormais alimentée par son
 * endpoint dédié (avg_kda + compteurs coéquipier/adversaire).
 */
import { useMemo, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import { SortableTh } from '@/components/ui/sortable-th'
import type { EncounterDTO } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { formatMessage } from '@/lib/i18n/format'
import { careerManifest, type CareerManifestKey } from '@/lib/i18n/generated/career'
import { useAppShellStore } from '@/stores/appShellStore'
import { useCareerEncounters } from './queries'

interface Props {
  playerSlug: string
}

type EncounterSortKey = 'gamertag' | 'match_count' | 'as_teammate' | 'as_enemy' | 'avg_kda'

// Valeur brute triable (nulls coalescés en `null` explicite — toujours rangés en
// bas quel que soit le sens, cf. compareEncounters). gamertag en minuscules pour
// un tri alpha insensible à la casse.
function encounterRawValue(e: EncounterDTO, key: EncounterSortKey): string | number | null {
  switch (key) {
    case 'gamertag':
      return e.gamertag.toLowerCase()
    case 'match_count':
      return e.match_count
    case 'as_teammate':
      return e.as_teammate
    case 'as_enemy':
      return e.as_enemy
    case 'avg_kda':
      return e.avg_kda
  }
}

function compareEncounters(a: EncounterDTO, b: EncounterDTO, key: EncounterSortKey, dir: 'asc' | 'desc'): number {
  const va = encounterRawValue(a, key)
  const vb = encounterRawValue(b, key)
  if (va == null && vb == null) return 0
  if (va == null) return 1
  if (vb == null) return -1
  const cmp = typeof va === 'string' && typeof vb === 'string' ? va.localeCompare(vb, undefined, { numeric: true, sensitivity: 'base' }) : (va as number) - (vb as number)
  return dir === 'asc' ? cmp : -cmp
}

function EncounterTable({ items }: { items: EncounterDTO[] }) {
  // Phase D plan multi-titres : libellés métier issus du backend TOML avec
  // fallback gracieux sur les valeurs FR locales si MULTI_TITLE_API_ENABLED off.
  const { data: fieldMappings } = useFieldMappings()
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CareerManifestKey) => formatMessage(careerManifest, key, locale)
  const labelOf = (key: string): string =>
    fieldMappings?.fields[key]?.label ?? key

  // I16 : tri CLIENT par clic sur les en-têtes. Aucun tri actif par défaut →
  // l'ordre serveur (teammates puis enemies, cf. CareerEncountersSection) reste
  // affiché tant qu'aucun en-tête n'a été cliqué.
  const [sortKey, setSortKey] = useState<EncounterSortKey | null>(null)
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc')
  function toggleSort(key: EncounterSortKey) {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      // Texte → ascendant au 1er clic ; numérique → descendant au 1er clic.
      setSortDir(key === 'gamertag' ? 'asc' : 'desc')
    }
  }
  const sortedItems = useMemo(() => {
    if (!sortKey) return items
    return [...items].sort((a, b) => compareEncounters(a, b, sortKey, sortDir))
  }, [items, sortKey, sortDir])

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border text-xs font-medium text-muted-foreground">
            <SortableTh
              label={t('career.encounters.col_player')}
              active={sortKey === 'gamertag'}
              dir={sortDir}
              onClick={() => toggleSort('gamertag')}
              className="pb-2 text-left"
            />
            <SortableTh
              label={labelOf('total_matches_played')}
              active={sortKey === 'match_count'}
              dir={sortDir}
              onClick={() => toggleSort('match_count')}
              className="pb-2 text-right"
            />
            <SortableTh
              label={t('career.encounters.col_as_teammate')}
              active={sortKey === 'as_teammate'}
              dir={sortDir}
              onClick={() => toggleSort('as_teammate')}
              className="pb-2 text-right"
            />
            <SortableTh
              label={t('career.encounters.col_as_enemy')}
              active={sortKey === 'as_enemy'}
              dir={sortDir}
              onClick={() => toggleSort('as_enemy')}
              className="pb-2 text-right"
            />
            <SortableTh
              label={labelOf('kda')}
              active={sortKey === 'avg_kda'}
              dir={sortDir}
              onClick={() => toggleSort('avg_kda')}
              className="pb-2 text-right"
            />
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {sortedItems.map((enc) => (
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
