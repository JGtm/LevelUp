/**
 * HomePage — Accueil Mission Control (Slice 5).
 */
import { useState, useRef, useEffect, type CSSProperties } from 'react'
import { useParams, useNavigate } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { CompositeProgressBar } from '@/components/ui/composite-progress-bar'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { PrivacyBanner } from '@/components/ui/privacy-banner'
import { MatchCard } from '@/components/ui/match-card'
import { Carousel, CarouselItem } from '@/components/ui/carousel'
import { HomeBattlePassPanel } from './HomeBattlePassPanel'
import { HomeHeroBanner } from './HomeHeroBanner'
import { HomeChallengesList } from './HomeChallengesList'
import { HomeRecentPlaylistsCard } from './HomeRecentPlaylistsCard'
import { RecentMediaRail } from './RecentMediaRail'
import { ChallengesCarousel } from '@/features/prestige/components/ChallengesCarousel'
import { useHomePage, useSeasonPassPreview } from './queries'
import { useSetMatchFavorite } from '@/features/match-history/queries'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { useAppShellStore } from '@/stores/appShellStore'
import { getPerfColor } from '@/lib/perf-color'
import { kdScale, accuracyScale } from '@/lib/accessibility/scales'
import { tokenCssVar } from '@/lib/accessibility'
import type { HomeSkillPeakSummary, SessionSummaryItem, HighlightItem, HighlightSlide, HighlightValueColor } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { getHighlightText, resolveTitle, resolveLabel, resolveDetail, resolveColSpan } from './highlights.i18n'
import { getKPIText } from './kpi.i18n'
import { getSpartanIdentityText } from './spartanIdentity.i18n'

// Grille fine de 20 sous-unités sur lg+. On utilise arbitrary values Tailwind v4
// pour autoriser un span > 12 (les classes col-span-N s'arrêtent à 12). Les classes
// sont littérales (JIT) et conditionnées par le breakpoint lg.
const HIGHLIGHT_SPAN_CLASS: Record<number, string> = {
  1: 'lg:[grid-column:span_1/span_1]',
  2: 'lg:[grid-column:span_2/span_2]',
  3: 'lg:[grid-column:span_3/span_3]',
  4: 'lg:[grid-column:span_4/span_4]',
  5: 'lg:[grid-column:span_5/span_5]',
}

function KPICard({ label, value, compact = false }: { label: string; value: string | number; compact?: boolean }) {
  return (
    <div className={`flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted py-3 text-center ${compact ? 'px-2' : 'px-4'}`}>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-xl font-bold text-primary">{value}</p>
    </div>
  )
}

const HIGHLIGHT_COLOR_MAP: Record<string, string> = {
  positive:        tokenCssVar('divergent-pos'),
  warning:         tokenCssVar('perf-tier-2'),
  negative:        tokenCssVar('divergent-neg'),
  neutral:         tokenCssVar('perf-tier-3'),
  'perf-excellent': tokenCssVar('perf-tier-1'),
  'perf-good':     tokenCssVar('perf-tier-2'),
  'perf-ok':       tokenCssVar('perf-tier-3'),
  'perf-low':      tokenCssVar('perf-tier-4'),
  'perf-bad':      tokenCssVar('perf-tier-5'),
}

function highlightColorStyle(color?: HighlightValueColor): CSSProperties | undefined {
  if (!color) return undefined
  const cssVar = HIGHLIGHT_COLOR_MAP[color]
  return cssVar ? { color: cssVar } : undefined
}

function SerieTile({ title, slides, locale, className }: { title: string; slides: HighlightSlide[]; locale: string | null | undefined; className?: string }) {
  const [idx, setIdx] = useState(0)
  const [fading, setFading] = useState(false)
  const { data: fieldMappings } = useFieldMappings()
  useEffect(() => {
    if (slides.length <= 1) return
    const iv = window.setInterval(() => {
      setFading(true)
      window.setTimeout(() => {
        setIdx((i) => (i + 1) % slides.length)
        setFading(false)
      }, 250)
    }, 4000)
    return () => window.clearInterval(iv)
  }, [slides.length])
  const s = slides[idx]
  const slideLabel = s.label_key ? resolveLabel(locale, s.label_key, fieldMappings) : (s.label ?? '')
  const slideDetail = s.detail_key
    ? resolveDetail(locale, s.detail_key, s.detail_params)
    : (s.detail ?? '')
  return (
    <div className={`rounded-md border border-border p-3 ${className ?? ''}`}>
      <p className="text-xs font-medium text-muted-foreground leading-tight">{title}</p>
      <div
        className={`transition-opacity duration-200 ${fading ? 'opacity-0' : 'opacity-100'}`}
        aria-live="polite"
      >
        <p className="text-base font-bold" style={highlightColorStyle(s.value_color)}>{s.value}</p>
        <p className="text-[11px] text-muted-foreground/80 leading-tight">{slideLabel}</p>
        {slideDetail ? <p className="text-xs text-muted-foreground">{slideDetail}</p> : null}
      </div>
      {slides.length > 1 ? (
        <div className="mt-1 flex gap-1" aria-hidden="true">
          {slides.map((_, i) => (
            <span key={i} className={`h-1 w-4 rounded-full ${i === idx ? 'bg-primary' : 'bg-border'}`} />
          ))}
        </div>
      ) : null}
    </div>
  )
}

