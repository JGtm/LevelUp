/**
 * HomeHeroKPIGrid — grille des 8 tuiles KPI du hero (Matchs, KDA, Win Rate,
 * Temps, Playlist favorite, Off/Def, Précision, Arme favorite).
 *
 * P8.4 finition (revue 2026-04-29) : extrait de HomePage.tsx (~145L).
 *
 * Harmonisation UI (2026-06-06) : chaque tuile porte la coquille « KPI card »
 * du catalogue via la primitive `KpiCard` — bg-card + barre d'accent 3px en haut
 * + label uppercase. L'accent suit la frontière type 2 / type 4 : métriques à
 * sentiment (KDA, taux de victoire, précision) → accent DYNAMIQUE (perf-tier /
 * outcome) ; métriques descriptives → accent fixe. Les valeurs sont neutres
 * (`text-foreground`) : la barre d'accent porte seule la couleur (parité Explorer).
 * Contenu composite (OutcomeBar, CombatYieldDisplay, sous-comptes) et grille
 * proportionnelle (`kpi-stats-grid`) inchangés.
 */
import type { HeroKPIs } from '@/lib/api/types'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { kdaDivergentScale, accuracyScale } from '@/lib/accessibility/scales'
import type { getKPIText } from './kpi.i18n'
import { KpiCard } from '@/components/cards/KpiCard'
import { OutcomeBar } from '@/components/ui/outcome-bar'
import { CombatYieldDisplay } from '@/components/ui/combat-yield-display'
import { Tooltip } from '@/components/ui/tooltip'

interface HomeHeroKPIGridProps {
  kpis: HeroKPIs
  labelOf: (key: string) => string
  numberLocale: string
  kpiText: ReturnType<typeof getKPIText>
}

/** Classe label harmonisée (type 2 du catalogue : uppercase, 2xs, tracking).
 *  `min-h-[1lh]` réserve UNE ligne pour le titre dans chaque carte → les valeurs
 *  s'alignent (au desktop, rangée unique à largeurs proportionnelles, chaque label
 *  tient sur 1 ligne depuis le raccourcissement « Matchs » / « Temps »). */
const KPI_LABEL_CLS = 'min-h-[1lh] text-2xs uppercase tracking-label-xl text-muted-foreground'
/** Conteneur interne d'une tuile : titre aligné en haut (cf. demande user :
 *  tous les titres alignés), contenu qui descend ; remplit la carte en hauteur. */
const KPI_CONTENT_CLS = 'flex flex-1 flex-col items-center justify-start py-3 text-center'
/** Valeur neutre : la couleur de sentiment est portée par la barre d'accent.
 *  Taille plafonnée à « m » (text-base) : dans la hero KPI bar, la plus grande
 *  taille de texte ne dépasse pas medium (cf. demande user) ; les tailles plus
 *  petites (libellés, sous-comptes) restent inchangées. */
const KPI_VALUE_CLS = 'text-base font-bold text-foreground'

function formatPlaytime(secs: number, kpiText: ReturnType<typeof getKPIText>): string {
  if (secs <= 0) return '—'
  const totalMin = Math.floor(secs / 60)
  const h = Math.floor(totalMin / 60)
  const totalDays = Math.floor(h / 24)
  if (totalDays >= 365) {
    const years = Math.floor(totalDays / 365)
    const remDays = totalDays % 365
    const months = Math.floor(remDays / 30)
    const days = remDays % 30
    const parts = [`${years}${kpiText.units.year}`]
    if (months > 0) parts.push(`${months}${kpiText.units.month}`)
    if (days > 0) parts.push(`${days}${kpiText.units.day}`)
    return parts.join(' ')
  }
  if (totalDays >= 30) {
    const months = Math.floor(totalDays / 30)
    const days = totalDays % 30
    const parts = [`${months}${kpiText.units.month}`]
    if (days > 0) parts.push(`${days}${kpiText.units.day}`)
    return parts.join(' ')
  }
  if (totalDays >= 1) {
    const remH = h % 24
    return remH > 0 ? `${totalDays}${kpiText.units.day} ${remH}${kpiText.units.hour}` : `${totalDays}${kpiText.units.day}`
  }
  const m = totalMin % 60
  return h === 0 ? `${m}${kpiText.units.minute}` : `${h}${kpiText.units.hour}${m > 0 ? String(m).padStart(2, '0') : ''}`
}

