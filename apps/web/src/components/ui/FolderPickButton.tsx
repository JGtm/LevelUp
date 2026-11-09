/**
 * FolderPickButton — le bouton « Parcourir… » d'un champ qui attend un dossier.
 *
 * Offert quand le serveur publie la capacité `can_pick_folder` (la coquille de bureau sait ouvrir la
 * boîte de choix du système), jamais selon le système ni la présence du pont : sans la capacité le
 * bouton n'existe pas et le champ reste une saisie. `value` est la valeur courante du champ, d'où la
 * boîte part ; `onPick` reçoit le chemin choisi, tel que le système l'a donné (chaîne opaque), et
 * n'est PAS appelé quand l'utilisateur annule ou que l'appel échoue : le champ reste inchangé.
 */
import { useEffect, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { pickFolder } from '@/lib/desktop/pickFolder'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import { useAppShellStore } from '@/stores/appShellStore'
import { useDeploymentCapability } from '@/stores/useDeploymentCapability'

interface FolderPickButtonProps {
  /** Valeur courante du champ : le dossier d'où la boîte part. */
  value: string
  /** Reçoit le chemin choisi. */
  onPick: (path: string) => void
  disabled?: boolean
}

export function FolderPickButton({ value, onPick, disabled = false }: FolderPickButtonProps) {
  const offered = useDeploymentCapability('can_pick_folder')
  const locale = useAppShellStore((s) => s.locale)
  const [picking, setPicking] = useState(false)
  const mounted = useRef(true)
  useEffect(() => {
    mounted.current = true
    return () => {
      mounted.current = false
    }
  }, [])

  if (!offered) return null

  async function pick() {
    setPicking(true)
    try {
      const path = await pickFolder(value)
      if (path !== null && mounted.current) onPick(path)
    } finally {
      if (mounted.current) setPicking(false)
    }
  }

  return (
    <Button
      type="button"
      size="sm"
      variant="outline"
      disabled={disabled || picking}
      onClick={() => void pick()}
    >
      {formatMessage(commonManifest, 'common.settings.browse_folder', locale)}
    </Button>
  )
}
