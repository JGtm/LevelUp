/**
 * OpenSpartanImportCard — multipart upload of a SQLite OpenSpartan `.db`,
 * async job polling, and result display.
 *
 * Lifecycle:
 *   1. Idle           → dropzone + file picker
 *   2. file selected  → Import button enabled
 *   3. uploading      → mutation pending
 *   4. polling        → useJobStatus polls /jobs/{id} every 3s until terminal
 *   5. succeeded      → counts (matches/medals/sessions/perf) + reset CTA
 *   6. failed         → typed message keyed on Error.Code + retry CTA
 *
 * The backend stages the file to a temp path, validates the owner XUID
 * against the SSO session, then runs import + post-import recompute in a
 * background goroutine.
 */
import { useState, type DragEvent, type ChangeEvent } from 'react'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { useJobStatus } from '@/features/setup/queries'
import type { AsyncJobStatus, ApiErrorSchema } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

import { useStartOpenSpartanImport, type OpenSpartanImportResult } from './queries'

const MAX_FILE_SIZE_BYTES = 1 << 30 // 1 GiB

export function OpenSpartanImportCard() {
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [jobId, setJobId] = useState<string | null>(null)
  const [dragOver, setDragOver] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)
  // Incremented on reset to re-mount the <input type="file">, the only
  // reliable way to clear a file input without touching refs during render.
  const [fileInputKey, setFileInputKey] = useState(0)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey, vars?: Record<string, string | number>) =>
    formatMessage(commonManifest, key, locale, vars)

  const startMutation = useStartOpenSpartanImport()
  const jobQuery = useJobStatus(jobId ?? '', !!jobId)
  const job = jobQuery.data

  const status = job?.status
  const isTerminal = status === 'succeeded' || status === 'failed' || status === 'cancelled' || status === 'interrupted'
  const isPolling = !!jobId && !isTerminal

  function handleSelect(file: File | null) {
    setUploadError(null)
    if (!file) {
      setSelectedFile(null)
      return
    }
    if (!file.name.toLowerCase().endsWith('.db')) {
      setUploadError(t('common.onboarding.import_only_db'))
      return
    }
    if (file.size > MAX_FILE_SIZE_BYTES) {
      setUploadError(t('common.onboarding.import_too_large', { max: MAX_FILE_SIZE_BYTES >> 30 }))
      return
    }
    setSelectedFile(file)
  }

  async function handleSubmit() {
    if (!selectedFile) return
    setUploadError(null)
    try {
      const resp = await startMutation.mutateAsync(selectedFile)
      setJobId(resp.job_id)
    } catch (err) {
      setUploadError(err instanceof Error ? err.message : t('common.onboarding.import_start_failed'))
    }
  }

  function handleDrop(e: DragEvent<HTMLLabelElement>) {
    e.preventDefault()
    setDragOver(false)
    handleSelect(e.dataTransfer.files?.[0] ?? null)
  }

  function handleDragOver(e: DragEvent<HTMLLabelElement>) {
    e.preventDefault()
    setDragOver(true)
  }

  function handleReset() {
    setSelectedFile(null)
    setJobId(null)
    setUploadError(null)
    setFileInputKey((k) => k + 1)
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('common.onboarding.import_card_title')}</CardTitle>
        <CardDescription>
          {t('common.onboarding.import_card_desc')} <code>.db</code> {t('common.onboarding.import_card_desc_suffix')}
        </CardDescription>
      </CardHeader>
      <CardContent>
        {!jobId && (
          <IdleStage
            selectedFile={selectedFile}
            dragOver={dragOver}
            uploadError={uploadError}
            isSubmitting={startMutation.isPending}
            fileInputKey={fileInputKey}
            onFileChange={(e: ChangeEvent<HTMLInputElement>) => handleSelect(e.target.files?.[0] ?? null)}
            onDrop={handleDrop}
            onDragOver={handleDragOver}
            onDragLeave={() => setDragOver(false)}
            onSubmit={handleSubmit}
            onReset={handleReset}
          />
        )}
        {isPolling && <PollingStage job={job} />}
        {jobId && status === 'succeeded' && (
          <SuccessStage result={job?.result as OpenSpartanImportResult | null} onReset={handleReset} />
        )}
        {jobId && (status === 'failed' || status === 'cancelled' || status === 'interrupted') && (
          <FailureStage error={job?.error ?? null} status={status} onReset={handleReset} />
        )}
      </CardContent>
    </Card>
  )
}

