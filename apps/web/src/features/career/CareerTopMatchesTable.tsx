/**
 * CareerTopMatchesTable — tableau des meilleurs/pires matchs.
 * A2/A3 NATIVE_COMPONENTS — colonnes K/D/A, badge typé, clic → Match View.
 * V8b (2026-07-07) — consomme TopMatchDTO (shape réelle de l'endpoint
 * /pages/career/top-matches : best_matches / worst_matches). Les listes best/worst
 * sont déjà séparées côté backend — le composant reçoit un tableau prêt à rendre.
 */
import { useMemo, useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { SortableTh } from '@/components/ui/sortable-th'
import type { TopMatchDTO } from '@/lib/api/types'
import { tokenCssVar } from '@/lib/accessibility'
import { outcomeKey } from '@/lib/outcome-color'
import { formatDate, intlLocale as toIntlLocale } from '@/lib/formatters'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import { buildWaypointMatchUrl, waypointLogoSrc } from '@/lib/match-nav/waypointUrl'
import { formatMessage } from '@/lib/i18n/format'
import { careerManifest, type CareerManifestKey } from '@/lib/i18n/generated/career'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'
import { useCapability } from '@/lib/capabilities/capabilities'

interface Props {
  items: TopMatchDTO[]
  /** 'best' | 'worst' | undefined — pilote le titre par défaut (les listes sont
   *  déjà filtrées côté backend, ce composant ne re-filtre pas). */
  variant?: 'best' | 'worst'
  title?: string
  playerSlug?: string
}

type TopMatchSortKey = 'start_time' | 'map_mode' | 'kills' | 'deaths' | 'kda' | 'performance_score' | 'outcome_code'

// Valeur brute triable (I16). `map_mode` trie sur la carte (map_ui), pas le
// libellé "Carte · Mode" affiché. Nulls coalescés en `null` explicite → rangés
// en bas quel que soit le sens (cf. compareTopMatches).
function topMatchRawValue(m: TopMatchDTO, key: TopMatchSortKey): string | number | null {
  switch (key) {
    case 'start_time':
      return m.start_time ? Date.parse(m.start_time) : null
    case 'map_mode':
      return (m.map_ui ?? '').toLowerCase()
    case 'kills':
      return m.kills
    case 'deaths':
      return m.deaths
    case 'kda':
      return m.kda
    case 'performance_score':
      return m.performance_score
    case 'outcome_code':
      return m.outcome_code
  }
}

function compareTopMatches(a: TopMatchDTO, b: TopMatchDTO, key: TopMatchSortKey, dir: 'asc' | 'desc'): number {
  const va = topMatchRawValue(a, key)
  const vb = topMatchRawValue(b, key)
  if (va == null && vb == null) return 0
  if (va == null) return 1
  if (vb == null) return -1
  const cmp = typeof va === 'string' && typeof vb === 'string' ? va.localeCompare(vb, undefined, { numeric: true, sensitivity: 'base' }) : (va as number) - (vb as number)
  return dir === 'asc' ? cmp : -cmp
}

/**
 * Variante de badge (pill) selon le CODE d'outcome — jamais sur le label
 * localisé (CR C2 : `includes('victoire')` cassait tous les badges en EN).
 */
function outcomeBadgeVariant(code: number | null | undefined): 'success' | 'destructive' | 'secondary' {
  switch (outcomeKey(code ?? 0)) {
    case 'win':
      return 'success'
    case 'loss':
      return 'destructive'
    default:
      return 'secondary'
  }
}

export function CareerTopMatchesTable({ items, variant, title, playerSlug: slugProp }: Props) {
  const params = useParams({ strict: false }) as { playerSlug?: string }
  const playerSlug = slugProp ?? params.playerSlug ?? ''
  const navigateToMatch = useNavigateToMatch(playerSlug)
  const locale = useAppShellStore((s) => s.locale)
  const intlLocale = toIntlLocale(locale)
  const t = (key: CareerManifestKey) => formatMessage(careerManifest, key, locale)
  // Colonne « Ouvrir sur Halo Waypoint » (I19) : gating par capability (absente
  // pour Halo 5) ET par préférence LOCALE (Apparence → « Colonne Halo Waypoint
  // sur les listes de matchs », défaut ON).
  const waypointCapability = useCapability('waypoint_match_url')
  const showWaypointColumnPref = useSettingsDraftStore((s) => s.localUiPrefs.showWaypointColumn)
  const showWaypoint = waypointCapability && showWaypointColumnPref
  const theme = useSettingsDraftStore((s) => s.localUiPrefs.theme)

  const defaultTitle =
    variant === 'worst'
      ? t('career.top_matches.default_title_worst')
      : variant === 'best'
        ? t('career.top_matches.default_title_best')
        : t('career.top_matches.default_title_neutral')

  // I16 : tri CLIENT par clic sur les en-têtes. Aucun tri actif par défaut →
  // l'ordre serveur (liste curée best/worst, cf. CareerRepo.GetHighlightMatchIDs)
  // reste affiché tant qu'aucun en-tête n'a été cliqué.
  const [sortKey, setSortKey] = useState<TopMatchSortKey | null>(null)
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc')
  function toggleSort(key: TopMatchSortKey) {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir(key === 'map_mode' ? 'asc' : 'desc')
    }
  }
  const sortedItems = useMemo(() => {
    if (!sortKey) return items
    return [...items].sort((a, b) => compareTopMatches(a, b, sortKey, sortDir))
  }, [items, sortKey, sortDir])

  function goToMatch(matchId: string) {
    navigateToMatch(matchId, {
      source: 'history',
      matchIds: items.map((m) => m.match_id),
      contextDescriptor: { kind: 'top_matches' },
    })
  }

  if (items.length === 0) {
    return null
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title ?? defaultTitle}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-xs font-medium text-muted-foreground">
                {/* Index de ligne (position d'affichage) : jamais triable — se
                    renumérote naturellement selon l'ordre courant. */}
                <th className="pb-2 text-left">{t('career.top_matches.col_index')}</th>
                {/* Colonne Waypoint (I19) : jamais triable, comme partout ailleurs. */}
                {showWaypoint && <th className="pb-2 text-left" />}
                <SortableTh
                  label={t('career.top_matches.col_date')}
                  active={sortKey === 'start_time'}
                  dir={sortDir}
                  onClick={() => toggleSort('start_time')}
                  className="pb-2 text-left"
                />
                <SortableTh
                  label={t('career.top_matches.col_map_mode')}
                  active={sortKey === 'map_mode'}
                  dir={sortDir}
                  onClick={() => toggleSort('map_mode')}
                  className="pb-2 text-left"
                />
                <SortableTh
                  label={t('career.top_matches.col_kills_short')}
                  active={sortKey === 'kills'}
                  dir={sortDir}
                  onClick={() => toggleSort('kills')}
                  className="pb-2 text-right"
                />
                <SortableTh
                  label={t('career.top_matches.col_deaths_short')}
                  active={sortKey === 'deaths'}
                  dir={sortDir}
                  onClick={() => toggleSort('deaths')}
                  className="pb-2 text-right"
                />
                <SortableTh
                  label={t('career.top_matches.col_kd')}
                  active={sortKey === 'kda'}
                  dir={sortDir}
                  onClick={() => toggleSort('kda')}
                  className="pb-2 text-right"
                />
                <SortableTh
                  label={t('career.top_matches.col_score')}
                  active={sortKey === 'performance_score'}
                  dir={sortDir}
                  onClick={() => toggleSort('performance_score')}
                  className="pb-2 text-right"
                />
                <SortableTh
                  label={t('career.top_matches.col_outcome')}
                  active={sortKey === 'outcome_code'}
                  dir={sortDir}
                  onClick={() => toggleSort('outcome_code')}
                  className="pb-2 text-right"
                />
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {sortedItems.map((m, idx) => (
                <tr
                  key={m.match_id}
                  className="cursor-pointer transition-colors hover:bg-accent"
                  onClick={() => goToMatch(m.match_id)}
                >
                  <td className="py-1.5 text-muted-foreground font-mono text-xs">{idx + 1}</td>
                  {showWaypoint && (
                    <td className="py-1.5">
                      <a
                        href={buildWaypointMatchUrl(playerSlug, m.match_id)}
                        target="_blank"
                        rel="noopener noreferrer"
                        onClick={(e) => e.stopPropagation()}
                        aria-label={t('career.top_matches.col_waypoint_aria')}
                        title={t('career.top_matches.col_waypoint_aria')}
                        className="group flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
                      >
                        <img
                          src={waypointLogoSrc(theme)}
                          alt=""
                          aria-hidden
                          className="h-4 w-4 opacity-60 group-hover:opacity-100 transition-opacity"
                        />
                      </a>
                    </td>
                  )}
                  <td className="py-1.5 text-muted-foreground whitespace-nowrap">
                    {formatDate(m.start_time, intlLocale, { dateStyle: 'short' }, '—')}
                  </td>
                  <td className="py-1.5">
                    <span className="font-medium text-foreground">{m.map_ui ?? '—'}</span>
                    {m.mode_ui && (
                      <span className="ml-1 text-xs text-muted-foreground">· {m.mode_ui}</span>
                    )}
                  </td>
                  <td className="py-1.5 text-right font-mono" style={{ color: tokenCssVar('perf-tier-2') }}>
                    {m.kills}
                  </td>
                  <td className="py-1.5 text-right font-mono" style={{ color: tokenCssVar('divergent-neg') }}>
                    {m.deaths}
                  </td>
                  <td className="py-1.5 text-right font-mono text-foreground">
                    {m.kda != null ? m.kda.toFixed(1) : '—'}
                  </td>
                  <td className="py-1.5 text-right text-muted-foreground">
                    {m.performance_score != null ? m.performance_score.toFixed(1) : '—'}
                  </td>
                  <td className="py-1.5 text-right">
                    {m.outcome_label && (
                      <Badge variant={outcomeBadgeVariant(m.outcome_code)}>
                        {m.outcome_label}
                      </Badge>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  )
}
