/**
 * LeveragePanel — Section C : leviers agrégats LUSR + suggestions de défis.
 *
 * Affiche les top 2 leviers (ProfileService.C) et les défis suggérés.
 * Distinct des leviers Pattern Engine (LeverList) — cf. plan §architecture.
 */
import type { ProgressionLeverage, SuggestedChallenge } from './types'
import type { AscensionText } from './i18n'

interface LeveragePanelProps {
  leverages: ProgressionLeverage[]
  challenges: SuggestedChallenge[]
  locale: 'fr' | 'en'
  t: AscensionText
}

export function LeveragePanel({ leverages, challenges, locale, t }: LeveragePanelProps) {
  const top = leverages.slice(0, 2)
  const suggested = challenges.slice(0, 3)

  if (top.length === 0 && suggested.length === 0) return null

  return (
    <div className="space-y-3">
      {top.length > 0 && (
        <div>
          <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            {t.profileLeveragesTitle}
          </p>
          <div className="space-y-2">
            {top.map((lev) => (
              <LeverageCard key={lev.component} leverage={lev} t={t} />
            ))}
          </div>
        </div>
      )}

      {suggested.length > 0 && (
        <div>
          <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            {t.profileSuggestedChallenges}
          </p>
          <div className="flex flex-wrap gap-2">
            {suggested.map((ch) => (
              <ChallengePill key={ch.template_id} challenge={ch} locale={locale} />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function LeverageCard({ leverage: lev, t }: { leverage: ProgressionLeverage; t: AscensionText }) {
  const pct = Math.min(Math.round(lev.leverage_value * 100), 100)
  const label = t.lusrComponent?.[lev.component] ?? lev.component

  return (
    <div className="flex items-center gap-3 rounded-md border border-border bg-card p-2">
      <div className="flex-1">
        <p className="text-sm font-medium">{label}</p>
        <p className="text-xs text-muted-foreground">
          {lev.narrative_axes.join(', ')}
        </p>
      </div>
      <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-xs font-bold text-primary">
        +{pct}%
      </div>
    </div>
  )
}

function ChallengePill({
  challenge: ch,
  locale,
}: {
  challenge: SuggestedChallenge
  locale: 'fr' | 'en'
}) {
  const label = locale === 'fr' ? ch.label_fr : ch.label_en
  const tierColor =
    ch.target_tier === 'legendary'
      ? 'border-amber-500/50 bg-amber-500/10 text-amber-700 dark:text-amber-300' // color-allow: prestige tier badge — CLAUDE.md §20
      : ch.target_tier === 'heroic'
        ? 'border-purple-500/50 bg-purple-500/10 text-purple-700 dark:text-purple-300' // color-allow: prestige tier badge — CLAUDE.md §20
        : 'border-border bg-muted text-muted-foreground'

  return (
    <span
      className={`inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium ${tierColor}`}
    >
      {label ?? ch.template_id}
    </span>
  )
}
