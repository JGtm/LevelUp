/**
 * AdminActionButton — bouton d'opération one-shot du dashboard admin
 * (inspiration commands board csstat) : confirmation native → exécution →
 * suivi inline du job (si asynchrone) → callback terminal pour invalidations.
 *
 * Le caller fournit `onRun` qui retourne :
 * - un jobId à suivre (action asynchrone, 202 + JobStore), ou
 * - null si l'action est synchrone et déjà terminée (toast géré par le caller).
 * Les erreurs de `onRun` sont déjà traitées par le caller (toast) — le bouton
 * se contente de sortir de l'état busy.
 */
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import type { AsyncJobStatus } from '@/lib/api/types'
import { JobProgressInline } from './JobProgressInline'

interface AdminActionButtonProps {
  label: string
  /** Confirmation native avant exécution (recommandé pour les opérations lourdes). */
  confirmMessage?: string
  busyLabel?: string
  /** Retourne le jobId à suivre, ou null si action synchrone terminée. */
  onRun: () => Promise<string | null>
  /** Déclenché une fois quand le job suivi atteint un état terminal. */
  onJobTerminal?: (job: AsyncJobStatus) => void
  disabled?: boolean
  variant?: 'default' | 'outline' | 'destructive'
}

export function AdminActionButton({
  label,
  confirmMessage,
  busyLabel,
  onRun,
  onJobTerminal,
  disabled,
  variant = 'outline',
}: AdminActionButtonProps) {
  const [busy, setBusy] = useState(false)
  const [jobId, setJobId] = useState<string | null>(null)
  const [jobDone, setJobDone] = useState(false)

  async function handleClick() {
    if (confirmMessage && !confirm(confirmMessage)) return
    setBusy(true)
    setJobDone(false)
    try {
      const id = await onRun()
      setJobId(id)
    } finally {
      setBusy(false)
    }
  }

  const followingActiveJob = !!jobId && !jobDone

  return (
    <div>
      <Button
        size="sm"
        variant={variant}
        onClick={handleClick}
        disabled={disabled || busy || followingActiveJob}
      >
        {busy || followingActiveJob ? (busyLabel ?? label) : label}
      </Button>
      {jobId && (
        <JobProgressInline
          jobId={jobId}
          onTerminal={(job) => {
            setJobDone(true)
            onJobTerminal?.(job)
          }}
        />
      )}
    </div>
  )
}
