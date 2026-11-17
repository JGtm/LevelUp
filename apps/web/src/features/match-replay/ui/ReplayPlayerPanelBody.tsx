/**
 * ReplayPlayerPanelBody — LE CONTENU COMMUN DE LA FICHE D'UN JOUEUR, indépendant de ce qui la porte.
 *
 * DE HAUT EN BAS :
 *  - la TÊTE : le nom à l'encre de son camp (l'allégeance du film), les compteurs frags / morts /
 *    assistances AU FORMAT DE LA FICHE (`ReplayCountersBadge`, ceux de l'instant — datés par l'horloge
 *    du match —, ou les totaux de la feuille quand le film ne publie pas ce joueur), puis `headEnd` ;
 *  - les TROIS ACTIONS sur la carte — Suivre, Seul, Masquer — boutons à bascule (`aria-pressed`) ;
 *    leurs exclusions et leur remise à zéro sont dans `playerPanelLogic` ;
 *  - une région qui DÉFILE (par défaut : `scroll: 'internal'`) : le bloc « À L'INSTANT »
 *    (`ReplayPlayerNow` : bouclier, santé, armes, grenades et équipement du joueur à l'image lue), puis les LISTES de totaux de fin de match
 *    (précision, dégâts, détail du score, objectif, armes) lues sur la feuille. Chaque titre dit « sur
 *    le match » : le film ne publie pas la précision en cours de partie. Un joueur sans ligne de
 *    feuille n'a pas de listes : un bloc pointillé le dit, plutôt que des zéros.
 *
 * CE QUE LE CORPS NE PORTE PAS : tout ce qui change d'une surface à l'autre — le retour au fil, la
 * fermeture, le sélecteur de joueur, l'écoute d'Échap, le nom accessible de la surface, sa hauteur.
 * Le conteneur donne une colonne flex de hauteur bornée (`min-h-0`) et, s'il veut un bouton à droite
 * de la tête, le passe par `headEnd`. Le corps rend un fragment : ses trois blocs sont les enfants
 * directs du conteneur. Quand le conteneur fait défiler TOUTE la fiche (`scroll: 'outer'`), la région
 * basse ne défile plus d'elle-même : elle prend sa hauteur naturelle et c'est le conteneur qui défile.
 *
 * Le corps reçoit tout par props et ne lit que ce qu'on lui donne. Ses boutons suivent la règle des
 * boutons du rejeu : la souris ne prend pas le focus, Espace reste « Lecture » (`PRESS_GUARDS`).
 * Aucun littéral de couleur : l'encre du camp et les neutres sont des jetons sémantiques.
 */
import { useMemo, type ReactNode } from 'react'

import { Button } from '@/components/ui/button'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { teamTokenCssVar } from '@/features/match-view/teamSeriesColor'
import type { FilmAllegiance } from '@/lib/replay/filmAllegiance'
import { formatClock } from '@/lib/replay/replayLogic'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'
import { playerCountersAt, scoreTimelineOf } from '@/lib/replay/scoreTimeline'

import { useReadingAges } from '../hooks/useCardScene'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { GABARIT_NORMAL } from '../model/cardGabarit'
import { HUD_BAND_CLASS, hudBandStyle } from '../model/hudBand'
import { cardName } from '../model/playerCardReadings'
import { playerPanelSheet, type PlayerPanelAction } from '../model/playerPanelLogic'
import { displayClockMs, type ReplayWindowBounds } from '../model/replayWindow'
import { ReplayCountersBadge } from './ReplayCountersBadge'
import { ReplayPlayerNow } from './ReplayPlayerNow'
import { PanelSectionList } from './ReplayPlayerPanelList'
import { PRESS_GUARDS } from './replayPressGuards'

/** L'ordre des trois actions : celui de la maquette. */
const ACTIONS: readonly PlayerPanelAction[] = ['follow', 'solo', 'hide']

