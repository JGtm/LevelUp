// cross-feature-allow: tab orchestrateur Ascension — agrège les composants
// prestige (ChallengeCard, ArcSummary…) et coach (proposals).
/**
 * AscensionProfileTab — tab "Profil & objectifs" (refonte 2026-05-26).
 *
 * Composition (verticale, du plus actionnable au plus analytique) :
 *   1. CoachProposalsCard — suggestions proactives
 *   2. CampaignTracker     — campagne en cours (si active)
 *   3. PlayerProfileV3     — identité, style, performance, leviers + CTAs
 *   4. Mes objectifs actifs — pilot toggle, libres, pilotés, créer
 *   5. Mes arcs en cours
 *   6. Patterns contextuels + comportementaux + leviers calibrés
 *
 * Tout le contenu V1 (Profil) et V2 (Patterns + Coach) est préservé. La
 * navigation Coach → création d'objectif reste sur le même écran (modale
 * StartCampaign).
 */
import { useState } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettings } from '@/features/settings/queries'
import { CoachProposalsCard } from '@/features/coach/CoachProposalsCard'
import { getCoachStrings } from '@/features/coach/i18n'
import { CampaignTracker } from './campaign/CampaignTracker'
import { StartCampaignModal } from './campaign/StartCampaignModal'
import { CreateChallengeForm } from '@/features/prestige/components/CreateChallengeForm'
import { ChallengeCard } from '@/features/prestige/components/ChallengeCard'
import { ArcSummary } from '@/features/prestige/components/ArcSummary'
import { Tooltip } from '@/components/ui/tooltip'
import { useChallenges, useArcs, useAbandonChallenge } from '@/features/prestige/hooks'
import type { Challenge, Arc } from '@/lib/prestige'
import type { AxisKind } from '@/lib/playerProfile'
import { PlayerProfileV3 } from './profile/PlayerProfileV3'
import { useActiveCampaign } from './profile/queries'
import { usePatterns } from './queries'
import { getAscensionText } from './i18n'
import { PatternContextGrid } from './PatternContextGrid'
import { SquadVsSoloCard } from './SquadVsSoloCard'
import { BehaviorAlertList } from './BehaviorAlertList'
import { LeverList } from './LeverList'

const TITLE_SLUG = 'halo_infinite'

export function AscensionProfileTab() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const playerSlug = currentPlayer?.player_slug ?? ''
  const locale = useAppShellStore((s) => s.locale)
  const t = getAscensionText(locale)
  const coachT = getCoachStrings(locale)
  const { data: settings } = useSettings()
  const proactiveEnabled = settings?.coach_proactive_mode ?? false

  const { data: activeCampaign } = useActiveCampaign(playerSlug)
  const hasActiveCampaign = !!activeCampaign && activeCampaign.status === 'active'

  const [campaignModal, setCampaignModal] = useState<{ open: boolean; axis: string; axisKind: AxisKind }>(
    { open: false, axis: '', axisKind: 'lusr_component' },
  )
  const openStartCampaign = (axis: string, axisKind: AxisKind) =>
    setCampaignModal({ open: true, axis, axisKind })

  if (!playerSlug) {
    return (
      <p className="p-6 text-sm text-muted-foreground">
        {locale === 'en' ? 'Select a player to view objectives.' : 'Sélectionne un joueur pour voir tes objectifs.'}
      </p>
    )
  }

  return (
    <div className="space-y-10">
      {/* ─── Couche Prestige (autonome) ──────────────────────────────────── */}
      <LayerSection title={t.prestigeLayerTitle} description={t.prestigeLayerDescription}>
        <MyObjectivesSection playerSlug={playerSlug} locale={locale} />
        <MyArcsSection playerSlug={playerSlug} locale={locale} />
      </LayerSection>

      {/* ─── Couche Ascension (coaching s'appuyant sur Prestige) ──────────── */}
      <LayerSection title={t.ascensionLayerTitle} description={t.ascensionLayerDescription}>
        <CoachProposalsCard playerSlug={playerSlug} proactiveEnabled={proactiveEnabled} t={coachT} />
        {hasActiveCampaign && <CampaignTracker playerSlug={playerSlug} campaign={activeCampaign} />}
        <PlayerProfileV3
          playerSlug={playerSlug}
          onStartCampaign={hasActiveCampaign ? undefined : openStartCampaign}
        />
        <PatternsSection playerSlug={playerSlug} t={t} />
      </LayerSection>

      <StartCampaignModal
        open={campaignModal.open}
        playerSlug={playerSlug}
        axis={campaignModal.axis}
        axisKind={campaignModal.axisKind}
        onOpenChange={(open) => setCampaignModal((s) => ({ ...s, open }))}
      />
    </div>
  )
}

