/**
 * ReplayCinemaFrame — LE CONTENU DES ZONES DU CINÉMA ET DU PLEIN CADRE, qui partagent la même structure :
 * l'en-tête du cadre, la zone `rangee` et la colonne. La scène et les avis sont ceux de la page.
 *
 * # DEUX DISPOSITIONS DES ÉQUIPES, UNE SEULE DÉCISION (`useCardRowLayout`, règle pure `dispositionEquipes`)
 *
 *  - RANGÉE posée (deux camps, la largeur du corps suffit) : la zone `rangee` porte `ReplayTeamsRow`, le score
 *    est au centre de la rangée (ni sur la toile ni dans l'en-tête), et le fil des éliminations est posé SUR LE
 *    PLAN (`ReplayFeedOverlay`, emplacement de la toile). La colonne n'est JAMAIS affichée : le plan prend sa
 *    largeur, et le fil complet y reste monté, caché, avec sa position ;
 *  - COLONNE (chacun pour soi, fenêtre trop étroite, largeur pas encore mesurée) : la zone `rangee` est vide,
 *    le score est dans l'EN-TÊTE (cellule de 440 px, comme le Standard ; sur la toile seulement si le corps est
 *    trop étroit pour l'y loger), et la colonne range les équipes en lignes (`ReplayRosterColumn`) au-dessus du
 *    fil, comme le Standard.
 * LA FICHE D'UN JOUEUR NE PREND JAMAIS LA COLONNE : dans les deux dispositions un clic sur une fiche de la rangée,
 * ou sur une ligne de la colonne, ouvre `ReplayPlayerPopover`, une fenêtre ancrée sous elle qui laisse le plan,
 * le fil et la barre utilisables. La colonne est à droite du plan (sous lui sous `xl`), sa hauteur est celle du
 * plan ; dans le Plein cadre elle descend jusqu'au bas du cadre. Passer d'une disposition à l'autre ne remonte ni
 * la toile ni la cellule du fil : elles gardent leur place dans l'arbre.
 */
import type { ReactNode } from 'react'

import type { CardRowLayout } from '../../hooks/useCardRowLayout'
import { playerKeys } from '../../model/feedFollowed'
import type { ReplayFrame } from '../../model/replayFrame'
import { ReplayFeedOrPanel } from '../ReplayFeedOrPanel'
import { ReplayKillFeed } from '../ReplayKillFeed'
import { ReplayPlayerPopover } from '../ReplayPlayerPopover'
import { ReplayRosterColumn } from '../ReplayRosterColumn'
import { ReplayScoreBanner } from '../ReplayScoreBanner'
import { ReplayTeamsRow } from '../ReplayTeamsRow'
import { ficheViewpointOf } from '../fiche/ficheViewpoint'
import { ReplayFrameHeader } from './ReplayFrameHeader'
import type { ReplayFrameChrome, ReplayFrameView, ReplayFrameZoneContent } from './replayFrameTypes'

/**
 * La colonne du Cinéma et du Plein cadre : les équipes au-dessus (plafond en pixels) quand la rangée n'est pas
 * posée, puis le fil. Le bloc des équipes absent laisse sa place vide dans l'arbre : la cellule du fil garde sa
 * position, donc son instance. La fiche du joueur n'est pas ici : c'est une fenêtre (`playerPopover`).
 */
function cinemaColumn(view: ReplayFrameView, withTeams: boolean, popover: ReactNode) {
  const { doc, model, playerPanel, playFrame, nowMs, locale } = view
  return (
    <div className="flex max-h-[80vh] flex-col gap-3 xl:absolute xl:inset-0 xl:max-h-none">
      {withTeams && (
        <div className="flex max-h-[60vh] min-h-0 flex-col overflow-hidden xl:max-h-[30rem]">
          <ReplayRosterColumn
            doc={doc}
            players={model.players}
            frame={playFrame}
            locale={locale}
            allegiance={model.allegiance}
            tracked={playerPanel.tracked}
            onCard={playerPanel.onCard}
            selectedXuid={playerPanel.focus?.player.xuid ?? null}
            tileActions={playerPanel.tileActions}
          />
        </div>
      )}
      <ReplayFeedOrPanel
        feed={
          <ReplayKillFeed
            entries={model.feed}
            nowMs={nowMs}
            playWindow={model.window}
            scoreboard={model.scoreboard}
            xuidMeta={model.identity}
            allegiance={model.allegiance}
            locale={locale}
            marks={model.marks}
            zoneOf={view.feedZoneOf}
            doc={doc}
            playbackStore={view.playbackStore}
            weaponIcons={view.weaponIcons}
            followedKeys={playerKeys(playerPanel.camera.player)}
          />
        }
      />
      {popover}
    </div>
  )
}

