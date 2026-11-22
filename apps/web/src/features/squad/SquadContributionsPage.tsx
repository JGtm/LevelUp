/**
 * SquadContributionsPage — onglet Contributions de l'Escouade.
 *
 * Consomme le contexte SquadContext fourni par SquadLayout. Affiche les
 * charts de contribution par joueur : K/D/A par minute, synergies radar, frags et armes
 * (Répartition des frags, Outils de destruction — arrivés d'Usages au lot L2 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26), performance, impact des coéquipiers (un tableau,
 * une ligne par session, dépliable sur sa matrice), médailles,
 * mécaniques de frag. L'objectif est parti dans l'onglet Emprise (2026-10-07). Le
 * « Premier frag / première mort » a rejoint l'onglet Dynamique (chart lanes) ;
 * l'impact et les médailles sont arrivés de Synergies (lot 3, 2026-09-22).
 *
 * Multi-titres : strings UI via getSquadText.
 */
import { withoutNullRows } from './withoutNullRows'
import { useMemo } from 'react'
import { Link } from '@tanstack/react-router'
import { SectionTitle } from '@/components/ui/detail-section'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { useAppShellStore } from '@/stores/appShellStore'
import { substituteHpToken, useEffectiveHpToKill } from '@/lib/damage/effectiveHp'
import type { MedalDigestEntry, TeammatesPageResponse } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { useSquadContext } from './SquadContext'
import { useSquadPlayerPalette } from './useSquadPlayerPalette'
import { getSquadText, type SquadText } from './i18n'
import { SquadPerMinuteChart } from './SquadPerMinuteChart'
import { squadHasDamagePerMinute } from './charts/squadPerMinuteChart'
import { SquadSynergyRadarChart } from './SquadSynergyRadarChart'
import { SquadPerformanceCharts } from './SquadPerformanceCharts'
import { SquadKillMechanicsChart } from './SquadKillMechanicsChart'
import { SquadFragSection } from './SquadFragSection'
import { MedalDigest } from './MedalDigest'
import { IMPACT_SESSIONS_TEXT } from './impact-sessions/impactSessionsText'
import { SquadImpactSessionsTable } from './impact-sessions/SquadImpactSessionsTable'
import { FeatureGate } from '@/lib/capabilities/FeatureGate'
import { useDataCapability } from '@/lib/capabilities/dataCapabilities'
import { useSquadFilterStore } from '@/stores/squadFilterStore'

/** Référence STABLE : un `?? []` neuf à chaque rendu casserait les mémos du tableau d'impact. */
const NO_SESSIONS: string[] = []