// ─── STAGES ──────────────────────────────────────────────────────────────────

interface IdleStageProps {
  selectedFile: File | null
  dragOver: boolean
  uploadError: string | null
  isSubmitting: boolean
  fileInputKey: number
  onFileChange: (e: ChangeEvent<HTMLInputElement>) => void
  onDrop: (e: DragEvent<HTMLLabelElement>) => void
  onDragOver: (e: DragEvent<HTMLLabelElement>) => void
  onDragLeave: () => void
  onSubmit: () => void
  onReset: () => void
}

function IdleStage(props: IdleStageProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const dropzoneClass = `block border-2 border-dashed rounded-md p-6 text-center cursor-pointer transition-colors ${
    props.dragOver ? 'border-primary bg-primary/10' : 'border-border hover:border-primary/50'
  }`
  return (
    <>
      {/*
       * Native <label> wrapping a hidden file input gives us free
       * accessibility (click/keyboard activation, focus ring on the label)
       * without a ref or programmatic .click() — and re-mounting the input
       * via `key` clears its value on reset.
       */}
      <label
        htmlFor="openspartan-file"
        className={dropzoneClass}
        onDrop={props.onDrop}
        onDragOver={props.onDragOver}
        onDragLeave={props.onDragLeave}
        data-testid="openspartan-dropzone"
      >
        {props.selectedFile ? (
          <p className="text-sm">
            <strong>{props.selectedFile.name}</strong>{' '}
            <span className="text-muted-foreground">
              ({(props.selectedFile.size / 1024 / 1024).toFixed(1)} Mo)
            </span>
          </p>
        ) : (
          <p className="text-sm text-muted-foreground">
            {t('common.onboarding.drop_file_here')} <code>.db</code> {t('common.onboarding.drop_file_or_browse')}
          </p>
        )}
        <input
          key={props.fileInputKey}
          id="openspartan-file"
          type="file"
          accept=".db"
          className="sr-only"
          onChange={props.onFileChange}
          data-testid="openspartan-file-input"
        />
      </label>

      {props.uploadError && (
        <p
          className="mt-3 text-sm"
          style={{ color: tokenCssVar('destructive') }}
          role="alert"
        >
          {props.uploadError}
        </p>
      )}

      <div className="mt-4 flex justify-end gap-2">
        {props.selectedFile && (
          <Button variant="ghost" onClick={props.onReset} disabled={props.isSubmitting}>
            {t('common.onboarding.cancel')}
          </Button>
        )}
        <Button
          onClick={props.onSubmit}
          disabled={!props.selectedFile || props.isSubmitting}
          data-testid="openspartan-submit"
        >
          {props.isSubmitting ? (
            <>
              <Spinner size="sm" /> {t('common.onboarding.uploading')}
            </>
          ) : (
            t('common.onboarding.import_action')
          )}
        </Button>
      </div>
    </>
  )
}

function PollingStage({ job }: { job: AsyncJobStatus | undefined }) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const pct = job?.progress_pct ?? null
  return (
    <div className="space-y-3" data-testid="openspartan-polling">
      <div className="flex items-center gap-2 text-sm">
        <Spinner size="sm" />
        <span>{job?.current_step ?? t('common.onboarding.importing')}</span>
      </div>
      {job?.matches_total !== null && job?.matches_total !== undefined && (
        <p className="text-xs text-muted-foreground">
          {job.matches_done ?? 0} / {job.matches_total} {t('common.onboarding.matches_processed')}
        </p>
      )}
      {pct !== null && pct !== undefined && (
        <div className="w-full bg-muted rounded-full h-2 overflow-hidden">
          <div
            className="h-2 rounded-full transition-all bg-primary"
            style={{ width: `${pct}%` }}
          />
        </div>
      )}
    </div>
  )
}