function HighlightTile({ h, locale }: { h: HighlightItem; locale: string | null | undefined }) {
  const title = h.title_key ? resolveTitle(locale, h.title_key) : (h.title ?? '')
  const spanClass = HIGHLIGHT_SPAN_CLASS[resolveColSpan(h.title_key)] ?? ''
  if (h.slides && h.slides.length > 0) {
    return <SerieTile title={title} slides={h.slides} locale={locale} className={spanClass} />
  }
  const detail = h.detail_key
    ? resolveDetail(locale, h.detail_key, h.detail_params)
    : (h.detail ?? '')
  return (
    <div className={`rounded-md border border-border p-3 ${spanClass}`}>
      <p className="text-xs font-medium text-muted-foreground">{title}</p>
      <p className="text-base font-bold" style={highlightColorStyle(h.value_color)}>{h.value}</p>
      {detail ? <p className="text-xs text-muted-foreground">{detail}</p> : null}
    </div>
  )
}

function ChevronUpIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="m18 15-6-6-6 6" />
    </svg>
  )
}

function ChevronDownIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="m6 9 6 6 6-6" />
    </svg>
  )
}

// Barre d'outcomes 4 segments proportionnels.
interface OutcomeBarProps { wins: number; draws: number; losses: number; dnfs: number }
function OutcomeBar({ wins, draws, losses, dnfs }: OutcomeBarProps) {
  const total = wins + draws + losses + dnfs
  if (total === 0) return null
  const pct = (n: number) => `${(n / total) * 100}%`
  return (
    <div className="flex h-1.5 w-full overflow-hidden rounded-full gap-px">
      {wins   > 0 && <div style={{ width: pct(wins),   backgroundColor: tokenCssVar('outcome-win') }} />}
      {draws  > 0 && <div style={{ width: pct(draws),  backgroundColor: tokenCssVar('outcome-draw') }} />}
      {dnfs   > 0 && <div style={{ width: pct(dnfs),   backgroundColor: tokenCssVar('outcome-dnf') }} />}
      {losses > 0 && <div style={{ width: pct(losses), backgroundColor: tokenCssVar('outcome-loss') }} />}
    </div>
  )
}

function formatSessionDate(startedAt: string | null): string {
  if (!startedAt) return ''
  const d = new Date(startedAt)
  return (
    d.toLocaleDateString('fr-FR', { day: 'numeric', month: 'short' }) +
    ' à ' +
    d.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })
  )
}

function formatSessionDuration(startedAt: string | null, endedAt: string | null): string {
  if (!startedAt || !endedAt) return ''
  const diffMs = new Date(endedAt).getTime() - new Date(startedAt).getTime()
  if (diffMs <= 0) return ''
  const totalMin = Math.round(diffMs / 60000)
  const h = Math.floor(totalMin / 60)
  const m = totalMin % 60
  if (h === 0) return `${m}min`
  return `${h}h${m > 0 ? String(m).padStart(2, '0') : ''}`
}

interface SessionCarouselCardProps {
  sessions: SessionSummaryItem[]
  idx: number
  onIdxChange: (idx: number) => void
  variant: 'solo' | 'squad'
  playerSlug: string
  onNavigate: (sessionLabel: string) => void
}

