/**
 * SquadSynergiesPage — onglet Synergies de l'Escouade.
 *
 * Distingue 2 états vides diagnosticables :
 *  - no_selection : aucun coéquipier confirmé.
 *  - invalid_selection : confirmedGts > 0 mais selectedRows vide.
 */
import { useMemo } from 'react'

import { Card, CardContent } from '@/components/ui/card'
import { SectionTitle } from '@/components/ui/detail-section'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { useAppShellStore } from '@/stores/appShellStore'
import { useCapability } from '@/lib/capabilities/capabilities'
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
import { SquadAssistPairsChart } from './SquadAssistPairsChart'
import { SquadEchangeConstatCard } from './SquadEchangeConstatCard'
import { SquadEchangeCompteCard } from './SquadEchangeCompteCard'
import { SquadEchangeDelaiCard } from './SquadEchangeDelaiCard'
import { SquadEchangeDonneRecuCard } from './SquadEchangeDonneRecuCard'
import { SquadEchangeMatrixCard } from './SquadEchangeMatrixCard'
import { SquadEchangeTauxSessionCard } from './SquadEchangeTauxSessionCard'
import { SquadIsolementNuageCard } from './SquadIsolementNuageCard'
import { SquadSynergyHistoryTable } from './SquadSynergyHistoryTable'
import { SquadImpactScoreboard } from './SquadImpactScoreboard'
import { MedalDigest } from './MedalDigest'
import { EquipmentUsageSection } from '@/features/_shared/usage/EquipmentUsageSection'
import { USAGE_TEXT } from '@/features/_shared/usage/usageI18n'
import { FormesRetenuesSection } from './formes/FormesRetenuesSection'
import { SquadFragSection } from './SquadFragSection'
import { SquadFdaGapCumulativeCard } from './SquadFdaGapCumulativeCard'
import { getSquadPlayerColors } from './colors'

