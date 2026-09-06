/**
 * ReplayTimelineTracks — LA FRISE ET SES PISTES : tes éliminations et tes morts, celles de tes
 * alliés, la DOMINANCE (aux frags), le SCORE du mode, les MÉDIAS, puis le curseur de lecture.
 *
 * DOMINANCE ET SCORE SE LISENT ENSEMBLE, et c'est tout l'intérêt de les empiler : la première
 * dit qui gagne les duels, la seconde qui gagne le match. Sur un mode à objectif elles se
 * séparent — c'est là qu'on voit une équipe dominer les frags et perdre les captures. La
 * seconde n'existe donc PAS en Slayer, où elle répéterait la première (cf. `scoreTrack`).
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

import { REPLAY_TEXT, type ReplayLocale } from './i18n/i18n'
import { ReplayMediaLightbox } from './ReplayMediaLightbox'
import {
  clipFrameCount,
  trackLeft,
  trackWidth,
  type DominanceSegment,
  type PlacedMedia,
  type ReplayScoreTrack,
  type RoundSeparator,
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
  /** Les segments de dominance AUX FRAGS (cf. buildFragDominance) et le camp de chacun. */
  dominance: readonly DominanceSegment[]
  /**
   * La piste SCORE, ou `null` quand ce match n'en a pas : Slayer (le score y EST le compte des
   * frags), calque de score absent, camps non identifiés. `null` n'est pas « vide » — la
   * rangée n'existe alors pas, et son absence dit quelque chose du mode.
   */
  score: ReplayScoreTrack | null
  allyOf: (teamId: number) => boolean | null
  labelOf: (teamId: number) => string
  /** Les médias posés sur le match (cf. placeMedia). Vide = la piste reste, sans vignette. */
  media: readonly PlacedMedia[]
  /** Le titre déclare-t-il `media` ? Faux = la rangée n'existe pas (cf. l'en-tête). */
  showMediaTrack: boolean
  /** Frise dépliée (les pistes) ou repliée (la seule barre de lecture). Préférence persistée. */
  tracksExpanded: boolean
  onToggleTracks: () => void
  /** Ouvrir un média met le rejeu EN PAUSE : la lightbox le dit, l'appelant l'applique. */
  playing: boolean
  onRequestPause: () => void
  /**
   * LA BULLE DE TEMPS, écrite par `useReplayClock` en impératif (demande utilisateur du
   * 2026-09-02, qui remplace les trois bornes début/milieu/fin). Elle ne reçoit pas une chaîne
   * mais une RÉFÉRENCE, et c'est tout l'intérêt : le texte change soixante fois par seconde,
   * le passer en prop coûterait un rendu par image de la frise entière — pistes, vignettes de
   * médias et champ compris.
   */
  clockRef: RefObject<HTMLSpanElement | null>
  locale: ReplayLocale
}