function SessionCarouselCard({ sessions, idx, onIdxChange, variant, onNavigate }: SessionCarouselCardProps) {
  const [displayIdx, setDisplayIdx] = useState(idx)
  const [isAnimating, setIsAnimating] = useState(false)
  const contentRef = useRef<HTMLDivElement>(null)
  const cleanupRef = useRef<(() => void) | null>(null)

  // Nettoyage si le composant est démonté en cours d'animation
  useEffect(() => () => { cleanupRef.current?.() }, [])

  const handleChange = (newIdx: number) => {
    if (isAnimating) return
    const el = contentRef.current
    if (!el) {
      setDisplayIdx(newIdx)
      onIdxChange(newIdx)
      return
    }

    const dir = newIdx < displayIdx ? 'up' : 'down'
    if (cleanupRef.current) cleanupRef.current()
    setIsAnimating(true)

    const exitY = dir === 'up' ? '-10px' : '10px'
    el.style.transition = 'transform 0.1s ease-in, opacity 0.1s ease-in'
    el.style.transform = `translateY(${exitY})`
    el.style.opacity = '0'

    const t1 = setTimeout(() => {
      setDisplayIdx(newIdx)
      onIdxChange(newIdx)
      const entryY = dir === 'up' ? '10px' : '-10px'
      el.style.transition = 'none'
      el.style.transform = `translateY(${entryY})`
      el.style.opacity = '0'

      const rafId = requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          el.style.transition = 'transform 0.15s ease-out, opacity 0.12s ease-out'
          el.style.transform = 'translateY(0)'
          el.style.opacity = '1'
          const t2 = setTimeout(() => setIsAnimating(false), 160)
          cleanupRef.current = () => clearTimeout(t2)
        })
      })
      cleanupRef.current = () => cancelAnimationFrame(rafId)
    }, 110)

    cleanupRef.current = () => clearTimeout(t1)
  }

  const session = sessions[displayIdx]
  const total = sessions.length
  const variantLabel = variant === 'solo' ? 'Solo' : 'Escouade'
  const cardClass = variant === 'solo' ? 'rounded-md bg-muted p-3' : 'rounded-md bg-primary/10 p-3'
  const chevronBottomCls = 'flex w-full cursor-pointer items-center justify-center py-2 text-muted-foreground/40 transition-colors hover:text-muted-foreground focus-visible:outline-none focus-visible:text-foreground disabled:cursor-not-allowed disabled:opacity-20'

  return (
    <div className="flex flex-col">
      {/* Ligne supérieure : label variant (absolu à gauche) + chevron haut centré */}
      <div className="relative mb-1 flex w-full items-center justify-center">
        <span className="absolute left-0 text-xs font-semibold text-muted-foreground">{variantLabel}</span>
        <button
          type="button"
          disabled={displayIdx === 0 || isAnimating}
          onClick={() => handleChange(displayIdx - 1)}
          className={chevronBottomCls}
          aria-label="Session plus récente"
        >
          <ChevronUpIcon />
        </button>
      </div>

      {/* Contenu animé */}
      <div ref={contentRef}>
        {session ? (
          <button
            type="button"
            className={`${cardClass} w-full cursor-pointer text-left hover:opacity-80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring`}
            onClick={() => onNavigate(session.session_label)}
            aria-label={`Voir le détail de la session ${session.session_label}`}
          >
            {/* Scores de performance */}
            <div className="mb-2 flex items-baseline gap-3">
              {session.avg_team_performance != null && (
                <>
                  <span
                    className="text-2xl font-black leading-none"
                    style={{ color: getPerfColor(session.avg_team_performance) }}
                  >
                    {Math.round(session.avg_team_performance)}
                  </span>
                  <span className="text-xs text-muted-foreground">Équipe</span>
                </>
              )}
              {session.avg_player_performance != null && (
                <>
                  <span
                    className="text-2xl font-black leading-none"
                    style={{ color: getPerfColor(session.avg_player_performance) }}
                  >
                    {Math.round(session.avg_player_performance)}
                  </span>
                  <span className="text-xs text-muted-foreground">Perso</span>
                </>
              )}
            </div>

            {/* Barre d'outcomes */}
            <OutcomeBar
              wins={session.wins}
              draws={session.draws}
              losses={session.losses}
              dnfs={session.dnfs}
            />

            {/* Décompte des outcomes avec nombre de parties */}
            <p className="mt-1.5 flex flex-wrap gap-x-2 text-xs">
              <span className="font-medium text-foreground">{session.match_count} partie{session.match_count > 1 ? 's' : ''}</span>
              {session.wins > 0 && (
                <span style={{ color: tokenCssVar('outcome-win') }}>{session.wins} Victoire{session.wins > 1 ? 's' : ''}</span>
              )}
              {session.losses > 0 && (
                <span style={{ color: tokenCssVar('outcome-loss') }}>{session.losses} Défaite{session.losses > 1 ? 's' : ''}</span>
              )}
              {session.draws > 0 && (
                <span style={{ color: tokenCssVar('outcome-draw') }}>{session.draws} Égalité{session.draws > 1 ? 's' : ''}</span>
              )}
              {session.dnfs > 0 && (
                <span style={{ color: tokenCssVar('outcome-dnf') }}>{session.dnfs} Non terminé{session.dnfs > 1 ? 's' : ''}</span>
              )}
            </p>

            {/* FDA moyen + playlist dominante + mode dominant */}
            <p className="mt-0.5 text-xs text-muted-foreground">
              {session.avg_kda != null && (
                <span>
                  FDA{' '}
                  <span style={{ color: tokenCssVar(kdScale(session.avg_kda)) }}>
                    {session.avg_kda.toFixed(2)}
                  </span>
                </span>
              )}
              {(session.avg_kda != null && (session.dominant_playlist || session.dominant_mode)) && ' · '}
              {session.dominant_playlist && (
                <span className="font-bold text-white">{session.dominant_playlist}</span>
              )}
              {session.dominant_playlist && session.dominant_mode && ' · '}
              {session.dominant_mode && (
                <span className="font-bold text-white">{session.dominant_mode}</span>
              )}
              {(session.dominant_playlist || session.dominant_mode) && (
                <span className="ml-1 inline-flex">
                  <InfoTooltip
                    content="Playlist et mode les plus joués lors de cette session"
                    iconClass="w-3 h-3"
                  />
                </span>
              )}
            </p>

            {/* Date de début + durée */}
            {session.started_at && (
              <p className="mt-0.5 text-xs text-muted-foreground">
                {formatSessionDate(session.started_at)}
                {session.ended_at
                  ? ` · Durée de la session : ${formatSessionDuration(session.started_at, session.ended_at)}`
                  : null}
              </p>
            )}
          </button>
        ) : null}
      </div>

      {/* Chevron bas */}
      <button
        type="button"
        disabled={displayIdx >= total - 1 || isAnimating}
        onClick={() => handleChange(displayIdx + 1)}
        className={chevronBottomCls}
        aria-label="Session plus ancienne"
      >
        <ChevronDownIcon />
      </button>
    </div>
  )
}

interface HomeSkillPeakCardProps {
  label: string
  peak: HomeSkillPeakSummary | null
  numberLocale: string
  testIdPrefix: string
  state: 'value' | 'placement' | 'neutral' | 'absent'
  detail: string
}

