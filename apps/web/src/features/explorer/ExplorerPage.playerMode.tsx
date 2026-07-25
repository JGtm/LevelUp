/**
 * ExplorerPage — bloc "Mode Joueur".
 *
 * Découpé depuis ExplorerPage.tsx (audit #6 god-file split).
 * Contenu : GamertagSearchInput + badges encounter + briefing + tables ally/enemy
 * + heatmap d'activité commune.
 */
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { GamertagSearchInput } from './GamertagSearchInput'
import { ExplorerMatchesTable } from './ExplorerMatchesTable'
import { normalizeExplorerTableRows } from './explorerTableRows'
import { ExplorerEncounterBriefing } from './ExplorerEncounterBriefing'
import { ExplorerActivityHeatmapChart } from './ExplorerActivityHeatmapChart'
import { ExplorerTargetProfileCard } from './ExplorerTargetProfileCard'
import { ExplorerCombatProfile } from './ExplorerCombatProfile'
import { NarrativeBadge } from '@/components/feedback/NarrativeBadge'
import { tokenVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { formatMessage } from '@/lib/i18n/format'
import { squadManifest, type SquadManifestKey } from '@/lib/i18n/generated/squad'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type {
  ExplorerMatchesQueryResponse,
  ExplorerPlayerQueryResponse,
  MatchEncounterBadge,
} from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

function isEncounterSemanticToken(s: string): s is SemanticToken {
  return s.startsWith('narrative-') || s.startsWith('outcome-') || s.startsWith('perf-')
}

function renderEncounterBadges(badges: MatchEncounterBadge[], locale: string) {
  const manifestLocale: Locale = locale === 'en' ? 'en' : 'fr'
  const sqT = (key: SquadManifestKey, values?: Record<string, string | number>) =>
    formatMessage(squadManifest, key, manifestLocale, values)
  return badges.map((badge, i) => {
    const labelKey = badge.label_key as SquadManifestKey
    const ordinal =
      badge.detail && typeof badge.detail['ordinal'] === 'number'
        ? (badge.detail['ordinal'] as number)
        : undefined
    const label = ordinal !== undefined ? sqT(labelKey, { ordinal }) : sqT(labelKey)
    const colorVar = isEncounterSemanticToken(badge.color_token)
      ? tokenVar(badge.color_token as SemanticToken)
      : undefined
    return <NarrativeBadge key={i} label={label} colorVar={colorVar} solid size="lg" />
  })
}

export interface ExplorerPlayerModeProps {
  playerSlug: string
  targetGamertag: string
  locale: string
  t: (key: ExplorerManifestKey, values?: Record<string, string | number>) => string
  playerQuery: {
    data: ExplorerPlayerQueryResponse | undefined
    isLoading: boolean
    isError: boolean
  }
  allyMatchIds: string[]
  enemyMatchIds: string[]
  allyMatchesData: ExplorerMatchesQueryResponse | undefined
  enemyMatchesData: ExplorerMatchesQueryResponse | undefined
  onSelectTarget: (gamertag: string) => void
  onOpenHeadToHead: (gamertag: string) => void
}

export function ExplorerPlayerMode({
  playerSlug,
  targetGamertag,
  locale,
  t,
  playerQuery,
  allyMatchIds,
  enemyMatchIds,
  allyMatchesData,
  enemyMatchesData,
  onSelectTarget,
  onOpenHeadToHead,
}: ExplorerPlayerModeProps) {
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 flex-wrap">
        <GamertagSearchInput onSelect={onSelectTarget} initialValue={targetGamertag} />
        {targetGamertag && !!playerQuery.data?.badges?.length && (
          <div className="flex flex-wrap gap-1.5">
            {renderEncounterBadges(playerQuery.data!.badges!, locale)}
          </div>
        )}
        {targetGamertag && playerQuery.data?.encounter_stats && (
          <button
            type="button"
            onClick={() =>
              onOpenHeadToHead(playerQuery.data?.target_gamertag ?? targetGamertag)
            }
            className="inline-flex h-9 items-center rounded border border-input bg-background px-3 text-xs font-medium hover:bg-muted transition-colors"
          >
            {t('explorer.player.head_to_head')}
          </button>
        )}
      </div>

      {!targetGamertag && (
        <Card>
          <CardContent className="py-4 pt-4">
            <EmptyStateNotice
              title={t('explorer.player.no_selection_title')}
              description={t('explorer.player.no_selection_description')}
            />
          </CardContent>
        </Card>
      )}

      {targetGamertag && playerQuery.isLoading && !playerQuery.data && (
        <div className="flex justify-center py-8">
          <Spinner label={t('explorer.player.searching')} />
        </div>
      )}

      {targetGamertag && playerQuery.isError && (
        <Card>
          <CardContent className="py-4 pt-4">
            <EmptyStateNotice
              title={t('explorer.player.error_title')}
              description={t('explorer.player.error_description')}
            />
          </CardContent>
        </Card>
      )}

      {targetGamertag &&
        !playerQuery.isLoading &&
        !playerQuery.isError &&
        !playerQuery.data && (
          <Card>
            <CardContent className="py-4 pt-4">
              <EmptyStateNotice
                title={t('explorer.player.empty_title')}
                description={t('explorer.player.empty_description')}
              />
            </CardContent>
          </Card>
        )}

      {targetGamertag && playerQuery.data && (
        <>
          {/* Encart "Profil joueur cible" — bandeau identité + carrière live
              + sample stats sur common_matches. Rendu juste après la search
              bar pour donner du contexte avant le briefing détaillé. */}
          {playerQuery.data.target_profile && (
            <ExplorerTargetProfileCard
              profile={playerQuery.data.target_profile}
              gamertag={playerQuery.data.target_gamertag || targetGamertag}
              encounterStats={playerQuery.data.encounter_stats}
            />
          )}

          {/* Profil de combat — 5 graphes sur les derniers matchs PvP de la
              cible (FDA, dégâts, score+placement, folie/parfaits, donut modes).
              Masqué seulement si la cible n'a aucun match PvP ET que le statut
              live ne signale rien (A3) : sinon monté quand même pour afficher
              le badge/notice expliquant l'absence de données. */}
          {(playerQuery.data.target_profile?.combat_profile?.length ||
            playerQuery.data.target_profile?.combat_profile_local?.length ||
            (playerQuery.data.target_profile?.live_status?.combat_live &&
              playerQuery.data.target_profile.live_status.combat_live !== 'ok')) ? (
            <ExplorerCombatProfile
              liveMatches={playerQuery.data.target_profile.combat_profile ?? []}
              localMatches={playerQuery.data.target_profile.combat_profile_local ?? []}
              locale={locale}
              t={t}
              topMedals={playerQuery.data.target_profile.top_medals ?? []}
              combatLiveStatus={playerQuery.data.target_profile.live_status?.combat_live}
            />
          ) : null}

          {playerQuery.data.encounter_stats ? (
            <ExplorerEncounterBriefing
              stats={playerQuery.data.encounter_stats}
              locale={locale}
            />
          ) : (
            <Card>
              <CardContent className="py-4 pt-4">
                <EmptyStateNotice
                  title={t('explorer.player.no_common_matches_title')}
                  description={t('explorer.player.no_common_matches_description')}
                />
              </CardContent>
            </Card>
          )}

          {/* Tableau "matchs en allié" — affiché uniquement si on a au moins
              un match commun en tant qu'alliés. Pattern team-banner aligné
              sur MatchScoreboard (token team-ally). 10 lignes par défaut +
              expander pour passer à 20. */}
          {allyMatchIds.length > 0 && allyMatchesData && (
            <ExplorerMatchesTable
              rows={normalizeExplorerTableRows(allyMatchesData.table.items)}
              playerSlug={playerSlug}
              teamBanner={{
                variant: 'ally',
                label: t('explorer.player.table_as_ally'),
              }}
              contextDescriptor={{
                kind: 'with_player',
                gamertag: playerQuery.data.target_gamertag || targetGamertag,
              }}
              alwaysShowPagination
              defaultPageSize={10}
              // I16 : même endpoint/tri backend (date DESC) que le mode Matchs
              // → le défaut interne de ExplorerMatchesTable reproduit déjà
              // l'ordre serveur, aucun `defaultSort` à surcharger.
              sortable
            />
          )}

          {/* Tableau "matchs en ennemi" — token team-enemy. */}
          {enemyMatchIds.length > 0 && enemyMatchesData && (
            <ExplorerMatchesTable
              rows={normalizeExplorerTableRows(enemyMatchesData.table.items)}
              playerSlug={playerSlug}
              teamBanner={{
                variant: 'enemy',
                label: t('explorer.player.table_as_enemy'),
              }}
              contextDescriptor={{
                kind: 'with_player',
                gamertag: playerQuery.data.target_gamertag || targetGamertag,
              }}
              alwaysShowPagination
              defaultPageSize={10}
              sortable
            />
          )}

          {/* Heatmap d'activité commune — agrégat jour × heure de tous
              les matchs communs (alliés + ennemis). Coloration par
              intensité d'activité (count), pas par win-rate. */}
          {(playerQuery.data.activity_heatmap?.length ?? 0) > 0 && (
            <ExplorerActivityHeatmapChart
              title={t('explorer.player.activity_heatmap_title')}
              cells={playerQuery.data.activity_heatmap ?? []}
            />
          )}
        </>
      )}
    </div>
  )
}
