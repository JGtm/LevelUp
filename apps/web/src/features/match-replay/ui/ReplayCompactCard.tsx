/**
 * ReplayCompactCard — LA FICHE COMPACTE d'une Grande équipe : une pastille à la couleur du camp, le nom
 * (points de suspension quand il est long), le bouclier au-dessus de la santé. Ni frags / morts /
 * assistances, ni arme, grenades ou équipement : ils sont dans la fiche du joueur, un clic plus loin.
 *
 * UNE FICHE, DEUX FORMES (`form`) : `'stacked'` sur deux lignes (le nom, puis les jauges) pour la rangée du
 * Cinéma ; `'line'` sur une ligne (pastille, nom, jauges étroites) pour la colonne du Standard, où les
 * fiches se rangent par paires. Les cotes sont `GABARIT_FICHE_COMPACTE`.
 *
 * LA FICHE NE LIT RIEN ET NE CALCULE RIEN : `playerCardReadings` relit, comme pour la fiche et la ligne,
 * l'état vital, l'objectif porté et les effets ; `respawnShort` le compte à rebours. Les âges de lecture
 * et la scène des effets viennent de `useCardScene`. LES RÈGLES DES FICHES VALENT ICI : une valeur non
 * lue n'est pas rendue, une lecture ancienne pâlit (`VitalityBar`), aucun littéral de couleur.
 *
 * LES EFFETS GARDÉS À CETTE TAILLE : la couche sous le contenu (éclat de mort et de réapparition, verre,
 * voiles, cadres) et l'incrustation de zone (nuage, éclairs, capteur, croix, fourreau), au rayon de la
 * fiche ; l'infobulle dit les états actifs. Le filigrane de porteur n'a pas de place : il devient la
 * marque à droite des jauges (le glyphe de l'objectif, la phrase de l'objectif en infobulle).
 *
 * UN JOUEUR ÉLIMINÉ s'éteint (le nom) et écrit « Réapp. 4,2 s » à la place des jauges, le texte long en
 * infobulle. Un BOT porte la pastille « bot » (prédicat du roster, `player.bot`). Le JOUEUR SUIVI a la
 * bordure à la couleur de suivi et le nom en gras. UNE PLACE EN ATTENTE est une fiche tiretée de même
 * boîte (`seatWaitingReading`). Un clic rend la page (`onCard`), au contrat des lignes
 * (`data-replay-player-card`, gardes d'appui).
 */
import type { CSSProperties, ReactNode } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { Allegiance } from '@/lib/replay/filmAllegiance'

import { CardUnderLayers, ZoneFxOverlay } from './ReplayCardFx'
import { ReplayObjectiveGlyph } from './ReplayObjectiveMark'
import { VitalityBar } from './ReplayVitality'
import { PRESS_CLASS, pressProps, SELECTED_CLASS } from './replayCardPress'
import type { CardScene } from '../hooks/useCardScene'
import { campInk } from '../model/campInk'
import { GABARIT_FICHE_COMPACTE as G } from '../model/cardGabarit'
import { cardChrome, SEAT_WAITING_CHROME } from '../model/playerCardFx'
import { playerCardReadings, seatWaitingReading, type PlayerCardReadings } from '../model/playerCardReadings'
import { respawnShort, respawnText } from '../model/rosterRowReadings'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'

type ReplayText = (typeof REPLAY_TEXT)[ReplayLocale]

/** La forme de la fiche : sur deux lignes (rangée du Cinéma) ou sur une ligne (colonne du Standard). */
export type CompactForm = 'stacked' | 'line'

export interface ReplayCompactCardProps {
  player: ReplayPlayer
  doc: ReplayDocumentReady
  frame: number
  locale: ReplayLocale
  form: CompactForm
  /** Âges de lecture, scène des effets et calque de score, construits une fois par la colonne ou la rangée (`useCardScene`). */
  scene: CardScene
  /** L'allégeance du camp du joueur, vue du joueur regardé : la couleur de la pastille. */
  ally: Allegiance
  /** Vrai = le joueur suivi (le joueur regardé par la page) : bordure à la couleur de suivi, nom en gras. */
  tracked?: boolean
  /** Vrai = la fiche de ce joueur est ouverte (`aria-pressed`, anneau). */
  selected?: boolean
  /** Un clic sur la fiche, avec la clé de roster du joueur. Absent = fiche inerte. */
  onCard?: (xuid: string) => void
}

