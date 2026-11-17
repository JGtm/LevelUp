/**
 * ReplayPlayerCard — LA TUILE d'un siège : ce qu'un joueur est à l'instant lu, vivant ou mort,
 * bouclier, armes portées, temps avant le retour, et les effets qui l'habillent.
 *
 * EXTRAITE DE `ReplayTeams.tsx` LE 2026-09-06 (plan fiches compactes, item 1.3). Les couches
 * d'effets (sous le contenu, incrustation, tables d'éclairs et de croix) vivent dans
 * `ReplayCardFx.tsx`, que la tuile pose sans leur prêter sa forme. Les LECTURES (compteurs, état
 * vital, armes, zones, objectif, composition des effets) sont parties le même jour dans
 * `model/playerCardReadings.ts` : cette tuile ne calcule plus rien, elle rend. La colonne
 * (`ReplayTeams.tsx`) ne garde que les camps et la grille des sièges ; la scène des effets
 * vient de `hooks/useCardScene.ts`.
 *
 * LES COTES VIENNENT DU GABARIT (`model/cardGabarit.ts`) — des nombres et des booléens, jamais
 * un mot de densité : la même tuile rend la fiche normale (235 px, corps de 35) et, sur une
 * Grande équipe, la tuile compacte (115 × 62, corps de 31). La MISE EN PAGE se lit dans une
 * table FERMÉE par la hauteur du corps (`TILE_LAYOUT`, 2026-09-07) : un corps de 35 rend la
 * fiche d'aujourd'hui classe pour classe (fixation `__fixtures__/replayTeams.4v4.html`), un
 * corps de 31 la tuile à trois lignes. La profondeur du DOM autour du nom ne bouge dans aucun
 * des deux : dix tests atteignent la tuile par `getByText(nom).parentElement.parentElement`.
 *
 * TROIS SORTES DE TUILE, UNE SEULE BOÎTE (2026-10-06, retour utilisateur : la tuile « pas encore
 * apparu » n'avait ni la largeur ni la hauteur d'une fiche, et toute la colonne sautait). La fiche
 * d'un joueur présent et la tuile d'attente (`ReplaySeatWaiting` : place LIBRE, Q20, ou occupant
 * PAS ENCORE APPARU, Q21) se rendent dans le MÊME squelette (`TileFrame`) : boîte
 * `TILE_LAYOUT[...].tile`, ligne du nom, corps à hauteur fixe `BODY_CLASS[...]` — classe pour
 * classe, dans les deux gabarits. Seul l'HABIT change (le chrome tireté des tuiles d'attente,
 * `SEAT_WAITING_CHROME`) : une tuile d'attente ne décale plus rien.
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
import { PRESS_CLASS, pressProps, SELECTED_CLASS, type TilePress } from './replayCardPress'
import type { CardGabarit } from '../model/cardGabarit'
import type { EquippedReading } from '../model/equippedLogic'
import { handCellHint } from '../model/handCellHint'
import { cardChrome, SEAT_WAITING_CHROME } from '../model/playerCardFx'
import { playerCardReadings, seatWaitingReading, type CardFxScene } from '../model/playerCardReadings'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { PlayerState, ReplayPlayer, VitalityPresence } from '../../../lib/replay/rosterLogic'

/**
 * LA MISE EN PAGE DE LA TUILE, PAR HAUTEUR DE CORPS — une table FERMÉE de littéraux Tailwind
 * (une valeur arbitraire interpolée ne produit AUCUNE règle, en silence : cf. cardGabarit.ts et
 * `rosterHeight.guard.test.ts`). La clé est `CardGabarit['bodyPx']`, une union de littéraux :
 * un corps qui n'est pas dans la table ne compile pas.
 *
 * 35 — LA FICHE D'AUJOURD'HUI (option 2a du handoff 2026-08-27), classe pour classe : deux
 * lignes, le nom et le triplet côte à côte, puis le corps (jauges 5 / 3 px, rangée de 18 px à
 * cellules fixes). Le nom est en `flex-1` : la ligne est à lui et au triplet.
 *
 * 31 — LA TUILE COMPACTE (maquette A2, plan fiches compactes 2026-09-06) : 115 × 62, marge
 * 6 px, rayon 6 px, TROIS lignes. Ligne 1 (14 px) : le nom SEUL, pleine ligne, `leading-[14px]`
 * — sans lui le preflight Tailwind (1,5) donne 17,25 px et la tuile 65 ; le nom n'est PAS en
 * `flex-1`, pour que l'infobulle de la TUILE (`fx.title`) reste atteignable sur le reste de la
 * ligne (décision D6 : le `title` d'une cellule masque celui de la tuile). Ligne 2 (12 px) : les
 * jauges 4 / 2 px en `flex-1` à gauche (≈ 42 px), le triplet à droite (≈ 54 px). Ligne 3
 * (16 px) : arme en main 48, grenade sélectionnée 14, capacité 16, `gap 5` — 88 px, 13 d'air.
 * Interlignes 3 px ; corps FIXE 31 = 12 + 3 + 16 ; tuile 1 + 6 + 14 + 3 + 31 + 6 + 1 = 62.
 */
