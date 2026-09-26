/**
 * segmentLabelFit — une valeur s'écrit DANS son segment de barre seulement si elle y tient
 * avec 6 px de marge de chaque côté ; sinon elle part sur une ligne de repli (règle S3 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 : jamais tronquée, jamais seulement en
 * infobulle).
 *
 * Mesure au PIXEL, pas à la fraction de piste : un « 3 » tient dans 20 px, un « 121 » non —
 * un seuil en pourcentage (celui de `StackedTrack`) se trompe dans les deux sens selon la
 * largeur de la carte. L'étiquette reste dans le DOM (`visibility: hidden` quand elle ne tient
 * pas) pour pouvoir être remesurée au redimensionnement.
 *
 * Contrat DOM du hook : dans le conteneur, chaque segment porte `data-fit-key="<clé>"` et son
 * étiquette `data-fit-label`. Hors navigateur (jsdom, largeurs nulles), aucune étiquette ne
 * « tient » : toutes les valeurs passent au repli — visibles, jamais perdues.
 */
import { useLayoutEffect, useState, type RefObject } from 'react'

/** Marge minimale entre l'étiquette et chaque bord de son segment (px). */
export const SEGMENT_LABEL_MARGIN_PX = 6

/** L'étiquette de largeur `labelPx` tient-elle dans un segment de `segmentPx` ? */
export function labelFitsInSegment(
  labelPx: number,
  segmentPx: number,
  marginPx: number = SEGMENT_LABEL_MARGIN_PX,
): boolean {
  return labelPx > 0 && labelPx + 2 * marginPx <= segmentPx
}

/** Clés des segments du conteneur dont l'étiquette ne tient pas. */
export function measureHiddenSegments(container: HTMLElement): Set<string> {
  const hidden = new Set<string>()
  container.querySelectorAll<HTMLElement>('[data-fit-key]').forEach((seg) => {
    const key = seg.dataset.fitKey ?? ''
    const label = seg.querySelector<HTMLElement>('[data-fit-label]')
    const labelPx = label?.getBoundingClientRect().width ?? 0
    if (!labelFitsInSegment(labelPx, seg.getBoundingClientRect().width)) hidden.add(key)
  })
  return hidden
}

function sameSet(a: ReadonlySet<string>, b: ReadonlySet<string>): boolean {
  if (a.size !== b.size) return false
  for (const k of a) if (!b.has(k)) return false
  return true
}

/**
 * Mesure, après chaque rendu du contenu (`contentKey`) et à chaque redimensionnement du
 * conteneur, les segments dont l'étiquette ne tient pas.
 */
export function useSegmentLabelFit(
  ref: RefObject<HTMLElement | null>,
  contentKey: unknown,
): ReadonlySet<string> {
  const [hidden, setHidden] = useState<ReadonlySet<string>>(() => new Set())
  useLayoutEffect(() => {
    const el = ref.current
    if (!el) return
    const update = () => {
      const next = measureHiddenSegments(el)
      setHidden((prev) => (sameSet(prev, next) ? prev : next))
    }
    update()
    if (typeof ResizeObserver === 'undefined') return
    const ro = new ResizeObserver(update)
    ro.observe(el)
    return () => ro.disconnect()
  }, [ref, contentKey])
  return hidden
}
