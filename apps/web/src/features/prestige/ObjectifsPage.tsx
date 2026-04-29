/**
 * ObjectifsPage — page principale du module Prestige.
 *
 * 2 onglets : Défis (actifs + créer) et Mon parcours (rétrospective + arcs).
 * Référence : Axe 8 du plan PLAN_challenges_xp_system.md.
 *
 * Toggle "Défis pilotés" (mode pilote) en tête de l'onglet Défis.
 * Phase 5 : version minimale fonctionnelle, à enrichir avec vrais flows UX.
 */
import { useState } from 'react'
import { useSearch, useNavigate } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import type { Challenge, Arc, UserPrestige } from '@/lib/prestige'
import { useAssetLabel } from '@/lib/i18n/fieldMappings'
import { ChallengeCard } from './components/ChallengeCard'
import { CreateChallengeForm } from './components/CreateChallengeForm'
import { ArcSummary } from './components/ArcSummary'
import { MomentCard } from './components/MomentCard'
import { StatsGlobales } from './components/StatsGlobales'
import { PRESTIGE_LEVEL_NAMES_FALLBACK } from './fallback.i18n'
import { useChallenges, useArcs, useMyPrestige, useAbandonChallenge } from './hooks'

type TabKey = 'challenges' | 'parcours'

export function ObjectifsPage() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const search = useSearch({ strict: false }) as { tab?: string }
  const navigate = useNavigate()
  const tab: TabKey = search.tab === 'parcours' ? 'parcours' : 'challenges'

  if (!currentPlayer) {
    return (
      <div className="p-6 text-sm text-muted-foreground">
        Sélectionne un joueur pour voir tes objectifs.
      </div>
    )
  }

  const userId = currentPlayer.player_slug // proxy pour user_id (titre courant)
  const titleSlug = 'halo_infinite'

  return (
    <div className="space-y-4 p-4">
      <header className="space-y-1">
        <h1 className="text-2xl font-bold">Objectifs</h1>
        <p className="text-sm text-muted-foreground">
          Défis personnels, arcs narratifs et parcours Prestige.
        </p>
      </header>

      <nav className="flex border-b border-border">
        <TabButton
          label="Défis"
          active={tab === 'challenges'}
          onClick={() => navigate({ search: {} as never })}
        />
        <TabButton
          label="Mon parcours"
          active={tab === 'parcours'}
          onClick={() => navigate({ search: { tab: 'parcours' } as never })}
        />
      </nav>

      {tab === 'challenges' ? (
        <ChallengesTab userId={userId} titleSlug={titleSlug} />
      ) : (
        <ParcoursTab userId={userId} titleSlug={titleSlug} />
      )}
    </div>
  )
}

interface TabButtonProps {
  label: string
  active: boolean
  onClick: () => void
}

function TabButton({ label, active, onClick }: TabButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={[
        'border-b-2 px-4 py-2 text-sm font-medium transition-colors',
        active
          ? 'border-primary text-foreground'
          : 'border-transparent text-muted-foreground hover:text-foreground',
      ].join(' ')}
    >
      {label}
    </button>
  )
}

// ───────────────── Onglet Défis ─────────────────

interface TabProps {
  userId: string
  titleSlug: string
}

