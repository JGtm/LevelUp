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
 * `?frame=` (lien vers le rejeu depuis une cellule) est servi par `TacticalCellCard` depuis
 * le lot M1 (Tactique S.1) : `useTacticalCellule` résout les contributions {match_id,
 * instant_ms, xuid} de la cellule sélectionnée, avec le MÊME périmètre que le raster —
 * la requête ne part QUE quand une cellule est sélectionnée (`selected !== null`).
 */
import { useMemo, useState } from 'react'

import { KPIStrip, type KPICardData } from '@/components/layout/KPIStrip'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import type { TacticalRaster } from '@/lib/api/types'
import { intlLocale } from '@/lib/formatters'
import { withLowSampleNote } from '@/lib/formatters/lowSampleNote'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'
import { useTacticalCellule, useTacticalRaster } from './queries'
import { TacticalCellCard } from './TacticalCellCard'
import { TacticalCoordinationCard } from './TacticalCoordinationCard'
import { TacticalPlanCard } from './TacticalPlanCard'
import { TacticalToolbar } from './TacticalToolbar'
import {
  libelleRayons,
  pageTitle,
  ratioSafe,
  sourceForQuestion,
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

  const celluleSelectionnee =
    selected && raster.data ? trouveCellule(raster.data.cellules ?? [], selected.col, selected.row) : null

  // LE DÉTAIL D'UNE CELLULE (lot M1) : MÊME périmètre + question + qui + spawn que le
  // raster, plus l'adresse cliquée. `selected` à `null` → la requête n'est pas lancée
  // (cf. `useTacticalCellule`), ce qui est l'état NORMAL avant tout clic.
  const cellule = useTacticalCellule(
    playerSlug,
    mapId,
    selected && raster.data
      ? { col: selected.col, lig: selected.row, pas_m: raster.data.pas_m }
      : null,
    params,
  )

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
          <KPIStrip cards={buildKpiCards(t, locale, raster.data, question)} />
          <TacticalPlanCard
            t={t}
            locale={locale}
            playerSlug={playerSlug}
            mapId={mapId}
            question={question}
            cellules={raster.data.cellules ?? []}
            bornes={raster.data.bornes}
            echelle={raster.data.echelle}
            pasM={raster.data.pas_m}
            matchsFiltres={raster.data.matchs_filtres}
            matchsRetenus={raster.data.matchs_retenus}
            matchsEnAttente={raster.data.matchs_en_attente ?? 0}
            matchsNonCuisables={raster.data.matchs_non_cuisables ?? 0}
            selected={selected}
            onCellSelect={(col, row) => setSelected({ col, row })}
          />
          <TacticalCellCard
            t={t}
            locale={locale}
            playerSlug={playerSlug}
            question={question}
            cellule={celluleSelectionnee}
            contributions={cellule.data?.contributions ?? null}
            contributionsLoading={cellule.isPending && selected !== null}
            matchsNonOuvrables={cellule.data?.matchs_non_ouvrables ?? 0}
          />
          {raster.data.coordination && (
            <TacticalCoordinationCard
              t={t}
              locale={locale}
              coordination={raster.data.coordination}
              echange={raster.data.echange ?? null}
              isolement={raster.data.isolement ?? null}
              matchsFiltres={raster.data.matchs_filtres}
            />
          )}
        </div>
      )}
    </>
  )
}

/**
 * buildKpiCards — les QUATRE tuiles du bandeau de la maquette 034b1915 : matchs retenus,
 * couverture, morts en isolement, échange après ma mort.
 *
 * CHAQUE SOUS-TITRE DIT LA RÈGLE, PAS UN COMPTE DE PLUS (c'est la différence avec la
 * version précédente, où « 38 sur 56 » était répété sous deux tuiles) : d'où viennent les
 * matchs (rejeux cuits / base partagée), la fenêtre de l'échange (5 s, publiée par le
 * serveur), le rayon de l'isolement (la portée du radar mesurée par variante).
 *
 * ÉCHANGE ET ISOLEMENT SONT OMIS QUAND LE CONTRAT NE LES PUBLIE PAS (titre qui ne sait pas
 * lire la source des morts) — une tuile à 0 % mentirait, l'absence de tuile ne ment pas.
 */
function buildKpiCards(
  t: TacticalText,
  locale: Locale,
  data: TacticalRaster,
  question: TacticalQuestion,
): KPICardData[] {
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
      secondary: t.kpiMatchsRetainedSecondary(data.matchs_filtres),
    },
    {
      id: 'tactical-coverage',
      label: t.kpiCoverage,
      primary: pct.format(ratioSafe(data.matchs_retenus, data.matchs_filtres)),
      // LA PROVENANCE, pas le rapport déjà lu sur la tuile voisine : une couverture de
      // 68 % ne se lit pas pareil selon qu'elle dépend d'une cuisson (elle montera) ou de
      // la base partagée (elle ne montera pas).
      secondary: sourceForQuestion(t, question) === t.sourceReplay
        ? t.kpiCoverageReplay
        : t.kpiCoverageShared,
    },
  ]
  if (data.isolement) {
    const sansRayon = data.matchs_sans_rayon ?? 0
    const rayons = data.coordination?.rayons_m ?? []
    cards.push({
      id: 'tactical-isolation',
      label: t.kpiIsolation,
      primary: pct.format(data.isolement.taux),
      secondary: withLowSampleNote(
        rayons.length > 0
          ? t.kpiIsolationRadius(libelleRayons(t, rayons, locale))
          : t.kpiSecondary(data.isolement.brut, data.isolement.n),
        data.isolement.echantillon_faible,
        t.lowSample,
      ),
      // ▼ : moins on meurt isolé, mieux c'est. LA FLÈCHE DIT LE SENS SOUHAITABLE DE LA
      // GRANDEUR, PAS UNE VARIATION — d'où le mot à côté, et d'où le refus du `trend` de
      // `KPIStrip`, qui se compare à une référence que cet onglet ne sert pas. La maquette
      // 034b1915 posait ▼ seul ; seul, il se lit comme « en baisse ».
      custom: (
        <div className="flex flex-col gap-0.5">
          <span className="text-2xs text-muted-foreground">{t.kpiLowerIsBetter}</span>
          {sansRayon > 0 && (
            <span className="text-2xs text-muted-foreground" data-testid="tactical-isolation-no-radius">
              {t.kpiNoRadiusNote(sansRayon)}
            </span>
          )}
        </div>
      ),
    })
  }
  if (data.echange) {
    cards.push({
      id: 'tactical-trade',
      label: t.kpiRiposte,
      primary: pct.format(data.echange.taux),
      // LA FENÊTRE EST PUBLIÉE PAR LE SERVEUR, jamais recopiée ici : elle divergerait du
      // calcul au premier ajustement. Absente (réponse d'une version antérieure), on
      // retombe sur le compte brut — « sous 0 s » aurait été un chiffre FAUX, pas une
      // valeur manquante.
      secondary: withLowSampleNote(
        data.coordination && data.coordination.fenetre_echange_secondes > 0
          ? t.kpiRiposteWindow(data.coordination.fenetre_echange_secondes)
          : t.kpiSecondary(data.echange.brut, data.echange.n),
        data.echange.echantillon_faible,
        t.lowSample,
      ),
    })
  }
  return cards
}

