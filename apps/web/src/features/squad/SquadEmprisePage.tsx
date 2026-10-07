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
 *   2. « Prises par joueur » : les fiches des joueurs de l'escouade, à même la section (comme les
 *      médailles : titre et aide sur l'intertitre), pleine largeur ;
 *   3. « Carte par carte » : « Contrôle des ressources, match par match », pleine largeur ;
 *   4. « Prendre, et s'en servir » : « Frags obtenus avec les ressources » | « Rendement face à
 *      l'adversaire », côte à côte, même hauteur (la survivante prend la rangée) ;
 *   5. « Groupés ou isolés » : « Placement et rendement de chaque vie » (nuage, pleine largeur)
 *      puis « Part des vies par placement » (barres), sous le nuage (lot V4 du plan
 *      PLAN_EMPRISE_VIES_2026-09-28) ; absent sans vie mesurée (Halo 5, portée du radar inconnue) ;
 *   6. « Par rapport à d'habitude » : « Contrôle des ressources, soirée après soirée », dans la
 *      colonne de gauche comme dans la maquette (la carte d'isolement de droite est hors
 *      périmètre : rien n'est rendu à sa place) ;
 *   7. « Objectif » puis « Répartition de l'objectif dans l'escouade » (arrivés de Contributions le
 *      2026-10-07, `SquadObjectiveSection`) : lus sur la feuille de match, présents aussi sans film.
 *
 * Aucune fiche ni ligne du reste du camp : les joueurs inconnus ne se lisent pas ici.
 *
 * Sans film (Halo 5, D10), seuls les frags aux armes spéciales restent (barre épaisse seule) ;
 * sans rien à montrer, la barre d'onglets masque l'onglet (même prédicat : `empriseContent.ts`).
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
import { InfoTooltip } from '@/components/ui/info-tooltip'
import type { Locale } from '@/lib/i18n/locale'
import { dominanceLabels } from '@/lib/narrative/dominance'
import { useAppShellStore } from '@/stores/appShellStore'

import { empriseSections, type EmpriseSections } from './emprise/empriseContent'
import { EMPRISE_TEXT, type EmpriseText } from './emprise/empriseStrings'
import { HabitCard } from './emprise/HabitCard'
import type { HabitView } from './emprise/habit.logic'
import { PickupSheetsCard } from './emprise/PickupSheetsCard'
import type { PlacementBlock } from './emprise/placementCharts'
import { PLACEMENT_TEXT } from './emprise/placementStrings'
import { PlacementQuartsCard } from './emprise/PlacementQuartsCard'
import { PlacementVieCard } from './emprise/PlacementVieCard'
import { ProductionCard } from './emprise/ProductionCard'
import type { ProductionRow, YieldRow } from './emprise/production.logic'
import { ResourceControlCard } from './emprise/ResourceControlCard'
import { ResourceFilCard } from './emprise/ResourceFilCard'
import { ResourceMatchGridCard } from './emprise/ResourceMatchGridCard'
import { useEmpriseModels } from './emprise/useEmpriseModels'
import { useOutcomeLabels } from './emprise/useOutcomeLabels'
import type { VehicleCoverage } from './emprise/vehicles.logic'
import { YieldCard } from './emprise/YieldCard'
import { getSquadText } from './i18n'
import { hasSquadObjective } from './objectif/objectif.logic'
import { SquadObjectiveSection } from './objectif/SquadObjectiveSection'
import { useSquadContext } from './SquadContext'

export function SquadEmprisePage() {
  const { selectedRows, confirmedGamertags, pageData, playerSlug } = useSquadContext()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const et = EMPRISE_TEXT[locale]
  const dominance = useMemo(() => dominanceLabels(locale), [locale])
  const outcomes = useOutcomeLabels()
  const mainPlayer = pageData?.main_player ?? playerSlug
  const models = useEmpriseModels(pageData, mainPlayer, locale)
  const matchHistory = useMemo(() => pageData?.match_history ?? [], [pageData?.match_history])
  const medalDigest = useMemo(() => pageData?.medal_digest ?? [], [pageData?.medal_digest])
  const { objectName, controlRows, fil, sheets, grid, production, yieldRows, vehicleCoverage, habit, placement, identities, playerName } = models
  const show = empriseSections(models)

  if (confirmedGamertags.length === 0 || selectedRows.length === 0) {
    const none = confirmedGamertags.length === 0
    return (
      <EmptyCard
        title={none ? t.empty.noSelectionTitle : t.empty.invalidSelectionTitle}
        description={none ? t.empty.noSelectionDescription : t.empty.invalidSelectionDescription}
      />
    )
  }

  if (!Object.values(show).some(Boolean) && !hasSquadObjective(pageData?.formes_retenues)) {
    return <EmptyCard title={t.empty.noDecodedFilmTitle} description={t.empty.noDecodedFilmDescription} />
  }

  return (
    <div className="space-y-6" data-testid="squad-emprise-page">
      {show.bilan && fil && (
        <section className="space-y-2" data-testid="emprise-section-bilan">
          <SectionTitle>{et.sections.bilan}</SectionTitle>
          <div className="grid gap-4 lg:grid-cols-2">
            <ResourceControlCard rows={controlRows} t={et} />
            <ResourceFilCard fil={fil} dominanceLabels={dominance} outcomeLabels={outcomes} locale={locale} t={et} />
          </div>
        </section>
      )}
      {show.roles && sheets && (
        <section className="space-y-2" data-testid="emprise-section-roles">
          <SectionTitle className="flex items-center gap-1.5">
            {et.sections.roles}
            <InfoTooltip content={et.sheets.info} />
          </SectionTitle>
          <PickupSheetsCard sheets={sheets} identities={identities} itemName={(line) => objectName(line.object)} bare t={et} />
        </section>
      )}
      {show.carte && grid && (
        <section className="space-y-2" data-testid="emprise-section-carte">
          <SectionTitle>{et.sections.carte}</SectionTitle>
          <ResourceMatchGridCard
            grid={grid}
            itemName={(row) => (row.object ? objectName(row.object) : '')}
            playerName={playerName}
            outcomeLabels={outcomes}
            locale={locale}
            t={et}
            namedOnly
          />
        </section>
      )}
      <UsageSections
        show={show}
        production={production}
        yieldRows={yieldRows}
        vehicleCoverage={vehicleCoverage}
        habit={habit}
        placement={placement}
        locale={locale}
        et={et}
      />
      <SquadObjectiveSection
        block={pageData?.formes_retenues}
        matchHistory={matchHistory}
        objectiveHistory={pageData?.squad_objective_history}
        medalDigest={medalDigest}
        mainPlayerLabel={mainPlayer}
        locale={locale}
      />
    </div>
  )
}

/** Un état vide de l'onglet, dans sa carte. */
function EmptyCard({ title, description }: { title: string; description: string }) {
  return (
    <Card>
      <CardContent className="pt-4">
        <EmptyStateNotice title={title} description={description} />
      </CardContent>
    </Card>
  )
}

/** Blocs 4 à 6 : « Prendre, et s'en servir », « Groupés ou isolés », puis « Par rapport à d'habitude ». */
function UsageSections({
  show,
  production,
  yieldRows,
  vehicleCoverage,
  habit,
  placement,
  locale,
  et,
}: {
  show: EmpriseSections
  production: ProductionRow[]
  yieldRows: YieldRow[]
  vehicleCoverage: VehicleCoverage | null
  habit: HabitView
  placement: PlacementBlock | null
  locale: Locale
  et: EmpriseText
}) {
  const pt = PLACEMENT_TEXT[locale]
  return (
    <>
      {show.prendre && (
        <section className="space-y-2" data-testid="emprise-section-prendre">
          <SectionTitle>{et.sections.prendre}</SectionTitle>
          {/* Une carte seule (sans rendement : Halo 5, D10) prend la rangée (précédent : Dynamique, L1). */}
          <div className="grid gap-4 lg:grid-cols-2 lg:[&>*:only-child]:col-span-2">
            {production.length > 0 && <ProductionCard rows={production} t={et} />}
            {yieldRows.length > 0 && <YieldCard rows={yieldRows} coverage={vehicleCoverage} t={et} />}
          </div>
        </section>
      )}
      {show.placement && placement && (
        <section className="space-y-2" data-testid="emprise-section-placement">
          <SectionTitle>{pt.section}</SectionTitle>
          <div className="space-y-4">
            <PlacementVieCard placement={placement} locale={locale} t={pt} />
            <PlacementQuartsCard placement={placement} locale={locale} t={pt} />
          </div>
        </section>
      )}
      {show.habitude && habit.kind !== 'none' && (
        <section className="space-y-2" data-testid="emprise-section-habitude">
          <SectionTitle>{et.sections.habitude}</SectionTitle>
          {/* Demi-largeur à gauche, comme la maquette : la carte d'isolement de droite est hors périmètre. */}
          <div className="grid gap-4 lg:grid-cols-2">
            <HabitCard view={habit} locale={locale} t={et} />
          </div>
        </section>
      )}
    </>
  )
}
