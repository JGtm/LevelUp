/**
 * SessionFragCard — « Répartition des frags » v2 d'UNE session : sunburst hiérarchique
 * classe→rôle (FragSunburst) + un 2e graphe qui DÉPEND du titre :
 *   - titre fournissant la précision par arme (Halo 5, table weapon_accuracy) →
 *     « Précision par arme » (SynthesisWeaponAccuracyChart, survol lié au sunburst) ;
 *   - sinon (Infinite) → « Détails des frags » (FragWeaponBreakdown = armes du registre
 *     + détail mêlée/grenade/capacités via buildFragDetailBreakdown).
 * MÊME rendu final que le Match view : compteur SEUL centré, légende à gauche, survol
 * LIÉ sunburst ↔ 2e graphe.
 *
 * Responsive : CÔTE À CÔTE (sunburst 2/3 | 2e graphe 1/3) en vue large ; EMPILÉ (2e graphe
 * SOUS le sunburst, légende repassée en bas) quand la colonne est étroite (`stacked` — drawer
 * de comparaison ouvert / vue compacte). Alimentée par l'agrégat de session (entry.frag_
 * distribution + entry.top_weapon_kills + entry.weapon_accuracy). Rend null si aucune donnée.
 *
 * Hauteur (I5, V7.1) : sunburst et 2e graphe partagent la MÊME hauteur fixe FRAG_CARD_HEIGHT,
 * dans les DEUX états (côte à côte ET empilé) — sans ça, le sunburst se met à l'échelle de sa
 * largeur de colonne (variable selon 2/3 vs pleine largeur) alors que le 2e graphe calait sa
 * hauteur sur son nombre d'armes : les deux cartes divergeaient visuellement, surtout quand le
 * drawer de comparaison est déployé (colonnes rétrécies). Valeur alignée sur le défaut de
 * ChartCard (320) pour rester cohérente avec le reste des surfaces frags.
 */
import { useState } from 'react'

import { FragSunburst } from '@/components/charts/FragSunburst'
import { FragWeaponBreakdown } from '@/components/charts/FragWeaponBreakdown'
import { buildFragDetailBreakdown } from '@/components/charts/fragDetailBreakdown'
// Précision par arme : réutilise le graphe Synthesis (déjà recoloré par classe + survol
// lié). Import cross-feature durable déclaré (session-detail=>synthesis, cf.
// tools/lint-cross-feature-imports.mjs) — analogue à session-detail=>explorer.
import { SynthesisWeaponAccuracyChart } from '@/features/synthesis/SynthesisWeaponAccuracyChart'
import { formatMessage } from '@/lib/i18n/format'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import type { SessionCompareEntry } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

/** Hauteur fixe PARTAGÉE par le sunburst et le 2e graphe (I5 — même hauteur des deux cartes
 *  dans les deux états compare/non-compare). Alignée sur le défaut ChartCard. */
const FRAG_CARD_HEIGHT = 320

interface Props {
  entry: SessionCompareEntry | null
  /** Empile le 2e graphe SOUS le sunburst (colonne étroite : drawer comparaison / vue compacte). */
  stacked?: boolean
}

export function SessionFragCard({ entry, stacked = false }: Props) {
  const appLocale = useAppShellStore((s) => s.locale)
  const [hoveredClass, setHoveredClass] = useState<string | null>(null)

  const distribution = entry?.frag_distribution ?? null
  const classLabel = (c: string) => formatMessage(fragsManifest, `frags.class.${c}` as never, appLocale)
  const roleLabel = (r: string) => formatMessage(fragsManifest, `frags.role.${r}` as never, appLocale)
  const detailTitle = formatMessage(fragsManifest, 'frags.charts.detail_title', appLocale)
  const breakdown = buildFragDetailBreakdown(distribution, entry?.top_weapon_kills ?? [], { roleLabel, classLabel })
  // Précision par arme native (Halo 5) : quand l'agrégat de session la porte, le 2e graphe
  // devient « Précision par arme » à la place de « Détails des frags » (Infinite = vide → repli).
  const accuracy = entry?.weapon_accuracy ?? []

  const hasSunburst = (distribution?.total_kills ?? 0) > 0
  if (!hasSunburst && breakdown.length === 0) return null

  return (
    <div className={stacked ? 'grid grid-cols-1 gap-4' : 'grid grid-cols-1 gap-4 xl:grid-cols-3'}>
      <FragSunburst
        distribution={distribution}
        externalHoveredClass={hoveredClass}
        onClassHover={setHoveredClass}
        className={stacked ? '' : 'xl:col-span-2'}
        hideCenterLabel
        maxWidthPx={480}
        legendSide={stacked ? 'bottom' : 'left'}
        heightPx={FRAG_CARD_HEIGHT}
      />
      {accuracy.length > 0 ? (
        <div className={stacked ? 'flex min-w-0 flex-col' : 'flex min-w-0 flex-col xl:col-span-1'}>
          <SynthesisWeaponAccuracyChart
            weapons={accuracy}
            weaponKills={entry?.top_weapon_kills ?? []}
            hoveredClass={hoveredClass}
            onClassHover={setHoveredClass}
            height={FRAG_CARD_HEIGHT}
          />
        </div>
      ) : (
        <FragWeaponBreakdown
          weapons={breakdown}
          title={detailTitle}
          hoveredClass={hoveredClass}
          onClassHover={setHoveredClass}
          height={FRAG_CARD_HEIGHT}
          className={stacked ? '' : 'xl:col-span-1'}
        />
      )}
    </div>
  )
}
