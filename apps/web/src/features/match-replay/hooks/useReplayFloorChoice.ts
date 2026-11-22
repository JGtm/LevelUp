/**
 * useReplayFloorChoice — LA COUPE RETENUE ET LE SUIVI DU JOUEUR OUVERT, un seul état pour la commande d'étage.
 *
 * Le plan du fond Relief (D.1, choix 5) en fait UNE commande : un choix MANUEL d'étage (pastille, « Tout »,
 * curseur) COUPE le suivi, et c'est le suivi qui pose la coupe quand il est allumé. Les deux vivent donc
 * dans le même état, sans quoi un geste de l'utilisateur et un pas du suivi se disputeraient la coupe.
 *
 * LA PORTÉE (`scopeKey`, la carte) : une autre carte repart de « Tout », suivi éteint. Rien n'est persisté :
 * la coupe et le suivi naissent avec la page.
 *
 * LE SUIVI NE SURVIT PAS À LA FICHE : sans joueur ouvert (`canFollow` faux) il s'éteint, et rouvrir une fiche
 * ne le rallume pas — l'utilisateur le rallume (maquette D.1 : l'interrupteur est désactivé sans joueur ouvert).
 */
import { useCallback, useState } from 'react'

/** L'état rangé avec sa portée. */
interface Choice {
  key: string
  cut: number | null
  follow: boolean
}

export interface FloorChoice {
  /** La hauteur de la coupe ; `null` = « Tout ». */
  cut: number | null
  /** Le suivi est allumé (et la fiche d'un joueur est ouverte). */
  follow: boolean
  /** Un choix MANUEL : pose la coupe (`null` = « Tout ») et COUPE le suivi. */
  choose: (z: number | null) => void
  /** Le pas du suivi : pose la coupe, sans effet une fois le suivi coupé. */
  followTo: (z: number) => void
  setFollow: (on: boolean) => void
}

export function useReplayFloorChoice(scopeKey: string, canFollow: boolean): FloorChoice {
  const [raw, setRaw] = useState<Choice>({ key: scopeKey, cut: null, follow: false })
  // Sans joueur ouvert, le suivi s'éteint : l'état est ajusté pendant le rendu, jamais dans un effet.
  if (raw.follow && !canFollow) setRaw({ ...raw, follow: false })
  const here = raw.key === scopeKey
  const choose = useCallback((z: number | null) => setRaw({ key: scopeKey, cut: z, follow: false }), [scopeKey])
  const followTo = useCallback(
    (z: number) => setRaw((s) => (s.key !== scopeKey || !s.follow || s.cut === z ? s : { ...s, cut: z })),
    [scopeKey],
  )
  const setFollow = useCallback(
    (on: boolean) => setRaw((s) => ({ key: scopeKey, cut: s.key === scopeKey ? s.cut : null, follow: on })),
    [scopeKey],
  )
  return { cut: here ? raw.cut : null, follow: here && raw.follow && canFollow, choose, followTo, setFollow }
}
