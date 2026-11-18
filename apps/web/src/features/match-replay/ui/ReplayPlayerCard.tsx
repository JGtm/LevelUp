/**
 * ReplayPlayerCard — LA TUILE d'un siège de la rangée des équipes (Cinéma, Plein cadre) : ce qu'un
 * joueur est à l'instant lu, vivant ou mort — bouclier, santé, triplet, arme en main, grenade et
 * capacité, temps avant le retour — et les effets qui l'habillent.
 *
 * Les couches d'effets (sous le contenu, incrustation, tables d'éclairs et de croix) vivent dans
 * `ReplayCardFx.tsx`, que la tuile pose sans leur prêter sa forme. Les LECTURES (compteurs, état
 * vital, armes, zones, objectif, composition des effets) vivent dans
 * `model/playerCardReadings.ts` : cette tuile ne calcule plus rien, elle rend. La rangée des
 * équipes (`ReplayTeamsRow.tsx`) ne garde que les camps et la grille des places ; la scène des
 * effets vient de `hooks/useCardScene.ts`.
 *
 * LES COTES VIENNENT DU GABARIT COMPACT (`model/cardGabarit.ts`) — des nombres, jamais un mot de
 * densité : tuile de 115 × 62, corps de 31. La MISE EN PAGE est écrite ici en littéraux Tailwind
 * (une valeur arbitraire interpolée ne produit AUCUNE règle, en silence) : TROIS lignes — le nom
 * SEUL sur la première (14 px, `leading-[14px]` : sans lui le preflight Tailwind donne 17,25 px
 * et la tuile 65), les jauges 4 / 2 px à gauche et le triplet à droite sur la deuxième (12 px),
 * puis la rangée arme 48, grenade sélectionnée 14, capacité 16 (16 px, `gap 5`). Interlignes
 * 3 px ; corps FIXE 31 = 12 + 3 + 16 ; tuile 1 + 6 + 14 + 3 + 31 + 6 + 1 = 62. Le nom n'est PAS en
 * `flex-1`, pour que l'infobulle de la TUILE (`fx.title`) reste atteignable sur le reste de la
 * ligne : le `title` d'une cellule masque celui de la tuile. La profondeur du DOM autour du nom
 * ne bouge pas : dix tests atteignent la tuile par `getByText(nom).parentElement.parentElement`.
 *
 * TROIS SORTES DE TUILE, UNE SEULE BOÎTE : la fiche d'un joueur présent et la tuile d'attente
 * (`ReplaySeatWaiting` : place LIBRE, ou occupant PAS ENCORE APPARU) se rendent dans le MÊME
 * squelette (`TileFrame`) : boîte `TILE`, ligne du nom, corps à hauteur fixe `BODY_CLASS` —
 * classe pour classe. Seul l'HABIT change (le chrome tireté des tuiles d'attente,
 * `SEAT_WAITING_CHROME`) : une tuile d'attente ne décale rien.
 *
 * TROIS RÈGLES QUI NE SE NÉGOCIENT PAS ICI :
 *   1. Une valeur non lue s'affiche comme une lacune, jamais comme un zéro ni une moyenne.
 *   2. Une lecture ancienne PÂLIT et dit son âge — l'inventaire ne se lit qu'aux images-clés,
 *      une toutes les ~20 s, et le faire passer pour l'instant courant était un défaut réel.
 *   3. Aucun littéral de couleur : les rôles passent par des tokens sémantiques.
 */
import type { CSSProperties, ReactNode } from 'react'

import type { ReplayScoreTimelineReady } from '@/lib/replay/scoreTimeline'

import { CardUnderLayers, ZoneFxOverlay } from './ReplayCardFx'
import { ReplayCountersBadge } from './ReplayCountersBadge'
import { ReplayInventoryRow } from './ReplayInventoryRow'
import { EliminatedBox, VitalityBar } from './ReplayVitality'
import { ReplayWeaponsRow } from './ReplayWeaponsRow'
import { PRESS_CLASS, pressProps, selectedChrome, type TilePress } from './replayCardPress'
import { GABARIT_COMPACT as G } from '../model/cardGabarit'
import type { EquippedReading } from '../model/equippedLogic'
import { handCellHint } from '../model/handCellHint'
import { cardChrome, SEAT_WAITING_CHROME } from '../model/playerCardFx'
import { playerCardReadings, seatWaitingReading, type CardFxScene } from '../model/playerCardReadings'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { PlayerState, ReplayPlayer, VitalityPresence } from '../../../lib/replay/rosterLogic'

