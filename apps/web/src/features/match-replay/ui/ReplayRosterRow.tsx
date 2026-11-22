/**
 * ReplayRosterRow — UNE LIGNE PAR PLACE dans la colonne des équipes : le nom, le bouclier et la santé en
 * deux barres, l'arme en main, les grenades, l'équipement, et les frags / morts / assistances. Un
 * joueur mort écrit « Réapparition dans … » à la place des barres. La ligne du joueur SUIVI se
 * déplie (`ReplayRosterRowDetail`).
 *
 * LA LIGNE NE LIT RIEN ET NE CALCULE RIEN : `playerCardReadings` relit, comme pour la fiche, l'état,
 * les armes et les effets ; `rowInventory` les grenades et l'équipement ; `countersShown` le triplet.
 * Les âges de lecture et la scène des effets viennent de `useCardScene`, construits une fois par la
 * colonne. LES RÈGLES DES FICHES VALENT ICI : une valeur non lue n'est pas rendue (cellule vide), une
 * lecture ancienne pâlit et dit son âge, aucun littéral de couleur.
 *
 * LES EFFETS DE LA FICHE SONT GARDÉS, à la hauteur de la ligne : les couches posées sous le contenu
 * (éclat de mort et de réapparition, verre, voiles, cadres, filigrane de porteur) et l'incrustation
 * (nuage, éclairs, capteur, croix, fourreau de translocation) sont celles de `ReplayCardFx`, au rayon
 * de la ligne ; l'infobulle de la ligne dit les états actifs. Le joueur mort s'éteint (nom, cellules
 * et triplet).
 *
 * UNE PLACE SANS FICHE (libre, ou dont l'occupant n'est pas encore apparu) est une ligne d'attente de
 * même hauteur : l'habit tireté, le nom estompé, l'état écrit à la place des barres
 * (`seatWaitingReading`, la règle de la tuile d'attente).
 *
 * LES DEUX ACTIONS (D-6, `ReplayTileActionButtons`) : quand la page les donne (`actions`), la ligne porte Suivre et
 * l'œil à droite de la cellule du nom — au survol ou au focus de la ligne (elle porte `group`), toujours visibles
 * actifs. La ligne d'un joueur MASQUÉ écrit « masqué » à côté de son nom, pâli.
 *
 * UN CLIC REND LA PAGE (`onCard`) : la ligne est un bouton, au même contrat que la fiche
 * (`data-replay-player-card`, gardes d'appui `PRESS_GUARDS`).
 */
import { Fragment, type CSSProperties, type ReactNode } from 'react'

import { tokenCssVar, type SemanticToken } from '@/lib/accessibility/semantic-tokens'

import { CardUnderLayers, ZoneFxOverlay } from './ReplayCardFx'
import { EquipmentCell, GrenadeCell, WeaponCell } from './ReplayRosterCells'
import { ReplayTileActionButtons } from './ReplayTileActionButtons'
import { ReplayRosterRowDetail } from './ReplayRosterRowDetail'
import { VitalityBar } from './ReplayVitality'
import { PRESS_CLASS, pressProps, selectedChrome } from './replayCardPress'
import type { CardScene } from '../hooks/useCardScene'
import { GABARIT_LIGNE as G } from '../model/cardGabarit'
import { SEAT_WAITING_CHROME } from '../model/playerCardFx'
import type { TileActions } from '../model/playerTileActions'
import { countersShown, playerCardReadings, seatWaitingReading, type PlayerCardReadings } from '../model/playerCardReadings'
import { respawnText, rowInventory } from '../model/rosterRowReadings'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'

type ReplayText = (typeof REPLAY_TEXT)[ReplayLocale]

/** Les colonnes d'une ligne : nom et jauges (souple), arme, grenades, équipement, triplet. */
export const ROW_COLUMNS = `minmax(0, 1fr) ${G.weaponColW}px ${G.grenadeColW}px ${G.equipmentColW}px ${G.kdaColW}px`

/**
 * La bordure TRANSPARENTE d'une ligne vivante a la même épaisseur que la bordure tiretée d'une ligne
 * d'attente : changer d'état ne décale ni le nom ni les cellules.
 */
const ROW_BASE = 'relative border border-transparent'

export interface ReplayRosterRowProps {
  player: ReplayPlayer
  doc: ReplayDocumentReady
  frame: number
  locale: ReplayLocale
  /** Âges de lecture, scène des effets et calque de score, construits une fois par la colonne (`useCardScene`). */
  scene: CardScene
  /** Vrai = le joueur suivi (le joueur regardé par la page) : nom en gras, mention, ligne dépliée. */
  tracked: boolean
  /** Vrai = la fiche de ce joueur est ouverte (`aria-pressed`, anneau). */
  selected?: boolean
  /** Un clic sur la ligne, avec la clé de roster du joueur. Absent = ligne inerte. */
  onCard?: (xuid: string) => void
  /** Suivre et l'œil de CE joueur (D-6) : absent = la ligne n'a pas ces boutons. */
  actions?: TileActions
}

