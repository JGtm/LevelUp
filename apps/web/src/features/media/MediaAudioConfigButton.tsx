/**
 * MediaAudioConfigButton — bouton engrenage de la barre d'outils médias ouvrant la
 * modale de réglage des pistes audio (voix / jeu / autres) : mode automatique (l'analyse
 * NNLS décide) ou manuel (le joueur déclare l'ordre de ses pistes source).
 *
 * Auto-contenu (query + mutation + i18n) — rendu dans MediaToolbar.
 */
import { useEffect, useState } from 'react'
import { Select } from '@/components/ui/select'
import { useAppShellStore } from '@/stores/appShellStore'
import { apiErrorMessage } from '@/lib/api/client'
import { getMediaAudioConfigText, type MediaAudioConfigText } from './i18n-audio-config'
import {
  useMediaAudioConfig,
  useUpdateMediaAudioConfig,
  type AudioTrackRole,
  type MediaAudioMode,
} from './queries'

const MAX_TRACKS = 16
const ROLE_VALUES: AudioTrackRole[] = ['game', 'voice', 'other']
// Référence STABLE pour « aucune piste » : le seed-effect depend de l'identité de
// `data` ; un `?? []` inline créerait un tableau neuf à chaque passage et empêcherait
// le bail-out React (setState même valeur) si le caller fournit un `data` instable.
const EMPTY_ROLES: AudioTrackRole[] = []

interface MediaAudioConfigButtonProps {
  playerSlug: string
}

export function MediaAudioConfigButton({ playerSlug }: MediaAudioConfigButtonProps) {
  const [open, setOpen] = useState(false)
  const locale = useAppShellStore((s) => s.locale)
  const text = getMediaAudioConfigText(locale)

  return (
    <>
      <button
        type="button"
        aria-label={text.gearAriaLabel}
        title={text.gearAriaLabel}
        onClick={() => setOpen(true)}
        className="flex h-8 items-center justify-center rounded-md border border-input px-2 text-base leading-none text-muted-foreground transition-colors hover:text-foreground"
      >
        <span aria-hidden="true">⚙</span>
      </button>
      {open && (
        <MediaAudioConfigModal playerSlug={playerSlug} text={text} onClose={() => setOpen(false)} />
      )}
    </>
  )
}

interface ModalProps {
  playerSlug: string
  text: MediaAudioConfigText
  onClose: () => void
}

function MediaAudioConfigModal({ playerSlug, text, onClose }: ModalProps) {
  const { data } = useMediaAudioConfig(playerSlug)
  const update = useUpdateMediaAudioConfig(playerSlug)
  const [mode, setMode] = useState<MediaAudioMode>('auto')
  const [roles, setRoles] = useState<AudioTrackRole[]>([])

  // Seed depuis le réglage chargé (dès qu'il arrive). setState idempotent : valeurs
  // stables (EMPTY_ROLES) pour garantir le bail-out React si `data` change d'identité
  // sans changer de contenu.
  useEffect(() => {
    if (!data) return
    setMode(data.mode ?? 'auto')
    setRoles(data.track_roles && data.track_roles.length > 0 ? data.track_roles : EMPTY_ROLES)
  }, [data])

  // Escape ferme la modale.
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose])

  function selectManual() {
    setMode('manual')
    if (roles.length === 0) setRoles(['game', 'voice'])
  }

  const manualInvalid = mode === 'manual' && roles.length === 0
  function handleSave() {
    if (manualInvalid) return
    update.mutate(
      { mode, track_roles: mode === 'manual' ? roles : undefined },
      { onSuccess: onClose },
    )
  }

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-background/70"
      onClick={onClose}
    >
      <div
        className="mx-4 flex max-h-[85vh] w-full max-w-md flex-col rounded-lg bg-background shadow-xl"
        role="dialog"
        aria-modal="true"
        aria-label={text.title}
        onClick={(e) => e.stopPropagation()}
      >
        <header className="flex items-center justify-between border-b border-border px-5 py-3">
          <h2 className="text-base font-semibold">{text.title}</h2>
          <button
            type="button"
            onClick={onClose}
            aria-label={text.closeAriaLabel}
            className="rounded p-1 text-sm text-muted-foreground hover:bg-accent"
          >
            ✕
          </button>
        </header>

        <div className="flex flex-col gap-4 overflow-y-auto px-5 py-4">
          <p className="text-xs text-muted-foreground">{text.intro}</p>

          <ModeRadio
            checked={mode === 'auto'}
            onSelect={() => setMode('auto')}
            label={text.modeAuto}
            hint={text.modeAutoHint}
          />
          <ModeRadio
            checked={mode === 'manual'}
            onSelect={selectManual}
            label={text.modeManual}
            hint={text.modeManualHint}
          />

          {mode === 'manual' && (
            <ManualTracksEditor text={text} roles={roles} onChange={setRoles} />
          )}

          {update.isError && (
            <p className="text-xs text-destructive">
              {text.saveError} — {apiErrorMessage(update.error)}
            </p>
          )}
        </div>

        <footer className="flex items-center justify-end gap-2 border-t border-border px-5 py-3">
          <button
            type="button"
            onClick={onClose}
            className="rounded-md border border-input px-3 py-1.5 text-sm text-foreground hover:bg-accent"
          >
            {text.cancel}
          </button>
          <button
            type="button"
            onClick={handleSave}
            disabled={manualInvalid || update.isPending}
            className="rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground disabled:opacity-40"
          >
            {update.isPending ? text.saving : text.save}
          </button>
        </footer>
      </div>
    </div>
  )
}

