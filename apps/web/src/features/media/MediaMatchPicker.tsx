/**
 * MediaMatchPicker — Modal de réassociation manuelle d'un média à un match.
 *
 * Liste les matchs du joueur dans une fenêtre temporelle (±15 min par défaut,
 * extensible à ±60 puis ±180). Chaque candidat affiche :
 *   - miniature de la map (gauche)
 *   - map · mode (normalisés FR) · playlist
 *   - heure locale + delta vs capture
 *   - résultat (V/D/=/X)
 *   - lobby complet par équipe
 *
 * Sélection en 2 étapes :
 *   1er click  → met le candidat en surbrillance + bouton "Confirmer" en bas
 *   confirmer  → POST /associate, invalide le cache, ferme la modal
 */
import { useState } from 'react'
import { useMediaMatchCandidates, useAssociateMediaToMatch } from './queries'
import type { MediaMatchCandidate } from '@/lib/api/types'
import { useFieldMappings, useAssetLabel } from '@/lib/i18n/fieldMappings'
import { OUTCOME_LABELS_FALLBACK_FR } from './fallback.i18n'

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

// outcomeKeyOf mappe le code outcome Halo (2 win, 3 loss, 1 tie, 4 dnf)
// vers la clé canonique correspondante côté outcomes.toml.
function outcomeKeyOf(outcome: number | null | undefined): string | null {
  switch (outcome) {
    case 2:
      return 'win'
    case 3:
      return 'loss'
    case 1:
      return 'tie'
    case 4:
      return 'dnf'
    default:
      return null
  }
}

const outcomeClassByCode: Record<number, string> = {
  2: 'bg-success/15 text-success border-success/40',
  3: 'bg-destructive/15 text-destructive border-destructive/40',
  1: 'bg-warning/15 text-warning border-warning/40',
  4: 'bg-muted text-muted-foreground border-border',
}

// Localise map_name et mode_name via useAssetLabel ('map'/'mode' kinds dans assets.toml).
// Fallback sur la valeur brute si l'asset n'est pas défini côté backend.
function MapModeLabel({ mapName, modeName }: { mapName: string | null | undefined; modeName: string | null | undefined }) {
  const localizedMap = useAssetLabel('map', mapName ?? '')
  const localizedMode = useAssetLabel('mode', modeName ?? '')
  const mapStr = mapName ? localizedMap : '?'
  const modeStr = modeName ? localizedMode : '?'
  return <>{mapStr} · {modeStr}</>
}

function LobbyTeams({ lobby }: { lobby: MediaMatchCandidate['lobby'] }) {
  if (!lobby || lobby.length === 0) {
    return <p className="text-[11px] italic text-muted-foreground">Lobby indisponible</p>
  }
  const teams = new Map<string, { teamID: number | null; players: typeof lobby }>()
  for (const p of lobby) {
    const key = p.team_id == null ? 'null' : String(p.team_id)
    const existing = teams.get(key)
    if (existing) existing.players.push(p)
    else teams.set(key, { teamID: p.team_id ?? null, players: [p] })
  }
  return (
    <div className="flex flex-wrap gap-x-4 gap-y-1 text-[11px]">
      {Array.from(teams.values()).map((team, idx) => (
        <div key={idx} className="flex flex-col">
          <span className="font-semibold text-muted-foreground">
            {team.teamID == null ? 'Spectateurs' : `Équipe ${team.teamID + 1}`}
          </span>
          <ul>
            {team.players.map((p) => (
              <li
                key={p.gamertag}
                className={p.is_self ? 'font-semibold text-primary' : 'text-foreground/85'}
              >
                {p.gamertag}{p.is_self && ' (toi)'}
              </li>
            ))}
          </ul>
        </div>
      ))}
    </div>
  )
}