/** Ce que la ligne écrit pour un joueur mort : le compte à rebours, à la couleur d'alerte. */
function RespawnLine({ text, title }: { text: string; title: string }) {
  return (
    <span
      className="truncate"
      style={{ fontSize: G.respawnPx, height: G.respawnLinePx, lineHeight: `${G.respawnLinePx}px`, color: tokenCssVar('destructive') }}
      title={title}
    >
      {text}
    </span>
  )
}

/** Les deux barres d'un joueur vivant, le bouclier au-dessus de la santé (l'ordre dans lequel le jeu les encaisse). */
function Gauges({ read, scene, t }: { read: PlayerCardReadings; scene: CardScene; t: ReplayText }) {
  const { state } = read
  if (!state.shield && !state.health) return null
  return (
    <div className="flex flex-col" style={{ width: G.gaugeWidthPx, maxWidth: '100%', gap: G.gaugeGapPx }}>
      <VitalityBar reading={state.shield} fade={scene.vitalityFade} name={t.shieldLabel} token="info" heightPx={G.gaugeShieldPx} />
      <VitalityBar reading={state.health} fade={scene.vitalityFade} name={t.healthLabel} token="success" heightPx={G.gaugeHealthPx} />
    </div>
  )
}

/** La place des deux actions : à droite de la cellule du nom, à la hauteur de sa première ligne. */
const ACTIONS_PLACE = 'absolute right-0 top-0 z-[1] flex gap-0.5'

/** Le nom, la mention « suivi » du joueur regardé, puis les barres ou le compte à rebours ; les deux actions de la ligne. */
function NameCell({ read, tracked, scene, doc, frame, locale, actions }: {
  read: PlayerCardReadings
  tracked: boolean
  scene: CardScene
  doc: ReplayDocumentReady
  frame: number
  locale: ReplayLocale
  actions?: TileActions
}) {
  const { state, name } = read
  const t = REPLAY_TEXT[locale]
  return (
    <div className="relative flex min-w-0 flex-col" style={{ gap: G.nameGapPx }}>
      <b
        className={`truncate ${actions?.hidden ? 'text-muted-foreground' : 'text-foreground'} ${tracked ? 'font-bold' : 'font-medium'}`}
        style={{ fontSize: G.namePx, lineHeight: 1.35, opacity: state.alive ? 1 : G.deadOpacity }}
        title={name}
      >
        {name}
        {tracked && (
          <em className="ml-1.5 font-normal not-italic" style={{ fontSize: G.mentionPx, color: 'var(--ring)' }}>
            {t.roster.followed}
          </em>
        )}
        {actions?.hidden && (
          <em className="ml-1.5 font-normal not-italic text-muted-foreground" style={{ fontSize: G.mentionPx }}>
            {t.playerPanel.tile.hiddenMention}
          </em>
        )}
      </b>
      {actions && <ReplayTileActionButtons name={name} actions={actions} locale={locale} className={ACTIONS_PLACE} />}
      {state.alive ? (
        <Gauges read={read} scene={scene} t={t} />
      ) : (
        <RespawnLine text={respawnText(state, doc, frame, t)} title={state.departed ? t.presenceLeft : t.eliminatedLabel} />
      )}
    </div>
  )
}

/** Les frags, morts et assistances, chacun à la couleur de sa statistique ; la source dit laquelle des deux horloges on lit. */
function KdaCell({ player, read, locale }: { player: ReplayPlayer; read: PlayerCardReadings; locale: ReplayLocale }) {
  const t = REPLAY_TEXT[locale]
  const shown = countersShown(read.live, player.board)
  if (!shown) return <span aria-hidden />
  const parts: [number | null | undefined, SemanticToken][] = [
    [shown.kills, 'stat-kills'],
    [shown.deaths, 'stat-deaths'],
    [shown.assists, 'stat-assists'],
  ]
  return (
    <span
      className="text-right tabular-nums"
      style={{ fontSize: G.kdaPx, opacity: read.state.alive ? 1 : G.deadOpacity }}
      title={read.live ? t.countersLive : t.countersMatch}
    >
      {parts.map(([v, token], i) => (
        <Fragment key={token}>
          {i > 0 && <span className="mx-0.5 text-muted-foreground">/</span>}
          <span style={{ color: tokenCssVar(token) }}>{v ?? '?'}</span>
        </Fragment>
      ))}
    </span>
  )
}

