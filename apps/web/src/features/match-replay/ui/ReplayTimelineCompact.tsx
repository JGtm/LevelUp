/**
 * ReplayTimelineCompact — LA FRISE DE LA BARRE D'UNE LIGNE : sur 30 px, quatre couches. Au-dessus, les
 * repères d'éliminations (les frags de TOUS les joueurs, à la couleur du camp du tueur) et les repères de
 * médias ; au milieu, le rail et sa progression ; dessous, la bande de dominance aux frags ; le curseur.
 *
 * ELLE LIT LE MÊME OBJET `timeline` QUE LA FRISE À PISTES (`ReplayTimelineTracks`) : mêmes bornes, même
 * dominance, mêmes médias, et des repères tirés du même parcours du fil — une ligne ajoutée au fil apparaît
 * sur les deux. Aucune lecture du document n'a lieu ici.
 *
 * LE GESTE EST CELUI DE LA FRISE À PISTES : le champ `input[type=range]` est le MÊME mécanisme (`sliderRef`,
 * `onScrub`) que la boucle de dessin pilote sans rendu React ; clic et glisser posent le curseur, et
 * `data-replay-timeline` exempte ce champ de la garde anti-frappe des raccourcis (Espace et les flèches
 * continuent de commander la lecture après un clic sur la frise). Le champ est transparent, posé par-dessus
 * des couches décoratives qui ne captent pas le pointeur ; seuls les repères de médias, qui sont des boutons,
 * le captent.
 *
 * LA GÉOMÉTRIE EST CELLE DES MARQUES (`trackLeft`, `trackWidth`, `trackLeftVar`) : un repère tombe sous le
 * centre du curseur à l'instant qu'il désigne. La progression se pose sur la racine par la variable que
 * `useReplayPlayback.writeCursor` écrit (`data-replay-cursor-host`). Les cotes sont `COTES_BARRE_COMPACTE`.
 */
import { useState } from 'react'

import { CURSOR_RATIO_VAR } from '../hooks/useReplayPlayback'
import type { ReplayTimeline } from '../hooks/useReplayTimeline'
import { REPLAY_TEXT } from '../i18n/i18n'
import { campInk, dominanceInk } from '../model/campInk'
import { COTES_BARRE_COMPACTE as C, friseVars } from '../model/cotesBarreCompacte'
import { trackLeft, trackLeftVar, trackWidth } from '../model/replayTimelineTracksLogic'
import { ReplayMediaLightbox } from './ReplayMediaLightbox'

/** Le champ : transparent, il porte le geste ; son curseur est le seul dessin que le navigateur y fait. */
const FIELD_CLASS = `absolute inset-x-0 top-0 z-10 m-0 w-full cursor-pointer appearance-none bg-transparent
  [&::-webkit-slider-runnable-track]:h-[var(--bc-frise)] [&::-webkit-slider-runnable-track]:bg-transparent
  [&::-webkit-slider-thumb]:box-border [&::-webkit-slider-thumb]:h-[var(--bc-boite)] [&::-webkit-slider-thumb]:w-[var(--bc-boite)]
  [&::-webkit-slider-thumb]:mt-[var(--bc-haut)] [&::-webkit-slider-thumb]:appearance-none
  [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:bg-foreground [&::-webkit-slider-thumb]:bg-clip-padding
  [&::-webkit-slider-thumb]:border-[length:var(--bc-bord)] [&::-webkit-slider-thumb]:border-solid [&::-webkit-slider-thumb]:border-transparent
  [&::-webkit-slider-thumb]:shadow-[0_0_0_var(--bc-halo)_color-mix(in_oklab,var(--foreground)_14%,transparent)]
  [&::-moz-range-track]:h-[var(--bc-frise)] [&::-moz-range-track]:bg-transparent
  [&::-moz-range-thumb]:box-border [&::-moz-range-thumb]:h-[var(--bc-boite)] [&::-moz-range-thumb]:w-[var(--bc-boite)]
  [&::-moz-range-thumb]:rounded-full [&::-moz-range-thumb]:bg-foreground [&::-moz-range-thumb]:bg-clip-padding
  [&::-moz-range-thumb]:border-[length:var(--bc-bord)] [&::-moz-range-thumb]:border-solid [&::-moz-range-thumb]:border-transparent
  [&::-moz-range-thumb]:shadow-[0_0_0_var(--bc-halo)_color-mix(in_oklab,var(--foreground)_14%,transparent)]`

