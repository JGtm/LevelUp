/**
 * queries.ts — OpenSpartan import upload mutation + result typing.
 *
 * The job status itself is polled via the generic useJobStatus hook from
 * features/setup (cached at /api/v1/jobs/{job_id}).
 */
import { useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api/client'

/** Response of POST /api/v1/import/openspartan (202 Accepted). */
export interface ImportStartResponse {
  job_id: string
  status: string
}

/**
 * OpenSpartanImportResult mirrors the `Result` map the backend writes onto
 * the job entry when the import succeeds. All fields come from JSON, so
 * numbers arrive as JS numbers. Keys match the backend exactly.
 */
export interface OpenSpartanImportResult {
  detected_owner_xuid: string
  confidence: string
  total_matches: number
  inserted_matches: number
  inserted_participants: number
  inserted_medals: number
  inserted_highlights: number
  inserted_aliases: number
  stashed_friends: number
  errors_count: number
  post_import?: {
    sessions_touched: number
    perf_scores_touched: number
    citations_backfilled: boolean
    errors_count: number
  }
}

/**
 * useStartOpenSpartanImport posts the SQLite file as multipart/form-data
 * under field name "db" and returns the async job descriptor.
 */
export function useStartOpenSpartanImport() {
  return useMutation({
    mutationFn: async (file: File) => {
      const form = new FormData()
      form.append('db', file, file.name)
      return api.postForm<ImportStartResponse>('/import/openspartan', form)
    },
  })
}
