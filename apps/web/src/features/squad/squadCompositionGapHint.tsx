/**
 * squadCompositionGapHint — contenu de l'info-bulle qui EXPLIQUE l'écart
 * "composition exacte" publié sur la L2 (ADR 0033, D1 : phase A3 du plan
 * `.ai/PLAN_ESCOUADE_HORS_CADRE_2026-09-09.md`).
 *
 * La L2 (`PeriodSessionRail`) affiche déjà « 4 sur 7 matchs » quand la
 * composition exacte écarte des matchs d'une session (chantier A2). Ce module
 * construit le DÉTAIL survolable : la liste des matchs écartés, datés,
 * cartés, et nommant le(s) coéquipier(s) responsable(s) — sans quoi l'écart
 * est visible mais pas EXPLIQUÉ (critère de succès n°3 du plan).
 *
 * Reste un simple bâtisseur de `ReactNode` : le rendu passe par le patron
 * d'aide d'en-tête déjà en place (`InfoTooltip`, convention V73-L2 2.4c) —
 * aucun nouveau primitif de tooltip n'est introduit ici.
 */
import type { ReactNode } from 'react'
import { formatDate } from '@/lib/formatters/date'
import { intlLocale } from '@/lib/formatters/intlLocale'
import type { Locale } from '@/lib/i18n/locale'
import type { SquadText } from './i18n'
import type { SquadSessionExcludedMatch } from './squadSessionCounts'

/** Joint les coéquipiers responsables d'UN match écarté en un libellé unique
 *  ("Nilton410" / "Nilton410 et passivemarquise") ; repli textuel quand aucun
 *  coéquipier connu n'est identifiable (jamais une ligne muette). */
function culpritsLabelFor(match: SquadSessionExcludedMatch, t: SquadText['compositionGap'], locale: Locale): {
  label: string
  count: number
} {
  const names = match.extra_gamertags
  if (names && names.length > 0) {
    return { label: names.join(locale === 'fr' ? ' et ' : ' and '), count: names.length }
  }
  return { label: t.culpritUnknown, count: 1 }
}

/**
 * Construit le contenu de l'info-bulle pour UNE session. `undefined` quand
 * aucun match n'est écarté (pas de tooltip à afficher — la L2 n'a alors rien
 * à expliquer).
 */
export function buildCompositionGapHint(
  excluded: SquadSessionExcludedMatch[],
  locale: Locale,
  t: SquadText['compositionGap'],
): ReactNode | undefined {
  if (excluded.length === 0) return undefined
  const dateLocale = intlLocale(locale)
  return (
    <div className="space-y-1.5">
      <p className="font-medium">{t.heading(excluded.length)}</p>
      <ul className="space-y-1">
        {excluded.map((match) => {
          const dateLabel = formatDate(match.start_time, dateLocale, { day: 'numeric', month: 'short' })
          const culprits = culpritsLabelFor(match, t, locale)
          return (
            <li key={match.match_id}>{t.excludedLine(dateLabel, match.map_ui, culprits.label, culprits.count)}</li>
          )
        })}
      </ul>
    </div>
  )
}