interface TileLayout {
  /** La tuile : chrome, marges, rayon. */
  tile: string
  /** La ligne du nom. */
  nameLine: string
  /** Le nom, hors sa taille (`NAME_CLASS`) et hors son encre (vivant / mort). */
  name: string
  /** Le corps, SANS sa hauteur (`BODY_CLASS`) ni son `overflow-hidden` (la garantie, ajoutée). */
  body: string
  /** Le triplet F/M/A vit sur la ligne du nom (vrai) ou sur la ligne des jauges, dans le corps. */
  countersOnNameLine: boolean
  /** La pile des deux jauges. */
  gauges: string
  /** La ligne des jauges quand elle porte AUSSI le triplet (corps de 31) ; '' sinon. */
  vitalsLine: string
  /** La rangée armes + inventaire. */
  row: string
  /**
   * Le RAYON des couches d'effets — couche sous le contenu, incrustation (nuage, éclairs,
   * anneau du capteur, fourreau de translocation) et filigrane de porteur : LE MÊME que celui
   * de la tuile, sans quoi une couche `inset-0` à 8 px déborderait les coins d'une tuile à 6
   * (étape 4.6, 2026-09-07). `rounded-lg` sur la fiche normale (la chaîne d'aujourd'hui, à
   * l'octet), `rounded-md` sur la compacte. Un littéral en clair par gabarit, jamais interpolé.
   */
  layerRadius: string
}

const TILE_LAYOUT: Record<CardGabarit['bodyPx'], TileLayout> = {
  35: {
    tile: 'relative flex shrink-0 flex-col rounded-lg border px-2.5 py-2',
    nameLine: 'relative flex items-baseline gap-1.5',
    name: 'min-w-0 flex-1 truncate',
    body: 'relative mt-[7px]',
    countersOnNameLine: true,
    gauges: 'flex flex-col gap-[3px]',
    vitalsLine: '',
    row: 'mt-[6px] flex h-[18px] flex-nowrap items-center gap-x-[5px] overflow-hidden',
    layerRadius: 'rounded-lg',
  },
  31: {
    tile: 'relative flex shrink-0 flex-col rounded-md border px-1.5 py-1.5',
    nameLine: 'relative flex leading-[14px]',
    name: 'min-w-0 truncate',
    body: 'relative mt-[3px]',
    countersOnNameLine: false,
    gauges: 'flex min-w-0 flex-1 flex-col gap-[2px]',
    vitalsLine: 'flex h-[12px] items-center gap-[5px]',
    row: 'mt-[3px] flex h-[16px] flex-nowrap items-center gap-x-[5px] overflow-hidden',
    layerRadius: 'rounded-md',
  },
}

/** La hauteur FIXE du corps, en classe : le littéral en clair, jamais interpolé. */
const BODY_CLASS: Record<CardGabarit['bodyPx'], string> = { 35: 'h-[35px]', 31: 'h-[31px]' }
/** La taille du nom, en classe — une seule valeur, les deux gabarits la partagent. */
const NAME_CLASS: Record<CardGabarit['namePx'], string> = { 11.5: 'text-[11.5px]' }

