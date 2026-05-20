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
import { toast } from 'sonner'
import { useAppShellStore } from '@/stores/appShellStore'
import { getNotificationsText } from './i18n'
import { useNotificationPreferences } from './queries'
import { useSendTestNotification, useUpdatePreferences } from './mutations'
import { ALL_CATEGORIES, type NotificationCategory, type NotificationDelivery, type NotificationPreference } from './types'

const LS_KEY_MASTER = 'levelup:notifications:master'
const LS_KEY_TOASTS = 'levelup:notifications:toasts'
const LS_KEY_RETENTION = 'levelup:notifications:retention'
const LS_KEY_SUB_OBJECTIVES = 'levelup:notifications:sub:objectives'
const LS_KEY_SUB_MEDIA = 'levelup:notifications:sub:media'
const RETENTION_DEFAULT = 200
const RETENTION_MIN = 50
const RETENTION_MAX = 500

export function NotificationsSettingsTab() {
  const locale = useAppShellStore((s) => s.locale)
  const t = getNotificationsText(locale)
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const playerSlug = currentPlayer?.player_slug ?? ''

  const [masterEnabled, setMasterEnabled] = useState(() => readBool(LS_KEY_MASTER, true))
  const [toastsEnabled, setToastsEnabled] = useState(() => readBool(LS_KEY_TOASTS, true))
  const [retention, setRetention] = useState(() => readInt(LS_KEY_RETENTION, RETENTION_DEFAULT))

  useEffect(() => writeBool(LS_KEY_MASTER, masterEnabled), [masterEnabled])
  useEffect(() => writeBool(LS_KEY_TOASTS, toastsEnabled), [toastsEnabled])
  useEffect(() => writeInt(LS_KEY_RETENTION, retention), [retention])

  const { data, isLoading } = useNotificationPreferences(playerSlug, !!playerSlug)
  const updatePrefs = useUpdatePreferences({ playerSlug })
  const sendTest = useSendTestNotification({ playerSlug })

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

  // Abonnements : raccourcis qui activent/désactivent plusieurs catégories d'un coup.
  function setSubscriptionObjectives(on: boolean) {
    writeBool(LS_KEY_SUB_OBJECTIVES, on)
    updatePrefs.mutate([
      { category: 'objective_assigned', enabled: on, delivery: on ? 'both' : 'off' },
      { category: 'objective_completed', enabled: on, delivery: on ? 'both' : 'off' },
    ])
  }
  function setSubscriptionMedia(on: boolean) {
    writeBool(LS_KEY_SUB_MEDIA, on)
    updatePrefs.mutate([
      { category: 'media_added', enabled: on, delivery: on ? 'both' : 'off' },
    ])
  }
  // Lecture initiale (l'état "réel" est le pref backend ; le LS est juste un cache UI)
  const subObjectivesOn =
    prefsByCat.get('objective_completed')?.enabled ??
    readBool(LS_KEY_SUB_OBJECTIVES, true)
  const subMediaOn =
    prefsByCat.get('media_added')?.enabled ??
    readBool(LS_KEY_SUB_MEDIA, true)

  function handleSendTest() {
    sendTest.mutate(undefined, {
      onSuccess: () => toast.success(t.settingsTestSent),
      onError: () => toast.error(t.dropdownErrorLoading),
    })
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

      {/* Abonnements per-player */}
      {playerSlug && (
        <section className="space-y-3 rounded-md border border-border bg-popover p-4">
          <header>
            <h3 className="text-sm font-semibold">{t.settingsSubscriptionsTitle}</h3>
            <p className="text-xs text-muted-foreground">
              {t.settingsSubscriptionsDescription}
              {currentPlayer?.gamertag ? ` (${currentPlayer.gamertag})` : ''}
            </p>
          </header>
          <ToggleRow
            label={t.categoryLabel.objective_completed}
            description={t.categoryDescription.objective_completed}
            checked={subObjectivesOn && masterEnabled}
            disabled={!masterEnabled}
            onChange={setSubscriptionObjectives}
          />
          <ToggleRow
            label={t.categoryLabel.media_added}
            description={t.categoryDescription.media_added}
            checked={subMediaOn && masterEnabled}
            disabled={!masterEnabled}
            onChange={setSubscriptionMedia}
          />
        </section>
      )}

      {/* Rétention (UI client-side, le serveur cap déjà à 500) */}
      <section className="space-y-2 rounded-md border border-border bg-popover p-4">
        <h3 className="text-sm font-semibold">{t.settingsRetentionTitle}</h3>
        <label className="block text-xs text-muted-foreground">
          {t.settingsRetentionLabel.replace('{n}', String(retention))}
        </label>
        <input
          type="range"
          min={RETENTION_MIN}
          max={RETENTION_MAX}
          step={50}
          value={retention}
          onChange={(e) => setRetention(Number(e.target.value))}
          className="w-full accent-primary"
          aria-label={t.settingsRetentionTitle}
        />
        <div className="flex justify-between text-2xs text-muted-foreground">
          <span>{RETENTION_MIN}</span>
          <span>{RETENTION_MAX}</span>
        </div>
      </section>

      {/* Bouton de test */}
      <section className="rounded-md border border-border bg-popover p-4">
        <button
          type="button"
          onClick={handleSendTest}
          disabled={!playerSlug || sendTest.isPending || !masterEnabled}
          className="rounded-md border border-border bg-accent px-3 py-1.5 text-sm text-accent-foreground hover:bg-accent/80 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {t.settingsTestButton}
        </button>
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

function readInt(key: string, fallback: number): number {
  if (typeof window === 'undefined') return fallback
  const raw = window.localStorage.getItem(key)
  if (raw == null) return fallback
  const n = Number.parseInt(raw, 10)
  return Number.isFinite(n) ? n : fallback
}

function writeInt(key: string, value: number): void {
  if (typeof window === 'undefined') return
  window.localStorage.setItem(key, String(value))
}

function writeBool(key: string, value: boolean): void {
  if (typeof window === 'undefined') return
  window.localStorage.setItem(key, value ? '1' : '0')
}
