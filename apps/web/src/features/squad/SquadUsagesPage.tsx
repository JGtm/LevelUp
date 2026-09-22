/**
 * SquadUsagesPage — onglet Usages de l'Escouade.
 *
 * UN SEUL AXE DE LECTURE : ce que la composition UTILISE — ses frags et ses armes,
 * son équipement, les formes qu'elle retient. Les trois blocs vivaient sur Synergies
 * (lot 3 « sections », 2026-09-22) ; l'onglet y mélangeait ce que l'escouade PRODUIT
 * ensemble (l'échange, les cartes, l'historique) et ce qu'elle CONSOMME.
 *
 * AUCUNE REQUÊTE NEUVE : la page lit `SquadContext`, publié par `SquadLayout` depuis
 * l'unique `useTeammates`. Mêmes gardes que Synergies (no_selection /
 * invalid_selection), plus un état vide propre quand la sélection n'a aucun usage à
 * montrer (aucun film décodé) — jamais un onglet vide et muet.
 */
import { useMemo } from 'react'

import { Card, CardContent } from '@/components/ui/card'
import { SectionTitle } from '@/components/ui/detail-section'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { useAppShellStore } from '@/stores/appShellStore'
import { useCapability } from '@/lib/capabilities/capabilities'
import { EquipmentUsageSection } from '@/features/_shared/usage/EquipmentUsageSection'
import { usageAvailability } from '@/features/_shared/usage/usageAvailability'
import { USAGE_TEXT } from '@/features/_shared/usage/usageI18n'
import { useSquadContext } from './SquadContext'
import { getSquadText } from './i18n'
import { FormesRetenuesSection } from './formes/FormesRetenuesSection'
import { SquadFragSection } from './SquadFragSection'
import { SquadFdaGapCumulativeCard } from './SquadFdaGapCumulativeCard'
import { getSquadPlayerColors } from './colors'

