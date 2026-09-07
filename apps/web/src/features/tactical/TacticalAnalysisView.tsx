/**
 * TacticalAnalysisView.tsx — Vue d'analyse tactique pour une carte.
 *
 * Affiche : barre d'outils, KPI, plan, cellule sélectionnée, coordination.
 * États : loading, error, empty, pending, unavailable.
 */
import { useCallback, useMemo, useState } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { getTacticalText, type TacticalLocale } from './i18n'
import { useTacticalRaster } from './queries'
import { TacticalKPIStrip } from './TacticalKPIStrip'
import { TacticalPlanCard } from './TacticalPlanCard'
import { TacticalCellCard } from './TacticalCellCard'
import {
  generateProcessingMessage,
  validateQuestion,
  type QuestionType,
  type WhoType,
  type SpawnType,
} from './tacticalView.logic'

export interface TacticalAnalysisViewProps {
  playerSlug: string
  titleSlug: string
  mapId: string
  mapName: string
  filterHash: string
  grappes?: Array<{ id: string; nom_fr: string; nom_en: string }>
}

export function TacticalAnalysisView({
  playerSlug,
  titleSlug,
  mapId,
  mapName,
  filterHash,
  grappes = [],
}: TacticalAnalysisViewProps) {
  const locale = useAppShellStore((s) => s.locale) as TacticalLocale
  const text = getTacticalText(locale)

  // État local
  const [question, setQuestion] = useState<QuestionType>('morts')
  const [who, setWho] = useState<WhoType>('moi')
  const [spawn, setSpawn] = useState<SpawnType>('tous')
  const [selectedCell, setSelectedCell] = useState<{
    col: number
    row: number
    value: number
    matchCount: number
  } | null>(null)

  // Charger le raster
  const { data: rasterData, isLoading, isError } = useTacticalRaster(
    playerSlug,
    titleSlug,
    mapId,
    filterHash,
    question,
    who,
    spawn,
    true, // enabled
  )

  // Déterminer les états
  const isEmpty = useMemo(() => {
    return !isLoading && !isError && (!rasterData || rasterData.cells.length === 0)
  }, [isLoading, isError, rasterData])

  const { status: processingStatus, message: processingMessage } = useMemo(() => {
    if (!rasterData) return { status: 'idle' as const, message: '' }
    return generateProcessingMessage(rasterData.matchs_en_attente, rasterData.matchs_non_cuisables, locale)
  }, [rasterData, locale])

  const showPending = useMemo(() => {
    return processingStatus === 'processing' || processingStatus === 'unavailable'
  }, [processingStatus])

  // Gestionnaires
  const handleQuestionChange = useCallback((e: React.ChangeEvent<HTMLSelectElement>) => {
    if (validateQuestion(e.target.value)) {
      setQuestion(e.target.value)
      setSelectedCell(null)
    }
  }, [])

  const handleWhoChange = useCallback((newWho: WhoType) => {
    setWho(newWho)
    setSelectedCell(null)
  }, [])

  const handleSpawnChange = useCallback((newSpawn: SpawnType) => {
    setSpawn(newSpawn)
    setSelectedCell(null)
  }, [])

  const handleCellSelect = useCallback(
    (col: number, row: number, value: number, matchCount: number) => {
      setSelectedCell({ col, row, value, matchCount })
    },
    [],
  )

  // Générer les options de spawn
  const spawnOptions = useMemo(() => {
    return [
      { value: 'tous', label: locale === 'fr' ? 'Tous' : 'All' },
      ...grappes.map((g) => ({
        value: g.id,
        label: locale === 'fr' ? g.nom_fr : g.nom_en,
      })),
    ]
  }, [grappes, locale])

  // Données pour les cartes de section
  const kpiData = useMemo(() => {
    if (!rasterData) {
      return {
        matchsRetained: 0,
        matchsFiltered: 0,
        coverage: 0,
        tradeRate: 0,
        isolatedDeathRate: 0,
        source: 'base' as const,
      }
    }

    return {
      matchsRetained: rasterData.matchs_retenus,
      matchsFiltered: rasterData.matchs_filtres,
      coverage: (rasterData.matchs_retenus / Math.max(rasterData.matchs_filtres, 1)) * 100,
      tradeRate: rasterData.echange * 100,
      isolatedDeathRate: rasterData.isolation ? Object.values(rasterData.isolation)[0]?.taux ?? 0 : 0,
      source: 'base' as const,
    }
  }, [rasterData])

  // Samples pour la cellule (fixtures pour la phase 5)
  const cellSamples = useMemo(() => {
    if (!selectedCell) return []
    // Placeholder - en phase 6, cela viendra du backend
    return [
      {
        date: '12/08',
        outcome: 'défaite' as const,
        time: '4:12',
        frame: undefined,
      },
      {
        date: '09/08',
        outcome: 'victoire' as const,
        time: '7:38',
        frame: undefined,
      },
      {
        date: '02/08',
        outcome: 'défaite' as const,
        time: '2:55',
        frame: undefined,
      },
    ]
  }, [selectedCell])

  return (
    <div className="flex flex-col gap-4">
      {/* KPI Strip */}
      <TacticalKPIStrip
        locale={locale}
        matchsRetained={kpiData.matchsRetained}
        matchsFiltered={kpiData.matchsFiltered}
        coverage={kpiData.coverage}
        tradeRate={kpiData.tradeRate}
        isolatedDeathRate={kpiData.isolatedDeathRate}
        source={kpiData.source}
        isLoading={isLoading}
      />

      {/* Message de traitement */}
      {showPending && (
        <div className="rounded-md border border-border bg-card/50 px-4 py-3 text-sm text-muted-foreground">
          {processingMessage}
        </div>
      )}

      {/* Toolbar */}
      <div className="flex flex-col gap-4 sm:flex-row sm:flex-wrap sm:items-end">
        <div className="flex flex-col gap-1">
          <label htmlFor="question-select" className="text-xs font-semibold uppercase text-muted-foreground">
            {text.questionWhere}
          </label>
          <select
            id="question-select"
            value={question}
            onChange={handleQuestionChange}
            className="rounded-md border border-input bg-background px-3 py-2 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-ring"
          >
            <option value="morts">{text.questionDeaths}</option>
            <option value="kills">{text.questionKills}</option>
            <option value="gagne">{text.questionWin}</option>
            <option value="temps">{text.questionTime}</option>
            <option value="routes">{text.questionRoutes}</option>
            <option value="isole">{text.questionIsolated}</option>
          </select>
        </div>

        <div className="flex flex-col gap-1">
          <label htmlFor="who-group" className="text-xs font-semibold uppercase text-muted-foreground">
            {text.whoMe === 'Moi' ? 'Qui' : 'Who'}
          </label>
          <div id="who-group" className="inline-flex gap-1 rounded-md border border-input bg-background p-1">
            {(['moi', 'escouade', 'adv'] as const).map((w) => (
              <button
                key={w}
                onClick={() => handleWhoChange(w)}
                aria-pressed={who === w}
                className={`rounded-sm px-3 py-1 text-sm font-medium transition-colors ${
                  who === w
                    ? 'bg-secondary text-foreground'
                    : 'bg-transparent text-muted-foreground hover:text-foreground'
                }`}
              >
                {w === 'moi'
                  ? text.whoMe
                  : w === 'escouade'
                    ? text.whoSquad
                    : text.whoEnemies}
              </button>
            ))}
          </div>
        </div>

        <div className="flex flex-col gap-1">
          <label htmlFor="spawn-group" className="text-xs font-semibold uppercase text-muted-foreground">
            {text.spawnLabel}
          </label>
          <div id="spawn-group" className="inline-flex gap-1 rounded-md border border-input bg-background p-1">
            {spawnOptions.map((opt) => (
              <button
                key={opt.value}
                onClick={() => handleSpawnChange(opt.value as SpawnType)}
                aria-pressed={spawn === opt.value}
                className={`rounded-sm px-3 py-1 text-sm font-medium transition-colors ${
                  spawn === opt.value
                    ? 'bg-secondary text-foreground'
                    : 'bg-transparent text-muted-foreground hover:text-foreground'
                }`}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Main content: Plan + Cell Card */}
      <div className="grid gap-4 lg:grid-cols-[1.72fr_1fr]">
        {/* Plan Card */}
        <TacticalPlanCard
          locale={locale}
          mapName={mapName}
          question={question}
          rasterData={rasterData}
          isLoading={isLoading}
          isError={isError}
          isEmpty={isEmpty}
          onCellSelect={handleCellSelect}
          selectedCell={selectedCell ? { col: selectedCell.col, row: selectedCell.row } : null}
        />

        {/* Right Column */}
        <div className="flex flex-col gap-4">
          {/* Cell Card */}
          <TacticalCellCard
            locale={locale}
            question={question}
            selectedCell={
              selectedCell
                ? {
                    value: selectedCell.value,
                    matchCount: selectedCell.matchCount,
                    samples: cellSamples,
                  }
                : null
            }
          />

          {/* Coordination Card (placeholder) */}
          <div className="rounded-md border border-border bg-card">
            <h3 className="border-b border-border px-3 py-2 text-sm font-medium">{text.coordinationTitle}</h3>
            <div className="p-3 text-xs text-muted-foreground">
              <p>{locale === 'fr' ? 'À venir en phase 7' : 'Coming in phase 7'}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
