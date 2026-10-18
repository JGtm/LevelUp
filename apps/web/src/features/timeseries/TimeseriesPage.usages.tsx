/**
 * TimeseriesPage — onglet « Usages » : L'EMPRISE DU PÉRIMÈTRE SOLO (plan
 * PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05, maquette v4 `.ai/V7.5/MAQUETTE_TIMESERIES_USAGES_2026-10-05.html`).
 *
 * Mêmes matchs que le reste de la page (fenêtre filtrée, contexte solo), dans l'ordre d'un débrief :
 *
 *   1. « Portée » (la section porte son titre) : « Portée par arme » | « Rôles de portée » ;
 *   2. « Ressources » : « Contrôle des ressources » | « Contrôle des ressources, cumul par match » ;
 *   3. « Par carte » : « Contrôle des ressources, par carte » ;
 *   4. « Prises » : « Contribution aux prises » ;
 *   5. « Rendement des ressources » : « Frags par ressource » | « Rendement par ressource » ;
 *   6. « Isolement » : « Isolement » ;
 *   7. « Objectif » : « Rapport de force » puis « Part du joueur à l'objectif » ;
 *   8. « Équipement » : « Usage d'équipements ».
 *
 * AUCUNE REQUÊTE NEUVE : tout arrive avec la réponse de page. Les cartes sont celles de l'Escouade
 * (textes du périmètre solo) et quatre cartes propres à l'onglet ; le joueur y est désigné par son
 * gamertag (`player`). Un bloc sans donnée se retire,
 * intertitre compris (prédicat unique `usagesSections`), SAUF quand le titre lit le film et que le film des
 * matchs n'est pas lu pour l'un d'eux au moins (`filmGapOfBlock`) : il garde alors son intertitre et un état
 * dans un bloc parent (« Analyse du film en attente » ou « Film non analysé »), jamais un trou. Sans rien à
 * montrer, l'onglet le DIT (bloc parent + pointillé dedans). Sans film (Halo 5), seuls les frags aux armes
 * spéciales de la feuille de match restent. L'objectif, tiré de la feuille de match, garde de même son
 * intertitre quand aucun match du périmètre n'est à objectif (« Aucun match à objectif sur ce périmètre »,
 * `ScopeEmptySection`) ; il ne se retire que si le titre ne mesure pas l'objectif.
 */
import type { ReactNode } from 'react'
import { useMemo } from 'react'

