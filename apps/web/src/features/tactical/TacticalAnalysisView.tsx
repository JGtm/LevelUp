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
 * détail (`useTacticalCellule`) part quand la lecture est prête, et nomme la zone sur le plan. Elle
 * s'ÉCARTE par un re-clic sur elle, un clic loin de toute cellule, ou Échap (sans mort choisie) ; elle
 * ne revient alors qu'au changement de carte, de lecture, de joueurs ou de réapparition.
 */
import { useCallback, useEffect, useMemo, useState } from 'react'

import type { TacticalContribution, TacticalRaster } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { carteLue, type CarteEffective } from './cockpit.logic'
import type { TacticalText } from './i18n'
import { etatDuPlan } from './plan.logic'
import { usePrechargementDesLectures, useTacticalCellule, useTacticalRaster } from './queries'
import { TacticalZoneCard } from './TacticalZoneCard'
import { classeRelecture, etatLecture, questionServie } from './tacticalLecture.logic'
import { TacticalPlanCard, type ReglagesDuPlan } from './TacticalPlanCard'
import { basculerChoix } from './tueurs.logic'
import { trouveCellule, type TacticalQuestion, type TacticalQui } from './tacticalView.logic'
import { useVueDeMort } from './useVueDeMort'
import {
  choixApresClic,
  choixDuClic,
  titreDeZone,
  zoneAffichee,
  ZONE_ECARTEE,
  zoneLaPlusChaude,
  type ChoixDeZone,
} from './zone.logic'

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
  /**
   * L'angle « Escouade » est demandé sans composition (`true`) ou quitté pour un autre angle
   * (`false`) : la page amène au sélecteur de composition de la barre.
   */
  onDemandeComposition?: (enCours: boolean) => void
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
  onDemandeComposition,
}: TacticalAnalysisViewProps) {
  const reglages = useReglages(coequipiers, onDemandeComposition)
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
  // L'INTENTION DE CHANGER DE LECTURE (pilule survolée ou prise au clavier) précharge les lectures
  // que le cache n'a pas : choisir l'une d'elles ensuite est instantané.
  const precharger = usePrechargementDesLectures(
    playerSlug,
    carte.mapId,
    params,
    t.analysisQuestions.map((q) => q.id),
  )
  const zone = useZone({
    playerSlug,
    mapId: carte.mapId,
    cle: `${carte.mapId}:${question}:${qui}:${spawn}`,
    lecture: lectureAffichee,
    pret: etatPlan === 'pret',
    params,
  })
  // La contribution mise en avant, partagée par la tuile et le plan ; remise à zéro avec la zone.
  const { survol, setSurvol, choisie, choisir } = useContributionChoisie(zone.cle)
  useEchapEcarteLaZone(zone.selected !== null && choisie === null, zone.ecarter)
  const { vue, etatSequence } = useVueDeMort(playerSlug, carte.mapId, zone.contributions, choisie)

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
        etiquette={zone.cellule && zone.detail.data ? titreDeZone(t, true, zone.detail.data) : null}
        reglages={{
          ...reglages,
          onQuestionIntent: etatPlan === 'pret' ? precharger : undefined,
          grappes: lectureAffichee?.grappes ?? [],
        }}
        selected={zone.selected}
        onPointSelect={zone.choisir}
        contributions={zone.contributions}
        survol={survol}
        onSurvol={setSurvol}
        choisie={choisie}
        onChoisir={choisir}
        vue={vue}
        etatSequence={etatSequence}
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
          survol={survol}
          onSurvol={setSurvol}
          choisie={choisie}
          onChoisir={choisir}
          bilan={zone.detail.data?.bilan}
          vue={vue}
        />
      </div>
    </div>
  )
}

/**
 * useReglages — l'état des trois réglages du bandeau.
 *
 * « Escouade » SANS COMPOSITION (rien de choisi, ou composition vidée dans la barre) : la lecture
 * reste sur « Moi » plutôt que d'envoyer un axe que le serveur refuse (arbitrage du 2026-09-06 :
 * l'angle Escouade est la composition choisie, jamais les coéquipiers de chaque match). Le clic est
 * retenu et signalé (`onDemandeComposition`) : la page ouvre le sélecteur, et l'angle s'applique dès
 * qu'une composition est choisie.
 */
function useReglages(
  coequipiers: readonly string[],
  onDemandeComposition: ((enCours: boolean) => void) | undefined,
): Omit<ReglagesDuPlan, 'grappes'> {
  const [question, setQuestion] = useState<TacticalQuestion>('morts')
  const [qui, setQui] = useState<TacticalQui>('moi')
  const [spawn, setSpawn] = useState('')
  const [zonesVisibles, setZonesVisibles] = useState(true)
  const escouadeDisponible = coequipiers.length > 0
  const escouadeEnAttente = qui === 'escouade' && !escouadeDisponible
  return {
    question,
    onQuestionChange: setQuestion,
    qui: escouadeEnAttente ? 'moi' : qui,
    onQuiChange: (valeur) => {
      setQui(valeur)
      onDemandeComposition?.(valeur === 'escouade' && !escouadeDisponible)
    },
    escouadeDisponible,
    escouadeEnAttente,
    spawn,
    onSpawnChange: setSpawn,
    zonesVisibles,
    onZonesVisiblesChange: setZonesVisibles,
  }
}

