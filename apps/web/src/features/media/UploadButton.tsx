/**
 * UploadButton — bouton + zone de drag & drop pour importer des médias.
 */
import { useRef, useState } from 'react'
import { useUploadMedia } from './queries'

const ACCEPTED_EXTS = '.mp4,.mov,.avi,.mkv,.webm,.png,.jpg,.jpeg,.bmp,.gif'
const ACCEPTED_MIME = new Set([
  'video/mp4', 'video/quicktime', 'video/x-msvideo', 'video/x-matroska', 'video/webm',
  'image/png', 'image/jpeg', 'image/bmp', 'image/gif',
])

interface Props {
  playerSlug: string
}

export function UploadButton({ playerSlug }: Props) {
  const inputRef = useRef<HTMLInputElement>(null)
  const { mutate, isPending, isSuccess, data, error, reset } = useUploadMedia(playerSlug)
  const [isDragging, setIsDragging] = useState(false)

  function handleFiles(list: FileList | null) {
    if (!list || list.length === 0) return
    reset()
    mutate(Array.from(list))
    if (inputRef.current) inputRef.current.value = ''
  }

  function onDragOver(e: React.DragEvent) {
    e.preventDefault()
    setIsDragging(true)
  }

  function onDragLeave(e: React.DragEvent) {
    // Ignorer les événements générés par les enfants
    if (e.currentTarget.contains(e.relatedTarget as Node)) return
    setIsDragging(false)
  }

  function onDrop(e: React.DragEvent) {
    e.preventDefault()
    setIsDragging(false)
    const files = filterAccepted(e.dataTransfer.files)
    handleFiles(files)
  }

  return (
    <div className="flex flex-col gap-2">
      <input
        ref={inputRef}
        type="file"
        multiple
        accept={ACCEPTED_EXTS}
        className="hidden"
        aria-hidden="true"
        onChange={(e) => handleFiles(e.target.files)}
      />

      {/* Zone de drop */}
      <div
        onDragOver={onDragOver}
        onDragLeave={onDragLeave}
        onDrop={onDrop}
        onClick={() => !isPending && inputRef.current?.click()}
        className={[
          'flex cursor-pointer flex-col items-center justify-center gap-1.5 rounded-lg border-2 border-dashed px-6 py-4 text-sm transition-colors select-none',
          isPending
            ? 'cursor-not-allowed border-input text-muted-foreground'
            : isDragging
              ? 'border-primary bg-primary/10 text-primary'
              : 'border-input text-muted-foreground hover:border-border hover:text-foreground',
        ].join(' ')}
      >
        {isPending ? (
          <>
            <span className="animate-spin text-xl">⏳</span>
            <span>Import en cours…</span>
          </>
        ) : isDragging ? (
          <>
            <span className="text-xl">📂</span>
            <span className="font-medium">Déposer les fichiers</span>
          </>
        ) : (
          <>
            <span className="text-xl">⬆</span>
            <span>Glisser-déposer ou <span className="underline">parcourir</span></span>
            <span className="text-xs opacity-60">MP4, MOV, AVI, MKV, PNG, JPG…</span>
          </>
        )}
      </div>

      {isSuccess && data && (
        <p className="text-xs text-success">
          {data.saved} fichier{data.saved !== 1 ? 's' : ''} importé{data.saved !== 1 ? 's' : ''}&nbsp;·&nbsp;
          {data.new_indexed} nouveau{data.new_indexed !== 1 ? 'x' : ''}&nbsp;·&nbsp;
          {data.associated} assoc.
          {data.thumbnails > 0 && <>&nbsp;· {data.thumbnails} miniature{data.thumbnails !== 1 ? 's' : ''}</>}
        </p>
      )}

      {error != null && (
        <p className="text-xs text-destructive" role="alert">
          {(error as { message?: string }).message ?? 'Erreur lors de l\'import'}
        </p>
      )}
    </div>
  )
}

function filterAccepted(files: FileList): FileList {
  const dt = new DataTransfer()
  Array.from(files).forEach((f) => {
    if (ACCEPTED_MIME.has(f.type)) dt.items.add(f)
  })
  return dt.files
}
