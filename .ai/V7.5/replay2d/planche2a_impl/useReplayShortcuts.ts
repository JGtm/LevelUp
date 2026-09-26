/**
 * DESTINATION : apps/web/src/features/match-replay/useReplayShortcuts.ts
 *
 * useReplayShortcuts — LE CLAVIER DU LECTEUR (demande utilisateur du 2026-08-27).
 *
 * LES RACCOURCIS SONT CEUX QUE TOUT LE MONDE CONNAÎT DÉJÀ, et pas un de plus : Espace et K
 * pour lecture/pause, ←/→ et J/L pour ±10 s, M pour le son, R pour recommencer, « , » et « . »
 * pour l'image par image. Inventer une convention maison ferait apprendre ce que le lecteur
 * sait déjà.
 *
 * IL NE VOLE PAS LA FRAPPE : un raccourci ne part jamais depuis un champ de saisie, un
 * `contenteditable`, ni avec un modificateur (Ctrl/Cmd/Alt) — sans quoi Espace dans une
 * recherche mettrait le rejeu en pause, et Cmd+R rechargerait ET rembobinerait.
 *
 * `preventDefault` N'EST APPELÉ QUE SUR LES TOUCHES QU'ON TRAITE : Espace ferait défiler la
 * page, ←/→ déplacerait le curseur natif de la frise EN PLUS de notre saut (double pas).
 */
import { useEffect } from 'react'

/** Ce que le clavier commande. Chaque entrée est une commande déjà existante du lecteur. */
export interface ReplayShortcutHandlers {
  togglePlay: () => void
  seekBy: (seconds: number) => void
  stepFrames: (frames: number) => void
  restart: () => void
  toggleSound: () => void
  /** Le saut des flèches, en secondes (cf. SKIP_SECONDS de la barre). */
  skipSeconds: number
  /** `false` quand la page n'a pas de rejeu chargé : rien n'est écouté. */
  enabled: boolean
}

function isTypingTarget(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null
  if (!el) return false
  const tag = el.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true
  return el.isContentEditable === true
}

export function useReplayShortcuts(h: ReplayShortcutHandlers): void {
  const { togglePlay, seekBy, stepFrames, restart, toggleSound, skipSeconds, enabled } = h
  useEffect(() => {
    if (!enabled) return
    function onKeyDown(e: KeyboardEvent) {
      if (e.ctrlKey || e.metaKey || e.altKey) return
      if (isTypingTarget(e.target)) return
      switch (e.key) {
        case ' ':
        case 'k':
        case 'K':
          e.preventDefault()
          togglePlay()
          return
        case 'ArrowLeft':
        case 'j':
        case 'J':
          e.preventDefault()
          seekBy(-skipSeconds)
          return
        case 'ArrowRight':
        case 'l':
        case 'L':
          e.preventDefault()
          seekBy(skipSeconds)
          return
        case ',':
          e.preventDefault()
          stepFrames(-1)
          return
        case '.':
          e.preventDefault()
          stepFrames(1)
          return
        case 'm':
        case 'M':
          e.preventDefault()
          toggleSound()
          return
        case 'r':
        case 'R':
          e.preventDefault()
          restart()
          return
        default:
          return
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [enabled, togglePlay, seekBy, stepFrames, restart, toggleSound, skipSeconds])
}
