/**
 * ReplayStandardFrame — LE CONTENU DES ZONES DU CADRE STANDARD : l'en-tête en trois cellules (dont le
 * bandeau de score, qui ne coiffe le terrain que si le corps est trop étroit pour la cellule) et la colonne des équipes et du fil. La scène et les avis
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
 * soit cinq lignes. LA FICHE D'UN JOUEUR REMPLACE TOUTE LA COLONNE (`ReplayFeedOrPanel`, `above`) : les équipes et
 * le fil restent montés dessous, masqués, pour retrouver leur défilement et le filtre du fil au retour ; la
 * ligne du joueur dont la fiche est ouverte est marquée. La fiche reçoit son sélecteur de joueurs et le pied
 * « Éliminations », qui lisent le MÊME fil que la colonne (`feedSource`). La cellule du fil occupe le même rang
 * dans la colonne que dans le Cinéma : elle survit au changement de cadre.
 *
 * LA COLONNE EST `absolute inset-0` à partir de `xl` : la carte impose la hauteur, jamais un pourcentage
 * d'elle ni une formule CSS (`ui/rosterHeight.guard.test.ts`). Sous `xl` elle passe sous la carte, bornée à
 * 80 % de la hauteur d'écran. Elle est toujours positionnée (`relative` sous `xl`) : la fiche s'y place.
 */
import { COTES_COLONNE_STANDARD, FIL_STANDARD_HAUTEUR_MIN_PX } from '../../model/cotesStandard'
import { ReplayFeedExtract } from '../ReplayFeedExtract'
import { ReplayFeedOrPanel } from '../ReplayFeedOrPanel'
import { ReplayKillFeed } from '../ReplayKillFeed'
import { ReplayPlayerPanel } from '../ReplayPlayerPanel'
import { ReplayPlayerPicker } from '../ReplayPlayerPicker'
import { ReplayRosterColumn } from '../ReplayRosterColumn'
import { ReplayScoreBanner } from '../ReplayScoreBanner'
import type { CardRowLayout } from '../../hooks/useCardRowLayout'
import { ReplayFrameHeader } from './ReplayFrameHeader'
import type { ReplayFrameChrome, ReplayFrameView, ReplayFrameZoneContent } from './replayFrameTypes'

const COLUMN_STYLE = Object.freeze({
  padding: `${COTES_COLONNE_STANDARD.margeVerticalePx}px ${COTES_COLONNE_STANDARD.margeLateralePx}px`,
  gap: COTES_COLONNE_STANDARD.ecartBlocsPx,
})

/** La colonne du Standard : les équipes, puis le fil ; la fiche d'un joueur les remplace toutes les deux. */
function standardColumn(view: ReplayFrameView) {
  const { doc, model, playerPanel, playFrame, nowMs, locale } = view
  // Le fil de la colonne et l'extrait de la fiche lisent la même source.
  const feedSource = {
    entries: model.feed,
    nowMs,
    playWindow: model.window,
    scoreboard: model.scoreboard,
    xuidMeta: model.identity,
    allegiance: model.allegiance,
    locale,
    marks: model.marks,
    zoneOf: view.feedZoneOf,
    doc,
    playbackStore: view.playbackStore,
    weaponIcons: view.weaponIcons,
  }
  const focus = playerPanel.focus
  return (
    <div
      className="relative flex max-h-[80vh] flex-col xl:absolute xl:inset-0 xl:max-h-none xl:border-l xl:border-border"
      style={COLUMN_STYLE}
      data-replay-column="standard"
    >
      {/* La place des équipes du Cinéma reste vide : la cellule du fil garde sa position, donc son instance, d'un cadre à l'autre. */}
      {null}
      <ReplayFeedOrPanel
        minHeightPx={FIL_STANDARD_HAUTEUR_MIN_PX}
        above={
          <ReplayRosterColumn
            doc={doc}
            players={model.players}
            frame={playFrame}
            locale={locale}
            allegiance={model.allegiance}
            tracked={playerPanel.tracked}
            onCard={playerPanel.onCard}
            selectedXuid={focus?.player.xuid ?? null}
          />
        }
        feed={<ReplayKillFeed {...feedSource} />}
        panel={
          focus && (
            <ReplayPlayerPanel
              player={focus.player}
              actions={focus}
              onToggle={playerPanel.onToggle}
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
              column={{
                picker: (
                  <ReplayPlayerPicker
                    players={model.players}
                    frame={playFrame}
                    locale={locale}
                    allegiance={model.allegiance}
                    selectedXuid={focus.player.xuid}
                    onCard={playerPanel.onCard}
                  />
                ),
                footer: (close) => <ReplayFeedExtract {...feedSource} player={focus.player} onFullFeed={close} />,
              }}
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

/**
 * Les zones du Standard. Le corps est mesuré comme celui du Cinéma (`bodyRef`) : sous la place de la cellule du
 * score (`emplacementScoreDe`), le score coiffe la toile et l'en-tête tient sur une ligne, sans cellule de score.
 */
export function standardFrameZones(
  chrome: ReplayFrameChrome,
  view: ReplayFrameView | null,
  row: CardRowLayout,
): ReplayFrameZoneContent {
  return {
    tete: (
      <ReplayFrameHeader
        frame="standard"
        chrome={chrome}
        score={row.scorePlacement === 'entete' ? standardScore(view) : undefined}
      />
    ),
    rangee: null,
    colonne: view && standardColumn(view),
    bodyRef: row.bodyRef,
  }
}
