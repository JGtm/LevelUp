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
import { winRateColor, kdaNetColor } from '@/lib/colors/outcomePalette'
import { useNavigate, useParams } from '@tanstack/react-router'
import { useTitleSlug } from '@/lib/title-routing'

import { KpiCard } from '@/components/cards/KpiCard'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import { useLocalFilterBar } from '@/features/_shared/useLocalFilterBar'
import { NarrativeBadge } from '@/components/feedback/NarrativeBadge'
import { tokenCssVar, tokenVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { composeTierLabel } from '@/lib/skillTiers'
import { formatPercent } from '@/lib/formatters'
import type { FilterContextInput, RelationCSR, RelationInsight } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { useRelationsPrefsStore } from '@/stores/relationsPrefsStore'

import { getPalmaresText, normalizePalmaresLocale, type PalmaresText } from './i18n'
import type { Locale } from '@/lib/i18n/locale'
import { useRelationsMoments, useRelationsPage } from './queries'
import { RelationBadges } from './RelationBadges'
import { RelationSplitBar } from './RelationSplitBar'
import { RelationsMomentsSection } from './RelationsMomentsSection'
import { RelationsTable } from './RelationsTable'
import { RelationWinRateDonut } from './RelationWinRateDonut'
import {
  coreRelations,
  filterRelations,
  formatLastSeen,
  hasCrossGameRelations,
  type RelationFilter,
} from './relationsFilter'

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
// Les barres s'étirent (`flex-1`) pour occuper toute la largeur du bloc.
function OutcomeSparkline({ outcomes }: { outcomes: string[] }) {
  return (
    <div className="flex h-5 w-full items-end gap-0.5" aria-hidden="true">
      {outcomes.map((o, i) => (
        <span
          key={`${i}-${o}`}
          className="min-w-[2px] flex-1 rounded-sm"
          style={{
            height: o === 'win' ? '100%' : o === 'loss' ? '45%' : '70%',
            backgroundColor: tokenCssVar(duelOutcomeToken(o)),
          }}
        />
      ))}
    </div>
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
function nemesisRankLabel(csr: RelationCSR, locale: Locale): string | null {
  const tier = csr.tier?.trim()
  if (!tier) return null
  const base = composeTierLabel(tier, csr.sub_tier ?? 0, locale)
  if (tier.toLowerCase() === 'onyx' && csr.rating_value != null && Number.isFinite(csr.rating_value)) {
    return `${base} ${Math.round(csr.rating_value)}`
  }
  return base
}

// donutLabels — projette les libellés i18n vers le contrat du donut (partagé par
// les 3 cartes hero).
function donutLabels(labels: RelationsText) {
  return {
    wins: labels.donut.wins,
    losses: labels.donut.losses,
    personalAvg: labels.donut.personalAvg,
    pointsUnit: labels.donut.pointsUnit,
    liftTooltip: labels.core.liftTooltip,
  }
}

/**
 * HeroRelationCard — carte hero : binôme (mode ally) ou bête noire (mode enemy).
 * Grammaire commune avec la carte Noyau dur : accent EN HAUT → titre de bloc
 * (dans le bloc) → gamertag (identité, en blanc) → donut de taux de victoire avec
 * repère de la moyenne perso → [chips série/rang, barre frags/morts pour
 * l'ennemi] → sparkline labellée → footer détail (1 ligne).
 *  - ally : donut = WR ensemble vs moy. perso ; sparkline « Derniers matchs »
 *    (recentForm = top_ally_recent_form) ; footer FDA à tes côtés.
 *  - enemy : donut = WR face à lui vs moy. perso ; chip série + rang CSR ; barre
 *    Frags/morts ; sparkline « Derniers duels » ; footer ratio.
 */
function HeroRelationCard({
  title,
  emptyLabel,
  accent,
  relation,
  mode,
  labels,
  locale,
  onPlayerClick,
  playerWinRate,
  recentForm,
  streak,
  csr,
}: {
  title: string
  emptyLabel: string
  accent: Parameters<typeof KpiCard>[0]['accent']
  relation: RelationInsight | null
  mode: 'ally' | 'enemy'
  labels: RelationsText
  locale: Locale
  onPlayerClick: (gamertag: string) => void
  playerWinRate?: number | null
  recentForm?: string[] | null
  streak?: number
  csr?: RelationCSR | null
}) {
  if (!relation) {
    return (
      <KpiCard accent={accent} accentSide="top" className="flex flex-1 flex-col">
        <div className="flex flex-1 flex-col p-4">
          <p className="mb-2 text-xs font-semibold uppercase tracking-label text-muted-foreground">{title}</p>
          <p className="text-sm text-muted-foreground">{emptyLabel}</p>
        </div>
      </KpiCard>
    )
  }
  const isAlly = mode === 'ally'
  const wr = isAlly ? relation.teammate_win_rate : relation.enemy_win_rate
  const winQual = isAlly ? labels.hero.winQualAlly : labels.hero.winQualEnemy
  // Contexte CSR de la bête noire (lot relations-G, best-effort). null pour le
  // binôme ou une bête noire sans ligne CSR → rien n'est rendu (dégradation).
  const rankLabel = !isAlly && csr ? nemesisRankLabel(csr, locale) : null
  // Série en cours (bête noire uniquement).
  const streakNode = isAlly ? null : streakChip(streak, labels)

  // Cartes symétriques : signature FDA (à ses côtés / face à lui), barre composite
  // (alliés-adversaires / frags-morts), sparkline « Derniers matchs » (recentForm =
  // forme récente À CÔTÉS pour le binôme, CONTRE pour la bête noire — même source
  // fiable via l'overview), footer volume + dernière rencontre.
  const sigValue = isAlly ? relation.avg_kda_with : relation.avg_kda_against
  const sigLabel = isAlly ? labels.table.kdaTogether : labels.hero.kdaAgainst
  const sparkOutcomes = (recentForm ?? []).filter((o): o is string => typeof o === 'string')
  const volume = isAlly
    ? labels.hero.matchesPlayed(relation.teammate_matches.toLocaleString(locale))
    : labels.hero.duels(relation.enemy_matches.toLocaleString(locale))
  const lastSeen = formatLastSeen(relation.last_seen_at, labels.relative)

  return (
    <KpiCard accent={accent} accentSide="top" className="flex flex-1 flex-col">
      <div className="flex flex-1 flex-col p-4">
        {/* titre de bloc (dans le bloc) + identité en blanc */}
        <p className="mb-2 text-xs font-semibold uppercase tracking-label text-muted-foreground">{title}</p>
        <span className="whitespace-nowrap">
          <button
            type="button"
            className="text-left text-2xl font-semibold text-foreground hover:underline"
            onClick={() => onPlayerClick(relation.gamertag)}
          >
            {relation.gamertag}
          </button>
          <RelationBadges badges={relation.badges} locale={locale} />
        </span>

        {/* donut de taux de victoire + repère moyenne perso (caption à droite) */}
        <div className="mt-3">
          <RelationWinRateDonut
            winRate={wr}
            personalAvg={playerWinRate}
            labels={donutLabels(labels)}
            caption={winQual}
          />
        </div>

        {/* bête noire : rang CSR courant (best-effort classé) + série en cours */}
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
        {streakNode && <div className="mt-2">{streakNode}</div>}

        {/* SIGNATURE (mise en valeur, même position) : le FDA de CE JOUEUR avec toi
            (binôme) ou contre toi (bête noire). Couleur INVERSÉE pour la bête noire :
            un FDA élevé de l'adversaire est NÉGATIF pour toi (il domine) → rouge. */}
        {sigValue != null && Number.isFinite(sigValue) && (
          <div className="mt-3 flex items-baseline gap-2 rounded-md border border-border bg-muted/40 px-3 py-2">
            <span className="font-mono text-2xl font-bold" style={{ color: kdaNetColor(isAlly ? sigValue : -sigValue) }}>
              {formatRatio(sigValue)}
            </span>
            <span className="text-xs text-muted-foreground">{sigLabel}</span>
          </div>
        )}

        {/* barre composite pleine largeur (légende en dessous) : alliés/adversaires
            (binôme) ou frags/morts (bête noire) */}
        <div className="mt-3">
          {isAlly ? (
            <RelationSplitBar
              leftValue={relation.teammate_matches}
              rightValue={relation.enemy_matches}
              leftToken="team-ally"
              rightToken="team-enemy"
              leftLabel={labels.table.alliesUnit}
              rightLabel={labels.table.adversariesUnit}
              locale={locale}
            />
          ) : (
            <RelationSplitBar
              leftValue={relation.kills_dealt}
              rightValue={relation.deaths_suffered}
              leftToken="outcome-win"
              rightToken="outcome-loss"
              leftLabel={labels.table.fragsUnit}
              rightLabel={labels.table.deathsUnit}
              locale={locale}
            />
          )}
        </div>

        {/* sparkline unifiée « Derniers matchs » (pleine largeur) */}
        <SparklineSection label={labels.core.recentForm} outcomes={sparkOutcomes} />

        {/* footer : volume + dernière rencontre */}
        <p className="mt-3 text-xs text-muted-foreground">
          {volume}
          {lastSeen && <> · {lastSeen}</>}
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
 * CoreSummaryCard — résumé narratif du noyau dur. Accent EN HAUT + titre de bloc
 * (dans le bloc). Condense en une carte :
 *  - donut du WR moyen ensemble + repère de la moyenne perso historique (#1)
 *  - vus cette semaine (#3, si > 0)
 *  - sparkline « Derniers matchs » joués à côté d'un fidèle (#8, si recentForm fourni)
 *  - mini-tableau (sans en-têtes) des fidèles classés par WR, dépliable (#7)
 * recentForm vient de l'overview backend (optionnel) : rendu seulement quand la
 * donnée est présente, sinon la carte reste complète sans trou.
 */
function CoreSummaryCard({
  title,
  unit,
  coreRows,
  labels,
  locale,
  onPlayerClick,
  playerWinRate,
  recentForm,
}: {
  title: string
  unit: string
  coreRows: RelationInsight[]
  labels: RelationsText
  locale: Locale
  onPlayerClick: (gamertag: string) => void
  playerWinRate?: number | null
  recentForm?: string[] | null
}) {
  const [expanded, setExpanded] = useState(false)
  const count = coreRows.length
  const wrs = coreRows
    .map((r) => r.teammate_win_rate)
    .filter((v): v is number => v != null && Number.isFinite(v))
  const avgWr = wrs.length > 0 ? wrs.reduce((a, b) => a + b, 0) / wrs.length : null
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
    <KpiCard accent="info" accentSide="top" className="flex flex-1 flex-col">
      <div className="flex flex-1 flex-col p-4">
        {/* titre de bloc (dans le bloc) + effectif à gauche (même police que les
            noms de joueurs des autres cartes) et « vus cette semaine » à droite. */}
        <p className="mb-2 text-xs font-semibold uppercase tracking-label text-muted-foreground">{title}</p>
        <div className="flex items-baseline justify-between gap-2">
          <span className="text-2xl font-semibold text-foreground">
            {count.toLocaleString(locale)} {unit}
          </span>
          {seenThisWeek > 0 && (
            <span className="shrink-0 text-xs font-semibold" style={{ color: tokenCssVar('outcome-win') }}>
              {labels.core.seenThisWeek(seenThisWeek.toLocaleString(locale))}
            </span>
          )}
        </div>

        {/* donut du WR moyen ensemble + repère moyenne perso (caption à droite) (#1) */}
        <div className="mt-3">
          <RelationWinRateDonut
            winRate={avgWr}
            personalAvg={playerWinRate}
            labels={donutLabels(labels)}
            caption={labels.core.withThem}
          />
        </div>

        {/* sparkline des derniers matchs joués à côté d'un fidèle (#8, backend) */}
        <SparklineSection label={labels.core.recentForm} outcomes={form} />

        {/* mini-tableau (sans en-têtes) des fidèles classés par WR (#7).
            EXCEPTION tri client par en-têtes (I16) : pas de <thead> — ce n'est
            pas un tableau de données généraliste mais un aperçu classé (WR
            desc, tiebreak volume) avec expand/collapse ; rien à cliquer pour
            trier, le classement EST le contenu affiché. */}
        {ranked.length > 0 && (
          <div className="mt-3 border-t border-border pt-3">
            <table className="w-full border-collapse text-sm">
              <tbody>
                {visibleRanked.map((r, i) => (
                  <tr key={r.xuid} className="align-baseline">
                    <td className="py-0.5 pr-2 text-right font-mono text-xs text-muted-foreground tabular-nums">
                      {i + 1}
                    </td>
                    <td className="w-full py-0.5">
                      <button
                        type="button"
                        className="block max-w-full truncate text-left font-semibold text-foreground hover:underline"
                        onClick={() => onPlayerClick(r.gamertag)}
                      >
                        {r.gamertag}
                      </button>
                    </td>
                    <td className="whitespace-nowrap py-0.5 pl-2 text-right font-mono text-xs text-muted-foreground tabular-nums">
                      {labels.hero.matchesPlayed(r.total_matches.toLocaleString(locale))}
                    </td>
                    <td
                      className="whitespace-nowrap py-0.5 pl-2 text-right font-mono text-xs font-bold tabular-nums"
                      style={{ color: winRateColor(r.teammate_win_rate) }}
                    >
                      {formatPercent(r.teammate_win_rate, 0)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
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
  const titleSlug = useTitleSlug()
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
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/explorer',
      params: { titleSlug, playerSlug },
      search: { mode: 'player', target: gamertag },
    })
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
  playerSlug,
  filterContext,
  filterHash,
}: {
  data: NonNullable<ReturnType<typeof useRelationsPage>['data']>
  rel: RelationsText
  locale: Locale
  filter: RelationFilter
  setFilter: (f: RelationFilter) => void
  showCross: boolean
  includeNeverFaced: boolean
  setIncludeNeverFaced: (v: boolean) => void
  visibleRows: RelationInsight[]
  coreRows: RelationInsight[]
  onPlayerClick: (gamertag: string) => void
  playerSlug: string
  filterContext: FilterContextInput
  filterHash: string
}) {
  const ov = data.overview
  const relations = data.relations ?? []
  const allyRelation = findRelation(relations, ov.top_ally?.gamertag)
  const nemesisRelation = findRelation(relations, ov.top_nemesis?.gamertag)
  // Série en cours de la bête noire : réutilise la donnée Moments (même queryKey →
  // dédupliquée par TanStack Query, pas d'appel réseau supplémentaire). La sparkline
  // « Derniers matchs » vient désormais de l'overview (top_nemesis_recent_form),
  // fiable et symétrique du binôme — plus de dépendance aux duels asynchrones.
  const { data: momentsData } = useRelationsMoments(playerSlug, filterContext, filterHash, true)
  const nemesisRivalry = useMemo(() => {
    if (!momentsData || !nemesisRelation) return undefined
    return (momentsData.rivalries ?? []).find((r) => r.xuid === nemesisRelation.xuid)
  }, [momentsData, nemesisRelation])
  return (
    <>
      <div className="grid gap-4 lg:grid-cols-3" data-testid="palmares-relations-overview">
        <HeroRelationCard
          title={rel.hero.topAllyTitle}
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
        <HeroRelationCard
          title={rel.hero.topNemesisTitle}
          emptyLabel={rel.hero.topNemesisEmpty}
          accent="outcome-loss"
          relation={nemesisRelation}
          mode="enemy"
          labels={rel}
          locale={locale}
          onPlayerClick={onPlayerClick}
          playerWinRate={ov.player_win_rate}
          recentForm={ov.top_nemesis_recent_form}
          streak={nemesisRivalry?.current_streak}
          csr={ov.top_nemesis?.csr}
        />
        <CoreSummaryCard
          title={rel.hero.coreTitle}
          unit={rel.hero.coreUnit}
          coreRows={coreRows}
          labels={rel}
          locale={locale}
          onPlayerClick={onPlayerClick}
          playerWinRate={ov.player_win_rate}
          recentForm={ov.core_recent_form}
        />
      </div>

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
