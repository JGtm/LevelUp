/**
 * Onglet Settings > Notifications.
 *
 * - Master toggle (kill switch global, persistance localStorage)
 * - Toggle toasts (idem)
 * - Liste des catégories : pour chaque, switch enabled + delivery (both/inapp/toast/off)
 *   → persistance backend per-player via PATCH /notifications/preferences.
 *
 * Le master + toasts sont des préférences UI locales (settingsDraftStore non utilisé
 * pour rester découplé). Les catégories sont par-joueur côté serveur.
 */
import { useEffect, useState } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { getNotificationsText } from './i18n'
import { useNotificationPreferences } from './queries'
import { useUpdatePreferences } from './mutations'
import { ALL_CATEGORIES, type NotificationCategory, type NotificationDelivery, type NotificationPreference } from './types'

const LS_KEY_MASTER = 'levelup:notifications:master'
const LS_KEY_TOASTS = 'levelup:notifications:toasts'

export function NotificationsSettingsTab() {
  const locale = useAppShellStore((s) => s.locale)
  const t = getNotificationsText(locale)
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const playerSlug = currentPlayer?.player_slug ?? ''

  const [masterEnabled, setMasterEnabled] = useState(() => readBool(LS_KEY_MASTER, true))
  const [toastsEnabled, setToastsEnabled] = useState(() => readBool(LS_KEY_TOASTS, true))

  useEffect(() => writeBool(LS_KEY_MASTER, masterEnabled), [masterEnabled])
  useEffect(() => writeBool(LS_KEY_TOASTS, toastsEnabled), [toastsEnabled])

  const { data, isLoading } = useNotificationPreferences(playerSlug, !!playerSlug)
  const updatePrefs = useUpdatePreferences({ playerSlug })

  const prefsByCat = indexPrefs(data?.items ?? [])

  function patchPref(category: NotificationCategory, patch: Partial<NotificationPreference>) {
    const current = prefsByCat.get(category) ?? {
      category,
      enabled: true,
      delivery: 'both' as NotificationDelivery,
    }
    const next = { ...current, ...patch }
    updatePrefs.mutate([next])
  }

  return (
    <div className="space-y-8">
      <header>
        <h2 className="text-lg font-semibold">{t.settingsTitle}</h2>
        <p className="mt-1 text-sm text-muted-foreground">{t.settingsDescription}</p>
      </header>

      {/* Master + toasts */}
      <section className="space-y-3 rounded-md border border-border bg-popover p-4">
        <ToggleRow
          label={t.settingsMaster}
          description={t.settingsMasterDescription}
          checked={masterEnabled}
          onChange={setMasterEnabled}
        />
        <div className="border-t border-border pt-3">
          <ToggleRow
            label={t.settingsToasts}
            description={t.settingsToastsDescription}
            checked={toastsEnabled && masterEnabled}
            disabled={!masterEnabled}
            onChange={setToastsEnabled}
          />
        </div>
      </section>

      {/* Catégories */}
      <section>
        <h3 className="mb-1 text-sm font-semibold">{t.settingsCategoriesTitle}</h3>
        <p className="mb-3 text-xs text-muted-foreground">{t.settingsCategoriesDescription}</p>
        {!playerSlug ? (
          <div className="rounded-md border border-border bg-popover p-4 text-sm text-muted-foreground">
            …
          </div>
        ) : isLoading ? (
          <div className="rounded-md border border-border bg-popover p-4 text-sm text-muted-foreground">
            …
          </div>
        ) : (
          <ul className="divide-y divide-border rounded-md border border-border bg-popover">
            {ALL_CATEGORIES.map((cat) => {
              const pref = prefsByCat.get(cat)
              const enabled = pref?.enabled ?? true
              const delivery = pref?.delivery ?? 'both'
              return (
                <li key={cat} className="flex flex-wrap items-center gap-3 p-3">
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-medium">{t.categoryLabel[cat]}</p>
                    <p className="text-xs text-muted-foreground">{t.categoryDescription[cat]}</p>
                  </div>
                  <select
                    value={delivery}
                    disabled={!enabled || !masterEnabled}
                    onChange={(e) =>
                      patchPref(cat, { delivery: e.target.value as NotificationDelivery })
                    }
                    className="rounded-md border border-border bg-popover px-2 py-1 text-xs text-popover-foreground disabled:opacity-50"
                  >
                    <option value="both">{t.settingsDeliveryBoth}</option>
                    <option value="inapp">{t.settingsDeliveryInApp}</option>
                    <option value="toast">{t.settingsDeliveryToast}</option>
                  </select>
                  <label className="flex shrink-0 items-center gap-2 text-xs">
                    <input
                      type="checkbox"
                      checked={enabled}
                      disabled={!masterEnabled}
                      onChange={(e) => patchPref(cat, { enabled: e.target.checked })}
                    />
                    {enabled ? '' : t.settingsDeliveryOff}
                  </label>
                </li>
              )
            })}
          </ul>
        )}
      </section>
    </div>
  )
}

function ToggleRow(props: {
  label: string
  description: string
  checked: boolean
  disabled?: boolean
  onChange: (next: boolean) => void
}) {
  return (
    <div className="flex items-start gap-3">
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium">{props.label}</p>
        <p className="text-xs text-muted-foreground">{props.description}</p>
      </div>
      <label className="flex shrink-0 items-center gap-2">
        <input
          type="checkbox"
          checked={props.checked}
          disabled={props.disabled}
          onChange={(e) => props.onChange(e.target.checked)}
          aria-label={props.label}
        />
      </label>
    </div>
  )
}

function indexPrefs(prefs: NotificationPreference[]): Map<NotificationCategory, NotificationPreference> {
  const m = new Map<NotificationCategory, NotificationPreference>()
  for (const p of prefs) m.set(p.category, p)
  return m
}

function readBool(key: string, fallback: boolean): boolean {
  if (typeof window === 'undefined') return fallback
  const raw = window.localStorage.getItem(key)
  if (raw == null) return fallback
  return raw === '1'
}

function writeBool(key: string, value: boolean): void {
  if (typeof window === 'undefined') return
  window.localStorage.setItem(key, value ? '1' : '0')
}
