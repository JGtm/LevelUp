/**
 * ReplayTimelineTracks — LA FRISE ET SES PISTES : tes éliminations et tes morts, celles de tes
 * alliés, la DOMINANCE, les MÉDIAS, puis le curseur de lecture.
 *
 * ELLE REMPLACE LE `input[type=range]` NU de la barre de lecture (validé le 2026-08-28,
 * planche 2a). Le curseur reste ce même `input` — c'est lui que la boucle de dessin pilote
 * (`sliderRef`), et le rendre contrôlé par React coûterait un rendu par image. Ce qui change :
 * il est habillé (piste, remplissage, pastille) et il n'est plus seul — quatre pistes se
 * posent AU-DESSUS de lui, à la même échelle et à la même géométrie — celle que calcule
 * `replayTimelineTracksLogic.ts`. LE SUFFIXE `Logic` N'EST PAS DÉCORATIF : Windows ne
 * distingue pas `ReplayTimelineTracks.tsx` de `replayTimelineTracks.ts`, et TypeScript refuse
 * alors les deux fichiers dans le même programme (TS1149) — l'import de ce composant résolvait
 * vers le module de logique. C'est le patron du dépôt de toute façon (killFeedLogic,
 * victoryLogic, scoreBannerLogic).
 *
 * LES PISTES NE CAPTENT PAS LE POINTEUR (`pointer-events-none`), sauf les vignettes de médias
 * qui sont des boutons : la frise reste saisissable au pixel près, y compris SOUS une marque.
 *
 * LA PISTE MÉDIAS RESTE AFFICHÉE MÊME VIDE (demande utilisateur du 2026-08-28 : « une barre
 * Médias qu'on affichera toujours »). C'est l'exception à la règle du dépôt « pas de commande
 * quand il n'y a rien à commander » : ce n'est pas une commande, c'est un emplacement — il dit
 * où les médias du match vivront, et son vide est une information.
 *
 * VIDE N'EST PAS ABSENTE. « Aucun média sur ce match » est un fait ; « ce jeu n'a pas de
 * médias » n'en est pas un, et une rangée vide le dirait à tort. La rangée disparaît donc
 * quand le titre ne déclare pas la capability `media` (`showMediaTrack`) — le rejeu, lui,
 * n'est gardé que par `matchmaking`, les deux ne se recouvrent pas.
 */
import { useState, type ChangeEvent, type RefObject } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { ReplayMediaLightbox } from './ReplayMediaLightbox'
import {
  clipFrameCount,
  trackLeft,
  trackWidth,
  type DominanceSegment,
  type PlacedMedia,
  type TrackMark,
} from './replayTimelineTracksLogic'

interface ReplayTimelineTracksProps {
  /** Le curseur, piloté par la boucle de dessin — jamais contrôlé par React. */
  sliderRef: RefObject<HTMLInputElement | null>
  minFrame: number
  maxFrame: number
  onScrub: (e: ChangeEvent<HTMLInputElement>) => void
  /** Tes marques, celles de tes alliés (cf. buildEventTracks). */
  own: readonly TrackMark[]
  allies: readonly TrackMark[]
  /** Les segments de dominance (cf. buildDominance) et le camp de chacun. */
  dominance: readonly DominanceSegment[]
  allyOf: (teamId: number) => boolean | null
  labelOf: (teamId: number) => string
  /** Les médias posés sur le match (cf. placeMedia). Vide = la piste reste, sans vignette. */
  media: readonly PlacedMedia[]
  /** Le titre déclare-t-il `media` ? Faux = la rangée n'existe pas (cf. l'en-tête). */
  showMediaTrack: boolean
  /** Ouvrir un média met le rejeu EN PAUSE : la lightbox le dit, l'appelant l'applique. */
  playing: boolean
  onRequestPause: () => void
  /** Les bornes de l'axe, déjà en mm:ss (le foyer du dépôt les met en forme). */
  startClock: string
  midClock: string
  endClock: string
  locale: ReplayLocale
}

