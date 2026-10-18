/**
 * BuildQueueSection.test.tsx — ce que l'admin doit VOIR de la file de
 * construction et de ses ouvriers : le travail délégué, l'ouvrier qui le tient,
 * un ouvrier mort qui reste visible mais hors ligne, et l'avertissement quand
 * aucun ouvrier ne peut venir vider la file.
 */
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { AdminBuildQueueResponse } from '@/lib/api/types'
import { BuildQueueSection } from './BuildQueueSection'

const dataRef: { current: AdminBuildQueueResponse | undefined } = { current: undefined }

vi.mock('../monitoring/queries', () => ({
  useBuildQueue: () => ({ data: dataRef.current, isError: false }),
}))

function baseResponse(overrides: Partial<AdminBuildQueueResponse> = {}): AdminBuildQueueResponse {
  return {
    generated_at: '2026-08-14T10:00:00Z',
    enabled: true,
    counts: { queued: 1, running: 1, succeeded: 3, failed: 0 },
    jobs: [
      {
        job_id: 'job_1',
        job_type: 'replay_build',
        match_id: 'aaaa-bbbb-cccc',
        status: 'running',
        priority: 0,
        attempt: 0,
        worker_id: 'ouvrier-1',
        updated_at: '2026-08-14T09:59:00Z',
      },
      {
        job_id: 'job_2',
        job_type: 'replay_build',
        match_id: 'dddd-eeee-ffff',
        status: 'queued',
        priority: 0,
        attempt: 0,
        updated_at: '2026-08-14T09:58:00Z',
      },
    ],
    workers: [
      {
        worker_id: 'ouvrier-1',
        hostname: 'second-vps',
        jobs_done: 12,
        jobs_failed: 1,
        current_job_id: 'job_1',
        last_beat_at: '2026-08-14T09:59:50Z',
        online: true,
      },
      {
        worker_id: 'ouvrier-mort',
        hostname: 'poste',
        jobs_done: 4,
        jobs_failed: 0,
        last_beat_at: '2026-08-14T08:00:00Z',
        online: false,
      },
    ],
    match_delays: [
      {
        title_slug: 'halo_infinite',
        match_id: 'match-debrief-1',
        ended_at: '2026-08-14T09:50:00Z',
        citations_players: 2,
        total_seconds: 250,
        objective_met: true,
        steps: [
          { step: 'match_end', at: '2026-08-14T09:50:00Z', offset_seconds: 0 },
          { step: 'replay_enqueued', at: '2026-08-14T09:51:35Z', offset_seconds: 95 },
          { step: 'derivations_written', at: '2026-08-14T09:53:45Z', offset_seconds: 225 },
          { step: 'citations_written', at: '2026-08-14T09:54:10Z', offset_seconds: 250, detail: 'x2' },
        ],
      },
    ],
    ...overrides,
  }
}

describe('BuildQueueSection', () => {
  it('affiche les jobs de la file avec l’ouvrier qui les traite', () => {
    dataRef.current = baseResponse()
    render(<BuildQueueSection />)
    expect(screen.getByText('aaaa-bbbb-cccc')).toBeInTheDocument()
    expect(screen.getByText('dddd-eeee-ffff')).toBeInTheDocument()
    // L'ouvrier apparaît dans la colonne du job ET dans le tableau des ouvriers.
    expect(screen.getAllByText('ouvrier-1').length).toBeGreaterThanOrEqual(2)
  })

  it('un ouvrier disparu reste visible, marqué hors ligne', () => {
    dataRef.current = baseResponse()
    render(<BuildQueueSection />)
    expect(screen.getByText('ouvrier-mort')).toBeInTheDocument()
    expect(screen.getByText(/hors ligne|offline/i)).toBeInTheDocument()
  })

  it('sans jeton d’ouvrier, prévient que personne ne viendra vider la file', () => {
    dataRef.current = baseResponse({ enabled: false })
    render(<BuildQueueSection />)
    expect(screen.getByText(/personne ne viendra la vider|nobody will drain it/i)).toBeInTheDocument()
  })

  it('file vide : état vide explicite, pas un tableau nu', () => {
    dataRef.current = baseResponse({
      jobs: [],
      workers: [],
      counts: { queued: 0, running: 0, succeeded: 0, failed: 0 },
    })
    render(<BuildQueueSection />)
    expect(screen.getByText(/file vide|empty queue/i)).toBeInTheDocument()
    expect(screen.getByText(/aucun ouvrier connu|no known worker/i)).toBeInTheDocument()
  })

  it('délai du débrief : écart de chaque étape depuis la fin du match et objectif', () => {
    dataRef.current = baseResponse()
    render(<BuildQueueSection />)
    expect(screen.getByText('match-debrief-1')).toBeInTheDocument()
    expect(screen.getByText('+1:35')).toBeInTheDocument()
    // Les citations ferment le chemin : leur écart est aussi le total.
    expect(screen.getAllByText('+4:10')).toHaveLength(2)
    expect(screen.getByText(/^(tenu|met)$/i)).toBeInTheDocument()
  })

  it('délai du débrief : aucun match mesuré, état vide explicite', () => {
    dataRef.current = baseResponse({ match_delays: [] })
    render(<BuildQueueSection />)
    expect(screen.getByText(/aucun match mesuré|no measured match/i)).toBeInTheDocument()
  })
})
