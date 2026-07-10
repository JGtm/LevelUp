/**
 * Tests — OpenSpartanImportCard.
 *
 * Couvre :
 *   - rendu idle (dropzone + bouton Importer désactivé)
 *   - validation extension côté front
 *   - happy path : upload → polling → succès avec counts
 *   - chemin d'erreur typé (xuid_mismatch)
 *   - reset après succès
 */
import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

import { OpenSpartanImportCard, failureMessageFromCode } from './OpenSpartanImportCard'
import { server } from '@/test/setup'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

const API = '/api/v1'

// Traducteur FR pour les tests unitaires de failureMessageFromCode (le composant
// injecte le t() du store locale ; ici on force 'fr' pour asserter les libellés).
const tFr = (key: CommonManifestKey, vars?: Record<string, string | number>) =>
  formatMessage(commonManifest, key, 'fr', vars)

function renderCard() {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, refetchOnWindowFocus: false, staleTime: 0 },
      mutations: { retry: false },
    },
  })
  return render(
    <QueryClientProvider client={client}>
      <OpenSpartanImportCard />
    </QueryClientProvider>,
  )
}

function makeDBFile(name = 'fake.db', sizeBytes = 1024): File {
  const buffer = new ArrayBuffer(sizeBytes)
  return new File([buffer], name, { type: 'application/octet-stream' })
}

describe('OpenSpartanImportCard — idle state', () => {
  it('shows the dropzone and the submit button is disabled by default', () => {
    renderCard()
    expect(screen.getByTestId('openspartan-dropzone')).toBeTruthy()
    const submit = screen.getByTestId('openspartan-submit') as HTMLButtonElement
    expect(submit.disabled).toBe(true)
  })

  it('rejects non-.db files with an inline error message', () => {
    renderCard()
    const input = screen.getByTestId('openspartan-file-input') as HTMLInputElement
    fireEvent.change(input, {
      target: { files: [new File(['text'], 'report.txt', { type: 'text/plain' })] },
    })
    expect(screen.getByRole('alert').textContent).toMatch(/\.db OpenSpartan/i)
    const submit = screen.getByTestId('openspartan-submit') as HTMLButtonElement
    expect(submit.disabled).toBe(true)
  })

  it('enables the submit button once a .db file is selected', () => {
    renderCard()
    const input = screen.getByTestId('openspartan-file-input') as HTMLInputElement
    fireEvent.change(input, { target: { files: [makeDBFile()] } })
    const submit = screen.getByTestId('openspartan-submit') as HTMLButtonElement
    expect(submit.disabled).toBe(false)
    // The dropzone now shows the selected filename.
    expect(screen.getByTestId('openspartan-dropzone').textContent).toMatch(/fake\.db/)
  })
})

describe('OpenSpartanImportCard — happy path', () => {
  it('uploads, polls, and renders the success counts', async () => {
    // NOTE: undici's multipart parser in jsdom doesn't always accept the
    // File constructed via `new File([ArrayBuffer], …)`. We don't inspect
    // the body here — the happy path is fully covered by the success
    // response below + downstream rendering assertions.
    server.use(
      http.post(`${API}/import/openspartan`, () =>
        HttpResponse.json({ job_id: 'job-os-1', status: 'queued' }, { status: 202 }),
      ),
      http.get(`${API}/jobs/job-os-1`, () =>
        HttpResponse.json({
          job_id: 'job-os-1',
          job_type: 'openspartan_import',
          status: 'succeeded',
          progress_pct: 100,
          current_step: null,
          started_at: null,
          finished_at: null,
          result: {
            detected_owner_xuid: '2533274823110022',
            confidence: 'high',
            total_matches: 12,
            inserted_matches: 12,
            inserted_participants: 95,
            inserted_medals: 200,
            inserted_highlights: 50,
            inserted_aliases: 4,
            stashed_friends: 1,
            errors_count: 0,
            post_import: {
              sessions_touched: 12,
              perf_scores_touched: 0,
              citations_backfilled: true,
              errors_count: 0,
            },
          },
          error: null,
          phase_key: null,
          phase_label: null,
          matches_done: 12,
          matches_total: 12,
          subtasks_done: null,
          subtasks_total: null,
          eta_seconds: null,
          warnings: [],
        }),
      ),
    )

    renderCard()
    const input = screen.getByTestId('openspartan-file-input') as HTMLInputElement
    fireEvent.change(input, { target: { files: [makeDBFile()] } })
    // Wait for React to commit the state update before clicking — otherwise
    // the click can hit the still-disabled button and be a no-op.
    const submit = screen.getByTestId('openspartan-submit') as HTMLButtonElement
    await waitFor(() => expect(submit.disabled).toBe(false))
    fireEvent.click(submit)

    await waitFor(
      () => {
        expect(screen.getByTestId('openspartan-success')).toBeTruthy()
      },
      { timeout: 5000 },
    )

    const success = screen.getByTestId('openspartan-success')
    expect(success.textContent).toMatch(/Import réussi/i)
    expect(within(success).getByText('12 / 12')).toBeTruthy() // Matchs importés
    expect(within(success).getByText('200')).toBeTruthy() // Médailles
    expect(within(success).getByText(/sessions calcul/i)).toBeTruthy() // Post-import block visible
  })
})