import { Card, CardContent } from '@/components/ui/card'
import { SectionTitle } from '@/components/ui/detail-section'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { ScopeEmptySection } from '@/components/ui/scope-empty'
import { FilmGapSection } from '@/features/_shared/filmGap/FilmGapSection'
import { filmGapOfBlock } from '@/features/_shared/filmGap/filmGap.logic'
import { FILM_GAP_TEXT } from '@/features/_shared/filmGap/filmGapStrings'
import { PAIR_ROW_CLASS } from '@/components/ui/pair-row'
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
import { formatMessage } from '@/lib/i18n/format'
import { synthesisManifest } from '@/lib/i18n/generated/synthesis'
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
  // La carte se retire seule sans bloc de profils ; sans capability, elle n'existe pas.
  const rangeRoles = hasWeaponRange ? <TimeseriesRangeRolesCard bloc={data.range_profiles} /> : null

  // Le film des matchs n'est pas lu (analyse en cours, ou jamais faite) : un bloc tiré du film sans donnée
  // reste affiché avec son état. Objectif (feuille de match) n'en fait pas partie.
  const gap = filmGapOfBlock(data.emprise)
  const gapText = FILM_GAP_TEXT[locale]
  const rangeGap = gap && hasWeaponRange && !show.range && data.range_profiles == null

  if (!Object.values(show).some(Boolean) && !gap) {
    return (
      <Card data-testid="timeseries-usages-empty">
        <CardContent className="pt-4">
          <EmptyStateNotice title={t('timeseries.usages.empty_title')} description={t('timeseries.usages.empty_description')} />
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-8" data-testid="timeseries-usages">
      {/* Rôles de portée : la même famille de sujet, posée en écart au lobby, À DROITE de
          « Portée par arme » sur la même rangée. Sans bloc de portée par arme, elle reste seule. */}
      {show.range ? (
        <WeaponRangeSection range={data.weapon_range} aside={rangeRoles} />
      ) : (
        <>
          {rangeGap && gap && (
            <FilmGapSection
              title={formatMessage(synthesisManifest, 'synthesis.weapon_range.heading', locale)}
              testId="usages-section-range"
              gap={gap}
              t={gapText}
            />
          )}
          {rangeRoles}
        </>
      )}
      {show.bilan && m.fil ? (
        <Block id="bilan" title={ut.sections.bilan}>
          <div className="grid gap-4 lg:grid-cols-2">
            <ResourceControlCard rows={m.controlRows} t={et} />
            <ResourceFilCard fil={m.fil} dominanceLabels={dominance} outcomeLabels={outcomes} locale={locale} t={et} axe={u.filAxe} />
          </div>
        </Block>
      ) : (
        gap && <FilmGapSection title={ut.sections.bilan} testId="usages-section-bilan" gap={gap} t={gapText} />
      )}
      {show.carte && m.mapGrid ? (
        <Block id="carte" title={ut.sections.carte}>
          <ResourceMapGridCard grid={m.mapGrid} itemName={(row) => (row.object ? u.objectName(row.object) : '')} playerName={u.playerName} t={et} ut={ut.cards} />
        </Block>
      ) : (
        gap && <FilmGapSection title={ut.sections.carte} testId="usages-section-carte" gap={gap} t={gapText} />
      )}
      {show.mine && m.mine ? (
        <Block id="mine" title={ut.sections.mine}>
          <MinePickupsCard mine={m.mine} itemName={u.objectName} player={u.player} t={et} ut={ut.cards} />
        </Block>
      ) : (
        gap && <FilmGapSection title={ut.sections.mine} testId="usages-section-mine" gap={gap} t={gapText} />
      )}
      {show.prendre ? (
        <Block id="prendre" title={ut.sections.prendre}>
          {/* Une carte seule (sans rendement : Halo 5) prend la rangée. */}
          <div className={PAIR_ROW_CLASS}>
            {m.production.length > 0 && <ProductionCard rows={m.production} t={et} />}
            {m.yieldRows.length > 0 && <YieldCard rows={m.yieldRows} t={et} />}
          </div>
        </Block>
      ) : (
        gap && <FilmGapSection title={ut.sections.prendre} testId="usages-section-prendre" gap={gap} t={gapText} />
      )}
      {show.lives && m.lives ? (
        <Block id="lives" title={ut.sections.lives}>
          <LivesNearTeammateCard model={m.lives} player={u.player} ut={ut.cards} />
        </Block>
      ) : (
        gap && <FilmGapSection title={ut.sections.lives} testId="usages-section-lives" gap={gap} t={gapText} />
      )}
      {!show.objectif && m.objectiveEmpty && (
        <ScopeEmptySection title={ut.sections.objectif} kind="objective" locale={locale} testId="usages-section-objectif" />
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
      {show.equipment ? (
        <Block id="equipment" title={ut.sections.equipment}>
          <EquipmentOutcomesCard rows={m.equipment} familyLabel={u.equipmentLabel} player={u.player} ut={ut.cards} />
        </Block>
      ) : (
        gap && <FilmGapSection title={ut.sections.equipment} testId="usages-section-equipment" gap={gap} t={gapText} />
      )}
    </div>
  )
}

/** Un bloc de l'onglet : son intertitre, puis ses cartes. Aucune mention de couverture. */
function Block({ id, title, children }: { id: string; title: string; children: ReactNode }) {
  return (
    <section className="space-y-2" data-testid={`usages-section-${id}`}>
      <SectionTitle>{title}</SectionTitle>
      {children}
    </section>
  )
}
