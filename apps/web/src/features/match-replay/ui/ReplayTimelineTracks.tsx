/**
 * ReplayTimelineTracks — LA FRISE ET SES PISTES : les éliminations et les morts du joueur
 * REGARDÉ, les éliminations de ses COÉQUIPIERS, la DOMINANCE (aux frags), le SCORE du mode, les
 * MÉDIAS, puis le curseur de lecture.
 *
 * # LA PREMIÈRE RANGÉE SE CHOISIT (2026-09-07, lot L3)
 *
 * Son libellé disait « Toi ». Il est devenu un MENU (`ReplayViewpointSelect`) : l'endroit où
 * l'on lit de qui parle la piste est celui où l'on change de joueur. Le geste ne touche qu'au
 * POINT DE VUE — ni le curseur, ni la lecture, ni la vitesse ne bougent (décision 1 du plan) :
 * changer de joueur change ce qu'on voit, jamais où l'on en est.
 *
 * La seconde piste a changé de population le même jour (décision 5) : elle montrait les joueurs
 * marqués AMIS, y compris ceux de l'équipe adverse, et montre désormais les COÉQUIPIERS du
 * joueur regardé. Les amis n'ont plus une piste, ils ont une FORME — le losange, comme sur la
 * carte (cf. `MarkTrack`).
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
 *
 * # UN SEUL TRAIT DE LECTURE TRAVERSE LES PISTES (2026-09-06)
 *
 * Quatre pistes empilées disent la forme du match, mais rien ne reliait une marque de kill à
 * l'instant qu'on écoute : l'œil devait descendre jusqu'au curseur, retenir sa position, et
 * remonter. Un trait vertical unique, à l'aplomb du curseur, répond à la question sur place.
 *
 * DEUX GRILLES PLUTÔT QU'UNE, et c'est ce trait qui l'impose. Le trait doit couvrir les pistes
 * ET S'ARRÊTER AU-DESSUS de la pastille du curseur — le traverser en ferait une croix. Une
 * grille unique ne sait pas exprimer « du haut de la première rangée au bas de l'avant-dernière »
 * sans une hauteur codée en dur, qui redeviendrait fausse dès qu'une rangée s'ajoute (la piste
 * Score n'existe pas sur tous les modes, la piste Médias dépend d'une capability). Les pistes
 * ont donc leur propre grille, et le trait s'y pose en `inset-y-0` : sa hauteur EST celle des
 * pistes, quelles qu'elles soient, sans un seul pixel écrit à la main. La rangée de transport
 * (chevron + curseur) vit dans une seconde grille, aux MÊMES colonnes — celles que
 * `replayTimelineGrid.ts` définit une fois pour les trois lecteurs, le trait compris : il a
 * besoin des mêmes nombres pour savoir où commence la colonne des pistes.
 */
import { useState, type ChangeEvent, type RefObject } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import { CURSOR_RATIO_VAR } from '../hooks/useReplayPlayback'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { ReplayMediaLightbox } from './ReplayMediaLightbox'
import { ReplayPlayhead } from './ReplayPlayhead'
import { ReplayViewpointSelect } from './ReplayViewpointSelect'
import { TIMELINE_GRID_COLUMNS } from './replayTimelineGrid'
import type { ViewpointOptionGroup } from '../model/viewpointOptions'
import {
  clipFrameCount,
  trackLeft,
  trackLeftVar,
  trackWidth,
  type DominanceSegment,
  type PlacedMedia,
  type ReplayScoreTrack,
  type RoundSeparator,
  type TrackMark,
} from '../model/replayTimelineTracksLogic'

/**
 * LA MARGE QUE LA BULLE DE TEMPS GARDE AUX DEUX BOUTS de la frise : sa demi-largeur, à peu près.
 * Centrée sur le curseur, elle déborderait d'autant à 0 % et à 100 %. C'est une propriété de la
 * BULLE (son texte fait « 00:00 »), pas de la géométrie des pistes — d'où une constante à elle.
 */
const BUBBLE_EDGE = '1.4rem'