/**
 * useZoneChoisie — la cellule choisie sur le plan, remise à zéro quand la lecture change de forme
 * (`cle`). Ajustée PENDANT LE RENDU (patron React « adjusting state when a prop changes »), pas dans
 * un effet : un effet ferait clignoter l'ancienne cellule un rendu de plus.
 */
function useZoneChoisie(cle: string) {
  const [choix, setChoix] = useState<ChoixDeZone>(null)
  const [cleAnterieure, setCleAnterieure] = useState(cle)
  if (cle !== cleAnterieure) {
    setCleAnterieure(cle)
    if (choix !== null) setChoix(null)
  }
  return [choix, setChoix] as const
}

/**
 * useEchapEcarteLaZone — Échap écarte la zone affichée (`actif`). Inactif tant qu'une mort est choisie :
 * le même Échap referme d'abord la mort (`useContributionChoisie`), un second écarte la zone. Il ne
 * prend pas la touche d'un champ de saisie ni d'une fenêtre qui l'a déjà traitée.
 */
function useEchapEcarteLaZone(actif: boolean, ecarter: () => void) {
  useEffect(() => {
    if (!actif) return
    const surTouche = (event: KeyboardEvent) => {
      if (event.key !== 'Escape' || event.defaultPrevented) return
      const cible = event.target
      if (cible instanceof HTMLElement && (cible.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(cible.tagName))) return
      ecarter()
    }
    window.addEventListener('keydown', surTouche)
    return () => window.removeEventListener('keydown', surTouche)
  }, [actif, ecarter])
}

/** Aucune contribution : une référence stable, pour que les mémos de la couche ne se recalculent pas. */
const AUCUNE_CONTRIBUTION: TacticalContribution[] = []

/**
 * useContributionChoisie — la contribution survolée et la contribution CHOISIE (clic sur sa tuile ou
 * son point), partagées par la liste et le plan. Le choix tient jusqu'à un re-clic ou Échap, et se
 * remet à zéro avec la zone, la lecture, les joueurs et la réapparition (`cle`) — ajusté pendant le
 * rendu, comme `useZoneChoisie`. C'est l'état qui ouvre la vue d'une mort (l'étape de la séquence et sa
 * lecture vivent dans `useVueDeMort` : une autre mort, ou un re-clic, repart du début de sa séquence).
 */
function useContributionChoisie(cle: string) {
  const [survol, setSurvol] = useState<number | null>(null)
  const [choisie, setChoisie] = useState<number | null>(null)
  const [cleAnterieure, setCleAnterieure] = useState(cle)
  if (cle !== cleAnterieure) {
    setCleAnterieure(cle)
    if (survol !== null) setSurvol(null)
    if (choisie !== null) setChoisie(null)
  }
  useEffect(() => {
    if (choisie === null) return
    const surTouche = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setChoisie(null)
    }
    window.addEventListener('keydown', surTouche)
    return () => window.removeEventListener('keydown', surTouche)
  }, [choisie])
  return {
    survol,
    setSurvol,
    choisie,
    choisir: (index: number) => {
      setChoisie((courante) => basculerChoix(courante, index))
    },
  }
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
  const selected = zoneAffichee(choix, chaude ? { col: chaude.col, row: chaude.lig } : null)
  const ecarter = useCallback(() => setChoix(ZONE_ECARTEE), [setChoix])
  const cellule = selected && lecture ? trouveCellule(lecture.cellules ?? [], selected.col, selected.row) : null
  const requete = useTacticalCellule(
    playerSlug,
    mapId,
    selected && lecture && pret ? { col: selected.col, lig: selected.row, pas_m: lecture.pas_m } : null,
    params,
  )
  return {
    selected,
    // L'identité de la zone et de ses contributions : le survol qui en dépend ne survit pas à un changement.
    cle: `${cle}:${selected?.col}:${selected?.row}`,
    contributions: selected ? (requete.data?.contributions ?? AUCUNE_CONTRIBUTION) : AUCUNE_CONTRIBUTION,
    // Un clic sur la zone affichée, ou loin de toute cellule servie, l'écarte (choixApresClic).
    choisir: (point: { x: number; y: number }) => {
      const cliquee = lecture ? choixDuClic(lecture.cellules ?? [], point, lecture.pas_m) : null
      setChoix(choixApresClic(selected, cliquee))
    },
    ecarter,
    cellule,
    detail: {
      data: requete.data,
      isPending: requete.isPending && selected !== null,
      isError: requete.isError && selected !== null,
    },
  }
}