export function SquadSynergiesPage() {
  const { selectedRows, confirmedGamertags, pageData, playerSlug } = useSquadContext()
  const { data: mappings } = useFieldMappings()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  // FDA attendu natif (Infinite déclare `expected_stats`, Halo 5 non) → gate
  // PARENT du card « Écart cumulé au FDA attendu » : pas de colonne vide dans la
  // rangée 1 de SquadFragSection (le card conserve son self-gate en profondeur).
  const hasExpectedStats = useCapability('expected_stats')
  // Libellés des drapeaux de dominance (bande de résultats) — table canonique
  // partagée avec la colonne Dominance de l'Explorateur. Mémoïsé : la bande
  // recalcule son option ECharts quand cette référence change. Déclaré AVANT les
  // retours anticipés (règle des hooks).
  const tapeDominanceLabels = useMemo(() => buildDominanceLabels(locale), [locale])

  // Ordre / couleurs / libellés de résultat — MÉMOÏSÉS et déclarés AVANT les
  // retours anticipés (règle des hooks). Sans mémo, un simple rendu de la page
  // (changement de contexte, refetch qui rend la même donnée) fabriquait des
  // props neuves pour SquadFragSection, SquadAssistPairsChart et
  // OutcomeSequenceTape : les `useMemo` de ces graphes se re-déclenchaient, la
  // ChartCard rebâtissait son option ECharts (dont les `formatter`, comparés par
  // référence par echarts-for-react) et l'animation d'entrée REJOUAIT sans
  // qu'aucune valeur n'ait bougé.
  // Le backend renvoie s.gamertag (casse mixte ex "Madina97294") tandis que
  // playerSlug est l'URL param (souvent lowercase) : on aligne sur main_player.
  const mainPlayerKey = pageData?.main_player ?? playerSlug
  const fragClasses = pageData?.frag_classes
  const performanceSeries = pageData?.performance_series
  // Repli stable : `?? {}` écrit dans le JSX fabriquait un objet neuf à CHAQUE
  // rendu quand le bloc est absent — même symptôme, en pire (rien à afficher).
  const fragClassesByPlayer = useMemo(() => fragClasses ?? {}, [fragClasses])
  const perfSeriesByPlayer = useMemo(() => performanceSeries ?? {}, [performanceSeries])
  const playerColors = useMemo(
    () => getSquadPlayerColors(mainPlayerKey, confirmedGamertags),
    [mainPlayerKey, confirmedGamertags],
  )
  // Section « frags » (relocalisée depuis Contributions) : mêmes couleurs/ordre
  // que SquadContributionsPage — main_player (casse serveur) puis coéquipiers,
  // restreint aux joueurs ayant des frag_classes ou une performance_series.
  const playerOrder = useMemo(
    () =>
      [mainPlayerKey, ...confirmedGamertags].filter((p) => fragClasses?.[p] || performanceSeries?.[p]),
    [mainPlayerKey, confirmedGamertags, fragClasses, performanceSeries],
  )
  // Roster dans l'ordre de la page : joueur principal d'abord, puis les coéquipiers. Le
  // graphe des assistances s'en sert pour l'ordre des barres ET pour les couleurs par
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
  // L'ECHANGE (mort vengee dans les 5 s) : comme assist_pairs, son absence est un
  // ETAT (titre qui ne nomme pas le tueur de chaque mort, ou aucun match mesure) et
  // non un zero. Les blocs ne sont alors pas montes du tout.
  const echange = pageData?.echange

  return (
    <div className="space-y-4">
      {/* « Constat du moment » EN TÊTE, AU-DESSUS du « Compte » : c'est le titre narratif
          du récit de l'échange (le « Cap du moment » de la maquette 4c520da6), pas un
          doublon d'une des six cartes. Il se rend de lui-même sous ses deux seuils
          (30 morts d'équipe ET 5 points d'écart) : rien à passer ici, et rien du tout à
          l'écran quand il n'a rien à dire. */}
      <SquadEchangeConstatCard echange={echange} />
      {/* L'ÉCHANGE, SIX CARTES, DANS L'ORDRE DE LECTURE DE LA MAQUETTE 4c520da6 :
          combien (Le compte) → à quelle vitesse → qui couvre qui → pourquoi la vengeance
          ne vient pas → donné/reçu → l'évolution par soirée. Les deux premières comptent,
          la troisième dit qui, la quatrième explique — et c'est elle qui donne quelque
          chose à corriger. */}
      {echange && (
        <>
          <SquadEchangeCompteCard echange={echange} />
          <SquadEchangeDelaiCard echange={echange} />
        </>
      )}
      {/* « Qui couvre qui » et « Assistances dans l'escouade » SUR LA MÊME RANGÉE
          (2026-09-19) : deux lectures du même couple de joueurs, l'une par vengeance,
          l'autre par assistance — les empiler obligeait à faire défiler entre les deux.
          `items-stretch` par défaut : les deux cartes ont la même hauteur.

          LES DEUX SE MONTENT INDÉPENDAMMENT : l'échange vient du journal des morts, les
          assistances du résumé du film. Un titre qui ne nomme pas le tueur garde ses
          assistances — les lier aurait fait disparaître une mesure qui existe. */}
      {(echange || assistPairs) && (
        <div className="grid grid-cols-2 gap-4">
          {echange && <SquadEchangeMatrixCard echange={echange} />}
          {assistPairs && <SquadAssistPairsChart block={assistPairs} roster={roster} />}
        </div>
      )}
      {echange && (
        <>
          {echange.nuage_isolement && (
            <SquadIsolementNuageCard
              nuage={echange.nuage_isolement}
              joueurs={echange.joueurs ?? []}
            />
          )}
          <SquadEchangeDonneRecuCard echange={echange} />
          <SquadEchangeTauxSessionCard echange={echange} />
        </>
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
      {/* Section « frags » (relocalisée depuis Contributions), juste avant « Impact
          des coéquipiers ». Rangée 1 : sur Infinite « Écart cumulé au FDA attendu »
          (gate `expected_stats`) à GAUCHE de « Répartition des frags » ; sur Halo 5
          « Répartition » | « Précision par rôle ». Puis « Outils de destruction ».
          Le card FDA porte ses propres pastilles KPI « écart moyen / match » et
          garde son self-gate capability (défense en profondeur). */}
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
      {/* Sections non-graphes toujours montées : titre + état vide géré par le
          composant (cadre bordé / carte), au lieu de disparaître. */}
      <section className="space-y-3">
        <SectionTitle>{t.impact.title}</SectionTitle>
        <SquadImpactScoreboard
          matrix={pageData?.impact_matrix ?? { matches: [], players: [], cells: [], badge_ord: [] }}
        />
      </section>
      {/* Bloc « servi ou gâché » de l'équipement (PLAN_EQUIPEMENT_GACHIS_2026-09-09,
          E6.2-E6.4) — variante comptes (P9), une ligne par coéquipier suivi. Aucune
          requête neuve : lit `pageData.equipment_usage` de la même réponse déjà
          chargée par `useTeammates`. Le Go le publie sur `TeammatesPageResponse`
          (POST /pages/teammates, lot E6.1bis du 2026-09-09) ; la section se retire
          d'elle-même si le champ est absent (titre sans résumé d'usage, scope vide). */}
      <EquipmentUsageSection usage={pageData?.equipment_usage} mode="squad" t={USAGE_TEXT[locale]} locale={locale} />
      {/* « Les formes retenues » (artefact 2ec1b8eb, lot D2), CONTEXTE ESCOUADE SEUL
          (2026-09-19) : les neuf cartes du contexte solo vivent désormais sur Timeseries,
          onglet Progression — une page, un contexte. Aucune requête neuve : lit
          `pageData.formes_retenues` de la réponse déjà chargée par `useTeammates`. La
          section se retire d'elle-même quand le bloc est absent (titre sans film, scope
          vide). */}
      <FormesRetenuesSection
        block={pageData?.formes_retenues}
        locale={locale}
        contexte="squad"
        mainPlayerLabel={pageData?.main_player ?? playerSlug}
      />
      {/* MÉDAILLES EN DERNIER (décision utilisateur, 2026-09-13) : c'est un palmarès, pas
          une mesure — il se lit après tout ce qui explique le jeu, jamais avant. */}
      <section className="space-y-3">
        <SectionTitle>{t.medals.title}</SectionTitle>
        <MedalDigest
          entries={pageData?.medal_digest ?? []}
          mainPlayer={pageData?.main_player ?? playerSlug}
          t={t.medals}
        />
      </section>
    </div>
  )
}
