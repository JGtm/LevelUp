/**
 * FicheDepuisDebut — « DEPUIS LE DÉBUT DU MATCH » : ce que le joueur a fait jusqu'à l'instant lu, une ligne par
 * valeur. FDA (la valeur du jeu, jamais un quotient), séries, tirs à la tête, éliminations parfaites,
 * réapparitions, vie en cours et moyenne des vies, boucliers perdus, score et place dans le lobby.
 *
 * Une valeur que la lecture ne donne pas n'a pas de ligne : ni tiret ni texte de remplacement. Le rendu ne lit rien.
 */
import type { ReactNode } from 'react'

import { formatKDA } from '@/lib/formatters/number'

import { REPLAY_TEXT, type ReplayLocale } from '../../i18n/i18n'
import { formatCount, formatDuration } from '../../model/ficheFormat'
import type { SinceView } from '../../model/playerInstantView'
import { FicheRow, FicheRows, FicheSection, Note } from './ficheParts'

/** Une ligne à écrire quand sa valeur existe ; `null` sinon. */
function when<T>(value: T | null, write: (v: T) => ReactNode): ReactNode | null {
  return value === null ? null : write(value)
}

export function FicheDepuisDebut({ since, locale }: { since: SinceView; locale: ReplayLocale }) {
  const t = REPLAY_TEXT[locale].playerInstant
  const count = (n: number) => formatCount(n, locale)
  const rows: [string, ReactNode | null][] = [
    [t.fda, when(since.fda, (v) => formatKDA(v, locale))],
    [t.currentStreak, when(since.currentStreak, count)],
    [t.bestStreak, when(since.bestStreak, count)],
    [t.headshots, when(since.headshots, count)],
    [t.perfects, when(since.perfects, count)],
    [t.respawns, when(since.respawns, count)],
    [t.currentLife, when(since.currentLifeMs, formatDuration)],
    [t.averageLife, when(since.averageLifeMs, formatDuration)],
    [t.shieldsLost, when(since.shieldsLost, count)],
    [
      t.score,
      when(since.score, (v) => (
        <>
          {count(v)}
          {since.rank && <Note>{t.rankFmt(since.rank.place, since.rank.of)}</Note>}
        </>
      )),
    ],
  ]
  return (
    <FicheSection title={t.sinceStart}>
      <FicheRows>
        {rows.map(([label, value]) =>
          value === null ? null : (
            <FicheRow key={label} label={label}>
              {value}
            </FicheRow>
          ),
        )}
      </FicheRows>
    </FicheSection>
  )
}
