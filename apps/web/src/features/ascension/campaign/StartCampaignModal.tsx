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
import type { ProfileManifestKey } from '@/lib/i18n/generated/profile'
import type { AxisKind } from '@/lib/playerProfile'
import { useCampaignMutations } from '@/features/ascension/profile/queries'
import { useProfileI18n } from '@/features/ascension/profile/useProfileI18n'

const PLAYLIST_VALUES = ['all', 'arena_slayer', 'ranked', 'btb', 'social', 'fun'] as const

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
  const { t } = useProfileI18n()
  const [playlistGroup, setPlaylistGroup] = useState<string>('all')
  const muts = useCampaignMutations(playerSlug)
  const startError = apiErrorMessage(muts.start.error)

  useEffect(() => {
    if (open) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- reset du formulaire à l'ouverture (transition open), couplé au reset de la mutation start (2026-07-22)
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

  const axisLabelKey = (axisKind === 'radar'
    ? `profile.axis.${axis}`
    : `profile.lusr.${axis}`) as ProfileManifestKey
  const axisLabel = t(axisLabelKey)

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
            {t('campaign.modal.title', { axis: axisLabel })}
          </h2>
          <p className="mt-2 text-sm text-muted-foreground">{t('campaign.modal.subtitle')}</p>

          <div className="mt-4 space-y-2">
            <label
              htmlFor="campaign-playlist-group"
              className="block text-xs font-semibold uppercase text-muted-foreground"
            >
              {t('campaign.modal.playlist_label')}
            </label>
            <select
              id="campaign-playlist-group"
              value={playlistGroup}
              onChange={(e) => setPlaylistGroup(e.target.value)}
              disabled={muts.start.isPending}
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
            >
              {PLAYLIST_VALUES.map((value) => {
                const key = `campaign.playlist.${value}` as ProfileManifestKey
                return (
                  <option key={value} value={value}>
                    {t(key)}
                  </option>
                )
              })}
            </select>
            <p className="text-xs text-muted-foreground">{t('campaign.modal.playlist_help')}</p>
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
              {t('campaign.modal.skip')}
            </Button>
          )}
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => onOpenChange(false)}
            disabled={muts.start.isPending}
          >
            {t('campaign.modal.cancel')}
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
            {t('campaign.modal.submit')}
          </Button>
        </div>
      </div>
    </div>
  )
}