/** La tuile : chrome, marges, rayon. */
const TILE = 'relative flex shrink-0 flex-col rounded-md border px-1.5 py-1.5'
/** La ligne du nom : le nom seul, bridé à 14 px. */
const NAME_LINE = 'relative flex leading-[14px]'
/** Le nom, hors sa taille (`NAME_CLASS`) et hors son encre (vivant / mort). */
const NAME = 'min-w-0 truncate'
/** Le corps, avec sa hauteur FIXE et sa garantie `overflow-hidden`. */
const BODY_CLASS = 'relative mt-[3px] h-[31px] overflow-hidden'
/** La pile des deux jauges : à gauche de la ligne qui porte aussi le triplet. */
const GAUGES = 'flex min-w-0 flex-1 flex-col gap-[2px]'
/** La ligne des jauges, qui porte AUSSI le triplet. */
const VITALS_LINE = 'flex h-[12px] items-center gap-[5px]'
/** La rangée arme + grenade + capacité. */
const ROW = 'mt-[3px] flex h-[16px] flex-nowrap items-center gap-x-[5px] overflow-hidden'
/**
 * Le RAYON des couches d'effets — couche sous le contenu, incrustation (nuage, éclairs, anneau du
 * capteur, fourreau de translocation) et filigrane de porteur : LE MÊME que celui de la tuile, sans
 * quoi une couche `inset-0` à 8 px déborderait les coins d'une tuile à 6.
 */
const LAYER_RADIUS = 'rounded-md'
/** La taille du nom, en classe : le littéral en clair, jamais interpolé. */
const NAME_CLASS: Record<typeof G['namePx'], string> = { 11.5: 'text-[11.5px]' }

/**
 * La classe du NOM d'une tuile, la même pour les trois sortes de tuile : c'est elle qui donne sa
 * hauteur à la ligne du nom. Seule l'encre change — éteinte pour un mort et pour une tuile d'attente.
 */
function nameClass(eteint: boolean): string {
  return `${NAME} ${NAME_CLASS[G.namePx]} font-bold uppercase tracking-[.06em] ${
    eteint ? 'text-muted-foreground' : 'text-foreground'
  }`
}

interface TileFrameProps {
  /** L'habit de la tuile : fond et bordure (`cardChrome`, `SEAT_WAITING_CHROME`). */
  chrome: CSSProperties
  title: string | undefined
  /** Les couches posées SOUS le contenu (effets, filigrane) : avant la ligne du nom. */
  under?: ReactNode
  /** Ce que porte la ligne du nom. */
  nameLine: ReactNode
  /** Le contenu du corps à hauteur FIXE. */
  body: ReactNode
  /** L'incrustation posée PAR-DESSUS le contenu : après le corps. */
  over?: ReactNode
  /** Rend la tuile cliquable (panneau de joueur). Absent = boîte inerte. */
  press?: TilePress
}

/**
 * TileFrame — LE SQUELETTE d'une tuile de place : la boîte, la ligne du nom, le corps à hauteur
 * fixe. Les trois sortes de tuile y passent ; aucune ne réécrit une cote (garde-rail
 * `cardGabarit.guard.test.ts`, test DOM `ReplayTeamsRow.places.test.tsx`).
 */
function TileFrame({ chrome, title, under, nameLine, body, over, press }: TileFrameProps) {
  const className = press ? `${TILE} ${PRESS_CLASS}` : TILE
  return (
    <div className={className} style={press?.selected ? selectedChrome(chrome) : chrome} title={title} {...(press ? pressProps(press) : {})}>
      {under}
      <div className={NAME_LINE}>{nameLine}</div>
      <div className={BODY_CLASS}>{body}</div>
      {over}
    </div>
  )
}

