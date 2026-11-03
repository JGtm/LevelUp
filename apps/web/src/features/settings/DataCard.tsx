/**
 * DataCard — les données de l'application de bureau (lot L22) : la sauvegarde périodique en Parquet
 * vers un dossier au choix, et le plafond mémoire des bases. Chaque section est offerte par sa
 * capacité de déploiement (`can_backup_to_folder`, `can_limit_memory`) : sur le site, la carte
 * n'existe pas.
 *
 * Chaque changement part en PATCH /settings (champ texte : à la perte du focus ou par « Enregistrer »).
 * Le refus du serveur arrive avec un code ; le texte, écrit dans chaque langue, se pose sous le champ.
 */
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useAppShellStore } from '@/stores/appShellStore'
import { useDeploymentCapability } from '@/stores/useDeploymentCapability'
import { useSettings } from './queries'
import { useDataStatus } from './dataQueries'
import { DEFAULT_BACKUP_KEEP, knownInterval } from './dataLogic'
import { getDataText } from './dataText'
import { BackupSection, MemorySection } from './DataCardSections'
import { useDataSettings, type SavedDataSettings } from './useDataSettings'

export function DataCard() {
  const locale = useAppShellStore((s) => s.locale)
  const t = getDataText(locale)
  const frozen = !useDeploymentCapability('can_edit_settings')
  const canBackup = useDeploymentCapability('can_backup_to_folder')
  const canLimitMemory = useDeploymentCapability('can_limit_memory')
  const { data: settings } = useSettings()
  const { data: status, isError } = useDataStatus(canBackup || canLimitMemory)

  const saved: SavedDataSettings = {
    backup_dir: settings?.backup_dir ?? '',
    backup_interval: knownInterval(settings?.backup_interval),
    backup_keep: String(settings?.backup_keep ?? DEFAULT_BACKUP_KEEP),
    memory_limit_mb: String(settings?.memory_limit_mb ?? 0),
  }
  const form = useDataSettings(t, saved)

  // Une section absente de la réponse = capacité absente du serveur : on ne la montre pas.
  const showBackup = canBackup && (!status || status.backup !== undefined)
  const showMemory = canLimitMemory && (!status || status.memory !== undefined)
  if (!showBackup && !showMemory) return null

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t.title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-muted-foreground">{t.intro}</p>
        <fieldset disabled={frozen} className="m-0 min-w-0 space-y-6 border-0 p-0">
          {showBackup && (
            <BackupSection
              t={t}
              locale={locale}
              form={form}
              backup={status?.backup}
              savedDir={saved.backup_dir}
            />
          )}
          {showMemory && <MemorySection t={t} locale={locale} form={form} memory={status?.memory} />}
        </fieldset>
        {form.saved && (
          <p role="status" className="text-xs text-success">
            {t.saved}
          </p>
        )}
        {isError && (
          <p role="status" className="text-xs text-muted-foreground">
            {t.loadFailed}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
