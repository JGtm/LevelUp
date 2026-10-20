/**
 * PoolSharingBanner — question de consentement « mise en commun du jeton ».
 *
 * Affichée en tête de l'AppShell tant que le user connecté n'a pas décidé
 * (bootstrap.pool_sharing === 'undecided'), donc à la première connexion Xbox. Les deux
 * boutons ont le même poids et aucun n'est présélectionné : un accord par défaut ne serait
 * pas un consentement. Tant que rien n'est choisi, le jeton ne sert que son propriétaire.
 * Absente en démo : l'instance publique ne détient aucun vrai jeton.
 */
import { Link } from '@tanstack/react-router'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { useAppShellStore } from '@/stores/appShellStore'
import { getPoolSharingText } from '@/features/settings/poolSharingText'
import { useSetPoolSharing } from '@/features/settings/poolSharingQueries'

export function PoolSharingBanner() {
  const poolSharing = useAppShellStore((s) => s.poolSharing)
  const demoMode = useAppShellStore((s) => s.demoMode)
  const locale = useAppShellStore((s) => s.locale)
  const mutation = useSetPoolSharing()
  const t = getPoolSharingText(locale)

  if (demoMode || poolSharing !== 'undecided') return null

  function decide(granted: boolean) {
    mutation.mutate(granted, {
      onSuccess: () => toast.success(granted ? t.toastGranted : t.toastDeclined),
      onError: () => toast.error(t.toastFailed),
    })
  }

  return (
    <div
      role="region"
      aria-label={t.bannerTitle}
      className="border-b border-border bg-muted/40 px-4 py-3 text-sm"
    >
      <p className="font-medium text-foreground">{t.bannerTitle}</p>
      <p className="mt-1 max-w-3xl text-muted-foreground">
        {t.bannerBody}{' '}
        <Link to="/privacy" className="underline underline-offset-2 hover:text-foreground">
          {t.learnMore}
        </Link>
      </p>
      <div className="mt-3 flex flex-wrap gap-2">
        <Button size="sm" variant="outline" disabled={mutation.isPending} onClick={() => decide(true)}>
          {t.accept}
        </Button>
        <Button size="sm" variant="outline" disabled={mutation.isPending} onClick={() => decide(false)}>
          {t.decline}
        </Button>
      </div>
    </div>
  )
}
