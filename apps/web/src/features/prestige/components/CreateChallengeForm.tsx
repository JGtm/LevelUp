/**
 * CreateChallengeForm — formulaire de création d'un défi.
 *
 * 3 modes (Axe 7 du plan conceptuel) :
 *   - hybride (défaut) : suggestion d'un template, ajustable
 *   - libre            : tous les champs définis par le joueur
 *   - automatique      : 3 propositions du catalogue, accepte ou rejette
 *
 * Phase 5 : version minimale fonctionnelle. Le mode automatique pioche dans
 * le catalogue via `useSuggestedTemplates`. Le mode libre expose tous les
 * champs (métrique, cible, fenêtre, cadence). Mode hybride = template +
 * cible ajustable.
 */
import { useState } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import {
  type Cadence,
  type CreateChallengeBody,
  type EvalType,
  type Template,
  type WindowType,
} from '@/lib/prestige'
import { useAssetLabel } from '@/lib/i18n/fieldMappings'
import { useMetricLabel, PRESTIGE_METRIC_OPTIONS } from '@/lib/i18n/metricLabel'
import { getPrestigeText } from '../i18n'
import { useCreateChallenge, useSuggestedTemplates } from '../hooks'

type FormMode = 'hybride' | 'libre' | 'automatique'

interface CreateChallengeFormProps {
  userId: string
  titleSlug: string
  onSuccess?: () => void
  onCancel?: () => void
}

export function CreateChallengeForm({
  userId,
  titleSlug,
  onSuccess,
  onCancel,
}: CreateChallengeFormProps) {
  const [mode, setMode] = useState<FormMode>('hybride')
  const locale = useAppShellStore((s) => s.locale)
  const t = getPrestigeText(locale)
  // Libellé "Libre" servi par cadence.free des TOML (source unique — les deux
  // titres le déclarent ; plus aucun repli FR local depuis le 2026-08-02).
  const libreLabel = useAssetLabel('cadence', 'free')

  return (
    <div className="space-y-4">
      <header className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">{t.formNewChallenge}</h2>
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            className="text-sm text-muted-foreground hover:text-foreground"
          >
            {t.formCancel}
          </button>
        )}
      </header>

      <div className="flex gap-1 rounded-md border border-border bg-card p-0.5">
        <ModeButton active={mode === 'hybride'} onClick={() => setMode('hybride')} label={t.formModeHybrid} />
        <ModeButton active={mode === 'libre'} onClick={() => setMode('libre')} label={libreLabel} />
        <ModeButton active={mode === 'automatique'} onClick={() => setMode('automatique')} label={t.formModeAuto} />
      </div>

      {mode === 'libre' && (
        <FreeForm userId={userId} titleSlug={titleSlug} onSuccess={onSuccess} />
      )}
      {mode === 'hybride' && (
        <HybridForm userId={userId} titleSlug={titleSlug} onSuccess={onSuccess} />
      )}
      {mode === 'automatique' && (
        <AutoForm userId={userId} titleSlug={titleSlug} onSuccess={onSuccess} />
      )}
    </div>
  )
}

function ModeButton({ active, onClick, label }: { active: boolean; onClick: () => void; label: string }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={[
        'flex-1 rounded px-3 py-1.5 text-xs transition-colors',
        active ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground',
      ].join(' ')}
    >
      {label}
    </button>
  )
}

// Une option = un composant : le libellé vient d'un hook (useMetricLabel), qui
// ne peut pas être appelé dans le .map() du <select>. L'élément rendu reste un
// <option>, donc un enfant direct valide du <select>.
function MetricOption({ value }: { value: string }) {
  const label = useMetricLabel(value)
  return <option value={value}>{label}</option>
}

// ─── Mode libre : tous les champs ───