/**
 * La fiche du joueur ouverte, en fenêtre ancrée sous la fiche ou la ligne cliquée. La clé porte le joueur ET la
 * disposition : changer de joueur, ou passer de la rangée à la colonne (la fiche cliquée change de nœud), remonte la
 * fenêtre, qui se ré-ancre.
 */
function playerPopover(view: ReplayFrameView, disposition: CardRowLayout['disposition']): ReactNode {
  const { doc, model, playerPanel, playFrame, nowMs, locale } = view
  const focus = playerPanel.focus
  if (!focus) return null
  return (
    <ReplayPlayerPopover
      key={`${focus.player.xuid}:${disposition}`}
      player={focus.player}
      solo={focus.solo}
      onSolo={() => playerPanel.onSolo(focus.player.xuid)}
      viewpoint={ficheViewpointOf(focus.player, view.viewpoint)}
      onClose={playerPanel.onClose}
      doc={doc}
      frame={playFrame}
      nowMs={nowMs}
      playWindow={model.window}
      allegiance={model.allegiance}
      locale={locale}
      players={model.players}
      feed={model.feed}
      weaponIcons={view.weaponIcons}
      antagonists={view.antagonists}
    />
  )
}

/** La rangée des équipes : le score au centre (emplacement `rangee`), les fiches de chaque camp de part et d'autre. */
function teamsRow(view: ReplayFrameView, grandeEquipe: boolean) {
  const { doc, model, playerPanel, playFrame, nowMs, locale } = view
  return (
    <ReplayTeamsRow
      doc={doc}
      players={model.players}
      frame={playFrame}
      locale={locale}
      allegiance={model.allegiance}
      tracked={playerPanel.tracked}
      onCard={playerPanel.onCard}
      selectedXuid={playerPanel.focus?.player.xuid ?? null}
      tileActions={playerPanel.tileActions}
      grandeEquipe={grandeEquipe}
      score={
        <ReplayScoreBanner
          doc={doc}
          allegiance={model.allegiance}
          frame={playFrame}
          nowMs={nowMs}
          playWindow={model.window}
          locale={locale}
          placement="rangee"
        />
      }
    />
  )
}

/** Le bandeau de score de l'en-tête (cellule centrale) ; `undefined` : l'en-tête n'a pas de cellule de score. */
function headerScore(view: ReplayFrameView | null, row: CardRowLayout): ReactNode {
  if (!view || row.scorePlacement !== 'entete') return undefined
  return (
    <ReplayScoreBanner
      doc={view.doc}
      allegiance={view.model.allegiance}
      frame={view.playFrame}
      nowMs={view.nowMs}
      playWindow={view.model.window}
      locale={view.locale}
      placement="entete"
    />
  )
}

export function cinemaFrameZones(
  frame: Exclude<ReplayFrame, 'standard'>,
  chrome: ReplayFrameChrome,
  view: ReplayFrameView | null,
  row: CardRowLayout,
): ReplayFrameZoneContent {
  const inRow = row.disposition !== 'colonne'
  return {
    tete: <ReplayFrameHeader frame={frame} chrome={chrome} score={headerScore(view, row)} />,
    rangee: view && inRow ? teamsRow(view, row.disposition === 'rangee-grande-equipe') : null,
    colonne: view && cinemaColumn(view, !inRow, playerPopover(view, row.disposition)),
    colonneCachee: row.planFeedPosed,
    bodyRef: row.bodyRef,
  }
}
