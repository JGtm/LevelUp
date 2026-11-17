/**
 * ReplayStandardFrame — LE CONTENU DES ZONES DU CADRE STANDARD : l'en-tête en trois cellules (dont le
 * bandeau de score, qui ne coiffe plus le terrain) et la colonne des fiches et du fil. La scène et les avis
 * sont ceux de la page, communs aux trois cadres ; la zone `rangee` reste vide (la rangée de fiches
 * appartient au Cinéma et au Plein cadre).
 *
 * # LA COLONNE : FICHES AU-DESSUS, FIL EN DESSOUS, MÊME LARGEUR
 *
 * Un rejeu se lit en balayant du terrain vers les joueurs, puis vers l'événement. Les fiches gardent leur
 * hauteur naturelle (bornée, elles défilent au-delà : BTB) ; le fil remplit tout le reste et défile dedans.
 * Le panneau de joueur prend la place du fil sans le détruire (`ReplayFeedOrPanel`).
 *
 * LE PLAFOND DES FICHES EST EN PIXELS, borné par la place du fil : `xl:max-h-[30rem]` pour les fiches et le
 * plancher du fil dans sa cellule. Jamais un pourcentage de la carte : la hauteur du plan est élastique, et
 * un pourcentage tronquerait un 4v4 sur un écran contraint (`ui/rosterHeight.guard.test.ts`).
 */
import { ReplayFeedOrPanel } from '../ReplayFeedOrPanel'
import { ReplayKillFeed } from '../ReplayKillFeed'
import { ReplayPlayerPanel } from '../ReplayPlayerPanel'
import { ReplayScoreBanner } from '../ReplayScoreBanner'
import { ReplayTeams } from '../ReplayTeams'
import { ReplayFrameHeader } from './ReplayFrameHeader'
import type { ReplayFrameChrome, ReplayFrameView, ReplayFrameZoneContent } from './replayFrameTypes'

/** La colonne du Standard. Sous `xl` elle passe sous le plan, chaque bloc borné à 60 % de la hauteur d'écran. */
function standardColumn(view: ReplayFrameView) {
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

/** Le bandeau de la cellule centrale de l'en-tête ; absent tant que le rejeu n'est pas prêt. */
function standardScore(view: ReplayFrameView | null) {
  if (!view) return null
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

export function standardFrameZones(chrome: ReplayFrameChrome, view: ReplayFrameView | null): ReplayFrameZoneContent {
  return {
    tete: <ReplayFrameHeader frame="standard" chrome={chrome} score={standardScore(view)} />,
    rangee: null,
    colonne: view && standardColumn(view),
  }
}