/**
 * La classe du NOM d'une tuile, la même pour les trois sortes de tuile : c'est elle qui donne sa
 * hauteur à la ligne du nom (corps 11,5 px, interligne du thème ; `leading-[14px]` sur la tuile
 * compacte). Seule l'encre change — éteinte pour un mort et pour une tuile d'attente.
 */
function nameClass(gabarit: CardGabarit, eteint: boolean): string {
  return `${TILE_LAYOUT[gabarit.bodyPx].name} ${NAME_CLASS[gabarit.namePx]} font-bold uppercase tracking-[.06em] ${
    eteint ? 'text-muted-foreground' : 'text-foreground'
  }`
}

interface TileFrameProps {
  gabarit: CardGabarit
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
 * `cardGabarit.guard.test.ts`, test DOM `ReplayTeams.places.test.tsx`).
 */
function TileFrame({ gabarit, chrome, title, under, nameLine, body, over, press }: TileFrameProps) {
  const L = TILE_LAYOUT[gabarit.bodyPx]
  const className = press ? `${L.tile} ${PRESS_CLASS}${press.selected ? ` ${SELECTED_CLASS}` : ''}` : L.tile
  return (
    <div className={className} style={chrome} title={title} {...(press ? pressProps(press) : {})}>
      {under}
      <div className={L.nameLine}>{nameLine}</div>
      <div className={`${L.body} ${BODY_CLASS[gabarit.bodyPx]} overflow-hidden`}>{body}</div>
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
  gabarit: CardGabarit
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
  player, doc, frame, presence, vitalityFade, readingFull, flashFrames, locale, scoreTimeline, fxScene, gabarit, panel,
  tracked = false,
}: ReplayPlayerCardProps) {
  const t = REPLAY_TEXT[locale]
  const { live, state, name, equipped, filmIndex, zones, objective, fx } = playerCardReadings({
    player, doc, frame, presence, flashFrames, scoreTimeline, fxScene, text: t,
  })
  const L = TILE_LAYOUT[gabarit.bodyPx]
  // LE TRIPLET EST LE MÊME ÉLÉMENT dans les deux gabarits ; seule sa PLACE change (ligne du
  // nom, ou ligne des jauges — et l'encadré « Éliminé » sur une fiche morte compacte). Il reste
  // visible mort ou vif : la mort ne retire pas les compteurs.
  const counters = <ReplayCountersBadge board={player.board} live={live} locale={locale} gabarit={gabarit} />
  return (
    // LA TUILE (option 2a du handoff 2026-08-27) : chaque fiche porte sa bordure et son
    // fond — dégradé court autour de `card` en vie, `card` teinté destructive en mort
    // (cf. playerCardFx.cardChrome). `shrink-0` : une tuile ne se tasse jamais, la colonne
    // défile.
    //
    // AUCUNE MARQUE D'IDENTITÉ SUR LA FICHE (demande utilisateur du 2026-08-25) : le glyphe
    // « ami » a été retiré de la colonne. Il reste au FIL des éliminations, où il sert à
    // reconnaître un nom au milieu d'événements qui défilent ; sur une fiche, la colonne
    // d'équipe et le nom disent déjà tout ce qu'il y a à savoir. AUCUN ÉTAT DE MOUVEMENT non
    // plus (décision utilisateur du 2026-09-23, Q16) : le document garde `stances[]`, la fiche
    // ne le lit pas.
    <TileFrame
      gabarit={gabarit}
      chrome={chromeOf(state.alive, tracked)}
      title={fx.title}
      press={panel ? { label: t.playerPanel.cardOpenFmt(name), selected: panel.selected, onActivate: panel.onSelect } : undefined}
      under={
        <CardUnderLayers fx={fx} objective={objective} radiusClass={L.layerRadius} watermarkPx={gabarit.watermarkPx} />
      }
      nameLine={
        <>
          <span className={nameClass(gabarit, !state.alive)} title={name}>
            {name}
          </span>
          {L.countersOnNameLine && counters}
        </>
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
          gabarit={gabarit}
          counters={counters}
        />
      }
      over={
        <ZoneFxOverlay
          zones={zones}
          translocationDelay={fx.translocationDelay}
          boltCount={gabarit.boltCount}
          radiusClass={L.layerRadius}
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
  gabarit: CardGabarit
  /** Le triplet F/M/A, construit une fois par la fiche : il vit sur la ligne du nom OU ici. */
  counters: ReactNode
}

/**
 * CardBody — LE CORPS À HAUTEUR FIXE d'une fiche (`bodyPx` : 35 = barres 11 + marge 6 +
 * inventaire 18 ; 31 = jauges 12 + 3 + rangée 16), la même vivante et morte. La mort remplace son
 * CONTENU — l'encadré « Éliminé » remplit toute la zone — jamais la zone : une fiche qui change de
 * hauteur fait sauter toute la colonne à chaque mort (retour utilisateur du 2026-08-24).
 */
function CardBody({
  doc, frame, state, equipped, filmIndex, readingFull, vitalityFade, locale, gabarit, counters,
}: CardBodyProps) {
  const t = REPLAY_TEXT[locale]
  const L = TILE_LAYOUT[gabarit.bodyPx]
  if (!state.alive) {
    return (
      <EliminatedBox
        state={state}
        doc={doc}
        frame={frame}
        locale={locale}
        bodyPx={gabarit.bodyPx}
        counters={L.countersOnNameLine ? undefined : counters}
      />
    )
  }
  // Le bouclier AU-DESSUS de la santé : l'ordre dans lequel le jeu les encaisse, dit aussi par
  // l'épaisseur (5 / 3 en normal, 4 / 2 en compact). Sans `sh`/`hp` dans le document, aucune
  // barre n'existe (`VitalityBar` rend null sur une lecture nulle).
  const gauges = (
    <div className={L.gauges}>
      <VitalityBar reading={state.shield} fade={vitalityFade} name={t.shieldLabel} token="info" heightPx={gabarit.gaugeShieldPx} />
      <VitalityBar reading={state.health} fade={vitalityFade} name={t.healthLabel} token="success" heightPx={gabarit.gaugeHealthPx} />
    </div>
  )
  // SANS CELLULE DE MUNITIONS (`showAmmo: false`), c'est l'ARME EN MAIN qui dit ses munitions
  // et les marques de la lecture d'inventaire, en infobulle (décision D5) : la fiche compose le
  // texte (`model/handCellHint.ts`, une lecture d'inventaire de plus par fiche compacte — le
  // volet JS du 5.2 le mesure) et le confie à la rangée d'armes. Jamais sur la fiche normale.
  const handHint =
    gabarit.showAmmo || !state.life
      ? undefined
      : (handCellHint(t, doc, state.life.slot, frame, equipped) ?? undefined)
  return (
    <>
      {L.countersOnNameLine ? (
        gauges
      ) : (
        <div className={L.vitalsLine}>
          {gauges}
          {counters}
        </div>
      )}
      {/* ARMES ET INVENTAIRE SUR UNE GRILLE À CELLULES FIXES (demande utilisateur du
          2026-08-24) : chaque rangée émet des cellules à largeur constante — pour que les
          fiches s'alignent en colonnes. `flex-nowrap` + `overflow-hidden` : la rangée ne se
          replie jamais. */}
      <div className={L.row}>
        <ReplayWeaponsRow
          doc={doc}
          state={state}
          read={equipped}
          frame={frame}
          readingFull={readingFull}
          filmIndex={filmIndex}
          locale={locale}
          gabarit={gabarit}
          handHint={handHint}
        />
        {state.life && (
          <ReplayInventoryRow
            doc={doc}
            slot={state.life.slot}
            equipped={equipped}
            frame={frame}
            readingFull={readingFull}
            locale={locale}
            gabarit={gabarit}
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
export function ReplaySeatWaiting({
  occupant,
  gabarit,
  locale,
}: {
  occupant: ReplayPlayer | null
  gabarit: CardGabarit
  locale: ReplayLocale
}) {
  const { name, label, hint } = seatWaitingReading(occupant, REPLAY_TEXT[locale])
  return (
    <TileFrame
      gabarit={gabarit}
      chrome={SEAT_WAITING_CHROME}
      title={hint}
      nameLine={
        <span aria-hidden={name === null || undefined} className={nameClass(gabarit, true)} title={name ?? undefined}>
          {name ?? '\u00a0'}
        </span>
      }
      body={<WaitingState label={label} />}
    />
  )
}
