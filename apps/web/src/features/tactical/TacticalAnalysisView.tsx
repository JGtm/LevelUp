/**
 * TacticalAnalysisView — la vue d'analyse d'UNE carte (Phase 5, items 5.2-5.6).
 *
 * Question / qui / spawn PILOTENT UNE SEULE LECTURE (`useTacticalRaster`), sur le MÊME
 * périmètre que la grille (`matchIds`, `coequipiers` — résolus et passés par
 * `TacticalPage`, jamais refaits ici : une deuxième résolution du périmètre serait une
 * deuxième vérité).
 *
 * L'ÉTAT DES TROIS CONTRÔLES VIT EN LOCAL (`useState`), PAS DANS L'URL : aucun
 * `validateSearch` de la route ne les porte (`tacticalScope.ts` ne porte que `carte`),
 * et les y ajouter pour trois réglages de confort — perdus de toute façon au clic sur
 * une autre carte — aurait touché la route sans nécessité.
 *
 * `?frame=` (lien vers le rejeu depuis une cellule) est REPORTÉ AU LOT D
 * (`playbackStore`, item 5.5 du plan) : `CelluleTactique` ne publie qu'un COMPTE de
 * matchs contributeurs, jamais leurs identifiants — rien à quoi lier tant que ce
 * contrat ne change pas.
 */
import { useMemo, useState } from 'react'