// ─── Layer wrapper (Prestige / Ascension) ───────────────────────────────────

interface LayerSectionProps {
  title: string
  description: string
  children: React.ReactNode
}

function LayerSection({ title, description, children }: LayerSectionProps) {
  return (
    <section className="space-y-4">
      <header className="space-y-1 border-l-2 border-primary/40 pl-3">
        <h2 className="text-lg font-bold tracking-tight">{title}</h2>
        <p className="text-sm text-muted-foreground">{description}</p>
      </header>
      <div className="space-y-6">{children}</div>
    </section>
  )
}

// ─── Mes objectifs actifs ────────────────────────────────────────────────────

interface PlayerLocaleSectionProps {
  playerSlug: string
  locale: 'fr' | 'en'
}

function MyObjectivesSection({ playerSlug, locale }: PlayerLocaleSectionProps) {
  const { data, isLoading, isError } = useChallenges(playerSlug, TITLE_SLUG)
  const abandon = useAbandonChallenge(playerSlug, TITLE_SLUG)
  const [showForm, setShowForm] = useState(false)

  if (isError) {
    return (
      <SectionShell title={locale === 'en' ? 'My objectives' : 'Mes objectifs'}>
        <p className="text-sm text-muted-foreground">
          {locale === 'en'
            ? 'The Prestige module is not enabled on this server.'
            : "Le module Prestige n'est pas activé sur ce serveur."}
        </p>
      </SectionShell>
    )
  }

  const challenges: Challenge[] = data?.challenges ?? []
  const libres = challenges.filter((c) => c.mode === 'libre')
  const pilotes = challenges.filter((c) => c.mode === 'pilote')

  const handleAbandon = (id: string) => {
    const msg =
      locale === 'en'
        ? 'Abandon this objective? 48h cooldown on the metric.'
        : 'Abandonner cet objectif ? Cooldown 48h sur la métrique.'
    if (confirm(msg)) abandon.mutate(id)
  }

  return (
    <SectionShell title={locale === 'en' ? 'My active objectives' : 'Mes objectifs actifs'}>
      <PilotModeToggle locale={locale} />
      {showForm ? (
        <div className="rounded-lg border border-border bg-card p-4">
          <CreateChallengeForm
            userId={playerSlug}
            titleSlug={TITLE_SLUG}
            onSuccess={() => setShowForm(false)}
            onCancel={() => setShowForm(false)}
          />
        </div>
      ) : (
        <>
          <ChallengeGroup
            title={`${locale === 'en' ? 'Free objectives' : 'Objectifs libres'} (${libres.length})`}
            challenges={libres}
            loading={isLoading}
            emptyMessage={locale === 'en' ? 'No free objective active.' : 'Aucun objectif libre actif.'}
            onAbandon={handleAbandon}
            onCreate={() => setShowForm(true)}
            createLabel={locale === 'en' ? '+ New objective' : '+ Nouvel objectif'}
          />
          {pilotes.length > 0 && (
            <ChallengeGroup
              title={`${locale === 'en' ? 'Piloted objectives' : 'Objectifs pilotés'} (${pilotes.length})`}
              challenges={pilotes}
              loading={false}
              emptyMessage=""
              onAbandon={handleAbandon}
            />
          )}
        </>
      )}
    </SectionShell>
  )
}

interface ChallengeGroupProps {
  title: string
  challenges: Challenge[]
  loading: boolean
  emptyMessage: string
  onAbandon: (id: string) => void
  onCreate?: () => void
  createLabel?: string
}

