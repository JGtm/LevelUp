/**
 * PalmaresRelationsPage — hub Communauté > Relations (Phase 2).
 *
 * Consomme l'endpoint backend réel POST /pages/palmares/relations (forme
 * {overview, relations[]}). Barre de segmentation serveur (useLocalFilterBar :
 * expérience / saison / période / playlist / mode / vue solo-escouade), hero
 * enrichi (binôme / bête noire / noyau dur), segmented control + toggle « jamais affrontés »,
 * tableau paginé (langage MatchEncountersTable) et section Moments & Rivalités.
 */
import { useMemo, useState, type ReactNode } from 'react'
import { winRateColor, ratioColor, kdaNetColor } from '@/lib/colors/outcomePalette'
import { useNavigate, useParams } from '@tanstack/react-router'

import { KpiCard } from '@/components/cards/KpiCard'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import { useLocalFilterBar } from '@/features/_shared/useLocalFilterBar'
import { NarrativeBadge } from '@/components/feedback/NarrativeBadge'
import { tokenCssVar, tokenVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { composeTierLabel } from '@/lib/skillTiers'
import { formatPercent } from '@/lib/formatters'
import type { FilterContextInput, RelationCSR, RelationDuelEntry, RelationInsight } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { useRelationsPrefsStore } from '@/stores/relationsPrefsStore'

import { getPalmaresText, normalizePalmaresLocale, type PalmaresLocale, type PalmaresText } from './i18n'
import { useRelationsMoments, useRelationsPage } from './queries'
import { RelationBadges } from './RelationBadges'
import { RelationSplitBar } from './RelationSplitBar'
import { RelationsMomentsSection } from './RelationsMomentsSection'
import { RelationsTable } from './RelationsTable'
import { RelationsWhatsNewStrip } from './RelationsWhatsNewStrip'
import { coreRelations, filterRelations, hasCrossGameRelations, type RelationFilter } from './relationsFilter'

type RelationsText = PalmaresText['relations']

const FILTER_CHIPS: RelationFilter[] = ['all', 'core', 'allies', 'rivals', 'recent']

/** findRelation — retrouve la ligne complète (RelationInsight) d'une référence hero. */
function findRelation(relations: RelationInsight[], gamertag: string | undefined | null): RelationInsight | null {
  if (!gamertag) return null
  return relations.find((r) => r.gamertag === gamertag) ?? null
}

function formatRatio(v: number | null | undefined): string {
  if (v == null || !Number.isFinite(v)) return '—'
  return v.toFixed(2)
}

// kdaColor : le KDA est un NET signé (peut être négatif) — vert si positif,
// rouge si négatif, neutre à 0 (cohérent avec kdaDivergentScale).
// duelOutcomeToken : couleur d'un carré de la mini-frise (win/loss/neutre).
function duelOutcomeToken(outcome: string): SemanticToken {
  if (outcome === 'win') return 'outcome-win'
  if (outcome === 'loss') return 'outcome-loss'
  return 'outcome-draw'
}

// OutcomeSparkline — frise W/L/neutre partagée par les 3 cartes (binôme, bête
// noire, noyau). La hauteur encode l'issue (victoire haute, défaite basse), la
// couleur via token. Décorative (aria-hidden) ; ancien→récent (gauche→droite).
function OutcomeSparkline({ outcomes }: { outcomes: string[] }) {
  return (
    <div className="flex h-5 items-end gap-0.5" aria-hidden="true">
      {outcomes.map((o, i) => (
        <span
          key={`${i}-${o}`}
          className="w-1.5 rounded-sm"
          style={{
            height: o === 'win' ? '100%' : o === 'loss' ? '45%' : '70%',
            backgroundColor: tokenCssVar(duelOutcomeToken(o)),
          }}
        />
      ))}
    </div>
  )
}

// LiftChip — pastille « +N pts vs moy. perso. historique » (vert si positif, rouge
// sinon). lift = fraction signée (0..1) ; masquée si < 0,5 pt. Partagée binôme/noyau.
function LiftChip({ lift, labels }: { lift: number | null; labels: RelationsText }) {
  if (lift == null || !Number.isFinite(lift) || Math.abs(lift) < 0.005) return null
  return (
    <span
      className="font-mono text-xs font-bold"
      style={{ color: lift >= 0 ? tokenCssVar('outcome-win') : tokenCssVar('outcome-loss') }}
      title={labels.core.liftTooltip}
    >
      {lift >= 0 ? '+' : '−'}
      {Math.round(Math.abs(lift) * 100)} {labels.core.liftPoints}
    </span>
  )
}

// SparklineSection — label uppercase + sparkline (mise en page partagée).
function SparklineSection({ label, outcomes }: { label: string; outcomes: string[] }) {
  if (outcomes.length === 0) return null
  return (
    <div className="mt-3">
      <p className="mb-1 text-[10px] uppercase tracking-label-md text-muted-foreground">{label}</p>
      <OutcomeSparkline outcomes={outcomes} />
    </div>
  )
}

// streakChip — pastille « N victoires/défaites de suite » (bête noire) selon la
// série en cours (>0 victoires, <0 défaites). Masquée si |série| < 2.
function streakChip(streak: number | undefined, labels: RelationsText): ReactNode {
  if (streak == null) return null
  if (streak >= 2) {
    return (
      <span className="font-mono text-xs font-bold" style={{ color: tokenCssVar('outcome-win') }}>
        {labels.hero.streakWins(streak.toString())}
      </span>
    )
  }
  if (streak <= -2) {
    return (
      <span className="font-mono text-xs font-bold" style={{ color: tokenCssVar('outcome-loss') }}>
        {labels.hero.streakLosses((-streak).toString())}
      </span>
    )
  }
  return null
}

/**
 * nemesisRankLabel — libellé du rang CSR courant de la bête noire (lot relations-G).
 * Compose « Nom + sous-palier » via composeTierLabel (source unique) ; pour Onyx
 * (palier ouvert), suffixe la valeur CSR si disponible (« Onyx 1523 »). Renvoie null
 * si le palier est absent → rien n'est affiché (dégradation gracieuse, pas de « N/A »).
 */
function nemesisRankLabel(csr: RelationCSR, locale: 'fr' | 'en'): string | null {
  const tier = csr.tier?.trim()
  if (!tier) return null
  const base = composeTierLabel(tier, csr.sub_tier ?? 0, locale)
  if (tier.toLowerCase() === 'onyx' && csr.rating_value != null && Number.isFinite(csr.rating_value)) {
    return `${base} ${Math.round(csr.rating_value)}`
  }
  return base
}

/**
 * HeroRelationCard — carte hero : binôme (mode ally) ou bête noire (mode enemy).
 * Grammaire commune avec la carte Noyau dur : uplabel → gamertag (identité, en
 * tête) → ligne métrique (% de victoires + qualificatif + chip) → [barre
 * frags/morts pour l'ennemi] → sparkline labellée → footer détail (1 ligne).
 *  - ally : chip = lift (réutilise playerWinRate) ; sparkline « Derniers matchs
 *    ensemble » (recentForm = top_ally_recent_form) ; footer FDA à tes côtés.
 *  - enemy : chip = série en cours ; barre Frags/morts ; sparkline « Derniers
 *    duels » (issues des duels) ; footer ratio.
 */
function HeroRelationCard({
  emptyLabel,
  accent,
  relation,
  mode,
  labels,
  locale,
  onPlayerClick,
  duels,
  playerWinRate,
  recentForm,
  streak,
  csr,
}: {
  emptyLabel: string
  accent: Parameters<typeof KpiCard>[0]['accent']
  relation: RelationInsight | null
  mode: 'ally' | 'enemy'
  labels: RelationsText
  locale: 'fr' | 'en'
  onPlayerClick: (gamertag: string) => void
  duels?: RelationDuelEntry[]
  playerWinRate?: number | null
  recentForm?: string[] | null
  streak?: number
  csr?: RelationCSR | null
}) {
  if (!relation) {
    return (
      <KpiCard accent={accent} accentSide="left" className="flex flex-1 flex-col">
        <div className="flex flex-1 flex-col p-4">
          <p className="text-sm text-muted-foreground">{emptyLabel}</p>
        </div>
      </KpiCard>
    )
  }
  const isAlly = mode === 'ally'
  const wr = isAlly ? relation.teammate_win_rate : relation.enemy_win_rate
  const winQual = isAlly ? labels.hero.winQualAlly : labels.hero.winQualEnemy
  // ally : lift vs moyenne perso historique ; enemy : chip série en cours.
  const lift =
    isAlly && wr != null && playerWinRate != null && Number.isFinite(playerWinRate) ? wr - playerWinRate : null
  const allyForm = (recentForm ?? []).filter((o): o is string => typeof o === 'string')
  const duelOutcomes = (duels ?? []).slice(-25).map((d) => d.outcome)
  // Contexte CSR de la bête noire (lot relations-G, best-effort). null pour le
  // binôme ou une bête noire sans ligne CSR → rien n'est rendu (dégradation).
  const rankLabel = !isAlly && csr ? nemesisRankLabel(csr, locale) : null

  return (
    <KpiCard accent={accent} accentSide="left" className="flex flex-1 flex-col">
      <div className="flex flex-1 flex-col p-4">
        <span className="whitespace-nowrap">
          <button
            type="button"
            className="text-left text-2xl font-semibold text-info hover:underline"
            onClick={() => onPlayerClick(relation.gamertag)}
          >
            {relation.gamertag}
          </button>
          <RelationBadges badges={relation.badges} locale={locale} />
        </span>

        {/* ligne métrique : % de victoires + qualificatif + chip */}
        <div className="mt-2 flex flex-wrap items-baseline gap-x-2 gap-y-1">
          <span className="font-mono text-xl font-bold" style={{ color: winRateColor(wr) }}>
            {formatPercent(wr, 0)}
          </span>
          <span className="text-xs text-muted-foreground">{winQual}</span>
          {isAlly ? <LiftChip lift={lift} labels={labels} /> : streakChip(streak, labels)}
        </div>

        {/* bête noire : rang CSR courant (lot relations-G, best-effort classé).
            Rendu uniquement si un snapshot CSR existe — sinon rien (dégradation). */}
        {rankLabel && (
          <div
            className="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1"
            data-testid="nemesis-current-rank"
          >
            <span className="text-xs text-muted-foreground">{labels.hero.currentRank}</span>
            <NarrativeBadge
              label={rankLabel}
              colorVar={tokenVar('narrative-encounter-tough-enemy')}
              solid
              size="sm"
            />
          </div>
        )}

        {/* bête noire : barre Frags / morts conservée */}
        {!isAlly && (
          <div className="mt-3">
            <RelationSplitBar
              label={labels.table.fragsDeaths}
              leftValue={relation.kills_dealt}
              rightValue={relation.deaths_suffered}
              leftToken="outcome-win"
              rightToken="outcome-loss"
              locale={locale}
            />
          </div>
        )}

        {/* sparkline : matchs ensemble (binôme) ou duels (bête noire) */}
        {isAlly ? (
          <SparklineSection label={labels.core.recentForm} outcomes={allyForm} />
        ) : (
          <SparklineSection label={labels.hero.recentDuels} outcomes={duelOutcomes} />
        )}

        {/* footer détail : FDA (binôme) ou ratio (bête noire) + volume */}
        <p className="mt-3 text-xs text-muted-foreground">
          {isAlly ? (
            <>
              {relation.avg_kda_with != null && Number.isFinite(relation.avg_kda_with) && (
                <>
                  <span className="font-mono font-bold" style={{ color: kdaNetColor(relation.avg_kda_with) }}>
                    {formatRatio(relation.avg_kda_with)}
                  </span>{' '}
                  {labels.table.kdaTogether}
                  {' · '}
                </>
              )}
              {labels.hero.matchesPlayed(relation.teammate_matches.toLocaleString(locale))}
            </>
          ) : (
            <>
              {relation.duel_ratio != null && Number.isFinite(relation.duel_ratio) && (
                <>
                  <span className="font-mono font-bold" style={{ color: ratioColor(relation.duel_ratio) }}>
                    {formatRatio(relation.duel_ratio)}
                  </span>{' '}
                  {labels.table.ratio}
                  {' · '}
                </>
              )}
              {labels.hero.duels(relation.enemy_matches.toLocaleString(locale))}
            </>
          )}
        </p>
      </div>
    </KpiCard>
  )
}

// Fenêtre « vus cette semaine » (7 jours) pour la carte résumé du noyau dur.
const CORE_WEEK_MS = 7 * 24 * 60 * 60 * 1000
// Mini-classement : nombre de fidèles affichés avant le bouton « voir les autres ».
const CORE_RANKING_PREVIEW = 3

// countSeenThisWeek — fidèles vus il y a moins de 7 jours (last_seen_at). Helper
// hors composant : la règle react-hooks/purity n'interdit l'appel impur Date.now()
// que dans le corps d'un composant/hook (cf. relationsFilter.ts, même pattern).
function countSeenThisWeek(rows: RelationInsight[]): number {
  const since = Date.now() - CORE_WEEK_MS
  return rows.filter((r) => {
    if (!r.last_seen_at) return false
    const ts = new Date(r.last_seen_at).getTime()
    return Number.isFinite(ts) && ts >= since
  }).length
}

/**
 * CoreSummaryCard — résumé narratif du noyau dur. Condense en une carte :
 *  - WR moyen ensemble + lift vs ta moyenne perso historique (#1, si player_win_rate fourni)
 *  - vus cette semaine (#3, si > 0)
 *  - sparkline des derniers matchs joués à tes côtés avec un fidèle (#8, si recentForm fourni)
 *  - mini-classement dépliable des fidèles par WR (#7)
 * lift et recentForm viennent de l'overview backend (optionnels) : rendus seulement
 * quand la donnée est présente, sinon la carte reste complète sans trou.
 */
function CoreSummaryCard({
  unit,
  coreRows,
  labels,
  locale,
  onPlayerClick,
  onViewSquad,
  playerWinRate,
  recentForm,
}: {
  unit: string
  coreRows: RelationInsight[]
  labels: RelationsText
  locale: 'fr' | 'en'
  onPlayerClick: (gamertag: string) => void
  onViewSquad: () => void
  playerWinRate?: number | null
  recentForm?: string[] | null
}) {
  const [expanded, setExpanded] = useState(false)
  const count = coreRows.length
  const wrs = coreRows
    .map((r) => r.teammate_win_rate)
    .filter((v): v is number => v != null && Number.isFinite(v))
  const avgWr = wrs.length > 0 ? wrs.reduce((a, b) => a + b, 0) / wrs.length : null
  // Lift : WR ensemble − WR perso HISTORIQUE (tout-temps). En points (0..100).
  const lift =
    avgWr != null && playerWinRate != null && Number.isFinite(playerWinRate) ? avgWr - playerWinRate : null
  // Vus cette semaine : fidèles avec un last_seen_at de moins de 7 jours.
  const seenThisWeek = countSeenThisWeek(coreRows)
  // Classement des fidèles par WR ensemble (desc), nuls relégués, tiebreak volume.
  const ranked = useMemo(
    () =>
      [...coreRows].sort((a, b) => {
        const av = a.teammate_win_rate ?? -1
        const bv = b.teammate_win_rate ?? -1
        if (bv !== av) return bv - av
        return b.total_matches - a.total_matches
      }),
    [coreRows],
  )
  const visibleRanked = expanded ? ranked : ranked.slice(0, CORE_RANKING_PREVIEW)
  const hiddenCount = ranked.length - CORE_RANKING_PREVIEW
  const form = (recentForm ?? []).filter((o): o is string => typeof o === 'string')

  return (
    <KpiCard accent="info" accentSide="left" className="flex flex-1 flex-col">
      <div className="flex flex-1 flex-col p-4">
        {/* WR moyen ensemble + lift vs ta moyenne (#1) */}
        <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1">
          {avgWr != null ? (
            <>
              <span className="font-mono text-3xl font-bold" style={{ color: winRateColor(avgWr) }}>
                {formatPercent(avgWr, 0)}
              </span>
              <span className="text-xs text-muted-foreground">{labels.core.withThem}</span>
            </>
          ) : (
            <span className="text-2xl font-semibold text-foreground">
              {count.toLocaleString(locale)} <span className="text-base font-normal text-muted-foreground">{unit}</span>
            </span>
          )}
          <LiftChip lift={lift} labels={labels} />
        </div>

        {/* vus cette semaine (#3) — affiché seulement s'il y en a */}
        {seenThisWeek > 0 && (
          <p className="mt-2 text-xs">
            <span className="font-semibold" style={{ color: tokenCssVar('outcome-win') }}>
              {labels.core.seenThisWeek(seenThisWeek.toLocaleString(locale))}
            </span>
          </p>
        )}

        {/* sparkline des derniers matchs joués à tes côtés avec un fidèle (#8, backend) */}
        <SparklineSection label={labels.core.recentForm} outcomes={form} />

        {/* mini-classement dépliable des fidèles (#7) */}
        {ranked.length > 0 && (
          <div className="mt-3 border-t border-border pt-3">
            <ul className="flex flex-col gap-1.5">
              {visibleRanked.map((r, i) => (
                <li key={r.xuid} className="flex items-baseline justify-between gap-2 text-sm">
                  <button
                    type="button"
                    className="flex min-w-0 items-baseline text-left font-semibold text-info hover:underline"
                    onClick={() => onPlayerClick(r.gamertag)}
                  >
                    <span className="mr-1.5 shrink-0 text-xs text-muted-foreground">{i + 1}</span>
                    <span className="truncate">{r.gamertag}</span>
                  </button>
                  {r.teammate_win_rate != null && Number.isFinite(r.teammate_win_rate) && (
                    <span className="shrink-0 font-mono text-xs text-muted-foreground">
                      {labels.hero.matchesPlayed(r.total_matches.toLocaleString(locale))}
                      {' · '}
                      <span className="font-bold" style={{ color: winRateColor(r.teammate_win_rate) }}>
                        {formatPercent(r.teammate_win_rate, 0)}
                      </span>
                    </span>
                  )}
                </li>
              ))}
            </ul>
            {hiddenCount > 0 && (
              <button
                type="button"
                className="mt-2 text-xs font-semibold text-info hover:underline"
                onClick={() => setExpanded((v) => !v)}
                aria-expanded={expanded}
              >
                {expanded ? labels.core.collapse : labels.core.showOthers(hiddenCount.toLocaleString(locale))}
              </button>
            )}
          </div>
        )}

        {/* Pont vers Escouade (« nous » = groupe déclaré) : lien discret en pied de carte. */}
        <button
          type="button"
          className="mt-3 self-start text-xs font-medium text-muted-foreground hover:text-info hover:underline"
          onClick={onViewSquad}
        >
          {labels.core.viewSquad}
        </button>
      </div>
    </KpiCard>
  )
}

/** SegmentedFilter — segmented control (charte) : une piste, segment actif plein.
 * Le chip « Multi-jeux » (cross) n'est rendu que si `showCross` (au moins une
 * relation croisée sur un autre titre) — pas de segment mort en mono-titre. */
function SegmentedFilter({
  active,
  onChange,
  labels,
  showCross,
}: {
  active: RelationFilter
  onChange: (f: RelationFilter) => void
  labels: RelationsText['chips']
  showCross: boolean
}) {
  const text: Record<RelationFilter, string> = {
    all: labels.all,
    core: labels.core,
    allies: labels.allies,
    rivals: labels.rivals,
    recent: labels.recent,
    cross: labels.cross,
  }
  const chips: RelationFilter[] = showCross ? [...FILTER_CHIPS, 'cross'] : FILTER_CHIPS
  return (
    <div
      className="inline-flex flex-wrap rounded-lg border border-border bg-card p-0.5"
      data-testid="palmares-relations-chips"
    >
      {chips.map((chip) => {
        const isActive = chip === active
        return (
          <button
            key={chip}
            type="button"
            aria-pressed={isActive}
            onClick={() => onChange(chip)}
            className={`rounded-md px-3 py-1 text-sm font-medium transition-colors ${
              isActive ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            {text[chip]}
          </button>
        )
      })}
    </div>
  )
}

export function PalmaresRelationsPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = normalizePalmaresLocale(useAppShellStore((state) => state.locale))
  const text = getPalmaresText(locale)
  const rel = text.relations
  const navigate = useNavigate()
  const filter = useRelationsPrefsStore((s) => s.filter)
  const setFilter = useRelationsPrefsStore((s) => s.setFilter)
  const includeNeverFaced = useRelationsPrefsStore((s) => s.includeNeverFaced)
  const setIncludeNeverFaced = useRelationsPrefsStore((s) => s.setIncludeNeverFaced)

  const { committedFilterContext, committedHash, bar } = useLocalFilterBar({
    playerSlug,
    labels: {
      experience: rel.filters.experience,
      experienceAll: rel.filters.experienceAll,
      experienceRanked: rel.filters.experienceRanked,
      experienceUnranked: rel.filters.experienceUnranked,
      playlists: rel.filters.playlists,
      modes: rel.filters.modes,
      reset: rel.filters.reset,
      analyser: rel.filters.analyser,
    },
    viewLabels: {
      view: rel.filters.view,
      viewAll: rel.filters.viewAll,
      viewSolo: rel.filters.viewSolo,
      viewSquad: rel.filters.viewSquad,
    },
  })

  const { data, isLoading, isError, error, refetch } = useRelationsPage(
    playerSlug,
    committedFilterContext,
    committedHash,
  )

  function goToExplorer(gamertag: string) {
    void navigate({
      to: '/players/$playerSlug/explorer',
      params: { playerSlug },
      search: { mode: 'player', target: gamertag },
    })
  }

  function goToSquad() {
    void navigate({ to: '/players/$playerSlug/squad', params: { playerSlug } })
  }

  // Relations déjà servies par le backend — tout le filtrage ci-dessous est client.
  const relations = useMemo(() => data?.relations ?? [], [data])
  // Chip « Multi-jeux » : rendu seulement si au moins une relation cross-jeu.
  const showCross = useMemo(() => hasCrossGameRelations(relations), [relations])
  // Garde-fou : un filtre persisté 'cross' sans donnée cross-jeu retombe sur 'all'
  // sans réécrire le store (la préférence reste, réactivée si la donnée revient).
  const effectiveFilter: RelationFilter = filter === 'cross' && !showCross ? 'all' : filter

  // Filtre segment (client) + toggle « jamais affrontés » : OFF (défaut), on masque
  // les relations purement coéquipières (jamais affrontées, enemy_matches === 0).
  const visibleRows = useMemo(() => {
    const base = filterRelations(relations, effectiveFilter)
    return includeNeverFaced ? base : base.filter((r) => r.enemy_matches > 0)
  }, [relations, effectiveFilter, includeNeverFaced])
  const coreRows = useMemo(() => coreRelations(relations), [relations])

  let body: ReactNode
  if (isLoading) {
    body = (
      <div className="flex items-center justify-center py-24">
        <Spinner size="lg" />
      </div>
    )
  } else if (isError || !data) {
    body = (
      <EmptyStateCard
        title={rel.unavailableTitle}
        description={error?.message ?? rel.unavailableDescription}
        actionLabel={rel.retry}
        onAction={() => refetch()}
      />
    )
  } else if ((data.relations?.length ?? 0) === 0) {
    body = <EmptyStateCard title={rel.emptyTitle} description={rel.emptyDescription} />
  } else {
    body = (
      <RelationsContent
        data={data}
        rel={rel}
        locale={locale}
        filter={effectiveFilter}
        setFilter={setFilter}
        showCross={showCross}
        includeNeverFaced={includeNeverFaced}
        setIncludeNeverFaced={setIncludeNeverFaced}
        visibleRows={visibleRows}
        coreRows={coreRows}
        onPlayerClick={goToExplorer}
        onViewSquad={goToSquad}
        playerSlug={playerSlug}
        filterContext={committedFilterContext}
        filterHash={committedHash}
      />
    )
  }

  return (
    <div className="flex flex-col">
      {bar}
      <div className="flex flex-col gap-6 p-6">{body}</div>
    </div>
  )
}

function RelationsContent({
  data,
  rel,
  locale,
  filter,
  setFilter,
  showCross,
  includeNeverFaced,
  setIncludeNeverFaced,
  visibleRows,
  coreRows,
  onPlayerClick,
  onViewSquad,
  playerSlug,
  filterContext,
  filterHash,
}: {
  data: NonNullable<ReturnType<typeof useRelationsPage>['data']>
  rel: RelationsText
  locale: PalmaresLocale
  filter: RelationFilter
  setFilter: (f: RelationFilter) => void
  showCross: boolean
  includeNeverFaced: boolean
  setIncludeNeverFaced: (v: boolean) => void
  visibleRows: RelationInsight[]
  coreRows: RelationInsight[]
  onPlayerClick: (gamertag: string) => void
  onViewSquad: () => void
  playerSlug: string
  filterContext: FilterContextInput
  filterHash: string
}) {
  const ov = data.overview
  const relations = data.relations ?? []
  const allyRelation = findRelation(relations, ov.top_ally?.gamertag)
  const nemesisRelation = findRelation(relations, ov.top_nemesis?.gamertag)
  // Frise du hero « Bête noire » : réutilise la donnée Moments (même queryKey →
  // dédupliquée par TanStack Query, pas d'appel réseau supplémentaire).
  const { data: momentsData } = useRelationsMoments(playerSlug, filterContext, filterHash, true)
  const nemesisRivalry = useMemo(() => {
    if (!momentsData || !nemesisRelation) return undefined
    return (momentsData.rivalries ?? []).find((r) => r.xuid === nemesisRelation.xuid)
  }, [momentsData, nemesisRelation])
  const nemesisDuels = nemesisRivalry?.duels ?? undefined
  return (
    <>
      <div className="grid gap-4 lg:grid-cols-3" data-testid="palmares-relations-overview">
        <div className="flex flex-col gap-2">
          <h3 className="text-sm font-semibold text-foreground">{rel.hero.topAllyTitle}</h3>
          <HeroRelationCard
            emptyLabel={rel.hero.topAllyEmpty}
            accent="outcome-win"
            relation={allyRelation}
            mode="ally"
            labels={rel}
            locale={locale}
            onPlayerClick={onPlayerClick}
            playerWinRate={ov.player_win_rate}
            recentForm={ov.top_ally_recent_form}
          />
        </div>
        <div className="flex flex-col gap-2">
          <h3 className="text-sm font-semibold text-foreground">{rel.hero.topNemesisTitle}</h3>
          <HeroRelationCard
            emptyLabel={rel.hero.topNemesisEmpty}
            accent="outcome-loss"
            relation={nemesisRelation}
            mode="enemy"
            labels={rel}
            locale={locale}
            onPlayerClick={onPlayerClick}
            duels={nemesisDuels}
            streak={nemesisRivalry?.current_streak}
            csr={ov.top_nemesis?.csr}
          />
        </div>
        <div className="flex flex-col gap-2">
          <h3 className="text-sm font-semibold text-foreground">{rel.hero.coreTitle}</h3>
          <CoreSummaryCard
            unit={rel.hero.coreUnit}
            coreRows={coreRows}
            labels={rel}
            locale={locale}
            onPlayerClick={onPlayerClick}
            onViewSquad={onViewSquad}
            playerWinRate={ov.player_win_rate}
            recentForm={ov.core_recent_form}
          />
        </div>
      </div>

      <RelationsWhatsNewStrip relations={relations} labels={rel} onPlayerClick={onPlayerClick} />

      <div className="flex flex-wrap items-center gap-3">
        <SegmentedFilter active={filter} onChange={setFilter} labels={rel.chips} showCross={showCross} />
        <button
          type="button"
          aria-pressed={includeNeverFaced}
          onClick={() => setIncludeNeverFaced(!includeNeverFaced)}
          className={`rounded-lg border px-3 py-1 text-sm font-medium transition-colors ${
            includeNeverFaced
              ? 'border-info text-foreground'
              : 'border-border text-muted-foreground hover:text-foreground'
          }`}
        >
          {includeNeverFaced ? rel.filters.neverFacedIncluded : rel.filters.includeNeverFaced}
        </button>
      </div>

      <RelationsTable
        rows={visibleRows}
        labels={rel}
        locale={locale}
        onPlayerClick={onPlayerClick}
        emptyMessage={rel.filterEmptyDescription}
      />

      <RelationsMomentsSection
        playerSlug={playerSlug}
        filterContext={filterContext}
        filterHash={filterHash}
        text={rel.moments}
      />
    </>
  )
}
