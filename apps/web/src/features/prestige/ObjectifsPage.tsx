/**
 * ObjectifsPage — page principale du module Prestige.
 *
 * 2 onglets : Défis (actifs + créer) et Mon parcours (rétrospective + arcs).
 * Référence : Axe 8 du plan PLAN_challenges_xp_system.md.
 *
 * Toggle "Défis pilotés" (mode pilote) en tête de l'onglet Défis.
 * Phase 5 : version minimale fonctionnelle, à enrichir avec vrais flows UX.
 */
import { useSearch, useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { useAppShellStore } from '@/stores/appShellStore'
import { prestigeApi } from '@/lib/prestige'
import type { Challenge, Arc, UserPrestige } from '@/lib/prestige'
import { ChallengeCard } from './components/ChallengeCard'

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
  const { data, isLoading, isError } = useQuery({
    queryKey: ['prestige', 'challenges', userId, titleSlug],
    queryFn: () => prestigeApi.listActiveChallenges(userId, titleSlug),
    retry: false,
  })

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
            className="rounded-md border border-border px-3 py-1 text-xs hover:bg-accent"
          >
            + Nouveau défi
          </button>
        </header>
        <ChallengeGrid challenges={libres} loading={isLoading} emptyMessage="Aucun défi libre actif." />
      </section>

      {pilotes.length > 0 && (
        <section>
          <h2 className="mb-2 text-sm font-semibold uppercase text-muted-foreground">
            Défis pilotés ({pilotes.length})
          </h2>
          <ChallengeGrid challenges={pilotes} loading={false} emptyMessage="" />
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
}

function ChallengeGrid({ challenges, loading, emptyMessage }: ChallengeGridProps) {
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
        <ChallengeCard key={c.id} challenge={c} />
      ))}
    </div>
  )
}

// ───────────────── Onglet Mon parcours ─────────────────

function ParcoursTab({ userId, titleSlug }: TabProps) {
  const { data: prestigeData, isError: prestigeErr } = useQuery({
    queryKey: ['prestige', 'me', userId, titleSlug],
    queryFn: () => prestigeApi.getMyPrestige(userId, titleSlug),
    retry: false,
  })

  const { data: arcsData } = useQuery({
    queryKey: ['prestige', 'arcs', userId, titleSlug],
    queryFn: () => prestigeApi.listArcs(userId, titleSlug),
    retry: false,
  })

  if (prestigeErr) {
    return (
      <div className="rounded-lg border border-dashed border-border p-6 text-sm text-muted-foreground">
        Le module Prestige n'est pas encore activé sur ce serveur.
      </div>
    )
  }

  const prestige: UserPrestige | undefined = prestigeData
  const arcs: Arc[] = arcsData?.arcs ?? []

  return (
    <div className="space-y-6">
      <PrestigeBadge prestige={prestige} />

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
            {arcs.map((a) => (
              <li
                key={a.id}
                className="rounded-lg border border-border bg-card p-3"
              >
                <h3 className="font-medium">{a.title}</h3>
                {a.description && (
                  <p className="mt-1 text-sm text-muted-foreground">{a.description}</p>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>

      <section>
        <h2 className="mb-2 text-sm font-semibold uppercase text-muted-foreground">
          Historique
        </h2>
        <div className="rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
          Les moment cards et l'historique apparaîtront ici à la validation
          de tes premiers défis.
        </div>
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

  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xs uppercase tracking-widest text-muted-foreground">
            Niveau Prestige
          </h2>
          <p className="text-2xl font-bold">{levelName(level)}</p>
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

function levelName(idx: number): string {
  const names = ['Recrue', 'Soldat', 'Vétéran', 'Spécialiste', 'Élite', 'Légendaire']
  return names[idx] ?? names[0]
}