export interface ReplayPlayerPanelBodyProps {
  player: ReplayPlayer
  /** Les trois actions et leur état : l'état est tenu par la page (`reducePlayerPanel`). */
  actions: Record<PlayerPanelAction, boolean>
  onToggle: (action: PlayerPanelAction) => void
  doc: ReplayDocumentReady
  /** L'image publiée (les compteurs de la tête sont ceux de cet instant). */
  frame: number
  /** L'instant courant sur l'axe BRUT du film, et la fenêtre qui le recale pour l'affichage. */
  nowMs: number
  playWindow: ReplayWindowBounds | null
  allegiance: FilmAllegiance
  locale: ReplayLocale
  /** Ce que le conteneur pose à droite de la bande de tête (retour, fermeture) ; rien par défaut. */
  headEnd?: ReactNode
  /**
   * Qui fait défiler la partie basse : le corps lui-même (`'internal'`, défaut : la tête et les actions
   * restent en place) ou son conteneur (`'outer'` : la région garde sa hauteur naturelle).
   */
  scroll?: 'internal' | 'outer'
}

/** La région basse du corps, selon qui la fait défiler. */
const LOWER_CLASS: Record<NonNullable<ReplayPlayerPanelBodyProps['scroll']>, string> = {
  internal: 'min-h-0 flex-1 space-y-3 overflow-y-auto pr-1',
  outer: 'space-y-3',
}

export function ReplayPlayerPanelBody({
  player, actions, onToggle, doc, frame, nowMs, playWindow, allegiance, locale, headEnd, scroll = 'internal',
}: ReplayPlayerPanelBodyProps) {
  const t = REPLAY_TEXT[locale]
  const text = t.playerPanel
  const name = cardName(player, t)
  const scoreTimeline = useMemo(() => scoreTimelineOf(doc), [doc])
  const live = playerCountersAt(scoreTimeline, player.xuid, frame)
  const sheet = useMemo(() => playerPanelSheet(player.board), [player.board])
  const ally = allegiance.ofXuid(player.xuid)
  const accent = ally === null ? null : teamTokenCssVar(ally)
  const ages = useReadingAges(doc)
  const clock = formatClock(displayClockMs(nowMs, playWindow))
  const caption = live ? text.atFmt(clock) : text.totals

  return (
    <>
      <div className="flex items-center gap-2">
        <div
          className={`${HUD_BAND_CLASS} flex min-w-0 flex-1 items-center gap-2 text-foreground`}
          style={hudBandStyle(accent)}
        >
          <span className="min-w-0 truncate" title={name}>
            {name}
          </span>
          <ReplayCountersBadge board={player.board} live={live} locale={locale} gabarit={GABARIT_NORMAL} />
          <span className="shrink-0 font-sans text-[10.5px] font-normal normal-case tracking-normal text-muted-foreground">
            {caption}
          </span>
        </div>
        {headEnd}
      </div>
      <div className="flex gap-1" role="group" aria-label={name}>
        {ACTIONS.map((action) => (
          <Button
            key={action}
            type="button"
            size="sm"
            variant={actions[action] ? 'default' : 'outline'}
            className="flex-1"
            aria-pressed={actions[action]}
            title={text.actionHint[action]}
            {...PRESS_GUARDS}
            onClick={() => onToggle(action)}
          >
            {text.action[action]}
          </Button>
        ))}
      </div>
      <div className={LOWER_CLASS[scroll]}>
        <ReplayPlayerNow player={player} doc={doc} frame={frame} clock={clock} ages={ages} locale={locale} />
        {/* SANS LIGNE DE FEUILLE, OU AVEC UNE FEUILLE ENTIÈREMENT NON LUE : le bloc pointillé, jamais
            une liste de zéros. */}
        {sheet.sections.length > 0 ? (
          sheet.sections.map((s) => <PanelSectionList key={s.id} section={s} text={text} locale={locale} />)
        ) : (
          <EmptyStateNotice size="compact" title={text.noSheet} description={text.noSheetHint} />
        )}
      </div>
    </>
  )
}
