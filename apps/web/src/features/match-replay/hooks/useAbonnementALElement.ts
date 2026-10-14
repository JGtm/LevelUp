/**
 * useAbonnementALElement — ABONNE des écouteurs à l'élément qu'une référence désigne, et SUIT
 * cet élément quand il change.
 *
 * # POURQUOI PAS UN EFFET À DÉPENDANCE `[ref]`
 *
 * L'objet référence est stable : un effet qui ne dépend que de lui ne tourne qu'au premier
 * montage. Si la toile n'existe pas encore à ce moment (un plan qui attend ses données rend
 * d'abord un état vide, puis sa toile), l'écouteur n'est jamais posé : la molette et le clavier
 * restent sans effet sur un plan pourtant affiché. Ici, chaque rendu compare l'élément désigné à
 * celui qui est abonné, et réabonne quand il a changé (apparition, remplacement, disparition).
 *
 * `abonner` reçoit l'élément et rend la fonction qui désabonne. Il n'est rappelé qu'au
 * changement d'élément : l'état vif qu'il lit doit passer par une référence (cf. les appelants).
 */
import { useEffect, useRef, type RefObject } from 'react'

export function useAbonnementALElement<T extends Element>(
  ref: RefObject<T | null>,
  abonner: (el: T) => () => void,
): void {
  const abonne = useRef<{ el: T; desabonner: () => void } | null>(null)

  // Sans tableau de dépendances : la comparaison se refait après chaque rendu, et ne coûte
  // qu'une égalité de références tant que l'élément ne change pas.
  useEffect(() => {
    const el = ref.current
    if (abonne.current?.el === el) return
    abonne.current?.desabonner()
    abonne.current = el ? { el, desabonner: abonner(el) } : null
  })

  // Au démontage, le dernier abonnement se retire.
  useEffect(
    () => () => {
      abonne.current?.desabonner()
      abonne.current = null
    },
    [],
  )
}
