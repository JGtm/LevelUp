/**
 * MatchStatCards — composants KPI pour le Match View.
 *
 * C3 : StatExpectedCard     — réel vs attendu (K/D/A)
 * C4 : MatchRankBadge       — delta rang CSR/LUSR après match
 * C5 : KdIndicatorCard      — K/D vs nemesis principal
 * C6 : MatchVsStatCard      — générique "X vs Y + delta" (MMR, frags, morts, vie)
 * C7 : MatchSummaryCardsSection — grille 4 cartes onglet Résumé
 *
 * NATIVE_COMPONENTS items C3, C4, C5, C6, C7.
 */
import type { MatchViewRank, MatchExpectedStats, MatchNemesisRow, MatchSummaryKpis } from '@/lib/api/types'
import { skillDeltaScale, kdScale } from '@/lib/accessibility/scales'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { formatRankDelta } from '@/lib/formatters'
import {
  formatOffensiveConversion,
  formatDefensiveResistance,
  combatYieldToken,
} from '@/lib/formatters/combatYield'
import { KpiCard } from '@/components/cards/KpiCard'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { formatMessage } from '@/lib/i18n/format'
import { matchViewManifest, type MatchViewManifestKey } from '@/lib/i18n/generated/match_view'
import { useAppShellStore } from '@/stores/appShellStore'
import { useProvidesDamageTaken, useProvidesTeamMmr } from '@/lib/damage/effectiveHp'

// ---------------------------------------------------------------------------
// C3 — StatExpectedCard (réel vs attendu)
// ---------------------------------------------------------------------------

interface StatExpectedCardProps {
  label: string
  actual: number | null
  expected: number | null
  lowerIsBetter?: boolean
  /** Si false, card grisée */
  hasData: boolean
}

export function StatExpectedCard({ label, actual, expected, lowerIsBetter = false, hasData }: StatExpectedCardProps) {
  const delta = actual != null && expected != null ? actual - expected : null
  const isFavorable =
    delta == null ? null : lowerIsBetter ? delta < 0 : delta > 0
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: MatchViewManifestKey) => formatMessage(matchViewManifest, key, locale)

  return (
    <div
      className={`rounded-lg border px-4 py-3 text-center ${
        hasData ? 'border-border bg-card' : 'border-border/40 bg-card/50 opacity-50'
      }`}
    >
      <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">{label}</p>
      <p className="text-2xl font-bold text-foreground">{actual ?? '—'}</p>
      {hasData && expected != null ? (
        <p className="text-xs text-muted-foreground mt-0.5">
          {t('match_view.cards.expected_prefix')} {expected.toFixed(1)}{' '}
          {delta != null && (
            <span
              className="font-semibold"
              style={{
                color: isFavorable
                  ? tokenCssVar('divergent-pos')
                  : isFavorable === false
                    ? tokenCssVar('divergent-neg')
                    : undefined,
              }}
            >
              ({delta > 0 ? '+' : ''}{delta.toFixed(1)})
            </span>
          )}
        </p>
      ) : (
        <p className="text-xs text-muted-foreground mt-0.5">{t('match_view.cards.no_csr_data')}</p>
      )}
    </div>
  )
}

interface ExpectedCardsSectionProps {
  kpis: { kills: number | null; deaths: number | null; assists: number | null }
  expectedStats: MatchExpectedStats
}

export function ExpectedCardsSection({ kpis, expectedStats }: ExpectedCardsSectionProps) {
  const { expected_kills, expected_deaths, expected_assists } = expectedStats
  // hasExpected dérivé de la présence d'au moins une stat attendue. Le champ
  // has_expected_data a été retiré du contrat (Phase 3a-B) : la dérivation est
  // strictement équivalente (= ExpectedKills!=nil || ... côté Go).
  const hasExpected = expected_kills != null || expected_deaths != null || expected_assists != null
  const { data: fieldMappings } = useFieldMappings()
  const labelOf = (key: string): string =>
    fieldMappings?.fields[key]?.label ?? key
  return (
    <div className="grid grid-cols-3 gap-3">
      <StatExpectedCard
        label={labelOf('kills')}
        actual={kpis.kills}
        expected={expected_kills ?? null}
        lowerIsBetter={false}
        hasData={hasExpected}
      />
      <StatExpectedCard
        label={labelOf('deaths')}
        actual={kpis.deaths}
        expected={expected_deaths ?? null}
        lowerIsBetter={true}
        hasData={hasExpected}
      />
      <StatExpectedCard
        label={labelOf('assists')}
        actual={kpis.assists}
        expected={expected_assists ?? null}
        lowerIsBetter={false}
        hasData={hasExpected}
      />
    </div>
  )
}

