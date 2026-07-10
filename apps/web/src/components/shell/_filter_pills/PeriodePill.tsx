/**
 * PeriodePill — pill de filtre date avec inputs `<input type="date">`
 * + presets rapides (7j, 30j, 90j, Toutes).
 *
 * Sélectionner une période vide automatiquement la session active
 * (logique dans FilterOmnibar via `onSetPeriod`).
 */
import type { PeriodInput, PeriodPresetCount } from '@/lib/api/types'
import { PERIOD_PRESETS, detectActivePreset, presetPeriod, useDismissable } from './_hooks'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

export interface PeriodePillProps {
  open: boolean
  onToggle: () => void
  onClose: () => void
  period: PeriodInput | undefined
  onSetPeriod: (p: PeriodInput) => void
  /** Counts par preset (plan smart-filter-counts). Optionnel : si absent,
   *  le pill fonctionne comme avant sans afficher de compteurs. */
  presetCounts?: PeriodPresetCount[]
}

export function PeriodePill({
  open,
  onToggle,
  onClose,
  period,
  onSetPeriod,
  presetCounts,
}: PeriodePillProps) {
  const ref = useDismissable(open, onClose)
  const detected = detectActivePreset(period)
  const hasPeriod = !!(period?.start_date || period?.end_date)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  let triggerLabel = t('common.period.all')
  if (hasPeriod) {
    const preset = PERIOD_PRESETS.find((p) => p.id === detected)
    triggerLabel =
      preset && preset.id !== 'all'
        ? formatMessage(commonManifest, 'common.period.prefix_named', locale, { label: t(preset.labelKey) })
        : t('common.period.custom_label')
  }

  function applyPreset(days: number) {
    onSetPeriod(presetPeriod(days) ?? { start_date: null, end_date: null })
  }

  return (
    <div ref={ref} className="relative shrink-0">
      <button
        type="button"
        onClick={onToggle}
        aria-haspopup="dialog"
        aria-expanded={open}
        className={[
          'flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs font-medium transition-colors',
          hasPeriod
            ? 'border-primary bg-primary/10 text-primary hover:bg-primary/20'
            : 'border-input bg-background text-muted-foreground hover:bg-muted hover:text-foreground',
        ].join(' ')}
      >
        <span>{triggerLabel}</span>
        <span className="text-2xs opacity-60">▾</span>
      </button>

      {open && (
        <div
          role="dialog"
          aria-label={t('common.period.dialog_aria')}
          className="absolute left-0 top-full z-40 mt-1 flex w-80 flex-col gap-3 rounded-md border border-border bg-background p-3 shadow-lg"
        >
          <div className="flex flex-wrap gap-3">
            <label className="flex items-center gap-2 text-xs text-muted-foreground">
              {t('common.period.from')}
              <input
                type="date"
                value={period?.start_date ?? ''}
                max={period?.end_date ?? undefined}
                onChange={(e) =>
                  onSetPeriod({
                    ...(period ?? {}),
                    start_date: e.target.value || null,
                  })
                }
                className="rounded border border-input bg-background px-2 py-1 text-xs text-foreground focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
              />
            </label>
            <label className="flex items-center gap-2 text-xs text-muted-foreground">
              {t('common.period.to')}
              <input
                type="date"
                value={period?.end_date ?? ''}
                min={period?.start_date ?? undefined}
                onChange={(e) =>
                  onSetPeriod({
                    ...(period ?? {}),
                    end_date: e.target.value || null,
                  })
                }
                className="rounded border border-input bg-background px-2 py-1 text-xs text-foreground focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
              />
            </label>
          </div>

          <div className="flex flex-wrap gap-2">
            {PERIOD_PRESETS.map((p) => {
              const isActive = detected === p.id
              const count = presetCounts?.find((c) => c.preset_id === p.id)?.count
              const isEmpty = count === 0
              return (
                <button
                  key={p.id}
                  type="button"
                  onClick={() => !isEmpty && applyPreset(p.days)}
                  disabled={isEmpty}
                  title={isEmpty ? t('common.period.empty_title') : undefined}
                  className={[
                    'rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors',
                    isEmpty
                      ? 'cursor-not-allowed bg-muted/40 text-muted-foreground opacity-50'
                      : isActive
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-muted text-foreground hover:bg-accent',
                  ].join(' ')}
                >
                  {t(p.labelKey)}
                  {count !== undefined && (
                    <span className="ml-1 text-2xs tabular-nums opacity-70">({count})</span>
                  )}
                </button>
              )
            })}
          </div>

          <p className="text-2xs text-muted-foreground">
            {t('common.filters.period_session_auto')}
          </p>
        </div>
      )}
    </div>
  )
}
