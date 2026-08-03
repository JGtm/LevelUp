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
import { useMemo, useState } from 'react'
import { toast } from 'sonner'
import { useMediaMatchCandidates, useAssociateMediaToMatch } from './queries'
import { apiErrorMessage } from '@/lib/api/client'
import type { MediaMatchCandidate } from '@/lib/api/types'
import { useFieldMappings, useAssetLabel } from '@/lib/i18n/fieldMappings'
import { resolveTeamNameFromID } from '@/lib/halo/teamNames'
import { outcomeCodeToValue } from '@/lib/outcome'
import { tokenCssVar } from '@/lib/accessibility'
import { getMediaModalsText, type MatchPickerText } from './i18n-modals'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import type { ManifestLocale } from '@/lib/i18n/format'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

const WINDOW_OPTIONS = [15, 60, 180] as const

interface Props {
  playerSlug: string
  filePath: string
  onClose: () => void
  /** Si false (média sans match associé), les libellés passent à "Associer"
   *  / "Confirmer l'association" — sinon (par défaut), "Réassocier". */
  hasCurrentMatch?: boolean
}

function formatLocalTime(iso: string | null | undefined, locale: ManifestLocale): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleString(intlLocale(locale), {
    day: '2-digit',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatDelta(deltaSeconds: number | null | undefined, mp: MatchPickerText): string {
  if (deltaSeconds == null) return ''
  const m = Math.round(Math.abs(deltaSeconds) / 60)
  if (m === 0) return mp.deltaUnder1Min
  return mp.deltaMinFormat(m)
}

// Mapping code outcome → clé outcomes.toml : centralisé dans `@/lib/outcome`
// (`outcomeCodeToValue`, défaut null).

const outcomeClassByCode: Record<number, string> = {
  2: 'bg-success/15 text-success border-success/40',
  3: 'bg-destructive/15 text-destructive border-destructive/40',
  1: 'bg-warning/15 text-warning border-warning/40',
  4: 'bg-muted text-muted-foreground border-border',
}

// Heading d'un candidat : score X-Y (POV joueur) en tête quand dispo, puis
// map · mode localisés via useAssetLabel ('map'/'mode' kinds dans assets.toml).
// Skippe silencieusement les segments manquants au lieu d'afficher "?".
// Fallback "Match" si tout est null (match_registry mal renseigné).
function CandidateHeading({ candidate }: { candidate: MediaMatchCandidate }) {
  const localizedMap = useAssetLabel('map', candidate.map_name ?? '')
  const localizedMode = useAssetLabel('mode', candidate.mode_name ?? '')
  const parts: string[] = []
  if (candidate.own_score != null && candidate.enemy_score != null) {
    parts.push(`${candidate.own_score} - ${candidate.enemy_score}`)
  }
  if (candidate.map_name) parts.push(localizedMap)
  if (candidate.mode_name) parts.push(localizedMode)
  return <>{parts.length > 0 ? parts.join(' · ') : 'Match'}</>
}

// Libellé d'équipe : nom officiel Halo (Eagle/Cobra/…) si team_id ∈ [0..8],
// sinon fallback "Équipe N" ; null → "Spectateurs". Aligné sur MatchScoreboard.
// Bilingue via MatchPickerText (GH2-B7).
function teamHeaderLabel(teamID: number | null, mp: MatchPickerText): string {
  if (teamID == null) return mp.spectators
  const official = resolveTeamNameFromID(teamID)
  return official ? mp.teamLabel(official) : mp.teamLabel(teamID + 1)
}

function LobbyTeams({ lobby, mp }: { lobby: MediaMatchCandidate['lobby']; mp: MatchPickerText }) {
  // Groupement + détection de l'équipe du joueur — recompute uniquement si
  // l'array lobby change (référence stable côté react-query entre rerenders).
  const grouped = useMemo(() => {
    const teams = new Map<string, { teamID: number | null; players: NonNullable<typeof lobby> }>()
    let mineTeamID: number | null | undefined
    for (const p of lobby ?? []) {
      const key = p.team_id == null ? 'null' : String(p.team_id)
      const existing = teams.get(key)
      if (existing) existing.players.push(p)
      else teams.set(key, { teamID: p.team_id ?? null, players: [p] })
      if (p.is_self) mineTeamID = p.team_id ?? null
    }
    return { teams: Array.from(teams.values()), mineTeamID }
  }, [lobby])

  if (!lobby || lobby.length === 0) {
    return <p className="text-3xs italic text-muted-foreground">{mp.lobbyUnavailable}</p>
  }

  return (
    <div className="flex flex-wrap gap-x-4 gap-y-1 text-3xs">
      {grouped.teams.map((team, idx) => {
        // Couleurs sémantiques alignées sur MatchScoreboard : team-ally pour
        // l'équipe du joueur, team-enemy pour les autres équipes, foreground
        // neutre pour les spectateurs (team_id null).
        const isMine = team.teamID != null && team.teamID === grouped.mineTeamID
        const isSpectator = team.teamID == null
        const headerColor = isSpectator
          ? undefined
          : isMine
            ? tokenCssVar('team-ally')
            : tokenCssVar('team-enemy')
        return (
          <div key={idx} className="flex flex-col">
            <span
              className="font-semibold"
              style={headerColor ? { color: headerColor } : undefined}
            >
              {teamHeaderLabel(team.teamID, mp)}
            </span>
            <ul>
              {team.players.map((p) => (
                <li
                  key={p.gamertag}
                  className={p.is_self ? 'font-semibold text-primary' : 'text-foreground/85'}
                >
                  {p.gamertag}
                  {p.is_bot && (
                    <span className="ml-1 rounded bg-muted px-1 py-0 text-[9px] font-bold uppercase tracking-wide text-muted-foreground">
                      Bot
                    </span>
                  )}
                </li>
              ))}
            </ul>
          </div>
        )
      })}
    </div>
  )
}

export function MediaMatchPicker({ playerSlug, filePath, onClose, hasCurrentMatch = true }: Props) {
  const [windowMinutes, setWindowMinutes] = useState<number>(15)
  const [pendingMatchID, setPendingMatchID] = useState<string | null>(null)
  const { data, isLoading, isError } = useMediaMatchCandidates(playerSlug, filePath, windowMinutes)
  const associate = useAssociateMediaToMatch(playerSlug)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  // GH2-B7 : dictionnaire bilingue de la popup (i18n-modals.ts, enfin câblé).
  const mp = getMediaModalsText(locale).matchPicker
  // Libellés d'issue servis par outcomes.toml (source unique — plus de repli FR
  // local). Le dictionnaire brut est lu ici, au niveau du composant : la
  // résolution se fait ensuite dans une boucle, où un hook serait illégal.
  const { data: fieldMappings } = useFieldMappings()
  const outcomeLabel = (outcome: number | null | undefined): { text: string; cls: string } => {
    if (outcome == null || !(outcome in outcomeClassByCode)) {
      return { text: '—', cls: 'bg-muted text-muted-foreground border-border' }
    }
    const cls = outcomeClassByCode[outcome]
    const key = outcomeCodeToValue(outcome)
    const text = (key && fieldMappings?.outcomes?.[key]?.label) ?? '—'
    return { text, cls }
  }

  const candidates = data?.candidates ?? []
  const pending = candidates.find((c) => c.match_id === pendingMatchID) ?? null

  function handleConfirm() {
    if (!pending) return
    associate.mutate(
      { file_path: filePath, match_id: pending.match_id },
      {
        onSuccess: () => {
          toast.success(t('common.media.associate_success'))
          onClose()
        },
        onError: (err) =>
          toast.error(t('common.media.associate_error'), {
            description: apiErrorMessage(err),
          }),
      },
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
            <h2 className="text-base font-semibold">{hasCurrentMatch ? mp.title : mp.titleAssociate}</h2>
            {data?.capture_utc && (
              <p className="text-xs text-muted-foreground">
                {mp.capturePrefix} {formatLocalTime(data.capture_utc, locale)}
              </p>
            )}
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded p-1 text-sm text-muted-foreground hover:bg-accent"
            aria-label={mp.closeAriaLabel}
          >
            ✕
          </button>
        </header>

        <div className="flex items-center gap-2 border-b border-border px-5 py-2 text-xs">
          <span className="text-muted-foreground">{mp.windowLabel}</span>
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
              {mp.deltaMinFormat(w)}
            </button>
          ))}
          {data && (
            <span className="ml-auto text-muted-foreground">
              {mp.matchesFound(candidates.length)}
            </span>
          )}
        </div>

        <div className="flex-1 overflow-y-auto px-3 py-2">
          {isLoading && <p className="p-4 text-center text-sm text-muted-foreground">{mp.loading}</p>}
          {isError && <p className="p-4 text-center text-sm text-destructive">{mp.error}</p>}
          {!isLoading && !isError && candidates.length === 0 && (
            <p className="p-4 text-center text-sm text-muted-foreground">
              {mp.noMatchesFound}
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
                          <CandidateHeading candidate={c} />
                        </span>
                        <div className="flex shrink-0 items-center gap-2">
                          <span className={`rounded border px-1.5 py-0.5 text-2xs font-semibold ${out.cls}`}>
                            {out.text}
                          </span>
                          {isCurrent && (
                            <span className="rounded bg-primary px-1.5 py-0.5 text-2xs font-semibold text-primary-foreground">
                              {mp.currentBadge}
                            </span>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-3 text-3xs text-muted-foreground">
                        <span>{formatLocalTime(c.start_time, locale)}</span>
                        <span className="opacity-60">{formatDelta(c.delta_seconds, mp)}</span>
                        {c.playlist_name && <span className="ml-auto truncate">{c.playlist_name}</span>}
                      </div>
                      <LobbyTeams lobby={c.lobby} mp={mp} />
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
              <p className="font-medium">{hasCurrentMatch ? mp.confirmTitle : mp.confirmTitleAssociate}</p>
              <p className="text-xs text-muted-foreground">
                <CandidateHeading candidate={pending} /> · {formatLocalTime(pending.start_time, locale)}
              </p>
            </div>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setPendingMatchID(null)}
                className="rounded border border-border px-3 py-1.5 text-xs hover:bg-accent"
              >
                {mp.cancel}
              </button>
              <button
                type="button"
                disabled={associate.isPending}
                onClick={handleConfirm}
                className="rounded bg-primary px-3 py-1.5 text-xs font-semibold text-primary-foreground hover:opacity-90 disabled:opacity-50"
              >
                {associate.isPending ? mp.applying : mp.confirm}
              </button>
            </div>
          </footer>
        )}
      </div>
    </div>
  )
}
