import { useEffect, useRef } from 'react'
import { toast } from 'sonner'
import type { AsyncJobStatus, JobStatus } from '@/lib/api/types'

const TERMINAL: readonly JobStatus[] = ['succeeded', 'failed', 'cancelled', 'interrupted']

export interface JobToastLabels {
  succeeded: string
  succeededWithWarnings: string
  failed: string
  cancelled: string
}

/**
 * Émet un toast Sonner lorsque jobStatus transite vers un état terminal.
 * Ne fire pas au montage si le job est déjà terminal (évite les faux positifs
 * lors d'un remontage de composant).
 */
export function useJobToasts(
  jobStatus: AsyncJobStatus | undefined,
  labels: JobToastLabels,
): void {
  const prevStatusRef = useRef<JobStatus | null>(null)
  // Refs pour lire les données fraîches sans en faire des dépendances de l'effet.
  // Affectées dans un effet (pas pendant le rendu) ; lues uniquement par l'effet
  // principal ci-dessous (déclaré après → valeurs fraîches dans le même commit).
  const jobStatusRef = useRef(jobStatus)
  const labelsRef = useRef(labels)
  useEffect(() => {
    jobStatusRef.current = jobStatus
    labelsRef.current = labels
  })

  useEffect(() => {
    const status = jobStatusRef.current?.status
    if (!status) return

    const prev = prevStatusRef.current
    prevStatusRef.current = status

    // Ne rien émettre si :
    // – pas encore en état terminal
    // – c'est la première observation (prev === null) : job déjà terminé au montage
    if (!TERMINAL.includes(status) || prev === null) return

    const l = labelsRef.current
    const js = jobStatusRef.current!
    const warnings = js.warnings ?? []

    switch (status) {
      case 'succeeded':
        if (warnings.length > 0) {
          toast.warning(l.succeededWithWarnings)
        } else {
          toast.success(l.succeeded, { description: js.current_step ?? undefined })
        }
        break
      case 'failed':
        toast.error(l.failed, {
          description: js.error?.message ?? undefined,
          duration: Infinity,
        })
        break
      case 'cancelled':
      case 'interrupted':
        toast.warning(l.cancelled)
        break
    }
    // Intentionnel : on écoute uniquement le changement de status.
    // Les autres données (warnings, error…) sont lues via refs.

  }, [jobStatus?.status])
}
