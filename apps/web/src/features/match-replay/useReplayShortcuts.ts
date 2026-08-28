/**
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
 * ...SAUF DEPUIS LA FRISE DU LECTEUR, ET C'EST UNE DÉCISION UTILISATEUR (gate du 2026-08-28).
 * Un `input[type=range]` est un champ de saisie pour le navigateur : la garde ci-dessus
 * l'attrapait, et les raccourcis MOURAIENT dès qu'on avait cliqué sur la frise — c'est-à-dire
 * exactement au moment où l'on analyse un match. Espace ne répondait plus, ←/→ avançaient d'UNE
 * image (le pas natif du champ) au lieu de sauter 10 s. L'exemption est portée par un ATTRIBUT
 * (`data-replay-timeline`) et non par un test de type : elle vise CE champ-là, pas la famille.
 *
 * LE VOLUME, LUI, RESTE UN CHAMP DE SAISIE — c'est le même élément HTML, et c'est pourquoi
 * l'exemption devait être nominative : qui vient de cliquer sur le volume et presse ← attend
 * que le volume baisse, pas que le film saute de dix secondes.
 *
 * `preventDefault` N'EST APPELÉ QUE SUR LES TOUCHES QU'ON TRAITE, et c'est lui qui supprime le
 * double pas : sur la frise focalisée, ←/→ ferait NOTRE saut PLUS le pas natif du champ.
 */
import { useEffect } from 'react'

/**
 * L'ATTRIBUT QUI REND SA FRAPPE À LA FRISE. Il est posé par `ReplayTimelineTracks` sur le seul
 * champ concerné et lu ici — le lien entre les deux est la pièce fragile de ce mécanisme, il a
 * son garde-fou dans `ReplayTimelineTracks.test.tsx`.
 */
export const TIMELINE_SHORTCUT_ATTR = 'data-replay-timeline'

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
  // LA FRISE DU LECTEUR EST EXEMPTÉE, ET ELLE SEULE (cf. l'en-tête) : l'exemption est portée par
  // un attribut, pas par un test de type — `input[type=range]` couvrirait aussi le volume, dont
  // les flèches doivent rester natives. `hasAttribute` est appelé en optionnel : la cible d'un
  // événement clavier peut être `window` ou `document`, qui n'ont pas de méthode d'attribut.
  if (el.hasAttribute?.(TIMELINE_SHORTCUT_ATTR)) return false
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
