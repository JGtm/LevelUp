/**
 * MediaFolderNotice — bandeau au-dessus de la grille quand le dossier des captures pose problème.
 *
 * Un seul bandeau à la fois, le dossier inaccessible d'abord :
 *   - dossier inaccessible : aucun média n'est marqué introuvable tant qu'il ne répond pas ;
 *   - retrait massif en attente : rien n'est masqué avant la confirmation, un clic l'applique.
 *
 * Le bandeau porte sa propre mutation : quand il disparaît, l'état du clic disparaît avec lui.
 */
import { Card } from '@/components/ui/card'
import { useAppShellStore } from '@/stores/appShellStore'
import { getMediaModalsText } from './i18n-modals'
import { folderNoticeKind } from './mediaMissing.logic'
import { useConfirmBulkMissing } from './missingQueries'

interface Props {
  playerSlug: string
  rootUnavailable: boolean
  bulkPending: number
}

export function MediaFolderNotice({ playerSlug, rootUnavailable, bulkPending }: Props) {
  const locale = useAppShellStore((s) => s.locale)
  const mp = getMediaModalsText(locale).missing
  const confirm = useConfirmBulkMissing(playerSlug)
  const kind = folderNoticeKind({ rootUnavailable, bulkPending })
  if (kind === null) return null

  return (
    <Card className="mx-6 overflow-hidden">
      <div
        role="status"
        className="flex items-center justify-between gap-4 border-l-4 border-warning bg-warning/10 px-4 py-3 text-sm text-warning"
      >
        <p>{kind === 'root-unavailable' ? mp.rootUnavailable : mp.bulkPending(bulkPending)}</p>
        {kind === 'bulk-pending' && (
          <button
            type="button"
            disabled={confirm.isPending || confirm.isSuccess}
            onClick={() => confirm.mutate()}
            className="shrink-0 rounded border border-warning/60 px-3 py-1.5 text-xs font-semibold hover:bg-warning/20 disabled:opacity-50"
          >
            {mp.bulkConfirm}
          </button>
        )}
      </div>
    </Card>
  )
}
