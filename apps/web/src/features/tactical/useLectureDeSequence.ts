/**
 * useLectureDeSequence — la LECTURE ANIMÉE d'une séquence de mort : une tête de lecture qui avance en
 * temps réel de la première à la dernière étape servie (−4 s à +3 s), s'arrête sur la dernière, et se
 * reprend, se met en pause ou saute à une étape.
 *
 * LA TÊTE DE LECTURE N'EST PAS UN ÉTAT REACT. Elle avance à chaque image (`requestAnimationFrame`) ;
 * la passer par `useState` repeindrait tout le plan soixante fois par seconde. Elle vit dans une
 * horloge à abonnés (`HorlogeDeLecture`, lue par `useSyncExternalStore`) que seuls la toile et
 * l'horloge écrite écoutent. React ne voit passer que ce qui change rarement : l'état de la lecture
 * (`lecture`, `pause`, `fin`) et l'étape en vigueur (une fois par seconde).
 *
 * DÉPART : lecture automatique depuis la première étape ; sous `prefers-reduced-motion`, pas de lecture
 * automatique — la tête se pose sur l'étape de la mort et la lecture ne part que sur demande.
 * Une nouvelle séquence (autre mort) repart de son début : l'état est réinitialisé PENDANT LE RENDU
 * quand la séquence change, jamais dans un effet (aucune image de l'ancienne tête sur la nouvelle mort).
 */
import { useCallback, useEffect, useMemo, useState } from 'react'

import type { TacticalSequence } from '@/lib/api/types'

import { avancer, bornesDeLecture, etapeALInstant, etapeDeLaMort } from './sequenceLecture.logic'

/** Une horloge à abonnés : la valeur courante de la tête de lecture et de quoi s'y abonner. */
export interface HorlogeDeLecture {
  lire: () => number
  abonner: (rappel: () => void) => () => void
}

/** `lecture` avance, `pause` est arrêtée (mi-course ou sur une étape), `fin` a atteint la dernière étape. */
export type EtatDeLecture = 'lecture' | 'pause' | 'fin'

export interface LectureDeSequence {
  horloge: HorlogeDeLecture
  etat: EtatDeLecture
  /** Repart de la première étape. */
  rejouer: () => void
  /** Pause, ou reprise de là où la tête s'est arrêtée ; depuis la fin, repart du début. */
  basculer: () => void
  /** Pause et saut au décalage donné (ms depuis la mort). */
  allerA: (offsetMs: number) => void
  /** Pause : la tête reste où elle est, la vue de l'instantané reprend la scène. */
  suspendre: () => void
}

interface HorlogeEcrivable extends HorlogeDeLecture {
  poser: (offsetMs: number) => void
}

function creerHorloge(depart: number): HorlogeEcrivable {
  let valeur = depart
  const abonnes = new Set<() => void>()
  return {
    lire: () => valeur,
    abonner: (rappel) => {
      abonnes.add(rappel)
      return () => abonnes.delete(rappel)
    },
    poser: (offsetMs) => {
      if (offsetMs === valeur) return
      valeur = offsetMs
      abonnes.forEach((rappel) => rappel())
    },
  }
}

/** L'état de départ d'une séquence : le début en lecture, ou l'étape de la mort en pause (mouvement réduit). */
function depart(seq: TacticalSequence | null, mouvementReduit: boolean): { offsetMs: number; etat: EtatDeLecture; etape: number | null } {
  const bornes = seq ? bornesDeLecture(seq) : null
  if (!seq || !bornes) return { offsetMs: 0, etat: 'pause', etape: null }
  if (mouvementReduit) {
    const mort = etapeDeLaMort(seq)
    return { offsetMs: 0, etat: 'pause', etape: mort ?? etapeALInstant(seq, 0) }
  }
  return { offsetMs: bornes.debutMs, etat: 'lecture', etape: 0 }
}

/**
 * useLectureDeSequence — rend la lecture et l'étape en vigueur (`null` = vue de l'instantané). `null`
 * sans séquence lisible (moins de deux étapes).
 */
export function useLectureDeSequence(
  sequence: TacticalSequence | null,
  mouvementReduit: boolean,
): { lecture: LectureDeSequence | null; etape: number | null; choisirEtape: (etape: number | null) => void } {
  const bornes = useMemo(() => (sequence ? bornesDeLecture(sequence) : null), [sequence])
  const initial = useMemo(() => depart(sequence, mouvementReduit), [sequence, mouvementReduit])
  const horloge = useMemo(() => creerHorloge(initial.offsetMs), [initial])
  const [etat, setEtat] = useState<EtatDeLecture>(initial.etat)
  const [etape, setEtape] = useState<number | null>(initial.etape)
  const [suivi, setSuivi] = useState(initial)
  if (suivi !== initial) {
    setSuivi(initial)
    setEtat(initial.etat)
    setEtape(initial.etape)
  }

  useEffect(() => {
    if (etat !== 'lecture' || !sequence || !bornes) return
    let image = 0
    let dernier: number | null = null
    const boucle = (maintenant: number) => {
      const dt = dernier === null ? 0 : maintenant - dernier
      dernier = maintenant
      const pas = avancer(horloge.lire(), dt, bornes)
      horloge.poser(pas.offsetMs)
      setEtape(etapeALInstant(sequence, pas.offsetMs))
      if (pas.fini) {
        setEtat('fin')
        return
      }
      image = requestAnimationFrame(boucle)
    }
    image = requestAnimationFrame(boucle)
    return () => cancelAnimationFrame(image)
  }, [etat, sequence, bornes, horloge])

  const rejouer = useCallback(() => {
    if (!bornes) return
    horloge.poser(bornes.debutMs)
    setEtape(0)
    setEtat('lecture')
  }, [bornes, horloge])
  const basculer = useCallback(() => {
    if (etat === 'lecture') setEtat('pause')
    else if (etat === 'fin' || (bornes !== null && horloge.lire() >= bornes.finMs)) rejouer()
    else setEtat('lecture')
  }, [etat, bornes, horloge, rejouer])
  const allerA = useCallback(
    (offsetMs: number) => {
      if (!sequence) return
      horloge.poser(offsetMs)
      setEtape(etapeALInstant(sequence, offsetMs))
      setEtat('pause')
    },
    [sequence, horloge],
  )
  const suspendre = useCallback(() => setEtat((e) => (e === 'lecture' ? 'pause' : e)), [])
  const choisirEtape = useCallback(
    (cible: number | null) => {
      if (cible === null) {
        setEtape(null)
        suspendre()
      } else if (sequence?.etapes?.[cible]) allerA(sequence.etapes[cible].offset_ms)
    },
    [sequence, allerA, suspendre],
  )

  const lecture = useMemo<LectureDeSequence | null>(
    () => (bornes ? { horloge, etat, rejouer, basculer, allerA, suspendre } : null),
    [bornes, horloge, etat, rejouer, basculer, allerA, suspendre],
  )
  return { lecture, etape: bornes ? etape : null, choisirEtape }
}
