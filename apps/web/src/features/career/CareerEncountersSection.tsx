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
import { SortableTh } from '@/components/ui/sortable-th'
import type { EncounterDTO } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { useColumnSort, type SortColumn } from '@/lib/table/useColumnSort'
import { formatMessage } from '@/lib/i18n/format'
import { careerManifest, type CareerManifestKey } from '@/lib/i18n/generated/career'
import { useAppShellStore } from '@/stores/appShellStore'
import { useCareerEncounters } from './queries'

interface Props {
  playerSlug: string
}

type EncounterSortKey = 'gamertag' | 'match_count' | 'as_teammate' | 'as_enemy' | 'avg_kda'

// Valeurs triables : une valeur absente (avg_kda) est toujours rangée en bas quel que soit le sens ;
// gamertag en minuscules pour un tri alpha insensible à la casse. Texte → ascendant au 1er clic ;
// numérique → descendant au 1er clic.
const SORT_COLUMNS: Record<EncounterSortKey, SortColumn<EncounterDTO>> = {
  gamertag: { value: (e) => e.gamertag.toLowerCase(), firstDir: 'asc' },
  match_count: { value: (e) => e.match_count },
  as_teammate: { value: (e) => e.as_teammate },
  as_enemy: { value: (e) => e.as_enemy },
  avg_kda: { value: (e) => e.avg_kda },
}

function EncounterTable({ items }: { items: EncounterDTO[] }) {
  // Phase D plan multi-titres : libellés métier issus du backend TOML, avec
  // repli sur les valeurs FR locales si la réponse manque (le flag serveur qui
  // pouvait éteindre l'endpoint a été retiré le 2026-08-02).
  const { data: fieldMappings } = useFieldMappings()
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CareerManifestKey) => formatMessage(careerManifest, key, locale)
  const labelOf = (key: string): string =>
    fieldMappings?.fields[key]?.label ?? key

  // I16 : tri CLIENT par clic sur les en-têtes. Aucun tri actif par défaut →
  // l'ordre serveur (teammates puis enemies, cf. CareerEncountersSection) reste
  // affiché tant qu'aucun en-tête n'a été cliqué.
  const sort = useColumnSort(items, SORT_COLUMNS)
  const sortedItems = sort.rows

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border text-xs font-medium text-muted-foreground">
            <SortableTh
              label={t('career.encounters.col_player')}
              {...sort.header('gamertag')}
              className="pb-2 text-left"
            />
            <SortableTh
              label={labelOf('total_matches_played')}
              {...sort.header('match_count')}
              className="pb-2 text-right"
              tooltip={t('career.encounters.col_count_tooltip')}
            />
            <SortableTh
              label={t('career.encounters.col_as_teammate')}
              {...sort.header('as_teammate')}
              className="pb-2 text-right"
              tooltip={t('career.encounters.col_as_teammate_tooltip')}
            />
            <SortableTh
              label={t('career.encounters.col_as_enemy')}
              {...sort.header('as_enemy')}
              className="pb-2 text-right"
              tooltip={t('career.encounters.col_as_enemy_tooltip')}
            />
            <SortableTh
              label={labelOf('kda')}
              {...sort.header('avg_kda')}
              className="pb-2 text-right"
              tooltip={t('career.encounters.col_avg_kda_tooltip')}
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