export function SquadContributionsPage() {
  const { confirmedGamertags, pageData, playerSlug } = useSquadContext()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  // Tous les dérivés de cette page sont MÉMOÏSÉS : ils descendent en props dans
  // des ChartCard, qui reconstruisent leur option ECharts (et rejouent donc leur
  // animation d'entrée) dès qu'une prop change d'identité — même si sa valeur
  // est la même. Les `?? []` / `?? {}` écrits à la volée en fabriquaient une
  // neuve à chaque rendu.
  const perMinuteRows = useMemo(() => pageData?.per_minute_stats ?? [], [pageData?.per_minute_stats])
  // « Stats par minute » : les dégâts par minute s'ajoutent au graphe (axe de droite) quand un joueur en
  // porte ; le titre les nomme alors, avec une aide sur le calcul.
  const hasDamagePerMinute = useMemo(() => squadHasDamagePerMinute(perMinuteRows), [perMinuteRows])
  const synergyRadar = useMemo(() => pageData?.synergy_radar ?? [], [pageData?.synergy_radar])
  const performanceSeries = pageData?.performance_series
  const perfSeriesByPlayer = useMemo(() => withoutNullRows(performanceSeries), [performanceSeries])
  // Le backend renvoie s.gamertag (casse mixte ex "Joueur90002") tandis que
  // playerSlug est l'URL param (souvent lowercase). On aligne sur main_player
  // pour que l'ordre des joueurs matche les clés des SquadPerMinuteEntry.player
  // / SquadSynergyRadarSeries.player etc.
  const mainPlayerKey = pageData?.main_player ?? playerSlug
  const palette = useSquadPlayerPalette()
  const playerColors = palette.colorByPlayer
  const playerOrder = useMemo(
    () => [mainPlayerKey, ...confirmedGamertags].filter((p) => performanceSeries?.[p]),
    [mainPlayerKey, confirmedGamertags, performanceSeries],
  )
  // Frags et armes : joueurs ayant des classes de frags OU une série de performance
  // (même prédicat que l'onglet Usages qui les montait jusqu'au lot L2).
  const fragClasses = pageData?.frag_classes
  const fragClassesByPlayer = useMemo(() => withoutNullRows(fragClasses), [fragClasses])
  const fragPlayerOrder = useMemo(
    () => [mainPlayerKey, ...confirmedGamertags].filter((p) => fragClasses?.[p] || performanceSeries?.[p]),
    [mainPlayerKey, confirmedGamertags, fragClasses, performanceSeries],
  )
  const medalDigest = useMemo(() => pageData?.medal_digest ?? [], [pageData?.medal_digest])
  const synergyAxisLabels = useMemo<Record<string, string>>(
    () => ({
      combat: t.synergyRadar.axes.combat,
      survival: t.synergyRadar.axes.survival,
      support: t.synergyRadar.axes.support,
      score: t.synergyRadar.axes.score,
      objective: t.synergyRadar.axes.objective,
      impact: t.synergyRadar.axes.impact,
    }),
    [t],
  )

  return (
    <div className="space-y-4">
      {/* Graphes toujours montés : chaque ChartCard affiche son état vide
          (titre + message) au lieu de disparaître quand sa source est vide. */}
      <div className="grid gap-4 lg:grid-cols-2">
        <SquadPerMinuteChart
          title={
            hasDamagePerMinute ? (
              <span className="flex items-center gap-1.5">
                {t.perMinute.titleWithDamage}
                <InfoTooltip content={t.perMinute.damageInfo} />
              </span>
            ) : (
              t.perMinute.title
            )
          }
          emptyMessage={t.empty.noBlockData}
          rows={perMinuteRows}
          colorByPlayer={playerColors}
          metricLabels={{
            frags: t.perMinute.frags,
            deaths: t.perMinute.deaths,
            assists: t.perMinute.assists,
            dealt: t.perMinute.damageDealt,
            taken: t.perMinute.damageTaken,
          }}
          damageAxisName={t.perMinute.damageAxis}
          perMinuteSuffix={t.perMinute.suffix}
        />

        <SquadSynergyRadarChart
          title={<SynergyRadarTitle t={t} />}
          rows={synergyRadar}
          emptyMessage={t.empty.noBlockData}
          colorByPlayer={playerColors}
          axisLabels={synergyAxisLabels}
          rawLabel={t.synergyRadar.rawLabel}
        />
      </div>

      {/* FRAGS ET ARMES — arrivés d'Usages (lot L2 du plan
          PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) : qui apporte quoi à l'escouade.
          « Répartition des frags » pleine largeur (à côté de « Précision par rôle » sur
          Halo 5), « Outils de destruction » juste en dessous, pleine largeur. Toujours
          montés, comme les autres graphes de la page : chaque carte gère son état vide. */}
      <section className="space-y-3">
        <SectionTitle>{t.sections.fragsArmes}</SectionTitle>
        <SquadFragSection
          fragClassesByPlayer={fragClassesByPlayer}
          weaponTools={pageData?.weapon_tools}
          weaponAccuracy={pageData?.weapon_accuracy}
          playerColors={playerColors}
          playerOrder={fragPlayerOrder}
          locale={locale}
          t={t}
        />
      </section>

      <section className="space-y-3">
        <SectionTitle>{t.performanceCharts.title}</SectionTitle>
        <SquadPerformanceCharts
          emptyMessage={t.empty.noBlockData}
          rowsByPlayer={perfSeriesByPlayer}
          playerOrder={playerOrder}
          colorByPlayer={playerColors}
          labels={t.performanceCharts}
        />
      </section>

      <ImpactMedalsMechanics
        pageData={pageData}
        inkOf={palette.inkOf}
        medalDigest={medalDigest}
        playerColors={playerColors}
        locale={locale}
        t={t}
      />
    </div>
  )
}

