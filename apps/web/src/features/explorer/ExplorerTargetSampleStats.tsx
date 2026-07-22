/**
 * ExplorerTargetSampleStats — bloc « Répartition des frags » de la section "Sur N
 * matchs joués ensemble" de l'encart adversaire.
 *
 * Frag v2 COMPLET : quand le backend fournit `frag_distribution` (sunburst hiérarchique
 * classe→rôle), rend FragSunburst + « Outils de destruction » (FragWeaponBreakdown) —
 * MÊME rendu que les 5 autres surfaces (Synthesis/Match view/Timeseries/Sessions/Escouade),
 * survol LIÉ sunburst ↔ breakdown. Repli (cible sans données d'arme sur les matchs communs)
 * : donut kill-type (mêlée/lourde/grenade/autre) + top armes legacy. Ce fichier exporte
 * AUSSI ExplorerTargetSampleKpis (rangée KPI) et ExplorerTargetOutcome (bilan V/N/D) —
 * inchangés.
 *
 * Calcul local depuis common_matches (DuckDB), indépendant des tokens Halo.
 * Affichée seulement quand `sampleStats != null && sample_size > 0`.
 */
import { useState } from 'react'

import { OutcomeBar } from '@/components/ui/outcome-bar'
import { KillTypesDonut, type DonutSlice } from '@/components/charts/KillTypesDonut'
import { FragSunburst, FragClassLegend } from '@/components/charts/FragSunburst'
import { fragClassColor } from '@/lib/accessibility/scales'
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import { useProvidesDamageTaken } from '@/lib/damage/effectiveHp'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { ExplorerTargetSampleStats, ExplorerWeaponKill, FragDistribution, SynthesisWeaponKillEntry } from '@/lib/api/types'

/** Libellé universel (FR=EN) quand la Résistance n'est pas calculable faute de
 *  damage_taken (Halo 5). Aligné sur `notAvailable: 'N/A'` du module compare. */
const DR_NA_LABEL = 'N/A'

interface ExplorerTargetSampleStatsProps {
  sampleStats: ExplorerTargetSampleStats
}

type TFn = (key: ExplorerManifestKey, values?: Record<string, string | number>) => string

function fmtPctRatio(value: number, locale: string): string {
  return `${(value * 100).toLocaleString(locale, { maximumFractionDigits: 1 })}%`
}

function fmtNumber(value: number, locale: string, fractionDigits = 2): string {
  return value.toLocaleString(locale, { maximumFractionDigits: fractionDigits })
}

function fmtInt(value: number, locale: string): string {
  return value.toLocaleString(locale, { maximumFractionDigits: 0 })
}

