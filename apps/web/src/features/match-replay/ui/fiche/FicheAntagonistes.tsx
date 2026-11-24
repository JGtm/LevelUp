/**
 * FicheAntagonistes — « ANTAGONISTES » : les deux titres du JOUEUR DE LA FICHE à l'instant lu. La fiche est
 * celle de n'importe quel joueur, pas seulement du point de vue de la page : le sujet est `player`, désigné
 * par sa clé du film.
 *
 *  - « Némésis » : l'adversaire qui a éliminé ce joueur le plus de fois jusqu'à cet instant — « X, k éliminations » ;
 *  - « Souffre-douleur » : l'adversaire que ce joueur a éliminé le plus de fois — « X, éliminé k fois » ;
 *  - à égalité de compte le titre reste à celui qui l'avait ; les autres adversaires au même compte s'écrivent
 *    après le tenant, « , à égalité avec Y » ;
 *  - l'icône de chaque titre en donne la définition en infobulle (`title`, comme sur la frise) ;
 *  - un titre sans tenant n'est pas écrit ; sans aucun titre, un cadre pointillé le dit.
 *
 * Les noms portent l'encre de leur camp (jetons, `campInk`, vu du point de vue de la page comme la colonne des
 * équipes). La lecture vient de la fabrique du rejeu (`ReplayAntagonists`) : ni rapprochement de clés ni décompte
 * ici. Les bots comptent comme tout joueur, dans les deux rôles (cf. `replayAntagonists`).
 */
import { Fragment, useMemo, type ReactNode } from 'react'

import type { AntagonistRole, AntagonistReading } from '@/lib/antagonists/antagonists'
import type { FilmAllegiance } from '@/lib/replay/filmAllegiance'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { ANTAGONISTS_TEXT, type AntagonistsText } from '../../i18n/antagonists.i18n'
import { REPLAY_TEXT, type ReplayLocale } from '../../i18n/i18n'
import { antagonistNameOf } from '../../model/antagonistNames'
import { campInk } from '../../model/campInk'
import { cardName } from '../../model/playerCardReadings'
import type { ReplayAntagonists } from '../../model/replayAntagonists'
import { AntagonistIcon } from '../AntagonistIcon'
import { FicheRow, FicheRows, FicheSection } from './ficheParts'

/** Le côté de l'icône à côté du libellé du titre. */
const ICON_PX = 12

export interface FicheAntagonistesProps {
  /** Le joueur de la fiche : le sujet des deux titres. */
  player: ReplayPlayer
  antagonists: ReplayAntagonists
  /** Le lobby du match : les noms des tenants se lisent parmi eux. */
  players: readonly ReplayPlayer[]
  /** L'allégeance du film vue du point de vue de la page : l'encre du camp de chaque nom. */
  allegiance: Pick<FilmAllegiance, 'ofXuid'>
  /** L'instant lu, sur l'axe BRUT du film (millisecondes). */
  nowMs: number
  locale: ReplayLocale
}

/** Les noms d'une liste, chacun à l'encre de son camp : « Y », « Y et Z », « Y, Z et W ». */
function NameList({ keys, nameOf, inkOf, text }: {
  keys: readonly string[]
  nameOf: (key: string) => string
  inkOf: (key: string) => string
  text: AntagonistsText['fiche']
}) {
  return (
    <>
      {keys.map((key, i) => (
        <Fragment key={key}>
          {i > 0 && (i === keys.length - 1 ? text.listLast : text.listSeparator)}
          <span data-fiche-nom="" style={{ color: inkOf(key) }}>{nameOf(key)}</span>
        </Fragment>
      ))}
    </>
  )
}

/** La valeur d'un titre : le tenant, son compte, puis les adversaires à égalité avec lui. */
function Value({ role, reading, nameOf, inkOf, text }: {
  role: AntagonistRole
  reading: AntagonistReading
  nameOf: (key: string) => string
  inkOf: (key: string) => string
  text: AntagonistsText['fiche']
}): ReactNode {
  return (
    <span data-fiche-antagoniste={role}>
      <NameList keys={[reading.holder]} nameOf={nameOf} inkOf={inkOf} text={text} />
      {text.partSeparator}
      {text[role].countFmt(reading.count)}
      {reading.tiedWith.length > 0 && (
        <>
          {text.partSeparator}
          {`${text.tiedWith} `}
          <NameList keys={reading.tiedWith} nameOf={nameOf} inkOf={inkOf} text={text} />
        </>
      )}
    </span>
  )
}

const ROLES: readonly AntagonistRole[] = ['nemesis', 'victim']

export function FicheAntagonistes({ player, antagonists, players, allegiance, nowMs, locale }: FicheAntagonistesProps) {
  const t = REPLAY_TEXT[locale]
  const text = ANTAGONISTS_TEXT[locale].fiche
  const nameOf = useMemo(() => antagonistNameOf(players, t), [players, t])
  const reading = antagonists.forSubject(player.xuid).at(nowMs)
  const subject = cardName(player, t)
  const inkOf = (key: string) => campInk(allegiance.ofXuid(key))
  const present = ROLES.filter((role) => reading[role] !== null)
  return (
    <FicheSection title={text.title}>
      {present.length === 0 ? (
        <p className="rounded border border-dashed border-border px-2 py-1.5 text-muted-foreground">
          {text.empty}
        </p>
      ) : (
        <FicheRows>
          {present.map((role) => {
            const held = reading[role] as AntagonistReading
            return (
              <FicheRow
                key={role}
                label={text[role].label}
                aside={
                  <span title={text[role].hintFmt(subject)} className="inline-flex">
                    <AntagonistIcon role={role} size={ICON_PX} />
                  </span>
                }
              >
                <Value role={role} reading={held} nameOf={nameOf} inkOf={inkOf} text={text} />
              </FicheRow>
            )
          })}
        </FicheRows>
      )}
    </FicheSection>
  )
}
