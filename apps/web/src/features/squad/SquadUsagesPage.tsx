/**
 * SquadUsagesPage — onglet Usages de l'Escouade.
 *
 * UN SEUL AXE DE LECTURE : ce que la composition UTILISE — son équipement, les formes
 * qu'elle retient. Les blocs vivaient sur Synergies (lot 3 « sections », 2026-09-22) ;
 * l'onglet y mélangeait ce que l'escouade PRODUIT ensemble (l'échange, les cartes,
 * l'historique) et ce qu'elle CONSOMME. Les frags et armes (Répartition des frags, Outils
 * de destruction) sont partis sur Contributions au lot L2 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26.
 *
 * AUCUNE REQUÊTE NEUVE : la page lit `SquadContext`, publié par `SquadLayout` depuis
 * l'unique `useTeammates`. Mêmes gardes que Synergies (no_selection /
 * invalid_selection), plus un état vide propre quand la sélection n'a aucun usage à
 * montrer (aucun film décodé) — jamais un onglet vide et muet.
 */
import { Card, CardContent } from '@/components/ui/card'
import { SectionTitle } from '@/components/ui/detail-section'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { useAppShellStore } from '@/stores/appShellStore'
import { EquipmentUsageSection } from '@/features/_shared/usage/EquipmentUsageSection'
import { usageAvailabilityKind } from '@/features/_shared/usage/usageAvailability'
import { USAGE_TEXT } from '@/features/_shared/usage/usageI18n'
import { useSquadContext } from './SquadContext'
import { getSquadText } from './i18n'
import { FormesRetenuesSection } from './formes/FormesRetenuesSection'

export function SquadUsagesPage() {
  const { selectedRows, confirmedGamertags, pageData, playerSlug } = useSquadContext()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const usageText = USAGE_TEXT[locale]

  const equipmentUsage = pageData?.equipment_usage
  const formes = pageData?.formes_retenues
  // CE QUE L'ONGLET A RÉELLEMENT À MONTRER, bloc par bloc — mêmes prédicats que les
  // composants eux-mêmes (usageAvailabilityKind pour l'équipement, `available` pour les
  // formes). Les deux à faux : l'onglet le DIT, au lieu de laisser une page nue.
  const hasEquipment = usageAvailabilityKind(equipmentUsage) !== 'hidden'
  const hasFormes = formes != null && formes.available
  const hasAnyBlock = hasEquipment || hasFormes

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
      {/* ÉQUIPEMENT ET ARMES DE SOCLE — bloc « servi ou gâché », variante comptes (une
          ligne par coéquipier suivi). Aucune requête neuve : lit `pageData.equipment_usage`
          de la réponse déjà chargée par `useTeammates`. LE TITRE NOMME LES DEUX FAMILLES DE
          RAMASSAGES qu'il coiffe — l'équipement (usages, part du lobby) et les armes posées
          sur les socles (armes spéciales, niveaux) — PAS SA PREMIÈRE CARTE, qui s'appelle
          déjà « Usages d'équipement » : reprendre ce libellé ici l'aurait écrit deux fois de
          suite (arbitrage du 2026-09-22). La section se retire d'elle-même sans bloc. */}
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
