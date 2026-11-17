/**
 * ReplayCinemaFrame — LE CONTENU DES ZONES DU CINÉMA ET DU PLEIN CADRE, qui partagent la même structure :
 * l'en-tête du cadre, la zone `rangee` et la colonne. La scène et les avis sont ceux de la page.
 *
 * LA ZONE `rangee` EST VIDE tant que la rangée de fiches n'existe pas : le Cinéma et le Plein cadre
 * n'affichent donc, pour l'instant, que le plan.
 *
 * LES ÉQUIPES ET LE FIL gardent une place lisible dans la zone `colonne`, à droite du plan (sous lui
 * sous `xl`), comme aujourd'hui : fiches au-dessus, fil ou panneau de joueur dessous, la hauteur de la
 * colonne étant celle du plan. Dans le Plein cadre la colonne descend jusqu'au bas du cadre.
 */
import type { ReplayFrame } from '../../model/replayFrame'
import { ReplayFeedOrPanel } from '../ReplayFeedOrPanel'
import { ReplayKillFeed } from '../ReplayKillFeed'
import { ReplayPlayerPanel } from '../ReplayPlayerPanel'
import { ReplayTeams } from '../ReplayTeams'
import { ReplayFrameHeader } from './ReplayFrameHeader'
import type { ReplayFrameChrome, ReplayFrameView, ReplayFrameZoneContent } from './replayFrameTypes'

/** La colonne du Cinéma et du Plein cadre : fiches au-dessus (plafond en pixels), fil ou panneau dessous. */
function cinemaColumn(view: ReplayFrameView) {
  const { doc, model, playerPanel, header, playFrame, nowMs, locale } = view
  return (
    <div className="flex max-h-[80vh] flex-col gap-3 xl:absolute xl:inset-0 xl:max-h-none">
      <div className="flex max-h-[60vh] min-h-0 flex-col overflow-hidden xl:max-h-[30rem]">
        <ReplayTeams
          doc={doc}
          players={model.players}
          frame={playFrame}
          locale={locale}
          allegiance={model.allegiance}
          header={header}
          panel={{ xuid: playerPanel.focus?.player.xuid ?? null, onSelect: playerPanel.onCard }}
        />
      </div>
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

export function cinemaFrameZones(
  frame: Exclude<ReplayFrame, 'standard'>,
  chrome: ReplayFrameChrome,
  view: ReplayFrameView | null,
): ReplayFrameZoneContent {
  return {
    tete: <ReplayFrameHeader frame={frame} chrome={chrome} />,
    rangee: null,
    colonne: view && cinemaColumn(view),
  }
}