function HomeSkillPeakCard({ label, peak, numberLocale, testIdPrefix, state, detail }: HomeSkillPeakCardProps) {
  const isPlacement = state === 'placement'
  const hasValue = Boolean(peak)

  return (
    <div
      data-testid={`${testIdPrefix}-card`}
      className={`flex h-full min-w-[11rem] items-center gap-3 rounded-2xl border px-4 py-3 shadow-[0_12px_30px_rgba(8,15,28,0.24)] backdrop-blur-sm ${hasValue ? 'border-cyan-100/12 bg-slate-950/35' : 'border-white/10 bg-slate-950/22'}`}
    >
      {peak?.badge_image_url ? (
        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-slate-950/60 p-1.5">
          <img
            data-testid={`${testIdPrefix}-badge`}
            src={peak.badge_image_url}
            alt={label}
            className="h-full w-full object-contain"
            loading="lazy"
            decoding="async"
          />
        </div>
      ) : isPlacement ? (
        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-slate-950/60 p-1.5">
          <img
            data-testid={`${testIdPrefix}-unranked`}
            src="/static/ranks/Unranked.png"
            alt="En placement"
            className="h-full w-full object-contain opacity-80"
            loading="lazy"
            decoding="async"
          />
        </div>
      ) : (
        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-slate-950/60 text-[10px] font-semibold uppercase tracking-[0.24em] text-cyan-100/75">
          {label.replace(/[^A-Z]/gi, '').slice(0, 4) || 'MMR'}
        </div>
      )}

      <div className="min-w-0">
        <p className="text-[11px] uppercase tracking-[0.24em] text-cyan-100/68">{label}</p>
        <p data-testid={`${testIdPrefix}-value`} className="mt-1 text-xl font-semibold text-white sm:text-2xl">
          {peak ? peak.rating_value.toLocaleString(numberLocale, { maximumFractionDigits: 0 }) : '—'}
        </p>
        {(peak?.tier_label || detail) && (
          <p
            data-testid={peak?.tier_label ? `${testIdPrefix}-tier` : `${testIdPrefix}-detail`}
            className="truncate text-xs text-cyan-100/78"
          >
            {peak?.tier_label ?? detail}
          </p>
        )}
      </div>
    </div>
  )
}

function resolveSkillPeakState(
  peak: HomeSkillPeakSummary | null,
  hasHistory: boolean,
  mode: 'ranked' | 'unranked',
): Pick<HomeSkillPeakCardProps, 'state' | 'detail'> {
  if (peak) {
    return { state: 'value', detail: '' }
  }
  if (hasHistory) {
    return mode === 'ranked'
      ? { state: 'placement', detail: 'En placement' }
      : { state: 'neutral', detail: 'Sans classement' }
  }
  return {
    state: 'absent',
    detail: mode === 'ranked' ? 'Aucune partie classée' : 'Aucune partie non classée',
  }
}

