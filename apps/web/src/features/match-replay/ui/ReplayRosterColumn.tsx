/**
 * ReplayRosterColumn — LES ÉQUIPES EN LIGNES, dans la colonne de 360 px du cadre Standard : les camps l'un
 * sous l'autre, chacun avec son en-tête (pastille à la couleur du camp et son nom), puis une ligne par
 * place. Un seul en-tête de colonnes (Arme, Gr., Éq., F / M / A) sous le premier camp.
 *
 * LES PLACES ET LES CAMPS SONT CEUX DU FILM, comme pour les fiches : une ligne est une PLACE, pas un
 * joueur (`buildSeats`, `groupSeatsByTeam`, `seatTileAt`) — son occupant à l'instant lu, ou la ligne
 * d'attente d'une place libre ou d'un occupant pas encore apparu. Jamais plus de lignes que de places,
 * jamais un joueur parti. En Grande équipe, une place qui ne rend rien à l'image lue garde sa cellule
 * (vide, de la taille d'une fiche) : les paires suivantes ne se décalent pas, chaque joueur garde sa
 * position d'une image à l'autre. Un camp sans désignateur (mode sans camps) n'a ni en-tête ni pastille.
 *
 * UNE GRANDE ÉQUIPE (`grandeEquipe` : plus de `PLACES_PAR_CAMP_MAX_RANGEE` places dans un camp) se range
 * en PAIRES DE FICHES COMPACTES sur une ligne (`ReplayCompactCard`, forme `'line'`), côte à côte, sous
 * l'en-tête de camp : ni lignes, ni en-tête de colonnes (la fiche n'a ni arme, ni grenades, ni équipement,
 * ni F / M / A), et la colonne ni ne défile ni ne se comprime. Les autres effectifs gardent leurs lignes : la part des
 * équipes est alors une zone à défilement vertical dont les lignes gardent leur hauteur, et l'en-tête de
 * chaque camp reste collé en haut pendant que ses lignes passent.
 *
 * LE JOUEUR SUIVI ARRIVE PAR PROP (`tracked`, `playerPanel.tracked`) : la colonne ne le recalcule pas.
 * Les âges de lecture et la scène des effets sont construits une fois pour toute la colonne
 * (`useCardScene`).
 */
import { useMemo } from 'react'

import type { Allegiance, FilmAllegiance } from '@/lib/replay/filmAllegiance'

import { ReplayCompactCard, ReplayCompactWaitingCard } from './ReplayCompactCard'
import { ReplayRosterRow, ReplayRosterWaitingRow, ROW_COLUMNS } from './ReplayRosterRow'
import { useCardScene, type CardScene } from '../hooks/useCardScene'
import { campInk } from '../model/campInk'
import { GABARIT_FICHE_COMPACTE as C, GABARIT_LIGNE as G } from '../model/cardGabarit'
import { grandeEquipe } from '../model/rowLayout'
import { buildSeats, groupSeatsByTeam, seatTileAt, type ReplaySeatGroup } from '../model/seatLogic'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { campLabel } from '../../../lib/replay/replayCamps'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'

export interface ReplayRosterColumnProps {
  doc: ReplayDocumentReady
  /** La jointure film <-> base du match (`model.players`) : la page la construit une fois. */
  players: readonly ReplayPlayer[]
  frame: number
  locale: ReplayLocale
  /** L'allégeance lue dans le film, vue du joueur regardé : elle donne sa couleur à la pastille de chaque camp. */
  allegiance: FilmAllegiance
  /** Le joueur suivi (le joueur regardé par la page, `playerPanel.tracked`) : sa ligne se déplie. */
  tracked: ReplayPlayer | null
  /** Un clic sur la ligne d'un joueur présent (`playerPanel.onCard`), avec sa clé de roster. Absent = lignes inertes. */
  onCard?: (xuid: string) => void
  /** La clé de roster du joueur dont la fiche est ouverte (`aria-pressed`). */
  selectedXuid?: string | null
}

/** Les lignes de feuille de TOUS les occupants d'un camp, sur tout le match : ce qui le nomme. */
function sheetRowsOf(group: ReplaySeatGroup) {
  return group.seats.flatMap((s) => s.occupants.map((o) => o.player.board))
}

/** L'en-tête d'un camp : sa pastille carrée et son nom ; collé en haut de la zone qui défile. */
function TeamHead({ label, ally }: { label: string; ally: Allegiance }) {
  return (
    <h3
      className="sticky top-0 z-10 flex items-center bg-card font-semibold text-foreground"
      style={{ height: G.teamHeadPx, gap: G.teamHeadGapPx, fontSize: G.teamNamePx }}
    >
      <span
        aria-hidden
        className="inline-block shrink-0 rounded-sm"
        style={{ width: G.teamDotPx, height: G.teamDotPx, background: campInk(ally) }}
      />
      <span className="min-w-0 truncate">{label}</span>
    </h3>
  )
}

/** L'en-tête de colonnes, une seule fois : chaque titre dit son nom entier en infobulle. */
function ColumnHeads({ locale }: { locale: ReplayLocale }) {
  const r = REPLAY_TEXT[locale].roster
  const heads: [string, string][] = [
    [r.colWeapon, r.colWeaponTitle],
    [r.colGrenades, r.colGrenadesTitle],
    [r.colEquipment, r.colEquipmentTitle],
    [r.colKda, r.colKdaTitle],
  ]
  return (
    <div
      aria-hidden
      className="grid items-center border border-transparent text-right text-muted-foreground"
      style={{ gridTemplateColumns: ROW_COLUMNS, columnGap: G.colGapPx, height: G.headPx, fontSize: G.headTextPx }}
    >
      <span />
      {heads.map(([text, title]) => (
        <span key={title} title={title} className="whitespace-nowrap">
          {text}
        </span>
      ))}
    </div>
  )
}