export function ReplayTimelineTracks({
  sliderRef, minFrame, maxFrame, onScrub,
  own, allies, dominance, allyOf, labelOf, media, showMediaTrack,
  playing, onRequestPause, startClock, midClock, endClock, locale,
}: ReplayTimelineTracksProps) {
  const t = REPLAY_TEXT[locale]
  const [openId, setOpenId] = useState<string | null>(null)
  const open = media.find((m) => m.item.id === openId) ?? null

  const openMedia = (id: string) => {
    if (playing) onRequestPause()
    setOpenId(id)
  }

  return (
    <div className="relative">
      <div className="grid grid-cols-[76px_1fr] items-center gap-x-3 gap-y-[5px]">
        <TrackLabel>{t.trackYou}</TrackLabel>
        <MarkTrack marks={own} height="h-3.5" tall />

        <TrackLabel>{t.trackAllies}</TrackLabel>
        <MarkTrack marks={allies} height="h-3.5" tall={false} />

        <TrackLabel>{t.trackDominance}</TrackLabel>
        <div className="relative h-2.5 overflow-hidden rounded-full bg-muted/40">
          {dominance.map((s) => {
            const ally = allyOf(s.teamId)
            return (
              <span
                key={s.key}
                className="absolute top-0 block h-2.5 opacity-55"
                style={{
                  left: trackLeft(s.from),
                  width: trackWidth(s.from, s.to),
                  background:
                    ally === null ? 'var(--border)' : tokenCssVar(ally ? 'team-ally' : 'team-enemy'),
                }}
                title={t.dominanceOfFmt(labelOf(s.teamId))}
              />
            )
          })}
        </div>

        {showMediaTrack && (
          <>
            <TrackLabel>{t.mediaTrack}</TrackLabel>
            <div className="relative h-[26px] rounded-md border border-border bg-muted/30">
              {media.length === 0 && (
                <span className="pointer-events-none absolute inset-0 flex items-center justify-center text-[10px] text-muted-foreground">
                  {t.mediaEmpty}
                </span>
              )}
              {media.map(({ item, from, to }) => (
                <button
                  key={item.id}
                  type="button"
                  onClick={() => openMedia(item.id)}
                  className="absolute top-[3px] flex h-[18px] overflow-hidden rounded-sm border border-input transition-colors hover:border-foreground"
                  style={
                    item.kind === 'clip'
                      ? { left: trackLeft(from), width: trackWidth(from, to), minWidth: 14 }
                      : { left: trackLeft(from), width: 30, marginLeft: -15 }
                  }
                  aria-label={item.label ?? t.mediaOpen}
                  title={item.label ?? t.mediaOpen}
                >
                  {item.kind === 'clip' ? (
                    Array.from({ length: clipFrameCount(item.durationMs ?? 0) }).map((_, i) => (
                      <img
                        key={i}
                        src={item.thumbUrl}
                        alt=""
                        className="h-full min-w-0 flex-1 object-cover"
                      />
                    ))
                  ) : (
                    <img src={item.thumbUrl} alt="" className="h-full w-full object-cover" />
                  )}
                </button>
              ))}
            </div>
          </>
        )}

        <div />
        {/* LE CURSEUR. `--played` est écrit par la boucle de dessin (useReplayPlayback) : le
            remplissage suit donc la lecture sans un seul rendu React. */}
        <div className="relative mt-[3px]">
          {/* `data-replay-timeline` REND SA FRAPPE AU LECTEUR (décision utilisateur du
              2026-08-28, gate de la planche 2a). Un `input[type=range]` est un champ de saisie
              aux yeux du navigateur, et la garde anti-frappe de `useReplayShortcuts` l'attrapait :
              les raccourcis mouraient dès qu'on avait cliqué sur la frise — c'est-à-dire au
              moment précis où l'on analyse un match, là où Espace et les flèches sont les gestes
              qu'on fait. Cet attribut exempte CE champ, nommément : le curseur de volume, lui,
              reste un champ de saisie, et ses flèches continuent de régler le volume. */}
          <input
            ref={sliderRef}
            type="range"
            data-replay-timeline=""
            min={minFrame}
            max={maxFrame}
            defaultValue={minFrame}
            onChange={onScrub}
            aria-label={t.time}
            className="block h-4 w-full cursor-pointer appearance-none bg-transparent
              [&::-webkit-slider-runnable-track]:h-1.5 [&::-webkit-slider-runnable-track]:rounded-full
              [&::-webkit-slider-runnable-track]:bg-[linear-gradient(to_right,var(--foreground)_0_var(--played,0%),var(--input)_var(--played,0%)_100%)]
              [&::-webkit-slider-thumb]:-mt-1 [&::-webkit-slider-thumb]:h-3.5 [&::-webkit-slider-thumb]:w-3.5
              [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:rounded-full
              [&::-webkit-slider-thumb]:bg-foreground
              [&::-moz-range-track]:h-1.5 [&::-moz-range-track]:rounded-full
              [&::-moz-range-track]:bg-[linear-gradient(to_right,var(--foreground)_0_var(--played,0%),var(--input)_var(--played,0%)_100%)]
              [&::-moz-range-thumb]:h-3.5 [&::-moz-range-thumb]:w-3.5 [&::-moz-range-thumb]:border-0
              [&::-moz-range-thumb]:rounded-full [&::-moz-range-thumb]:bg-foreground"
          />
          <div className="pointer-events-none flex justify-between text-[9.5px] tabular-nums text-muted-foreground">
            <span>{startClock}</span>
            <span>{midClock}</span>
            <span>{endClock}</span>
          </div>
        </div>
      </div>

      {open && (
        <ReplayMediaLightbox
          item={open.item}
          locale={locale}
          onClose={() => setOpenId(null)}
        />
      )}
    </div>
  )
}

function TrackLabel({ children }: { children: React.ReactNode }) {
  return (
    <div className="text-[9.5px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
      {children}
    </div>
  )
}

/**
 * Une piste de marques. `tall` distingue la TIENNE (marques pleines, plus hautes) de celle des
 * alliés (plus basses, atténuées) : deux pistes de même poids se liraient comme une seule.
 */
function MarkTrack({
  marks, height, tall,
}: {
  marks: readonly TrackMark[]
  height: string
  tall: boolean
}) {
  return (
    <div className={`relative ${height} rounded-full bg-muted/40`}>
      {marks.map((m) => (
        <span
          key={m.key}
          className={`pointer-events-none absolute rounded-[2px] ${tall ? 'top-[3px] h-2 w-[3px]' : 'top-1 h-1.5 w-[2px] opacity-65'}`}
          style={{
            left: trackLeft(m.ratio),
            background: tokenCssVar(m.kind === 'kill' ? 'team-ally' : 'team-enemy'),
          }}
          title={m.clock}
        />
      ))}
    </div>
  )
}