export function ReplayTimelineCompact({ timeline }: { timeline: ReplayTimeline }) {
  const {
    sliderRef, minFrame, maxFrame, onScrub, fragMarks = [], dominance, allyOf, media, showMediaTrack,
    playing, onRequestPause, locale,
  } = timeline
  const t = REPLAY_TEXT[locale]
  const [openId, setOpenId] = useState<string | null>(null)
  const open = media.find((m) => m.item.id === openId) ?? null

  const openMedia = (id: string) => {
    if (playing) onRequestPause()
    setOpenId(id)
  }

  return (
    <div
      data-replay-cursor-host=""
      className="relative min-w-0 flex-1"
      style={{ height: C.friseHauteurPx, margin: `0 ${C.friseMargePx}px`, minWidth: C.friseLargeurMinPx, ...friseVars() }}
    >
      <div className="pointer-events-none absolute inset-x-0 top-0" style={{ height: C.reperesBandePx }}>
        {fragMarks.map((f) => (
          <i
            key={f.key}
            data-replay-frag-mark=""
            className="absolute top-0 block rounded-[1px]"
            style={{ left: trackLeft(f.ratio), width: C.repereLargeurPx, height: C.repereHauteurPx, marginLeft: -C.repereLargeurPx / 2, background: campInk(allyOf(f.teamId)) }}
          />
        ))}
      </div>

      <div className="pointer-events-none absolute inset-x-0 rounded-full bg-input" style={{ top: C.railHautPx, height: C.railEpaisseurPx }}>
        <i className="absolute inset-y-0 left-0 rounded-full bg-foreground" style={{ width: trackLeftVar(CURSOR_RATIO_VAR) }} />
      </div>

      <div
        className="pointer-events-none absolute inset-x-0 overflow-hidden rounded-full bg-muted/40"
        style={{ top: C.dominanceHautPx, height: C.dominanceEpaisseurPx }}
      >
        {dominance.map((s) => (
          <i
            key={s.key}
            className="absolute top-0 block h-full"
            style={{ left: trackLeft(s.from), width: trackWidth(s.from, s.to), background: dominanceInk(s.teamId, allyOf) }}
          />
        ))}
      </div>

      <input
        ref={sliderRef}
        type="range"
        data-replay-timeline=""
        min={minFrame}
        max={maxFrame}
        defaultValue={minFrame}
        onChange={onScrub}
        aria-label={t.time}
        className={FIELD_CLASS}
      />

      {showMediaTrack && media.length > 0 && (
        <div className="pointer-events-none absolute inset-x-0 top-0 z-20" style={{ height: C.reperesBandePx }}>
          {media.map(({ item, from, to }) => (
            <button
              key={item.id}
              type="button"
              onClick={() => openMedia(item.id)}
              aria-label={item.label ?? t.mediaOpen}
              title={item.label ?? t.mediaOpen}
              className="pointer-events-auto absolute top-0 cursor-pointer rounded-sm border border-input bg-secondary transition-colors hover:border-foreground"
              style={
                item.kind === 'clip'
                  ? { height: C.reperesBandePx, left: trackLeft(from), width: trackWidth(from, to), minWidth: C.mediaClipMinPx }
                  : { height: C.reperesBandePx, left: trackLeft(from), width: C.mediaPhotoPx, marginLeft: -C.mediaPhotoPx / 2 }
              }
            />
          ))}
        </div>
      )}

      {open && <ReplayMediaLightbox item={open.item} locale={locale} onClose={() => setOpenId(null)} />}
    </div>
  )
}
