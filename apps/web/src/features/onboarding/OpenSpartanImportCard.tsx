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
      setUploadError('Seuls les fichiers .db OpenSpartan sont acceptés.')
      return
    }
    if (file.size > MAX_FILE_SIZE_BYTES) {
      setUploadError(`Fichier trop volumineux (max ${MAX_FILE_SIZE_BYTES >> 30} Go).`)
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
      setUploadError(err instanceof Error ? err.message : "Échec du démarrage de l'import")
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
        <CardTitle>Importer depuis OpenSpartan</CardTitle>
        <CardDescription>
          Si tu as déjà utilisé OpenSpartan, importe ton fichier <code>.db</code> pour récupérer
          des matchs plus anciens que ce que l'API Halo expose.
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
            Glisse-dépose ton fichier <code>.db</code> ici ou clique pour parcourir
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
            Annuler
          </Button>
        )}
        <Button
          onClick={props.onSubmit}
          disabled={!props.selectedFile || props.isSubmitting}
          data-testid="openspartan-submit"
        >
          {props.isSubmitting ? (
            <>
              <Spinner size="sm" /> Envoi…
            </>
          ) : (
            'Importer'
          )}
        </Button>
      </div>
    </>
  )
}

function PollingStage({ job }: { job: AsyncJobStatus | undefined }) {
  const pct = job?.progress_pct ?? null
  return (
    <div className="space-y-3" data-testid="openspartan-polling">
      <div className="flex items-center gap-2 text-sm">
        <Spinner size="sm" />
        <span>{job?.current_step ?? 'Import en cours…'}</span>
      </div>
      {job?.matches_total !== null && job?.matches_total !== undefined && (
        <p className="text-xs text-muted-foreground">
          {job.matches_done ?? 0} / {job.matches_total} matchs traités
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
  if (!result) {
    return (
      <div className="space-y-3" data-testid="openspartan-success">
        <p className="text-sm font-medium" style={{ color: tokenCssVar('success') }}>
          ✓ Import terminé
        </p>
        <Button variant="ghost" onClick={onReset}>
          Importer un autre fichier
        </Button>
      </div>
    )
  }
  const post = result.post_import
  return (
    <div className="space-y-3" data-testid="openspartan-success">
      <p className="text-sm font-medium" style={{ color: tokenCssVar('success') }}>
        ✓ Import réussi
      </p>
      <dl className="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
        <dt className="text-muted-foreground">Matchs importés</dt>
        <dd className="font-mono">
          {result.inserted_matches} / {result.total_matches}
        </dd>
        <dt className="text-muted-foreground">Participants</dt>
        <dd className="font-mono">{result.inserted_participants}</dd>
        <dt className="text-muted-foreground">Médailles</dt>
        <dd className="font-mono">{result.inserted_medals}</dd>
        <dt className="text-muted-foreground">Highlight events</dt>
        <dd className="font-mono">{result.inserted_highlights}</dd>
        <dt className="text-muted-foreground">Alias XUID</dt>
        <dd className="font-mono">{result.inserted_aliases}</dd>
        {post && (
          <>
            <dt className="text-muted-foreground">Sessions calculées</dt>
            <dd className="font-mono">{post.sessions_touched}</dd>
            <dt className="text-muted-foreground">Performance scores</dt>
            <dd className="font-mono">{post.perf_scores_touched}</dd>
          </>
        )}
        {result.errors_count > 0 && (
          <>
            <dt className="text-muted-foreground">Erreurs ignorées</dt>
            <dd className="font-mono" style={{ color: tokenCssVar('warning') }}>
              {result.errors_count}
            </dd>
          </>
        )}
      </dl>
      <Button variant="ghost" onClick={onReset}>
        Importer un autre fichier
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
  return (
    <div className="space-y-3" data-testid="openspartan-failure">
      <p className="text-sm font-medium" style={{ color: tokenCssVar('destructive') }}>
        {status === 'interrupted' ? "Import interrompu" : "Échec de l'import"}
      </p>
      <p className="text-sm">{failureMessageFromCode(error)}</p>
      <Button variant="ghost" onClick={onReset}>
        Réessayer
      </Button>
    </div>
  )
}

/**
 * failureMessageFromCode maps the typed Error.Code returned by the backend
 * service to a user-facing French sentence. Falls back to the raw message
 * when the code is unknown.
 */
export function failureMessageFromCode(err: ApiErrorSchema | null): string {
  if (!err) return 'Erreur inconnue.'
  switch (err.code) {
    case 'xuid_mismatch':
      return "Cette base OpenSpartan n'appartient pas à ton compte Xbox connecté."
    case 'owner_low_confidence':
      return 'Impossible de vérifier que cette base est bien la tienne. Renomme le fichier en <ton-xuid>.db et réessaye.'
    case 'not_openspartan_db':
      return "Ce fichier ne ressemble pas à une base OpenSpartan reconnaissable."
    case 'upload_too_large':
      return 'Fichier trop volumineux (max 1 Go).'
    case 'demo_mode':
      return "L'import OpenSpartan est désactivé en mode démo."
    case 'halo_auth_required':
      return 'Connexion Xbox/Halo requise pour lancer un import.'
    default:
      return err.message || 'Erreur inconnue.'
  }
}
