/**
 * GroupsPage — gestion end-user des groupes/familles (accès mutuel aux données).
 *
 * Liste les groupes du user, permet de créer/renommer/supprimer, voir les membres,
 * et générer une invitation "rejoindre le groupe" (lien copiable → /join?invite=).
 * Le retrait de membre / quitter sera ajouté en Phase 4.
 */
import { useState } from 'react'
import { toast } from 'sonner'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import type { Group } from '@/lib/api/types'
import {
  useMyGroups,
  useCreateGroup,
  useRenameGroup,
  useDeleteGroup,
  useCreateGroupInvite,
} from './queries'

type GroupsT = (key: CommonManifestKey) => string

export function GroupsPage() {
  const locale = useAppShellStore((s) => s.locale)
  const t: GroupsT = (key) => formatMessage(commonManifest, key, locale)
  const { data: groups, isLoading } = useMyGroups()
  const createGroup = useCreateGroup()
  const [newName, setNewName] = useState('')

  function handleCreate() {
    const name = newName.trim()
    if (!name) return
    createGroup.mutate(name, {
      onSuccess: () => setNewName(''),
      onError: (err) => toast.error(err instanceof Error ? err.message : 'Erreur'),
    })
  }

  return (
    <div className="mx-auto w-full max-w-3xl space-y-6 p-6">
      <div>
        <h1 className="text-xl font-semibold text-foreground">{t('common.groups.title')}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t('common.groups.intro')}</p>
      </div>

      <Card>
        <CardContent className="flex items-center gap-2 pt-6">
          <input
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
            placeholder={t('common.groups.new_placeholder')}
            maxLength={60}
            className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
          />
          <Button onClick={handleCreate} disabled={createGroup.isPending || !newName.trim()}>
            {t('common.groups.create')}
          </Button>
        </CardContent>
      </Card>

      {isLoading ? (
        <p className="text-sm text-muted-foreground">{t('common.groups.loading')}</p>
      ) : !groups?.length ? (
        <p className="text-sm text-muted-foreground">{t('common.groups.empty')}</p>
      ) : (
        <div className="space-y-4">
          {groups.map((g) => (
            <GroupCard key={g.id} group={g} t={t} />
          ))}
        </div>
      )}
    </div>
  )
}

function GroupCard({ group, t }: { group: Group; t: GroupsT }) {
  const renameGroup = useRenameGroup()
  const deleteGroup = useDeleteGroup()
  const createInvite = useCreateGroupInvite()
  const currentXuid = useAppShellStore((s) => s.linkedHaloIdentity?.xuid)
  const isOwner = !!currentXuid && group.owner_xuid === currentXuid
  const [editing, setEditing] = useState(false)
  const [name, setName] = useState(group.name)

  function handleRename() {
    const trimmed = name.trim()
    if (!trimmed || trimmed === group.name) {
      setEditing(false)
      return
    }
    renameGroup.mutate({ id: group.id, name: trimmed }, { onSettled: () => setEditing(false) })
  }

  function handleDelete() {
    if (!window.confirm(t('common.groups.confirm_delete'))) return
    deleteGroup.mutate(group.id)
  }

  function handleInvite() {
    createInvite.mutate(
      { id: group.id },
      {
        onSuccess: (invite) => {
          const link = `${window.location.origin}/join?invite=${invite.code}`
          void navigator.clipboard?.writeText(link)
          toast.success(`${t('common.groups.link_copied')} — ${invite.code}`)
        },
        onError: () => toast.error(t('common.groups.invite_error')),
      },
    )
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-2">
        {editing ? (
          <input
            autoFocus
            value={name}
            onChange={(e) => setName(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleRename()}
            onBlur={handleRename}
            maxLength={60}
            className="rounded-md border border-input bg-background px-2 py-1 text-base focus:outline-none focus:ring-2 focus:ring-ring"
          />
        ) : (
          <CardTitle className="text-base">{group.name}</CardTitle>
        )}
        <div className="flex items-center gap-2">
          <Button size="sm" onClick={handleInvite} disabled={createInvite.isPending}>
            {t('common.groups.invite')}
          </Button>
          {isOwner && (
            <>
              <Button size="sm" variant="outline" onClick={() => setEditing(true)}>
                {t('common.groups.rename')}
              </Button>
              <Button size="sm" variant="outline" onClick={handleDelete}>
                {t('common.groups.delete')}
              </Button>
            </>
          )}
        </div>
      </CardHeader>
      <CardContent>
        <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          {t('common.groups.members')} ({group.members.length})
        </p>
        <ul className="space-y-1">
          {group.members.map((m) => (
            <li key={m.xuid} className="flex items-center gap-2 text-sm text-foreground">
              <span>{m.gamertag || m.xuid}</span>
              {m.role === 'owner' && (
                <span className="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
                  {t('common.groups.owner')}
                </span>
              )}
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  )
}
