/**
 * TacticalAnalysisView — le centre et la droite du cockpit : la carte du plan de la carte affichée
 * et, à sa droite, le détail de la zone choisie.
 *
 * Lecture / joueurs / réapparition PILOTENT UNE SEULE LECTURE (`useTacticalRaster`), sur le MÊME
 * périmètre que la liste des cartes (`matchIds`, `coequipiers` — résolus et passés par
 * `TacticalPage`, jamais refaits ici : une deuxième résolution du périmètre serait une deuxième
 * vérité). La lecture ne part que pour une carte LUE (`carteLue`) : une carte d'URL hors du filtre
 * ou pas encore jugée n'est jamais lue.
 *
 * L'ÉTAT DES TROIS RÉGLAGES VIT EN LOCAL (`useState`), PAS DANS L'URL : `tacticalScope.ts` ne porte
 * que `carte`, et les y ajouter pour trois réglages de confort — perdus de toute façon au changement
 * de carte — aurait touché la route sans nécessité.
 *
 * LA CARTE DU PLAN EST TOUJOURS RENDUE (retours rejeu L2, 2026-09-23) : ce qui se pose sur le fond
 * dépend de l'état du plan (`etatDuPlan`) — premier chargement, relecture estompée sous « Mise à
 * jour… », échec (lecture, périmètre ou composition impossible), carte hors du filtre.
 *
 * Le détail d'une zone (`useTacticalCellule`) ne part QUE quand une cellule est sélectionnée.
 */
import { useMemo, useState } from 'react'

import type { TacticalRaster } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { carteLue, type CarteEffective } from './cockpit.logic'
import type { TacticalText } from './i18n'
import { etatDuPlan } from './plan.logic'
import { useTacticalCellule, useTacticalRaster } from './queries'
import { TacticalCellCard } from './TacticalCellCard'
import { classeRelecture, etatLecture, questionServie } from './tacticalLecture.logic'
import { TacticalPlanCard, type ReglagesDuPlan } from './TacticalPlanCard'
import { trouveCellule, type TacticalQuestion, type TacticalQui } from './tacticalView.logic'

export interface TacticalAnalysisViewProps {
  playerSlug: string
  /** La carte affichée et d'où elle vient (`carteEffective`) ; `mapId` vide tant qu'aucune n'est connue. */
  carte: CarteEffective
  /** Nom affichable de la carte (celui que la liste des cartes connaît), '' sans carte. */
  mapName: string
  locale: Locale
  t: TacticalText
  /** Périmètre résolu par la barre L2 — même source que la liste des cartes. `null` = pas encore résolu. */
  matchIds: string[] | null
  /** Xuids de la composition choisie — même restriction que la liste des cartes. */
  coequipiers: string[]
  /** Le périmètre affiché est l'ANCIEN (un nouveau filtre est en cours de résolution). */
  perimetreEnRelecture?: boolean
  /** La résolution du périmètre a ÉCHOUÉ : la lecture, suspendue, ne répondra pas. */
  perimetreEnEchec?: boolean
  /** Les coéquipiers INTROUVABLES quand la composition est impossible, `null` sinon. */
  coequipiersInconnus?: string[] | null
}

export function TacticalAnalysisView({
  playerSlug,
  carte,
  mapName,
  locale,
  t,
  matchIds,
  coequipiers,
  perimetreEnRelecture = false,
  perimetreEnEchec = false,
  coequipiersInconnus = null,
}: TacticalAnalysisViewProps) {
  const reglages = useReglages(coequipiers)
  const { question, qui, spawn } = reglages

  const params = useMemo(
    () => ({
      match_ids: carteLue(carte) ? matchIds : null,
      coequipiers,
      question,
      qui,
      spawn: spawn || undefined,
    }),
    [carte, matchIds, coequipiers, question, qui, spawn],
  )
  const { etat, lecture, questionLue } = useLecturePlan(playerSlug, carte.mapId, params, question, {
    enRelecture: perimetreEnRelecture,
    enEchec: perimetreEnEchec || coequipiersInconnus !== null,
  })
  const etatPlan = etatDuPlan(carte.origine, etat)
  const [selected, setSelected] = useZoneChoisie(`${carte.mapId}:${question}:${qui}:${spawn}`)

  const { celluleSelectionnee, cellule } = useDetailCellule({
    playerSlug,
    mapId: carte.mapId,
    selected,
    lecture,
    pret: etatPlan === 'pret',
    params,
  })
  const lectureAffichee = etatPlan === 'pret' || etatPlan === 'relecture' ? lecture : undefined

  return (
    <div
      className="grid min-w-0 grid-cols-1 items-start gap-3 lg:grid-cols-[minmax(0,1fr)_var(--tac-zone-l)]"
      aria-busy={etatPlan === 'attente' || etatPlan === 'relecture'}
      data-testid="tactical-analysis-body"
    >
      <TacticalPlanCard
        t={t}
        locale={locale}
        playerSlug={playerSlug}
        mapId={carte.mapId}
        titre={mapName}
        question={questionLue}
        lecture={lecture}
        etat={etatPlan}
        inconnus={coequipiersInconnus}
        reglages={{ ...reglages, grappes: lectureAffichee?.grappes ?? [] }}
        selected={selected}
        onCellSelect={(col, row) => setSelected({ col, row })}
      />
      {lectureAffichee && (
        // TRANSITOIRE : la carte de la zone choisie (lot L6 du plan Tactique v2 la remplace).
        <div className={classeRelecture(etatPlan === 'relecture')}>
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
        </div>
      )}
    </div>
  )
}