export function ExplorerTargetSampleStats({ sampleStats }: ExplorerTargetSampleStatsProps) {
  const appLocale = useAppShellStore((s) => s.locale)
  const locale = intlLocale(appLocale)
  const t: TFn = (key, values) => formatMessage(explorerManifest, key, appLocale, values)

  // Frag v2 COMPLET : le backend fournit la répartition hiérarchique (sunburst
  // classe→rôle) → FragSunburst + « Outils de destruction », comme les 5 autres surfaces.
  // Sinon (cible sans données d'arme sur les matchs communs) → repli donut kill-type.
  const distribution = sampleStats.frag_distribution ?? null
  if ((distribution?.total_kills ?? 0) > 0) {
    return (
      <ExplorerTargetFragV2
        distribution={distribution}
        weaponKills={sampleStats.top_weapon_kills ?? []}
        appLocale={appLocale}
      />
    )
  }

  // ── Repli legacy : partition des frags par TYPE D'ARME (mutuellement exclusifs) :
  // melee / arme lourde / grenade / autres → la somme = total des frags. "Autres" =
  // frags à l'arme normale = kills - (melee + lourde + grenade).
  //
  // IMPORTANT : on ne soustrait PAS les headshots. Un headshot est ORTHOGONAL au
  // type d'arme (un frag à l'arme normale ou lourde peut être un headshot) ; le
  // compter ici ferait que le donut ne somme plus au total des frags. Les
  // "Tirs à la tête" restent exposés en KPI ("Taux de tête"), hors donut.
  const weaponTyped = sampleStats.melee_kills + sampleStats.power_weapon_kills + sampleStats.grenade_kills
  const other = Math.max(0, sampleStats.kills - weaponTyped)
  //
  // Couleurs : indices chart-series DISTINCTS (1/6/7/8) et NON 2-5 — dans la
  // palette par défaut 1..5 est un dégradé bleu/indigo séquentiel (illisible en
  // catégoriel, pas color-blind friendly). 1/6/7/8 sont distincts dans toutes les
  // palettes (Okabe-Ito CB incluse). Via tokenCssVar → suit la palette active.
  const donutSlices: DonutSlice[] = [
    { label: t('explorer.target_profile.kill_type_melee'), count: sampleStats.melee_kills, token: 'chart-series-1' as SemanticToken },
    { label: t('explorer.target_profile.kill_type_power_weapon'), count: sampleStats.power_weapon_kills, token: 'chart-series-6' as SemanticToken },
    { label: t('explorer.target_profile.kill_type_grenade'), count: sampleStats.grenade_kills, token: 'chart-series-7' as SemanticToken },
    { label: t('explorer.target_profile.kill_type_other'), count: other, token: 'chart-series-8' as SemanticToken },
  ].filter((s) => s.count > 0)

  // Bloc « Répartition des frags » : titre en barre (chrome ChartCard) + donut
  // agrandi centré. Hauteur naturelle : le bloc cohabite avec le bilan V/N/D dans
  // la même colonne (cf. ExplorerTargetProfileCard). Titre de section + rangée KPI
  // rendus hors bloc.
  const topWeapons = sampleStats.top_weapons ?? []
  return (
    <div className="flex flex-col overflow-hidden rounded-lg border border-border bg-card">
      <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">
        {t('explorer.target_profile.label_kill_types')}
      </div>
      {/* Donut (gauche) + top 3 armes (droite) si dispo, sinon donut centré seul. */}
      <div className={`grid items-center gap-3 p-3 ${topWeapons.length > 0 ? 'sm:grid-cols-[2fr_1fr]' : ''}`}>
        <DonutColumn slices={donutSlices} locale={locale} t={t} />
        {topWeapons.length > 0 && <WeaponsTop weapons={topWeapons} locale={appLocale} t={t} />}
      </div>
    </div>
  )
}

// ─── Frag v2 : sunburst classe→rôle (« Répartition des frags ») + « Top armes » ──
//
// Encart adversaire COMPACT : UN SEUL bloc (titre « Répartition des frags ») — sunburst
// borné (maxWidthPx) + légende sous l'anneau à GAUCHE, et « Top armes » (top 5, recolorées
// par classe) à DROITE. Survol LIÉ sunburst ↔ liste (estompage des armes hors classe
// survolée). Le repli legacy (donut) est géré par l'appelant si pas de frag_distribution.
function ExplorerTargetFragV2({
  distribution,
  weaponKills,
  appLocale,
}: {
  distribution: FragDistribution | null
  weaponKills: SynthesisWeaponKillEntry[]
  appLocale: 'fr' | 'en'
}) {
  const [hoveredClass, setHoveredClass] = useState<string | null>(null)
  const t: TFn = (key, values) => formatMessage(explorerManifest, key, appLocale, values)
  // « Top armes » : top 5 armes de la cible (weaponKills déjà trié kills desc côté backend).
  const topArmes = weaponKills.slice(0, 5)
  return (
    <div className="flex flex-col overflow-hidden rounded-lg border border-border bg-card">
      <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">
        {t('explorer.target_profile.label_kill_types')}
      </div>
      <div className="p-3">
        {/* Rangée : sunburst NU (gauche) + Top armes (droite, top 5). */}
        <div className={`grid items-center gap-3 ${topArmes.length > 0 ? 'sm:grid-cols-[2fr_1fr]' : ''}`}>
          <FragSunburst
            distribution={distribution}
            externalHoveredClass={hoveredClass}
            onClassHover={setHoveredClass}
            hideCenterLabel
            maxWidthPx={480}
            legendSide="none"
            bare
          />
          {topArmes.length > 0 && (
            <TopArmes
              weapons={topArmes}
              title={t('explorer.target_profile.top_weapons_title')}
              locale={appLocale}
              hoveredClass={hoveredClass}
              onClassHover={setHoveredClass}
            />
          )}
        </div>
        {/* Légende des classes centrée EN BAS du bloc entier (hors rangée). */}
        <FragClassLegend
          distribution={distribution}
          hoveredClass={hoveredClass}
          onClassHover={setHoveredClass}
          className="mt-3"
        />
      </div>
    </div>
  )
}

