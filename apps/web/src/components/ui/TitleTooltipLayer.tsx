import { useEffect, useId, useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { attachTitleTooltips } from './titleTooltip.controller'
import { addDescribedBy, removeDescribedBy } from './titleTooltip.logic'
import { placeTooltip, TooltipPanel, type TooltipPlacement } from './tooltip-panel'

interface Shown {
  owner: Element
  text: string
}

/**
 * Couche globale des infobulles `title` : à monter UNE fois, à la racine de l'application (hors coquille
 * comprise : connexion, installation, erreurs). Elle remplace l'infobulle native des attributs `title` —
 * invisible dans un partage de fenêtre — par l'infobulle de l'application, même habillage que `Tooltip`
 * (`TooltipPanel`). Le détail du cycle (survol, focus clavier, restitution du `title`) est dans
 * `attachTitleTooltips`.
 */
export function TitleTooltipLayer() {
  const id = useId()
  const [shown, setShown] = useState<Shown | null>(null)
  const [placement, setPlacement] = useState<TooltipPlacement | null>(null)
  const panelRef = useRef<HTMLDivElement | null>(null)

  useEffect(
    () =>
      attachTitleTooltips(document, {
        show: (owner, text) => setShown({ owner, text }),
        hide: () => {
          setShown(null)
          setPlacement(null)
        },
      }),
    [],
  )

  // Relie l'élément à l'infobulle pendant l'affichage seulement.
  useEffect(() => {
    if (!shown) return
    addDescribedBy(shown.owner, id)
    return () => removeDescribedBy(shown.owner, id)
  }, [shown, id])

  useLayoutEffect(() => {
    if (!shown || !panelRef.current) return
    setPlacement(
      placeTooltip(shown.owner.getBoundingClientRect(), panelRef.current.getBoundingClientRect(), {
        width: window.innerWidth,
        height: window.innerHeight,
      }),
    )
  }, [shown])

  if (!shown) return null
  return createPortal(
    <TooltipPanel id={id} innerRef={panelRef} placement={placement} keepLineBreaks>
      {shown.text}
    </TooltipPanel>,
    document.body,
  )
}