interface ReplayPlayerCardProps {
  player: ReplayPlayer
  doc: ReplayDocumentReady
  frame: number
  presence: VitalityPresence
  vitalityFade: number
  readingFull: number
  flashFrames: number
  locale: ReplayLocale
  /** Calque de score du film, déjà passé par la garde d'horloge. */
  scoreTimeline?: ReplayScoreTimelineReady
  fxScene: CardFxScene
  /**
   * Le panneau de joueur : la fiche devient un bouton qui l'ouvre. `selected` = le panneau de CE joueur
   * est ouvert. Absent = fiche inerte (page sans panneau).
   */
  panel?: { selected: boolean; onSelect: () => void }
  /** Vrai = le joueur suivi (le joueur regardé par la page) : la bordure prend la couleur de suivi. */
  tracked?: boolean
}

/** L'habit d'une fiche vivante ou morte ; le joueur suivi y prend la bordure à la couleur de suivi. */
function chromeOf(alive: boolean, tracked: boolean): CSSProperties {
  const chrome = cardChrome(alive)
  return tracked ? { ...chrome, borderColor: 'var(--ring)' } : chrome
}

export function ReplayPlayerCard({
  player, doc, frame, presence, vitalityFade, readingFull, flashFrames, locale, scoreTimeline, fxScene, panel,
  tracked = false,
}: ReplayPlayerCardProps) {
  const t = REPLAY_TEXT[locale]
  const { live, state, name, equipped, filmIndex, zones, objective, fx } = playerCardReadings({
    player, doc, frame, presence, flashFrames, scoreTimeline, fxScene, text: t,
  })
  // LE TRIPLET vit sur la ligne des jauges — ou dans l'encadré « Éliminé » sur une fiche morte :
  // la mort ne retire pas les compteurs, il reste visible mort ou vif.
  const counters = <ReplayCountersBadge board={player.board} live={live} locale={locale} gabarit={G} />
  return (
    // LA TUILE : chaque fiche porte sa bordure et son fond — dégradé court autour de `card` en
    // vie, `card` teinté destructive en mort (cf. playerCardFx.cardChrome). `shrink-0` : une
    // tuile ne se tasse jamais.
    //
    // AUCUNE MARQUE D'IDENTITÉ SUR LA FICHE : le glyphe « ami » reste au FIL des éliminations, où
    // il sert à reconnaître un nom au milieu d'événements qui défilent ; sur une fiche, la
    // colonne d'équipe et le nom disent déjà tout ce qu'il y a à savoir. AUCUN ÉTAT DE MOUVEMENT
    // non plus : le document garde `stances[]`, la fiche ne le lit pas.
    <TileFrame
      chrome={chromeOf(state.alive, tracked)}
      title={fx.title}
      press={panel ? { label: t.playerPanel.cardOpenFmt(name), selected: panel.selected, onActivate: panel.onSelect } : undefined}
      under={<CardUnderLayers fx={fx} objective={objective} radiusClass={LAYER_RADIUS} watermarkPx={G.watermarkPx} />}
      nameLine={
        <span className={nameClass(!state.alive)} title={name}>
          {name}
        </span>
      }
      body={
        <CardBody
          doc={doc}
          frame={frame}
          state={state}
          equipped={equipped}
          filmIndex={filmIndex}
          readingFull={readingFull}
          vitalityFade={vitalityFade}
          locale={locale}
          counters={counters}
        />
      }
      over={
        <ZoneFxOverlay
          zones={zones}
          translocationDelay={fx.translocationDelay}
          boltCount={G.boltCount}
          radiusClass={LAYER_RADIUS}
        />
      }
    />
  )
}

interface CardBodyProps {
  doc: ReplayDocumentReady
  frame: number
  state: PlayerState
  equipped: EquippedReading | null
  filmIndex: number | null
  readingFull: number
  vitalityFade: number
  locale: ReplayLocale
  /** Le triplet F/M/A, construit une fois par la fiche : il vit sur la ligne des jauges, ou dans l'encadré « Éliminé ». */
  counters: ReactNode
}

/**
 * CardBody — LE CORPS À HAUTEUR FIXE d'une fiche (`bodyPx` 31 = jauges 12 + 3 + rangée 16), le même
 * vivant et mort. La mort remplace son CONTENU — l'encadré « Éliminé » remplit toute la zone —
 * jamais la zone : une fiche qui change de hauteur fait sauter toute la colonne à chaque mort.
 */