// ─── « Top armes » v2 : top 5 armes recolorées par classe (survol lié sunburst) ──
function TopArmes({
  weapons,
  title,
  locale,
  hoveredClass,
  onClassHover,
}: {
  weapons: SynthesisWeaponKillEntry[]
  title: string
  locale: 'fr' | 'en'
  hoveredClass: string | null
  onClassHover: (c: string | null) => void
}) {
  const numberLocale = intlLocale(locale)
  const maxKills = Math.max(1, ...weapons.map((w) => w.kills))
  return (
    <div className="flex flex-col gap-2">
      <span className="text-2xs uppercase tracking-label-xl text-muted-foreground">{title}</span>
      <ol className="flex flex-col gap-2">
        {weapons.map((w, i) => {
          const pct = Math.round((w.kills / maxKills) * 100)
          const dim = hoveredClass != null && (w.class ?? '') !== hoveredClass
          return (
            <li
              key={`${w.label}-${i}`}
              className={`flex flex-col gap-1 transition-opacity ${dim ? 'opacity-40' : ''}`}
              onMouseEnter={() => onClassHover(w.class ?? null)}
              onMouseLeave={() => onClassHover(null)}
            >
              <div className="flex items-baseline justify-between gap-2">
                <span className="flex min-w-0 items-baseline gap-1.5">
                  <span className="text-2xs font-bold tabular-nums text-muted-foreground">{i + 1}</span>
                  <span className="truncate text-xs font-medium text-foreground">{w.label}</span>
                </span>
                <span className="flex-shrink-0 text-xs font-semibold tabular-nums text-foreground">
                  {w.kills.toLocaleString(numberLocale)}
                </span>
              </div>
              <div className="h-1 w-full overflow-hidden rounded-full bg-muted-foreground/15">
                <div className="h-full rounded-full" style={{ width: `${pct}%`, backgroundColor: fragClassColor(w.class) }} />
              </div>
            </li>
          )
        })}
      </ol>
    </div>
  )
}

// ─── Top armes (à droite du donut) ───────────────────────────────────────────

function WeaponsTop({ weapons, locale, t }: { weapons: ExplorerWeaponKill[]; locale: 'fr' | 'en'; t: TFn }) {
  const numberLocale = intlLocale(locale)
  const maxKills = Math.max(1, ...weapons.map((w) => w.kills))
  return (
    <div className="flex flex-col gap-2">
      <span className="text-2xs uppercase tracking-label-xl text-muted-foreground">
        {t('explorer.target_profile.top_weapons_title')}
      </span>
      <ol className="flex flex-col gap-2">
        {weapons.map((w, i) => {
          const name = locale === 'en' ? w.label_en || w.label_fr : w.label_fr || w.label_en
          const pct = Math.round((w.kills / maxKills) * 100)
          return (
            <li key={w.weapon_id} className="flex flex-col gap-1">
              <div className="flex items-baseline justify-between gap-2">
                <span className="flex min-w-0 items-baseline gap-1.5">
                  <span className="text-2xs font-bold tabular-nums text-muted-foreground">{i + 1}</span>
                  <span className="truncate text-xs font-medium text-foreground">{name}</span>
                </span>
                <span className="flex-shrink-0 text-xs font-semibold tabular-nums text-foreground">
                  {w.kills.toLocaleString(numberLocale)}
                </span>
              </div>
              <div className="h-1 w-full overflow-hidden rounded-full bg-muted-foreground/15">
                <div
                  className="h-full rounded-full"
                  style={{ width: `${pct}%`, backgroundColor: tokenCssVar('chart-series-1') }}
                />
              </div>
            </li>
          )
        })}
      </ol>
    </div>
  )
}

/**
 * ExplorerTargetOutcome — bilan V/N/D des matchs communs, rendu dans une section
 * séparée pleine largeur sous le donut + cadence (OutcomeBar + légende). nil si
 * aucun résultat exploitable.
 */
