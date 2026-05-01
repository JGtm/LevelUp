/**
 * SessionBriefingMock — mock statique de la fusion KPI bar + Squad verdict.
 *
 * Sandbox dev (route /lab/briefing) pour comparer 2 variantes de layout :
 *   - Variant A : Rail descriptif + Verdict + 5 KPI cards évaluatives
 *   - Variant B : Verdict + 8 KPI cards (layout actuel preservé)
 *
 * Toggles :
 *   - mode = solo | squad   (le verdict band ne s'affiche qu'en squad)
 *   - drill-down            (click sur card joueur en squad → KPI grid recalculé)
 *
 * Sémantique trend ▲/▼ : vs moyenne d'équipe sur la session (PAS vs all-time).
 *
 * Données hardcodées — exemption no-hardcoded-strings (sandbox dev).
 */
/* eslint-disable @levelup/no-hardcoded-strings */
import { useMemo, useState } from 'react'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'

// ─── Types ───────────────────────────────────────────────────────────────────

interface PlayerKpis {
  matches: number
  totalPlaySec: number
  avgMatchSec: number
  killsPerMatch: number
  killsPerMin: number
  deathsPerMatch: number
  deathsPerMin: number
  assistsPerMatch: number
  assistsPerMin: number
  accuracyPct: number
  lifespanSec: number
  wins: number
  losses: number
  ties: number
  dnf: number
}

interface Player {
  xuid: string
  gamertag: string
  score: number
  kpis: PlayerKpis
}

type Mode = 'solo' | 'squad'

// ─── Données mock ────────────────────────────────────────────────────────────

const ACTIVE_PLAYER: Player = {
  xuid: 'me',
  gamertag: 'Spartan-117',
  score: 38,
  kpis: {
    matches: 10,
    totalPlaySec: 6540,
    avgMatchSec: 521,
    killsPerMatch: 8.70,
    killsPerMin: 1.00,
    deathsPerMatch: 10.80,
    deathsPerMin: 1.24,
    assistsPerMatch: 4.50,
    assistsPerMin: 0.52,
    accuracyPct: 46.92,
    lifespanSec: 37,
    wins: 3,
    losses: 7,
    ties: 0,
    dnf: 0,
  },
}

const TEAMMATES: Player[] = [
  {
    xuid: 'choco',
    gamertag: 'Chocoboflor',
    score: 44,
    kpis: {
      matches: 10, totalPlaySec: 6540, avgMatchSec: 521,
      killsPerMatch: 6.5, killsPerMin: 0.75,
      deathsPerMatch: 11.2, deathsPerMin: 1.29,
      assistsPerMatch: 3.2, assistsPerMin: 0.37,
      accuracyPct: 42.0, lifespanSec: 28,
      wins: 3, losses: 7, ties: 0, dnf: 0,
    },
  },
  {
    xuid: 'ghost',
    gamertag: 'Ghost',
    score: 51,
    kpis: {
      matches: 10, totalPlaySec: 6540, avgMatchSec: 521,
      killsPerMatch: 11.0, killsPerMin: 1.27,
      deathsPerMatch: 9.5, deathsPerMin: 1.09,
      assistsPerMatch: 6.5, assistsPerMin: 0.75,
      accuracyPct: 55.5, lifespanSec: 51,
      wins: 3, losses: 7, ties: 0, dnf: 0,
    },
  },
]

const SQUAD_SCORE = 44
const SQUAD_BASE_AVG = 44.3
const SQUAD_GRADE = 'C'

// ─── Helpers de formatage ────────────────────────────────────────────────────

function formatDurationDhm(sec: number): string {
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  return h > 0 ? `${h}h ${String(m).padStart(2, '0')}min` : `${m}min`
}

function formatMmss(sec: number): string {
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return `${m}:${String(s).padStart(2, '0')}`
}

// ─── Score → palier (5 tiers) ────────────────────────────────────────────────

interface ScoreTier {
  label: string
  token: SemanticToken
}

function getScoreTier(score: number): ScoreTier {
  if (score >= 75) return { label: 'Excellent', token: 'perf-tier-1' }
  if (score >= 60) return { label: 'Solide', token: 'perf-tier-2' }
  if (score >= 45) return { label: 'Correct', token: 'perf-tier-3' }
  if (score >= 30) return { label: 'Mauvais', token: 'perf-tier-4' }
  return { label: 'Pourri', token: 'perf-tier-5' }
}

