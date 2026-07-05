/**
 * AscensionProfileTab — tab "Profil & objectifs" (couche Prestige autonome).
 *
 * Depuis le split en 2 onglets (2026-06-08), ce tab ne porte QUE la couche
 * Prestige : objectifs + arcs. La couche coaching (proposals, campagne, profil,
 * patterns) vit dans AscensionCoachingTab (onglet « Entraînement »).
 */
import { useState } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { CreateChallengeForm } from '@/features/prestige/components/CreateChallengeForm'
import { ChallengeCard } from '@/features/prestige/components/ChallengeCard'
import { ArcSummary } from '@/features/prestige/components/ArcSummary'
import { CreateArcForm } from '@/features/prestige/components/CreateArcForm'
import { ArcPresetPicker } from '@/features/prestige/components/ArcPresetPicker'
import { Tooltip } from '@/components/ui/tooltip'
import { useChallenges, useArcs, useAbandonChallenge, useDeleteArc } from '@/features/prestige/hooks'
import { getPrestigeText } from '@/features/prestige/i18n'
import type { Challenge, Arc } from '@/lib/prestige'
import { getAscensionText } from './i18n'
import { LayerSection, SectionShell } from './AscensionLayers'

export function AscensionProfileTab() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const playerSlug = currentPlayer?.player_slug ?? ''
  const locale = useAppShellStore((s) => s.locale)
  const t = getAscensionText(locale)

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
    </div>
  )
}

// ─── Mes objectifs actifs ────────────────────────────────────────────────────

interface PlayerLocaleSectionProps {
  playerSlug: string
  locale: 'fr' | 'en'
}

function MyObjectivesSection({ playerSlug, locale }: PlayerLocaleSectionProps) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const { data, isLoading, isError } = useChallenges(playerSlug, titleSlug)
  const abandon = useAbandonChallenge(playerSlug, titleSlug)
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
        ? 'Abandon this objective? 24h cooldown on the metric.'
        : 'Abandonner cet objectif ? Cooldown 24h sur la métrique.'
    if (confirm(msg)) abandon.mutate(id)
  }

  return (
    <SectionShell title={locale === 'en' ? 'My active objectives' : 'Mes objectifs actifs'}>
      <PilotModeToggle locale={locale} />
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
  const t = getAscensionText(locale)
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
        title={t.prestigeDisabledHint}
        disabled
      >
        {labelOff}
      </button>
    </div>
  )
}

// ─── Mes arcs ────────────────────────────────────────────────────────────────

function MyArcsSection({ playerSlug, locale }: PlayerLocaleSectionProps) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const { data: arcsData } = useArcs(playerSlug, titleSlug)
  const { data: challengesData } = useChallenges(playerSlug, titleSlug)
  const deleteArc = useDeleteArc(playerSlug, titleSlug)
  const arcs: Arc[] = arcsData?.arcs ?? []
  const challenges: Challenge[] = challengesData?.challenges ?? []
  const [showArcForm, setShowArcForm] = useState(false)
  const [showPicker, setShowPicker] = useState(false)
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null)

  const stepsByArc = new Map<string, { completed: number; total: number }>()
  for (const c of challenges) {
    if (!c.arc_id) continue
    const cur = stepsByArc.get(c.arc_id) ?? { completed: 0, total: 0 }
    cur.total += 1
    if (c.status === 'completed') cur.completed += 1
    stepsByArc.set(c.arc_id, cur)
  }

  const newArcLabel = locale === 'en' ? '+ New arc' : '+ Nouvel arc'
  const browseLabel = locale === 'en' ? 'Browse presets' : 'Parcourir les presets'

  return (
    <SectionShell title={locale === 'en' ? 'My active arcs' : 'Mes arcs en cours'}>
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
            {locale === 'en'
              ? 'No arc in progress. Adopt a preset arc or create your own.'
              : 'Aucun arc en cours. Adopte un arc preset ou crée le tien.'}
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
  locale: 'fr' | 'en'
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