export function HomePage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const navigate = useNavigate()
  const locale = useAppShellStore((s) => s.locale)
  const userTimezone = useAppShellStore((s) => s.userTimezone)
  const { data: fieldMappings } = useFieldMappings()
  const { data, isLoading, isError, refetch } = useHomePage(playerSlug)
  const {
    data: seasonPass,
    isLoading: isSeasonPassLoading,
    error: seasonPassError,
  } = useSeasonPassPreview(playerSlug)
  const [matchTab, setMatchTab] = useState<'recent' | 'favorites'>('recent')
  const [soloIdx, setSoloIdx] = useState(0)
  const [squadIdx, setSquadIdx] = useState(0)
  const favoriteMutation = useSetMatchFavorite(playerSlug)

  function goToMatch(matchId: string) {
    void navigate({
      to: '/players/$playerSlug/matches/$matchId',
      params: { playerSlug, matchId },
    })
  }

  function goToSession(sessionLabel: string) {
    void navigate({
      to: '/players/$playerSlug/stats/sessions',
      params: { playerSlug },
      search: { session: sessionLabel },
    })
  }

  if (isLoading) return null

  if (isError) {
    return (
      <div className="flex min-h-[55vh] items-center justify-center px-6 py-10">
        <div className="mx-auto w-full max-w-lg">
          <EmptyStateCard
            title="Accueil indisponible"
            description="La page d'accueil n'a pas pu être chargée pour ce joueur. Vérifie la session ou relance la requête."
            actionLabel="Réessayer"
            onAction={() => refetch()}
          />
        </div>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="flex min-h-[55vh] items-center justify-center px-6 py-10">
        <div className="mx-auto w-full max-w-lg">
          <EmptyStateCard
            title="Accueil vide"
            description="Aucune donnée d'accueil n'a été renvoyée pour ce joueur. Vérifie le bootstrap ou les données locales avant de continuer."
            actionLabel="Relancer"
            onAction={() => refetch()}
          />
        </div>
      </div>
    )
  }

  const { hero } = data
  const highlights = data.highlights ?? []
  const recentMatches = data.recent_matches ?? []
  const favoriteMatches = data.favorite_matches ?? []
  const spartanIdentity = data.spartan_identity ?? null
  const highestCSR = spartanIdentity?.highest_csr ?? null
  const highestLUSR = spartanIdentity?.highest_lusr ?? null
  const hasRankedHistory = data.has_ranked_history ?? Boolean(highestCSR)
  const hasUnrankedHistory = data.has_unranked_history ?? Boolean(highestLUSR)
  const hasAnySkillHistory = hasRankedHistory || hasUnrankedHistory
  const csrState = resolveSkillPeakState(highestCSR, hasRankedHistory, 'ranked')
  const lusrState = resolveSkillPeakState(highestLUSR, hasUnrankedHistory, 'unranked')
  const careerRank = spartanIdentity?.career_rank ?? null
  const careerAdornmentUrl = careerRank?.adornment_image_url ?? null
  const identityMonogram = hero.player_name.trim().slice(0, 1).toUpperCase() || 'S'
  const soloSession = data.solo_session ?? null
  const squadSession = data.squad_session ?? null
  const soloSessions = data.solo_sessions ?? (soloSession ? [soloSession] : [])
  const squadSessions = data.squad_sessions ?? (squadSession ? [squadSession] : [])
  const challenges = seasonPass?.challenges
  const challengesCompleted = challenges?.completed ?? 0
  const challengesTotal = challenges?.total ?? 0
  const challengesCompletedLabel = challenges?.available
    ? `${challengesCompleted} / ${challengesTotal} complétés`
    : null
  const numberLocale = locale === 'en' ? 'en-US' : 'fr-FR'
  const kpiText = getKPIText(locale)
  // Phase D multi-titres : résout les libellés métier via le backend TOML.
  // Fallback gracieux sur les libellés locaux de kpi.i18n.ts si l'endpoint
  // est absent (flag MULTI_TITLE_API_ENABLED off ou 404).
  const labelOf = (key: string, fallback: string): string =>
    fieldMappings?.fields[key]?.label ?? fallback
  const spartanText = getSpartanIdentityText(locale)
  const labels = spartanText.labels
  const hasPrivacyWarning = !!data.privacy_warning?.level && data.privacy_warning.level !== 'none'
  const emptySkillPanelTitle = hasPrivacyWarning
    ? spartanText.emptyPanel.titleUnavailable
    : spartanText.emptyPanel.titleNone
  const emptySkillPanelDescription = hasPrivacyWarning
    ? spartanText.emptyPanel.descriptionUnavailable
    : spartanText.emptyPanel.descriptionNone

  return (
    <div className="relative isolate flex flex-col">
      <div className="sticky top-0 z-0 px-6 pt-0" data-testid="home-hero-banner-sticky">
        <HomeHeroBanner />
      </div>

      <div className="relative z-10 space-y-6 bg-background px-6 pb-6 pt-6">
        {/* B9 : signal discret si données partielles (compte privé) */}
        <PrivacyBanner warning={data.privacy_warning} />

        {/* Hero KPIs */}
        <div>
          <div className="space-y-4">
            {spartanIdentity && (
              <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,19.5rem)] lg:items-stretch">
                <div className="overflow-hidden rounded-2xl border border-border bg-muted/60 shadow-sm">
                  <div
                    data-testid="home-spartan-identity-banner"
                    className="relative overflow-hidden bg-slate-950"
                  >
                    {spartanIdentity.banner_image_url && (
                      <img
                        data-testid="home-spartan-banner-surface"
                        src={spartanIdentity.banner_image_url}
                        alt=""
                        aria-hidden="true"
                        className="absolute inset-0 h-full w-full object-cover"
                        loading="lazy"
                        decoding="async"
                      />
                    )}
                    {careerAdornmentUrl && (
                      <div className="pointer-events-none absolute right-2 top-0 z-[1] flex h-full items-start">
                        <img
                          data-testid="home-spartan-adornment-image"
                          src={careerAdornmentUrl}
                          alt=""
                          aria-hidden="true"
                          className="h-full w-auto object-contain object-top drop-shadow-[0_14px_20px_rgba(8,15,28,0.48)]"
                          loading="lazy"
                          decoding="async"
                        />
                      </div>
                    )}
                    <div
                      data-testid="home-spartan-banner-shell"
                      className="relative flex flex-col gap-6 pt-1 pb-5 pl-5 pr-28 text-white sm:pl-6 sm:pr-32 lg:min-h-[9rem] lg:flex-row lg:items-start lg:justify-between"
                      style={{ textShadow: '0 1px 6px rgba(0,0,0,0.85)' }}
                    >
                      <div className="flex min-w-0 items-center gap-4 lg:self-center">
                        <div className="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-full border-2 border-cyan-300/60 bg-slate-950/60 shadow-[0_0_0_4px_rgba(8,15,28,0.35)] sm:h-24 sm:w-24">
                          {spartanIdentity.emblem_image_url ? (
                            <img
                              data-testid="home-spartan-emblem-image"
                              src={spartanIdentity.emblem_image_url}
                              alt={`Emblème ${hero.player_name}`}
                              className="h-full w-full object-cover"
                              loading="lazy"
                              decoding="async"
                            />
                          ) : (
                            <span className="text-3xl font-semibold tracking-[0.18em] text-cyan-100">{identityMonogram}</span>
                          )}
                        </div>

                        <div className="min-w-0">
                          <p
                            data-testid="home-spartan-gamertag"
                            className="truncate text-3xl font-semibold text-white sm:text-4xl"
                          >
                            {hero.player_name}
                          </p>
                          {spartanIdentity.spartan_id ? (
                            <p
                              data-testid="home-spartan-id-value"
                              className="mt-2 text-2xl font-medium italic tracking-[0.34em] text-cyan-50 sm:text-3xl"
                            >
                              {spartanIdentity.spartan_id}
                            </p>
                          ) : (
                            <p className="mt-2 text-sm text-cyan-100/70">Identité Spartan indisponible</p>
                          )}
                        </div>
                      </div>

                      {careerRank && (
                        <div className="flex items-center gap-4 self-start">
                          <div className="min-w-0 rounded-xl bg-slate-950/15 px-3 py-2 text-right backdrop-blur-sm lg:max-w-[16rem]">
                            <p data-testid="home-career-rank-title" className="text-lg font-semibold text-white sm:text-xl">
                              {careerRank.rank_title}
                            </p>
                          </div>
                        </div>
                      )}
                    </div>
                  </div>

                  {careerRank && (
                    <div className="border-t border-border/70 bg-background/80 px-5 py-4 sm:px-6">
                      <div className="space-y-3">
                        <div className="flex items-center justify-between gap-3 text-xs text-muted-foreground">
                          <span>
                            {careerRank.is_max_rank
                              ? labels.maxRank
                              : labels.progressTowardsRank(careerRank.next_rank_title ?? '')}
                          </span>
                          <span>
                            {careerRank.is_max_rank
                              ? labels.maxRank
                              : `${careerRank.progress_pct.toLocaleString(numberLocale, { maximumFractionDigits: 0 })} %`}
                          </span>
                        </div>
                        <div className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3">
                          <span
                            data-testid="home-career-rank-progress-current"
                            className="shrink-0 whitespace-nowrap text-[11px] font-medium text-foreground/85 sm:text-xs"
                          >
                            {`${careerRank.current_xp.toLocaleString(numberLocale)} XP`}
                          </span>
                          <div className="min-w-0">
                            <CompositeProgressBar
                              value={careerRank.progress_pct}
                              fillTestId="home-career-rank-progress-fill"
                            />
                          </div>
                          <span
                            data-testid="home-career-rank-progress-target"
                            className="shrink-0 whitespace-nowrap text-[11px] font-medium text-foreground/85 sm:text-xs"
                          >
                            {careerRank.is_max_rank
                              ? labels.maxRank
                              : `${careerRank.xp_for_next_rank.toLocaleString(numberLocale)} XP`}
                          </span>
                        </div>
                      </div>
                    </div>
                  )}
                </div>

                <div
                  data-testid="home-skill-peaks-panel"
                  className="grid gap-3 sm:grid-cols-2 lg:grid-cols-1 lg:auto-rows-fr"
                >
                  {!highestCSR && !highestLUSR && !hasAnySkillHistory ? (
                    <div
                      data-testid="home-skill-peaks-empty"
                      className="rounded-2xl border border-dashed border-white/10 bg-slate-950/22 px-4 py-4 text-white shadow-[0_12px_30px_rgba(8,15,28,0.2)]"
                    >
                      <p className="text-sm font-semibold">{emptySkillPanelTitle}</p>
                      <p className="mt-2 text-sm text-cyan-100/72">{emptySkillPanelDescription}</p>
                    </div>
                  ) : (
                    <>
                      <HomeSkillPeakCard
                        label={labels.highestCsr}
                        peak={highestCSR}
                        numberLocale={numberLocale}
                        testIdPrefix="home-highest-csr"
                        state={csrState.state}
                        detail={csrState.detail}
                      />
                      <HomeSkillPeakCard
                        label={labels.highestLusr}
                        peak={highestLUSR}
                        numberLocale={numberLocale}
                        testIdPrefix="home-highest-lusr"
                        state={lusrState.state}
                        detail={lusrState.detail}
                      />
                    </>
                  )}
                </div>
              </div>
            )}

            <div className="kpi-stats-grid items-stretch">
              {/* 1 — Parties */}
              <KPICard label={labelOf('total_matches_played', 'Parties')} value={hero.kpis.total_matches.toLocaleString(numberLocale)} compact />

              {/* 2 — KDA/FDA coloré comme les tuiles match */}
              {(() => {
                const kda = hero.kpis.avg_kda
                const kdaStyle = kda != null ? { color: tokenCssVar(kdScale(kda)) } : undefined
                return (
                  <div className="flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted px-2 py-3 text-center">
                    <p className="text-xs text-muted-foreground">{labelOf('kda', 'KDA')}</p>
                    <p className="text-xl font-bold text-muted-foreground" style={kdaStyle}>{kda != null ? kda.toFixed(2) : '—'}</p>
                  </div>
                )
              })()}

              {/* 3 — Taux de victoire + barre composite outcomes */}
              {(() => {
                const wins = hero.kpis.wins
                const losses = hero.kpis.losses
                const draws = hero.kpis.draws ?? 0
                const dnfs = hero.kpis.dnfs ?? 0
                const neutral = draws + dnfs
                return (
                  <div className="flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted px-4 py-3 text-center">
                    <p className="text-xs text-muted-foreground">{labelOf('win_rate', 'Taux de victoire')}</p>
                    <p className="text-xl font-bold text-primary">{`${(hero.kpis.win_rate * 100).toFixed(0)}%`}</p>
                    <div className="mt-2 w-full">
                      <OutcomeBar wins={wins} draws={draws} losses={losses} dnfs={dnfs} />
                    </div>
                    <div className="mt-1.5 flex justify-center gap-3 text-xs font-semibold tabular-nums">
                      <span style={{ color: tokenCssVar('outcome-win') }}>{wins}</span>
                      {neutral > 0 && <span style={{ color: tokenCssVar('outcome-draw') }}>{neutral}</span>}
                      <span style={{ color: tokenCssVar('outcome-loss') }}>{losses}</span>
                    </div>
                  </div>
                )
              })()}

              {/* 4 — Durée totale */}
              {(() => {
                const secs = hero.kpis.total_playtime_secs ?? 0
                let formatted: string
                if (secs <= 0) {
                  formatted = '—'
                } else {
                  const totalMin = Math.floor(secs / 60)
                  const h = Math.floor(totalMin / 60)
                  const totalDays = Math.floor(h / 24)
                  if (totalDays >= 365) {
                    const years = Math.floor(totalDays / 365)
                    const remDays = totalDays % 365
                    const months = Math.floor(remDays / 30)
                    const days = remDays % 30
                    const parts = [`${years}${kpiText.units.year}`]
                    if (months > 0) parts.push(`${months}${kpiText.units.month}`)
                    if (days > 0) parts.push(`${days}${kpiText.units.day}`)
                    formatted = parts.join(' ')
                  } else if (totalDays >= 30) {
                    const months = Math.floor(totalDays / 30)
                    const days = totalDays % 30
                    const parts = [`${months}${kpiText.units.month}`]
                    if (days > 0) parts.push(`${days}${kpiText.units.day}`)
                    formatted = parts.join(' ')
                  } else if (totalDays >= 1) {
                    const remH = h % 24
                    formatted = remH > 0 ? `${totalDays}${kpiText.units.day} ${remH}${kpiText.units.hour}` : `${totalDays}${kpiText.units.day}`
                  } else {
                    const m = totalMin % 60
                    formatted = h === 0 ? `${m}${kpiText.units.minute}` : `${h}${kpiText.units.hour}${m > 0 ? String(m).padStart(2, '0') : ''}`
                  }
                }
                return <KPICard label={kpiText.labels.totalTime} value={formatted} />
              })()}

              {/* 5 — Playlist favorite */}
              <div className="flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted px-4 py-3 text-center">
                <p className="text-xs text-muted-foreground">{kpiText.labels.favoritePlaylist}</p>
                <p className="w-full truncate text-sm font-bold text-primary leading-tight mt-1">
                  {hero.kpis.favorite_playlist_name || '—'}
                </p>
                {hero.kpis.favorite_playlist_count > 0 && (
                  <p className="text-xs text-muted-foreground mt-0.5">
                    {hero.kpis.favorite_playlist_count.toLocaleString(numberLocale)} {kpiText.matches(hero.kpis.favorite_playlist_count)}
                  </p>
                )}
              </div>

              {/* 7 — Rendement / Résistance (barre composite) */}
              {(() => {
                const offConv = hero.kpis.avg_offensive_conversion
                const defRes = hero.kpis.avg_defensive_resistance
                const hasData = offConv != null || defRes != null
                const off = offConv ?? 0
                const def = defRes ?? 0
                const total = off + def
                return (
                  <div className="flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted px-4 py-3 text-center">
                    <p className="text-xs text-muted-foreground mb-1.5">{kpiText.labels.offDef}</p>
                    {hasData ? (
                      <div className="w-full">
                        <div className="h-2 w-full rounded-full overflow-hidden flex">
                          {off > 0 && <div className="h-full" style={{ width: total > 0 ? `${(off / total) * 100}%` : '50%', backgroundColor: tokenCssVar('divergent-pos') }} />}
                          {def > 0 && <div className="h-full" style={{ width: total > 0 ? `${(def / total) * 100}%` : '50%', backgroundColor: tokenCssVar('divergent-neutral') }} />}
                        </div>
                        <div className="flex justify-center gap-3 mt-2">
                          <span className="text-sm font-bold leading-none" style={{ color: tokenCssVar('divergent-pos') }}>{off.toFixed(2)}</span>
                          <span className="text-sm font-bold leading-none" style={{ color: tokenCssVar('divergent-neutral') }}>{def.toFixed(2)}</span>
                        </div>
                      </div>
                    ) : (
                      <p className="text-xl font-bold text-muted-foreground">—</p>
                    )}
                  </div>
                )
              })()}

              {/* 8 — Précision avec code couleur */}
              {(() => {
                const acc = hero.kpis.avg_accuracy
                const accStyle = acc != null ? { color: tokenCssVar(accuracyScale(acc)) } : undefined
                return (
                  <div className="flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted px-2 py-3 text-center">
                    <p className="text-xs text-muted-foreground">{labelOf('accuracy', 'Précision')}</p>
                    <p className="text-xl font-bold text-primary" style={accStyle}>{acc != null ? `${acc.toFixed(0)}%` : '—'}</p>
                  </div>
                )
              })()}

              {/* 9 — Arme favorite */}
              <div className="flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted px-4 py-3 text-center">
                <p className="text-xs text-muted-foreground">{kpiText.labels.favoriteWeapon}</p>
                <p className="w-full truncate text-sm font-bold text-primary leading-tight mt-1">
                  {hero.kpis.favorite_weapon_name || '—'}
                </p>
                {hero.kpis.favorite_weapon_kills > 0 && (
                  <p className="text-xs text-muted-foreground mt-0.5">
                    {hero.kpis.favorite_weapon_kills.toLocaleString(numberLocale)} {kpiText.kills(hero.kpis.favorite_weapon_kills)}
                  </p>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Battle Pass + Défis */}
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(320px,0.65fr)]">
          <HomeBattlePassPanel
            loading={isSeasonPassLoading}
            data={seasonPass}
            errorHint={seasonPassError instanceof Error ? seasonPassError.message : null}
          />

          <Card data-testid="home-challenges-card" className="flex min-h-[14rem] self-start flex-col">
            <CardHeader className="flex flex-row items-center justify-between gap-3 space-y-0">
              <CardTitle className="text-base">Défis actifs</CardTitle>
              {challengesCompletedLabel && (
                <p data-testid="home-challenges-completed" className="shrink-0 text-sm text-foreground">
                  <strong className="text-primary">{challengesCompleted}</strong> / {challengesTotal} complétés
                </p>
              )}
            </CardHeader>
            <CardContent className="flex flex-1 flex-col">
              {isSeasonPassLoading && !seasonPass ? (
                <div className="flex flex-1 items-center justify-center">
                  <p className="text-sm text-muted-foreground">Chargement des défis…</p>
                </div>
              ) : challenges?.available ? (
                <div className="space-y-3">
                  {Array.isArray(challenges.items) && challenges.items.length > 0 ? (
                    <HomeChallengesList items={challenges.items} />
                  ) : (
                    <div className="flex min-h-[8.5rem] items-center justify-center">
                      <EmptyStateNotice
                        title="Aucun défi actif"
                        description="Tous les défis visibles sont terminés ou aucun défi détaillé n'est disponible pour le moment."
                        className="w-full max-w-sm"
                      />
                    </div>
                  )}
                  {challenges.xp_available != null && challenges.xp_available > 0 && (
                    <p className="text-xs text-muted-foreground">{challenges.xp_available.toLocaleString('fr-FR')} XP disponibles</p>
                  )}
                </div>
              ) : seasonPassError ? (
                <div className="flex flex-1 items-center justify-center">
                  <EmptyStateNotice
                    title="Défis indisponibles"
                    description="Les défis actifs n'ont pas pu être chargés pour le moment."
                    className="w-full max-w-sm"
                  />
                </div>
              ) : (
                <div className="flex flex-1 items-center justify-center">
                  <EmptyStateNotice
                    title="Défis indisponibles"
                    description="Aucune donnée de défis n'est disponible pour le moment."
                    className="w-full max-w-sm"
                  />
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Playlists récentes + Rang | Sessions récentes */}
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(320px,0.65fr)_minmax(0,1.35fr)]">
          <HomeRecentPlaylistsCard
            recentPlaylistRanks={data.recent_playlist_ranks}
          />

          {/* Sessions récentes */}
          <Card>
            <CardHeader className="space-y-0 pb-3">
              <CardTitle className="text-base">Sessions récentes</CardTitle>
            </CardHeader>
            <CardContent>
              {soloSessions.length > 0 || squadSessions.length > 0 ? (
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  {soloSessions.length > 0 && (
                    <SessionCarouselCard
                      sessions={soloSessions}
                      idx={soloIdx}
                      onIdxChange={setSoloIdx}
                      variant="solo"
                      playerSlug={playerSlug}
                      onNavigate={goToSession}
                    />
                  )}
                  {squadSessions.length > 0 && (
                    <SessionCarouselCard
                      sessions={squadSessions}
                      idx={squadIdx}
                      onIdxChange={setSquadIdx}
                      variant="squad"
                      playerSlug={playerSlug}
                      onNavigate={goToSession}
                    />
                  )}
                </div>
              ) : (
                <EmptyStateNotice
                  title="Aucune session récente disponible"
                  description="Aucune session solo ou escouade n'a été calculée pour le scope actuel."
                />
              )}
            </CardContent>
          </Card>
        </div>

        {/* Carousel défis Prestige (au-dessus des Faits Marquants) */}
        <Card>
          <CardContent className="pt-4">
            <ChallengesCarousel
              userId={playerSlug}
              titleSlug="halo_infinite"
              playerSlug={playerSlug}
            />
          </CardContent>
        </Card>

        {/* Highlights */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-1.5 text-base">
              {getHighlightText(locale).section.title}
              <InfoTooltip content={<p>{getHighlightText(locale).section.tooltipIntro}</p>} />
            </CardTitle>
          </CardHeader>
          <CardContent>
            {highlights.length > 0 ? (
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:[grid-template-columns:repeat(20,minmax(0,1fr))]">
                {highlights.map((h, i) => (
                  <HighlightTile key={i} h={h} locale={locale} />
                ))}
              </div>
            ) : (
              <EmptyStateNotice
                title="Aucun point saillant disponible"
                description="Le backend n'a renvoyé aucun highlight exploitable pour cette période."
              />
            )}
          </CardContent>
        </Card>

        {/* Matchs récents / Favoris — 4 tuiles MatchCard */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">
                {matchTab === 'recent' ? 'Matchs récents' : 'Matchs favoris'}
              </CardTitle>
              <div className="flex items-center gap-1 border-b border-transparent">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setMatchTab('recent')}
                  className={`rounded-none border-b-2 px-3 py-1.5 text-xs ${
                    matchTab === 'recent'
                      ? 'border-primary text-primary font-semibold'
                      : 'border-transparent text-muted-foreground hover:text-foreground'
                  }`}
                >
                  Récents
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setMatchTab('favorites')}
                  className={`rounded-none border-b-2 px-3 py-1.5 text-xs ${
                    matchTab === 'favorites'
                      ? 'border-primary text-primary font-semibold'
                      : 'border-transparent text-muted-foreground hover:text-foreground'
                  }`}
                >
                  Favoris
                  {favoriteMatches.length > 0 && (
                    <span className="ml-1.5 rounded-full bg-amber-500/20 px-1.5 py-0.5 text-[10px] text-amber-400">
                      {favoriteMatches.length}
                    </span>
                  )}
                </Button>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {matchTab === 'recent' ? (
              recentMatches.length > 0 ? (
                <Carousel>
                  {recentMatches.map((m) => (
                    <CarouselItem key={m.match_id} className="w-72">
                      <MatchCard
                        match={m}
                        locale={locale}
                        timezone={userTimezone}
                        onClick={() => goToMatch(m.match_id)}
                        onToggleFavorite={() =>
                          favoriteMutation.mutate({ matchId: m.match_id, favorite: !m.is_favorite })
                        }
                        favoriteDisabled={favoriteMutation.isPending && favoriteMutation.variables?.matchId === m.match_id}
                      />
                    </CarouselItem>
                  ))}
                </Carousel>
              ) : (
                <EmptyStateNotice
                  title="Aucun match récent disponible"
                  description="Les matchs récents apparaîtront ici après une synchronisation."
                />
              )
            ) : (
              favoriteMatches.length > 0 ? (
                <Carousel>
                  {favoriteMatches.map((m) => (
                    <CarouselItem key={m.match_id} className="w-72">
                      <MatchCard
                        match={m}
                        locale={locale}
                        timezone={userTimezone}
                        onClick={() => goToMatch(m.match_id)}
                        onToggleFavorite={() =>
                          favoriteMutation.mutate({ matchId: m.match_id, favorite: false })
                        }
                        favoriteDisabled={favoriteMutation.isPending && favoriteMutation.variables?.matchId === m.match_id}
                      />
                    </CarouselItem>
                  ))}
                </Carousel>
              ) : (
                <EmptyStateNotice
                  title="Aucun match favori"
                  description="Marquez vos meilleurs matchs avec l'icône étoile pour les retrouver ici."
                />
              )
            )}
          </CardContent>
        </Card>

        <RecentMediaRail playerSlug={playerSlug} />
      </div>
    </div>
  )
}