// ─── Trend vs moyenne équipe (session) ───────────────────────────────────────

type TrendState = 'above' | 'near' | 'below' | 'none'

function trendVsTeam(
  current: number | null | undefined,
  teamAvg: number | null | undefined,
  lowerIsBetter = false,
  threshold = 0.05,
): TrendState {
  if (current == null || teamAvg == null || teamAvg === 0) return 'none'
  const ratio = current / teamAvg
  const upper = 1 + threshold
  const lower = 1 - threshold
  if (ratio >= upper) return lowerIsBetter ? 'below' : 'above'
  if (ratio <= lower) return lowerIsBetter ? 'above' : 'below'
  return 'near'
}

function computeTeamAvg(players: Player[], picker: (k: PlayerKpis) => number): number {
  if (players.length === 0) return 0
  return players.reduce((acc, p) => acc + picker(p.kpis), 0) / players.length
}

// ─── Composants atomiques ────────────────────────────────────────────────────

function TrendBadge({ state }: { state: TrendState }) {
  if (state === 'none') return null
  const sym = state === 'above' ? '▲' : state === 'below' ? '▼' : '━'
  const tok: SemanticToken =
    state === 'above' ? 'divergent-pos' : state === 'below' ? 'divergent-neg' : 'divergent-neutral'
  return (
    <span className="text-xs font-semibold ml-1" style={{ color: tokenCssVar(tok) }}>
      {sym}
    </span>
  )
}

interface KpiCellProps {
  label: string
  value: string
  sub?: string
  trend?: TrendState
  wide?: boolean
}

function KpiCell({ label, value, sub, trend = 'none', wide = false }: KpiCellProps) {
  return (
    <div
      className={`rounded border border-border bg-[#1d2328] px-3 py-2 ${wide ? 'col-span-2' : ''}`}
    >
      <p className="text-[11px] text-muted-foreground uppercase tracking-wide">{label}</p>
      <div className="flex items-baseline">
        <span className="text-lg font-bold text-foreground">{value}</span>
        <TrendBadge state={trend} />
      </div>
      {sub && <p className="text-[10px] text-muted-foreground mt-0.5">{sub}</p>}
    </div>
  )
}

function ResultsBar({ kpis }: { kpis: PlayerKpis }) {
  const total = kpis.wins + kpis.losses + kpis.ties + kpis.dnf
  if (total === 0) return null
  const segs: Array<{ value: number; token: SemanticToken; label: string }> = [
    { value: kpis.wins, token: 'outcome-win', label: `${kpis.wins} V` },
    { value: kpis.losses, token: 'outcome-loss', label: `${kpis.losses} D` },
    { value: kpis.ties, token: 'outcome-draw', label: `${kpis.ties} N` },
    { value: kpis.dnf, token: 'outcome-dnf', label: `${kpis.dnf} DNF` },
  ]
  return (
    <div className="flex flex-col gap-1 min-w-[180px]">
      <p className="text-[11px] text-muted-foreground uppercase tracking-wide">Résultats</p>
      <div className="flex h-3 rounded overflow-hidden">
        {segs.map((s, i) =>
          s.value > 0 ? (
            <div
              key={i}
              style={{ flex: s.value, backgroundColor: tokenCssVar(s.token) }}
              title={s.label}
            />
          ) : null,
        )}
      </div>
      <p className="text-[10px] text-muted-foreground">
        {kpis.wins} V · {kpis.losses} D
        {kpis.ties > 0 ? ` · ${kpis.ties} N` : ''}
        {kpis.dnf > 0 ? ` · ${kpis.dnf} DNF` : ''}
      </p>
    </div>
  )
}

interface ScoreCardProps {
  player: Player
  isActive: boolean
  isViewed: boolean
  badgeVsAvg?: 'above' | 'below' | null
  onClick?: () => void
  compact?: boolean
}

