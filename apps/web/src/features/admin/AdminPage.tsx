/**
 * AdminPage — gestion des utilisateurs et des invitations.
 */
import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useAppShellStore } from '@/stores/appShellStore'
import {
  useAdminUsers,
  useDeleteUser,
  useChangeRole,
  useResetPassword,
  useAdminInvites,
  useGenerateInvite,
  useRevokeInvite,
} from '@/features/auth/queries'

export function AdminPage() {
  const navigate = useNavigate()
  const isAdmin = useAppShellStore((s) => s.isAdmin)
  const currentUsername = useAppShellStore((s) => s.currentUsername)

  if (!isAdmin) {
    navigate({ to: '/' })
    return null
  }

  return (
    <div className="min-h-screen bg-background p-6">
      <div className="mx-auto max-w-4xl space-y-8">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold text-foreground">Administration</h1>
          <Button variant="outline" onClick={() => navigate({ to: '/' })}>
            Retour
          </Button>
        </div>

        <UsersSection currentUsername={currentUsername} />
        <InvitesSection />
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Section Utilisateurs
// ---------------------------------------------------------------------------

function UsersSection({ currentUsername }: { currentUsername: string | null | undefined }) {
  const queryClient = useQueryClient()
  const { data: users, isLoading } = useAdminUsers()
  const deleteUser = useDeleteUser()
  const changeRole = useChangeRole()
  const resetPassword = useResetPassword()

  const [resetTarget, setResetTarget] = useState<string | null>(null)
  const [newPassword, setNewPassword] = useState('')

  function handleDelete(username: string) {
    if (!confirm(`Supprimer l'utilisateur "${username}" ?`)) return
    deleteUser.mutate(username, {
      onSuccess: () => queryClient.invalidateQueries({ queryKey: ['admin', 'users'] }),
    })
  }

  function handleToggleRole(username: string, currentRole: string) {
    const newRole = currentRole === 'admin' ? 'user' : 'admin'
    changeRole.mutate(
      { username, role: newRole as 'admin' | 'user' },
      { onSuccess: () => queryClient.invalidateQueries({ queryKey: ['admin', 'users'] }) },
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
        <h2 className="text-lg font-semibold text-foreground mb-4">Utilisateurs</h2>
        {isLoading ? (
          <p className="text-sm text-muted-foreground">Chargement…</p>
        ) : !users?.length ? (
          <p className="text-sm text-muted-foreground">Aucun utilisateur.</p>
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
                        {u.role === 'admin' ? '→ user' : '→ admin'}
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => setResetTarget(resetTarget === u.username ? null : u.username)}
                      >
                        MDP
                      </Button>
                      <Button size="sm" variant="destructive" onClick={() => handleDelete(u.username)}>
                        Supprimer
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
                  Nouveau MDP pour <strong>{resetTarget}</strong> :
                </span>
                <input
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  className="w-40 rounded-md border border-input bg-background px-2 py-1 text-sm"
                  placeholder="8 car. min."
                />
                <Button size="sm" onClick={() => handleResetPassword(resetTarget)}>
                  OK
                </Button>
                <Button size="sm" variant="outline" onClick={() => { setResetTarget(null); setNewPassword('') }}>
                  Annuler
                </Button>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

// ---------------------------------------------------------------------------
// Section Invitations
// ---------------------------------------------------------------------------

function InvitesSection() {
  const queryClient = useQueryClient()
  const { data: invites, isLoading } = useAdminInvites()
  const generateInvite = useGenerateInvite()
  const revokeInvite = useRevokeInvite()

  function handleGenerate() {
    generateInvite.mutate(7, {
      onSuccess: () => queryClient.invalidateQueries({ queryKey: ['admin', 'invites'] }),
    })
  }

  function handleRevoke(code: string) {
    revokeInvite.mutate(code, {
      onSuccess: () => queryClient.invalidateQueries({ queryKey: ['admin', 'invites'] }),
    })
  }

  return (
    <Card>
      <CardContent className="pt-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-foreground">Codes d'invitation</h2>
          <Button size="sm" onClick={handleGenerate} disabled={generateInvite.isPending}>
            {generateInvite.isPending ? 'Génération…' : 'Générer un code'}
          </Button>
        </div>

        {isLoading ? (
          <p className="text-sm text-muted-foreground">Chargement…</p>
        ) : !invites?.length ? (
          <p className="text-sm text-muted-foreground">Aucun code d'invitation.</p>
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
                    <span>par {inv.created_by}</span>
                    <span>expire {new Date(inv.expires_at).toLocaleDateString('fr-FR')}</span>
                    {inv.used_by && <span className="text-primary">utilisé par {inv.used_by}</span>}
                  </div>
                </div>
                {inv.valid && !inv.used_by && (
                  <Button size="sm" variant="outline" onClick={() => handleRevoke(inv.code)}>
                    Révoquer
                  </Button>
                )}
                {!inv.valid && !inv.used_by && (
                  <span className="text-xs text-muted-foreground">expiré</span>
                )}
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
