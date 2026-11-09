/**
 * ArcPresetPicker — catalogue d'arcs preset à adopter.
 *
 * Liste les presets du titre (via useArcPresets), affiche un aperçu des étapes
 * (nombre d'objectifs) et permet d'adopter un preset (création de l'arc + ses
 * objectifs côté backend). À l'adoption, ferme le picker ; l'arc apparaît alors
 * dans « Mes arcs en cours » (invalidation des queries).
 */
import { useArcPresets, useAdoptArcPreset } from '@/features/prestige/hooks'
import type { PresetArc } from '@/lib/prestige'
import type { Locale } from '@/lib/i18n/locale'

interface ArcPresetPickerProps {
  playerSlug: string
  titleSlug: string
  locale: Locale
  onClose: () => void
}

interface PickerText {
  title: string
  close: string
  adopt: string
  adopting: string
  loading: string
  empty: string
  error: string
  objectives: (n: number) => string
}

/** Textes du sélecteur, par langue (`Record<Locale, …>` : une langue manquante ne compile pas). */
const PICKER_TEXT: Record<Locale, PickerText> = {
  fr: {
    title: 'Arcs preset',
    close: 'Fermer',
    adopt: 'Adopter',
    adopting: 'Adoption…',
    loading: 'Chargement des presets…',
    empty: 'Aucun preset disponible pour ce titre.',
    error: "Le module Prestige n'est pas activé sur ce serveur.",
    objectives: (n) => `${n} objectif${n > 1 ? 's' : ''}`,
  },
  en: {
    title: 'Preset arcs',
    close: 'Close',
    adopt: 'Adopt',
    adopting: 'Adopting…',
    loading: 'Loading presets…',
    empty: 'No preset available for this title.',
    error: 'The Prestige module is not enabled on this server.',
    objectives: (n) => `${n} objective${n > 1 ? 's' : ''}`,
  },
}

export function ArcPresetPicker({ playerSlug, titleSlug, locale, onClose }: ArcPresetPickerProps) {
  const { data, isLoading, isError } = useArcPresets(playerSlug, titleSlug)
  const adopt = useAdoptArcPreset(playerSlug, titleSlug)
  const presets = data?.presets ?? []

  const t = PICKER_TEXT[locale]

  const handleAdopt = (presetId: string) => {
    adopt.mutate(presetId, { onSuccess: onClose })
  }

  return (
    <div className="space-y-3 rounded-lg border border-border bg-background p-4">
      <header className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">{t.title}</h3>
        <button
          type="button"
          onClick={onClose}
          className="text-xs text-muted-foreground hover:text-foreground"
        >
          {t.close}
        </button>
      </header>

      {isError ? (
        <p className="text-sm text-muted-foreground">{t.error}</p>
      ) : isLoading ? (
        <p className="text-sm text-muted-foreground">{t.loading}</p>
      ) : presets.length === 0 ? (
        <p className="text-sm text-muted-foreground">{t.empty}</p>
      ) : (
        <ul className="space-y-2">
          {presets.map((p) => (
            <PresetRow
              key={p.id}
              preset={p}
              adoptLabel={adopt.isPending ? t.adopting : t.adopt}
              objectivesLabel={t.objectives((p.steps ?? []).length)}
              disabled={adopt.isPending}
              onAdopt={() => handleAdopt(p.id)}
            />
          ))}
        </ul>
      )}
    </div>
  )
}

interface PresetRowProps {
  preset: PresetArc
  adoptLabel: string
  objectivesLabel: string
  disabled: boolean
  onAdopt: () => void
}

function PresetRow({ preset, adoptLabel, objectivesLabel, disabled, onAdopt }: PresetRowProps) {
  // Titre et description arrivent dans la langue de la requête (le serveur choisit, repli anglais).
  const title = preset.title
  const description = preset.description
  return (
    <li className="flex items-start justify-between gap-3 rounded-md border border-border p-3">
      <div className="min-w-0 flex-1">
        <h4 className="font-medium">{title}</h4>
        {description && <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>}
        <p className="mt-1 text-xs text-muted-foreground">{objectivesLabel}</p>
      </div>
      <button
        type="button"
        onClick={onAdopt}
        disabled={disabled}
        className="shrink-0 rounded-md bg-primary px-3 py-1 text-xs font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
      >
        {adoptLabel}
      </button>
    </li>
  )
}
