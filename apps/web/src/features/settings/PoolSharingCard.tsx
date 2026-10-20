/**
 * PoolSharingCard — réglage « mise en commun du jeton » (onglet Synchronisation).
 *
 * Masquée quand le user n'a pas de jeton à mettre en commun (compte password sans lien
 * Xbox, ou instance sans résolveur) : `poolSharing === 'unavailable'`.
 */
import { toast } from 'sonner'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useAppShellStore } from '@/stores/appShellStore'
import { ToggleRow } from './_settingsShared'
import { getPoolSharingText } from './poolSharingText'
import { useSetPoolSharing } from './poolSharingQueries'

export function PoolSharingCard() {
  const poolSharing = useAppShellStore((s) => s.poolSharing)
  const locale = useAppShellStore((s) => s.locale)
  const mutation = useSetPoolSharing()
  const t = getPoolSharingText(locale)

  if (poolSharing === 'unavailable') return null

  const stateText = {
    granted: t.stateGranted,
    declined: t.stateDeclined,
    undecided: t.stateUndecided,
  }[poolSharing]

  function handleChange(granted: boolean) {
    mutation.mutate(granted, {
      onSuccess: () => toast.success(granted ? t.toastGranted : t.toastDeclined),
      onError: () => toast.error(t.toastFailed),
    })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t.cardTitle}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        <ToggleRow
          label={t.cardToggle}
          value={poolSharing === 'granted'}
          onChange={handleChange}
          disabled={mutation.isPending}
        />
        <p className="text-sm text-muted-foreground">{stateText}</p>
      </CardContent>
    </Card>
  )
}