/** La hauteur de la fiche selon sa forme. */
function heightOf(form: CompactForm): number {
  return form === 'stacked' ? G.stackedPx : G.linePx
}

/** La boîte de la fiche, commune à la fiche d'un joueur et à la fiche d'attente : mêmes cotes, habit différent. */
const BOX_CLASS = `relative flex min-w-0 border ${G.layerRadius}`

function boxStyle(form: CompactForm, chrome: CSSProperties): CSSProperties {
  const stacked = form === 'stacked'
  return {
    height: heightOf(form),
    paddingInline: G.padX,
    flexDirection: stacked ? 'column' : 'row',
    justifyContent: stacked ? 'center' : 'flex-start',
    alignItems: stacked ? undefined : 'center',
    gap: stacked ? G.stackGapPx : G.lineGapPx,
    ...chrome,
  }
}

/** La pastille de camp, devant le nom : jamais une bordure colorée. */
function Dot({ ally }: { ally: Allegiance }) {
  return (
    <span
      aria-hidden
      className="inline-block shrink-0 rounded-sm"
      style={{ width: G.dotPx, height: G.dotPx, background: campInk(ally) }}
    />
  )
}

/** Le nom : une ligne, points de suspension, gras pour le joueur suivi, éteint pour un éliminé. */
function Name({ text, bold, dim, muted, title }: {
  text: string
  bold: boolean
  dim: boolean
  muted: boolean
  title?: string
}) {
  return (
    <b
      className={`block min-w-0 flex-1 truncate ${bold ? 'font-bold' : 'font-medium'} ${muted ? 'text-muted-foreground' : 'text-foreground'}`}
      style={{ fontSize: G.namePx, lineHeight: `${G.nameLeadingPx}px`, opacity: dim ? G.deadOpacity : 1 }}
      title={title}
    >
      {text}
    </b>
  )
}

/** Les deux barres, le bouclier au-dessus de la santé ; sur une ligne, à largeur fixe, sur deux lignes, à toute la place. */
function Gauges({ read, scene, t, form }: { read: PlayerCardReadings; scene: CardScene; t: ReplayText; form: CompactForm }) {
  const { state } = read
  if (!state.shield && !state.health) return null
  return (
    <span
      className="flex min-w-0 flex-col"
      style={form === 'line' ? { width: G.gaugeLineW, flex: 'none', gap: G.gaugeGapPx } : { flex: '1 1 0%', gap: G.gaugeGapPx }}
    >
      <VitalityBar reading={state.shield} fade={scene.vitalityFade} name={t.shieldLabel} token="info" heightPx={G.gaugeShieldPx} />
      <VitalityBar reading={state.health} fade={scene.vitalityFade} name={t.healthLabel} token="success" heightPx={G.gaugeHealthPx} />
    </span>
  )
}

/** Le compte à rebours abrégé, à la couleur d'alerte ; le texte long en infobulle. */
function Respawn({ read, doc, frame, t }: { read: PlayerCardReadings; doc: ReplayDocumentReady; frame: number; t: ReplayText }) {
  return (
    <span
      className="truncate whitespace-nowrap"
      style={{ fontSize: G.respawnPx, lineHeight: `${G.respawnLeadingPx}px`, color: tokenCssVar('destructive') }}
      title={respawnText(read.state, doc, frame, t)}
    >
      {respawnShort(read.state, doc, frame, t)}
    </span>
  )
}

/** Les marques à droite des jauges : le glyphe de l'objectif porté, la pastille « bot ». */
function Marks({ read, bot, t }: { read: PlayerCardReadings; bot: boolean; t: ReplayText }): ReactNode {
  const { objective } = read
  return (
    <>
      {objective && (
        <span className="inline-flex shrink-0 text-foreground" title={t.objectiveCarry[objective]}>
          <ReplayObjectiveGlyph kind={objective} sizePx={G.markPx} />
        </span>
      )}
      {bot && (
        <span
          className="inline-grid shrink-0 place-items-center rounded-[3px] border border-border font-normal leading-none text-muted-foreground"
          style={{ height: G.capHeightPx, paddingInline: G.capPadX, fontSize: G.capTextPx }}
          title={t.roster.botHint}
        >
          {t.roster.bot}
        </span>
      )}
    </>
  )
}

