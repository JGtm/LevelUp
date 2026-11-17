/**
 * ReplayTeamsRow — LA RANGÉE DES ÉQUIPES du Cinéma et du Plein cadre, au-dessus du plan :
 * `[fiches du camp de gauche] [score] [fiches du camp de droite]`. Le camp du joueur regardé est à gauche
 * (comme la barre alliée du bandeau) ; rien n'est écrit sous le score.
 *
 * LA RANGÉE REMPLIT LA LARGEUR DU CORPS : les fiches se partagent la place à parts égales (colonnes de
 * grille posées en `style`, `minmax(0, 1fr)`), la largeur minimale de `COTES_RANGEE` n'étant que le seuil de
 * la règle de tenue (`model/rowLayout.ts`, que le cadre applique avant de monter la rangée). Les trois cellules
 * (camp, score, camp) et les colonnes de chaque camp sont séparées par le même écart : c'est la grille de la
 * maquette à 13 colonnes (4 + score + 4, ou 6 + score + 6), écrite en trois cellules pour que les deux camps
 * restent de même largeur quand l'un a moins de places que l'autre.
 *
 * DEUX FORMES, une seule rangée (`grandeEquipe`) :
 *  - fiches ordinaires : une `ReplayPlayerCard` au gabarit compact par place, arme, grenades et équipement
 *    compris, avec tous ses effets ; les places en attente (libre, occupant pas encore apparu) gardent leur
 *    boîte tiretée ;
 *  - Grande équipe : par camp, des EMPLACEMENTS de deux `ReplayCompactCard` empilées (le dernier n'en porte
 *    qu'une quand le camp est impair), sans frags, morts, assistances, arme ni équipement : ils sont dans la
 *    fiche du joueur, un clic plus loin.
 *
 * LES PLACES SONT CELLES DU FILM, comme dans la colonne (`buildSeats`, `groupSeatsByTeam`, `seatTileAt`) :
 * une place est une fiche, jamais un joueur parti ; une place qui ne rend rien à cette image garde sa
 * colonne vide, sans quoi les suivantes glisseraient. Le joueur suivi arrive par prop (`tracked`), la rangée
 * ne le recalcule pas ; un clic rend la page (`onCard`) avec la clé de roster du joueur.
 */
import { Fragment, useMemo, type ReactNode } from 'react'

import type { FilmAllegiance } from '@/lib/replay/filmAllegiance'

import { ReplayCompactCard, ReplayCompactWaitingCard } from './ReplayCompactCard'
import { ReplayPlayerCard, ReplaySeatWaiting } from './ReplayPlayerCard'
import { useCardScene, type CardScene } from '../hooks/useCardScene'
import { COTES_RANGEE, COTES_RANGEE_GRANDE_EQUIPE } from '../model/cardGabarit'
import { alliesAGauche, colonnesGrandeEquipe, NB_CAMPS_RANGEE } from '../model/rowLayout'
import {
  buildSeats,
  groupSeatsByTeam,
  seatTileAt,
  sheetRowsOfGroup,
  type ReplaySeat,
  type ReplaySeatGroup,
} from '../model/seatLogic'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { campLabel } from '../../../lib/replay/replayCamps'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'

export interface ReplayTeamsRowProps {
  doc: ReplayDocumentReady
  /** La jointure film <-> base du match (`model.players`) : la page la construit une fois. */
  players: readonly ReplayPlayer[]
  frame: number
  locale: ReplayLocale
  /** L'allégeance lue dans le film, vue du joueur regardé : le camp allié est à gauche, la pastille a la couleur du camp. */
  allegiance: FilmAllegiance
  /** Le joueur suivi (le joueur regardé par la page, `playerPanel.tracked`) : sa fiche prend la bordure de suivi. */
  tracked: ReplayPlayer | null
  /** Un clic sur la fiche d'un joueur présent (`playerPanel.onCard`), avec sa clé de roster. Absent = fiches inertes. */
  onCard?: (xuid: string) => void
  /** La clé de roster du joueur dont la fiche est ouverte (`aria-pressed`). */
  selectedXuid?: string | null
  /** Le bandeau de score (emplacement `rangee`), posé dans la cellule centrale ; la rangée n'y lit rien. */
  score: ReactNode
  /** Vrai = Grande équipe : des emplacements de deux fiches compactes empilées au lieu d'une fiche par place. */
  grandeEquipe: boolean
}

/** Ce que le rendu des places d'un camp partage. */
interface CampCtx
  extends Pick<ReplayTeamsRowProps, 'doc' | 'frame' | 'locale' | 'tracked' | 'onCard' | 'selectedXuid' | 'grandeEquipe'> {
  scene: CardScene
}