export function ExplorerTargetOutcome({ sampleStats }: ExplorerTargetSampleStatsProps) {
  const appLocale = useAppShellStore((s) => s.locale)
  const locale = intlLocale(appLocale)
  const t: TFn = (key, values) => formatMessage(explorerManifest, key, appLocale, values)
  if (sampleStats.wins + sampleStats.draws + sampleStats.losses === 0) return null
  return (
    <div className="overflow-hidden rounded-lg border border-border bg-card">
      <div className="border-b border-border px-3 py-2 text-sm font-medium">
        {t('explorer.target_profile.results_title')}
      </div>
      <div className="p-3">
        <OutcomeLegend sampleStats={sampleStats} locale={locale} t={t} />
      </div>
    </div>
  )
}

// ─── Colonne 1 : donut frags à connecteurs ──────────────────────────────────

function DonutColumn({ slices, locale, t }: { slices: DonutSlice[]; locale: string; t: TFn }) {
  return (
    <div className="flex w-full flex-col items-center gap-2">
      {slices.length > 0 ? (
        <KillTypesDonut slices={slices} locale={locale} />
      ) : (
        <span className="text-xs text-muted-foreground">{t('explorer.target_profile.value_unavailable')}</span>
      )}
    </div>
  )
}

// ─── Rangée KPI (hors bloc, sous le titre) ───────────────────────────────────

/**
 * ExplorerTargetSampleKpis — rangée de KPI cards rendue HORS du bloc, juste sous
 * le titre « Sur N matchs joués ensemble » (parité avec la rangée de tuiles de
 * « Carrière complète »). La dernière carte porte le rendement/résistance (OC/DR)
 * à la place de l'ancienne barre composite.
 */
export function ExplorerTargetSampleKpis({ sampleStats }: ExplorerTargetSampleStatsProps) {
  const appLocale = useAppShellStore((s) => s.locale)
  const locale = intlLocale(appLocale)
  const t: TFn = (key, values) => formatMessage(explorerManifest, key, appLocale, values)
  const providesDamageTaken = useProvidesDamageTaken()
  const dash = t('explorer.target_profile.value_unavailable')
  const pct = (v: number | null | undefined) => (v != null ? fmtPctRatio(v, locale) : dash)
  const num = (v: number | null | undefined) => (v != null ? fmtNumber(v, locale) : dash)

  const oc = sampleStats.offensive_conversion ?? null
  const dr = sampleStats.defensive_resistance ?? null
  const ocLabel = oc != null ? `${Math.round(oc * 100)}%` : dash
  // Halo 5 (API sans damage_taken) → Résistance non calculable : N/A au lieu d'un
  // « +0% » trompeur. Défaut true (Infinite) → libellé DR inchangé.
  const drLabel = !providesDamageTaken
    ? DR_NA_LABEL
    : dr == null
      ? dash
      : dr < 0
        ? '∞'
        : `${dr >= 1 ? '+' : ''}${Math.round((dr - 1) * 100)}%`
  const dmgPerKill = sampleStats.kills > 0 ? Math.round(sampleStats.damage_dealt / sampleStats.kills) : null
  // dégâts/mort = sous-valeur de la Résistance : null si DR non calculable (h5),
  // sinon « 0 dégâts/mort » trompeur (damage_taken absent côté API).
  const dmgPerDeath =
    providesDamageTaken && sampleStats.deaths > 0
      ? Math.round(sampleStats.damage_taken / sampleStats.deaths)
      : null

  // lg : KDA (1re piste) réduite à 0.7fr, Rendement/Résistance (dernière) élargie à 1.3fr.
  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-[0.7fr_1fr_1fr_1fr_1fr_1.3fr]">
      <SmallTile label={t('explorer.target_profile.label_kda')} value={num(sampleStats.kda)} accent="perf-tier-2" />
      <SmallTile label={t('explorer.target_profile.label_accuracy')} value={pct(sampleStats.accuracy)} accent="info" />
      <SmallTile label={t('explorer.target_profile.label_headshot_rate')} value={pct(sampleStats.headshot_rate)} accent="chart-series-1" />
      <SmallTile label={t('explorer.target_profile.label_avg_score')} value={sampleStats.avg_personal_score != null ? fmtInt(sampleStats.avg_personal_score, locale) : dash} accent="chart-series-4" />
      <SmallTile label={t('explorer.target_profile.label_perfect_kills')} value={fmtInt(sampleStats.perfect_kills ?? 0, locale)} accent="outcome-win" />
      <YieldTile ocLabel={ocLabel} drLabel={drLabel} dmgPerKill={dmgPerKill} dmgPerDeath={dmgPerDeath} t={t} />
    </div>
  )
}