/**
 * ReplayCompactCard — la fiche compacte d'un joueur PRÉSENT.
 */
export function ReplayCompactCard({
  player, doc, frame, locale, form, scene, ally, tracked = false, selected = false, onCard,
}: ReplayCompactCardProps) {
  const t = REPLAY_TEXT[locale]
  const read = playerCardReadings({
    player, doc, frame, presence: scene.presence, flashFrames: scene.flashFrames,
    scoreTimeline: scene.scoreTimeline, fxScene: scene.fxScene, text: t,
  })
  const { state, fx, zones, name } = read
  const bot = player.bot === true
  const press = onCard
    ? { label: t.playerPanel.cardOpenFmt(name), selected, onActivate: () => onCard(player.xuid) }
    : undefined
  const chrome: CSSProperties = cardChrome(state.alive)
  if (tracked) chrome.borderColor = 'var(--ring)'
  const marks = <Marks read={read} bot={bot} t={t} />
  const gauges = state.alive ? <Gauges read={read} scene={scene} t={t} form={form} /> : <Respawn read={read} doc={doc} frame={frame} t={t} />
  const nameEl = <Name text={name} bold={tracked} dim={!state.alive} muted={bot} title={name} />
  return (
    <div
      className={`${BOX_CLASS}${press ? ` ${PRESS_CLASS}${selected ? ` ${SELECTED_CLASS}` : ''}` : ''}`}
      style={boxStyle(form, chrome)}
      title={fx.title}
      {...(press ? pressProps(press) : {})}
    >
      <CardUnderLayers fx={fx} objective={null} radiusClass={G.layerRadius} watermarkPx={0} />
      {form === 'stacked' ? (
        <>
          <span className="relative flex min-w-0 items-center" style={{ gap: G.dotGapPx }}>
            <Dot ally={ally} />
            {nameEl}
          </span>
          <span className="relative flex min-w-0 items-center" style={{ gap: G.marksGapPx, minHeight: G.gaugeShieldPx + G.gaugeGapPx + G.gaugeHealthPx }}>
            {gauges}
            {marks}
          </span>
        </>
      ) : (
        <>
          <span className="relative inline-flex shrink-0"><Dot ally={ally} /></span>
          <span className="relative flex min-w-0 flex-1 items-center">{nameEl}</span>
          <span className="relative flex shrink-0 items-center" style={{ gap: G.marksGapPx }}>
            {gauges}
            {marks}
          </span>
        </>
      )}
      <ZoneFxOverlay zones={zones} translocationDelay={fx.translocationDelay} boltCount={G.boltCount} radiusClass={G.layerRadius} />
    </div>
  )
}

/**
 * ReplayCompactWaitingCard — la fiche d'une place sans fiche, de même boîte : l'habit tireté, le nom de
 * l'occupant qui la tient (aucun nom pour une place libre) et son état à la place des jauges.
 */
export function ReplayCompactWaitingCard({ occupant, locale, form }: {
  occupant: ReplayPlayer | null
  locale: ReplayLocale
  form: CompactForm
}) {
  const { name, label, hint } = seatWaitingReading(occupant, REPLAY_TEXT[locale])
  const nameEl = (
    <span
      aria-hidden={name === null || undefined}
      className="min-w-0 truncate font-medium text-muted-foreground"
      style={{ fontSize: G.namePx, lineHeight: `${G.nameLeadingPx}px`, flex: form === 'line' ? '1 1 auto' : undefined }}
      title={name ?? undefined}
    >
      {name ?? ' '}
    </span>
  )
  const state = (
    <span
      className="truncate whitespace-nowrap uppercase tracking-wider text-muted-foreground"
      style={{ fontSize: G.respawnPx, lineHeight: `${G.respawnLeadingPx}px` }}
    >
      {label}
    </span>
  )
  return (
    <div className={BOX_CLASS} style={boxStyle(form, SEAT_WAITING_CHROME)} title={hint}>
      {nameEl}
      {state}
    </div>
  )
}