function SuccessStage({
  result,
  onReset,
}: {
  result: OpenSpartanImportResult | null
  onReset: () => void
}) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  if (!result) {
    return (
      <div className="space-y-3" data-testid="openspartan-success">
        <p className="text-sm font-medium" style={{ color: tokenCssVar('success') }}>
          {t('common.onboarding.import_finished')}
        </p>
        <Button variant="ghost" onClick={onReset}>
          {t('common.onboarding.import_another_file')}
        </Button>
      </div>
    )
  }
  const post = result.post_import
  return (
    <div className="space-y-3" data-testid="openspartan-success">
      <p className="text-sm font-medium" style={{ color: tokenCssVar('success') }}>
        {t('common.onboarding.import_success')}
      </p>
      <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
        <dt className="text-muted-foreground">{t('common.onboarding.matches_imported')}</dt>
        <dd className="font-mono">
          {result.inserted_matches} / {result.total_matches}
        </dd>
        <dt className="text-muted-foreground">{t('common.onboarding.participants')}</dt>
        <dd className="font-mono">{result.inserted_participants}</dd>
        <dt className="text-muted-foreground">{t('common.onboarding.medals')}</dt>
        <dd className="font-mono">{result.inserted_medals}</dd>
        <dt className="text-muted-foreground">{t('common.onboarding.highlight_events')}</dt>
        <dd className="font-mono">{result.inserted_highlights}</dd>
        <dt className="text-muted-foreground">{t('common.onboarding.xuid_aliases')}</dt>
        <dd className="font-mono">{result.inserted_aliases}</dd>
        {post && (
          <>
            <dt className="text-muted-foreground">{t('common.onboarding.sessions_computed')}</dt>
            <dd className="font-mono">{post.sessions_touched}</dd>
            <dt className="text-muted-foreground">{t('common.onboarding.perf_scores')}</dt>
            <dd className="font-mono">{post.perf_scores_touched}</dd>
          </>
        )}
        {result.errors_count > 0 && (
          <>
            <dt className="text-muted-foreground">{t('common.onboarding.errors_ignored')}</dt>
            <dd className="font-mono" style={{ color: tokenCssVar('warning') }}>
              {result.errors_count}
            </dd>
          </>
        )}
      </dl>
      <p className="text-xs text-muted-foreground">{t('common.onboarding.combat_details_note')}</p>
      <Button variant="ghost" onClick={onReset}>
        {t('common.onboarding.import_another_file')}
      </Button>
    </div>
  )
}

function FailureStage({
  error,
  status,
  onReset,
}: {
  error: ApiErrorSchema | null
  status: string | undefined
  onReset: () => void
}) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey, vars?: Record<string, string | number>) =>
    formatMessage(commonManifest, key, locale, vars)
  return (
    <div className="space-y-3" data-testid="openspartan-failure">
      <p className="text-sm font-medium" style={{ color: tokenCssVar('destructive') }}>
        {status === 'interrupted'
          ? t('common.onboarding.import_interrupted')
          : t('common.onboarding.import_failed')}
      </p>
      <p className="text-sm">{failureMessageFromCode(error, t)}</p>
      <Button variant="ghost" onClick={onReset}>
        {t('common.onboarding.retry')}
      </Button>
    </div>
  )
}

/**
 * failureMessageFromCode maps the typed Error.Code returned by the backend
 * service to a localised, user-facing sentence via the injected translator.
 * Falls back to the raw message when the code is unknown.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function failureMessageFromCode(
  err: ApiErrorSchema | null,
  t: (key: CommonManifestKey, vars?: Record<string, string | number>) => string,
): string {
  if (!err) return t('common.onboarding.import_err_unknown')
  switch (err.code) {
    case 'xuid_mismatch':
      return t('common.onboarding.import_err_xuid_mismatch')
    case 'owner_low_confidence':
      return t('common.onboarding.import_err_low_confidence')
    case 'not_openspartan_db':
      return t('common.onboarding.import_err_not_openspartan')
    case 'upload_too_large':
      return t('common.onboarding.import_too_large', { max: 1 })
    case 'demo_mode':
      return t('common.onboarding.import_err_demo')
    case 'halo_auth_required':
      return t('common.onboarding.import_err_auth_required')
    default:
      return err.message || t('common.onboarding.import_err_unknown')
  }
}
