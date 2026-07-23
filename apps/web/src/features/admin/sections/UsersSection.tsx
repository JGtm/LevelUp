/**
 * UsersSection — gestion des comptes utilisateurs (extraction 1:1 depuis
 * l'ancienne AdminPage, restructuration en sous-routes du dashboard admin).
 */
import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import {
  useAdminUsers,
  useDeleteUser,
  useChangeRole,
  useResetPassword,
} from '@/features/auth/queries'
import { queryKeys } from '@/lib/query/keys'
import { useT } from '../useAdminText'

export function UsersSection({ currentUsername }: { currentUsername: string | null | undefined }) {
  const queryClient = useQueryClient()
  const { data: users, isLoading } = useAdminUsers()
  const deleteUser = useDeleteUser()
  const changeRole = useChangeRole()
  const resetPassword = useResetPassword()
  const t = useT()

  const [resetTarget, setResetTarget] = useState<string | null>(null)
  const [newPassword, setNewPassword] = useState('')

  function handleDelete(username: string) {
    if (!confirm(`${t('common.admin.delete_user_confirm')} "${username}" ?`)) return
    deleteUser.mutate(username, {
      onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.adminUsers }),
    })
  }

  function handleToggleRole(username: string, currentRole: string) {
    const newRole = currentRole === 'admin' ? 'user' : 'admin'
    changeRole.mutate(
      { username, role: newRole as 'admin' | 'user' },
      { onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.adminUsers }) },
    )
  }

  function handleResetPassword(username: string) {
    if (!newPassword || newPassword.length < 8) return
    resetPassword.mutate(
      { username, newPassword },
      {
        onSuccess: () => {
          setResetTarget(null)
          setNewPassword('')
        },
      },
    )
  }

  return (
    <Card>
      <CardContent className="pt-6">
        {isLoading ? (
          <p className="text-sm text-muted-foreground">{t('common.admin.loading')}</p>
        ) : !users?.length ? (
          <p className="text-sm text-muted-foreground">{t('common.admin.no_users')}</p>
        ) : (
          <div className="space-y-3">
            {users.map((u) => (
              <div key={u.username} className="flex items-center justify-between rounded-md border px-4 py-3">
                <div>
                  <span className="font-medium text-foreground">{u.username}</span>
                  <span className="ml-2 rounded bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                    {u.role}
                  </span>
                  {u.gamertag && (
                    <span className="ml-2 text-xs text-muted-foreground">({u.gamertag})</span>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  {u.username !== currentUsername && (
                    <>
                      <Button size="sm" variant="outline" onClick={() => handleToggleRole(u.username, u.role)}>
                        {u.role === 'admin'
                          ? t('common.admin.demote_to_user')
                          : t('common.admin.promote_to_admin')}
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => setResetTarget(resetTarget === u.username ? null : u.username)}
                      >
                        {t('common.admin.password_btn')}
                      </Button>
                      <Button size="sm" variant="destructive" onClick={() => handleDelete(u.username)}>
                        {t('common.admin.delete')}
                      </Button>
                    </>
                  )}
                </div>
              </div>
            ))}

            {/* Inline reset password form */}
            {resetTarget && (
              <div className="flex items-center gap-2 rounded-md border border-dashed px-4 py-3">
                <span className="text-sm text-muted-foreground">
                  {t('common.admin.new_password_for')} <strong>{resetTarget}</strong> :
                </span>
                <input
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  className="w-40 rounded-md border border-input bg-background px-2 py-1 text-sm"
                  placeholder={t('common.auth.password_placeholder_short')}
                />
                <Button size="sm" onClick={() => handleResetPassword(resetTarget)}>
                  {t('common.admin.ok')}
                </Button>
                <Button size="sm" variant="outline" onClick={() => { setResetTarget(null); setNewPassword('') }}>
                  {t('common.admin.cancel')}
                </Button>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
