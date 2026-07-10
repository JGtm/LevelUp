/**
 * AdminSyncSettingsSection — configuration de synchronisation (app_settings)
 * rapatriée depuis l'onglet « Synchronisation » des Réglages utilisateur.
 *
 * Ces réglages (auto-sync planifiée, watcher de présence, backfill, amis par
 * défaut) sont des paramètres d'instance : ils vivent désormais dans la page
 * Admin · Sync & Jobs, plus dans les Réglages utilisateur. Le composant porte
 * son propre câblage settings (useSettings/useUpdateSettings) et réutilise tel
 * quel le SyncTab existant.
 */
import { useEffect, useState } from 'react'

import { useSettings, useUpdateSettings } from '@/features/settings/queries'
import { getSettingsText, normalizeSettingsLocale } from '@/features/settings/i18n'
import { useAppShellStore } from '@/stores/appShellStore'
import { SyncTab } from '@/features/settings/SyncTab'
import type { SettingsResponse } from '@/lib/api/types'
import { SectionHeader } from '../components/SectionHeader'

export function AdminSyncSettingsSection() {
  const { data: settings, isLoading } = useSettings()
  const mutation = useUpdateSettings()
  const locale = normalizeSettingsLocale(useAppShellStore((s) => s.locale))
  const t = getSettingsText(locale)

  const [local, setLocal] = useState<Partial<SettingsResponse>>({})
  useEffect(() => {
    if (settings) setLocal(settings)
  }, [settings])

  function handleChange<K extends keyof SettingsResponse>(field: K, value: SettingsResponse[K]) {
    setLocal((prev) => ({ ...prev, [field]: value }))
    mutation.mutate({ [field]: value } as Partial<SettingsResponse>)
  }

  if (isLoading) return null

  return (
    <section className="space-y-3">
      <SectionHeader title={t.tabSync} />
      <SyncTab merged={local} handleChange={handleChange} t={t} />
    </section>
  )
}
