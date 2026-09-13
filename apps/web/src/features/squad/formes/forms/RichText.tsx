/**
 * RichText.tsx — LE GRAS DES TEXTES DU BLOC. Les constats, lexiques et notes de
 * l'artefact portent des passages en gras qui font partie du message (« Les
 * grenades sont sorties du bloc », « Ce qu'elle abandonne »).
 *
 * Les dictionnaires les marquent `**ainsi**`, et ce composant les rend. AUCUN
 * `dangerouslySetInnerHTML` : le texte reste du texte, le balisage reste une
 * paire d'astérisques, et une chaîne traduite ne peut pas injecter de HTML.
 */
import { Fragment } from 'react'

import { splitEmphasis } from '../richText'

export function RichText({ text }: { text: string }) {
  const parts = splitEmphasis(text)
  return (
    <>
      {parts.map((part, i) =>
        i % 2 === 1 ? (
          <strong key={`${i}-${part}`} className="font-semibold text-foreground">
            {part}
          </strong>
        ) : (
          <Fragment key={`${i}-${part}`}>{part}</Fragment>
        ),
      )}
    </>
  )
}
