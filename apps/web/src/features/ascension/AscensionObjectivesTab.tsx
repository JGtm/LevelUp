/**
 * AscensionObjectivesTab — onglet "Objectifs" (couche Prestige autonome).
 *
 * Restructuration 4 onglets (2026-07, DEC-3) : ex-AscensionProfileTab renommé.
 * Ce tab ne porte QUE la couche Prestige : objectifs + arcs. L'identité/profil
 * vit dans l'onglet "Profil" (index, AscensionProfilTab) ; le coaching dans
 * l'onglet "Entraînement" (AscensionCoachingTab).
 */
import { useState, type ReactNode } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { CreateChallengeForm } from '@/features/prestige/components/CreateChallengeForm'
import { ChallengeCard } from '@/features/prestige/components/ChallengeCard'
import { ArcSummary } from '@/features/prestige/components/ArcSummary'
import { CreateArcForm } from '@/features/prestige/components/CreateArcForm'
import { ArcPresetPicker } from '@/features/prestige/components/ArcPresetPicker'
import { Tooltip } from '@/components/ui/tooltip'
import { AlertDialog } from '@/components/ui/alert-dialog'
import {
  useChallenges,
  useChallengeHistory,
  useArcs,
  useAbandonChallenge,
  useDeleteArc,
  usePilotMode,
} from '@/features/prestige/hooks'
import { getPrestigeText } from '@/features/prestige/i18n'
import type { Challenge, Arc } from '@/lib/prestige'
import { getAscensionText } from './i18n'
import { LayerSection, SectionShell } from './AscensionLayers'
import type { Locale } from '@/lib/i18n/locale'

export function AscensionObjectivesTab() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const playerSlug = currentPlayer?.player_slug ?? ''
  const locale = useAppShellStore((s) => s.locale)
  const t = getAscensionText(locale)

  if (!playerSlug) {
    return (
      <p className="p-6 text-sm text-muted-foreground">
        {t.profileSelectPlayer}
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
    </div>
  )
}

// ─── Mes objectifs actifs ────────────────────────────────────────────────────

interface PlayerLocaleSectionProps {
  playerSlug: string
  locale: Locale
}