function ChallengeGroup({
  title,
  challenges,
  loading,
  emptyMessage,
  onAbandon,
  onCreate,
  createLabel,
}: ChallengeGroupProps) {
  return (
    <div>
      <header className="mb-2 flex items-center justify-between">
        <h3 className="text-sm font-semibold uppercase text-muted-foreground">{title}</h3>
        {onCreate && createLabel && (
          <button
            type="button"
            onClick={onCreate}
            className="rounded-md border border-border px-3 py-1 text-xs hover:bg-accent"
          >
            {createLabel}
          </button>
        )}
      </header>
      {loading ? (
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
          {[0, 1, 2].map((i) => (
            <div key={i} className="h-32 animate-pulse rounded-lg bg-muted" />
          ))}
        </div>
      ) : challenges.length === 0 ? (
        <div className="rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
          {emptyMessage || '—'}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
          {challenges.map((c) => (
            <div key={c.id} className="space-y-1">
              <ChallengeCard challenge={c} />
              {c.status === 'active' && (
                <button
                  type="button"
                  onClick={() => onAbandon(c.id)}
                  className="text-xs text-muted-foreground hover:text-destructive"
                >
                  {/* Conservé en FR strict — pas de string EN dans l'orig. */}
                  Abandonner
                </button>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function PilotModeToggle({ locale }: { locale: 'fr' | 'en' }) {
  const labelOff = locale === 'en' ? 'Disabled' : 'Désactivé'
  const help =
    locale === 'en'
      ? 'The system assigns you daily/weekly/monthly objectives with caps.'
      : "Le système t'attribue des objectifs quotidiens, hebdo et mensuels avec des plafonds."
  return (
    <div className="flex items-center justify-between rounded-md border border-border bg-card px-4 py-3">
      <div>
        <h3 className="text-sm font-semibold">{locale === 'en' ? 'Pilot mode' : 'Mode pilote'}</h3>
        <p className="flex items-center gap-1 text-xs text-muted-foreground">
          {help}
          <Tooltip content="3 daily · 5 weekly · 2 monthly">
            <span className="inline-flex h-3.5 w-3.5 cursor-default select-none items-center justify-center rounded-full border border-muted-foreground/40 text-[9px] leading-none text-muted-foreground">
              i
            </span>
          </Tooltip>
        </p>
      </div>
      <button
        type="button"
        className="rounded-md border border-border px-3 py-1 text-xs"
        title="Phase 5 minimale : non implémenté côté backend"
        disabled
      >
        {labelOff}
      </button>
    </div>
  )
}

// ─── Mes arcs ────────────────────────────────────────────────────────────────

function MyArcsSection({ playerSlug, locale }: PlayerLocaleSectionProps) {
  const { data: arcsData } = useArcs(playerSlug, TITLE_SLUG)
  const { data: challengesData } = useChallenges(playerSlug, TITLE_SLUG)
  const arcs: Arc[] = arcsData?.arcs ?? []
  const challenges: Challenge[] = challengesData?.challenges ?? []

  const stepsByArc = new Map<string, { completed: number; total: number }>()
  for (const c of challenges) {
    if (!c.arc_id) continue
    const cur = stepsByArc.get(c.arc_id) ?? { completed: 0, total: 0 }
    cur.total += 1
    if (c.status === 'completed') cur.completed += 1
    stepsByArc.set(c.arc_id, cur)
  }

  return (
    <SectionShell title={locale === 'en' ? 'My active arcs' : 'Mes arcs en cours'}>
      {arcs.length === 0 ? (
        <div className="rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
          {locale === 'en'
            ? 'No arc in progress. Choose a preset arc or create your own.'
            : 'Aucun arc en cours. Choisis un arc preset ou crée le tien.'}
        </div>
      ) : (
        <ul className="space-y-2">
          {arcs.map((a) => {
            const steps = stepsByArc.get(a.id) ?? { completed: 0, total: 0 }
            return (
              <li key={a.id}>
                <ArcSummary arc={a} completedSteps={steps.completed} totalSteps={steps.total} />
              </li>
            )
          })}
        </ul>
      )}
    </SectionShell>
  )
}

// ─── Patterns ────────────────────────────────────────────────────────────────

interface PatternsSectionProps {
  playerSlug: string
  t: ReturnType<typeof getAscensionText>
}

function PatternsSection({ playerSlug, t }: PatternsSectionProps) {
  const { data: patterns, isLoading } = usePatterns(playerSlug)
  if (isLoading) return null
  const contextPatterns = patterns?.context_patterns ?? []
  const behaviorPatterns = patterns?.behavior_patterns ?? []
  const levers = patterns?.levers ?? []

  if (contextPatterns.length === 0 && behaviorPatterns.length === 0 && levers.length === 0) {
    return null
  }

  return (
    <div className="space-y-6">
      {contextPatterns.length > 0 && (
        <SectionShell title={t.patternsSectionTitle}>
          <PatternContextGrid patterns={contextPatterns} t={t} />
          <SquadVsSoloCard patterns={contextPatterns} t={t} />
        </SectionShell>
      )}
      {behaviorPatterns.length > 0 && (
        <SectionShell title={t.behaviorsSectionTitle}>
          <BehaviorAlertList patterns={behaviorPatterns} t={t} />
        </SectionShell>
      )}
      {levers.length > 0 && (
        <SectionShell title={t.leversSectionTitle}>
          <LeverList levers={levers} t={t} />
        </SectionShell>
      )}
    </div>
  )
}

// ─── Section helper ──────────────────────────────────────────────────────────

function SectionShell({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="rounded-lg border border-border bg-card p-4">
      <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-muted-foreground">
        {title}
      </h2>
      <div className="space-y-3">{children}</div>
    </section>
  )
}
