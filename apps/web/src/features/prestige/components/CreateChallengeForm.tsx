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
import {
  type Cadence,
  type CreateChallengeBody,
  type EvalType,
  type Template,
  type WindowType,
} from '@/lib/prestige'
import { useAssetLabel } from '@/lib/i18n/fieldMappings'
import { CADENCE_FREE_FALLBACK_FR } from '../fallback.i18n'
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
  // Phase 4 plan finition multi-titres : libellé "Libre" via cadence.free du TOML.
  const libreLabelFromTOML = useAssetLabel('cadence', 'free')
  const libreLabel =
    libreLabelFromTOML !== 'free' ? libreLabelFromTOML : CADENCE_FREE_FALLBACK_FR

  return (
    <div className="space-y-4">
      <header className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Nouveau défi</h2>
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            className="text-sm text-muted-foreground hover:text-foreground"
          >
            Annuler
          </button>
        )}
      </header>

      <div className="flex gap-1 rounded-md border border-border bg-card p-0.5">
        <ModeButton active={mode === 'hybride'} onClick={() => setMode('hybride')} label="Hybride" />
        <ModeButton active={mode === 'libre'} onClick={() => setMode('libre')} label={libreLabel} />
        <ModeButton active={mode === 'automatique'} onClick={() => setMode('automatique')} label="Automatique" />
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

// ─── Mode libre : tous les champs ───

function FreeForm({ userId, titleSlug, onSuccess }: TabFormProps) {
  const [metric, setMetric] = useState('FieldKDA')
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
      <Field label="Métrique">
        <select
          value={metric}
          onChange={(e) => setMetric(e.target.value)}
          className="w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm"
        >
          <option value="FieldKDA">KDA</option>
          <option value="FieldKDR">K/D</option>
          <option value="FieldAccuracy">Précision</option>
          <option value="FieldHeadshotKills">Headshots</option>
          <option value="FieldDamageDealt">Dégâts infligés</option>
          <option value="FieldPersonalScore">Score perso</option>
          <option value="FieldWinRate">Taux de victoire (%)</option>
        </select>
      </Field>

      <Field label="Cible">
        <input
          type="number"
          step="0.01"
          min="0"
          value={target}
          onChange={(e) => setTarget(e.target.value)}
          required
          className="w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm"
        />
      </Field>

      <div className="grid grid-cols-2 gap-3">
        <Field label="Fenêtre">
          <select
            value={windowType}
            onChange={(e) => setWindowType(e.target.value as WindowType)}
            className="w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm"
          >
            <option value="session">Sessions</option>
            <option value="rolling_days">Jours glissants</option>
            <option value="deadline">Deadline</option>
          </select>
        </Field>
        <Field label={windowType === 'deadline' ? 'Date (YYYY-MM-DD)' : 'Valeur'}>
          <input
            type="text"
            value={windowValue}
            onChange={(e) => setWindowValue(e.target.value)}
            className="w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm"
          />
        </Field>
      </div>

      <Field label="Cadence">
        <select
          value={cadence}
          onChange={(e) => setCadence(e.target.value as Cadence)}
          className="w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm"
        >
          <option value="free">Libre</option>
          <option value="daily">Quotidien</option>
          <option value="weekly">Hebdomadaire</option>
          <option value="monthly">Mensuel</option>
        </select>
      </Field>

      <Field label="Label (optionnel)">
        <input
          type="text"
          value={label}
          onChange={(e) => setLabel(e.target.value)}
          maxLength={128}
          placeholder="Ex. Slayer Lv.2"
          className="w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm"
        />
      </Field>

      <SubmitRow loading={create.isPending} error={create.error as Error | null} />
    </form>
  )
}

// ─── Mode hybride : template + ajustement ───

function HybridForm({ userId, titleSlug, onSuccess }: TabFormProps) {
  const { data, isLoading } = useSuggestedTemplates(userId, titleSlug, 5)
  const [selected, setSelected] = useState<Template | null>(null)
  const [customTarget, setCustomTarget] = useState<string>('')
  const create = useCreateChallenge(userId, titleSlug)

  if (isLoading) {
    return (
      <div className="rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
        Chargement des suggestions…
      </div>
    )
  }

  const templates = data?.templates ?? []
  if (templates.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
        Aucun template disponible. Bascule en mode libre pour créer un défi personnalisé.
      </div>
    )
  }

  const handleSelect = (t: Template) => {
    setSelected(t)
    setCustomTarget(String(t.heroic_target)) // par défaut palier Heroic
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
      <p className="text-xs text-muted-foreground">
        Choisis un template ; ajuste la cible si tu veux.
      </p>
      <ul className="space-y-2">
        {templates.map((t) => (
          <li
            key={t.id}
            className={[
              'cursor-pointer rounded-md border p-3 text-sm transition-colors',
              selected?.id === t.id
                ? 'border-primary bg-primary/5'
                : 'border-border hover:border-primary/50',
            ].join(' ')}
            onClick={() => handleSelect(t)}
          >
            <h3 className="font-medium">{t.label_fr}</h3>
            {t.description_fr && (
              <p className="mt-0.5 text-xs text-muted-foreground">{t.description_fr}</p>
            )}
            <p className="mt-1 text-xs text-muted-foreground">
              N: {t.normal_target} · H: {t.heroic_target} · L: {t.legendary_target} · M:{' '}
              {t.mythic_target}
            </p>
          </li>
        ))}
      </ul>

      {selected && (
        <Field label="Cible ajustée">
          <input
            type="number"
            step="0.01"
            value={customTarget}
            onChange={(e) => setCustomTarget(e.target.value)}
            className="w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm"
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
  const { data, isLoading } = useSuggestedTemplates(userId, titleSlug, 3)
  const create = useCreateChallenge(userId, titleSlug)

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">Génération des propositions…</p>
  }

  const templates = data?.templates ?? []

  const handleAccept = (t: Template) => {
    const body: CreateChallengeBody = {
      user_id: userId,
      title_slug: titleSlug,
      template_id: t.id,
      metric: t.metric,
      target: t.heroic_target,
      window_type: t.window_type,
      window_value: t.window_value,
      cadence: t.cadence,
      eval_type: t.eval_type,
      mode: 'libre',
      label: t.label_fr,
    }
    create.mutate(body, { onSuccess })
  }

  return (
    <div className="space-y-3">
      <p className="text-xs text-muted-foreground">
        3 défis tirés du catalogue, à accepter directement.
      </p>
      <ul className="space-y-2">
        {templates.map((t) => (
          <li key={t.id} className="rounded-md border border-border p-3">
            <h3 className="text-sm font-medium">{t.label_fr}</h3>
            {t.description_fr && (
              <p className="mt-0.5 text-xs text-muted-foreground">{t.description_fr}</p>
            )}
            <button
              type="button"
              onClick={() => handleAccept(t)}
              disabled={create.isPending}
              className="mt-2 rounded-md border border-border px-3 py-1 text-xs hover:bg-accent disabled:opacity-50"
            >
              Accepter (Heroic : {t.heroic_target})
            </button>
          </li>
        ))}
      </ul>
      {create.error != null && (
        <p className="text-xs text-destructive">{(create.error as Error).message}</p>
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
  return (
    <div className="space-y-2">
      {error && <p className="text-xs text-destructive">{error.message}</p>}
      <button
        type={onSubmit ? 'button' : 'submit'}
        onClick={onSubmit}
        disabled={loading || disabled}
        className="w-full rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
      >
        {loading ? 'Création…' : 'Créer le défi'}
      </button>
    </div>
  )
}