interface ReplayTimelineTracksProps {
  /** Le curseur, piloté par la boucle de dessin — jamais contrôlé par React. */
  sliderRef: RefObject<HTMLInputElement | null>
  minFrame: number
  maxFrame: number
  onScrub: (e: ChangeEvent<HTMLInputElement>) => void
  /** Les marques du joueur regardé, celles de ses coéquipiers (cf. buildEventTracks). */
  own: readonly TrackMark[]
  teammates: readonly TrackMark[]
  /** Le point de vue courant : la valeur affichée par le menu de la première rangée. */
  viewpoint: string | null
  /** Les sections du menu, un camp par section (cf. `buildViewpointOptions`). */
  viewpointGroups: readonly ViewpointOptionGroup[]
  /** Poser le point de vue. `null` revient au joueur de la page. */
  onSelectViewpoint: (xuid: string | null) => void
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
  own, teammates, viewpoint, viewpointGroups, onSelectViewpoint,
  dominance, score, allyOf, labelOf, media, showMediaTrack,
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
    // `data-replay-cursor-host` — LA RACINE PORTE LES VARIABLES DE POSITION du curseur
    // (`--played`, `--played-r`), et c'est ICI que `useReplayPlayback.writeCursor` vient les
    // poser : il remonte depuis le champ par cet attribut (cf. `CURSOR_HOST_ATTR`). Sur la
    // rangée du champ, comme jusqu'au 2026-09-06, les pistes ne les auraient pas vues — une
    // propriété personnalisée n'hérite que vers le bas. Rien dans le typage ne relie l'écriture
    // et cette pose : c'est un test qui le fait (`ReplayTimelineTracks.test.tsx`).
    <div className="relative" data-replay-cursor-host="">
      {tracksExpanded && (
        <div className="relative mb-[5px] grid items-center gap-y-[5px]" style={TIMELINE_GRID_COLUMNS}>
          {/* LE LIBELLÉ DE LA PREMIÈRE RANGÉE EST LA COMMANDE (cf. l'en-tête) : c'est là qu'on
              lit de qui parle la piste, donc là qu'on change de joueur. */}
          <ReplayViewpointSelect
            value={viewpoint}
            groups={viewpointGroups}
            label={t.viewpointLabel}
            onSelect={onSelectViewpoint}
          />
          <MarkTrack marks={own} height="h-3.5" tall />

          <TrackLabel>{t.trackTeammates}</TrackLabel>
          <MarkTrack marks={teammates} height="h-3.5" tall={false} />

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

          {/* LE TRAIT DE LECTURE FERME LA PILE : il est le dernier enfant pour passer AU-DESSUS
              des pistes, et il ne vit qu'ici — repliée, la frise n'a rien à traverser. */}
          <ReplayPlayhead />
        </div>
      )}

      <div className="grid items-center" style={TIMELINE_GRID_COLUMNS}>
        <TracksToggle
          expanded={tracksExpanded}
          onToggle={onToggleTracks}
          label={tracksExpanded ? t.tracksCollapse : t.tracksExpand}
        />
        {/* LE CURSEUR. `--played` (le remplissage) et `--played-r` (la position des pistes)
            sont écrits par la boucle de dessin (useReplayPlayback) sur la RACINE ci-dessus : le
            champ les reçoit par héritage, et tout suit la lecture sans un seul rendu React. */}
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

              SA POSITION EST CELLE DES MARQUES, PAS UN POURCENTAGE BRUT (corrigé le
              2026-09-06). Elle lisait `var(--played)` — la part parcourue en pourcentage de la
              largeur — alors que tout ce qui se pose sur les pistes vit dans la géométrie du
              CURSEUR (`trackLeft` : le curseur natif réserve sa demi-largeur à chaque bout).
              La bulle portait donc jusqu'à 8 px de décalage avec l'instant qu'elle annonce ;
              invisible tant qu'aucun repère ne passait par là, criant depuis que le trait de
              lecture le fait. Elle emploie maintenant `trackLeftVar` — la MÊME formule que les
              marques et que le trait, écrite une seule fois dans le module de logique.

              Le chemin, lui, ne change pas : une variable écrite en impératif par
              `useReplayPlayback.writeCursor` sur la racine de la frise (cf. son commentaire).
              Texte et position suivent la lecture sans un seul rendu React.

