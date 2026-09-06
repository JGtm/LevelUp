/**
 * ReplayMediaLightbox — UN MÉDIA DU JOUEUR, en grand, par dessus le rejeu.
 *
 * ELLE MET LE REJEU EN PAUSE, et le DIT (pastille « rejeu en pause ») : sans la pastille, un
 * lecteur qui referme la lightbox ne saurait pas pourquoi le film n'a pas avancé. La pause
 * elle-même est appliquée par l'appelant (ReplayTimelineTracks -> onRequestPause) : ce
 * composant ne connaît pas la boucle de lecture, il ne l'a jamais connue.
 *
 * PANNEAU EN SURIMPRESSION DANS LE CADRE DU REJEU, comme le tiroir de réglages (retour de
 * planche du 16/08) : le canvas ne se retaille pas, le rendu ne saute pas. Trois sorties, les
 * mêmes que le tiroir — Échap, le bouton, un clic dehors.
 *
 * LA BANDE D'IMAGES D'UN CLIP N'EST PAS DÉCORATIVE : elle dit la DURÉE. Un clip et une capture
 * s'ouvrent dans le même cadre ; sans la bande, rien à l'écran ne les distinguerait avant de
 * cliquer sur lecture. Le nombre d'images vient de `clipFrameCount` (une toutes les trois
 * secondes, bornée) — pas une par seconde, qui donnerait une bande illisible.
 *
 * UN CLIP EST UNE VIDÉO MÊME SANS DURÉE CONNUE (corrigé le 2026-08-28). La condition d'origine
 * confondait « c'est un clip » et « on connaît sa durée » : un clip dont la base ignore la
 * durée (ffprobe absent à l'ingestion) partait dans la branche image, et un `<img src=...mp4>`
 * ne montre rien. Le défaut n'était pas atteignable tant que la piste était vide ; il l'est
 * devenu avec la donnée. La durée ne commande plus que la BANDE et l'horloge.
 *
 * LES CLIPS TRANSCODÉS SONT DU HLS : leur `file_path` mue vers un `master.m3u8`, qu'un
 * `<video src>` nu ne lit pas sur Chrome/Firefox. L'attache est celle de la galerie, extraite
 * dans `@/lib/media/useHlsVideo` — pas une seconde copie.
 */
import { useEffect, useRef, useState } from 'react'

import { formatClockMMSS } from '@/lib/formatters'
import { useHlsVideo } from '@/lib/media/useHlsVideo'

import { REPLAY_TEXT, type ReplayLocale } from './i18n/i18n'
import { clipFrameCount, type ReplayMediaItem } from './replayTimelineTracksLogic'

interface ReplayMediaLightboxProps {
  item: ReplayMediaItem
  locale: ReplayLocale
  onClose: () => void
}

export function ReplayMediaLightbox({ item, locale, onClose }: ReplayMediaLightboxProps) {
  const t = REPLAY_TEXT[locale]
  const panelRef = useRef<HTMLDivElement>(null)
  const videoRef = useRef<HTMLVideoElement | null>(null)
  const [error, setError] = useState<string | null>(null)
  const isClip = item.kind === 'clip'
  const hasDuration = isClip && !!item.durationMs
  // Un seul média à l'écran : les segments se chargent tout de suite (le coverflow, lui, monte
  // cinq lecteurs et doit retenir les quatre autres).
  const { isHls } = useHlsVideo({
    videoRef,
    src: item.url,
    onFailure: (kind) => setError(kind === 'unsupported' ? t.mediaHlsUnsupported : t.mediaHlsError),
  })

  useEffect(() => {
    panelRef.current?.focus({ preventScroll: true })
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [onClose])

  return (
    <div
      className="absolute inset-0 z-30 flex flex-col items-center justify-center gap-3 bg-background/90 p-5"
      onPointerDown={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div
        ref={panelRef}
        tabIndex={-1}
        role="dialog"
        aria-label={item.label ?? t.mediaOpen}
        className="flex w-full max-w-[560px] flex-col gap-3 outline-none"
      >
        <div className="flex items-center gap-2.5">
          <span className="text-[13px] font-medium">{item.label ?? t.mediaOpen}</span>
          {hasDuration && (
            <span className="text-xs tabular-nums text-muted-foreground">
              {formatClockMMSS(item.durationMs ?? 0)}
            </span>
          )}
          <div className="flex-1" />
          <span className="inline-flex h-[22px] items-center gap-1.5 rounded-full bg-muted px-2.5 text-[10.5px] font-medium text-muted-foreground">
            <PauseGlyph />
            {t.mediaPausedHint}
          </span>
          <button
            type="button"
            onClick={onClose}
            aria-label={t.mediaClose}
            title={t.mediaClose}
            className="flex h-6 w-6 items-center justify-center rounded-full bg-muted text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          >
            ×
          </button>
        </div>

        {isClip ? (
          /* LE FOND DU LECTEUR EST UN TOKEN, jamais une couleur brute : une classe de couleur
             Tailwind dans `features/` est interdite par la règle du dépôt (skill color-tokens),
             et un letterbox noir en thème clair jurerait avec le reste de la page. `bg-muted`
             tient le rôle — une surface neutre derrière une vidéo qui ne remplit pas son cadre.

             PAS D'ATTRIBUT `src` SUR UN FLUX : c'est hls.js qui alimente l'élément (MSE). Le
             poser ferait tenter au navigateur une lecture directe du manifest, qui échoue. */
          <video
            ref={videoRef}
            src={isHls ? undefined : item.url}
            controls
            autoPlay
            className="w-full rounded bg-muted"
          />
        ) : (
          <img src={item.url} alt={item.label ?? ''} className="w-full rounded" />
        )}

        {error && <p className="text-xs text-muted-foreground">{error}</p>}

        {hasDuration && (
          <div className="flex flex-col gap-1.5">
            <div className="flex h-[34px] gap-[2px]">
              {Array.from({ length: clipFrameCount(item.durationMs ?? 0) }).map((_, i) => (
                <img key={i} src={item.thumbUrl} alt="" className="min-w-0 flex-1 rounded-[2px] object-cover" />
              ))}
            </div>
            <div className="flex justify-between text-[9.5px] tabular-nums text-muted-foreground">
              <span>{formatClockMMSS(0)}</span>
              <span>{formatClockMMSS(item.durationMs ?? 0)}</span>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

/** Icône pause, décorative : le libellé vit sur la pastille. */
function PauseGlyph() {
  return (
    <svg viewBox="0 0 16 16" className="h-2.5 w-2.5" fill="currentColor" aria-hidden="true">
      <rect x="3.5" y="2.5" width="3.4" height="11" rx="1" />
      <rect x="9.1" y="2.5" width="3.4" height="11" rx="1" />
    </svg>
  )
}
