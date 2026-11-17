/**
 * ReplayStandardFrame — LE CONTENU DES ZONES DU CADRE STANDARD : l'en-tête en trois cellules (dont le
 * bandeau de score, qui ne coiffe plus le terrain) et la colonne des équipes et du fil. La scène et les avis
 * sont ceux de la page, communs aux trois cadres ; la zone `rangee` reste vide (la rangée de fiches
 * appartient au Cinéma et au Plein cadre) et rien ne se pose sous le lecteur.
 *
 * # LA COLONNE : LES ÉQUIPES AU-DESSUS, LE FIL EN DESSOUS
 *
 * Un rejeu se lit en balayant du terrain vers les joueurs, puis vers l'événement. La colonne a la largeur
 * de `COTES_COLONNE_STANDARD` (grille de `ReplayFrameLayout`), un filet neutre à sa gauche, sa marge
 * intérieure et l'écart entre ses deux blocs. Les équipes (`ReplayRosterColumn`) gardent leur hauteur
 * naturelle et c'est ELLES qui cèdent quand la place manque : leur part défile (une Grande équipe tient
 * sans défiler) ; le fil remplit tout le reste et ne descend jamais sous `FIL_STANDARD_HAUTEUR_MIN_PX`,
 * soit cinq lignes. La fiche d'un joueur prend la place du fil sans le détruire (`ReplayFeedOrPanel`) ; la
 * ligne du joueur dont la fiche est ouverte est marquée.
 *
 * LA COLONNE EST `absolute inset-0` à partir de `xl` : la carte impose la hauteur, jamais un pourcentage
 * d'elle ni une formule CSS (`ui/rosterHeight.guard.test.ts`). Sous `xl` elle passe sous la carte, bornée à
 * 80 % de la hauteur d'écran.
 */
import { COTES_COLONNE_STANDARD, FIL_STANDARD_HAUTEUR_MIN_PX } from '../../model/cotesStandard'
import { ReplayFeedOrPanel } from '../ReplayFeedOrPanel'
import { ReplayKillFeed } from '../ReplayKillFeed'
import { ReplayPlayerPanel } from '../ReplayPlayerPanel'
import { ReplayRosterColumn } from '../ReplayRosterColumn'
import { ReplayScoreBanner } from '../ReplayScoreBanner'
import { ReplayFrameHeader } from './ReplayFrameHeader'
import type { ReplayFrameChrome, ReplayFrameView, ReplayFrameZoneContent } from './replayFrameTypes'

const COLUMN_STYLE = Object.freeze({
  padding: `${COTES_COLONNE_STANDARD.margeVerticalePx}px ${COTES_COLONNE_STANDARD.margeLateralePx}px`,
  gap: COTES_COLONNE_STANDARD.ecartBlocsPx,
})

/** La colonne du Standard : les équipes, puis le fil (ou la fiche du joueur à sa place). */
function standardColumn(view: ReplayFrameView) {
  const { doc, model, playerPanel, playFrame, nowMs, locale } = view
  return (
    <div
      className="flex max-h-[80vh] flex-col xl:absolute xl:inset-0 xl:max-h-none xl:border-l xl:border-border"
      style={COLUMN_STYLE}
      data-replay-column="standard"
    >
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
      <ReplayFeedOrPanel
        minHeightPx={FIL_STANDARD_HAUTEUR_MIN_PX}
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