function ChallengesTab({ userId, titleSlug }: TabProps) {
  const { data, isLoading, isError } = useChallenges(userId, titleSlug)
  const abandon = useAbandonChallenge(userId, titleSlug)
  const [showForm, setShowForm] = useState(false)

  if (isError) {
    return (
      <div className="rounded-lg border border-dashed border-border p-6 text-sm text-muted-foreground">
        Le module Prestige n'est pas encore activé sur ce serveur (PRESTIGE_ENABLED=false).
      </div>
    )
  }

  const challenges: Challenge[] = data?.challenges ?? []
  const libres = challenges.filter((c) => c.mode === 'libre')
  const pilotes = challenges.filter((c) => c.mode === 'pilote')

  if (showForm) {
    return (
      <div className="rounded-lg border border-border bg-card p-4">
        <CreateChallengeForm
          userId={userId}
          titleSlug={titleSlug}
          onSuccess={() => setShowForm(false)}
          onCancel={() => setShowForm(false)}
        />
      </div>
    )
  }

  const handleAbandon = (id: string) => {
    if (confirm('Abandonner ce défi ? Cooldown 48h sur la métrique.')) {
      abandon.mutate(id)
    }
  }

  return (
    <div className="space-y-4">
      <PilotModeToggle />

      <section>
        <header className="mb-2 flex items-center justify-between">
          <h2 className="text-sm font-semibold uppercase text-muted-foreground">
            Défis libres ({libres.length})
          </h2>
          <button
            type="button"
            onClick={() => setShowForm(true)}
            className="rounded-md border border-border px-3 py-1 text-xs hover:bg-accent"
          >
            + Nouveau défi
          </button>
        </header>
        <ChallengeGrid
          challenges={libres}
          loading={isLoading}
          emptyMessage="Aucun défi libre actif."
          onAbandon={handleAbandon}
        />
      </section>

      {pilotes.length > 0 && (
        <section>
          <h2 className="mb-2 text-sm font-semibold uppercase text-muted-foreground">
            Défis pilotés ({pilotes.length})
          </h2>
          <ChallengeGrid challenges={pilotes} loading={false} emptyMessage="" onAbandon={handleAbandon} />
        </section>
      )}
    </div>
  )
}

function PilotModeToggle() {
  // Phase 5 minimale : toggle local non persisté (Phase 6 le fera côté backend).
  return (
    <div className="flex items-center justify-between rounded-md border border-border bg-card px-4 py-3">
      <div>
        <h3 className="text-sm font-semibold">Mode pilote</h3>
        <p className="text-xs text-muted-foreground">
          Le système t'attribue des défis quotidiens, hebdo et mensuels avec des plafonds (3/5/2).
        </p>
      </div>
      <button
        type="button"
        className="rounded-md border border-border px-3 py-1 text-xs"
        title="Phase 5 minimale : non implémenté côté backend"
        disabled
      >
        Désactivé
      </button>
    </div>
  )
}

interface ChallengeGridProps {
  challenges: Challenge[]
  loading: boolean
  emptyMessage: string
  onAbandon?: (id: string) => void
}

function ChallengeGrid({ challenges, loading, emptyMessage, onAbandon }: ChallengeGridProps) {
  if (loading) {
    return (
      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
        {[0, 1, 2].map((i) => (
          <div key={i} className="h-32 animate-pulse rounded-lg bg-muted" />
        ))}
      </div>
    )
  }
  if (challenges.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
        {emptyMessage || 'Vide.'}
      </div>
    )
  }
  return (
    <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
      {challenges.map((c) => (
        <ChallengeCardWithActions key={c.id} challenge={c} onAbandon={onAbandon} />
      ))}
    </div>
  )
}

function ChallengeCardWithActions({
  challenge,
  onAbandon,
}: {
  challenge: Challenge
  onAbandon?: (id: string) => void
}) {
  return (
    <div className="space-y-1">
      <ChallengeCard challenge={challenge} />
      {onAbandon && challenge.status === 'active' && (
        <button
          type="button"
          onClick={() => onAbandon(challenge.id)}
          className="text-xs text-muted-foreground hover:text-destructive"
        >
          Abandonner
        </button>
      )}
    </div>
  )
}

// ───────────────── Onglet Mon parcours ─────────────────

