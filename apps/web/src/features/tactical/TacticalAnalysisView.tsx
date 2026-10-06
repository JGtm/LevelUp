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
 * LA ZONE LA PLUS CHAUDE EST PRÉSÉLECTIONNÉE (`useZone`) tant que l'utilisateur n'a rien choisi ; son
 * détail (`useTacticalCellule`) part quand la lecture est prête, et nomme la zone sur le plan.
 */
import { useMemo, useState } from 'react'

import type { TacticalRaster } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { carteLue, type CarteEffective } from './cockpit.logic'
import type { TacticalText } from './i18n'
import { etatDuPlan } from './plan.logic'
import { useTacticalCellule, useTacticalRaster } from './queries'
import { TacticalZoneCard } from './TacticalZoneCard'
import { classeRelecture, etatLecture, questionServie } from './tacticalLecture.logic'
import { TacticalPlanCard, type ReglagesDuPlan } from './TacticalPlanCard'
import { trouveCellule, type TacticalQuestion, type TacticalQui } from './tacticalView.logic'
import { titreDeZone, zoneLaPlusChaude } from './zone.logic'

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
  const lectureAffichee = etatPlan === 'pret' || etatPlan === 'relecture' ? lecture : undefined
  const zone = useZone({
    playerSlug,
    mapId: carte.mapId,
    cle: `${carte.mapId}:${question}:${qui}:${spawn}`,
    lecture: lectureAffichee,
    pret: etatPlan === 'pret',
    params,
  })

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
        etiquette={zone.cellule && zone.detail.data ? titreDeZone(t, locale, true, zone.detail.data) : null}
        reglages={{ ...reglages, grappes: lectureAffichee?.grappes ?? [] }}
        selected={zone.selected}
        onCellSelect={zone.choisir}
      />
      {/* LA COLONNE PREND LA HAUTEUR DE LA CARTE DU PLAN (trois colonnes) : `contain: size` l'empêche
          de peser sur la rangée, l'étirement lui donne la hauteur du plan, et sa liste défile. */}
      <div className={`min-h-0 lg:self-stretch lg:[contain:size] ${classeRelecture(etatPlan === 'relecture')}`}>
        <TacticalZoneCard
          t={t}
          locale={locale}
          playerSlug={playerSlug}
          question={questionLue}
          signee={lectureAffichee?.echelle.symetrique ?? false}
          pasM={lectureAffichee?.pas_m ?? 0}
          cellule={zone.cellule}
          detail={zone.detail}
        />
      </div>
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
 * useZone — la zone affichée par la colonne de droite et son DÉTAIL.
 *
 * LA ZONE LA PLUS CHAUDE EST PRÉSÉLECTIONNÉE (D10) tant que l'utilisateur n'a rien choisi ; son choix
 * est remis à zéro quand la lecture change de forme (`cle` : carte, lecture, joueurs, réapparition).
 * Le détail (`useTacticalCellule`) porte le MÊME périmètre que le raster, plus l'adresse ; il attend
 * la fin d'une relecture (`pret`) : le `pas_m` d'une réponse PRÉCÉDENTE n'adresse pas forcément la
 * même cellule dans la nouvelle.
 */
function useZone({
  playerSlug,
  mapId,
  cle,
  lecture,
  pret,
  params,
}: {
  playerSlug: string
  mapId: string
  cle: string
  /** La lecture AFFICHÉE (courante ou en relecture), `undefined` sinon. */
  lecture: TacticalRaster | undefined
  pret: boolean
  params: ParamsLecture
}) {
  const [choix, setChoix] = useZoneChoisie(cle)
  const chaude = lecture ? zoneLaPlusChaude(lecture.cellules ?? [], lecture.echelle.symetrique) : null
  const selected = choix ?? (chaude ? { col: chaude.col, row: chaude.lig } : null)
  const cellule = selected && lecture ? trouveCellule(lecture.cellules ?? [], selected.col, selected.row) : null
  const requete = useTacticalCellule(
    playerSlug,
    mapId,
    selected && lecture && pret ? { col: selected.col, lig: selected.row, pas_m: lecture.pas_m } : null,
    params,
  )
  return {
    selected,
    choisir: (col: number, row: number) => setChoix({ col, row }),
    cellule,
    detail: { data: requete.data, isPending: requete.isPending && selected !== null },
  }
}
