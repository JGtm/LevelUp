/**
 * ExplorerTargetVersusDonuts — dernière rangée de la section « matchs joués
 * ensemble » de l'encart adversaire.
 *
 * À GAUCHE : deux donuts de taux de victoire empilés — « ensemble » (en allié)
 * et « face à lui » (en ennemi) — chacun avec le REPÈRE de la moyenne perso
 * historique du joueur principal (anneau secondaire + delta signé).
 * À DROITE : la courbe d'« écart de frags cumulé » duel par duel contre la cible.
 *
 * Les deux briques visuelles sont RÉUTILISÉES telles quelles du hub Communauté >
 * Relations (RelationWinRateDonut + CumulativeFragGapChart), alimentées par les
 * mêmes données côté backend (WR historique + timeline de duels).
 *
 * Dégradation : un rôle sans match (ex. cible jamais alliée) affiche « — » sous
 * son libellé ; la colonne graphe est masquée quand aucun duel n'existe (jamais
 * affrontés en ennemi) → les donuts passent alors côte à côte, pleine largeur.
 */
import { RelationWinRateDonut } from '@/features/palmares/RelationWinRateDonut'
import { CumulativeFragGapChart } from '@/features/palmares/CumulativeFragGapChart'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { ExplorerEncounterStats } from '@/lib/api/types'

interface Props {
  encounterStats: ExplorerEncounterStats
}

interface DonutLabels {
  wins: string
  losses: string
  personalAvg: string
  pointsUnit: string
  liftTooltip: string
}

export function ExplorerTargetVersusDonuts({ encounterStats }: Props) {
  const appLocale = useAppShellStore((s) => s.locale)
  const t = (key: ExplorerManifestKey) => formatMessage(explorerManifest, key, appLocale)

  const personalAvg = encounterStats.player_win_rate ?? null
  const wrTogether = encounterStats.winrate_as_ally ?? null
  const wrVersus = encounterStats.winrate_vs_enemy ?? null

  const donutLabels: DonutLabels = {
    wins: t('explorer.target_profile.donut_wins'),
    losses: t('explorer.target_profile.donut_losses'),
    personalAvg: t('explorer.target_profile.donut_personal_avg'),
    pointsUnit: t('explorer.target_profile.donut_points_unit'),
    liftTooltip: t('explorer.target_profile.donut_lift_tooltip'),
  }

  // frag_gap_series (backend) a EXACTEMENT la forme de CumulativeFragPoint
  // ({cumulative, outcome}) → passage direct, sans remapping.
  const fragPoints = encounterStats.frag_gap_series ?? []
  const hasFragGap = fragPoints.length > 0

  // Garde-fou : rien à afficher si aucun taux dans les deux rôles ET aucun duel
  // (la section n'est de toute façon rendue que si sample_size > 0).
  if (wrTogether == null && wrVersus == null && !hasFragGap) return null

  return (
    <div className="grid gap-4 lg:grid-cols-3" data-testid="explorer-target-versus">
      {/* Colonne donuts : empilés (1/3) quand le graphe est présent, sinon côte
          à côte pleine largeur. */}
      <div className={hasFragGap ? 'flex flex-col gap-4' : 'grid gap-4 sm:grid-cols-2 lg:col-span-3'}>
        <DonutCard
          caption={t('explorer.target_profile.wr_together_caption')}
          winRate={wrTogether}
          personalAvg={personalAvg}
          labels={donutLabels}
        />
        <DonutCard
          caption={t('explorer.target_profile.wr_versus_caption')}
          winRate={wrVersus}
          personalAvg={personalAvg}
          labels={donutLabels}
        />
      </div>

      {hasFragGap && (
        <div className="overflow-hidden rounded-lg border border-border bg-card lg:col-span-2">
          <div className="border-b border-border px-3 py-2 text-sm font-medium">
            {t('explorer.target_profile.frag_gap_title')}
          </div>
          <div className="p-3">
            <CumulativeFragGapChart points={fragPoints} height={180} />
          </div>
        </div>
      )}
    </div>
  )
}

function DonutCard({
  caption,
  winRate,
  personalAvg,
  labels,
}: {
  caption: string
  winRate: number | null
  personalAvg: number | null
  labels: DonutLabels
}) {
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      {winRate != null ? (
        <RelationWinRateDonut winRate={winRate} personalAvg={personalAvg} labels={labels} caption={caption} />
      ) : (
        <div className="flex flex-col gap-1">
          <p className="text-2xs uppercase tracking-label-md text-muted-foreground">{caption}</p>
          <span className="font-mono text-2xl font-bold text-muted-foreground">—</span>
        </div>
      )}
    </div>
  )
}