              `clamp` retient la bulle dans la frise à ses deux extrémités : centrée sur le
              curseur, elle déborderait de sa demi-largeur à 0 % et à 100 %. Ses deux bornes
              sont des marges de BULLE (sa demi-largeur), pas de la géométrie de piste — c'est
              pourquoi elles restent écrites ici et que le garde-rail les exempte nommément. */}
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
              style={{
                left: `clamp(${BUBBLE_EDGE}, ${trackLeftVar(CURSOR_RATIO_VAR)}, calc(100% - ${BUBBLE_EDGE}))`,
              }}
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
 * Une piste de marques. `tall` distingue celle du joueur REGARDÉ (marques pleines, plus hautes)
 * de celle de ses coéquipiers (plus basses, atténuées) : deux pistes de même poids se liraient
 * comme une seule.
 *
 * UN AMI PREND LE LOSANGE (décision 4 du plan « frise, point de vue », 2026-09-07). C'est la
 * grammaire de la carte, transposée telle quelle : la FORME dit l'identité, la COULEUR dit le
 * camp — et ici la couleur est déjà prise deux fois (kill à l'encre alliée, mort à l'encre
 * adverse), sur deux tokens qui reprennent les couleurs d'équipe choisies par l'utilisateur en
 * jeu. Il n'en reste aucune de libre, et en inventer une troisième pour « ami » ferait dire
 * deux choses à la même grandeur.
 *
 * LE MOT EST CELUI DE LA CARTE, PAS UN SECOND VOCABULAIRE : `MarkerShape = 'diamond'`
 * (`layers/replayMarkers.ts`) nomme déjà cette forme pour la même population.
 *
 * LA MARQUE AMIE EST CARRÉE AVANT D'ÊTRE TOURNÉE, et c'est ce qui en fait un losange : une
 * barre de 2 à 3 px de large pivotée de 45° donne une oblique, pas un losange — la forme ne se
 * lirait pas à cette taille. Elle garde le CENTRE des autres (même `top` + même translation) et
 * la même encre : seule sa silhouette change.
 *
 * UNE MARQUE EST CENTRÉE SUR SON INSTANT, pas posée à sa droite (décision utilisateur du
 * 2026-09-06, prise en même temps que le trait de lecture). Elle se posait par son BORD GAUCHE
 * sur `trackLeft(ratio)` : large de deux à trois pixels, elle débordait donc tout entière vers
 * la droite, et son milieu — ce que l'œil lit comme « l'endroit » de la marque — tombait un
 * pixel et demi après le frag. Le décalage était invisible tant que rien ne passait par là ;
 * le trait de lecture, lui, l'aurait exhibé à chaque kill.
 *
 * LA TRANSLATION EST DONC LE CENTRAGE, et elle vaut la demi-largeur de la marque quelle qu'elle
 * soit (`-translate-x-1/2` se mesure sur l'élément, pas sur la piste) : les marques hautes et
 * les basses n'ont pas la même largeur et n'ont pas à s'en soucier. L'ANCRE, elle, ne change
 * pas — c'est toujours `trackLeft(ratio)`, le point exact où la pastille du curseur se centre
 * et où passe le trait de lecture. Les trois coïncident maintenant au pixel.
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
          className={`pointer-events-none absolute -translate-x-1/2 rounded-[2px] ${markShape(m.friend, tall)}`}
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

/**
 * LA SILHOUETTE D'UNE MARQUE : barre verticale par défaut, LOSANGE pour un ami (décision 4).
 *
 * LES QUATRE VARIANTES PARTAGENT LEUR CENTRE VERTICAL — y = 7 px sur une piste de 14 (`h-3.5`) :
 * `top-[3px]` + 8 px de haut, `top-1` + 6, `top-[4px]` + 6 pour les deux losanges. Sans cette
 * égalité, une marque amie flotterait un pixel plus haut que ses voisines et la piste se lirait
 * comme deux lignes. Le losange est plus court que la barre haute parce qu'il occupe sa
 * DIAGONALE une fois tourné (6 px de côté ≈ 8,5 px en travers) : à taille égale il dépasserait
 * de la piste.
 *
 * L'ATTÉNUATION reste celle de la piste, pas de la forme : la piste des coéquipiers est plus
 * discrète que celle du joueur regardé, qu'on y soit ami ou non.
 */
function markShape(friend: boolean, tall: boolean): string {
  if (friend) return tall ? 'top-[4px] h-1.5 w-1.5 rotate-45' : 'top-[4px] h-1.5 w-1.5 rotate-45 opacity-65'
  return tall ? 'top-[3px] h-2 w-[3px]' : 'top-1 h-1.5 w-[2px] opacity-65'
}