function FreeForm({ userId, titleSlug, onSuccess }: TabFormProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getPrestigeText(locale)
  const [metric, setMetric] = useState(PRESTIGE_METRIC_OPTIONS[0])
  const [target, setTarget] = useState('1.5')
  const [windowType, setWindowType] = useState<WindowType>('session')
  const [windowValue, setWindowValue] = useState('3')
  const [cadence, setCadence] = useState<Cadence>('free')
  const [evalType] = useState<EvalType>('threshold')
  const [label, setLabel] = useState('')

  const create = useCreateChallenge(userId, titleSlug)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const body: CreateChallengeBody = {
      user_id: userId,
      title_slug: titleSlug,
      metric,
      target: parseFloat(target),
      window_type: windowType,
      window_value: windowValue,
      cadence,
      eval_type: evalType,
      mode: 'libre',
      label: label || undefined,
    }
    create.mutate(body, { onSuccess })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <Field label={t.formFieldMetric}>
        <select
          value={metric}
          onChange={(e) => setMetric(e.target.value)}
          className="w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground"
        >
          {PRESTIGE_METRIC_OPTIONS.map((opt) => (
            <MetricOption key={opt} value={opt} />
          ))}
        </select>
      </Field>

      <Field label={t.formFieldTarget}>
        <input
          type="number"
          step="0.01"
          min="0"
          value={target}
          onChange={(e) => setTarget(e.target.value)}
          required
          className="w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground"
        />
      </Field>

      <div className="grid grid-cols-2 gap-3">
        <Field label={t.formFieldWindow}>
          <select
            value={windowType}
            onChange={(e) => setWindowType(e.target.value as WindowType)}
            className="w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground"
          >
            <option value="session">{t.windowSessions}</option>
            <option value="rolling_days">{t.windowRollingDays}</option>
            <option value="deadline">{t.windowDeadline}</option>
          </select>
        </Field>
        <Field label={windowType === 'deadline' ? t.formFieldDate : t.formFieldValue}>
          <input
            type="text"
            value={windowValue}
            onChange={(e) => setWindowValue(e.target.value)}
            className="w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground"
          />
        </Field>
      </div>

      <Field label={t.formFieldCadence}>
        <select
          value={cadence}
          onChange={(e) => setCadence(e.target.value as Cadence)}
          className="w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground"
        >
          <option value="free">{t.cadenceFree}</option>
          <option value="daily">{t.cadenceDaily}</option>
          <option value="weekly">{t.cadenceWeekly}</option>
          <option value="monthly">{t.cadenceMonthly}</option>
        </select>
      </Field>

      <Field label={t.formFieldLabel}>
        <input
          type="text"
          value={label}
          onChange={(e) => setLabel(e.target.value)}
          maxLength={128}
          placeholder={t.formLabelPlaceholder}
          className="w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground"
        />
      </Field>

      <SubmitRow loading={create.isPending} error={create.error as Error | null} />
    </form>
  )
}

// ─── Mode hybride : template + ajustement ───

function HybridForm({ userId, titleSlug, onSuccess }: TabFormProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getPrestigeText(locale)
  const { data, isLoading } = useSuggestedTemplates(userId, titleSlug, 5)
  const [selected, setSelected] = useState<Template | null>(null)
  const [customTarget, setCustomTarget] = useState<string>('')
  const create = useCreateChallenge(userId, titleSlug)

  if (isLoading) {
    return (
      <div className="rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
        {t.formLoadingSuggestions}
      </div>
    )
  }

  const templates = data?.templates ?? []
  if (templates.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
        {t.formNoTemplates}
      </div>
    )
  }

  const handleSelect = (tpl: Template) => {
    setSelected(tpl)
    setCustomTarget(String(tpl.heroic_target)) // par défaut palier Heroic
  }

  const handleSubmit = () => {
    if (!selected) return
    const body: CreateChallengeBody = {
      user_id: userId,
      title_slug: titleSlug,
      template_id: selected.id,
      metric: selected.metric,
      target: parseFloat(customTarget) || selected.heroic_target,
      window_type: selected.window_type,
      window_value: selected.window_value,
      cadence: selected.cadence,
      eval_type: selected.eval_type,
      mode: 'libre',
      label: selected.label_fr,
    }
    create.mutate(body, { onSuccess })
  }

  return (
    <div className="space-y-3">
      <p className="text-xs text-muted-foreground">{t.formHybridHint}</p>
      <ul className="space-y-2">
        {templates.map((tpl) => {
          const cd = cooldownEndsAt(tpl)
          return (
            <li
              key={tpl.id}
              aria-disabled={cd ? true : undefined}
              className={[
                'rounded-md border p-3 text-sm transition-colors',
                cd
                  ? 'cursor-not-allowed border-border opacity-60'
                  : selected?.id === tpl.id
                    ? 'cursor-pointer border-primary bg-primary/5'
                    : 'cursor-pointer border-border hover:border-primary/50',
              ].join(' ')}
              onClick={() => !cd && handleSelect(tpl)}
            >
              <h3 className="font-medium">
                {tpl.label_fr}
                {cd && <CooldownBadge end={cd} t={t} />}
              </h3>
              {tpl.description_fr && (
                <p className="mt-0.5 text-xs text-muted-foreground">{tpl.description_fr}</p>
              )}
              <p className="mt-1 text-xs text-muted-foreground">
                N: {tpl.normal_target} · H: {tpl.heroic_target} · L: {tpl.legendary_target} · M:{' '}
                {tpl.mythic_target}
              </p>
            </li>
          )
        })}
      </ul>

      {selected && (
        <Field label={t.formFieldAdjustedTarget}>
          <input
            type="number"
            step="0.01"
            value={customTarget}
            onChange={(e) => setCustomTarget(e.target.value)}
            className="w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground"
          />
        </Field>
      )}

      <SubmitRow
        loading={create.isPending}
        error={create.error as Error | null}
        disabled={!selected}
        onSubmit={handleSubmit}
      />
    </div>
  )
}

