import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import type { AsyncJobStatus, JobStatus } from '@/lib/api/types'

vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    warning: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  },
}))

// Import après le mock pour récupérer les spies.
import { toast } from 'sonner'
import { useJobToasts } from './useJobToasts'
import type { JobToastLabels } from './useJobToasts'

// ─── Fixtures ────────────────────────────────────────────────────────────────

function job(status: JobStatus, overrides: Partial<AsyncJobStatus> = {}): AsyncJobStatus {
  return {
    job_id: 'j1',
    job_type: 'backfill',
    status,
    progress_pct: null,
    current_step: null,
    started_at: null,
    finished_at: null,
    result: null,
    error: null,
    phase_key: null,
    phase_label: null,
    matches_done: null,
    matches_total: null,
    subtasks_done: null,
    subtasks_total: null,
    eta_seconds: null,
    warnings: [],
    ...overrides,
  }
}

const LABELS: JobToastLabels = {
  succeeded: 'Terminé',
  succeededWithWarnings: 'Terminé avec avertissements',
  failed: 'Échoué',
  cancelled: 'Annulé',
}

// ─── Tests ───────────────────────────────────────────────────────────────────

beforeEach(() => {
  vi.mocked(toast.success).mockClear()
  vi.mocked(toast.warning).mockClear()
  vi.mocked(toast.error).mockClear()
})

describe('useJobToasts', () => {
  it('ne fire pas si jobStatus est undefined au montage', () => {
    renderHook(() => useJobToasts(undefined, LABELS))
    expect(toast.success).not.toHaveBeenCalled()
    expect(toast.warning).not.toHaveBeenCalled()
    expect(toast.error).not.toHaveBeenCalled()
  })

  it('ne fire pas si le job est déjà terminal au montage (guard prev=null)', () => {
    renderHook(() => useJobToasts(job('succeeded'), LABELS))
    expect(toast.success).not.toHaveBeenCalled()
  })

  it('ne fire pas tant que le job est en cours', () => {
    const { rerender } = renderHook(
      ({ js }: { js: AsyncJobStatus | undefined }) => useJobToasts(js, LABELS),
      { initialProps: { js: undefined } },
    )
    rerender({ js: job('queued') })
    rerender({ js: job('running') })
    expect(toast.success).not.toHaveBeenCalled()
    expect(toast.warning).not.toHaveBeenCalled()
    expect(toast.error).not.toHaveBeenCalled()
  })

  it('fire toast.success sur succeeded sans warnings', () => {
    const { rerender } = renderHook(
      ({ js }: { js: AsyncJobStatus | undefined }) => useJobToasts(js, LABELS),
      { initialProps: { js: undefined } },
    )
    rerender({ js: job('running') })
    rerender({ js: job('succeeded', { current_step: '10 matchs traités' }) })
    expect(toast.success).toHaveBeenCalledOnce()
    expect(toast.success).toHaveBeenCalledWith('Terminé', { description: '10 matchs traités' })
    expect(toast.warning).not.toHaveBeenCalled()
  })

  it('fire toast.warning sur succeeded avec warnings', () => {
    const { rerender } = renderHook(
      ({ js }: { js: AsyncJobStatus | undefined }) => useJobToasts(js, LABELS),
      { initialProps: { js: undefined } },
    )
    rerender({ js: job('running') })
    rerender({ js: job('succeeded', { warnings: ['WARN: weapons ignorés'] }) })
    expect(toast.warning).toHaveBeenCalledOnce()
    expect(toast.warning).toHaveBeenCalledWith('Terminé avec avertissements')
    expect(toast.success).not.toHaveBeenCalled()
  })

  it("fire toast.error sur failed avec le message d'erreur", () => {
    const { rerender } = renderHook(
      ({ js }: { js: AsyncJobStatus | undefined }) => useJobToasts(js, LABELS),
      { initialProps: { js: undefined } },
    )
    rerender({ js: job('running') })
    rerender({ js: job('failed', { error: { code: 'db_error', message: 'connexion perdue' } }) })
    expect(toast.error).toHaveBeenCalledOnce()
    expect(toast.error).toHaveBeenCalledWith('Échoué', expect.objectContaining({
      description: 'connexion perdue',
      duration: Infinity,
    }))
  })

  it('fire toast.warning sur cancelled', () => {
    const { rerender } = renderHook(
      ({ js }: { js: AsyncJobStatus | undefined }) => useJobToasts(js, LABELS),
      { initialProps: { js: undefined } },
    )
    rerender({ js: job('running') })
    rerender({ js: job('cancelled') })
    expect(toast.warning).toHaveBeenCalledWith('Annulé')
  })

  it('fire toast.warning sur interrupted', () => {
    const { rerender } = renderHook(
      ({ js }: { js: AsyncJobStatus | undefined }) => useJobToasts(js, LABELS),
      { initialProps: { js: undefined } },
    )
    rerender({ js: job('running') })
    rerender({ js: job('interrupted') })
    expect(toast.warning).toHaveBeenCalledWith('Annulé')
  })

  it('ne fire pas deux fois si le statut terminal ne change pas (idempotence)', () => {
    const { rerender } = renderHook(
      ({ js }: { js: AsyncJobStatus | undefined }) => useJobToasts(js, LABELS),
      { initialProps: { js: undefined } },
    )
    rerender({ js: job('running') })
    rerender({ js: job('succeeded') })
    rerender({ js: job('succeeded') }) // deuxième appel, même statut
    expect(toast.success).toHaveBeenCalledOnce()
  })

  it('respecte les labels fournis', () => {
    const customLabels: JobToastLabels = {
      succeeded: 'Sync OK',
      succeededWithWarnings: 'Sync OK (warnings)',
      failed: 'Sync KO',
      cancelled: 'Sync annulée',
    }
    const { rerender } = renderHook(
      ({ js }: { js: AsyncJobStatus | undefined }) => useJobToasts(js, customLabels),
      { initialProps: { js: undefined } },
    )
    rerender({ js: job('running') })
    rerender({ js: job('succeeded') })
    expect(toast.success).toHaveBeenCalledWith('Sync OK', expect.any(Object))
  })
})
