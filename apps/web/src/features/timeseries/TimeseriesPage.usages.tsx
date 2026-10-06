/**
 * TimeseriesPage — onglet « Usages » : L'EMPRISE DU PÉRIMÈTRE SOLO (plan
 * PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05, maquette v4 `.ai/V7.5/MAQUETTE_TIMESERIES_USAGES_2026-10-05.html`).
 *
 * Mêmes matchs que le reste de la page (fenêtre filtrée, contexte solo), dans l'ordre d'un débrief :
 *
 *   1. « Portée » (la section porte son titre) puis « Rôles de portée » ;
 *   2. « Ressources » : « Prises par camp » | « Prises par camp, cumul par match » ;
 *   3. « Par carte » : « Prises par camp, par carte » ;
 *   4. « Prises » : « Part du joueur dans les prises du camp » ;
 *   5. « Rendement des ressources » : « Frags par ressource » | « Rendement par ressource » ;
 *   6. « Isolement » : « Vies à portée d'un coéquipier, vies isolées » ;
 *   7. « Objectif » : « Objectif par camp » puis « Part du joueur à l'objectif » ;
 *   8. « Équipement » : « Équipement : servi, gardé, lâché ».
 *
 * AUCUNE REQUÊTE NEUVE : tout arrive avec la réponse de page. Les cartes sont celles de l'Escouade
 * (textes du périmètre solo) et quatre cartes propres à l'onglet ; le joueur y est désigné par son
 * gamertag (`player`). Un bloc sans donnée se retire,
 * intertitre compris (prédicat unique `usagesSections`) ; sans rien à montrer, l'onglet le DIT.
 * Sans film (Halo 5), seuls les frags aux armes spéciales de la feuille de match restent.
 */
import type { ReactNode } from 'react'
import { useMemo } from 'react'

import { SectionTitle } from '@/components/ui/detail-section'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { ProductionCard } from '@/features/squad/emprise/ProductionCard'
import { ResourceControlCard } from '@/features/squad/emprise/ResourceControlCard'
import { ResourceFilCard } from '@/features/squad/emprise/ResourceFilCard'
import { useOutcomeLabels } from '@/features/squad/emprise/useOutcomeLabels'
import { YieldCard } from '@/features/squad/emprise/YieldCard'
import { ObjectiveBalanceCard } from '@/features/squad/objectif/ObjectiveBalanceCard'
import { ObjectiveSoloSheetCard } from '@/features/squad/objectif/ObjectiveSoloSheetCard'
import { useCapability } from '@/lib/capabilities/capabilities'
import type { TimeseriesPageResponse } from '@/lib/api/types'
import type { TimeseriesManifestKey } from '@/lib/i18n/generated/timeseries'
import type { Locale } from '@/lib/i18n/locale'
import { dominanceLabels } from '@/lib/narrative/dominance'

import { TimeseriesRangeRolesCard } from './TimeseriesRangeRolesCard'
import { EquipmentOutcomesCard } from './usages/EquipmentOutcomesCard'
import { LivesNearTeammateCard } from './usages/LivesNearTeammateCard'
import { MinePickupsCard } from './usages/MinePickupsCard'
import { ResourceMapGridCard } from './usages/ResourceMapGridCard'
import { useUsagesModels } from './usages/useUsagesModels'
import { EMPRISE_TEXT_SOLO, OBJECTIF_TEXT_SOLO, USAGES_TEXT } from './usages/usagesText'
import { WeaponRangeSection } from './WeaponRangeSection'

export interface TimeseriesUsagesTabProps {
  data: TimeseriesPageResponse
  locale: Locale
  t: (key: TimeseriesManifestKey) => string
}