// ─── Mode automatique : 3 propositions ───

function AutoForm({ userId, titleSlug, onSuccess }: TabFormProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getPrestigeText(locale)
  const { data, isLoading } = useSuggestedTemplates(userId, titleSlug, 3)
  const create = useCreateChallenge(userId, titleSlug)

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">{t.formGenerating}</p>
  }

  const templates = data?.templates ?? []

  const handleAccept = (tpl: Template) => {
    const body: CreateChallengeBody = {
      user_id: userId,
      title_slug: titleSlug,
      template_id: tpl.id,
      metric: tpl.metric,
      target: tpl.heroic_target,
      window_type: tpl.window_type,
      window_value: tpl.window_value,
      cadence: tpl.cadence,
      eval_type: tpl.eval_type,
      mode: 'libre',
      label: tpl.label_fr,
    }
    create.mutate(body, { onSuccess })
  }

  return (
    <div className="space-y-3">
      <p className="text-xs text-muted-foreground">{t.formAutoHint}</p>
      <ul className="space-y-2">
        {templates.map((tpl) => {
          const cd = cooldownEndsAt(tpl)
          return (
            <li key={tpl.id} className="rounded-md border border-border p-3">
              <h3 className="text-sm font-medium">
                {tpl.label_fr}
                {cd && <CooldownBadge end={cd} t={t} />}
              </h3>
              {tpl.description_fr && (
                <p className="mt-0.5 text-xs text-muted-foreground">{tpl.description_fr}</p>
              )}
              <button
                type="button"
                onClick={() => handleAccept(tpl)}
                disabled={create.isPending || !!cd}
                className="mt-2 rounded-md border border-border px-3 py-1 text-xs hover:bg-accent disabled:opacity-50"
              >
                {t.formAcceptHeroic.replace('{target}', String(tpl.heroic_target))}
              </button>
            </li>
          )
        })}
      </ul>
      {resolveCreateError(create.error as Error | null, t) != null && (
        <p className="text-xs text-destructive">{resolveCreateError(create.error as Error | null, t)}</p>
      )}
    </div>
  )
}

// ─── Helpers ───

interface TabFormProps {
  userId: string
  titleSlug: string
  onSuccess?: () => void
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1 block text-xs font-medium text-muted-foreground">{label}</span>
      {children}
    </label>
  )
}

// ─── Cooldown anti-farming (métrique en repos) ───

/** Retourne la date de fin de cooldown si elle est dans le futur, sinon null. */
function cooldownEndsAt(tpl: Template): Date | null {
  if (!tpl.cooldown_ends_at) return null
  const end = new Date(tpl.cooldown_ends_at)
  return end.getTime() > Date.now() ? end : null
}

/** Formate le délai restant en badge localisé (« Dispo dans 3 h » / « 2 j »). */
function formatCooldown(end: Date, t: ReturnType<typeof getPrestigeText>): string {
  const hours = Math.ceil((end.getTime() - Date.now()) / 3_600_000)
  const time =
    hours >= 24 ? `${Math.ceil(hours / 24)} ${t.cooldownUnitDay}` : `${hours} ${t.cooldownUnitHour}`
  return t.cooldownBadge.replace('{time}', time)
}

/** Mappe un refus cooldown (429) sur un message lisible, sinon le message brut. */
function resolveCreateError(error: Error | null, t: ReturnType<typeof getPrestigeText>): string | null {
  if (!error) return null
  if ((error as { code?: string }).code === 'cooldown_active') return t.cooldownErrorMessage
  return error.message
}

function CooldownBadge({ end, t }: { end: Date; t: ReturnType<typeof getPrestigeText> }) {
  return (
    <span className="ml-2 rounded border border-border px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
      {formatCooldown(end, t)}
    </span>
  )
}

function SubmitRow({
  loading,
  error,
  disabled,
  onSubmit,
}: {
  loading: boolean
  error: Error | null
  disabled?: boolean
  onSubmit?: () => void
}) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getPrestigeText(locale)
  const errorMessage = resolveCreateError(error, t)
  return (
    <div className="space-y-2">
      {errorMessage && <p className="text-xs text-destructive">{errorMessage}</p>}
      <button
        type={onSubmit ? 'button' : 'submit'}
        onClick={onSubmit}
        disabled={loading || disabled}
        className="w-full rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
      >
        {loading ? t.formCreating : t.formCreate}
      </button>
    </div>
  )
}
