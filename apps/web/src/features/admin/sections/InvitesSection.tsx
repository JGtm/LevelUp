/**
 * InvitesSection — codes d'invitation (extraction 1:1 depuis l'ancienne
 * AdminPage, restructuration en sous-routes du dashboard admin).
 */
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useAdminInvites, useGenerateInvite, useRevokeInvite, adminKeys } from '@/features/auth/queries'
import { useT, useDateLocale } from '../useAdminText'

export function InvitesSection() {
  const queryClient = useQueryClient()
  const { data: invites, isLoading } = useAdminInvites()
  const generateInvite = useGenerateInvite()
  const revokeInvite = useRevokeInvite()
  const t = useT()
  const dateLocale = useDateLocale()

  function handleGenerate() {
    generateInvite.mutate(7, {
      onSuccess: (data) => {
        queryClient.invalidateQueries({ queryKey: adminKeys.invites })
        toast.success(`${t('common.admin.invite_generated')} ${data.code}`)
      },
      onError: (err) => {
        // Sans ce feedback, un échec (403 non-admin, réseau…) donnait
        // l'impression que le bouton « ne fait rien ».
        toast.error(err instanceof Error ? err.message : t('common.admin.invite_generate_failed'))
      },
    })
  }

  function handleRevoke(code: string) {
    revokeInvite.mutate(code, {
      onSuccess: () => queryClient.invalidateQueries({ queryKey: adminKeys.invites }),
    })
  }

  return (
    <Card>
      <CardContent className="pt-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-foreground">{t('common.admin.invitation_codes_section')}</h2>
          <Button size="sm" onClick={handleGenerate} disabled={generateInvite.isPending}>
            {generateInvite.isPending ? t('common.admin.generating') : t('common.admin.generate_code')}
          </Button>
        </div>

        {isLoading ? (
          <p className="text-sm text-muted-foreground">{t('common.admin.loading')}</p>
        ) : !invites?.length ? (
          <p className="text-sm text-muted-foreground">{t('common.admin.no_invitation_codes')}</p>
        ) : (
          <div className="space-y-2">
            {invites.map((inv) => (
              <div
                key={inv.code}
                className="flex items-center justify-between rounded-md border px-4 py-3"
              >
                <div className="space-y-0.5">
                  <span className="font-mono font-bold tracking-wider text-foreground select-all">
                    {inv.code}
                  </span>
                  <div className="flex gap-3 text-xs text-muted-foreground">
                    <span>{t('common.admin.invite_by')} {inv.created_by}</span>
                    <span>{t('common.admin.invite_expires')} {new Date(inv.expires_at).toLocaleDateString(dateLocale)}</span>
                    {inv.used_by && (
                      <span className="text-primary">{t('common.admin.invite_used_by')} {inv.used_by}</span>
                    )}
                  </div>
                </div>
                {inv.valid && !inv.used_by && (
                  <Button size="sm" variant="outline" onClick={() => handleRevoke(inv.code)}>
                    {t('common.admin.revoke')}
                  </Button>
                )}
                {!inv.valid && !inv.used_by && (
                  <span className="text-xs text-muted-foreground">{t('common.admin.expired')}</span>
                )}
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
