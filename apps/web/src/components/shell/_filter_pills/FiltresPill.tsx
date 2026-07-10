/**
 * FiltresPill — pill cascade pour les filtres avancés (4 catégories).
 *
 * Cascade hiérarchique : Type d'expérience → Playlists → Modes → Cartes.
 * Détection des "zombies" : valeurs sélectionnées mais absentes du dataset
 * post-cascade côté backend (incompatibles avec les autres filtres actifs).
 */
import { useMemo } from 'react'
import type { CascadeInput, LabelValue } from '@/lib/api/types'
import { useDismissable } from './_hooks'
import { CheckboxGroup } from './CheckboxGroup'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { useAppShellStore } from '@/stores/appShellStore'

export interface FiltresPillProps {
  open: boolean
  onToggle: () => void
  onClose: () => void
  available: {
    playlists: LabelValue[]
    modes: LabelValue[]
    maps: LabelValue[]
    experience_types: LabelValue[]
  }
  cascade: CascadeInput
  cascadeCount: number
  onSetCascade: (c: CascadeInput) => void
  isFetching?: boolean
}

export function FiltresPill({
  open,
  onToggle,
  onClose,
  available,
  cascade,
  cascadeCount,
  onSetCascade,
  isFetching = false,
}: FiltresPillProps) {
  const ref = useDismissable(open, onClose)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  function toggleValue(key: keyof CascadeInput, value: string) {
    const current = (cascade[key] ?? []) as string[]
    const next = current.includes(value)
      ? current.filter((v) => v !== value)
      : [...current, value]
    onSetCascade({ ...cascade, [key]: next })
  }

  // Valeurs sélectionnées absentes des options disponibles = incompatibles avec les filtres actifs.
  // Défense `?? []` : un slice Go nil sérialise en JSON null et crashe le front
  // sur .map(). Cf. testutil.RequireNoNilSlicesWithoutOmitempty.
  const availSets = useMemo(
    () => ({
      playlists: new Set((available.playlists ?? []).map((o) => o.value)),
      modes: new Set((available.modes ?? []).map((o) => o.value)),
      maps: new Set((available.maps ?? []).map((o) => o.value)),
      experience_types: new Set((available.experience_types ?? []).map((o) => o.value)),
    }),
    [available],
  )

  const zombies = useMemo(
    () => ({
      playlists: ((cascade.playlists ?? []) as string[]).filter((v) => !availSets.playlists.has(v)),
      modes: ((cascade.modes ?? []) as string[]).filter((v) => !availSets.modes.has(v)),
      maps: ((cascade.maps ?? []) as string[]).filter((v) => !availSets.maps.has(v)),
      experience_types: ((cascade.experience_types ?? []) as string[]).filter(
        (v) => !availSets.experience_types.has(v),
      ),
    }),
    [cascade, availSets],
  )

  const incompatibleCount =
    zombies.playlists.length +
    zombies.modes.length +
    zombies.maps.length +
    zombies.experience_types.length

  return (
    <div ref={ref} className="relative shrink-0">
      <button
        type="button"
        onClick={onToggle}
        aria-haspopup="dialog"
        aria-expanded={open}
        className={[
          'flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs font-medium transition-colors',
          incompatibleCount > 0
            ? 'border-destructive/50 bg-destructive/10 text-destructive hover:bg-destructive/20'
            : cascadeCount > 0
              ? 'border-primary bg-primary/10 text-primary hover:bg-primary/20'
              : 'border-input bg-background text-muted-foreground hover:bg-muted hover:text-foreground',
        ].join(' ')}
      >
        <span>{t('common.filters.pill_label')}</span>
        {cascadeCount > 0 && (
          <span
            className={[
              'rounded-full px-1.5 py-0.5 text-2xs font-medium',
              incompatibleCount > 0
                ? 'bg-destructive text-destructive-foreground'
                : 'bg-primary text-primary-foreground',
            ].join(' ')}
          >
            {cascadeCount}
          </span>
        )}
        {incompatibleCount > 0 && (
          <span
            title={formatMessage(commonManifest, 'common.filters.incompatible_tooltip', locale, { n: incompatibleCount })}
            aria-label={t('common.filters.incompatible_aria')}
          >
            ⚠
          </span>
        )}
        {isFetching && (
          <span
            className="h-2.5 w-2.5 animate-spin rounded-full border border-current border-t-transparent opacity-60"
            aria-hidden
          />
        )}
        <span className="text-2xs opacity-60">▾</span>
      </button>

      {open && (
        <div
          role="dialog"
          aria-label={t('common.filters.advanced_aria')}
          className="absolute left-0 top-full z-40 mt-1 grid w-[28rem] grid-cols-2 gap-3 rounded-md border border-border bg-background p-3 shadow-lg"
        >
          {isFetching && (
            <p className="col-span-2 text-2xs text-muted-foreground animate-pulse">
              {t('common.filters.options_updating')}
            </p>
          )}
          {!isFetching && incompatibleCount > 0 && (
            <p className="col-span-2 rounded border border-destructive/30 bg-destructive/10 px-2 py-1.5 text-3xs text-destructive">
              {formatMessage(commonManifest, 'common.filters.incompatible_full', locale, { n: incompatibleCount })}
            </p>
          )}
          <CheckboxGroup
            title={t('common.filters.experience_type_title')}
            options={available.experience_types}
            selected={(cascade.experience_types ?? []) as string[]}
            onToggle={(v) => toggleValue('experience_types', v)}
            zombies={zombies.experience_types}
            disableCollapse
          />
          <CheckboxGroup
            title={t('common.filters.playlists_title')}
            options={available.playlists}
            selected={(cascade.playlists ?? []) as string[]}
            onToggle={(v) => toggleValue('playlists', v)}
            zombies={zombies.playlists}
          />
          <CheckboxGroup
            title={t('common.filters.modes_title')}
            options={available.modes}
            selected={(cascade.modes ?? []) as string[]}
            onToggle={(v) => toggleValue('modes', v)}
            zombies={zombies.modes}
          />
          <CheckboxGroup
            title={t('common.filters.maps_title')}
            options={available.maps}
            selected={(cascade.maps ?? []) as string[]}
            onToggle={(v) => toggleValue('maps', v)}
            zombies={zombies.maps}
          />
        </div>
      )}
    </div>
  )
}