function ScoreCard({ player, isActive, isViewed, badgeVsAvg, onClick, compact }: ScoreCardProps) {
  const tier = getScoreTier(player.score)
  const clickable = !!onClick
  return (
    <button
      type="button"
      disabled={!clickable}
      onClick={onClick}
      className={[
        'rounded border px-3 py-2 text-left transition',
        clickable ? 'cursor-pointer hover:border-foreground/40' : 'cursor-default',
        isViewed ? 'border-foreground/60 bg-[#252b32]' : 'border-border bg-[#1d2328]',
      ].join(' ')}
    >
      <div className="flex items-center justify-between gap-2">
        <p className="text-[11px] uppercase tracking-wide text-muted-foreground">
          {player.gamertag}
          {isActive && <span className="ml-1 text-[10px] opacity-60">(moi)</span>}
        </p>
        {badgeVsAvg && (
          <TrendBadge state={badgeVsAvg === 'above' ? 'above' : 'below'} />
        )}
      </div>
      <div className="flex items-baseline gap-2 mt-1">
        <span className="text-2xl font-bold" style={{ color: tokenCssVar(tier.token) }}>
          {player.score}
        </span>
        {!compact && (
          <span className="text-xs text-muted-foreground">{tier.label}</span>
        )}
      </div>
    </button>
  )
}

// ─── Bandes ──────────────────────────────────────────────────────────────────

interface RailProps {
  kpis: PlayerKpis
  title: string
}

function RailDescriptive({ kpis, title }: RailProps) {
  return (
    <div className="flex items-center justify-between gap-6 px-4 py-3 bg-[#16191d] border border-border rounded">
      <div className="flex items-center gap-4">
        <span className="text-xs uppercase tracking-wider text-muted-foreground font-semibold">
          {title}
        </span>
        <span className="text-sm text-foreground">
          <strong>{kpis.matches}</strong> matchs
        </span>
        <span className="text-xs text-muted-foreground">·</span>
        <span className="text-sm text-foreground">
          ⌀ {formatMmss(kpis.avgMatchSec)}/match
        </span>
        <span className="text-xs text-muted-foreground">·</span>
        <span className="text-sm text-foreground">{formatDurationDhm(kpis.totalPlaySec)}</span>
      </div>
      <ResultsBar kpis={kpis} />
    </div>
  )
}

interface SquadVerdictProps {
  activePlayer: Player
  teammates: Player[]
  viewedXuid: string
  onSelectPlayer: (xuid: string) => void
}

function SquadVerdict({ activePlayer, teammates, viewedXuid, onSelectPlayer }: SquadVerdictProps) {
  const allPlayers = [activePlayer, ...teammates]
  const tier = getScoreTier(SQUAD_SCORE)
  const delta = Math.round(SQUAD_SCORE - SQUAD_BASE_AVG)
  const deltaText = delta === 0 ? 'base only' : delta > 0 ? `Δ +${delta} ▲` : `Δ ${delta} ▼`
  const deltaTok: SemanticToken =
    delta > 0 ? 'divergent-pos' : delta < 0 ? 'divergent-neg' : 'divergent-neutral'
  const avgScore = allPlayers.reduce((a, p) => a + p.score, 0) / allPlayers.length

  return (
    <div className="flex flex-wrap items-stretch gap-2 px-4 py-3 bg-[#16191d] border border-border rounded">
      {/* Team card */}
      <div className="rounded border border-border bg-[#1d2328] px-3 py-2 min-w-[180px]">
        <p className="text-[11px] uppercase tracking-wide text-muted-foreground">Score d'équipe</p>
        <div className="flex items-baseline gap-2 mt-1">
          <span className="text-2xl font-bold" style={{ color: tokenCssVar(tier.token) }}>
            {SQUAD_SCORE}
          </span>
          <span className="text-xs text-muted-foreground">{tier.label}</span>
          <span className="text-base font-bold ml-1 text-foreground">[{SQUAD_GRADE}]</span>
        </div>
        <p className="text-[10px] mt-0.5" style={{ color: tokenCssVar(deltaTok) }}>
          {deltaText} vs base
        </p>
      </div>

      {/* Player cards (clickables si onSelectPlayer fourni) */}
      {allPlayers.map((p) => {
        const isActive = p.xuid === activePlayer.xuid
        const isViewed = p.xuid === viewedXuid
        const badge =
          p.score > avgScore ? 'above' : p.score < avgScore ? 'below' : null
        return (
          <ScoreCard
            key={p.xuid}
            player={p}
            isActive={isActive}
            isViewed={isViewed}
            badgeVsAvg={badge}
            onClick={() => onSelectPlayer(p.xuid)}
          />
        )
      })}
    </div>
  )
}

interface KpiGridProps {
  player: Player
  teamAvg: PlayerKpis | null
  variant: 'A' | 'B'
  showHeader?: boolean
  isDrilledIn?: boolean
}