/** Le titre du radar synergie, avec l'aide de ses six axes et le lien vers le glossaire. */
function SynergyRadarTitle({ t }: { t: SquadText }) {
  const hp = useEffectiveHpToKill()
  return (
    <span className="flex items-center gap-1.5">
      {t.synergyRadar.title}
      <InfoTooltip
        content={
          <div className="space-y-1">
            <p><span className="font-medium">{t.synergyRadar.axes.impact}</span> — {substituteHpToken(t.synergyRadar.tooltip.impact, hp)}</p>
            <p><span className="font-medium">{t.synergyRadar.axes.combat}</span> — {t.synergyRadar.tooltip.combat}</p>
            <p><span className="font-medium">{t.synergyRadar.axes.survival}</span> — {substituteHpToken(t.synergyRadar.tooltip.survival, hp)}</p>
            <p><span className="font-medium">{t.synergyRadar.axes.support}</span> — {t.synergyRadar.tooltip.support}</p>
            <p><span className="font-medium">{t.synergyRadar.axes.score}</span> — {t.synergyRadar.tooltip.score}</p>
            <p><span className="font-medium">{t.synergyRadar.axes.objective}</span> — {t.synergyRadar.tooltip.objective}</p>
            <Link to="/help" search={{ tab: 'glossary' }} className="block mt-2 text-primary hover:underline">
              {t.synergyRadar.tooltip.glossaryLink}
            </Link>
          </div>
        }
      />
    </span>
  )
}

/**
 * Impact des coéquipiers (un tableau par session), médailles, puis mécaniques de frag (Halo 5). Sorti de
 * la page à la revue L6.1 (fonction ramenée sous sa taille d'avant le chantier).
 */
function ImpactMedalsMechanics({
  pageData,
  inkOf,
  medalDigest,
  playerColors,
  locale,
  t,
}: {
  pageData: TeammatesPageResponse | null
  inkOf: (player: string) => string
  medalDigest: MedalDigestEntry[]
  playerColors: Record<string, string>
  locale: Locale
  t: SquadText
}) {
  // Les sessions sélectionnées dans le filtre de la page s'ouvrent d'office dans le tableau d'impact : la source
  // est le store escouade (source unique des sessions pickées), lu ici par sélecteur et non par le layout.
  const pickedSessions = useSquadFilterStore((s) => s.filterContext.sessions?.picked_sessions) ?? NO_SESSIONS
  // Les rôles d'impact se lisent de la timeline horodatée : un titre qui la déclare garde la section (titre +
  // cadre vide quand la réponse n'a rien à montrer) ; un titre qui ne la déclare pas n'a pas la fonction, la
  // section entière est absente.
  const impactSupported = useDataCapability('match.events.timeline')
  return (
    <>
      {/* IMPACT DES COÉQUIPIERS — arrivé de Synergies (lot 3 « sections », 2026-09-22) :
          c'est une contribution par joueur, pas une production de la composition.
          Titre déclaré par le titre de jeu (match.events.timeline) : section montée avec son état vide géré
          par le tableau (cadre bordé) ; non déclaré : section entière absente. */}
      {impactSupported && (
        <section className="space-y-3">
          <SectionTitle>
            <span className="flex items-center gap-1.5">
              {t.impact.title}
              <InfoTooltip content={IMPACT_SESSIONS_TEXT[locale].info} />
            </span>
          </SectionTitle>
          <SquadImpactSessionsTable
            data={pageData?.squad_impact_sessions}
            pickedSessionLabels={pickedSessions}
            inkOf={inkOf}
            locale={locale}
            impact={t.impact}
            emptyMessage={t.empty.noBlockData}
          />
        </section>
      )}

      {/* MÉDAILLES — arrivées de Synergies avec l'impact. Elles restent EN DERNIER de
          leur groupe (décision utilisateur, 2026-09-13) : c'est un palmarès, pas une
          mesure — il se lit après tout ce qui explique le jeu, jamais avant. */}
      <section className="space-y-3">
        <SectionTitle>{t.medals.title}</SectionTitle>
        <MedalDigest
          entries={medalDigest}
          inkOf={inkOf}
          t={t.medals}
        />
      </section>

      {/* Halo 5 : mécaniques natives par coéquipier (assassinats + compétences
          spartiate). FeatureGate masque hors h5 ; le composant rend null sans données. */}
      <FeatureGate capability="native_kill_mechanics">
        <SquadKillMechanicsChart
          title={t.killMechanics.title}
          emptyMessage={t.empty.noBlockData}
          data={pageData?.native_kill_mechanics}
          colorByPlayer={playerColors}
          labelOf={(m) => t.killMechanics.labels[m as keyof typeof t.killMechanics.labels] ?? m}
        />
      </FeatureGate>
    </>
  )
}