/** La cellule d'une place sans rendu à cette image : vide, de la hauteur d'une fiche compacte sur une ligne. */
function EmptyCompactCell() {
  return <div aria-hidden data-replay-seat-empty="" style={{ height: C.linePx }} />
}

/** Ce que le rendu des places d'un camp partage : le document, l'image, la langue, la scène, le joueur suivi, le clic. */
interface SeatCtx extends Pick<ReplayRosterColumnProps, 'doc' | 'frame' | 'locale' | 'tracked' | 'onCard' | 'selectedXuid'> {
  scene: CardScene
  /** Vrai = Grande équipe : une fiche compacte par place, au lieu d'une ligne. */
  compact: boolean
  ally: Allegiance
}

/**
 * Les places d'un camp : UNE LIGNE (ou une fiche compacte) PAR PLACE, À CHAQUE IMAGE — son occupant à
 * l'instant lu, « pas encore apparu » s'il n'a pas de corps, ou la place VIDE. Une place qui ne rend
 * RIEN à cette image (`seatTileAt` nul) ne produit aucune ligne, sauf en Grande équipe : sa cellule est
 * alors gardée, vide (`EmptyCompactCell`), pour que la paire reste une paire.
 */
function seatsOf(group: ReplaySeatGroup, ctx: SeatCtx) {
  const { doc, frame, locale, scene, tracked, onCard, selectedXuid = null, compact, ally } = ctx
  return group.seats.map((seat) => {
    const lu = seatTileAt(seat, frame)
    if (lu === null) return compact ? <EmptyCompactCell key={seat.key} /> : null
    if (lu.kind !== 'present' || lu.player === null) {
      return compact
        ? <ReplayCompactWaitingCard key={seat.key} occupant={lu.player} locale={locale} form="line" />
        : <ReplayRosterWaitingRow key={seat.key} occupant={lu.player} locale={locale} />
    }
    const player = lu.player
    const shared = {
      player, doc, frame, locale, scene,
      tracked: tracked !== null && tracked.xuid === player.xuid,
      selected: selectedXuid === player.xuid,
      onCard,
    }
    return compact
      ? <ReplayCompactCard key={seat.key} {...shared} form="line" ally={ally} />
      : <ReplayRosterRow key={seat.key} {...shared} />
  })
}

export function ReplayRosterColumn({
  doc, players, frame, locale, allegiance, tracked, onCard, selectedXuid = null,
}: ReplayRosterColumnProps) {
  const t = REPLAY_TEXT[locale]
  // UNE LIGNE EST UNE PLACE, PAS UN JOUEUR : la place et la présence viennent du document.
  const seats = useMemo(() => buildSeats(players, doc), [players, doc])
  // LES COLONNES SONT LES CAMPS DU FILM : un camp par désignateur, jamais un camp « sans équipe ».
  const groups = useMemo(() => groupSeatsByTeam(seats), [seats])
  const labels = useMemo(() => groups.map((g) => campLabel(g, sheetRowsOf(g), t)), [groups, t])
  const scene = useCardScene(doc, players)

  if (groups.length === 0) {
    // LE DIAGNOSTIC DU PONT S'AFFICHE AVEC LE CONSTAT : « aucune vie rattachée » sans ses dénominateurs
    // se lirait comme un bug muet.
    const bridge = doc.coverage?.bridge
    return (
      <div className="rounded-lg border border-border bg-card p-3">
        <p className="text-xs text-muted-foreground">{t.rosterEmpty}</p>
        {bridge && (
          <p className="mt-1 text-3xs text-muted-foreground">
            {t.bridgeDiag(bridge.livesNamed, bridge.livesTotal, bridge.slotCollisions)}
          </p>
        )}
      </div>
    )
  }

  // LE SEUIL SE LIT SUR LE NOMBRE DE PLACES PAR CAMP, jamais sur un nom de mode (`grandeEquipe`).
  const compact = grandeEquipe(groups.map((g) => g.seats.length))
  return (
    <div
      data-replay-roster-column=""
      className={`flex flex-col${compact ? ' shrink-0' : ' min-h-0 overflow-y-auto overflow-x-hidden'}`}
      style={{ gap: G.teamsGapPx }}
    >
      {groups.map((group, gi) => {
        const ally = allegiance.ofTeam(group.team)
        const places = seatsOf(group, { doc, frame, locale, scene, tracked, onCard, selectedXuid, compact, ally })
        return (
          <section key={`camp:${group.team}`} aria-label={labels[gi]} className="flex shrink-0 flex-col">
            {group.team >= 0 && <TeamHead label={labels[gi]} ally={ally} />}
            {compact ? (
              <div
                className="grid"
                style={{ gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', columnGap: C.pairColGapPx, rowGap: C.pairRowGapPx }}
              >
                {places}
              </div>
            ) : (
              <>
                {gi === 0 && <ColumnHeads locale={locale} />}
                {places}
              </>
            )}
          </section>
        )
      })}
    </div>
  )
}
