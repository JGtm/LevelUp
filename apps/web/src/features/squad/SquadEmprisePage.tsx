/**
 * SquadEmprisePage — l'onglet « Emprise » de l'Escouade (ex-Usages ; lot L5 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, maquette
 * `.ai/V7.5/MAQUETTE_ONGLET_TACTIQUE_ESCOUADE_2026-09-26.html`, bloc « Proposition »).
 *
 * UN SEUL AXE DE LECTURE : qui a tenu la carte — les ressources (bonus, armes spéciales, armes
 * de râtelier) prises par notre camp face à l'adversaire, dans l'ordre d'un débrief :
 *
 *   1. « Bilan de la soirée » : « Contrôle des ressources » | « Contrôle des ressources au fil
 *      de la session », côte à côte, même hauteur (S2) ;
 *   2. « Rôles dans l'escouade » : « Répartition des prises dans l'escouade », pleine largeur ;
 *   3. « Carte par carte » : « Contrôle des ressources, match par match », pleine largeur.
 *
 * AUCUNE REQUÊTE NEUVE : la page lit `SquadContext` (bloc `squad_emprise`, historique de matchs
 * pour le résultat, la carte et la dominance ; emblèmes des fiches de médailles), publié par
 * `SquadLayout` depuis l'unique `useTeammates`. Mêmes gardes que l'onglet qu'elle remplace
 * (sélection vide ou invalide) ; sans rien à montrer, l'onglet le DIT au lieu de rester nu. Un
 * bloc sans donnée se retire, intertitre compris.
 */
import { useMemo } from 'react'

import { Card, CardContent } from '@/components/ui/card'
import { SectionTitle } from '@/components/ui/detail-section'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { dominanceLabels } from '@/lib/narrative/dominance'
import { useAppShellStore } from '@/stores/appShellStore'

import { EMPRISE_TEXT } from './emprise/empriseStrings'
import { PickupSheetsCard } from './emprise/PickupSheetsCard'
import { ResourceControlCard } from './emprise/ResourceControlCard'
import { ResourceFilCard } from './emprise/ResourceFilCard'
import { ResourceMatchGridCard } from './emprise/ResourceMatchGridCard'
import { useEmpriseModels } from './emprise/useEmpriseModels'
import { TEAM_REST_INK } from './formes/colors'
import { getSquadText } from './i18n'
import { useSquadContext } from './SquadContext'

export function SquadEmprisePage() {
  const { selectedRows, confirmedGamertags, pageData, playerSlug } = useSquadContext()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const et = EMPRISE_TEXT[locale]
  const dominance = useMemo(() => dominanceLabels(locale), [locale])
  const { objectName, controlRows, fil, sheets, grid, identities, playerName } = useEmpriseModels(
    pageData,
    pageData?.main_player ?? playerSlug,
    et.sheets.rest,
    locale,
  )

  if (confirmedGamertags.length === 0 || selectedRows.length === 0) {
    const none = confirmedGamertags.length === 0
    return (
      <Card>
        <CardContent className="pt-4">
          <EmptyStateNotice
            title={none ? t.empty.noSelectionTitle : t.empty.invalidSelectionTitle}
            description={none ? t.empty.noSelectionDescription : t.empty.invalidSelectionDescription}
          />
        </CardContent>
      </Card>
    )
  }

  const hasBilan = controlRows.length > 0 && fil != null
  const hasSheets = sheets != null && sheets.sections.some((s) => s.lines.length > 0)
  const hasGrid = grid != null && grid.columns.length > 0 && grid.sections.length > 0

  if (!hasBilan && !hasSheets && !hasGrid) {
    return (
      <Card>
        <CardContent className="pt-4">
          <EmptyStateNotice title={t.empty.noDecodedFilmTitle} description={t.empty.noDecodedFilmDescription} />
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-6" data-testid="squad-emprise-page">
      {hasBilan && (
        <section className="space-y-2" data-testid="emprise-section-bilan">
          <SectionTitle>{et.sections.bilan}</SectionTitle>
          <div className="grid gap-4 lg:grid-cols-2">
            <ResourceControlCard rows={controlRows} t={et} />
            <ResourceFilCard fil={fil} dominanceLabels={dominance} locale={locale} t={et} />
          </div>
        </section>
      )}
      {hasSheets && (
        <section className="space-y-2" data-testid="emprise-section-roles">
          <SectionTitle>{et.sections.roles}</SectionTitle>
          <PickupSheetsCard
            sheets={sheets}
            identities={identities}
            itemName={(line) => objectName(line.object)}
            restColor={TEAM_REST_INK}
            t={et}
          />
        </section>
      )}
      {hasGrid && (
        <section className="space-y-2" data-testid="emprise-section-carte">
          <SectionTitle>{et.sections.carte}</SectionTitle>
          <ResourceMatchGridCard
            grid={grid}
            itemName={(row) => (row.object ? objectName(row.object) : '')}
            playerName={playerName}
            dominanceLabels={dominance}
            locale={locale}
            t={et}
          />
        </section>
      )}
    </div>
  )
}