function MyObjectivesSection({ playerSlug, locale }: PlayerLocaleSectionProps) {
  const t = getAscensionText(locale)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const { data, isLoading, isError } = useChallenges(playerSlug, titleSlug)
  const abandon = useAbandonChallenge(playerSlug, titleSlug)
  const pilot = usePilotMode(playerSlug, titleSlug)
  const [showForm, setShowForm] = useState(false)

  if (isError) {
    return (
      <SectionShell title={t.profileMyObjectives}>
        <p className="text-sm text-muted-foreground">
          {t.profilePrestigeNotEnabled}
        </p>
      </SectionShell>
    )
  }

  const challenges: Challenge[] = data?.challenges ?? []
  const libres = challenges.filter((c) => c.mode === 'libre')
  const pilotes = challenges.filter((c) => c.mode === 'pilote')
  // État ON/OFF dérivé de la présence de défis pilote actifs (pas de flag serveur).
  const pilotActive = pilotes.some((c) => c.status === 'active')
  const pilotPending = pilot.enable.isPending || pilot.disable.isPending
  const togglePilot = () => (pilotActive ? pilot.disable.mutate() : pilot.enable.mutate())

  // La confirmation d'abandon passe désormais par un AlertDialog par carte (B5) ;
  // ce handler ne fait que déclencher la mutation une fois l'utilisateur confirmé.
  const handleAbandon = (id: string) => abandon.mutate(id)

  const enablePilotCta =
    !pilotActive && !showForm ? (
      <button
        type="button"
        onClick={togglePilot}
        disabled={pilotPending}
        className="mt-3 rounded-md border border-primary px-3 py-1 text-xs font-medium text-primary hover:bg-primary/10 disabled:opacity-50"
      >
        {pilotPending ? t.profilePilotPending : t.profilePilotEnableCta}
      </button>
    ) : undefined

  return (
    <SectionShell title={t.profileMyActiveObjectives}>
      <PilotModeToggle
        locale={locale}
        active={pilotActive}
        pending={pilotPending}
        onToggle={togglePilot}
      />
      {showForm ? (
        <div className="rounded-lg border border-border bg-card p-4">
          <CreateChallengeForm
            userId={playerSlug}
            titleSlug={titleSlug}
            onSuccess={() => setShowForm(false)}
            onCancel={() => setShowForm(false)}
          />
        </div>
      ) : (
        <>
          <ChallengeGroup
            title={`${t.profileFreeObjectives} (${libres.length})`}
            challenges={libres}
            loading={isLoading}
            emptyMessage={t.profileNoFreeObjective}
            emptyCta={enablePilotCta}
            onAbandon={handleAbandon}
            onCreate={() => setShowForm(true)}
            createLabel={t.profileNewObjective}
          />
          {pilotes.length > 0 && (
            <ChallengeGroup
              title={`${t.profilePilotedObjectives} (${pilotes.length})`}
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
  /** CTA optionnel rendu sous le message d'empty state (ex. activer le pilote). */
  emptyCta?: ReactNode
  onAbandon: (id: string) => void
  onCreate?: () => void
  createLabel?: string
}

function ChallengeGroup({
  title,
  challenges,
  loading,
  emptyMessage,
  emptyCta,
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
          {emptyCta}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
          {challenges.map((c) => (
            <ObjectiveCard key={c.id} challenge={c} onAbandon={onAbandon} />
          ))}
        </div>
      )}
    </div>
  )
}

// ─── Carte d'objectif avec action d'abandon (confirmation AlertDialog, B5) ────

function ObjectiveCard({
  challenge,
  onAbandon,
}: {
  challenge: Challenge
  onAbandon: (id: string) => void
}) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getAscensionText(locale)
  const [confirmOpen, setConfirmOpen] = useState(false)

  return (
    <div className="w-64 max-w-full">
      <ChallengeCard challenge={challenge} />
      {challenge.status === 'active' && (
        <div className="flex justify-end px-1 pt-1">
          <button
            type="button"
            onClick={() => setConfirmOpen(true)}
            className="text-xs text-muted-foreground hover:text-destructive"
          >
            {t.profileAbandonObjective}
          </button>
        </div>
      )}
      <AlertDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title={t.profileAbandonTitle}
        description={t.profileAbandonConfirm}
        confirmLabel={t.profileAbandonObjective}
        cancelLabel={t.profileAbandonCancel}
        destructive
        onConfirm={() => {
          onAbandon(challenge.id)
          setConfirmOpen(false)
        }}
      />
    </div>
  )
}

interface PilotModeToggleProps {
  locale: Locale
  active: boolean
  pending: boolean
  onToggle: () => void
}

function PilotModeToggle({ locale, active, pending, onToggle }: PilotModeToggleProps) {
  const t = getAscensionText(locale)
  const label = pending
    ? t.profilePilotPending
    : active
      ? t.profilePilotDisable
      : t.profilePilotEnable
  return (
    <div className="flex items-center justify-between rounded-md border border-border bg-card px-4 py-3">
      <div>
        <h3 className="text-sm font-semibold">{t.profilePilotMode}</h3>
        <p className="flex items-center gap-1 text-xs text-muted-foreground">
          {t.profilePilotHelp}
          <Tooltip content="3 daily · 5 weekly · 2 monthly">
            <span className="inline-flex h-3.5 w-3.5 cursor-default select-none items-center justify-center rounded-full border border-muted-foreground/40 text-[9px] leading-none text-muted-foreground">
              i
            </span>
          </Tooltip>
        </p>
      </div>
      <button
        type="button"
        onClick={onToggle}
        disabled={pending}
        aria-pressed={active}
        className={[
          'rounded-md border px-3 py-1 text-xs font-medium transition-colors disabled:opacity-50',
          active
            ? 'border-primary bg-primary/10 text-primary hover:bg-primary/20'
            : 'border-border hover:bg-accent',
        ].join(' ')}
      >
        {label}
      </button>
    </div>
  )
}

// ─── Mes arcs ────────────────────────────────────────────────────────────────

/**
 * computeArcStepCounts — décompte « complétées / total » des étapes par arc.
 *
 * F1 : le décompte DOIT inclure les défis TERMINAUX de l'arc et pas seulement
 * les actifs — sinon un arc en cours affiche toujours « 0/N » (les défis
 * `completed` ne sont jamais dans la liste active). `completed` = défis
 * `status === 'completed'` ; `total` = tous les défis rattachés à l'arc (actifs
 * + terminaux), dédupliqués par id. Les défis détachés (`arc_id` absent) sont
 * ignorés.
 */
// eslint-disable-next-line react-refresh/only-export-components -- helper pur testé unitairement (motif catalogue)
export function computeArcStepCounts(
  challenges: Challenge[],
): Map<string, { completed: number; total: number }> {
  const byArc = new Map<string, { completed: number; total: number }>()
  const seen = new Set<string>()
  for (const c of challenges) {
    if (!c.arc_id) continue
    if (seen.has(c.id)) continue
    seen.add(c.id)
    const cur = byArc.get(c.arc_id) ?? { completed: 0, total: 0 }
    cur.total += 1
    if (c.status === 'completed') cur.completed += 1
    byArc.set(c.arc_id, cur)
  }
  return byArc
}

function MyArcsSection({ playerSlug, locale }: PlayerLocaleSectionProps) {
  const t = getAscensionText(locale)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const { data: arcsData } = useArcs(playerSlug, titleSlug)
  const { data: challengesData } = useChallenges(playerSlug, titleSlug)
  // Le décompte des étapes complétées vient des défis TERMINAUX de l'arc, qui ne
  // figurent PAS dans la liste active (F1) — on fusionne les deux sources.
  const { data: historyData } = useChallengeHistory(playerSlug, titleSlug)
  const deleteArc = useDeleteArc(playerSlug, titleSlug)
  // Onglet Objectifs = actif uniquement (Lot C, C4) : les arcs terminés
  // (completed_at renseigné) migrent vers l'Historique de l'onglet Réalisations.
  const arcs: Arc[] = (arcsData?.arcs ?? []).filter((a) => !a.completed_at)
  const [showArcForm, setShowArcForm] = useState(false)
  const [showPicker, setShowPicker] = useState(false)
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null)

  const stepsByArc = computeArcStepCounts([
    ...(challengesData?.challenges ?? []),
    ...(historyData?.challenges ?? []),
  ])

  const newArcLabel = t.profileNewArc
  const browseLabel = t.profileBrowsePresets

  return (
    <SectionShell title={t.profileMyActiveArcs}>
      {showArcForm ? (
        <div className="rounded-lg border border-border bg-background p-4">
          <CreateArcForm
            userId={playerSlug}
            titleSlug={titleSlug}
            onSuccess={() => setShowArcForm(false)}
            onCancel={() => setShowArcForm(false)}
          />
        </div>
      ) : showPicker ? (
        <ArcPresetPicker
          playerSlug={playerSlug}
          titleSlug={titleSlug}
          locale={locale}
          onClose={() => setShowPicker(false)}
        />
      ) : arcs.length === 0 ? (
        <div className="rounded-lg border border-dashed border-border p-6 text-center">
          <p className="text-sm text-muted-foreground">
            {t.profileNoArc}
          </p>
          <div className="mt-3 flex justify-center gap-2">
            <button
              type="button"
              onClick={() => setShowPicker(true)}
              className="rounded-md border border-border px-3 py-1 text-xs hover:bg-accent"
            >
              {browseLabel}
            </button>
            <button
              type="button"
              onClick={() => setShowArcForm(true)}
              className="rounded-md border border-border px-3 py-1 text-xs hover:bg-accent"
            >
              {newArcLabel}
            </button>
          </div>
        </div>
      ) : (
        <>
          <div className="flex justify-end gap-2">
            <button
              type="button"
              onClick={() => setShowPicker(true)}
              className="rounded-md border border-border px-3 py-1 text-xs hover:bg-accent"
            >
              {browseLabel}
            </button>
            <button
              type="button"
              onClick={() => setShowArcForm(true)}
              className="rounded-md border border-border px-3 py-1 text-xs hover:bg-accent"
            >
              {newArcLabel}
            </button>
          </div>
          <ul className="space-y-2">
            {arcs.map((a) => {
              const steps = stepsByArc.get(a.id) ?? { completed: 0, total: 0 }
              return (
                <li key={a.id} className="space-y-1">
                  <ArcSummary arc={a} completedSteps={steps.completed} totalSteps={steps.total} />
                  {confirmDeleteId === a.id ? (
                    <ArcDeleteConfirm
                      locale={locale}
                      title={a.title}
                      objectivesCount={steps.total}
                      pending={deleteArc.isPending}
                      onConfirm={(cascade) => {
                        deleteArc.mutate({ id: a.id, cascade })
                        setConfirmDeleteId(null)
                      }}
                      onCancel={() => setConfirmDeleteId(null)}
                    />
                  ) : (
                    <div className="flex justify-end">
                      <button
                        type="button"
                        onClick={() => setConfirmDeleteId(a.id)}
                        className="text-xs text-muted-foreground hover:text-destructive"
                      >
                        {getPrestigeText(locale).arcDeleteButton}
                      </button>
                    </div>
                  )}
                </li>
              )
            })}
          </ul>
        </>
      )}
    </SectionShell>
  )
}

// ─── Confirmation de suppression d'arc ──────────────────────────────────────

interface ArcDeleteConfirmProps {
  locale: Locale
  title: string
  objectivesCount: number
  pending: boolean
  onConfirm: (cascade: boolean) => void
  onCancel: () => void
}

function ArcDeleteConfirm({
  locale,
  title,
  objectivesCount,
  pending,
  onConfirm,
  onCancel,
}: ArcDeleteConfirmProps) {
  const t = getPrestigeText(locale)
  const optionClass =
    'rounded-md border border-border px-3 py-1 text-xs hover:bg-accent disabled:opacity-50'
  return (
    <div className="space-y-2 rounded-md border border-border bg-background p-3">
      <p className="text-xs text-muted-foreground">{t.arcDeleteTitle.replace('{title}', title)}</p>
      <div className="flex flex-wrap gap-2">
        {objectivesCount > 0 ? (
          <>
            <button type="button" disabled={pending} onClick={() => onConfirm(true)} className={optionClass}>
              {t.arcDeleteWithObjectives.replace('{n}', String(objectivesCount))}
            </button>
            <button type="button" disabled={pending} onClick={() => onConfirm(false)} className={optionClass}>
              {t.arcDeleteKeepObjectives}
            </button>
          </>
        ) : (
          <button type="button" disabled={pending} onClick={() => onConfirm(true)} className={optionClass}>
            {t.arcDeleteButton}
          </button>
        )}
        <button
          type="button"
          onClick={onCancel}
          className="rounded-md px-3 py-1 text-xs text-muted-foreground hover:text-foreground"
        >
          {t.arcDeleteCancel}
        </button>
      </div>
    </div>
  )
}
