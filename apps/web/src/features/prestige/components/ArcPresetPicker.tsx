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

export function ArcPresetPicker({ playerSlug, titleSlug, locale, onClose }: ArcPresetPickerProps) {
  const { data, isLoading, isError } = useArcPresets(playerSlug, titleSlug)
  const adopt = useAdoptArcPreset(playerSlug, titleSlug)
  const presets = data?.presets ?? []

  const t = {
    title: locale === 'en' ? 'Preset arcs' : 'Arcs preset',
    close: locale === 'en' ? 'Close' : 'Fermer',
    adopt: locale === 'en' ? 'Adopt' : 'Adopter',
    adopting: locale === 'en' ? 'Adopting…' : 'Adoption…',
    loading: locale === 'en' ? 'Loading presets…' : 'Chargement des presets…',
    empty: locale === 'en' ? 'No preset available for this title.' : 'Aucun preset disponible pour ce titre.',
    error:
      locale === 'en'
        ? 'The Prestige module is not enabled on this server.'
        : "Le module Prestige n'est pas activé sur ce serveur.",
    objectives: (n: number) =>
      locale === 'en' ? `${n} objective${n > 1 ? 's' : ''}` : `${n} objectif${n > 1 ? 's' : ''}`,
  }

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
              locale={locale}
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
  locale: Locale
  adoptLabel: string
  objectivesLabel: string
  disabled: boolean
  onAdopt: () => void
}

function PresetRow({ preset, locale, adoptLabel, objectivesLabel, disabled, onAdopt }: PresetRowProps) {
  const title = locale === 'en' ? preset.title_en || preset.title_fr : preset.title_fr || preset.title_en
  const description =
    locale === 'en'
      ? preset.description_en || preset.description_fr
      : preset.description_fr || preset.description_en
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