function ParcoursTab({ userId, titleSlug }: TabProps) {
  const { data: prestigeData, isError: prestigeErr } = useMyPrestige(userId, titleSlug)
  const { data: arcsData } = useArcs(userId, titleSlug)
  const { data: challengesData } = useChallenges(userId, titleSlug)

  if (prestigeErr) {
    return (
      <div className="rounded-lg border border-dashed border-border p-6 text-sm text-muted-foreground">
        Le module Prestige n'est pas encore activé sur ce serveur.
      </div>
    )
  }

  const prestige: UserPrestige | undefined = prestigeData
  const arcs: Arc[] = arcsData?.arcs ?? []
  const challenges: Challenge[] = challengesData?.challenges ?? []
  const completed = challenges.filter((c) => c.status === 'completed' && c.completed_at)
  // Trier les complétés par date desc (plus récents d'abord) pour l'historique.
  const completedSorted = [...completed].sort((a, b) =>
    (b.completed_at ?? '').localeCompare(a.completed_at ?? ''),
  )

  // P6.5 : steps complétés par arc (compte des challenges status=completed dans l'arc).
  const stepsByArc = new Map<string, { completed: number; total: number }>()
  for (const c of challenges) {
    if (!c.arc_id) continue
    const cur = stepsByArc.get(c.arc_id) ?? { completed: 0, total: 0 }
    cur.total += 1
    if (c.status === 'completed') cur.completed += 1
    stepsByArc.set(c.arc_id, cur)
  }

  return (
    <div className="space-y-6">
      <PrestigeBadge prestige={prestige} />

      {/* P6.5 : composant StatsGlobales branché. */}
      <StatsGlobales challenges={challenges} />

      <section>
        <h2 className="mb-2 text-sm font-semibold uppercase text-muted-foreground">
          Arcs en cours
        </h2>
        {arcs.length === 0 ? (
          <div className="rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
            Aucun arc en cours. Choisis un arc preset ou crée le tien.
          </div>
        ) : (
          <ul className="space-y-2">
            {arcs.map((a) => {
              const steps = stepsByArc.get(a.id) ?? { completed: 0, total: 0 }
              return (
                <li key={a.id}>
                  {/* P6.5 : composant ArcSummary branché. */}
                  <ArcSummary
                    arc={a}
                    completedSteps={steps.completed}
                    totalSteps={steps.total}
                  />
                </li>
              )
            })}
          </ul>
        )}
      </section>

      <section>
        <h2 className="mb-2 text-sm font-semibold uppercase text-muted-foreground">
          Historique
        </h2>
        {completedSorted.length === 0 ? (
          <div className="rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
            Les moment cards apparaîtront ici à la validation de tes premiers défis.
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {completedSorted.map((c) => (
              // P6.5 : composant MomentCard branché.
              // achievedValue / matchCount viendront du backend dans une phase
              // ultérieure (champs non encore exposés par l'API challenges).
              <MomentCard
                key={c.id}
                challenge={c}
                achievedValue={c.target}
                matchCount={0}
                compact
              />
            ))}
          </div>
        )}
      </section>
    </div>
  )
}

interface PrestigeBadgeProps {
  prestige?: UserPrestige
}

function PrestigeBadge({ prestige }: PrestigeBadgeProps) {
  const totalPP = prestige?.total_pp ?? 0
  const level = prestige?.current_level ?? 0
  // Phase 4 plan finition multi-titres : libellé du niveau via assets.toml.
  // Fallback EN si l'endpoint /field-mappings n'est pas chargé.
  const levelKey = String(level)
  const levelLabel = useAssetLabel('prestige_level', levelKey)
  const displayLevelName =
    levelLabel !== levelKey
      ? levelLabel
      : (PRESTIGE_LEVEL_NAMES_FALLBACK[level] ?? PRESTIGE_LEVEL_NAMES_FALLBACK[0])

  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xs uppercase tracking-widest text-muted-foreground">
            Niveau Prestige
          </h2>
          <p className="text-2xl font-bold">{displayLevelName}</p>
        </div>
        <div className="text-right">
          <p className="text-xs uppercase tracking-widest text-muted-foreground">
            Points de Prestige
          </p>
          <p className="text-2xl font-bold">{totalPP.toLocaleString('fr-FR')} PP</p>
        </div>
      </div>
    </div>
  )
}
