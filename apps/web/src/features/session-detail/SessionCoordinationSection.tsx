/**
 * SessionCoordinationSection — LA SECTION « Coordination » de la colonne de session
 * (lot O, D22-6) : la carte « Appui reçu », seule dans sa rangée — demi-largeur en pleine page,
 * pleine colonne en vue compacte. « Riposte » a quitté la page (plan
 * PLAN_SESSIONS_EMPRISE_2026-10-06, V3) ; « Mes vies : près d'un coéquipier ou seul » (carte I)
 * reprend la question de l'entraide.
 *
 * LA DONNÉE ARRIVE DANS LA RÉPONSE EXISTANTE (`SessionPageResponse.coordination`, lot N1) —
 * aucune query de plus. Le drawer de comparaison monte CE MÊME composant avec
 * `compare_coordination` (lot S) : les deux colonnes rendent les mêmes cartes, chacune avec
 * SES données, et la rangée partagée D16 ne pose plus de placeholder dès que les deux
 * sessions ont un bloc. Aucune échelle partagée à poser : les jauges sont des rails 0..100 %
 * et les cases de bande des tons — les deux côtés se lisent déjà sur la même graduation.
 *
 * LE REPÈRE D'HABITUEL (lot S) est celui de la PÉRIODE DE RÉFÉRENCE, le même des deux côtés :
 * le serveur le sert dans chaque bloc (`appui.habituel_pct`), la carte ne le recalcule pas.
 *
 * FORMES PARTAGÉES : `UsageGaugeGrid` (deux jauges à parité, colonnes nommées par ce lot),
 * `UsageRegularityBand` (une case par match), `UsageBandLegend` — trois composants existants,
 * aucun graphe neuf.
 *
 * D22-VERBOSITÉ : PAS UNE PHRASE sous les formes. Les libellés de jauges sont tout ce qui est
 * écrit ; la méthode tient dans l'infobulle (i) du titre, trois phrases au plus
 * (`usageCardTitle`). Ne pas re-déverser de texte ici.
 *
 * `available = false` : la carte reste dans la rangée et NOMME SA CAUSE (D8,
 * `UsageEmptyNotice`) — un bloc escamoté laisse la rangée bancale et se lit comme un bug.
 *
 * Aucun calcul ici : tout vient de `coordinationModel.ts`.
 */
import { useMemo, type ReactNode } from 'react'

import { SectionCard } from '@/components/ui/section-card'
import { UsageBandLegend } from '@/features/_shared/usage/UsageBandLegend'
import { UsageEmptyNotice } from '@/features/_shared/usage/UsageEmptyNotice'
import { UsageGaugeGrid, type UsageGaugeColumn } from '@/features/_shared/usage/UsageForms'
import { UsageRegularityBand } from '@/features/_shared/usage/UsageRegularityBand'
import { usageCardTitle } from '@/features/_shared/usage/usageCardTitle'
import { USAGE_TEXT } from '@/features/_shared/usage/usageI18n'
import type { UsageBandCell } from '@/features/_shared/usage/usageRegularityBandModel'
import type { UsageGaugeRowModel } from '@/features/_shared/usage/usageGaugeModel'
import type { CoordinationBlock } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { COORDINATION_TEXT, type CoordinationText } from './coordinationI18n'
import { pairGridClass } from './_chartSections'
import { bandCaption, buildAppuiBand, buildAppuiGaugeRows } from './coordinationModel'

/**
 * La CAUSE d'une section vide, telle que `UsageEmptyNotice` la nomme. Le contrat rend une
 * `unavailable_reason` libre ; on la traduit dans les quatre causes connues plutôt que
 * d'écrire une chaîne serveur à l'écran (elle n'est ni localisée, ni destinée au lecteur).
 */
function emptyReason(block: CoordinationBlock | null | undefined) {
  if (block == null) return 'load-failed' as const
  return block.matches_measured > 0 ? 'no-objectives' : ('no-film' as const)
}

/** Le gabarit de la carte : titre + infobulle, jauges, bande. */
function CoordinationCard({
  title,
  info,
  rows,
  columns,
  bandLabel,
  cells,
  t,
  compact,
}: {
  title: string
  info: (label: string) => ReactNode
  rows: UsageGaugeRowModel[]
  columns: readonly UsageGaugeColumn[]
  bandLabel: string
  cells: UsageBandCell[]
  t: CoordinationText
  compact: boolean
}) {
  const usageT = USAGE_TEXT[useAppShellStore((s) => s.locale)]
  return (
    <SectionCard title={title} label={title} titleAdornment={info}>
      <div className="space-y-4 px-3 pb-3 pt-3">
        <UsageGaugeGrid rows={rows} dense={compact} columns={columns} />
        <div>
          <UsageRegularityBand
            label={bandLabel}
            cells={cells}
            caption={bandCaption(cells, t)}
            dense={compact}
          />
          <UsageBandLegend t={usageT} />
        </div>
      </div>
    </SectionCard>
  )
}

/** Une carte en état vide : elle garde son titre et dit POURQUOI elle est vide (D8). */
function CoordinationEmptyCard({
  title,
  block,
}: {
  title: string
  block: CoordinationBlock | null | undefined
}) {
  const usageT = USAGE_TEXT[useAppShellStore((s) => s.locale)]
  return (
    <SectionCard title={title} label={title}>
      <div className="px-3 pb-3 pt-3">
        <UsageEmptyNotice reason={emptyReason(block)} t={usageT} />
      </div>
    </SectionCard>
  )
}

export function SessionCoordinationSection({
  coordination,
  compact = false,
}: {
  /** Le bloc `coordination` de la réponse — absent (vieux serveur) : rien ne se rend. */
  coordination: CoordinationBlock | null | undefined
  /** Colonne divisée : mêmes formes, plus serrées ; la carte prend toute la colonne. */
  compact?: boolean
}) {
  const locale = useAppShellStore((s) => s.locale)
  const t = COORDINATION_TEXT[locale]
  const perMatch = useMemo(() => coordination?.per_match ?? [], [coordination])

  const appuiRows = useMemo(
    () => (coordination ? buildAppuiGaugeRows(coordination, t, locale) : []),
    [coordination, t, locale],
  )
  const appuiCells = useMemo(() => buildAppuiBand(perMatch, t, locale), [perMatch, t, locale])

  if (coordination == null) return null

  // Seule dans sa rangée : demi-largeur en pleine page, toute la colonne en vue compacte.
  const rowClass = pairGridClass(compact)

  if (!coordination.available) {
    return (
      <div className={rowClass} data-session-coordination="empty">
        <CoordinationEmptyCard title={t.cardAppui} block={coordination} />
      </div>
    )
  }

  const coverage = t.coverageMatchesFmt(coordination.matches_measured, coordination.matches_total)

  return (
    <div className={rowClass} data-session-coordination="">
      <CoordinationCard
        title={t.cardAppui}
        info={usageCardTitle(t.infoAppui1, t.infoAppui2, t.infoAppui3, coverage)}
        rows={appuiRows}
        columns={[
          { header: t.gaugePrepared },
          { header: t.gaugeAssistShare },
        ]}
        bandLabel={t.bandAppui}
        cells={appuiCells}
        t={t}
        compact={compact}
      />
    </div>
  )
}