export function HomeHeroKPIGrid({
  kpis,
  labelOf,
  numberLocale,
  kpiText,
}: HomeHeroKPIGridProps) {
  const kda = kpis.avg_kda
  // Accents dynamiques (sentiment) : KDA/précision via les échelles perf-tier,
  // taux de victoire au seuil 50 %. Fallback neutre (perf-tier-3) si donnée absente.
  const kdaAccent: SemanticToken = kda != null ? kdaDivergentScale(kda) : 'perf-tier-3'
  const wins = kpis.wins
  const losses = kpis.losses
  const draws = kpis.draws ?? 0
  const dnfs = kpis.dnfs ?? 0
  const playtime = formatPlaytime(kpis.total_playtime_secs ?? 0, kpiText)
  const acc = kpis.avg_accuracy
  const accAccent: SemanticToken = acc != null ? accuracyScale(acc) : 'perf-tier-3'
  const winRate = kpis.win_rate
  // Couleur « pas trop grave » : le système vise 50 %, donc une bande neutre
  // [45,55] % reste en outcome-draw (ni bien ni mal). Au-delà seulement → vert/rouge.
  const winRateAccent: SemanticToken =
    winRate > 0.55 ? 'outcome-win' : winRate < 0.45 ? 'outcome-loss' : 'outcome-draw'

  // Détail des issues : victoires (gauche) / défaites (droite) aux extrémités de
  // la barre ; le reste (nuls, abandons) dans un tooltip au survol (cf. demande user).
  const outcomeRows: { token: SemanticToken; label: string; value: number }[] = [
    { token: 'outcome-win', label: kpiText.outcomes.wins, value: wins },
    { token: 'outcome-draw', label: kpiText.outcomes.draws, value: draws },
    { token: 'outcome-dnf', label: kpiText.outcomes.dnfs, value: dnfs },
    { token: 'outcome-loss', label: kpiText.outcomes.losses, value: losses },
  ]
  const winOutcomeTooltip = (
    <div className="space-y-1 text-left">
      {outcomeRows
        .filter((o) => o.value > 0)
        .map((o) => (
          <div key={o.label} className="flex items-center gap-2">
            <span className="h-2 w-2 shrink-0 rounded-full" style={{ backgroundColor: tokenCssVar(o.token) }} />
            <span className="flex-1 whitespace-nowrap">{o.label}</span>
            <span className="font-semibold tabular-nums">{o.value}</span>
          </div>
        ))}
    </div>
  )

  return (
    <div className="kpi-stats-grid items-stretch">
      {/* 1 — Matchs (accent fixe : métrique descriptive) */}
      <KpiCard accent="chart-series-1" className="flex h-full flex-col">
        <div className={`${KPI_CONTENT_CLS} px-2`}>
          <p className={KPI_LABEL_CLS}>{kpiText.labels.matches}</p>
          <p className={KPI_VALUE_CLS}>{kpis.total_matches.toLocaleString(numberLocale)}</p>
        </div>
      </KpiCard>

      {/* 2 — Temps (accent fixe ; déplacé en 2e position, cf. demande user) */}
      <KpiCard accent="chart-series-2" className="flex h-full flex-col">
        <div className={`${KPI_CONTENT_CLS} px-4`}>
          <p className={KPI_LABEL_CLS}>{kpiText.labels.totalTime}</p>
          <p className={KPI_VALUE_CLS}>{playtime}</p>
        </div>
      </KpiCard>

      {/* 3 — KDA/FDA (accent dynamique : échelle perf kdScale) */}
      <KpiCard accent={kdaAccent} className="flex h-full flex-col">
        <div className={`${KPI_CONTENT_CLS} px-2`}>
          <p className={KPI_LABEL_CLS}>{labelOf('kda')}</p>
          <p className={KPI_VALUE_CLS}>{kda != null ? kda.toFixed(2) : '—'}</p>
        </div>
      </KpiCard>

      {/* 4 — Taux de victoire (accent dynamique : bande neutre autour de 50 %) + barre composite */}
      <KpiCard accent={winRateAccent} className="flex h-full flex-col">
        <div className={`${KPI_CONTENT_CLS} px-4`}>
          <p className={KPI_LABEL_CLS}>{labelOf('win_rate')}</p>
          <p className={KPI_VALUE_CLS}>{`${(winRate * 100).toFixed(0)}%`}</p>
          {/* Victoires (gauche) ─ barre (détail nuls/abandons en tooltip) ─ défaites (droite).
              mt-0 : écart resserré SOUS le XX% uniquement ; le line-height par défaut de la
              valeur préserve l'écart titre→XX% au-dessus (cf. demande user). */}
          <div className="mt-0 flex w-full items-center gap-2">
            <span className="shrink-0 text-xs font-semibold tabular-nums" style={{ color: tokenCssVar('outcome-win') }}>
              {wins}
            </span>
            <Tooltip className="min-w-0 flex-1" content={winOutcomeTooltip}>
              <OutcomeBar wins={wins} draws={draws} losses={losses} dnfs={dnfs} />
            </Tooltip>
            <span className="shrink-0 text-xs font-semibold tabular-nums" style={{ color: tokenCssVar('outcome-loss') }}>
              {losses}
            </span>
          </div>
        </div>
      </KpiCard>

      {/* 5 — Playlist favorite (accent fixe) */}
      <KpiCard accent="chart-series-3" className="flex h-full flex-col">
        <div className={`${KPI_CONTENT_CLS} px-4`}>
          <p className={KPI_LABEL_CLS}>{kpiText.labels.favoritePlaylist}</p>
          <p className="w-full truncate text-sm font-bold text-foreground leading-tight mt-1">
            {kpis.favorite_playlist_name || '—'}
          </p>
          {kpis.favorite_playlist_count > 0 && (
            <p className="text-xs text-muted-foreground mt-0.5">
              {kpis.favorite_playlist_count.toLocaleString(numberLocale)} {kpiText.matches(kpis.favorite_playlist_count)}
            </p>
          )}
        </div>
      </KpiCard>

      {/* 7 — Rendement / Résistance (accent fixe : la barre composite porte son signal) */}
      <KpiCard accent="chart-series-4" className="flex h-full flex-col">
        <div className={`${KPI_CONTENT_CLS} px-4`}>
          <p className={`${KPI_LABEL_CLS} mb-1.5`}>{kpiText.labels.offDef}</p>
          <CombatYieldDisplay
            className="w-full"
            offensiveConversion={kpis.avg_offensive_conversion}
            defensiveResistance={kpis.avg_defensive_resistance}
            dmgPerKill={kpis.dmg_per_kill}
            dmgPerDeath={kpis.dmg_per_death}
            align="center"
          />
        </div>
      </KpiCard>

      {/* 8 — Précision (accent dynamique : échelle accuracyScale) */}
      <KpiCard accent={accAccent} className="flex h-full flex-col">
        <div className={`${KPI_CONTENT_CLS} px-2`}>
          <p className={KPI_LABEL_CLS}>{labelOf('accuracy')}</p>
          <p className={KPI_VALUE_CLS}>{acc != null ? `${acc.toFixed(0)}%` : '—'}</p>
        </div>
      </KpiCard>

      {/* 9 — Arme favorite (accent fixe) */}
      <KpiCard accent="chart-series-5" className="flex h-full flex-col" testId="home-favorite-weapon">
        <div className={`${KPI_CONTENT_CLS} px-4`}>
          <p className={KPI_LABEL_CLS}>{kpiText.labels.favoriteWeapon}</p>
          <p
            data-testid="home-favorite-weapon-name"
            className="w-full truncate text-sm font-bold text-foreground leading-tight mt-1"
          >
            {kpis.favorite_weapon_name || '—'}
          </p>
          {kpis.favorite_weapon_kills > 0 && (
            <p className="text-xs text-muted-foreground mt-0.5">
              {kpis.favorite_weapon_kills.toLocaleString(numberLocale)} {kpiText.kills(kpis.favorite_weapon_kills)}
            </p>
          )}
        </div>
      </KpiCard>
    </div>
  )
}
