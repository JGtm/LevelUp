/**
 * useCopyToClipboard — hook canonique « copier + coche transitoire ».
 *
 * Quatre boutons portaient le même corps recopié (état `copied`, `setTimeout`,
 * `navigator.clipboard.writeText`, `catch` muet) avec deux durées différentes
 * (2000 ms et 1500 ms) : ShareLinkButton, IdentitiesSection (XuidCell),
 * MatchHeader.card et CopyCodeButton. Règle CLAUDE.md n°6 (à la 3e copie :
 * helper + garde-rail) — le ratchet est `useCopyToClipboard.guard.test.ts`.
 *
 * Ce que le hook corrige par rapport aux copies :
 *  - une seule durée de coche (`COPIED_FEEDBACK_MS`) ;
 *  - le minuteur est annulé au démontage ET à chaque nouveau clic (les copies
 *    laissaient la coche d'une 2e copie disparaître au minuteur de la 1re, et
 *    appelaient setState après démontage) ;
 *  - l'échec du presse-papier est journalisé (`log.error`) au lieu d'être avalé,
 *    et `copied` reste FAUX — jamais de fausse confirmation de copie
 *    (MatchHeader.card n'avait même pas de `catch` : rejet non géré).
 *
 * L'apparence (icône, libellés, classes) reste au site appelant : le hook ne
 * porte que la mécanique.
 */
import { useCallback, useEffect, useRef, useState } from 'react'

import { log } from './_logger'

/** Durée unique de la coche « copié » sur toute l'application. */
export const COPIED_FEEDBACK_MS = 2000

export interface UseCopyToClipboard {
  /** Copie `text` ; met `copied` à vrai le temps de `COPIED_FEEDBACK_MS`. */
  copy: (text: string) => Promise<void>
  /** Vrai pendant la fenêtre de feedback qui suit une copie RÉUSSIE. */
  copied: boolean
}

export function useCopyToClipboard(): UseCopyToClipboard {
  const [copied, setCopied] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const mountedRef = useRef(true)

  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
      if (timerRef.current !== null) clearTimeout(timerRef.current)
    }
  }, [])

  const copy = useCallback(async (text: string) => {
    // Un nouveau clic réarme la fenêtre de feedback depuis zéro.
    if (timerRef.current !== null) clearTimeout(timerRef.current)

    try {
      await navigator.clipboard.writeText(text)
    } catch (err) {
      // Presse-papier indisponible (contexte non sécurisé, permission refusée).
      // Jamais de coche dans ce cas : le texte reste sélectionnable à la main.
      log.error('clipboard:write_failed', 'navigator.clipboard.writeText a échoué', err)
      if (mountedRef.current) setCopied(false)
      return
    }

    if (!mountedRef.current) return
    setCopied(true)
    timerRef.current = setTimeout(() => {
      timerRef.current = null
      if (mountedRef.current) setCopied(false)
    }, COPIED_FEEDBACK_MS)
  }, [])

  return { copy, copied }
}
