/**
 * StartCampaignModal — mini-modale "Démarrer une campagne sur cet axe".
 *
 * Cf. PLAN_PLAYER_PROFILE_ASCENSION.md §5.2 (CTA Section C).
 *
 * Le caller passe l'axe + axisKind pré-sélectionnés (depuis le clic sur un
 * levier). L'utilisateur choisit le playlist_group cible (défaut "all") et
 * voit la phrase pédagogique R2 + une option "Skip — créer juste un défi".
 *
 * Pattern aligné sur AlertDialog : pas de Radix, ARIA `role="dialog"`,
 * fermeture Escape + clic backdrop.
 */
import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import { apiErrorMessage } from '@/lib/api/client'
import type { AxisKind } from '@/lib/playerProfile'
import { useCampaignMutations } from '../hooks/usePlayerProfile'

const AXIS_LABELS_FR: Record<string, string> = {
  combat: 'Combat',
  survival: 'Survie',
  support: 'Support',
  score: 'Score',
  objective: 'Objectif',
  impact: 'Impact',
  kills_vs_expected: 'Kills vs attendus',
  deaths_vs_expected: 'Morts vs attendues',
  win_factor: 'Facteur de victoire',
  damage_efficiency: 'Efficacité dégâts',
  accuracy_delta: 'Précision (delta)',
  medal_exploit: 'Exploits / médailles',
  offensive_conversion: 'Conversion offensive',
  defensive_resistance: 'Résistance défensive',
}

const PLAYLIST_OPTIONS = [
  { value: 'all', label: 'Toutes playlists' },
  { value: 'arena_slayer', label: 'Arena Slayer' },
  { value: 'ranked', label: 'Ranked' },
  { value: 'btb', label: 'Big Team Battle' },
  { value: 'social', label: 'Social' },
  { value: 'fun', label: 'Fun' },
] as const

interface StartCampaignModalProps {
  open: boolean
  playerSlug: string
  axis: string
  axisKind: AxisKind
  onOpenChange: (open: boolean) => void
  /** Callback "Skip — créer juste un défi libre" : remonte vers le flow CreateChallengeForm. */
  onSkipToFreeChallenge?: () => void
}

export function StartCampaignModal({
  open,
  playerSlug,
  axis,
  axisKind,
  onOpenChange,
  onSkipToFreeChallenge,
}: StartCampaignModalProps) {
  const [playlistGroup, setPlaylistGroup] = useState<string>('all')
  const muts = useCampaignMutations(playerSlug)
  const startError = apiErrorMessage(muts.start.error)

  useEffect(() => {
    if (open) {
      setPlaylistGroup('all')
      muts.start.reset()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  useEffect(() => {
    if (!open) return
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape' && !muts.start.isPending) {
        event.preventDefault()
        onOpenChange(false)
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [open, muts.start.isPending, onOpenChange])

  if (!open) return null

  const axisLabel = AXIS_LABELS_FR[axis] ?? axis

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="start-campaign-title"
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
    >
      <div
        aria-hidden="true"
        onClick={() => !muts.start.isPending && onOpenChange(false)}
        className={
          'absolute inset-0 bg-black/60 backdrop-blur-sm' +
          (muts.start.isPending ? ' cursor-wait' : ' cursor-pointer')
        }
      />
      <div className="relative z-10 w-full max-w-md rounded-lg border border-border bg-card text-card-foreground shadow-xl">
        <div className="px-6 py-5">
          <h2
            id="start-campaign-title"
            className="text-lg font-semibold tracking-tight"
          >
            Démarrer une campagne sur {axisLabel}
          </h2>
          <p className="mt-2 text-sm text-muted-foreground">
            On t&apos;aide à voir ta trajectoire — pas à la garantir. La progression
            vient de toi.
          </p>

          <div className="mt-4 space-y-2">
            <label
              htmlFor="campaign-playlist-group"
              className="block text-xs font-semibold uppercase text-muted-foreground"
            >
              Playlist cible
            </label>
            <select
              id="campaign-playlist-group"
              value={playlistGroup}
              onChange={(e) => setPlaylistGroup(e.target.value)}
              disabled={muts.start.isPending}
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
            >
              {PLAYLIST_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
            <p className="text-xs text-muted-foreground">
              Pour mesurer la campagne uniquement sur ces matchs. Tu peux laisser
              «&nbsp;toutes playlists&nbsp;» si tu joues varié.
            </p>
          </div>

          {startError && (
            <p className="mt-3 rounded border border-destructive bg-destructive/10 p-2 text-xs text-destructive">
              {startError}
            </p>
          )}
        </div>

        <div className="flex flex-wrap items-center justify-end gap-2 border-t border-border px-6 py-4">
          {onSkipToFreeChallenge && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => {
                onOpenChange(false)
                onSkipToFreeChallenge()
              }}
              disabled={muts.start.isPending}
            >
              Skip — créer juste un défi libre
            </Button>
          )}
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => onOpenChange(false)}
            disabled={muts.start.isPending}
          >
            Annuler
          </Button>
          <Button
            type="button"
            variant="default"
            size="sm"
            loading={muts.start.isPending}
            disabled={muts.start.isPending}
            onClick={() =>
              muts.start.mutate(
                { axis, axis_kind: axisKind, playlist_group: playlistGroup },
                { onSuccess: () => onOpenChange(false) },
              )
            }
          >
            Démarrer
          </Button>
        </div>
      </div>
    </div>
  )
}
