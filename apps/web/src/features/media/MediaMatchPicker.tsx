/**
 * MediaMatchPicker — Modal pour réassocier manuellement un média à un match.
 *
 * Affiche les matchs candidats du joueur dans une fenêtre temporelle (±15 min
 * par défaut, extensible à ±60 puis ±180). Chaque candidat montre map/mode/
 * heure/K-D-A pour aider à reconnaître le bon match.
 */
import { useState } from 'react'
import { useMediaMatchCandidates, useAssociateMediaToMatch } from './queries'
import type { MediaMatchCandidate } from '@/lib/api/types'

const WINDOW_OPTIONS = [15, 60, 180] as const

interface Props {
  playerSlug: string
  filePath: string
  onClose: () => void
}

function formatLocalTime(iso: string | null | undefined): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleString('fr-FR', {
    day: '2-digit',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatDelta(deltaSeconds: number | null | undefined): string {
  if (deltaSeconds == null) return ''
  const m = Math.round(Math.abs(deltaSeconds) / 60)
  if (m === 0) return '< 1 min'
  return `±${m} min`
}

function outcomeBadge(outcome: number | null | undefined): string {
  switch (outcome) {
    case 2: return 'V'
    case 3: return 'D'
    case 1: return '='
    case 4: return 'X'
    default: return '?'
  }
}

export function MediaMatchPicker({ playerSlug, filePath, onClose }: Props) {
  const [windowMinutes, setWindowMinutes] = useState<number>(15)
  const { data, isLoading, isError } = useMediaMatchCandidates(playerSlug, filePath, windowMinutes)
  const associate = useAssociateMediaToMatch(playerSlug)

  function handlePick(candidate: MediaMatchCandidate) {
    associate.mutate(
      { file_path: filePath, match_id: candidate.match_id },
      { onSuccess: () => onClose() },
    )
  }

  const candidates = data?.candidates ?? []

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/70"
      onClick={onClose}
    >
      <div
        className="mx-4 flex max-h-[85vh] w-full max-w-2xl flex-col rounded-lg bg-background shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <header className="flex items-center justify-between border-b border-border px-5 py-3">
          <div>
            <h2 className="text-base font-semibold">Réassocier ce média</h2>
            {data?.capture_utc && (
              <p className="text-xs text-muted-foreground">
                Capture : {formatLocalTime(data.capture_utc)}
              </p>
            )}
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded p-1 text-sm text-muted-foreground hover:bg-accent"
            aria-label="Fermer"
          >
            ✕
          </button>
        </header>

        <div className="flex items-center gap-2 border-b border-border px-5 py-2 text-xs">
          <span className="text-muted-foreground">Fenêtre de recherche :</span>
          {WINDOW_OPTIONS.map((w) => (
            <button
              key={w}
              type="button"
              onClick={() => setWindowMinutes(w)}
              className={`rounded px-2 py-1 transition-colors ${
                windowMinutes === w
                  ? 'bg-primary text-primary-foreground'
                  : 'hover:bg-accent'
              }`}
            >
              ±{w} min
            </button>
          ))}
          {data && <span className="ml-auto text-muted-foreground">{candidates.length} match(s) trouvé(s)</span>}
        </div>

        <div className="flex-1 overflow-y-auto px-2 py-2">
          {isLoading && <p className="p-4 text-center text-sm text-muted-foreground">Chargement…</p>}
          {isError && <p className="p-4 text-center text-sm text-destructive">Erreur de chargement</p>}
          {!isLoading && !isError && candidates.length === 0 && (
            <p className="p-4 text-center text-sm text-muted-foreground">
              Aucun match trouvé dans cette fenêtre. Élargis la recherche.
            </p>
          )}
          <ul className="flex flex-col gap-1">
            {candidates.map((c) => {
              const isCurrent = c.is_current
              return (
                <li key={c.match_id}>
                  <button
                    type="button"
                    disabled={associate.isPending || isCurrent}
                    onClick={() => handlePick(c)}
                    className={`flex w-full flex-col gap-1 rounded-md border px-3 py-2 text-left transition-colors ${
                      isCurrent
                        ? 'border-primary/60 bg-primary/5 cursor-default'
                        : 'border-border hover:border-primary/50 hover:bg-accent'
                    }`}
                  >
                    <div className="flex items-center justify-between text-sm">
                      <span className="font-medium">
                        {c.map_name ?? 'Carte ?'} · {c.mode_name ?? 'Mode ?'}
                      </span>
                      {isCurrent && (
                        <span className="rounded bg-primary px-1.5 py-0.5 text-[10px] font-semibold text-primary-foreground">
                          actuel
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-3 text-xs text-muted-foreground">
                      <span>{formatLocalTime(c.start_time)}</span>
                      <span className="opacity-60">{formatDelta(c.delta_seconds)}</span>
                      <span className="ml-auto tabular-nums">
                        {c.kills ?? '?'} / {c.deaths ?? '?'} / {c.assists ?? '?'}
                        <span className="ml-1 opacity-70">({outcomeBadge(c.outcome)})</span>
                      </span>
                    </div>
                    {c.playlist_name && (
                      <div className="text-[11px] text-muted-foreground opacity-70">{c.playlist_name}</div>
                    )}
                  </button>
                </li>
              )
            })}
          </ul>
        </div>

        {associate.isError && (
          <p className="border-t border-border px-5 py-2 text-xs text-destructive">
            Erreur lors de la réassociation : {associate.error instanceof Error ? associate.error.message : 'inconnue'}
          </p>
        )}
      </div>
    </div>
  )
}
