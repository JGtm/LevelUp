/**
 * SessionCoordinationSection — LA SECTION « Coordination » de la colonne de session
 * (lot O, D22-1 et D22-6) : deux cartes en rangée, « Riposte » et « Appui reçu ».
 *
 * LA DONNÉE ARRIVE DANS LA RÉPONSE EXISTANTE (`SessionPageResponse.coordination`, lot N1) —
 * aucune query de plus. Le drawer de comparaison n'a PAS de miroir (le contrat ne sert pas
 * de `compare_coordination`) : la section n'existe alors que du côté principal et la
 * rangée partagée D16 pose son placeholder en face.
 *
 * MÊMES FORMES QUE LE BLOC « USAGES » : `UsageGaugeGrid` (deux jauges à parité, colonnes
 * nommées par ce lot), `UsageRegularityBand` (une case par match), `UsageBandLegend` — trois
 * composants existants, aucun graphe neuf.
 *
 * D22-VERBOSITÉ : PAS UNE PHRASE sous les formes. Le chiffre d'appel (le délai médian de
 * riposte) et les libellés de jauges sont tout ce qui est écrit ; la méthode tient dans
 * l'infobulle (i) du titre, trois phrases au plus (`usageCardTitle`). Ne pas re-déverser
 * de texte ici.
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
import {
  bandCaption,
  buildAppuiBand,
  buildAppuiGaugeRows,
  buildRiposteBand,
  buildRiposteGaugeRows,
  fenetreSeconds,
  formatDelaiMedian,
} from './coordinationModel'

/**
 * La CAUSE d'une section vide, telle que `UsageEmptyNotice` la nomme. Le contrat rend une
 * `unavailable_reason` libre ; on la traduit dans les quatre causes connues plutôt que
 * d'écrire une chaîne serveur à l'écran (elle n'est ni localisée, ni destinée au lecteur).
 */
function emptyReason(block: CoordinationBlock | null | undefined) {
  if (block == null) return 'load-failed' as const
  return block.matches_measured > 0 ? 'no-objectives' : ('no-film' as const)
}

/** Le gabarit commun des deux cartes : titre + infobulle, chiffre d'appel, jauges, bande. */
function CoordinationCard({
  title,
  info,
  callout,
  rows,
  columns,
  bandLabel,
  cells,
  t,
  compact,
}: {
  title: string
  info: ReactNode
  /** Le CHIFFRE D'APPEL, seul texte autorisé sous le titre (D22) — absent = rien. */
  callout: string | null
  rows: UsageGaugeRowModel[]
  columns: readonly UsageGaugeColumn[]
  bandLabel: string
  cells: UsageBandCell[]
  t: CoordinationText
  compact: boolean
}) {
  const usageT = USAGE_TEXT[useAppShellStore((s) => s.locale)]
  return (
    <SectionCard title={title} label={title} titleAdornment={info as (label: string) => ReactNode}>
      <div className="space-y-4 px-3 pb-3 pt-3">
        {callout != null && (
          <p className="text-sm tabular-nums text-foreground" data-coordination-callout="">
            {callout}
          </p>
        )}
        <UsageGaugeGrid rows={rows} t={usageT} dense={compact} columns={columns} />
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
  /** Colonne divisée : mêmes formes, plus serrées ; les cartes s'empilent. */
  compact?: boolean
}) {
  const locale = useAppShellStore((s) => s.locale)
  const t = COORDINATION_TEXT[locale]
  const perMatch = useMemo(() => coordination?.per_match ?? [], [coordination])

  const riposteRows = useMemo(
    () => (coordination ? buildRiposteGaugeRows(coordination, t, locale) : []),
    [coordination, t, locale],
  )
  const appuiRows = useMemo(
    () => (coordination ? buildAppuiGaugeRows(coordination, t, locale) : []),
    [coordination, t, locale],
  )
  const riposteCells = useMemo(() => buildRiposteBand(perMatch, t, locale), [perMatch, t, locale])
  const appuiCells = useMemo(() => buildAppuiBand(perMatch, t, locale), [perMatch, t, locale])

  if (coordination == null) return null

  const rowClass = compact ? 'space-y-3' : 'grid grid-cols-1 gap-3 lg:grid-cols-2'

  if (!coordination.available) {
    return (
      <div className={rowClass} data-session-coordination="empty">
        <CoordinationEmptyCard title={t.cardRiposte} block={coordination} />
        <CoordinationEmptyCard title={t.cardAppui} block={coordination} />
      </div>
    )
  }

  const coverage = t.coverageMatchesFmt(coordination.matches_measured, coordination.matches_total)
  const delai = formatDelaiMedian(coordination, t, locale)

  return (
    <div className={rowClass} data-session-coordination="">
      <CoordinationCard
        title={t.cardRiposte}
        info={usageCardTitle(
          t.infoRiposte1(fenetreSeconds(coordination, locale)),
          t.infoRiposte2,
          t.infoRiposte3,
          coverage,
        )}
        callout={delai != null ? `${t.delaiMedian} : ${delai}` : null}
        rows={riposteRows}
        columns={[
          { header: t.gaugeCovered, denominator: 'team' },
          { header: t.gaugeIRiposte, denominator: 'team' },
        ]}
        bandLabel={t.bandRiposte}
        cells={riposteCells}
        t={t}
        compact={compact}
      />
      <CoordinationCard
        title={t.cardAppui}
        info={usageCardTitle(t.infoAppui1, t.infoAppui2, t.infoAppui3, coverage)}
        callout={null}
        rows={appuiRows}
        columns={[
          { header: t.gaugePrepared, denominator: 'team' },
          { header: t.gaugeAssistShare, denominator: 'team' },
        ]}
        bandLabel={t.bandAppui}
        cells={appuiCells}
        t={t}
        compact={compact}
      />
    </div>
  )
}
