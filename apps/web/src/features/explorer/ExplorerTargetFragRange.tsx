/**
 * ExplorerTargetFragRange — bloc « Portée des frags » de la section « matchs joués
 * ensemble » de l'encart adversaire (3e rangée, colonne de droite).
 *
 * PLACEHOLDER assumé : l'emplacement et le titre sont posés maintenant, la mesure
 * arrive avec le chantier « Profil d'armes » (portée par rôle : bâton P10-P90, losange
 * sur la médiane — `domain.SynthesisWeaponRange` via `port.WeaponRangeRepository`).
 * Aucune donnée n'est fabriquée ni approximée en attendant : le bloc dit ce qu'il est.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'

export function ExplorerTargetFragRange() {
  const appLocale = useAppShellStore((s) => s.locale)
  const t = (key: ExplorerManifestKey) => formatMessage(explorerManifest, key, appLocale)
  return (
    <div
      className="flex h-full flex-col overflow-hidden rounded-lg border border-border bg-card"
      data-testid="explorer-target-frag-range"
    >
      <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">
        {t('explorer.target_profile.frag_range_title')}
      </div>
      <div className="flex flex-1 items-center justify-center p-3">
        <span className="rounded-md border border-dashed border-border px-3 py-2 text-xs text-muted-foreground">
          {t('explorer.target_profile.frag_range_pending')}
        </span>
      </div>
    </div>
  )
}
