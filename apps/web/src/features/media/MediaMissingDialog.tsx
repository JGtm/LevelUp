/**
 * MediaMissingDialog — fenêtre « Fichiers introuvables ».
 *
 * Vue liste : les médias dont aucun fichier n'existe plus sur le disque (miniature, nom, date de
 * capture). Un clic sélectionne ; le pied propose « Retrouver dans la galerie » et « Oublier ».
 *   - Retrouver : vue des fiches présentes de même nature, sélection en deux temps, la fiche
 *     introuvable se fond dans celle choisie (marques « aimé » et match repris).
 *   - Oublier : confirmation en deux temps ; le fichier d'origine n'est jamais touché.
 *
 * La coquille, les états de liste, les lignes et les pieds de confirmation sont ceux de la
 * réassociation à un match (MediaPickerDialog).
 */
import { useState, type ReactNode } from 'react'
import type { MediaMergeCandidate, MediaMissingItem } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { useDeploymentCapability } from '@/stores/useDeploymentCapability'
import { getMediaModalsText } from './i18n-modals'
import {
  useForgetMissingMedia,
  useMergeCandidates,
  useMergeMissingMedia,
  useMissingMedia,
  useRevealMedia,
} from './missingQueries'
import { findMissingItem, formatCaptureDelta, formatCaptureTime } from './mediaMissing.logic'
import {
  MediaPickerConfirmBar,
  MediaPickerDialog,
  MediaPickerListStatus,
  MediaPickerRow,
} from './MediaPickerDialog'

interface Props {
  playerSlug: string
  onClose: () => void
}

function Thumbnail({ src }: { src: string | null | undefined }) {
  return src ? (
    <img src={src} alt="" className="h-16 w-24 shrink-0 rounded object-cover" loading="lazy" />
  ) : (
    <div className="h-16 w-24 shrink-0 rounded bg-muted" />
  )
}

interface RowTextProps {
  name: string
  captureLabel: string
  deltaLabel?: string
}

function RowText({ name, captureLabel, deltaLabel }: RowTextProps) {
  return (
    <div className="flex min-w-0 flex-1 flex-col gap-1">
      <span className="truncate text-sm font-medium">{name}</span>
      <div className="flex items-center gap-3 text-3xs text-muted-foreground">
        <span>{captureLabel}</span>
        {deltaLabel && <span className="opacity-60">{deltaLabel}</span>}
      </div>
    </div>
  )
}

interface SelectionFooterProps {
  findLabel: string
  forgetLabel: string
  onFind: () => void
  onForget: () => void
}

// Pied de la vue liste quand une ligne est sélectionnée : mêmes classes que MediaPickerConfirmBar.
function SelectionFooter({ findLabel, forgetLabel, onFind, onForget }: SelectionFooterProps) {
  return (
    <footer className="flex items-center justify-end gap-2 border-t border-border bg-muted/40 px-5 py-3 text-sm">
      <button
        type="button"
        onClick={onForget}
        className="rounded border border-border px-3 py-1.5 text-xs hover:bg-accent"
      >
        {forgetLabel}
      </button>
      <button
        type="button"
        onClick={onFind}
        className="rounded bg-primary px-3 py-1.5 text-xs font-semibold text-primary-foreground hover:opacity-90"
      >
        {findLabel}
      </button>
    </footer>
  )
}

interface CandidatesViewProps {
  playerSlug: string
  item: MediaMissingItem
  onBack: () => void
  onMerged: () => void
  onClose: () => void
}

// Vue des fiches présentes dans lesquelles fondre la fiche introuvable `item`.
function CandidatesView({ playerSlug, item, onBack, onMerged, onClose }: CandidatesViewProps) {
  const locale = useAppShellStore((s) => s.locale)
  const mp = getMediaModalsText(locale).missing
  const [targetPath, setTargetPath] = useState<string | null>(null)
  const candidates = useMergeCandidates(playerSlug, item.file_path)
  const candidateList: MediaMergeCandidate[] = candidates.data?.candidates ?? []
  const target = candidateList.find((c) => c.file_path === targetPath) ?? null
  const merge = useMergeMissingMedia(playerSlug)

  function handleMerge() {
    if (!target) return
    merge.mutate({ file_path: item.file_path, target_file_path: target.file_path }, { onSuccess: onMerged })
  }

  return (
    <MediaPickerDialog
      closeLabel={mp.closeAriaLabel}
      onClose={onClose}
      title={mp.candidatesTitle}
      subtitle={`${item.basename} · ${mp.capturePrefix} ${formatCaptureTime(item.capture_end_utc, locale)}`}
      toolbar={
        <button
          type="button"
          onClick={onBack}
          className="rounded px-2 py-1 transition-colors hover:bg-accent"
        >
          <span aria-hidden="true">←</span> {mp.back}
        </button>
      }
      footer={
        target && (
          <MediaPickerConfirmBar
            title={mp.confirmMergeTitle}
            detail={mp.confirmMergeDetail}
            cancelLabel={mp.cancel}
            confirmLabel={mp.confirm}
            busyLabel={mp.applying}
            busy={merge.isPending}
            onCancel={() => setTargetPath(null)}
            onConfirm={handleMerge}
          />
        )
      }
    >
      <MediaPickerListStatus
        isLoading={candidates.isLoading}
        isError={candidates.isError}
        isEmpty={candidateList.length === 0}
        loadingLabel={mp.loading}
        errorLabel={mp.error}
        emptyLabel={mp.candidatesEmpty}
      />
      <ul className="flex flex-col gap-2">
        {candidateList.map((c) => (
          <MediaPickerRow
            key={c.file_path}
            selected={c.file_path === targetPath}
            current={false}
            onSelect={() => setTargetPath(c.file_path === targetPath ? null : c.file_path)}
            leading={<Thumbnail src={c.thumbnail_path} />}
          >
            <RowText
              name={c.basename}
              captureLabel={formatCaptureTime(c.capture_end_utc, locale)}
              deltaLabel={formatCaptureDelta(c.delta_seconds, mp)}
            />
          </MediaPickerRow>
        ))}
      </ul>
    </MediaPickerDialog>
  )
}