interface ModeRadioProps {
  checked: boolean
  onSelect: () => void
  label: string
  hint: string
}

function ModeRadio({ checked, onSelect, label, hint }: ModeRadioProps) {
  return (
    <label className="flex cursor-pointer items-start gap-2 rounded-md border border-input p-2 hover:bg-accent">
      <input
        type="radio"
        name="media-audio-mode"
        checked={checked}
        onChange={onSelect}
        className="mt-0.5"
      />
      <span className="flex flex-col">
        <span className="text-sm font-medium text-foreground">{label}</span>
        <span className="text-xs text-muted-foreground">{hint}</span>
      </span>
    </label>
  )
}

interface EditorProps {
  text: MediaAudioConfigText
  roles: AudioTrackRole[]
  onChange: (roles: AudioTrackRole[]) => void
}

function ManualTracksEditor({ text, roles, onChange }: EditorProps) {
  function setRole(idx: number, role: AudioTrackRole) {
    onChange(roles.map((r, i) => (i === idx ? role : r)))
  }
  function addTrack() {
    if (roles.length >= MAX_TRACKS) return
    onChange([...roles, 'other'])
  }
  function removeTrack(idx: number) {
    onChange(roles.filter((_, i) => i !== idx))
  }
  const roleLabel: Record<AudioTrackRole, string> = {
    game: text.roleGame,
    voice: text.roleVoice,
    other: text.roleOther,
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col">
        <span className="text-xs font-semibold text-muted-foreground">{text.tracksLabel}</span>
        <span className="text-2xs text-muted-foreground">{text.tracksHint}</span>
      </div>
      {roles.length === 0 && (
        <p className="text-xs text-destructive">{text.emptyManual}</p>
      )}
      {roles.map((role, idx) => (
        <div key={idx} className="flex items-center gap-2">
          <span className="w-16 shrink-0 text-xs text-muted-foreground">{text.trackLabel(idx + 1)}</span>
          <Select
            aria-label={text.trackLabel(idx + 1)}
            className="h-8 w-auto flex-1 px-2 pr-6 text-xs"
            value={role}
            onChange={(e) => setRole(idx, e.target.value as AudioTrackRole)}
          >
            {ROLE_VALUES.map((value) => (
              <option key={value} value={value}>
                {roleLabel[value]}
              </option>
            ))}
          </Select>
          <button
            type="button"
            onClick={() => removeTrack(idx)}
            aria-label={text.removeTrack}
            title={text.removeTrack}
            className="rounded px-2 py-1 text-xs text-muted-foreground hover:bg-accent hover:text-foreground"
          >
            ✕
          </button>
        </div>
      ))}
      <button
        type="button"
        onClick={addTrack}
        disabled={roles.length >= MAX_TRACKS}
        className="self-start rounded-md border border-input px-2 py-1 text-xs text-foreground hover:bg-accent disabled:opacity-40"
      >
        + {text.addTrack}
      </button>
    </div>
  )
}
