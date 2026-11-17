/**
 * ReplayTransportIcons — LES ICÔNES DES BARRES DE LECTURE du rejeu. Toutes sont décoratives
 * (`aria-hidden`) : le bouton qui les porte garde le nom accessible. Elles vivent ici pour que
 * la barre complète et la barre d'une ligne les partagent sans les copier ; chacune prend la
 * couleur du texte du bouton (`currentColor`), jamais une couleur propre.
 */

/** Lecture : le triangle. */
export function PlayIcon() {
  return (
    <svg viewBox="0 0 16 16" className="h-[17px] w-[17px]" fill="currentColor" aria-hidden="true">
      <path d="M4.5 2.8a.8.8 0 0 1 1.2-.7l8 5.2a.8.8 0 0 1 0 1.4l-8 5.2a.8.8 0 0 1-1.2-.7z" />
    </svg>
  )
}

/** Pause : les deux barres. */
export function PauseIcon() {
  return (
    <svg viewBox="0 0 16 16" className="h-[17px] w-[17px]" fill="currentColor" aria-hidden="true">
      <rect x="3.5" y="2.5" width="3.4" height="11" rx="1" />
      <rect x="9.1" y="2.5" width="3.4" height="11" rx="1" />
    </svg>
  )
}

/** Recommencer : la flèche qui reboucle vers le début. */
export function RestartIcon() {
  return (
    <svg
      viewBox="0 0 16 16"
      className="h-[15px] w-[15px]"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M2.5 8a5.5 5.5 0 1 0 1.6-3.9" />
      <path d="M4.1 1.6v2.9H7" />
    </svg>
  )
}

/** Capture d'image : le boîtier d'un appareil photo et son objectif. */
export function CameraIcon() {
  return (
    <svg
      viewBox="0 0 16 16"
      className="h-[15px] w-[15px]"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M1.8 5.2h2.6l1.1-1.7h4.9l1.1 1.7h2.7v7.3H1.8z" />
      <circle cx="8" cy="8.9" r="2.4" />
    </svg>
  )
}

/**
 * Export : la flèche qui SORT du plateau. Volontairement différente du disque de
 * l'enregistrement : les deux commandes ne se confondent pas à l'œil, même si l'une remplace
 * l'autre dans la barre.
 */
export function ExportIcon() {
  return (
    <svg
      viewBox="0 0 16 16"
      className="h-[15px] w-[15px]"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M8 10.2V2.4" />
      <path d="M5.2 5.2 8 2.4l2.8 2.8" />
      <path d="M2.8 10.4v2.2a1 1 0 0 0 1 1h8.4a1 1 0 0 0 1-1v-2.2" />
    </svg>
  )
}

/** L'anneau qui tourne le temps d'un export. Même gabarit que l'icône d'export qu'il remplace. */
export function ExportSpinner() {
  return (
    <svg
      viewBox="0 0 16 16"
      className="h-[15px] w-[15px] animate-spin"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      aria-hidden="true"
    >
      <circle cx="8" cy="8" r="6" className="opacity-30" />
      <path d="M14 8a6 6 0 0 0-6-6" />
    </svg>
  )
}

/**
 * Lecture automatique, sur le modèle des lecteurs vidéo : un rail et sa pastille, pleins quand
 * c'est armé, en creux sinon. L'état se lit par la position de la pastille, pas par la couleur.
 */
export function AutoPlayIcon({ on }: { on: boolean }) {
  return (
    <svg viewBox="0 0 28 16" className="h-[15px] w-[26px]" aria-hidden="true">
      <rect
        x="1"
        y="3"
        width="26"
        height="10"
        rx="5"
        fill={on ? 'currentColor' : 'none'}
        stroke="currentColor"
        strokeWidth="1.5"
        opacity={on ? 0.35 : 0.6}
      />
      <circle cx={on ? 19 : 9} cy="8" r="3.2" fill="currentColor" />
    </svg>
  )
}

/** Plein écran du système : quatre coins vers l'extérieur pour entrer, vers l'intérieur pour sortir. */
export function FullscreenIcon({ exit }: { exit: boolean }) {
  return (
    <svg
      viewBox="0 0 16 16"
      className="h-[15px] w-[15px]"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {exit ? (
        <path d="M6 2.5V6H2.5M10 2.5V6h3.5M6 13.5V10H2.5M10 13.5V10h3.5" />
      ) : (
        <path d="M2.5 6V2.5H6M13.5 6V2.5H10M2.5 10v3.5H6M13.5 10v3.5H10" />
      )}
    </svg>
  )
}

/** Cadre Standard : une page à largeur bornée, son en-tête dessiné par un trait. */
export function StandardFrameIcon() {
  return (
    <svg
      viewBox="0 0 16 16"
      className="h-[15px] w-[15px]"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <rect x="2.5" y="2.5" width="11" height="11" rx="1.5" />
      <path d="M2.5 6h11" />
    </svg>
  )
}

/** Cadre Cinéma : un écran large, pleine largeur. */
export function CinemaFrameIcon() {
  return (
    <svg
      viewBox="0 0 16 16"
      className="h-[15px] w-[15px]"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <rect x="1.5" y="4" width="13" height="8" rx="1.5" />
    </svg>
  )
}

/** Cadre Plein cadre : un cadre qui contient un cadre plein, la fenêtre tout entière. */
export function FullFrameIcon() {
  return (
    <svg
      viewBox="0 0 16 16"
      className="h-[15px] w-[15px]"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <rect x="1.5" y="2" width="13" height="12" rx="1.5" />
      <rect x="4.5" y="5" width="7" height="6" rx="0.5" fill="currentColor" opacity="0.35" stroke="none" />
    </svg>
  )
}