import { KPIStrip, type KPICardData } from '@/components/layout/KPIStrip'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import type { TacticalRaster } from '@/lib/api/types'
import { intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'
import { useTacticalRaster } from './queries'
import { TacticalCellCard } from './TacticalCellCard'
import { TacticalPlanCard } from './TacticalPlanCard'
import { TacticalToolbar } from './TacticalToolbar'
import {
  pageTitle,
  ratioSafe,
  tacticalGridFromRaster,
  trouveCellule,
  type TacticalQuestion,
  type TacticalQui,
} from './tacticalView.logic'

export interface TacticalAnalysisViewProps {
  playerSlug: string
  mapId: string
  /** Nom affichable de la carte : celui que la grille de `TacticalPage` connaît déjà,
   *  ou `mapId` si la grille n'est pas (encore) chargée. */
  mapName: string
  locale: Locale
  t: TacticalText
  /** Périmètre résolu par la barre L2 — même source que la grille. `null` = pas encore résolu. */
  matchIds: string[] | null
  /** Xuids de la composition choisie — même restriction que la grille. */
  coequipiers: string[]
}

export function TacticalAnalysisView({
  playerSlug,
  mapId,
  mapName,
  locale,
  t,
  matchIds,
  coequipiers,
}: TacticalAnalysisViewProps) {
  const [question, setQuestion] = useState<TacticalQuestion>('morts')
  const [qui, setQui] = useState<TacticalQui>('moi')
  const [spawn, setSpawn] = useState('')
  const [selected, setSelected] = useState<{ col: number; row: number } | null>(null)

  const escouadeDisponible = coequipiers.length > 0
  // « Escouade » choisi puis la composition se vide (barre L2) : la lecture retombe sur
  // « Moi » plutôt que d'envoyer un axe que le serveur refuserait en silence.
  const effectiveQui = qui === 'escouade' && !escouadeDisponible ? 'moi' : qui

  const params = useMemo(
    () => ({
      match_ids: matchIds,
      coequipiers,
      question,
      qui: effectiveQui,
      spawn: spawn || undefined,
    }),
    [matchIds, coequipiers, question, effectiveQui, spawn],
  )
  const raster = useTacticalRaster(playerSlug, mapId, params)

  // La cellule affichée n'a plus de sens dès que la lecture change de forme. Ajustée
  // PENDANT LE RENDU (patron React officiel « adjusting state when a prop changes »),
  // pas dans un effet : un effet ferait clignoter l'ancienne cellule un rendu de plus,
  // et déclencherait le lint anti-cascade (`react-hooks/set-state-in-effect`).
  const cleLecture = `${mapId}:${question}:${effectiveQui}:${spawn}`
  const [cleAnterieure, setCleAnterieure] = useState(cleLecture)
  if (cleLecture !== cleAnterieure) {
    setCleAnterieure(cleLecture)
    if (selected !== null) setSelected(null)
  }

  const grid = raster.data
    ? tacticalGridFromRaster(raster.data.cellules ?? [], raster.data.bornes, raster.data.pas_m, raster.data.echelle)
    : null

  const celluleSelectionnee =
    selected && raster.data ? trouveCellule(raster.data.cellules ?? [], selected.col, selected.row) : null

  return (
    <>
      <h2
        className="px-3 pt-3 text-lg font-semibold text-foreground"
        data-testid="tactical-analysis-title"
      >
        {pageTitle(t, mapName, question)}
      </h2>
      <TacticalToolbar
        t={t}
        locale={locale}
        question={question}
        onQuestionChange={setQuestion}
        qui={effectiveQui}
        onQuiChange={setQui}
        escouadeDisponible={escouadeDisponible}
        spawn={spawn}
        onSpawnChange={setSpawn}
        grappes={raster.data?.grappes ?? []}
      />
      {raster.isPending && (
        <div className="flex justify-center p-8" data-testid="tactical-analysis-pending">
          <Spinner label={t.loading} />
        </div>
      )}
      {raster.isError && (
        <div className="p-3">
          <EmptyStateNotice title={t.analysisErrorTitle} description={t.analysisErrorDescription} />
        </div>
      )}
      {!raster.isPending && !raster.isError && raster.data && (
        <div className="flex flex-col gap-3 p-3">
          <KPIStrip cards={buildKpiCards(t, locale, raster.data)} />
          <TacticalPlanCard
            t={t}
            playerSlug={playerSlug}
            mapId={mapId}
            question={question}
            grid={grid}
            bornes={raster.data.bornes}
            pasM={raster.data.pas_m}
            matchsEnAttente={raster.data.matchs_en_attente ?? 0}
            matchsNonCuisables={raster.data.matchs_non_cuisables ?? 0}
            onCellSelect={(col, row) => setSelected({ col, row })}
          />
          <TacticalCellCard t={t} locale={locale} question={question} cellule={celluleSelectionnee} />
        </div>
      )}
    </>
  )
}

/**
 * buildKpiCards — les quatre tuiles du bandeau : matchs retenus, couverture, échange,
 * isolement. Échange et isolement sont OMIS quand le contrat ne les publie pas (question
 * qui ne les mesure pas) — une carte à 0 % mentirait, l'absence de carte ne ment pas.
 */
function buildKpiCards(t: TacticalText, locale: Locale, data: TacticalRaster): KPICardData[] {
  const pct = new Intl.NumberFormat(intlLocale(locale), {
    style: 'percent',
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  })
  const num = new Intl.NumberFormat(intlLocale(locale))

  const cards: KPICardData[] = [
    {
      id: 'tactical-matches-retained',
      label: t.kpiMatchsRetained,
      primary: num.format(data.matchs_retenus),
      secondary: t.kpiSecondary(data.matchs_retenus, data.matchs_filtres),
    },
    {
      id: 'tactical-coverage',
      label: t.kpiCoverage,
      primary: pct.format(ratioSafe(data.matchs_retenus, data.matchs_filtres)),
      secondary: t.kpiSecondary(data.matchs_retenus, data.matchs_filtres),
    },
  ]
  if (data.echange) {
    cards.push({
      id: 'tactical-trade',
      label: t.kpiTrade,
      primary: pct.format(data.echange.taux),
      secondary: t.kpiSecondary(data.echange.brut, data.echange.n),
    })
  }
  if (data.isolement) {
    cards.push({
      id: 'tactical-isolation',
      label: t.kpiIsolation,
      primary: pct.format(data.isolement.taux),
      secondary: t.kpiSecondary(data.isolement.brut, data.isolement.n),
    })
  }
  return cards
}
