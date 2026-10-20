import { useCallback, useRef, useState, type RefCallback } from 'react'

/**
 * useVisibleUneFois — dit si un élément est entré une fois dans le champ de vision, pour
 * ne déclencher qu'à ce moment un chargement coûteux (une image par vignette d'une liste longue).
 *
 * Le booléen ne repasse JAMAIS à faux : une vignette déjà vue a ce qu'il lui faut dans le cache,
 * et la voir clignoter à chaque défilement serait pire que l'avoir chargée. L'observateur se
 * débranche à la première entrée, et au démontage (la ref reçoit `null`).
 *
 * L'élément est suivi par rapport à la fenêtre, défilements intérieurs compris (la partie d'un
 * élément rognée par un conteneur à défilement ne compte pas comme visible). `margeDeChargement`
 * (en px) avance un peu le moment du chargement quand l'élément approche de la fenêtre.
 *
 * SANS `IntersectionObserver` (environnement qui n'en a pas), l'élément est réputé visible :
 * dégradation vers le chargement immédiat, jamais une image qui ne viendrait pas.
 */
export function useVisibleUneFois<T extends Element>(
  margeDeChargement = 100,
): [RefCallback<T>, boolean] {
  const [visible, setVisible] = useState(() => typeof IntersectionObserver === 'undefined')
  const observateur = useRef<IntersectionObserver | null>(null)

  const ref = useCallback<RefCallback<T>>(
    (element) => {
      observateur.current?.disconnect()
      observateur.current = null
      if (!element || typeof IntersectionObserver === 'undefined') return
      const obs = new IntersectionObserver(
        (entrees) => {
          if (!entrees.some((e) => e.isIntersecting)) return
          setVisible(true)
          obs.disconnect()
        },
        { rootMargin: `${margeDeChargement}px` },
      )
      obs.observe(element)
      observateur.current = obs
    },
    [margeDeChargement],
  )

  return [ref, visible]
}
