/**
 * standardPage — LA BOÎTE BORNÉE ET CENTRÉE DU CADRE STANDARD, posée à l'identique sur son en-tête et sur son
 * corps : même largeur maximale, mêmes marges latérales, donc les deux bords tombent l'un sur l'autre.
 * Le Cinéma et le Plein cadre n'en lisent rien.
 */
import type { CSSProperties } from 'react'

import { BOITE_PAGE_STANDARD_MAX_PX, COTES_PAGE_STANDARD } from '../../model/cotesStandard'

/** `w-full` + `mx-auto` : pleine largeur jusqu'au plafond, puis centrée. */
export const STANDARD_PAGE_CLASS = 'mx-auto w-full'

export const STANDARD_PAGE_STYLE: CSSProperties = Object.freeze({
  maxWidth: BOITE_PAGE_STANDARD_MAX_PX,
  paddingInline: COTES_PAGE_STANDARD.margeLateralePx,
})
