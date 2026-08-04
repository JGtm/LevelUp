/**
 * AddFriendFlow — composant partagé Squad + Settings pour ajouter un ami.
 *
 * §3 du plan Squad/Sessions overhaul. La modale propose une confirmation
 * explicite avant de PATCH /settings (ajout à `friend_gamertags`). Le
 * recompute `is_with_friends` s'exécute automatiquement côté serveur (§4).
 *
 * Note MVP : la création de profil joueur (POST /setup/players via
 * useCreatePlayer) + sync initial (POST /sync/initial via useStartInitialSync)
 * ne sont pas enchaînés ici — l'infrastructure existe mais n'est pas câblée
 * dans ce flow. Pour l'instant, l'ami est juste ajouté à la liste : le
 * recompute server-side se base sur `xuid_aliases`, donc il résout uniquement
 * si le gamertag a déjà été croisé en match. Cas non couvert : ami jamais
 * croisé → présent dans la liste mais aucun match flaggé `is_with_friends`.
 * Follow-up trivial : enchaîner les 3 hooks dans handleConfirm.
 */
import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useSettings, useUpdateSettings } from '@/features/settings/queries'
import { queryKeys } from '@/lib/query/keys'
import { Card, CardContent } from '@/components/ui/card'

// ─── Texte i18n ──────────────────────────────────────────────────────────────

interface Texts {
  title: (gamertag: string) => string
  confirmDescription: string
  syncWarning: string
  cancel: string
  confirm: string
  successToast: (gamertag: string) => string
  errorToast: (gamertag: string, msg: string) => string
  alreadyFriend: string
}

function getTexts(locale: string): Texts {
  const isFr = locale === 'fr'
  return {
    title: (gt) => isFr ? `Ajouter ${gt} comme ami ?` : `Add ${gt} as a friend?`,
    confirmDescription: isFr
      ? 'Le coéquipier sera ajouté à la liste d\'amis et apparaîtra dans le sélecteur Escouade.'
      : 'The teammate will be added to the friends list and shown in the Squad selector.',
    syncWarning: isFr
      ? 'Les matchs historiques joués ensemble seront re-classés en escouade en arrière-plan (peut prendre plusieurs minutes selon la taille du corpus).'
      : 'Historical matches played together will be reclassified as squad in the background (may take a few minutes for large corpora).',
    cancel:  isFr ? 'Annuler' : 'Cancel',
    confirm: isFr ? 'Ajouter' : 'Add',
    successToast: (gt) => isFr
      ? `${gt} ajouté aux amis. Recalcul des sessions en cours…`
      : `${gt} added to friends. Recomputing sessions…`,
    errorToast:  (gt, msg) => isFr ? `Impossible d'ajouter ${gt} : ${msg}` : `Failed to add ${gt}: ${msg}`,
    alreadyFriend: isFr ? 'Déjà en amis' : 'Already friends',
  }
}

// ─── Hook : useAddFriend ─────────────────────────────────────────────────────

/**
 * Hook qui chaîne PATCH /settings pour ajouter `gamertag` à friend_gamertags.
 * Le recompute `is_with_friends` côté backend est déclenché automatiquement
 * sur diff (§4). Invalide les queries Squad pour rafraîchir le dropdown.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function useAddFriend(locale: string = 'fr') {
  const t = getTexts(locale)
  const { data: settings } = useSettings()
  const update = useUpdateSettings()
  const qc = useQueryClient()

  return {
    addFriend: async (gamertag: string): Promise<{ ok: boolean; reason?: 'already' | 'error'; error?: string }> => {
      const trimmed = gamertag.trim()
      if (!trimmed) return { ok: false, reason: 'error', error: 'empty gamertag' }

      const current = settings?.friend_gamertags ?? []
      if (current.some((g) => g.toLowerCase() === trimmed.toLowerCase())) {
        return { ok: false, reason: 'already' }
      }

      try {
        const next = [...current, trimmed]
        await update.mutateAsync({ friend_gamertags: next })
        toast.success(t.successToast(trimmed))
        // Invalide tout ce qui dépend de friend_gamertags : Squad, settings.
        qc.invalidateQueries({ queryKey: queryKeys.teammatesAll })
        return { ok: true }
      } catch (err) {
        const msg = err instanceof Error ? err.message : String(err)
        toast.error(t.errorToast(trimmed, msg))
        return { ok: false, reason: 'error', error: msg }
      }
    },
    isAdding: update.isPending,
  }
}

// ─── Modale de confirmation ──────────────────────────────────────────────────

export interface AddFriendModalProps {
  /** Gamertag à ajouter (affiché dans le titre). */
  gamertag: string
  /** Si false, la modale n'est pas montée. */
  open: boolean
  /** Appelé après confirmation OK ou annulation. */
  onClose: () => void
  /** Locale UI (fr | en). */
  locale: string
  /** Callback optionnel après ajout réussi. */
  onSuccess?: (gamertag: string) => void
}

export function AddFriendModal({ gamertag, open, onClose, locale, onSuccess }: AddFriendModalProps) {
  const t = getTexts(locale)
  const { addFriend, isAdding } = useAddFriend(locale)
  const [submitted, setSubmitted] = useState(false)

  if (!open) return null

  async function handleConfirm() {
    setSubmitted(true)
    const result = await addFriend(gamertag)
    setSubmitted(false)
    if (result.ok) {
      onSuccess?.(gamertag)
      onClose()
    } else if (result.reason === 'already') {
      toast.info(t.alreadyFriend)
      onClose()
    }
  }

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={t.title(gamertag)}
      className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm"
      onClick={onClose}
    >
      <Card
        className="w-full max-w-md mx-4"
        onClick={(e) => e.stopPropagation()}
      >
        <CardContent className="flex flex-col gap-4 py-6">
          <h3 className="text-lg font-semibold">{t.title(gamertag)}</h3>
          <p className="text-sm text-muted-foreground">{t.confirmDescription}</p>
          <p className="text-xs text-muted-foreground italic">{t.syncWarning}</p>
          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              disabled={submitted || isAdding}
              className="rounded-md border border-input bg-background px-3 py-1.5 text-sm hover:bg-accent disabled:opacity-50"
            >
              {t.cancel}
            </button>
            <button
              type="button"
              onClick={handleConfirm}
              disabled={submitted || isAdding}
              className="rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {t.confirm}
            </button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
