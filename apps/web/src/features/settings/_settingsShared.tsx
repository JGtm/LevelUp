/**
 * Shared helpers + types for SettingsPage tabs.
 *
 * P8.4 (revue 2026-04-29) : extraits de SettingsPage.tsx pour permettre la
 * découpe en *Tab.tsx dédiés tout en partageant ToggleRow/BulletHint.
 */
import type { HintBullets } from '@/features/settings/i18n'
import type { getSettingsText } from '@/features/settings/i18n'
import type { SettingsResponse } from '@/lib/api/types'

export interface TabProps {
  merged: Partial<SettingsResponse>
  handleChange: <K extends keyof SettingsResponse>(field: K, value: SettingsResponse[K]) => void
  t: ReturnType<typeof getSettingsText>
}

interface ToggleRowProps {
  label: string
  value: boolean
  onChange: (v: boolean) => void
  disabled?: boolean
}

export function ToggleRow({ label, value, onChange, disabled }: ToggleRowProps) {
  return (
    <div className={`flex items-center justify-between py-2 ${disabled ? 'opacity-40' : ''}`}>
      <span className="text-sm text-foreground">{label}</span>
      <button
        onClick={() => !disabled && onChange(!value)}
        disabled={disabled}
        className={`relative inline-flex h-5 w-9 flex-shrink-0 rounded-full border-2 border-transparent transition-colors ${
          disabled ? 'cursor-not-allowed' : 'cursor-pointer'
        } ${value && !disabled ? 'bg-primary' : 'bg-muted'}`}
      >
        <span
          className={`pointer-events-none inline-block h-4 w-4 rounded-full bg-background shadow ring-0 transition-transform ${
            value ? 'translate-x-4' : 'translate-x-0'
          }`}
        />
      </button>
    </div>
  )
}

export function BulletHint({ hint }: { hint: HintBullets }) {
  return (
    <div className="mt-1 space-y-1 text-xs text-muted-foreground">
      <p>{hint.intro}</p>
      <ul className="ml-3 list-disc space-y-0.5">
        {hint.items.map((item, i) => (
          <li key={i}>{item}</li>
        ))}
      </ul>
      {hint.outro && <p>{hint.outro}</p>}
    </div>
  )
}
