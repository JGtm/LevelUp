/**
 * AdminInitialSyncCard — carte « Synchronisation initiale (un joueur) » de la
 * page Admin · Sync & Jobs.
 *
 * Re-import complet (RunFull, plafonné) d'un joueur au choix : sélecteur joueur
 * (GamertagCombobox, sélection unique), plafond de matchs optionnel, bouton
 * d'action asynchrone suivi inline (patron AdminQuickActions). Distinct du delta
 * (convergence) et du cycle global : ré-importe tout l'historique d'un joueur
 * jusqu'au plafond, sans arrêt au premier match connu.
 *
 * MULTI-TITRE : le job est lancé sur le titre ACTIF du shell (`currentTitleSlug`),
 * envoyé explicitement en `title_slug`. Sans ce champ le serveur retomberait sur
 * le paramètre `?title` puis sur le titre par défaut — un admin positionné sur
 * Halo 5 aurait re-importé le joueur sur Halo Infinite. Le titre ciblé est
 * AFFICHÉ sur la carte : l'action est lourde et irréversible, l'admin doit voir
 * sur quel jeu elle part avant de confirmer.
 */
import { useState } from 'react'
import { toast } from 'sonner'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { GamertagCombobox } from '@/components/ui/GamertagCombobox'
import { apiErrorMessage } from '@/lib/api/client'
import { useAppShellStore } from '@/stores/appShellStore'
import { conflictJobId, useRunInitialSync } from '../monitoring/mutations'
import { useAdminT } from '../useAdminText'
import { AdminActionButton } from '../components/AdminActionButton'

interface AdminInitialSyncCardProps {
  /** Invalidations à la fin de l'action (terminal du job). */
  onJobSettled: () => void
}

export function AdminInitialSyncCard({ onJobSettled }: AdminInitialSyncCardProps) {
  const tA = useAdminT()
  const runInitialSync = useRunInitialSync()
  const [selected, setSelected] = useState<string[]>([])
  const [maxMatches, setMaxMatches] = useState('')
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  // Libellé du titre actif (repli sur le slug tant que le bootstrap n'a pas
  // hydraté availableTitles) — sélecteur qui retourne une string, jamais un objet.
  const titleName = useAppShellStore(
    (s) => s.availableTitles.find((t) => t.slug === s.currentTitleSlug)?.name ?? s.currentTitleSlug,
  )

  const playerSlug = selected[0] ?? ''

  async function handleRun(): Promise<string | null> {
    if (!playerSlug) {
      toast.error(tA('admin.initsync.no_player'))
      return null
    }
    const parsed = parseInt(maxMatches, 10)
    const max = Number.isFinite(parsed) && parsed > 0 ? parsed : undefined
    try {
      const job = await runInitialSync.mutateAsync({
        player_slug: playerSlug,
        max_matches: max,
        title_slug: titleSlug,
      })
      toast.success(tA('admin.actions.started'))
      return job.job_id
    } catch (err) {
      const existing = conflictJobId(err)
      if (existing) {
        toast.info(tA('admin.actions.conflict'))
        return existing
      }
      toast.error(apiErrorMessage(err) ?? tA('admin.actions.failed'))
      return null
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <CardTitle className="text-base">{tA('admin.initsync.title')}</CardTitle>
          {/* Titre ciblé par le job — l'action part sur le jeu actif du shell. */}
          <span className="inline-flex items-center gap-1 rounded-full bg-muted px-2.5 py-0.5 text-xs font-medium text-foreground">
            <span className="text-muted-foreground">{tA('admin.initsync.target_title')}</span>
            <span>{titleName}</span>
          </span>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-muted-foreground">{tA('admin.initsync.description')}</p>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
          <label className="flex min-w-0 flex-1 flex-col gap-1">
            <span className="text-xs font-medium text-muted-foreground">{tA('admin.initsync.player_label')}</span>
            <GamertagCombobox
              selected={selected}
              onChange={setSelected}
              max={1}
              allowFreeInput={false}
              placeholder={tA('admin.initsync.player_placeholder')}
            />
          </label>
          <label className="flex flex-col gap-1">
            <span className="text-xs font-medium text-muted-foreground">{tA('admin.initsync.max_label')}</span>
            <input
              type="number"
              min={1}
              max={2000}
              value={maxMatches}
              onChange={(e) => setMaxMatches(e.target.value)}
              placeholder={tA('admin.initsync.max_placeholder')}
              className="w-32 rounded border border-border bg-background px-2 py-1 text-sm text-foreground"
            />
          </label>
        </div>
        <AdminActionButton
          label={tA('admin.initsync.run')}
          confirmMessage={tA('admin.initsync.confirm')}
          busyLabel={tA('admin.job.in_progress')}
          onRun={handleRun}
          disabled={!playerSlug}
          onJobTerminal={(job) => {
            if (job.status === 'succeeded') toast.success(tA('admin.actions.done'))
            else toast.error(tA('admin.actions.failed'))
            onJobSettled()
          }}
        />
      </CardContent>
    </Card>
  )
}