/**
 * YieldTile — carte rendement (OC, vert) / résistance (DR, bleu) au format KPI card.
 * dmg/frag (gauche) et dmg/mort (droite) en pied, petit + gris (aux extrémités).
 */
function YieldTile({
  ocLabel,
  drLabel,
  dmgPerKill,
  dmgPerDeath,
  t,
}: {
  ocLabel: string
  drLabel: string
  dmgPerKill: number | null
  dmgPerDeath: number | null
  t: TFn
}) {
  return (
    <div className="overflow-hidden rounded-md border border-border bg-card">
      <div className="h-[3px]" style={{ backgroundColor: tokenCssVar('divergent-pos') }} />
      <div className="flex flex-col gap-1 px-2 py-1.5">
        <span className="text-2xs uppercase tracking-label-xl text-muted-foreground">
          {t('explorer.target_profile.yield_offensive')} / {t('explorer.target_profile.yield_defensive')}
        </span>
        {/* Une seule ligne : dmg/frag (gauche) · valeurs OC/DR (centre) · dmg/mort (droite). */}
        <div className="flex items-baseline justify-between gap-1">
          <span className="text-3xs text-muted-foreground">
            {dmgPerKill != null ? t('explorer.target_profile.yield_dmg_per_kill', { n: dmgPerKill }) : ''}
          </span>
          <span className="text-sm font-semibold">
            <span style={{ color: tokenCssVar('divergent-pos') }}>{ocLabel}</span>
            <span className="text-muted-foreground"> / </span>
            <span style={{ color: tokenCssVar('divergent-neutral') }}>{drLabel}</span>
          </span>
          <span className="text-3xs text-muted-foreground">
            {dmgPerDeath != null ? t('explorer.target_profile.yield_dmg_per_death', { n: dmgPerDeath }) : ''}
          </span>
        </div>
      </div>
    </div>
  )
}

// ─── OutcomeBar légendée (V / N / D + taux) ──────────────────────────────────

function OutcomeLegend({ sampleStats, locale, t }: { sampleStats: ExplorerTargetSampleStats; locale: string; t: TFn }) {
  const { wins, draws, losses, win_rate: winRate } = sampleStats
  if (wins + draws + losses === 0) return null
  return (
    <div className="flex flex-col gap-1.5">
      <OutcomeBar wins={wins} draws={draws} losses={losses} />
      <ul className="flex flex-wrap items-center gap-x-3 gap-y-1 text-2xs text-muted-foreground">
        <OutcomeLegendItem token="outcome-win" label={t('explorer.target_profile.outcome_wins')} value={fmtInt(wins, locale)} />
        <OutcomeLegendItem token="outcome-draw" label={t('explorer.target_profile.outcome_draws')} value={fmtInt(draws, locale)} />
        <OutcomeLegendItem token="outcome-loss" label={t('explorer.target_profile.outcome_losses')} value={fmtInt(losses, locale)} />
        {winRate != null && (
          <li className="ml-auto font-semibold text-foreground">{fmtPctRatio(winRate, locale)}</li>
        )}
      </ul>
    </div>
  )
}

function OutcomeLegendItem({ token, label, value }: { token: SemanticToken; label: string; value: string }) {
  return (
    <li className="flex items-center gap-1">
      <span className="h-2 w-2 rounded-full" style={{ backgroundColor: tokenCssVar(token) }} aria-hidden="true" />
      <span>{value} {label}</span>
    </li>
  )
}

// ─── Tuile générique ─────────────────────────────────────────────────────────

interface SmallTileProps {
  label: string
  value: string
  /** Couleur de la barre d'accent (3px) en haut — parité Synthesis AccentCard. */
  accent: SemanticToken
}

function SmallTile({ label, value, accent }: SmallTileProps) {
  return (
    <div className="overflow-hidden rounded-md border border-border bg-card">
      <div className="h-[3px]" style={{ backgroundColor: tokenCssVar(accent) }} />
      <div className="flex flex-col gap-1 px-2 py-1.5">
        <span className="text-2xs uppercase tracking-label-xl text-muted-foreground">{label}</span>
        <span className="text-sm font-semibold text-foreground">{value}</span>
      </div>
    </div>
  )
}
