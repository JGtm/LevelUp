/**
 * ReplayPlayerPicker — LE SÉLECTEUR DE JOUEURS DE LA FICHE DU STANDARD : un rectangle arrondi par joueur, deux
 * initiales à la couleur de son camp, pour passer à la fiche d'un autre joueur sans revenir à la colonne.
 *
 *  - LES CAMPS sont ceux du film (`groupByTeam`), l'un après l'autre, séparés par un espace plus large ; la
 *    couleur de chaque rectangle est celle du camp vu du joueur regardé (`campInk`, jetons `team-*`).
 *  - UN JOUEUR MORT À L'IMAGE LUE est estompé : il reste choisissable (sa fiche s'ouvre), la couleur ne porte
 *    pas seule, car le bouton le dit aussi (`data-dead`).
 *  - LE JOUEUR DE LA FICHE est marqué par un double anneau (un liseré à la couleur du fond, puis l'anneau de
 *    premier plan) et `aria-pressed` ; le choisir de nouveau ne fait rien (il ne referme pas la fiche).
 *  - LE NOM COMPLET est l'infobulle et le nom accessible de chaque rectangle : deux initiales ne suffisent pas.
 *  - LES RANGÉES SE REPLIENT : un camp trop large pour la colonne passe sur plusieurs rangées au lieu de
 *    déborder (Grande équipe). L'anneau dépasse du rectangle : le bloc réserve cette marge en dedans pour que la
 *    zone qui défile ne le rogne pas.
 *
 * Aucun littéral de couleur : l'encre des camps et les neutres sont des jetons sémantiques.
 */
import { useMemo } from 'react'

import type { FilmAllegiance } from '@/lib/replay/filmAllegiance'
import { isAliveAt } from '@/lib/replay/replayLogic'
import { groupByTeam, type ReplayPlayer } from '@/lib/replay/rosterLogic'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { campInk } from '../model/campInk'
import { COTES_FICHE_STANDARD, SELECTEUR_ANNEAU_TOTAL_PX } from '../model/cotesFicheStandard'
import { cardName } from '../model/playerCardReadings'
import { PRESS_GUARDS } from './replayPressGuards'

const S = COTES_FICHE_STANDARD.selecteur

export interface ReplayPlayerPickerProps {
  /** La jointure film <-> base du match (`model.players`). */
  players: readonly ReplayPlayer[]
  /** L'image lue : elle dit qui est mort. */
  frame: number
  locale: ReplayLocale
  /** L'allégeance du film, vue du joueur regardé : la couleur de chaque camp. */
  allegiance: FilmAllegiance
  /** La clé de roster du joueur de la fiche ouverte. */
  selectedXuid: string
  /** Passer à la fiche de ce joueur (clé de roster) : `playerPanel.onCard`. */
  onCard: (xuid: string) => void
}

/** Les initiales d'un nom : ses premières lettres, en capitales (jamais coupées au milieu d'un caractère). */
function initialsOf(name: string): string {
  return Array.from(name.trim()).slice(0, S.initiales).join('').toUpperCase()
}

/** Le double anneau du joueur de la fiche : le fond de la fiche, puis le premier plan. */
const RING = `0 0 0 ${S.anneauFondPx}px var(--card), 0 0 0 ${SELECTEUR_ANNEAU_TOTAL_PX}px var(--foreground)`

export function ReplayPlayerPicker({ players, frame, locale, allegiance, selectedXuid, onCard }: ReplayPlayerPickerProps) {
  const t = REPLAY_TEXT[locale]
  const groups = useMemo(() => groupByTeam(players), [players])
  return (
    <div
      role="group"
      aria-label={t.roster.changePlayer}
      data-replay-player-picker=""
      className="flex flex-wrap"
      style={{
        columnGap: S.ecartCampsPx,
        rowGap: S.ecartRangeesPx,
        margin: -SELECTEUR_ANNEAU_TOTAL_PX,
        padding: SELECTEUR_ANNEAU_TOTAL_PX,
      }}
    >
      {groups.map((group) => (
        <div
          key={`camp:${group.team}`}
          data-replay-picker-camp=""
          className="flex min-w-0 max-w-full flex-wrap"
          style={{ columnGap: S.ecartPx, rowGap: S.ecartRangeesPx }}
        >
          {group.players.map((player) => {
            const name = cardName(player, t)
            const selected = player.xuid === selectedXuid
            const dead = !player.lives.some((life) => isAliveAt(life, frame))
            return (
              <button
                key={player.xuid}
                type="button"
                title={name}
                aria-label={name}
                aria-pressed={selected}
                data-dead={dead ? 'true' : undefined}
                className="grid shrink-0 cursor-pointer place-items-center font-bold"
                style={{
                  width: S.largeurPx,
                  height: S.hauteurPx,
                  borderRadius: S.rayonPx,
                  fontSize: S.taillePx,
                  background: campInk(allegiance.ofTeam(group.team)),
                  color: 'var(--background)',
                  opacity: dead ? S.mortOpacite : undefined,
                  boxShadow: selected ? RING : undefined,
                }}
                {...PRESS_GUARDS}
                onClick={() => {
                  if (!selected) onCard(player.xuid)
                }}
              >
                {initialsOf(name)}
              </button>
            )
          })}
        </div>
      ))}
    </div>
  )
}