export function TimeseriesUsagesTab({ data, locale, t }: TimeseriesUsagesTabProps) {
  // Portée mesurée des engagements : capability PRODUIT `weapon_range` (Halo 5 ne la déclare pas).
  const hasWeaponRange = useCapability('weapon_range')
  const u = useUsagesModels(data, locale, hasWeaponRange)
  const { models: m, show } = u
  const et = EMPRISE_TEXT_SOLO[locale]
  const ut = USAGES_TEXT[locale]
  const outcomes = useOutcomeLabels()
  const dominance = useMemo(() => dominanceLabels(locale), [locale])

  if (!Object.values(show).some(Boolean)) {
    return <EmptyStateNotice title={t('timeseries.usages.empty_title')} description={t('timeseries.usages.empty_description')} />
  }

  return (
    <div className="space-y-8" data-testid="timeseries-usages">
      {show.range && <WeaponRangeSection range={data.weapon_range} />}
      {/* Rôles de portée : la même famille de sujet, posée en écart au lobby (la carte se retire seule sans bloc). */}
      {hasWeaponRange && <TimeseriesRangeRolesCard bloc={data.range_profiles} />}
      {show.bilan && m.fil && (
        <Block id="bilan" title={ut.sections.bilan} sub={ut.sections.bilanCoverage(m.coverage.filmed, m.coverage.total)}>
          <div className="grid gap-4 lg:grid-cols-2">
            <ResourceControlCard rows={m.controlRows} t={et} />
            <ResourceFilCard fil={m.fil} dominanceLabels={dominance} outcomeLabels={outcomes} locale={locale} t={et} axe={u.filAxe} />
          </div>
        </Block>
      )}
      {show.carte && m.mapGrid && (
        <Block id="carte" title={ut.sections.carte}>
          <ResourceMapGridCard grid={m.mapGrid} itemName={(row) => (row.object ? u.objectName(row.object) : '')} playerName={u.playerName} t={et} ut={ut.cards} />
        </Block>
      )}
      {show.mine && m.mine && (
        <Block id="mine" title={ut.sections.mine}>
          <MinePickupsCard mine={m.mine} itemName={u.objectName} player={u.player} t={et} ut={ut.cards} />
        </Block>
      )}
      {show.prendre && (
        <Block id="prendre" title={ut.sections.prendre}>
          {/* Une carte seule (sans rendement : Halo 5) prend la rangée. */}
          <div className="grid gap-4 lg:grid-cols-2 lg:[&>*:only-child]:col-span-2">
            {m.production.length > 0 && <ProductionCard rows={m.production} t={et} />}
            {m.yieldRows.length > 0 && <YieldCard rows={m.yieldRows} coverage={m.vehicleCoverage} t={et} />}
          </div>
        </Block>
      )}
      {show.lives && m.lives && (
        <Block id="lives" title={ut.sections.lives}>
          <LivesNearTeammateCard model={m.lives} player={u.player} ut={ut.cards} />
        </Block>
      )}
      {show.objectif && (
        <Block id="objectif" title={ut.sections.objectif}>
          <div className="space-y-4">
            {u.balance.length > 0 && <ObjectiveBalanceCard families={u.balance} familyLabel={u.familyLabel} columns={u.columns} t={OBJECTIF_TEXT_SOLO[locale]} />}
            {u.soloSheet && (
              <ObjectiveSoloSheetCard sheet={u.soloSheet} name={u.sheetName} emblemUrl={u.emblemUrl} familyLabel={u.familyLabel} columns={u.columns} t={ut.sheet} />
            )}
          </div>
        </Block>
      )}
      {show.equipment && (
        <Block id="equipment" title={ut.sections.equipment}>
          <EquipmentOutcomesCard rows={m.equipment} familyLabel={u.equipmentLabel} player={u.player} ut={ut.cards} />
        </Block>
      )}
    </div>
  )
}

/** Un bloc de l'onglet : son intertitre (et sa couverture, à côté, en petit), puis ses cartes. */
function Block({ id, title, sub, children }: { id: string; title: string; sub?: string; children: ReactNode }) {
  return (
    <section className="space-y-2" data-testid={`usages-section-${id}`}>
      <SectionTitle>
        {title}
        {sub && <small className="ml-2 text-xs font-normal text-muted-foreground">{sub}</small>}
      </SectionTitle>
      {children}
    </section>
  )
}