describe('OpenSpartanImportCard — failure path', () => {
  it('renders the XUID mismatch message and offers a retry', async () => {
    server.use(
      http.post(`${API}/import/openspartan`, () =>
        HttpResponse.json({ job_id: 'job-os-fail', status: 'queued' }, { status: 202 }),
      ),
      http.get(`${API}/jobs/job-os-fail`, () =>
        HttpResponse.json({
          job_id: 'job-os-fail',
          job_type: 'openspartan_import',
          status: 'failed',
          progress_pct: null,
          current_step: null,
          started_at: null,
          finished_at: null,
          result: null,
          error: {
            code: 'xuid_mismatch',
            message: 'detected XXX, expected YYY',
            retryable: false,
          },
          phase_key: null,
          phase_label: null,
          matches_done: null,
          matches_total: null,
          subtasks_done: null,
          subtasks_total: null,
          eta_seconds: null,
          warnings: [],
        }),
      ),
    )

    renderCard()
    fireEvent.change(screen.getByTestId('openspartan-file-input') as HTMLInputElement, {
      target: { files: [makeDBFile()] },
    })
    const submit = screen.getByTestId('openspartan-submit') as HTMLButtonElement
    await waitFor(() => expect(submit.disabled).toBe(false))
    fireEvent.click(submit)

    await waitFor(
      () => {
        expect(screen.getByTestId('openspartan-failure')).toBeTruthy()
      },
      { timeout: 5000 },
    )
    const failure = screen.getByTestId('openspartan-failure')
    expect(failure.textContent).toMatch(/n'appartient pas à ton compte/i)
    // Retry button drops us back to idle.
    fireEvent.click(within(failure).getByRole('button', { name: /réessayer/i }))
    expect(screen.getByTestId('openspartan-dropzone')).toBeTruthy()
  })
})

describe('failureMessageFromCode — unit', () => {
  it('maps every documented Error.Code to a localised sentence', () => {
    const cases: Array<{ code: string; expected: RegExp }> = [
      { code: 'xuid_mismatch', expected: /pas à ton compte Xbox/i },
      { code: 'owner_low_confidence', expected: /vérifier/i },
      { code: 'not_openspartan_db', expected: /OpenSpartan reconnaissable/i },
      { code: 'upload_too_large', expected: /max 1 Go/i },
      { code: 'demo_mode', expected: /mode démo/i },
      { code: 'halo_auth_required', expected: /Xbox\/Halo/i },
    ]
    for (const c of cases) {
      const msg = failureMessageFromCode({ code: c.code, message: '', retryable: false }, tFr)
      expect(msg).toMatch(c.expected)
    }
  })

  it('falls back to the raw message for unknown codes', () => {
    const msg = failureMessageFromCode({ code: 'unknown_code', message: 'oops', retryable: false }, tFr)
    expect(msg).toBe('oops')
  })

  it('falls back to a default message when error is null', () => {
    expect(failureMessageFromCode(null, tFr)).toMatch(/inconnue/i)
  })
})