/**
 * useReglages — l'état des trois réglages du bandeau. « Escouade » choisi puis la composition se
 * vide (barre L2) : la lecture retombe sur « Moi » plutôt que d'envoyer un axe que le serveur
 * refuserait en silence.
 */
function useReglages(coequipiers: readonly string[]): Omit<ReglagesDuPlan, 'grappes'> {
  const [question, setQuestion] = useState<TacticalQuestion>('morts')
  const [qui, setQui] = useState<TacticalQui>('moi')
  const [spawn, setSpawn] = useState('')
  const escouadeDisponible = coequipiers.length > 0
  return {
    question,
    onQuestionChange: setQuestion,
    qui: qui === 'escouade' && !escouadeDisponible ? 'moi' : qui,
    onQuiChange: setQui,
    escouadeDisponible,
    spawn,
    onSpawnChange: setSpawn,
  }
}

/**
 * useZoneChoisie — la cellule choisie sur le plan, remise à zéro quand la lecture change de forme
 * (`cle`). Ajustée PENDANT LE RENDU (patron React « adjusting state when a prop changes »), pas dans
 * un effet : un effet ferait clignoter l'ancienne cellule un rendu de plus.
 */
function useZoneChoisie(cle: string) {
  const [selected, setSelected] = useState<{ col: number; row: number } | null>(null)
  const [cleAnterieure, setCleAnterieure] = useState(cle)
  if (cle !== cleAnterieure) {
    setCleAnterieure(cle)
    if (selected !== null) setSelected(null)
  }
  return [selected, setSelected] as const
}

type ParamsLecture = Parameters<typeof useTacticalRaster>[2]

/**
 * useLecturePlan — la lecture du plan, son ÉTAT (`etatLecture`) et la question À LAQUELLE elle
 * répond.
 *
 * LA LECTURE AFFICHÉE est la réponse courante, ou la PRÉCÉDENTE pendant une relecture
 * (`placeholderData`) ; une lecture en échec — la sienne ou celle de son PÉRIMÈTRE — n'affiche rien
 * de périmé comme courant. Unité, légende et source se calculent sur `questionLue`.
 */
function useLecturePlan(
  playerSlug: string,
  mapId: string,
  params: ParamsLecture,
  question: TacticalQuestion,
  perimetre: { enRelecture: boolean; enEchec: boolean },
) {
  const raster = useTacticalRaster(playerSlug, mapId, params)
  const etat = etatLecture({
    aDesDonnees: raster.data !== undefined,
    enEchec: raster.isError || perimetre.enEchec,
    surPlaceholder: raster.isPlaceholderData,
    perimetreEnRelecture: perimetre.enRelecture,
  })
  const lecture = etat === 'echec' ? undefined : raster.data
  const questionLue = lecture ? questionServie(lecture.question, question) : question
  return { etat, lecture, questionLue }
}

/**
 * useDetailCellule — la cellule choisie et son DÉTAIL : MÊME périmètre + lecture + joueurs +
 * réapparition que le raster, plus l'adresse cliquée. `selected` à `null` → la requête n'est pas
 * lancée. Elle attend aussi la fin d'une relecture (`pret`) : le `pas_m` d'une réponse PRÉCÉDENTE
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
    selected && lecture && pret ? { col: selected.col, lig: selected.row, pas_m: lecture.pas_m } : null,
    params,
  )
  return { celluleSelectionnee, cellule }
}
