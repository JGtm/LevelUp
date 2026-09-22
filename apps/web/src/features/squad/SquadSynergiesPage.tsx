/**
 * SquadSynergiesPage — onglet Synergies de l'Escouade.
 *
 * UN SEUL AXE DE LECTURE (lot 3 « sections », 2026-09-22) : ce que la composition
 * PRODUIT ENSEMBLE — l'échange, le taux de victoire et les cartes face à l'historique,
 * la suite des résultats, l'historique, la heatmap des cartes, la frise des sessions.
 * Ce qu'elle UTILISE (frags et armes, équipement, formes retenues) a rejoint l'onglet
 * Usages ; l'impact des coéquipiers et les médailles ont rejoint Contributions.
 *
 * Distingue 2 états vides diagnosticables :
 *  - no_selection : aucun coéquipier confirmé.
 *  - invalid_selection : confirmedGts > 0 mais selectedRows vide.
 */
import { useMemo } from 'react'

import { Card, CardContent } from '@/components/ui/card'
import { SectionTitle } from '@/components/ui/detail-section'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { InfoTooltip, TooltipParagraphs } from '@/components/ui/info-tooltip'
import { useAppShellStore } from '@/stores/appShellStore'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { OutcomeSequenceTape, type OutcomePoint } from '@/components/charts/OutcomeSequenceTape'
import { asDominance } from '@/components/charts/outcomeSequence'
import { ReviewBadge } from '@/components/charts/ReviewBadge'
import { dominanceLabels as buildDominanceLabels } from '@/lib/narrative/dominance'
import { outcomeCodeToTapeValue } from '@/lib/outcome'
import { useSquadContext } from './SquadContext'
import { getSquadText } from './i18n'
import { WinRateVsHistoryBulletChart } from './WinRateVsHistoryBulletChart'
import { MapPerfVsHistoryChart } from './MapPerfVsHistoryChart'
import { SquadMapHeatmapChart } from './SquadMapHeatmapChart'
import { SquadSessionTimelineChart } from './SquadSessionTimelineChart'
import { SquadAppuiCard } from './SquadAppuiCard'
import { SquadRiposteCard } from './SquadRiposteCard'
import { getSquadRiposteText } from './squadRiposteStrings'
import { SquadIsolementNuageCard } from './SquadIsolementNuageCard'
import { SquadRangeRolesCard } from './SquadRangeRolesCard'
import { SquadSynergyHistoryTable } from './SquadSynergyHistoryTable'

