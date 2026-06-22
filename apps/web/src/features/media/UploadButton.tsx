/**
 * UploadButton — bouton + zone de drag & drop pour importer des médias.
 */
import { useRef, useState } from 'react'
import { toast } from 'sonner'
import { useUploadMedia } from './queries'
import { apiErrorMessage } from '@/lib/api/client'
import { Spinner } from '@/components/ui/spinner'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

/** Flèche montante vers une barre — état idle de la zone de drop. */
function UploadIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden="true"
    >
      <path d="M12 16V4" />
      <path d="m7 9 5-5 5 5" />
      <path d="M5 20h14" />
    </svg>
  )
}

/** Dossier ouvert — état drag-over de la zone de drop. */
function FolderOpenIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden="true"
    >
      <path d="M6 14l1.45-2.9A2 2 0 0 1 9.24 10H20a2 2 0 0 1 1.94 2.5l-1.55 6A2 2 0 0 1 18.45 20H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h3.93a2 2 0 0 1 1.66.9l.82 1.2a2 2 0 0 0 1.66.9H18a2 2 0 0 1 2 2v2" />
    </svg>
  )
}

const ACCEPTED_EXTS = '.mp4,.mov,.avi,.mkv,.webm,.png,.jpg,.jpeg,.bmp,.gif'
const ACCEPTED_MIME = new Set([
  'video/mp4', 'video/quicktime', 'video/x-msvideo', 'video/x-matroska', 'video/webm',
  'image/png', 'image/jpeg', 'image/bmp', 'image/gif',
])

interface Props {
  playerSlug: string
  /** Afficher la zone de drop pleine largeur (mode galerie) */
  fullWidth?: boolean
}

export function UploadButton({ playerSlug, fullWidth = false }: Props) {
  const inputRef = useRef<HTMLInputElement>(null)
  const { mutate, isPending, reset } = useUploadMedia(playerSlug)
  const [isDragging, setIsDragging] = useState(false)
  const locale = useAppShellStore((s) => s.locale)
  // En démo, l'upload est figé : la zone reste visible mais inerte (le serveur
  // refuse aussi l'upload, cf. PostUploadMedia). Pas de modification possible.
  const demoMode = useAppShellStore((s) => s.demoMode)
  const t = (key: CommonManifestKey, vars?: Record<string, unknown>) =>
    formatMessage(commonManifest, key, locale, vars)

  function handleFiles(list: FileList | null) {
    if (demoMode || !list || list.length === 0) return
    reset()
    mutate(Array.from(list), {
      onSuccess: (data) =>
        toast.success(
          t('common.media.upload_success', {
            saved: data.saved,
            indexed: data.new_indexed,
            associated: data.associated,
            thumbnails: data.thumbnails,
          }),
        ),
      onError: (err) =>
        toast.error(t('common.media.upload_error'), {
          description: apiErrorMessage(err),
        }),
    })
    if (inputRef.current) inputRef.current.value = ''
  }

  function onDragOver(e: React.DragEvent) {
    if (demoMode) return
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
    if (demoMode) return
    const files = filterAccepted(e.dataTransfer.files)
    handleFiles(files)
  }

  return (
    <div className={fullWidth ? 'flex w-full flex-col gap-2' : 'flex flex-col gap-2'}>
      <input
        ref={inputRef}
        type="file"
        multiple
        accept={ACCEPTED_EXTS}
        className="hidden"
        aria-hidden="true"
        disabled={demoMode}
        onChange={(e) => handleFiles(e.target.files)}
      />

      {/* Zone de drop */}
      <div
        onDragOver={onDragOver}
        onDragLeave={onDragLeave}
        onDrop={onDrop}
        onClick={() => !demoMode && !isPending && inputRef.current?.click()}
        className={[
          'flex flex-col items-center justify-center gap-1.5 rounded-lg border-2 border-dashed text-sm transition-colors select-none',
          fullWidth ? 'w-full px-6 py-6' : 'px-6 py-4',
          demoMode
            ? 'cursor-not-allowed border-input text-muted-foreground opacity-50'
            : isPending
              ? 'cursor-not-allowed border-input text-muted-foreground'
              : isDragging
                ? 'cursor-pointer border-primary bg-primary/10 text-primary'
                : 'cursor-pointer border-input text-muted-foreground hover:border-border hover:text-foreground',
        ].join(' ')}
      >
        {isPending ? (
          <>
            <Spinner size="sm" />
            <span>{t('common.media.import_in_progress')}</span>
          </>
        ) : isDragging ? (
          <>
            <FolderOpenIcon className="h-6 w-6" />
            <span className="font-medium">{t('common.media.drop_files')}</span>
          </>
        ) : (
          <>
            <UploadIcon className="h-6 w-6" />
            <span>{t('common.media.drag_or')} <span className="underline">{t('common.media.browse')}</span></span>
            <span className="text-xs opacity-60">{t('common.media.supported_formats')}</span>
          </>
        )}
      </div>
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
