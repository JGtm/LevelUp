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
 *
 * QUATRE ÉTATS, UN SEUL CORPS (retours rejeu L2, 2026-09-23 ; `tacticalLecture.logic.ts`) :
 * premier chargement → le cadre et le fond sont posés, l'indicateur par-dessus ; relecture
 * (changement de question, de qui, de spawn ou de filtre) → rien n'est démonté, la réponse
 * PRÉCÉDENTE reste affichée, calque et KPI ESTOMPÉS sous « Mise à jour… » (décision Q26) ;
 * échec → le message, le fond reste. Auparavant `isPending` démontait tout le corps, fond
 * compris, à chaque nouvelle clé de cache.
 */
import { useMemo, useState } from 'react'

import { KPIStrip, type KPICardData } from '@/components/layout/KPIStrip'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { TacticalRaster } from '@/lib/api/types'
import { intlLocale } from '@/lib/formatters'
import { withLowSampleNote } from '@/lib/formatters/lowSampleNote'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'
import { useTacticalCellule, useTacticalRaster } from './queries'
import { TacticalCellCard } from './TacticalCellCard'
import { TacticalCoordinationCard } from './TacticalCoordinationCard'
import { etatLecture, questionServie } from './tacticalLecture.logic'
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
  /** Le périmètre affiché est l'ANCIEN (un nouveau filtre est en cours de résolution) :
   *  la lecture affichée est donc périmée, la vue le dit (« Mise à jour… »). */
  perimetreEnRelecture?: boolean
}

export function TacticalAnalysisView({
  playerSlug,
  mapId,
  mapName,
  locale,
  t,
  matchIds,
  coequipiers,
  perimetreEnRelecture = false,
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
  const { etat, lecture, questionLue } = useLecturePlan(
    playerSlug,
    mapId,
    params,
    question,
    perimetreEnRelecture,
  )

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

  const { celluleSelectionnee, cellule } = useDetailCellule({
    playerSlug,
    mapId,
    selected,
    lecture,
    pret: etat === 'pret',
    params,
  })

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
        grappes={lecture?.grappes ?? []}
      />
      {etat === 'echec' && (
        <div className="p-3">
          <EmptyStateNotice title={t.analysisErrorTitle} description={t.analysisErrorDescription} />
        </div>
      )}
      {/* LE CORPS N'EST JAMAIS DÉMONTÉ : la carte « Plan » est toujours rendue (cf. en-tête). */}
      <div
        className="flex flex-col gap-3 p-3"
        aria-busy={etat === 'attente' || etat === 'relecture'}
        data-testid="tactical-analysis-body"
      >
        {lecture && (
          <KPIStrip
            cards={buildKpiCards(t, locale, lecture, questionLue)}
            className={etat === 'relecture' ? 'opacity-50 transition-opacity' : 'transition-opacity'}
          />
        )}
        <TacticalPlanCard
          t={t}
          locale={locale}
          playerSlug={playerSlug}
          mapId={mapId}
          question={questionLue}
          lecture={lecture}
          etat={etat}
          selected={selected}
          onCellSelect={(col, row) => setSelected({ col, row })}
        />
        {lecture && (
          <TacticalCellCard
            t={t}
            locale={locale}
            playerSlug={playerSlug}
            question={questionLue}
            cellule={celluleSelectionnee}
            contributions={cellule.data?.contributions ?? null}
            contributionsLoading={cellule.isPending && selected !== null}
            matchsNonOuvrables={cellule.data?.matchs_non_ouvrables ?? 0}
          />
        )}
        {lecture?.coordination && (
          <TacticalCoordinationCard
            t={t}
            locale={locale}
            coordination={lecture.coordination}
            echange={lecture.echange ?? null}
            isolement={lecture.isolement ?? null}
            matchsFiltres={lecture.matchs_filtres}
          />
        )}
      </div>
    </>
  )
}

type ParamsLecture = Parameters<typeof useTacticalRaster>[2]

/**
 * useLecturePlan — la lecture du plan, son ÉTAT (`etatLecture`) et la question À LAQUELLE
 * elle répond.
 *
 * LA LECTURE AFFICHÉE est la réponse courante, ou la PRÉCÉDENTE pendant une relecture
 * (`placeholderData`) ; une lecture en échec n'affiche rien de périmé comme courant. Unité,
 * légende et source se calculent sur `questionLue` : pendant une relecture, c'est encore
 * l'ancienne question.
 */
function useLecturePlan(
  playerSlug: string,
  mapId: string,
  params: ParamsLecture,
  question: TacticalQuestion,
  perimetreEnRelecture: boolean,
) {
  const raster = useTacticalRaster(playerSlug, mapId, params)
  const etat = etatLecture({
    aDesDonnees: raster.data !== undefined,
    enEchec: raster.isError,
    surPlaceholder: raster.isPlaceholderData,
    perimetreEnRelecture,
  })
  const lecture = etat === 'echec' ? undefined : raster.data
  const questionLue = lecture ? questionServie(lecture.question, question) : question
  return { etat, lecture, questionLue }
}

/**
 * useDetailCellule — la cellule choisie et son DÉTAIL (lot M1) : MÊME périmètre + question +
 * qui + spawn que le raster, plus l'adresse cliquée. `selected` à `null` → la requête n'est
 * pas lancée (cf. `useTacticalCellule`), ce qui est l'état NORMAL avant tout clic. Elle
 * attend aussi la fin d'une relecture (`pret`) : le `pas_m` d'une réponse PRÉCÉDENTE
 * n'adresse pas forcément la même cellule dans la nouvelle.
 */
function useDetailCellule({
  playerSlug,
  mapId,
  selected,
  lecture,
  pret,
  params,
}: {
  playerSlug: string
  mapId: string
  selected: { col: number; row: number } | null
  lecture: TacticalRaster | undefined
  pret: boolean
  params: ParamsLecture
}) {
  const celluleSelectionnee =
    selected && lecture ? trouveCellule(lecture.cellules ?? [], selected.col, selected.row) : null
  const cellule = useTacticalCellule(
    playerSlug,
    mapId,
    selected && lecture && pret
      ? { col: selected.col, lig: selected.row, pas_m: lecture.pas_m }
      : null,
    params,
  )
  return { celluleSelectionnee, cellule }
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
 *
 * `question` est celle À LAQUELLE `data` RÉPOND (`questionServie`), pas la question demandée.
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