export function SquadSynergiesPage() {
  const { selectedRows, confirmedGamertags, pageData, playerSlug } = useSquadContext()
  const { data: mappings } = useFieldMappings()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const tRiposte = getSquadRiposteText(locale)
  // Libellés des drapeaux de dominance (bande de résultats) — table canonique
  // partagée avec la colonne Dominance de l'Explorateur. Mémoïsé : la bande
  // recalcule son option ECharts quand cette référence change. Déclaré AVANT les
  // retours anticipés (règle des hooks).
  const tapeDominanceLabels = useMemo(() => buildDominanceLabels(locale), [locale])

  // Ordre / couleurs / libellés de résultat — MÉMOÏSÉS et déclarés AVANT les
  // retours anticipés (règle des hooks). Sans mémo, un simple rendu de la page
  // (changement de contexte, refetch qui rend la même donnée) fabriquait des
  // props neuves pour SquadRiposteCard, SquadAppuiCard et
  // OutcomeSequenceTape : les `useMemo` de ces graphes se re-déclenchaient, la
  // ChartCard rebâtissait son option ECharts (dont les `formatter`, comparés par
  // référence par echarts-for-react) et l'animation d'entrée REJOUAIT sans
  // qu'aucune valeur n'ait bougé.
  // Le backend renvoie s.gamertag (casse mixte ex "Madina97294") tandis que
  // playerSlug est l'URL param (souvent lowercase) : on aligne sur main_player.
  const mainPlayerKey = pageData?.main_player ?? playerSlug
  // Roster dans l'ordre de la page : joueur principal d'abord, puis les coéquipiers. Le
  // graphe d appui s'en sert pour l'ordre des barres ET pour les couleurs par
  // joueur — mêmes teintes que partout ailleurs sur la page.
  const roster = useMemo(
    () => [mainPlayerKey, ...confirmedGamertags],
    [mainPlayerKey, confirmedGamertags],
  )
  const outcomes = mappings?.outcomes
  const outcomeLabels = useMemo(
    () => ({
      win: outcomes?.['win']?.label ?? t.history.outcomeLabel.win,
      loss: outcomes?.['loss']?.label ?? t.history.outcomeLabel.loss,
      tie: outcomes?.['tie']?.label ?? t.history.outcomeLabel.draw,
      dnf: outcomes?.['dnf']?.label ?? t.history.outcomeLabel.dnf,
    }),
    [outcomes, t],
  )

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

  const mapAssets = mappings?.assets?.['map']
  const mapLabelOf = (mapUI: string) => mapAssets?.[mapUI]?.label ?? mapUI
  const mapBreakdown = pageData?.map_breakdown ?? []
  const matchHistory = pageData?.match_history ?? []
  const sessionTimeline = pageData?.session_timeline ?? []
  const mapHeatmap = pageData?.map_heatmap
  // assist_pairs n'est PAS comblé par un défaut : son absence est un ÉTAT (aucun match
  // de la sélection n'a d'assistance mesurée — dont le cas d'un titre sans décodeur de
  // film). Le bloc n'est alors pas monté du tout, plutôt que d'afficher un cadre vide
  // qui laisserait croire à une escouade sans entraide.
  const assistPairs = pageData?.assist_pairs
  // LA RIPOSTE (mort de notre camp dont le tueur tombe dans les 5 s) : comme assist_pairs, son absence est un
  // ETAT (titre qui ne nomme pas le tueur de chaque mort, ou aucun match mesure) et
  // non un zero. Les blocs ne sont alors pas montes du tout.
  const echange = pageData?.echange
  // Les profils de PORTÉE par match (lot N2) : même politique que les deux blocs
  // ci-dessus — absent = aucun film décodé sur la sélection, jamais des zéros.
  const rangeProfiles = pageData?.range_profiles

  return (
    <div className="space-y-4">
      {/* SECTION « COORDINATION » (D19, 2026-09-21 ; TROIS RANGÉES, 2026-09-22) — DEUX
          NOTIONS, CINQ CARTES. Elle en montait HUIT empilées, dont sept portaient la même
          notion sous quatre noms différents (échange, vengeance, assistance croisée,
          riposte). La riposte vit désormais dans une seule carte-récit, l'appui dans la
          sienne.

          L'ORDRE EST CELUI DES RANGÉES, pas celui d'une pile :
            1. « Appui » et « Frags non ripostés » côte à côte — les deux faces de ce que
               l'escouade se doit l'une à l'autre : ce qu'elle donne, ce qu'elle ne rend pas ;
            2. la RIPOSTE, en TROIS BLOCS montés par `SquadRiposteCard` (un fragment, pas
               une carte) : « Morts ripostées » et « Temps de riposte » côte à côte sur une
               rangée, puis « Riposte » — frise et repli — pleine largeur sous elles ;
            3. « Rôles de portée » et « Rôles de hauteur » côte à côte — même bloc de
               données, même nuage, même grammaire (E1, D24 du 2026-09-22).

          CHAQUE CARTE SE MONTE INDÉPENDAMMENT : la riposte vient du journal des morts,
          l'appui du résumé du film, les rôles des films décodés. Un titre qui ne nomme pas
          le tueur de chaque mort garde son appui — les lier aurait fait disparaître une
          mesure qui existe. Une rangée dont une cellule manque reste une grille : la carte
          présente prend sa colonne. */}
      {(echange || assistPairs || rangeProfiles) && (
        <section className="space-y-4" aria-label={tRiposte.coordinationTitle}>
          <SectionTitle className="flex items-center gap-1.5">
            {tRiposte.coordinationTitle}
            <InfoTooltip
              content={
                <TooltipParagraphs
                  items={[
                    tRiposte.coordinationHelpRiposte((echange?.fenetre_ms ?? 5000) / 1000),
                    tRiposte.coordinationHelpAppui,
                  ]}
                />
              }
            />
          </SectionTitle>
          {/* RANGÉE 1 — « Appui » à gauche, « Frags non ripostés » à droite. */}
          {(assistPairs || echange?.nuage_isolement) && (
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              {assistPairs && <SquadAppuiCard block={assistPairs} roster={roster} />}
              {echange?.nuage_isolement && (
                <SquadIsolementNuageCard
                  nuage={echange.nuage_isolement}
                  joueurs={echange.joueurs ?? []}
                />
              )}
            </div>
          )}
          {/* RANGÉE 2 — la riposte : une rangée de deux blocs, puis la frise pleine largeur. */}
          {echange && <SquadRiposteCard echange={echange} />}
          {/* RANGÉE 3 — la PORTÉE et la HAUTEUR côte à côte : même bloc de données, même
              nuage, même grammaire — qui tient la distance, qui tient la hauteur. Leur
              absence est un ÉTAT (aucun film décodé sur la sélection) : la rangée n'est
              alors pas montée. La hauteur, elle, rend son état vide nommé quand aucun match
              de la sélection ne porte de dénivelé mesuré. */}
          {rangeProfiles && (
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              <SquadRangeRolesCard bloc={rangeProfiles} roster={roster} />
              <SquadRangeRolesCard bloc={rangeProfiles} roster={roster} grandeur="hauteur" />
            </div>
          )}
        </section>
      )}
      {/* Graphes toujours montés : ChartCard affiche son état vide (titre +
          message) au lieu de faire disparaître le bloc quand mapBreakdown
          est vide ou sans champs de performance. */}
      <div className="grid grid-cols-2 gap-4">
        <WinRateVsHistoryBulletChart
          title={
            <span className="flex items-center gap-1.5">
              {t.charts.winRateVsHistoryBulletTitle}
              <InfoTooltip content={t.charts.winRateVsHistoryBulletMapCountTooltip} />
            </span>
          }
          emptyMessage={t.empty.noBlockData}
          rows={mapBreakdown}
          mapLabelOf={mapLabelOf}
          sessionLabel={t.charts.winRateVsHistorySession}
          historyLabel={t.charts.winRateVsHistoryHistory}
          parityLabel={t.charts.winRateVsHistoryBulletParity}
          zeroWinrateLabel={t.charts.winRateVsHistoryBulletZero}
          countsLabel={t.charts.winRateVsHistoryBulletCounts}
        />
        <MapPerfVsHistoryChart
          title={t.charts.mapPerfVsHistoryTitle}
          emptyMessage={t.empty.noBlockData}
          rows={mapBreakdown}
          mapLabelOf={mapLabelOf}
          sessionLabel={t.charts.mapPerfVsHistorySession}
          historyLabel={t.charts.mapPerfVsHistoryHistory}
        />
      </div>
      {/* HISTORIQUE — le titre de section coiffe DEUX blocs : la bande de résultats et
          le tableau des matchs. L'intitulé de la bande (le `<p>` en capitales) reste en
          place, comme sous-titre du bloc. */}
      <section className="space-y-4">
        <SectionTitle>{t.sections.historique}</SectionTitle>
        {/* Séquence des résultats : on garde le libellé + un message court quand
            il n'y a pas d'historique, au lieu de masquer le bloc. */}
        <div>
          <p className="mb-1 flex items-center text-xs font-medium uppercase tracking-wide text-muted-foreground">
            {t.charts.outcomeSequenceTitle}
            <ReviewBadge reviewKey="squad.outcome_tape" />
          </p>
          {matchHistory.length > 0 ? (
            <OutcomeSequenceTape
              // matchHistory arrive DESC (récent→ancien) ; on inverse pour afficher
              // du plus vieux au plus récent (gauche→droite).
              matches={[...matchHistory].reverse().map<OutcomePoint>((m) => ({
                outcome: outcomeCodeToTapeValue(m.outcome),
                matchId: m.match_id,
                map: m.map_ui || undefined,
                mode: m.mode_ui || m.pair_name || undefined,
                // Absent (0/undefined, ex. Halo 5 sans timeline de score) → aucun
                // marqueur dessiné, aucun suffixe de tooltip.
                dominance: asDominance(m.dominance_flag),
              }))}
              labels={outcomeLabels}
              dominanceLabels={tapeDominanceLabels}
            />
          ) : (
            <p className="text-sm text-muted-foreground">{t.empty.noBlockData}</p>
          )}
        </div>
        <SquadSynergyHistoryTable rows={matchHistory} playerSlug={playerSlug} />
      </section>
      <SquadMapHeatmapChart
        title={t.heatmap.title}
        emptyMessage={t.empty.noBlockData}
        data={mapHeatmap && (mapHeatmap.players?.length ?? 0) > 0 && (mapHeatmap.maps_topn?.length ?? 0) > 0 ? mapHeatmap : undefined}
        mapLabelOf={mapLabelOf}
        pieceLabels={{
          tier1: t.heatmap.pieceTier1,
          tier2: t.heatmap.pieceTier2,
          tier3: t.heatmap.pieceTier3,
          tier4: t.heatmap.pieceTier4,
          tier5: t.heatmap.pieceTier5,
        }}
        noScoreLabel={t.heatmap.noScore}
        yAxisName={t.heatmap.yAxis}
      />
      <SquadSessionTimelineChart
        title={t.timeline.title}
        emptyMessage={t.empty.noBlockData}
        rows={sessionTimeline}
        perfLabel={t.timeline.perf}
        winRateLabel={t.timeline.winRate}
        mmrLabel={t.timeline.teamMmr}
        perfAxisLabel={t.timeline.perfAxis}
        mmrAxisLabel={t.timeline.mmrAxis}
      />
    </div>
  )
}