interface ListFooterProps {
  playerSlug: string
  selected: MediaMissingItem | null
  forgetAsked: boolean
  onFind: () => void
  onAskForget: () => void
  onCancelForget: () => void
  onForgotten: () => void
}

// Pied de la vue liste : rien sans sélection, actions avec une sélection, confirmation d'oubli.
function ListFooter({
  playerSlug,
  selected,
  forgetAsked,
  onFind,
  onAskForget,
  onCancelForget,
  onForgotten,
}: ListFooterProps): ReactNode {
  const locale = useAppShellStore((s) => s.locale)
  const mp = getMediaModalsText(locale).missing
  const forget = useForgetMissingMedia(playerSlug)
  if (!selected) return null
  if (!forgetAsked) {
    return <SelectionFooter findLabel={mp.findMedia} forgetLabel={mp.forget} onFind={onFind} onForget={onAskForget} />
  }
  return (
    <MediaPickerConfirmBar
      title={mp.confirmForgetTitle}
      detail={mp.confirmForgetDetail}
      cancelLabel={mp.cancel}
      confirmLabel={mp.confirm}
      busyLabel={mp.applying}
      busy={forget.isPending}
      onCancel={onCancelForget}
      onConfirm={() => forget.mutate({ file_path: selected.file_path }, { onSuccess: onForgotten })}
    />
  )
}

interface RevealButtonProps {
  playerSlug: string
  filePath: string
}

// Ouvre l'explorateur sur le fichier ; n'existe que si le déploiement l'offre.
function RevealButton({ playerSlug, filePath }: RevealButtonProps) {
  const locale = useAppShellStore((s) => s.locale)
  const mp = getMediaModalsText(locale).missing
  const reveal = useRevealMedia(playerSlug)
  return (
    <button
      type="button"
      title={mp.revealTitle}
      disabled={reveal.isPending}
      onClick={() => reveal.mutate({ file_path: filePath })}
      className="rounded border border-border bg-background px-2 py-0.5 text-xs text-foreground/80 hover:border-foreground/50 hover:text-foreground disabled:opacity-50"
    >
      {mp.revealButton}
    </button>
  )
}

export function MediaMissingDialog({ playerSlug, onClose }: Props) {
  const locale = useAppShellStore((s) => s.locale)
  const mp = getMediaModalsText(locale).missing
  const canReveal = useDeploymentCapability('can_reveal_files_in_explorer')
  const [selectedPath, setSelectedPath] = useState<string | null>(null)
  const [findingPath, setFindingPath] = useState<string | null>(null)
  const [forgetAsked, setForgetAsked] = useState(false)

  const missing = useMissingMedia(playerSlug)
  const items: MediaMissingItem[] = missing.data?.items ?? []
  const selected = findMissingItem(items, selectedPath)
  // Une fiche qui quitte la liste (fusion, oubli) ramène à la vue liste.
  const finding = findMissingItem(items, findingPath)

  if (finding) {
    return (
      <CandidatesView
        playerSlug={playerSlug}
        item={finding}
        onBack={() => setFindingPath(null)}
        onMerged={() => {
          setFindingPath(null)
          setSelectedPath(null)
        }}
        onClose={onClose}
      />
    )
  }

  return (
    <MediaPickerDialog
      closeLabel={mp.closeAriaLabel}
      onClose={onClose}
      title={mp.title}
      footer={
        <ListFooter
          playerSlug={playerSlug}
          selected={selected}
          forgetAsked={forgetAsked}
          onFind={() => setFindingPath(selected?.file_path ?? null)}
          onAskForget={() => setForgetAsked(true)}
          onCancelForget={() => setForgetAsked(false)}
          onForgotten={() => {
            setForgetAsked(false)
            setSelectedPath(null)
          }}
        />
      }
    >
      <MediaPickerListStatus
        isLoading={missing.isLoading}
        isError={missing.isError}
        isEmpty={items.length === 0}
        loadingLabel={mp.loading}
        errorLabel={mp.error}
        emptyLabel={mp.empty}
      />
      <ul className="flex flex-col gap-2">
        {items.map((item) => (
          <MediaPickerRow
            key={item.file_path}
            selected={item.file_path === selectedPath}
            current={false}
            onSelect={() => {
              setForgetAsked(false)
              setSelectedPath(item.file_path === selectedPath ? null : item.file_path)
            }}
            leading={<Thumbnail src={item.thumbnail_path} />}
            action={canReveal && <RevealButton playerSlug={playerSlug} filePath={item.file_path} />}
          >
            <RowText name={item.basename} captureLabel={formatCaptureTime(item.capture_end_utc, locale)} />
          </MediaPickerRow>
        ))}
      </ul>
    </MediaPickerDialog>
  )
}