/** Une place : la fiche de son occupant, la boîte d'attente, ou rien (la colonne reste, vide). */
function seatCell(seat: ReplaySeat, ally: boolean | null, ctx: CampCtx): ReactNode {
  const { doc, frame, locale, scene, tracked, onCard, selectedXuid = null, grandeEquipe } = ctx
  const lu = seatTileAt(seat, frame)
  if (lu === null) return grandeEquipe ? null : <div key={seat.key} aria-hidden />
  if (lu.kind !== 'present' || lu.player === null) {
    return grandeEquipe ? (
      <ReplayCompactWaitingCard key={seat.key} occupant={lu.player} locale={locale} form="stacked" />
    ) : (
      <ReplaySeatWaiting key={seat.key} occupant={lu.player} locale={locale} />
    )
  }
  const player = lu.player
  const isTracked = tracked !== null && tracked.xuid === player.xuid
  const selected = selectedXuid === player.xuid
  if (grandeEquipe) {
    return (
      <ReplayCompactCard
        key={seat.key}
        player={player}
        doc={doc}
        frame={frame}
        locale={locale}
        form="stacked"
        scene={scene}
        ally={ally}
        tracked={isTracked}
        selected={selected}
        onCard={onCard}
      />
    )
  }
  return (
    <ReplayPlayerCard
      key={seat.key}
      player={player}
      doc={doc}
      frame={frame}
      locale={locale}
      presence={scene.presence}
      vitalityFade={scene.vitalityFade}
      readingFull={scene.readingFull}
      flashFrames={scene.flashFrames}
      scoreTimeline={scene.scoreTimeline}
      fxScene={scene.fxScene}
      tracked={isTracked}
      panel={onCard ? { selected, onSelect: () => onCard(player.xuid) } : undefined}
    />
  )
}

/** Les cellules d'un camp, rangées par emplacements de `fichesParEmplacement` fiches en Grande équipe. */
function campCells(group: ReplaySeatGroup, ally: boolean | null, ctx: CampCtx): ReactNode[] {
  const cells = group.seats.map((seat) => seatCell(seat, ally, ctx))
  if (!ctx.grandeEquipe) return cells
  const { fichesParEmplacement, ecartEmplacementPx } = COTES_RANGEE_GRANDE_EQUIPE
  const slots: ReactNode[] = []
  for (let i = 0; i < cells.length; i += fichesParEmplacement) {
    slots.push(
      <div key={group.seats[i].key} className="flex min-w-0 flex-col" style={{ gap: ecartEmplacementPx }}>
        {cells.slice(i, i + fichesParEmplacement)}
      </div>,
    )
  }
  return slots
}

/** Un camp : une grille de `columns` colonnes à parts égales, séparées par l'écart de la rangée. */
function Camp({ label, columns, children }: { label: string; columns: number; children: ReactNode }) {
  return (
    <section
      aria-label={label}
      className="grid min-w-0"
      style={{ gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`, columnGap: COTES_RANGEE.ecartFiches }}
    >
      {children}
    </section>
  )
}

export function ReplayTeamsRow({
  doc, players, frame, locale, allegiance, tracked, onCard, selectedXuid = null, score, grandeEquipe,
}: ReplayTeamsRowProps) {
  const t = REPLAY_TEXT[locale]
  const groups = useMemo(
    () => alliesAGauche(groupSeatsByTeam(buildSeats(players, doc)), (team) => allegiance.ofTeam(team)),
    [players, doc, allegiance],
  )
  const labels = useMemo(() => groups.map((g) => campLabel(g, sheetRowsOfGroup(g), t)), [groups, t])
  const scene = useCardScene(doc, players)
  // La règle de tenue n'admet que deux camps ; sans eux, la rangée n'a rien à dire.
  if (groups.length !== NB_CAMPS_RANGEE) return null
  const places = groups.map((g) => g.seats.length)
  const columns = grandeEquipe ? colonnesGrandeEquipe(places) : Math.max(...places)
  const ctx: CampCtx = { doc, frame, locale, tracked, onCard, selectedXuid, grandeEquipe, scene }
  return (
    <div
      data-replay-teams-row={grandeEquipe ? 'grande-equipe' : 'fiches'}
      className="grid"
      style={{
        gridTemplateColumns: `minmax(0, 1fr) ${COTES_RANGEE.scoreCellW}px minmax(0, 1fr)`,
        columnGap: COTES_RANGEE.ecartFiches,
      }}
    >
      {groups.map((group, i) => (
        <Fragment key={`camp:${group.team}`}>
          {i === 1 && (
            <div data-replay-teams-row-score="" className="flex min-w-0 items-center justify-center">
              {score}
            </div>
          )}
          <Camp label={labels[i]} columns={columns}>
            {campCells(group, allegiance.ofTeam(group.team), ctx)}
          </Camp>
        </Fragment>
      ))}
    </div>
  )
}
