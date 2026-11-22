/**
 * FicheGroupes — LES DEUX GROUPES DU BAS DE LA FICHE, séparés d'un filet : « TOTAUX DU MATCH » (les chiffres
 * officiels de fin de match lus sur la feuille : précision, dégâts, score, éliminations à la tête — pas d'armes,
 * elles sont dans le tableau des éliminations) et « GESTES » (temps cumulé de sprint et de déplacement, sauts estimés,
 * rechargements, avec l'instant lu à côté du titre).
 *
 * Une valeur que la feuille ou le film ne donne pas n'a pas de ligne ; le zéro d'un geste non plus. Le rendu ne
 * lit rien.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../../i18n/i18n'
import { formatCount, formatDuration, formatPercent } from '../../model/ficheFormat'
import type { GesturesView } from '../../model/playerInstantView'
import type { PlayerMatchTotals } from '../../model/playerMatchTotals'
import { EstimateCapsule, FicheRow, FicheRows, FicheSection, Note } from './ficheParts'

export function FicheTotaux({ totals, locale }: { totals: PlayerMatchTotals; locale: ReplayLocale }) {
  const t = REPLAY_TEXT[locale].playerInstant
  const count = (n: number) => formatCount(n, locale)
  const { accuracy, damage } = totals
  return (
    <FicheSection title={t.matchTotals} kind="groupe">
      <FicheRows>
        {accuracy && (
          <FicheRow label={t.shotAccuracy}>
            {formatPercent(accuracy.pct, locale)}
            {accuracy.hit !== null && (
              <Note>
                {count(accuracy.hit)} / {count(accuracy.fired)}
              </Note>
            )}
          </FicheRow>
        )}
        {damage && (
          <FicheRow label={t.damageDealtTaken}>
            {count(damage.dealt)} / {count(damage.taken)}
          </FicheRow>
        )}
        {totals.score !== null && <FicheRow label={t.score}>{count(totals.score)}</FicheRow>}
        {totals.headshotKills !== null && <FicheRow label={t.headshotKills}>{count(totals.headshotKills)}</FicheRow>}
      </FicheRows>
    </FicheSection>
  )
}

export function FicheGestes({ gestures, clock, locale }: { gestures: GesturesView; clock: string; locale: ReplayLocale }) {
  const t = REPLAY_TEXT[locale].playerInstant
  return (
    <FicheSection
      title={t.gestures}
      kind="groupe"
      aside={<small className="font-normal normal-case tabular-nums text-muted-foreground">{clock}</small>}
    >
      <FicheRows>
        {gestures.sprintMs !== null && <FicheRow label={t.sprintTotal}>{formatDuration(gestures.sprintMs)}</FicheRow>}
        {gestures.moveMs !== null && <FicheRow label={t.moveTotal}>{formatDuration(gestures.moveMs)}</FicheRow>}
        {gestures.jumps !== null && (
          <FicheRow label={t.jumps} aside={<EstimateCapsule text={t.estimate} hint={t.estimateHint} />}>
            {formatCount(gestures.jumps, locale)}
          </FicheRow>
        )}
        {gestures.reloads !== null && (
          <FicheRow
            label={t.reloads}
            aside={gestures.reloadsMinimum ? <EstimateCapsule text={t.estimate} hint={t.reloadsHint} /> : undefined}
          >
            {formatCount(gestures.reloads, locale)}
          </FicheRow>
        )}
      </FicheRows>
    </FicheSection>
  )
}
