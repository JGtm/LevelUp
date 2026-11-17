/**
 * ReplayRosterColumn — LES ÉQUIPES EN LIGNES, dans la colonne de 360 px du cadre Standard : les camps l'un
 * sous l'autre, chacun avec son en-tête (pastille à la couleur du camp et son nom), puis une ligne par
 * place. Un seul en-tête de colonnes (Arme, Gr., Éq., F / M / A) sous le premier camp.
 *
 * LES PLACES ET LES CAMPS SONT CEUX DU FILM, comme pour les fiches : une ligne est une PLACE, pas un
 * joueur (`buildSeats`, `groupSeatsByTeam`, `seatTileAt`) — son occupant à l'instant lu, ou la ligne
 * d'attente d'une place libre ou d'un occupant pas encore apparu. Jamais plus de lignes que de places,
 * jamais un joueur parti. Un camp sans désignateur (mode sans camps) n'a ni en-tête ni pastille.
 *
 * LA PART DES ÉQUIPES DÉFILE, et elle seule : la colonne est une zone à défilement vertical dont les
 * lignes gardent leur hauteur (24 joueurs d'une Grande équipe tiennent par défilement, sans forme
 * compacte), et l'en-tête de chaque camp reste collé en haut pendant que ses lignes passent.
 *
 * LE JOUEUR SUIVI ARRIVE PAR PROP (`tracked`, `playerPanel.tracked`) : la colonne ne le recalcule pas.
 * Les âges de lecture et la scène des effets sont construits une fois pour toute la colonne
 * (`useCardScene`).
 */
import { useMemo } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { Allegiance, FilmAllegiance } from '@/lib/replay/filmAllegiance'

import { ReplayRosterRow, ReplayRosterWaitingRow, ROW_COLUMNS } from './ReplayRosterRow'
import { useCardScene } from '../hooks/useCardScene'
import { GABARIT_LIGNE as G } from '../model/cardGabarit'
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

/** L'encre d'un camp : la couleur d'allié ou d'adversaire, ou la bordure quand l'allégeance n'est pas connue. */
function campInk(ally: Allegiance): string {
  return ally === null ? 'var(--border)' : tokenCssVar(ally ? 'team-ally' : 'team-enemy')
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

  return (
    <div
      data-replay-roster-column=""
      className="flex min-h-0 flex-col overflow-y-auto overflow-x-hidden"
      style={{ gap: G.teamsGapPx }}
    >
      {groups.map((group, gi) => (
        <section key={`camp:${group.team}`} aria-label={labels[gi]} className="flex shrink-0 flex-col">
          {group.team >= 0 && <TeamHead label={labels[gi]} ally={allegiance.ofTeam(group.team)} />}
          {gi === 0 && <ColumnHeads locale={locale} />}
          {/* UNE LIGNE PAR PLACE, À CHAQUE IMAGE : son occupant à l'instant lu, « pas encore apparu » s'il
              n'a pas de corps, ou la place VIDE. Une place qui ne rend RIEN à cette image
              (`seatTileAt` nul) ne produit aucune ligne. */}
          {group.seats.map((seat) => {
            const lu = seatTileAt(seat, frame)
            if (lu === null) return null
            if (lu.kind !== 'present' || lu.player === null) {
              return <ReplayRosterWaitingRow key={seat.key} occupant={lu.player} locale={locale} />
            }
            const player = lu.player
            return (
              <ReplayRosterRow
                key={seat.key}
                player={player}
                doc={doc}
                frame={frame}
                locale={locale}
                scene={scene}
                tracked={tracked !== null && tracked.xuid === player.xuid}
                selected={selectedXuid === player.xuid}
                onCard={onCard}
              />
            )
          })}
        </section>
      ))}
    </div>
  )
}