export function SquadUsagesPage() {
  const { selectedRows, confirmedGamertags, pageData, playerSlug } = useSquadContext()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const usageText = USAGE_TEXT[locale]
  // FDA attendu natif (Infinite déclare `expected_stats`, Halo 5 non) → gate PARENT du
  // card « Écart cumulé au FDA attendu » : pas de colonne vide dans la rangée 1 de
  // SquadFragSection (le card conserve son self-gate en profondeur).
  const hasExpectedStats = useCapability('expected_stats')

  // Le backend renvoie s.gamertag (casse mixte) tandis que playerSlug est l'URL param
  // (souvent lowercase) : on aligne sur main_player.
  const mainPlayerKey = pageData?.main_player ?? playerSlug
  const fragClasses = pageData?.frag_classes
  const performanceSeries = pageData?.performance_series
  // Replis MÉMOÏSÉS : `?? {}` écrit dans le JSX fabriquerait un objet neuf à chaque
  // rendu → les ChartCard rebâtiraient leur option ECharts et rejoueraient leur
  // animation d'entrée sans qu'aucune valeur n'ait bougé.
  const fragClassesByPlayer = useMemo(() => fragClasses ?? {}, [fragClasses])
  const perfSeriesByPlayer = useMemo(() => performanceSeries ?? {}, [performanceSeries])
  const playerColors = useMemo(
    () => getSquadPlayerColors(mainPlayerKey, confirmedGamertags),
    [mainPlayerKey, confirmedGamertags],
  )
  const playerOrder = useMemo(
    () =>
      [mainPlayerKey, ...confirmedGamertags].filter(
        (p) => fragClasses?.[p] || performanceSeries?.[p],
      ),
    [mainPlayerKey, confirmedGamertags, fragClasses, performanceSeries],
  )

  const equipmentUsage = pageData?.equipment_usage
  const formes = pageData?.formes_retenues
  // CE QUE L'ONGLET A RÉELLEMENT À MONTRER, bloc par bloc — mêmes prédicats que les
  // composants eux-mêmes (usageAvailability pour l'équipement, `available` pour les
  // formes). Les trois à faux : l'onglet le DIT, au lieu de laisser une page nue.
  const hasFrags =
    playerOrder.length > 0 || pageData?.weapon_kills != null || pageData?.weapon_accuracy != null
  const hasEquipment = usageAvailability(equipmentUsage, usageText).kind !== 'hidden'
  const hasFormes = formes != null && formes.available
  const hasAnyBlock = hasFrags || hasEquipment || hasFormes

  const hasSelection = confirmedGamertags.length > 0
  const hasRows = selectedRows.length > 0

  if (!hasSelection) {
    return (
      <Card>
        <CardContent className="pt-4">
          <EmptyStateNotice
            title={t.empty.noSelectionTitle}
            description={t.empty.noSelectionDescription}
          />
        </CardContent>
      </Card>
    )
  }

  if (!hasRows) {
    return (
      <Card>
        <CardContent className="pt-4">
          <EmptyStateNotice
            title={t.empty.invalidSelectionTitle}
            description={t.empty.invalidSelectionDescription}
          />
        </CardContent>
      </Card>
    )
  }

  if (!hasAnyBlock) {
    return (
      <Card>
        <CardContent className="pt-4">
          <EmptyStateNotice
            title={t.empty.noDecodedFilmTitle}
            description={t.empty.noDecodedFilmDescription}
          />
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-4">
      {/* FRAGS ET ARMES. Rangée 1 : sur Infinite « Écart cumulé au FDA attendu »
          (gate `expected_stats`) à GAUCHE de « Répartition des frags » ; sur Halo 5
          « Répartition » | « Précision par rôle ». Puis « Outils de destruction ».
          Le titre de section coiffe les trois cartes — SquadFragSection n'en porte
          aucun de son côté. */}
      {hasFrags && (
        <section className="space-y-3">
          <SectionTitle>{t.sections.fragsArmes}</SectionTitle>
          <SquadFragSection
            fragClassesByPlayer={fragClassesByPlayer}
            weaponKills={pageData?.weapon_kills}
            weaponAccuracy={pageData?.weapon_accuracy}
            playerColors={playerColors}
            playerOrder={playerOrder}
            locale={locale}
            t={t}
            leftOfBreakdown={
              hasExpectedStats ? (
                <SquadFdaGapCumulativeCard
                  rowsByPlayer={perfSeriesByPlayer}
                  playerOrder={playerOrder}
                  colorByPlayer={playerColors}
                  t={t}
                  emptyMessage={t.empty.noBlockData}
                />
              ) : undefined
            }
          />
        </section>
      )}
      {/* ÉQUIPEMENT — bloc « servi ou gâché », variante comptes (une ligne par
          coéquipier suivi). Aucune requête neuve : lit `pageData.equipment_usage` de
          la réponse déjà chargée par `useTeammates`. LE TITRE DIT L'AXE, PAS LA
          PREMIÈRE CARTE : il coiffe les quatre cartes de la section (usages, part du
          lobby, armes spéciales, niveaux), dont l'une s'appelle déjà « Usages
          d'équipement » — reprendre ce libellé ici l'aurait écrit deux fois de suite
          (arbitrage du 2026-09-22). La section se retire d'elle-même sans bloc. */}
      {hasEquipment && (
        <section className="space-y-3">
          <SectionTitle>{t.sections.equipement}</SectionTitle>
          <EquipmentUsageSection usage={equipmentUsage} mode="squad" t={usageText} locale={locale} />
        </section>
      )}
      {/* « Les formes retenues », CONTEXTE ESCOUADE SEUL : elle porte déjà son propre
          titre de section — on ne le double pas. */}
      <FormesRetenuesSection
        block={formes}
        locale={locale}
        contexte="squad"
        mainPlayerLabel={pageData?.main_player ?? playerSlug}
      />
    </div>
  )
}