function KpiGrid({ player, teamAvg, variant, showHeader, isDrilledIn }: KpiGridProps) {
  const k = player.kpis
  const t = teamAvg

  const trendKills = trendVsTeam(k.killsPerMatch, t?.killsPerMatch)
  const trendDeaths = trendVsTeam(k.deathsPerMatch, t?.deathsPerMatch, true)
  const trendAssists = trendVsTeam(k.assistsPerMatch, t?.assistsPerMatch)
  const trendAcc = trendVsTeam(k.accuracyPct, t?.accuracyPct)
  const trendLife = trendVsTeam(k.lifespanSec, t?.lifespanSec)

  const cells = (
    <>
      {variant === 'B' && (
        <>
          <KpiCell label="Matchs joués" value={String(k.matches)} sub={`${formatMmss(k.avgMatchSec)}/match`} />
          <KpiCell label="Durée totale" value={formatDurationDhm(k.totalPlaySec)} />
        </>
      )}
      <KpiCell
        label="Frags par partie"
        value={k.killsPerMatch.toFixed(2)}
        sub={`${k.killsPerMin.toFixed(2)}/min`}
        trend={trendKills}
      />
      <KpiCell
        label="Morts par partie"
        value={k.deathsPerMatch.toFixed(2)}
        sub={`${k.deathsPerMin.toFixed(2)}/min`}
        trend={trendDeaths}
      />
      <KpiCell
        label="Assistances par partie"
        value={k.assistsPerMatch.toFixed(2)}
        sub={`${k.assistsPerMin.toFixed(2)}/min`}
        trend={trendAssists}
      />
      <KpiCell
        label="Précision moyenne"
        value={`${k.accuracyPct.toFixed(2)}%`}
        trend={trendAcc}
      />
      <KpiCell
        label="Durée de vie moyenne"
        value={formatMmss(k.lifespanSec)}
        trend={trendLife}
      />
      {variant === 'B' && (
        <div className="col-span-2 rounded border border-border bg-[#1d2328] px-3 py-2 flex items-center">
          <ResultsBar kpis={k} />
        </div>
      )}
    </>
  )

  return (
    <div>
      {showHeader && (
        <div className="flex items-center justify-between mb-2 px-1">
          <span className="text-xs uppercase tracking-wider text-muted-foreground font-semibold">
            {isDrilledIn ? `Vue : ${player.gamertag}` : `Mes stats sur cette session`}
          </span>
          {teamAvg && (
            <span className="text-[10px] text-muted-foreground">
              ▲/▼ vs moyenne d'équipe sur la session
            </span>
          )}
        </div>
      )}
      <div
        className={[
          'grid gap-2',
          variant === 'A' ? 'grid-cols-5' : 'grid-cols-8',
        ].join(' ')}
      >
        {cells}
      </div>
    </div>
  )
}

// ─── Variants Briefing ───────────────────────────────────────────────────────

interface BriefingProps {
  mode: Mode
  variant: 'A' | 'B'
}