// ---------------------------------------------------------------------------
// C4 — MatchRankBadge (delta rang CSR/LUSR)
// ---------------------------------------------------------------------------

interface MatchRankBadgeProps {
  rank: MatchViewRank
  hadBotTeammate?: boolean
}

export function MatchRankBadge({ rank, hadBotTeammate = false }: MatchRankBadgeProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: MatchViewManifestKey) => formatMessage(matchViewManifest, key, locale)
  if (rank.rating_type === 'none' || !rank.tier_label) return null

  const delta = rank.delta_value
  const deltaToken = delta != null ? skillDeltaScale(delta) : null
  const deltaColor = deltaToken === 'divergent-neutral' || deltaToken === null
    ? 'text-muted-foreground'
    : undefined
  const deltaStyle = deltaToken && deltaToken !== 'divergent-neutral'
    ? { color: tokenCssVar(deltaToken) }
    : undefined

  return (
    <div className="flex items-center gap-3 rounded-lg border border-border bg-card px-4 py-3">
      <div className="flex-1">
        <p className="text-xs text-muted-foreground uppercase tracking-wide">{rank.rating_type}</p>
        <div className="flex items-baseline gap-2 mt-0.5">
          <span className="text-lg font-bold text-foreground">{rank.tier_label}</span>
          {rank.numeric_value != null && (
            <span className="text-sm text-muted-foreground">{rank.numeric_value.toFixed(0)} pts</span>
          )}
        </div>
      </div>
      {delta != null && (
        <span
          className={`text-xl font-semibold ${deltaColor ?? ''}`}
          style={deltaStyle}
        >
          {formatRankDelta(delta, rank.rating_type)}
        </span>
      )}
      {hadBotTeammate && (
        <span className="ml-2 rounded bg-warning/20 px-2 py-0.5 text-2xs text-warning" title={t('match_view.cards.bot_teammate_title')}>
          🤖 bot
        </span>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// C5 — KdIndicatorCard (K/D vs nemesis)
// ---------------------------------------------------------------------------

interface KdIndicatorCardProps {
  nemesis: MatchNemesisRow | null
}

export function KdIndicatorCard({ nemesis }: KdIndicatorCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: MatchViewManifestKey) => formatMessage(matchViewManifest, key, locale)
  if (!nemesis) return null

  const kd = nemesis.killed_me > 0 ? nemesis.i_killed / nemesis.killed_me : nemesis.i_killed

  return (
    <div className="rounded-lg border border-border bg-card px-4 py-3">
      <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">
        {t('match_view.cards.kd_vs_nemesis')}
      </p>
      <div className="flex items-baseline gap-2">
        <span
          className="text-2xl font-bold"
          style={{ color: tokenCssVar(kdScale(kd)) }}
        >{kd.toFixed(2)}</span>
        <span className="text-sm text-muted-foreground">vs {nemesis.gamertag}</span>
      </div>
      <p className="text-xs text-muted-foreground mt-0.5">
        {nemesis.i_killed} kills · {nemesis.killed_me} morts
      </p>
    </div>
  )
}

// ---------------------------------------------------------------------------
// C6 — MatchVsStatCard (générique "X vs Y + delta")
// ---------------------------------------------------------------------------

interface MatchVsStatCardProps {
  label: string
  /** Valeur principale (gauche ou seule) */
  primary: number | string | null
  /** Valeur secondaire (droite, optionnelle) */
  secondary?: number | string | null
  /** Libellé sous la valeur primaire */
  primaryLabel?: string
  /** Libellé sous la valeur secondaire */
  secondaryLabel?: string
  /** Delta affiché en dessous */
  delta?: number | null
  /** Si true, un delta négatif est favorable (morts, durée de vie basse) */
  lowerIsBetter?: boolean
  /** Formater la valeur (ex. décimales) */
  precision?: number
  /**
   * Accent FIXE (type 2 du catalogue) appliqué quand il n'y a pas de delta —
   * ex. métrique sans comparaison (vie moyenne). Ignoré dès qu'un delta existe
   * (l'accent dynamique type 4 prend le dessus).
   */
  fixedAccent?: SemanticToken
}

export function MatchVsStatCard({
  label,
  primary,
  secondary,
  primaryLabel,
  secondaryLabel,
  delta,
  lowerIsBetter = false,
  precision = 0,
  fixedAccent,
}: MatchVsStatCardProps) {
  const fmt = (v: number | string | null | undefined) => {
    if (v == null) return '—'
    if (typeof v === 'string') return v
    return precision > 0 ? v.toFixed(precision) : Math.round(v).toString()
  }

  const isFavorable =
    delta == null ? null : lowerIsBetter ? delta < 0 : delta > 0

  // Accent dynamique (type 4 du catalogue) : barre 3px verte si favorable,
  // rouge si défavorable. Sans delta, on retombe sur l'accent fixe éventuel
  // (type 2 — ex. barre neutre pour une métrique sans comparaison).
  const accent: SemanticToken | undefined =
    isFavorable === null ? fixedAccent : isFavorable ? 'divergent-pos' : 'divergent-neg'

  const deltaStyle =
    isFavorable === null
      ? undefined
      : { color: tokenCssVar(isFavorable ? 'divergent-pos' : 'divergent-neg') }

  return (
    <KpiCard accent={accent} className="h-full">
      <div className="px-3 py-2.5">
        <p className="text-2xs text-muted-foreground uppercase tracking-wide mb-1.5">{label}</p>
        <div className="flex items-baseline gap-1.5">
          <div>
            <span className="text-lg font-bold text-foreground leading-none">{fmt(primary)}</span>
            {primaryLabel && (
              <p className="text-2xs text-muted-foreground mt-0.5">{primaryLabel}</p>
            )}
          </div>
          {secondary != null && (
            <>
              <span className="text-muted-foreground text-xs font-light">vs</span>
              <div>
                <span className="text-lg font-bold text-foreground leading-none">{fmt(secondary)}</span>
                {secondaryLabel && (
                  <p className="text-2xs text-muted-foreground mt-0.5">{secondaryLabel}</p>
                )}
              </div>
            </>
          )}
          {delta != null && (
            <span className="ml-auto text-xs font-semibold" style={deltaStyle}>
              {delta > 0 ? '+' : ''}{fmt(delta)}
            </span>
          )}
        </div>
      </div>
    </KpiCard>
  )
}

// ---------------------------------------------------------------------------
// C8 — MatchWinProbCard (résultat attendu : proba de victoire pré-match)
// ---------------------------------------------------------------------------

interface MatchWinProbCardProps {
  /** Proba de victoire pré-match de l'équipe du joueur (LUSR v2, 0..1). Toujours
   *  fournie : le call site ne rend PAS la card quand la proba est absente (une card
   *  grisée « pas d'estimation » n'apporte rien — DEC-3). */
  winProb: number
}

export function MatchWinProbCard({ winProb }: MatchWinProbCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: MatchViewManifestKey) => formatMessage(matchViewManifest, key, locale)
  const pct = Math.round(winProb * 100)

  const qualitativeKey: MatchViewManifestKey =
    pct >= 55
      ? 'match_view.cards.win_prob_favorite'
      : pct <= 45
        ? 'match_view.cards.win_prob_underdog'
        : 'match_view.cards.win_prob_balanced'

  const valueColor =
    pct > 45 && pct < 55
      ? undefined
      : pct >= 55
        ? tokenCssVar('divergent-pos')
        : tokenCssVar('divergent-neg')

  // Accent dynamique (type 4) : barre 3px verte si favori, rouge si outsider,
  // absente si équilibré.
  const accent: SemanticToken | undefined =
    pct > 45 && pct < 55 ? undefined : pct >= 55 ? 'divergent-pos' : 'divergent-neg'

  return (
    <KpiCard accent={accent} className="h-full">
      <div className="px-3 py-2.5">
        <p className="text-2xs text-muted-foreground uppercase tracking-wide mb-1.5">
          {t('match_view.cards.expected_result')}
        </p>
        <div className="flex items-baseline gap-2">
          <span
            className="text-lg font-bold text-foreground leading-none"
            style={valueColor ? { color: valueColor } : undefined}
          >
            {`${pct} %`}
          </span>
          <span className="text-2xs text-muted-foreground">{t('match_view.cards.win_prob_label')}</span>
        </div>
        <p className="text-2xs text-muted-foreground mt-0.5">{t(qualitativeKey)}</p>
      </div>
    </KpiCard>
  )
}

// ---------------------------------------------------------------------------
// C7 — MatchSummaryCardsSection (grille cartes onglet Résumé)
// ---------------------------------------------------------------------------

interface MatchSummaryCardsSectionProps {
  kpis: MatchSummaryKpis
  expectedStats: MatchExpectedStats
  /** Rendement (OffensiveConversion) du joueur — ligne is_me du scoreboard. */
  offensiveConversion?: number | null
  /** Résistance (DefensiveResistance) du joueur — ligne is_me du scoreboard. */
  defensiveResistance?: number | null
  /** Dégâts/frag (DamageDealt / kills) du joueur — sous-valeur du Rendement, comme la hero KPI accueil. */
  damagePerKill?: number | null
  /** Dégâts/mort (DamageTaken / morts) du joueur — sous-valeur de la Résistance. */
  damagePerDeath?: number | null
}

export function MatchSummaryCardsSection({
  kpis,
  expectedStats,
  offensiveConversion,
  defensiveResistance,
  damagePerKill,
  damagePerDeath,
}: MatchSummaryCardsSectionProps) {
  const { expected_kills, expected_deaths, expected_assists, expected_win_prob } = expectedStats
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: MatchViewManifestKey, vars?: Record<string, unknown>) =>
    formatMessage(matchViewManifest, key, locale, vars)
  // false (Halo 5 : API sans damage_taken) → Résistance non calculable : carte N/A
  // (au lieu de 0% trompeur), sans sous-valeur dégâts/mort ni accent rouge.
  // Défaut true (Infinite) → carte Résistance inchangée.
  const providesDamageTaken = useProvidesDamageTaken()
  // Capability MMR d'équipe : Halo 5 ne porte PAS de MMR (carnage cryptum sans
  // MMR équipe/adverse → no_team_mmr). On masque entièrement la card MMR pour
  // les titres sans la capability (au lieu d'afficher une card vide/à tiret).
  const providesTeamMmr = useProvidesTeamMmr()

  // Dégâts/frag (Rendement) et dégâts/mort (Résistance), arrondis comme l'Explorer
  // et la hero KPI de l'accueil. Affichés en sous-valeur sous le pourcentage.
  const dmgPerKillLabel =
    damagePerKill != null && Number.isFinite(damagePerKill)
      ? t('match_view.cards.yield_dmg_per_kill', { n: Math.round(damagePerKill) })
      : undefined
  const dmgPerDeathLabel =
    providesDamageTaken && damagePerDeath != null && Number.isFinite(damagePerDeath)
      ? t('match_view.cards.yield_dmg_per_death', { n: Math.round(damagePerDeath) })
      : undefined

  // expected K/D : valeurs du back directement. Pour Halo 5 (pas d'API skill), elles
  // viennent du modèle local count∝durée (TrueSkill 2-like, validé +13%/+5% sur 3135
  // matchs) ; sinon de l'API skill. Le flag locally_estimated pilote le label.
  const killsDelta =
    kpis.kills != null && expected_kills != null ? kpis.kills - expected_kills : null

  const deathsDelta =
    kpis.deaths != null && expected_deaths != null ? kpis.deaths - expected_deaths : null

  const assistsDelta =
    kpis.assists != null && expected_assists != null ? kpis.assists - expected_assists : null

  // « Estimé localement » : flag backend — les expected K/D viennent du modèle
  // local count∝durée (Halo 5, pas d'API skill), pas de l'API. On le signale
  // honnêtement plutôt que de les présenter au même rang que des valeurs API.
  const locallyEstimated = expectedStats.locally_estimated ?? false

  return (
    <div className="space-y-2">
      {/* Répartition fluide : auto-fit + minmax(1fr) → les cartes RÉELLEMENT
          rendues se partagent équitablement la largeur, sans trous quand des
          cartes sont masquées (H5 : pas de MMR/Résistance, éventuellement pas de
          Résultat attendu). Remplace la grille fixe xl:grid-cols-8 qui laissait
          des colonnes vides à droite. */}
      <div className="grid grid-cols-[repeat(auto-fit,minmax(9rem,1fr))] gap-3">
      {providesTeamMmr && (
        <MatchVsStatCard
          label={t('match_view.cards.mmr_team_vs_enemy')}
          primary={kpis.team_mmr ?? null}
          secondary={kpis.enemy_mmr ?? null}
          primaryLabel={t('match_view.cards.label_ally')}
          secondaryLabel={t('match_view.cards.label_enemy')}
          delta={kpis.delta_mmr ?? null}
          lowerIsBetter={false}
          precision={0}
        />
      )}
      <MatchVsStatCard
        label={t('match_view.cards.frags_vs_expected')}
        primary={kpis.kills ?? null}
        secondary={expected_kills != null ? Math.round(expected_kills) : null}
        primaryLabel={t('match_view.cards.label_real')}
        secondaryLabel={t('match_view.cards.label_expected')}
        delta={killsDelta}
        lowerIsBetter={false}
        precision={0}
      />
      <MatchVsStatCard
        label={t('match_view.cards.deaths_vs_expected')}
        primary={kpis.deaths ?? null}
        secondary={expected_deaths != null ? Math.round(expected_deaths) : null}
        primaryLabel={t('match_view.cards.label_real')}
        secondaryLabel={t('match_view.cards.label_expected')}
        delta={deathsDelta}
        lowerIsBetter={true}
        precision={0}
      />
      <MatchVsStatCard
        label={t('match_view.cards.assists_vs_expected')}
        primary={kpis.assists ?? null}
        secondary={expected_assists != null ? Math.round(expected_assists) : null}
        primaryLabel={t('match_view.cards.label_real')}
        secondaryLabel={t('match_view.cards.label_expected')}
        delta={assistsDelta}
        lowerIsBetter={false}
        precision={0}
      />
      {expected_win_prob != null && Number.isFinite(expected_win_prob) && (
        <MatchWinProbCard winProb={expected_win_prob} />
      )}
      <MatchVsStatCard
        label={t('match_view.cards.avg_life')}
        primary={kpis.average_life ?? null}
        fixedAccent="divergent-neutral"
      />
      <MatchVsStatCard
        label={t('match_view.cards.rendement')}
        primary={formatOffensiveConversion(offensiveConversion)}
        primaryLabel={dmgPerKillLabel}
        fixedAccent={combatYieldToken(offensiveConversion, null)}
      />
      {providesDamageTaken && (
        <MatchVsStatCard
          label={t('match_view.cards.resistance')}
          primary={formatDefensiveResistance(defensiveResistance)}
          primaryLabel={dmgPerDeathLabel}
          fixedAccent={combatYieldToken(null, defensiveResistance)}
        />
      )}
      </div>
      {locallyEstimated && (
        <p
          className="text-2xs italic text-muted-foreground"
          title={t('match_view.cards.locally_estimated_hint')}
        >
          {t('match_view.cards.locally_estimated')}
        </p>
      )}
    </div>
  )
}