/**
 * Les trois cellules d'inventaire d'un joueur VIVANT. Un joueur mort n'affirme plus rien : la fiche
 * ne montre alors que son encadré, et la ligne laisse ses colonnes vides (le triplet reste, éteint).
 */
function InventoryCells({ read, doc, frame, locale, scene }: {
  read: PlayerCardReadings
  doc: ReplayDocumentReady
  frame: number
  locale: ReplayLocale
  scene: CardScene
}): ReactNode {
  const { state, equipped } = read
  const slot = state.alive ? state.life?.slot : undefined
  if (slot === undefined) return <><span /><span /><span /></>
  const inventory = rowInventory(doc, locale, slot, frame)
  const base = { doc, locale, readingFull: scene.readingFull }
  return (
    <>
      <span className="flex justify-end">
        <WeaponCell {...base} frame={frame} slot={slot} equipped={equipped} />
      </span>
      <span className="flex justify-end">
        <GrenadeCell {...base} inventory={inventory} />
      </span>
      <span className="flex justify-end">
        <EquipmentCell {...base} inventory={inventory} />
      </span>
    </>
  )
}

/**
 * ReplayRosterRow — la ligne d'un joueur PRÉSENT, et son dépli quand c'est le joueur suivi.
 */
export function ReplayRosterRow({ player, doc, frame, locale, scene, tracked, selected = false, onCard, actions }: ReplayRosterRowProps) {
  const t = REPLAY_TEXT[locale]
  const read = playerCardReadings({
    player, doc, frame, presence: scene.presence, flashFrames: scene.flashFrames,
    scoreTimeline: scene.scoreTimeline, fxScene: scene.fxScene, text: t,
  })
  const { fx, objective, zones } = read
  const press = onCard
    ? { label: t.playerPanel.cardOpenFmt(read.name), selected, onActivate: () => onCard(player.xuid) }
    : undefined
  const className = `${ROW_BASE} ${G.layerRadius}${press ? ` ${PRESS_CLASS}` : ''}${actions ? ' group' : ''}`
  const style: CSSProperties = press && selected ? { height: G.rowPx, ...selectedChrome() } : { height: G.rowPx }
  return (
    <>
      <div className={className} style={style} title={fx.title} {...(press ? pressProps(press) : {})}>
        <CardUnderLayers fx={fx} objective={objective} radiusClass={G.layerRadius} watermarkPx={G.watermarkPx} />
        <div
          className="relative grid h-full items-center"
          style={{ gridTemplateColumns: ROW_COLUMNS, columnGap: G.colGapPx }}
        >
          <NameCell read={read} tracked={tracked} scene={scene} doc={doc} frame={frame} locale={locale} actions={actions} />
          <InventoryCells read={read} doc={doc} frame={frame} locale={locale} scene={scene} />
          <KdaCell player={player} read={read} locale={locale} />
        </div>
        <ZoneFxOverlay
          zones={zones}
          translocationDelay={fx.translocationDelay}
          boltCount={G.boltCount}
          radiusClass={G.layerRadius}
        />
      </div>
      {tracked && <ReplayRosterRowDetail player={player} doc={doc} frame={frame} locale={locale} scene={scene} />}
    </>
  )
}

/**
 * ReplayRosterWaitingRow — la ligne d'une place sans fiche, à la hauteur d'une ligne de joueur : le nom
 * de l'occupant qui la tient (`null` : place libre, aucun nom écrit) et son état à la place des barres.
 */
export function ReplayRosterWaitingRow({ occupant, locale }: { occupant: ReplayPlayer | null; locale: ReplayLocale }) {
  const { name, label, hint } = seatWaitingReading(occupant, REPLAY_TEXT[locale])
  return (
    <div
      className={`${ROW_BASE} ${G.layerRadius}`}
      style={{ height: G.rowPx, ...SEAT_WAITING_CHROME }}
      title={hint}
    >
      <div
        className="grid h-full items-center"
        style={{ gridTemplateColumns: ROW_COLUMNS, columnGap: G.colGapPx }}
      >
        <div className="flex min-w-0 flex-col" style={{ gap: G.nameGapPx }}>
          <b
            aria-hidden={name === null || undefined}
            className="truncate font-medium text-muted-foreground"
            style={{ fontSize: G.namePx, lineHeight: 1.35 }}
            title={name ?? undefined}
          >
            {name ?? ' '}
          </b>
          <span
            className="truncate uppercase tracking-wider text-muted-foreground"
            style={{ fontSize: G.respawnPx, height: G.respawnLinePx, lineHeight: `${G.respawnLinePx}px` }}
          >
            {label}
          </span>
        </div>
      </div>
    </div>
  )
}
