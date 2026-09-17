/**
 * InvitesSection — invitations SANS groupe, réservées à l'admin (D5).
 *
 * Deux invitations coexistent dans l'app et ne font pas la même chose : celle
 * d'un propriétaire de groupe (page « Amis et groupes ») fait REJOINDRE un
 * groupe ; celle-ci crée un compte qui ne voit que lui-même, et lui donne le
 * droit de créer SON profil joueur une fois, même sur instance verrouillée.
 *
 * Le chemin du lien (`/join?invite=`) vient du serveur (`join_url`) : le front
 * n'y préfixe que son origine.
 */
import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { useAdminT } from '../useAdminText'
import { useAdminInvites, useGenerateAdminInvite, useRevokeAdminInvite } from '@/features/auth/queries'

export function InvitesSection() {
  const tA = useAdminT()
  const { data: invites, isLoading } = useAdminInvites()
  const generate = useGenerateAdminInvite()
  const revoke = useRevokeAdminInvite()
  const [lastLink, setLastLink] = useState('')

  function handleGenerate() {
    generate.mutate(undefined, {
      onSuccess: (invite) => {
        const link = `${window.location.origin}${invite.join_url ?? ''}`
        setLastLink(link)
        void navigator.clipboard?.writeText(link)
        toast.success(tA('admin.invites.copied'))
      },
      onError: () => toast.error(tA('admin.invites.generate_error')),
    })
  }

  function handleCopy(link: string) {
    void navigator.clipboard?.writeText(link)
    toast.success(tA('admin.invites.copied'))
  }

  function handleRevoke(code: string) {
    revoke.mutate(code, {
      onError: () => toast.error(tA('admin.invites.revoke_error')),
    })
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">{tA('admin.invites.intro')}</p>

      <div className="flex items-center gap-2">
        <Button onClick={handleGenerate} disabled={generate.isPending}>
          {tA('admin.invites.generate')}
        </Button>
        {lastLink && (
          <>
            <code className="flex-1 truncate rounded-md border border-input bg-muted px-2 py-1 text-xs text-foreground">
              {lastLink}
            </code>
            <Button size="sm" variant="outline" onClick={() => handleCopy(lastLink)}>
              {tA('admin.invites.copy')}
            </Button>
          </>
        )}
      </div>

      <div>
        <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          {tA('admin.invites.list_title')}
        </p>
        {isLoading ? (
          <p className="text-sm text-muted-foreground">{tA('admin.invites.loading')}</p>
        ) : !invites?.length ? (
          <p className="text-sm text-muted-foreground">{tA('admin.invites.empty')}</p>
        ) : (
          <ul className="space-y-1">
            {invites.map((inv) => (
              <li key={inv.code} className="flex items-center justify-between gap-2 text-sm text-foreground">
                <span className="flex flex-wrap items-center gap-2">
                  <code className="rounded bg-muted px-1.5 py-0.5 text-xs">{inv.code}</code>
                  <span className="text-xs text-muted-foreground">
                    {tA('admin.invites.expires_at')} {inv.expires_at}
                  </span>
                  {inv.used_by && (
                    <span className="text-xs text-muted-foreground">
                      {tA('admin.invites.used_by')} {inv.used_by}
                    </span>
                  )}
                  {!inv.valid && !inv.used_by && (
                    <span className="text-xs text-muted-foreground">{tA('admin.invites.expired')}</span>
                  )}
                </span>
                {inv.valid && (
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={revoke.isPending}
                    onClick={() => handleRevoke(inv.code)}
                  >
                    {tA('admin.invites.revoke')}
                  </Button>
                )}
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
