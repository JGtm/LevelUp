/**
 * PlayerFriendsSection — la liste d'amis DU JOUEUR ACTIF, en tête de la page
 * « Amis et groupes ».
 *
 * Les amis appartiennent au profil joueur, pas à l'instance : la section suit
 * donc le joueur actif du shell (`currentPlayer`), la même source que la vue
 * match et l'Escouade — la page n'est pas scopée joueur dans son URL.
 *
 * Le mode lecture seule vient de `can_edit` renvoyé par l'API ; le front
 * n'interprète jamais un 403 pour décider de son affichage.
 */
import { useState } from 'react'
import { toast } from 'sonner'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { AddFriendModal } from './AddFriendFlow'
import { usePlayerFriends, useUpdatePlayerFriends } from './queries'

type FriendsT = (key: CommonManifestKey, vars?: Record<string, unknown>) => string

export function PlayerFriendsSection() {
  const locale = useAppShellStore((s) => s.locale)
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const t: FriendsT = (key, vars) => formatMessage(commonManifest, key, locale, vars)

  const playerSlug = currentPlayer?.player_slug ?? ''
  const gamertag = currentPlayer?.gamertag ?? playerSlug
  const { data, isLoading, isError } = usePlayerFriends(playerSlug)
  const update = useUpdatePlayerFriends(playerSlug)
  const [draft, setDraft] = useState('')
  // Le gamertag en attente de CONFIRMATION (la modale partagée). Distinct de la
  // saisie : taper ne doit pas ouvrir la modale, seul le bouton le fait.
  const [confirming, setConfirming] = useState<string | null>(null)

  const friends = data?.gamertags ?? []
  const canEdit = data?.can_edit ?? false

  function handleRemove(gt: string) {
    update.mutate(
      friends.filter((f) => f.toLowerCase() !== gt.toLowerCase()),
      {
        onSuccess: () => toast.success(t('common.friends.removed', { gamertag: gt })),
        onError: () => toast.error(t('common.friends.save_error')),
      },
    )
  }

  if (!playerSlug) {
    return (
      <Card>
        <CardContent className="pt-6">
          <p className="text-sm text-muted-foreground">{t('common.friends.no_player')}</p>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">
          {t('common.friends.section_title', { gamertag })}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-muted-foreground">{t('common.friends.section_intro')}</p>

        {canEdit && (
          <div className="flex items-center gap-2">
            <input
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && draft.trim() && setConfirming(draft.trim())}
              placeholder={t('common.friends.add_placeholder')}
              maxLength={50}
              className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
            />
            <Button
              disabled={!draft.trim() || update.isPending}
              onClick={() => setConfirming(draft.trim())}
            >
              {t('common.friends.add')}
            </Button>
          </div>
        )}
        {!canEdit && <p className="text-xs text-muted-foreground">{t('common.friends.read_only')}</p>}

        {isLoading ? (
          <p className="text-sm text-muted-foreground">{t('common.friends.loading')}</p>
        ) : isError ? (
          <p className="text-sm text-muted-foreground">{t('common.friends.error')}</p>
        ) : friends.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t('common.friends.empty')}</p>
        ) : (
          <ul className="space-y-1">
            {friends.map((gt) => (
              <li key={gt} className="flex items-center justify-between gap-2 text-sm text-foreground">
                <span>{gt}</span>
                {canEdit && (
                  <Button
                    size="sm"
                    variant="outline"
                    aria-label={t('common.friends.remove_label', { gamertag: gt })}
                    disabled={update.isPending}
                    onClick={() => handleRemove(gt)}
                  >
                    {t('common.friends.remove')}
                  </Button>
                )}
              </li>
            ))}
          </ul>
        )}
      </CardContent>

      {/* L'ajout passe par le MÊME flux de confirmation que l'Escouade et la vue
          match (une seule modale d'ajout d'ami dans l'app). */}
      {confirming && canEdit && (
        <AddFriendModal
          playerSlug={playerSlug}
          gamertag={confirming}
          open
          locale={locale}
          onClose={() => setConfirming(null)}
          onSuccess={() => setDraft('')}
        />
      )}
    </Card>
  )
}
