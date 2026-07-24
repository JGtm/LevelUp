/**
 * AdminJobsTable.test.tsx — tri CLIENT par clic sur les en-têtes (I16).
 */
import { describe, it, expect } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import type { AsyncJobStatus } from '@/lib/api/types'
import { AdminJobsTable } from './AdminJobsTable'

function makeJob(overrides: Partial<AsyncJobStatus> & { job_id: string; job_type: string }): AsyncJobStatus {
  return {
    status: 'succeeded',
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

/** Ordre des lignes du tbody, identifiées par le type de job (rendu tel quel,
 *  types inconnus du mapping FR/EN — cf. jobTypeLabel fallback). */
function rowOrder(names: string[]): string[] {
  const tbody = document.querySelector('tbody')
  return Array.from(tbody?.querySelectorAll('tr') ?? []).map((tr) => {
    const txt = tr.textContent ?? ''
    return names.find((n) => txt.includes(n)) ?? '?'
  })
}

describe('AdminJobsTable — tri CLIENT par en-têtes (I16)', () => {
  const names = ['alpha', 'bravo', 'charlie']

  function jobs(): AsyncJobStatus[] {
    return [
      makeJob({ job_id: 'j1', job_type: 'alpha' }),
      makeJob({ job_id: 'j2', job_type: 'bravo' }),
      makeJob({ job_id: 'j3', job_type: 'charlie' }),
    ]
  }

  it('sans clic : ordre serveur conservé', () => {
    render(<AdminJobsTable jobs={jobs()} />)
    expect(rowOrder(names)).toEqual(['alpha', 'bravo', 'charlie'])
  })

  it('clic sur « Type » trie alphabétiquement (asc au 1er clic)', () => {
    render(<AdminJobsTable jobs={[makeJob({ job_id: 'j1', job_type: 'charlie' }), makeJob({ job_id: 'j2', job_type: 'alpha' }), makeJob({ job_id: 'j3', job_type: 'bravo' })]} />)
    const header = screen.getByText('Type')
    fireEvent.click(header)
    expect(rowOrder(names)).toEqual(['alpha', 'bravo', 'charlie'])
  })
})
