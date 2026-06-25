/**
 * AdminBackupSection — statut + déclenchement de la sauvegarde restic des bases
 * DuckDB, rapatrié depuis l'onglet « Sauvegarde » des Réglages utilisateur.
 *
 * Le backup est une opération d'instance (toutes les bases, rétention gérée au
 * niveau serveur), pas une préférence utilisateur : sa place est la page
 * Admin · Système. Composant autonome qui câble son propre i18n — même pattern
 * que AdminSyncSettingsSection.
 */
import { getSettingsText, normalizeSettingsLocale } from '@/features/settings/i18n'
import { useAppShellStore } from '@/stores/appShellStore'
import { BackupTab } from '@/features/settings/BackupTab'

export function AdminBackupSection() {
  const locale = normalizeSettingsLocale(useAppShellStore((s) => s.locale))
  const t = getSettingsText(locale)

  return (
    <section className="space-y-3">
      <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
        {t.tabBackup}
      </h3>
      <BackupTab t={t} frozen={false} />
    </section>
  )
}