export function MediaMatchPicker({ playerSlug, filePath, onClose }: Props) {
  const [windowMinutes, setWindowMinutes] = useState<number>(15)
  const [pendingMatchID, setPendingMatchID] = useState<string | null>(null)
  const { data, isLoading, isError } = useMediaMatchCandidates(playerSlug, filePath, windowMinutes)
  const associate = useAssociateMediaToMatch(playerSlug)
  // Phase 4 plan finition multi-titres : libellés outcomes via TOML, fallback FR.
  const { data: fieldMappings } = useFieldMappings()
  const outcomeLabel = (outcome: number | null | undefined): { text: string; cls: string } => {
    if (outcome == null || !(outcome in outcomeClassByCode)) {
      return { text: '—', cls: 'bg-muted text-muted-foreground border-border' }
    }
    const cls = outcomeClassByCode[outcome]
    const key = outcomeKeyOf(outcome)
    const text =
      (key && fieldMappings?.outcomes?.[key]?.label) ?? OUTCOME_LABELS_FALLBACK_FR[outcome] ?? '—'
    return { text, cls }
  }

  const candidates = data?.candidates ?? []
  const pending = candidates.find((c) => c.match_id === pendingMatchID) ?? null

  function handleConfirm() {
    if (!pending) return
    associate.mutate(
      { file_path: filePath, match_id: pending.match_id },
      { onSuccess: () => onClose() },
    )
  }

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-background/70"
      onClick={onClose}
    >
      <div
        className="mx-4 flex max-h-[85vh] w-full max-w-3xl flex-col rounded-lg bg-background shadow-xl"
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
          <span className="text-muted-foreground">Fenêtre :</span>
          {WINDOW_OPTIONS.map((w) => (
            <button
              key={w}
              type="button"
              onClick={() => { setWindowMinutes(w); setPendingMatchID(null) }}
              className={`rounded px-2 py-1 transition-colors ${
                windowMinutes === w
                  ? 'bg-primary text-primary-foreground'
                  : 'hover:bg-accent'
              }`}
            >
              ±{w} min
            </button>
          ))}
          {data && (
            <span className="ml-auto text-muted-foreground">
              {candidates.length} match(s) trouvé(s)
            </span>
          )}
        </div>

        <div className="flex-1 overflow-y-auto px-3 py-2">
          {isLoading && <p className="p-4 text-center text-sm text-muted-foreground">Chargement…</p>}
          {isError && <p className="p-4 text-center text-sm text-destructive">Erreur de chargement</p>}
          {!isLoading && !isError && candidates.length === 0 && (
            <p className="p-4 text-center text-sm text-muted-foreground">
              Aucun match trouvé dans cette fenêtre. Élargis la recherche.
            </p>
          )}
          <ul className="flex flex-col gap-2">
            {candidates.map((c) => {
              const isCurrent = c.is_current
              const isPending = c.match_id === pendingMatchID
              const out = outcomeLabel(c.outcome)
              return (
                <li key={c.match_id}>
                  <button
                    type="button"
                    disabled={isCurrent}
                    onClick={() => setPendingMatchID(isPending ? null : c.match_id)}
                    className={`flex w-full gap-3 rounded-md border px-3 py-2 text-left transition-colors ${
                      isCurrent
                        ? 'border-primary/60 bg-primary/5 cursor-default'
                        : isPending
                        ? 'border-primary/80 bg-primary/10 ring-2 ring-primary/40'
                        : 'border-border hover:border-primary/50 hover:bg-accent'
                    }`}
                  >
                    {c.map_image_url ? (
                      <img
                        src={c.map_image_url}
                        alt={c.map_name ?? 'Map'}
                        className="h-16 w-24 shrink-0 rounded object-cover"
                        loading="lazy"
                      />
                    ) : (
                      <div className="h-16 w-24 shrink-0 rounded bg-muted" />
                    )}
                    <div className="flex min-w-0 flex-1 flex-col gap-1">
                      <div className="flex items-center justify-between gap-2 text-sm">
                        <span className="truncate font-medium">
                          <MapModeLabel mapName={c.map_name} modeName={c.mode_name} />
                        </span>
                        <div className="flex shrink-0 items-center gap-2">
                          <span className={`rounded border px-1.5 py-0.5 text-[10px] font-semibold ${out.cls}`}>
                            {out.text}
                          </span>
                          {isCurrent && (
                            <span className="rounded bg-primary px-1.5 py-0.5 text-[10px] font-semibold text-primary-foreground">
                              actuel
                            </span>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-3 text-[11px] text-muted-foreground">
                        <span>{formatLocalTime(c.start_time)}</span>
                        <span className="opacity-60">{formatDelta(c.delta_seconds)}</span>
                        {c.playlist_name && <span className="ml-auto truncate">{c.playlist_name}</span>}
                      </div>
                      <LobbyTeams lobby={c.lobby} />
                    </div>
                  </button>
                </li>
              )
            })}
          </ul>
        </div>

        {pending && (
          <footer className="flex items-center justify-between gap-3 border-t border-border bg-muted/40 px-5 py-3 text-sm">
            <div>
              <p className="font-medium">Confirmer la réassociation ?</p>
              <p className="text-xs text-muted-foreground">
                <MapModeLabel mapName={pending.map_name} modeName={pending.mode_name} /> · {formatLocalTime(pending.start_time)}
              </p>
            </div>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setPendingMatchID(null)}
                className="rounded border border-border px-3 py-1.5 text-xs hover:bg-accent"
              >
                Annuler
              </button>
              <button
                type="button"
                disabled={associate.isPending}
                onClick={handleConfirm}
                className="rounded bg-primary px-3 py-1.5 text-xs font-semibold text-primary-foreground hover:opacity-90 disabled:opacity-50"
              >
                {associate.isPending ? 'Application…' : 'Confirmer'}
              </button>
            </div>
          </footer>
        )}

        {associate.isError && (
          <p className="border-t border-border px-5 py-2 text-xs text-destructive">
            Erreur lors de la réassociation : {associate.error instanceof Error ? associate.error.message : 'inconnue'}
          </p>
        )}
      </div>
    </div>
  )
}
