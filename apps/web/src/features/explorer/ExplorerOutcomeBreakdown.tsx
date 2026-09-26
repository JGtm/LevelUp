/**
 * ExplorerOutcomeBreakdown — « Répartition des résultats » de l'encart cible, rendu
 * 2.A HORIZONTAL (maquette du 2026-09-21, décision utilisateur D17).
 *
 * UNE piste épaisse empilée (victoires / nuls / défaites) qui porte le compte ET la part
 * dans chaque segment quand il a la place — sinon repris en légende —, le taux de victoire
 * en chiffre d'appel au-dessus (couleur `outcome-win`), puis la bande des résultats
 * (`OutcomeSequenceTape`, déjà en service sur quatre pages) du plus ancien au plus récent.
 *
 * Remplace la `<OutcomeBar>` de 6 px : la plus fine de l'app portait trois grandeurs, le
 * nul à 7 % y était un trait invisible, et le taux — la seule chose qu'on vient chercher —
 * un petit texte gris poussé à droite. `OutcomeBar` reste en service ailleurs (Accueil,
 * Synthèse, briefing Explorer, carrousel de sessions) : elle n'est pas touchée.
 *
 * ORDRE de la bande : `common_matches` arrive du plus RÉCENT au plus ancien (Q19,
 * `ORDER BY start_time DESC`) — la bande l'inverse, comme l'Accueil et Escouade.
 */
import { OutcomeSequenceTape, type OutcomePoint } from '@/components/charts/OutcomeSequenceTape'
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { Locale } from '@/lib/i18n/locale'
import type { ExplorerCommonMatchRow } from '@/lib/api/types'

import { StackedTrack, type StackedTrackSegment } from '@/components/charts/StackedTrack'

/** Hauteur de la bande des résultats — plus basse que le défaut (100) : la carte est
 *  empilée avec « Part des assistances » dans une colonne du `lg:grid-cols-3`. */
const TAPE_HEIGHT_PX = 72

type OutcomeKind = 'win' | 'draw' | 'loss'

const OUTCOME_TOKEN: Record<OutcomeKind, SemanticToken> = {
  win: 'outcome-win',
  draw: 'outcome-draw',
  loss: 'outcome-loss',
}

const OUTCOME_LABEL_KEY: Record<OutcomeKind, ExplorerManifestKey> = {
  win: 'explorer.target_profile.outcome_wins',
  draw: 'explorer.target_profile.outcome_draws',
  loss: 'explorer.target_profile.outcome_losses',
}

export function ExplorerOutcomeBreakdown({
  wins,
  draws,
  losses,
  winRate,
  commonMatches,
  locale,
}: {
  wins: number
  draws: number
  losses: number
  winRate?: number | null
  /** Matchs communs servis par l'API (récent→ancien) ; vides → pas de bande. */
  commonMatches?: ExplorerCommonMatchRow[] | null
  locale: Locale
}) {
  const t = (key: ExplorerManifestKey, values?: Record<string, string | number>) =>
    formatMessage(explorerManifest, key, locale, values)
  const total = wins + draws + losses
  if (total === 0) return null
  const counts: Record<OutcomeKind, number> = { win: wins, draw: draws, loss: losses }
  const pct = (n: number) => `${Math.round((n / total) * 100)} %`
  const fmt = (n: number) => n.toLocaleString(locale)
  const kinds: OutcomeKind[] = ['win', 'draw', 'loss']

  const segments = kinds
    .filter((k) => counts[k] > 0)
    .map<StackedTrackSegment>((k) => ({
      key: k,
      widthPct: (counts[k] / total) * 100,
      color: tokenCssVar(OUTCOME_TOKEN[k]),
      label: `${fmt(counts[k])} · ${pct(counts[k])}`,
      tooltip: `${fmt(counts[k])} ${t(OUTCOME_LABEL_KEY[k])} · ${pct(counts[k])}`,
    }))

  const tapePoints = [...(commonMatches ?? [])]
    .reverse()
    .map<OutcomePoint>((m) => ({
      matchId: m.match_id,
      outcome: m.outcome ?? 'dnf',
      map: m.map_ui || undefined,
      mode: m.mode_ui || undefined,
    }))

  return (
    <div className="flex flex-col gap-2" data-testid="explorer-outcome-breakdown">
      {winRate != null && (
        <div className="flex items-baseline gap-2">
          <span
            className="font-mono text-3xl font-bold leading-none tabular-nums"
            style={{ color: tokenCssVar('outcome-win') }}
          >
            {`${(winRate * 100).toLocaleString(locale, { maximumFractionDigits: 1 })} %`}
          </span>
          <span className="text-2xs text-muted-foreground">
            {t('explorer.target_profile.outcome_win_rate_caption')}
          </span>
        </div>
      )}
      <StackedTrack
        segments={segments}
        ariaLabel={kinds.map((k) => `${fmt(counts[k])} ${t(OUTCOME_LABEL_KEY[k])}`).join(' · ')}
        testId="explorer-outcome-track"
      />
      <ul className="flex flex-wrap items-center gap-x-3 gap-y-1 text-2xs text-muted-foreground">
        {kinds.map((k) => (
          <li key={k} className="flex items-center gap-1">
            <span
              className="h-2 w-2 rounded-full"
              style={{ backgroundColor: tokenCssVar(OUTCOME_TOKEN[k]) }}
              aria-hidden="true"
            />
            <span>
              {fmt(counts[k])} {t(OUTCOME_LABEL_KEY[k])}
            </span>
          </li>
        ))}
      </ul>
      {tapePoints.length > 0 && (
        <div data-testid="explorer-outcome-tape">
          <p className="mb-1 text-3xs text-muted-foreground">
            {t('explorer.target_profile.outcome_tape_caption', { count: tapePoints.length })}
          </p>
          <OutcomeSequenceTape
            matches={tapePoints}
            labels={{
              win: t('explorer.target_profile.outcome_wins'),
              loss: t('explorer.target_profile.outcome_losses'),
              tie: t('explorer.target_profile.outcome_draws'),
              dnf: t('explorer.target_profile.outcome_dnf'),
            }}
            height={TAPE_HEIGHT_PX}
          />
        </div>
      )}
    </div>
  )
}