function CardBody({
  doc, frame, state, equipped, filmIndex, readingFull, vitalityFade, locale, counters,
}: CardBodyProps) {
  const t = REPLAY_TEXT[locale]
  if (!state.alive) {
    return <EliminatedBox state={state} doc={doc} frame={frame} locale={locale} bodyPx={G.bodyPx} counters={counters} />
  }
  // Le bouclier AU-DESSUS de la santé : l'ordre dans lequel le jeu les encaisse, dit aussi par
  // l'épaisseur (4 / 2). Sans `sh`/`hp` dans le document, aucune barre n'existe (`VitalityBar`
  // rend null sur une lecture nulle).
  const gauges = (
    <div className={GAUGES}>
      <VitalityBar reading={state.shield} fade={vitalityFade} name={t.shieldLabel} token="info" heightPx={G.gaugeShieldPx} />
      <VitalityBar reading={state.health} fade={vitalityFade} name={t.healthLabel} token="success" heightPx={G.gaugeHealthPx} />
    </div>
  )
  // C'est l'ARME EN MAIN qui dit ses munitions et les marques de la lecture d'inventaire, en
  // infobulle : la fiche compose le texte (`model/handCellHint.ts`, une lecture d'inventaire de
  // plus par fiche) et le confie à la rangée d'armes.
  const handHint = state.life ? (handCellHint(t, doc, state.life.slot, frame, equipped) ?? undefined) : undefined
  return (
    <>
      <div className={VITALS_LINE}>
        {gauges}
        {counters}
      </div>
      {/* ARME ET INVENTAIRE SUR UNE GRILLE À CELLULES FIXES : chaque cellule a une largeur
          constante — pour que les fiches s'alignent en colonnes. `flex-nowrap` +
          `overflow-hidden` : la rangée ne se replie jamais. */}
      <div className={ROW}>
        <ReplayWeaponsRow
          doc={doc}
          state={state}
          read={equipped}
          frame={frame}
          readingFull={readingFull}
          filmIndex={filmIndex}
          locale={locale}
          gabarit={G}
          handHint={handHint}
        />
        {state.life && (
          <ReplayInventoryRow
            doc={doc}
            slot={state.life.slot}
            frame={frame}
            readingFull={readingFull}
            locale={locale}
            gabarit={G}
          />
        )}
      </div>
    </>
  )
}

/**
 * WaitingState — le corps d'une tuile d'attente : l'état, centré dans la hauteur FIXE d'une
 * fiche (la zone même où une fiche morte pose son encadré « Éliminé »).
 */
function WaitingState({ label }: { label: string }) {
  return (
    <div className="flex h-full items-center justify-center">
      <span className="truncate text-3xs uppercase tracking-wider text-muted-foreground">{label}</span>
    </div>
  )
}

/**
 * ReplaySeatWaiting — LA TUILE D'UNE PLACE SANS FICHE, au gabarit d'une fiche (même squelette) :
 *
 *	`occupant` nul — (Q20) LA PLACE LIBRE : personne ne la tient à l'instant lu, son occupant est
 *	  parti et son remplaçant n'est pas encore arrivé, ou elle attend son premier occupant. Elle
 *	  reste à l'écran parce que le nombre de places est FINI (règle des places) : la retirer ferait
 *	  croire à une équipe plus petite, et la grille sauterait d'un cran puis reviendrait. Elle ne
 *	  montre AUCUN nom — un joueur parti n'est jamais affiché : sa ligne du nom ne porte qu'un espace
 *	  insécable, qui la tient à la hauteur de celle d'une fiche sans rien écrire ;
 *	un `occupant` — (Q21) IL TIENT LA PLACE, SANS CORPS ENCORE : au coup d'envoi, ou à son arrivée,
 *	  avant sa première apparition. Son nom, écrit comme sur sa fiche (`cardName`), et rien
 *	  d'autre : ni vitalité, ni armes, ni compteurs — lui prêter un état serait inventer.
 */
export function ReplaySeatWaiting({ occupant, locale }: { occupant: ReplayPlayer | null; locale: ReplayLocale }) {
  const { name, label, hint } = seatWaitingReading(occupant, REPLAY_TEXT[locale])
  return (
    <TileFrame
      chrome={SEAT_WAITING_CHROME}
      title={hint}
      nameLine={
        <span aria-hidden={name === null || undefined} className={nameClass(true)} title={name ?? undefined}>
          {name ?? ' '}
        </span>
      }
      body={<WaitingState label={label} />}
    />
  )
}
