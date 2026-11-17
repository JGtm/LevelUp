/**
 * ReplayCinemaFrame — LE CONTENU DES ZONES DU CINÉMA ET DU PLEIN CADRE, qui partagent la même structure :
 * l'en-tête du cadre, la zone `rangee` et la colonne. La scène et les avis sont ceux de la page.
 *
 * # DEUX DISPOSITIONS DES ÉQUIPES, UNE SEULE DÉCISION (`useCardRowLayout`, règle pure `dispositionEquipes`)
 *
 *  - RANGÉE posée (deux camps, la largeur du corps suffit) : la zone `rangee` porte `ReplayTeamsRow`, le score
 *    est au centre de la rangée et PLUS sur la toile, et le fil des éliminations est posé SUR LE PLAN
 *    (`ReplayFeedOverlay`, emplacement de la toile). La colonne ne s'affiche alors que lorsqu'une fiche de joueur
 *    est ouverte, et le plan prend sa largeur sinon ; le fil complet y reste monté, caché, avec sa position ;
 *  - COLONNE (chacun pour soi, fenêtre trop étroite, largeur pas encore mesurée) : la zone `rangee` est vide,
 *    le score reste sur la toile, et la colonne range les équipes en lignes (`ReplayRosterColumn`) au-dessus
 *    du fil, comme le Standard.
 * La colonne est à droite du plan (sous lui sous `xl`), sa hauteur est celle du plan ; dans le Plein cadre elle
 * descend jusqu'au bas du cadre. Passer d'une disposition à l'autre ne remonte ni la toile ni la cellule du fil :
 * elles gardent leur place dans l'arbre.
 */
import type { CardRowLayout } from '../../hooks/useCardRowLayout'
import type { ReplayFrame } from '../../model/replayFrame'
import { ReplayFeedOrPanel } from '../ReplayFeedOrPanel'
import { ReplayKillFeed } from '../ReplayKillFeed'
import { ReplayPlayerPanel } from '../ReplayPlayerPanel'
import { ReplayRosterColumn } from '../ReplayRosterColumn'
import { ReplayScoreBanner } from '../ReplayScoreBanner'
import { ReplayTeamsRow } from '../ReplayTeamsRow'
import { ReplayFrameHeader } from './ReplayFrameHeader'
import type { ReplayFrameChrome, ReplayFrameView, ReplayFrameZoneContent } from './replayFrameTypes'

/**
 * La colonne du Cinéma et du Plein cadre : les équipes au-dessus (plafond en pixels) quand la rangée n'est pas
 * posée, puis le fil ou le panneau de joueur. Le bloc des équipes absent laisse sa place vide dans l'arbre : la
 * cellule du fil garde sa position, donc son instance.
 */
function cinemaColumn(view: ReplayFrameView, withTeams: boolean) {
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
          />
        }
        panel={
          playerPanel.focus && (
            <ReplayPlayerPanel
              player={playerPanel.focus.player}
              actions={playerPanel.focus}
              onToggle={playerPanel.onToggle}
              onClose={playerPanel.onClose}
              doc={doc}
              frame={playFrame}
              nowMs={nowMs}
              playWindow={model.window}
              allegiance={model.allegiance}
              locale={locale}
            />
          )
        }
      />
    </div>
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

export function cinemaFrameZones(
  frame: Exclude<ReplayFrame, 'standard'>,
  chrome: ReplayFrameChrome,
  view: ReplayFrameView | null,
  row: CardRowLayout,
): ReplayFrameZoneContent {
  const inRow = row.disposition !== 'colonne'
  return {
    tete: <ReplayFrameHeader frame={frame} chrome={chrome} />,
    rangee: view && inRow ? teamsRow(view, row.disposition === 'rangee-grande-equipe') : null,
    colonne: view && cinemaColumn(view, !inRow),
    colonneCachee: row.planFeedPosed && !view?.playerPanel.focus,
    bodyRef: row.bodyRef,
  }
}
