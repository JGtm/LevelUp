/**
 * ReplayPlayerPanel — LE PANNEAU DE JOUEUR : ce qu'un clic sur une fiche ouvre, à la PLACE du fil
 * des événements (la carte n'est jamais couverte). « Retour au fil » le referme.
 *
 * DE HAUT EN BAS :
 *  - la TÊTE : le nom à l'encre de son camp (l'allégeance du film), les compteurs frags / morts /
 *    assistances AU FORMAT DE LA FICHE (`ReplayCountersBadge`, ceux de l'instant — datés par
 *    l'horloge du match —, ou les totaux de la feuille quand le film ne publie pas ce joueur), et
 *    « Retour au fil » ;
 *  - les TROIS ACTIONS sur la carte — Suivre, Seul, Masquer — boutons à bascule (`aria-pressed`) ;
 *    leurs exclusions et leur remise à zéro sont dans `playerPanelLogic` ;
 *  - les LISTES : précision, dégâts, détail du score, objectif, armes. Ce sont des TOTAUX DE FIN DE
 *    MATCH lus sur la feuille, et chaque titre dit « sur le match » : le film ne publie pas la
 *    précision en cours de partie. Un joueur sans ligne de feuille n'a pas de listes : un bloc
 *    pointillé le dit, plutôt que des zéros.
 *
 * LE PANNEAU NE DÉTRUIT PAS LE FIL : la page garde le fil monté sous lui (masqué), si bien que le
 * filtre et le défilement sont retrouvés au retour. Il garde son joueur quand le temps avance.
 * Ses boutons suivent la règle des boutons du rejeu : la souris ne prend pas le focus, Espace reste
 * « Lecture » (`PRESS_GUARDS`).
 *
 * ÉCHAP LE FERME, AVANT LE MODE PLEIN ÉCRAN : le panneau est une couche (`data-replay-escape-layer`,
 * cf. `useReplayFullscreen`), et rend le focus à la fiche du joueur, qu'il quitte par Échap ou par
 * « Retour au fil ». Une couche plus intérieure ouverte (menu de vitesse, de filtre, panneau de la
 * vue) se ferme d'abord : Échap ne ferme qu'une couche à la fois.
 *
 * Aucun littéral de couleur : l'encre du camp et les neutres sont des jetons sémantiques.
 */
import { useCallback, useEffect, useMemo, useRef, type RefObject } from 'react'

import { Button } from '@/components/ui/button'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { teamTokenCssVar } from '@/features/match-view/teamSeriesColor'
import type { FilmAllegiance } from '@/lib/replay/filmAllegiance'
import { formatClock } from '@/lib/replay/replayLogic'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'
import { playerCountersAt, scoreTimelineOf } from '@/lib/replay/scoreTimeline'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { GABARIT_NORMAL } from '../model/cardGabarit'
import { HUD_BAND_CLASS, hudBandStyle } from '../model/hudBand'
import { cardName } from '../model/playerCardReadings'
import { playerPanelSheet, type PlayerPanelAction } from '../model/playerPanelLogic'
import { displayClockMs, type ReplayWindowBounds } from '../model/replayWindow'
import { ReplayCountersBadge } from './ReplayCountersBadge'
import { PanelSectionList } from './ReplayPlayerPanelList'
import { PRESS_GUARDS } from './replayPressGuards'

/** L'ordre des trois actions : celui de la maquette. */
const ACTIONS: readonly PlayerPanelAction[] = ['follow', 'solo', 'hide']

export interface ReplayPlayerPanelProps {
  player: ReplayPlayer
  /** Les trois actions et leur état : l'état est tenu par la page (`reducePlayerPanel`). */
  actions: Record<PlayerPanelAction, boolean>
  onToggle: (action: PlayerPanelAction) => void
  onClose: () => void
  doc: ReplayDocumentReady
  /** L'image publiée (les compteurs de la tête sont ceux de cet instant). */
  frame: number
  /** L'instant courant sur l'axe BRUT du film, et la fenêtre qui le recale pour l'affichage. */
  nowMs: number
  playWindow: ReplayWindowBounds | null
  allegiance: FilmAllegiance
  locale: ReplayLocale
}

/** La fiche du joueur dont le panneau est ouvert : la tuile-bouton choisie (`aria-pressed`). */
const OPEN_CARD_SELECTOR = '[data-replay-player-card][aria-pressed="true"]'

/** Une AUTRE couche d'Échap est-elle ouverte que ce panneau ? Elle se ferme avant lui. */
function innerLayerOpen(self: Element | null): boolean {
  return Array.from(document.querySelectorAll('[data-replay-escape-layer]')).some((layer) => layer !== self)
}

/**
 * usePanelClose — LA SORTIE DU PANNEAU : « Retour au fil » et Échap ferment par le même chemin, qui
 * rend le focus à la fiche (sans quoi il tomberait sur la page avec le panneau retiré). Échap ne
 * ferme que si aucune AUTRE couche d'Échap n'est ouverte : elle se ferme la première, par son
 * propre écouteur.
 */
function usePanelClose(rootRef: RefObject<HTMLElement | null>, onClose: () => void): () => void {
  const close = useCallback(() => {
    const card = document.querySelector<HTMLElement>(OPEN_CARD_SELECTOR)
    onClose()
    card?.focus({ preventScroll: true })
  }, [onClose])
  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape' && !innerLayerOpen(rootRef.current)) close()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [rootRef, close])
  return close
}

export function ReplayPlayerPanel({
  player, actions, onToggle, onClose, doc, frame, nowMs, playWindow, allegiance, locale,
}: ReplayPlayerPanelProps) {
  const t = REPLAY_TEXT[locale]
  const rootRef = useRef<HTMLElement>(null)
  const close = usePanelClose(rootRef, onClose)
  const text = t.playerPanel
  const name = cardName(player, t)
  const scoreTimeline = useMemo(() => scoreTimelineOf(doc), [doc])
  const live = playerCountersAt(scoreTimeline, player.xuid, frame)
  const sheet = useMemo(() => playerPanelSheet(player.board), [player.board])
  const ally = allegiance.ofXuid(player.xuid)
  const accent = ally === null ? null : teamTokenCssVar(ally)
  const caption = live ? text.atFmt(formatClock(displayClockMs(nowMs, playWindow))) : text.totals

  return (
    <section
      ref={rootRef}
      // UNE COUCHE QU'ÉCHAP FERME AVANT LE MODE PLEIN ÉCRAN (cf. useReplayFullscreen).
      data-replay-escape-layer=""
      aria-label={text.labelFmt(name)}
      className="flex h-full min-h-0 flex-col gap-2 overflow-hidden rounded-lg border border-border bg-card p-2"
    >
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
        <Button type="button" variant="outline" size="sm" className="shrink-0" {...PRESS_GUARDS} onClick={close}>
          {text.back}
        </Button>
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
      <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
        {/* SANS LIGNE DE FEUILLE, OU AVEC UNE FEUILLE ENTIÈREMENT NON LUE : le bloc pointillé, jamais
            une liste de zéros. */}
        {sheet.sections.length > 0 ? (
          sheet.sections.map((s) => <PanelSectionList key={s.id} section={s} text={text} locale={locale} />)
        ) : (
          <EmptyStateNotice size="compact" title={text.noSheet} description={text.noSheetHint} />
        )}
      </div>
    </section>
  )
}