function SessionBriefing({ mode, variant }: BriefingProps) {
  const [viewedXuid, setViewedXuid] = useState<string>(ACTIVE_PLAYER.xuid)

  const viewedPlayer =
    [ACTIVE_PLAYER, ...TEAMMATES].find((p) => p.xuid === viewedXuid) ?? ACTIVE_PLAYER

  // Moyenne d'équipe (toujours sur tous les joueurs de l'escouade, pas seulement les non-viewed)
  const teamAvg = useMemo<PlayerKpis | null>(() => {
    if (mode === 'solo') return null
    const all = [ACTIVE_PLAYER, ...TEAMMATES]
    return {
      matches: ACTIVE_PLAYER.kpis.matches,
      totalPlaySec: ACTIVE_PLAYER.kpis.totalPlaySec,
      avgMatchSec: ACTIVE_PLAYER.kpis.avgMatchSec,
      killsPerMatch: computeTeamAvg(all, (k) => k.killsPerMatch),
      killsPerMin: computeTeamAvg(all, (k) => k.killsPerMin),
      deathsPerMatch: computeTeamAvg(all, (k) => k.deathsPerMatch),
      deathsPerMin: computeTeamAvg(all, (k) => k.deathsPerMin),
      assistsPerMatch: computeTeamAvg(all, (k) => k.assistsPerMatch),
      assistsPerMin: computeTeamAvg(all, (k) => k.assistsPerMin),
      accuracyPct: computeTeamAvg(all, (k) => k.accuracyPct),
      lifespanSec: computeTeamAvg(all, (k) => k.lifespanSec),
      wins: 0, losses: 0, ties: 0, dnf: 0,
    }
  }, [mode])

  const isDrilledIn = viewedXuid !== ACTIVE_PLAYER.xuid

  return (
    <div className="space-y-2">
      {/* Variant A : rail descriptif en haut */}
      {variant === 'A' && (
        <RailDescriptive kpis={viewedPlayer.kpis} title="Ma session" />
      )}

      {/* Bande Squad (toujours en mode squad) */}
      {mode === 'squad' && (
        <SquadVerdict
          activePlayer={ACTIVE_PLAYER}
          teammates={TEAMMATES}
          viewedXuid={viewedXuid}
          onSelectPlayer={setViewedXuid}
        />
      )}

      {/* Reset drill-down */}
      {isDrilledIn && (
        <div className="flex items-center gap-2 px-1">
          <span className="text-xs text-muted-foreground">
            Vue active : {viewedPlayer.gamertag}
          </span>
          <button
            type="button"
            onClick={() => setViewedXuid(ACTIVE_PLAYER.xuid)}
            className="text-xs underline text-muted-foreground hover:text-foreground"
          >
            ✕ revenir à mes stats
          </button>
        </div>
      )}

      {/* KPI grid */}
      <KpiGrid
        player={viewedPlayer}
        teamAvg={teamAvg}
        variant={variant}
        showHeader
        isDrilledIn={isDrilledIn}
      />
    </div>
  )
}

// ─── Page sandbox ────────────────────────────────────────────────────────────

export function SessionBriefingMockPage() {
  const [mode, setMode] = useState<Mode>('squad')

  return (
    <div className="container mx-auto py-6 space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Session Briefing — mock comparatif</CardTitle>
          <p className="text-sm text-muted-foreground">
            Sandbox dev pour comparer 2 variantes de fusion (KPI bar + Squad verdict).
            Toggle solo/squad ci-dessous. En mode squad, clic sur une card joueur =
            drill-down dans la KPI grid (avec bouton retour).
          </p>
          <div className="flex gap-2 mt-3">
            <button
              type="button"
              onClick={() => setMode('solo')}
              className={`px-3 py-1.5 text-xs rounded border ${mode === 'solo' ? 'border-foreground bg-foreground/10' : 'border-border'}`}
            >
              Mode solo
            </button>
            <button
              type="button"
              onClick={() => setMode('squad')}
              className={`px-3 py-1.5 text-xs rounded border ${mode === 'squad' ? 'border-foreground bg-foreground/10' : 'border-border'}`}
            >
              Mode squad
            </button>
          </div>
        </CardHeader>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Variant A — Rail descriptif + Verdict + 5 KPI cards</CardTitle>
          <p className="text-xs text-muted-foreground">
            Matchs / Durée / Résultats regroupés dans un rail haut (descriptif).
            Les 5 cards évaluatives en dessous portent les trends.
          </p>
        </CardHeader>
        <CardContent>
          <SessionBriefing mode={mode} variant="A" />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Variant B — Verdict + 8 KPI cards (layout actuel)</CardTitle>
          <p className="text-xs text-muted-foreground">
            La bande KPI à 8 cards reste strictement identique à l'écran Python actuel.
            Seule la bande Squad verdict est ajoutée au-dessus en mode squad.
          </p>
        </CardHeader>
        <CardContent>
          <SessionBriefing mode={mode} variant="B" />
        </CardContent>
      </Card>

      <Card>
        <CardContent className="pt-4 text-xs text-muted-foreground space-y-1">
          <p>
            <strong>Sémantique trend ▲/▼</strong> : comparaison vs moyenne d'équipe sur
            la session sélectionnée (PAS vs all-time). En mode solo : pas de moyenne →
            pas de trend affiché.
          </p>
          <p>
            <strong>Drill-down</strong> : visible uniquement en mode squad. Le clic sur
            une card joueur (incluant la mienne) recalcule la KPI grid pour ce joueur.
            La moyenne d'équipe reste la référence (donc les trends gardent du sens).
          </p>
          <p>
            <strong>Données API</strong> : la KPI grid drillée nécessite un endpoint
            lazy-fetch <code>/api/squad/kpis/{`{xuid}`}</code> sur le scope courant —
            pas dans ce mock (statique).
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