export function ReplayTimelineTracks({
  sliderRef, minFrame, maxFrame, onScrub,
  own, allies, dominance, score, allyOf, labelOf, media, showMediaTrack,
  tracksExpanded, onToggleTracks,
  playing, onRequestPause, clockRef, locale,
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
        {tracksExpanded && (
          <>
            <TrackLabel>{t.trackYou}</TrackLabel>
            <MarkTrack marks={own} height="h-3.5" tall />

            <TrackLabel>{t.trackAllies}</TrackLabel>
            <MarkTrack marks={allies} height="h-3.5" tall={false} />

            <TrackLabel>{t.trackDominance}</TrackLabel>
            <LeadTrack
              segments={dominance}
              allyOf={allyOf}
              titleOf={(teamId) =>
                teamId == null ? t.dominanceTied : t.dominanceOfFmt(labelOf(teamId))
              }
            />

            {/* LA PISTE SCORE N'EXISTE PAS SUR TOUS LES MATCHS (cf. `scoreTrack`) : en Slayer,
                le score EST le compte des frags et la rangée répéterait celle du dessus. Son
                absence est donc un fait du mode — pas une rangée vide à remplir plus tard. */}
            {score && (
              <>
                <TrackLabel>{t.trackScore}</TrackLabel>
                <LeadTrack
                  segments={score.segments}
                  allyOf={allyOf}
                  titleOf={(teamId) =>
                    teamId == null ? t.scoreTied : t.scoreOfFmt(labelOf(teamId))
                  }
                  rounds={score.rounds}
                  roundTitleOf={(endedIndex) => t.roundOverFmt(endedIndex)}
                />
              </>
            )}

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
          </>
        )}

        <TracksToggle
          expanded={tracksExpanded}
          onToggle={onToggleTracks}
          label={tracksExpanded ? t.tracksCollapse : t.tracksExpand}
        />
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
          {/* LE TEMPS SUIT LE POINT QUI AVANCE (demande utilisateur du 2026-09-02). Il
              remplace les trois bornes début / milieu / fin, qui disaient une information
              constante sur trois lignes de plus, et le grand chrono qui vivait vingt pixels
              plus bas dans une autre taille : DEUX éléments pour une seule question — « où
              j'en suis » — désormais répondue à l'endroit exact où on la pose.

              `left: var(--played)` : la même variable que le remplissage de la piste, écrite
              par `useReplayPlayback.writeCursor` sur le parent (cf. son commentaire). Texte et
              position suivent donc la lecture par le MÊME chemin impératif, sans un rendu.

              `clamp` retient la bulle dans la frise à ses deux extrémités : centrée sur le
              curseur, elle déborderait de sa demi-largeur à 0 % et à 100 %. */}
          <div className="pointer-events-none relative h-[15px]">
            {/* `aria-hidden` ET C'EST DÉLIBÉRÉ : le champ juste au-dessus porte déjà
                `aria-label={t.time}`. Nommer la bulle pareil donnerait DEUX éléments du même
                nom pour une seule information — une gêne pour qui navigue au lecteur d'écran,
                et une ambiguïté pour qui teste par le nom accessible. La bulle est le doublon
                VISIBLE d'une valeur que le champ expose déjà. */}
            <span
              ref={clockRef}
              aria-hidden="true"
              className="absolute -translate-x-1/2 whitespace-nowrap text-[11px] font-medium tabular-nums text-muted-foreground"
              style={{ left: 'clamp(1.4rem, var(--played, 0%), calc(100% - 1.4rem))' }}
            />
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

/**
 * LE CHEVRON PREND LA PLACE DE L'ÉTIQUETTE VIDE qui faisait face au curseur : la colonne des
 * libellés est là où l'œil cherche de quoi parle une rangée, et cette rangée-là est la seule
 * qui ne disparaît jamais.
 *
 * LE LIBELLÉ PORTE LE GESTE OFFERT (« replier » quand c'est déplié), l'ÉTAT vit dans
 * `aria-expanded` : c'est la convention des commandes, et les deux se contrediraient si le nom
 * disait l'état.
 */
function TracksToggle({
  expanded, onToggle, label,
}: {
  expanded: boolean
  onToggle: () => void
  label: string
}) {
  return (
    <button
      type="button"
      onClick={onToggle}
      aria-expanded={expanded}
      aria-label={label}
      title={label}
      className="mt-[3px] flex h-4 w-full items-center justify-center rounded text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
    >
      <TracksChevron expanded={expanded} />
    </button>
  )
}

/**
 * Chevron du repli, décoratif : le nom accessible vit sur le bouton. IL POINTE VERS LES PISTES
 * QU'IL COMMANDE, et elles sont AU-DESSUS de lui — donc vers le HAUT quand la frise est dépliée
 * (le geste offert les fait remonter et disparaître) et vers le BAS quand elle est repliée
 * (elles vont redescendre). Le sens était inversé jusqu'au 2026-08-28 : la flèche montrait le
 * curseur, qui ne bouge jamais.
 *
 * Deuxième dessin de chevron de la feature (l'autre ouvre le menu de vitesse) : sous le seuil
 * de factorisation du dépôt, et les deux n'ont ni la même taille ni la même rotation.
 */
function TracksChevron({ expanded }: { expanded: boolean }) {
  return (
    <svg
      viewBox="0 0 16 16"
      className={`h-2.5 w-2.5 transition-transform ${expanded ? 'rotate-180' : ''}`}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M4 6.5 8 10.5 12 6.5" />
    </svg>
  )
}

/**
 * UNE PISTE DE MENEUR : des bandes colorées bout à bout, et — pour le score d'un mode
 * multi-manche — les repères de bascule de manche.
 *
 * DEUX PISTES, UN SEUL DESSIN (dominance aux frags, score du mode) : elles répondent à deux
 * questions mais se lisent de la même façon, et les faire diverger à la main aurait fini par
 * leur donner deux hauteurs, deux opacités et deux comportements de survol. Le TEXTE, lui,
 * reste l'affaire de l'appelant (`titleOf`) : « mène aux frags » et « mène au score » ne sont
 * pas la même affirmation.
 *
 * LES SÉPARATEURS DE MANCHE SONT AU-DESSUS DES BANDES et n'en coupent aucune : une manche qui
 * se termine ne change pas qui mène — elle remet le compteur à zéro, et c'est justement ce que
 * le repère explique à l'œil.
 */
function LeadTrack({
  segments, allyOf, titleOf, rounds = [], roundTitleOf,
}: {
  segments: readonly DominanceSegment[]
  allyOf: (teamId: number) => boolean | null
  titleOf: (teamId: number | null) => string
  rounds?: readonly RoundSeparator[]
  roundTitleOf?: (endedIndex: number) => string
}) {
  return (
    <div className="relative h-2.5 overflow-hidden rounded-full bg-muted/40">
      {segments.map((s) => (
        <span
          key={s.key}
          // PLEINE ENCRE, comme la piste « Toi » (retour utilisateur du 2026-08-28 : « les
          // couleurs sont ternes, ce ne sont pas les mêmes que sur la frise Toi »). Les bandes
          // portaient `opacity-55` — la même couleur délavée n'est plus la même couleur, et
          // deux teintes pour un seul sens (allié / adverse) se lisent comme deux sens.
          className="absolute top-0 block h-2.5"
          style={{
            left: trackLeft(s.from),
            width: trackWidth(s.from, s.to),
            background: dominanceInk(s.teamId, allyOf),
          }}
          title={titleOf(s.teamId)}
        />
      ))}
      {rounds.map((r) => (
        <span
          key={r.key}
          className="absolute top-0 block h-2.5 w-[2px] bg-background"
          style={{ left: trackLeft(r.ratio), marginLeft: -1 }}
          title={roundTitleOf?.(r.endedIndex)}
        />
      ))}
    </div>
  )
}

/**
 * L'encre d'une bande de dominance. TROIS CAS, ET TROIS ENCRES DISTINCTES :
 *  - ÉGALITÉ (`teamId === null`) : l'encre d'égalité DU DÉPÔT (`outcome-draw`, le bleu des
 *    matchs nuls, des tuiles neutres et des barres de bilan) — demande utilisateur du
 *    2026-08-28. Une quatrième couleur inventée ici ferait diverger le vocabulaire.
 *  - CAMP CONNU : allié ou adverse, les deux encres du rejeu.
 *  - CAMP NON RÉSOLU (`allyOf` rend `null`, joueur hors scoreboard) : la bordure neutre —
 *    jamais l'une des deux couleurs par défaut, qui désignerait un camp au hasard.
 */
function dominanceInk(teamId: number | null, allyOf: (teamId: number) => boolean | null): string {
  if (teamId == null) return tokenCssVar('outcome-draw')
  const ally = allyOf(teamId)
  if (ally === null) return 'var(--border)'
  return tokenCssVar(ally ? 'team-ally' : 'team-enemy')
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
