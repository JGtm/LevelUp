/**
 * useTitleConvergence — convergence segment d'URL ↔ titre du magasin (D-6) du layout `/t/$titleSlug`.
 *
 * Le layout PROJETTE le verdict pur `resolveTitleGate` ; cet effet, lui, AGIT : quand le titre du
 * segment est valide mais diffère du titre courant (`diverges`), il l'applique. Rend l'état d'échec
 * (`applyFailed`, écran de réessai) et de quoi le lever (`retry`).
 */
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'

import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import { useAppShellStore } from '@/stores/appShellStore'
import { applyActiveTitle } from './applyActiveTitle'

export interface TitleConvergenceInput {
  /** Segment `$titleSlug` de l'URL. */
  titleSlug: string
  /** Titre valide mais ≠ titre courant du magasin. */
  diverges: boolean
  /** Une bascule est déjà en vol. */
  isTitleSwitching: boolean
}

export function useTitleConvergence({ titleSlug, diverges, isTitleSwitching }: TitleConvergenceInput): {
  applyFailed: boolean
  retry: () => void
} {
  const navigate = useNavigate()
  const [applyFailed, setApplyFailed] = useState(false)

  // Convergence D-6 : sur divergence segment↔store (titre valide mais ≠ courant),
  // appliquer le titre. Gardes anti-double-bascule : ni si une bascule est déjà en
  // vol (isTitleSwitching, y compris déclenchée par une ré-entrée), ni si une
  // bascule vient d'échouer SANS joueur courant (applyFailed → écran de réessai).
  // applyingRef dédoublonne les ré-entrées d'un même commit (StrictMode dev). À la
  // complétion réussie, currentTitleSlug converge → l'effet re-run et sort.
  //
  // Chemin d'erreur D-6 COMPLET (Phase 4b) : en échec d'applyActiveTitle, PAS de
  // rollback store-only (URL=cible, store=courant → divergence → boucle de
  // réconciliation). À la place :
  //  - joueur courant présent → toast + navigate REPLACE vers le titre COURANT (non
  //    basculé). Le segment redevient == store → plus de divergence → aucune
  //    re-tentative (anti-boucle). On NE pose PAS applyFailed : ce layout reste
  //    MONTÉ au changement de segment (param $titleSlug, même instance de route) —
  //    un applyFailed persistant bloquerait alors un futur switch légitime.
  //  - aucun joueur courant → pas de route joueur cible à viser : on conserve
  //    l'écran switch_failed (retry + retour manuel), fallback inchangé.
  const applyingRef = useRef(false)
  useEffect(() => {
    // Convergence ROBUSTE (course back/forward, 4c) : applyingRef ne sert qu'à
    // dédoublonner la double-invocation SYNCHRONE de StrictMode (dev). Il faut le
    // relâcher dès qu'aucune bascule n'est réellement en vol — sinon, quand une
    // bascule se termine (isTitleSwitching repasse false) ALORS qu'une divergence
    // subsiste (segment changé pendant le vol via popstate/refocus §7), l'effet
    // re-déclenché par le re-render synchrone du store verrait un applyingRef encore
    // armé (relâché seulement dans le .finally, plus tard) → la convergence CALERAIT.
    // On lit l'état LIVE du store (pas la closure) : après le 1er appel,
    // applyActiveTitle a posé isTitleSwitching=true de façon synchrone → la 2e
    // invocation StrictMode le voit true et NE relâche PAS le verrou (dédup préservée).
    if (!useAppShellStore.getState().isTitleSwitching) applyingRef.current = false
    if (!diverges || isTitleSwitching || applyFailed || applyingRef.current) return
    applyingRef.current = true
    applyActiveTitle(titleSlug)
      .catch(() => {
        const s = useAppShellStore.getState()
        const playerSlug = s.currentPlayer?.player_slug
        if (playerSlug) {
          toast.error(
            formatMessage(commonManifest, 'common.title_gate.switch_failed_toast', s.locale),
          )
          void navigate({
            to: '/{-$lang}/t/$titleSlug/players/$playerSlug/home',
            params: { titleSlug: s.currentTitleSlug, playerSlug },
            replace: true,
          })
        } else {
          setApplyFailed(true)
        }
      })
      .finally(() => {
        applyingRef.current = false
      })
  }, [diverges, titleSlug, isTitleSwitching, applyFailed, navigate])

  return { applyFailed, retry: () => setApplyFailed(false) }
}
