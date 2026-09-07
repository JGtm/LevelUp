/**
 * SlidersIcon — l'icône « curseurs de réglage » du bouton du tiroir (trois glissières à
 * taquets). Inline SVG, comme les autres icônes du dépôt (pas de lucide-react — idiome
 * ShareLinkButton). Décorative : le libellé vit sur le bouton (aria-label/title), jamais
 * dans le SVG. Extraite de ReplayCanvas.tsx le 2026-08-24 (plafond de taille du cliquet).
 */
export function SlidersIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 16 16"
      className="h-4 w-4"
      aria-hidden="true"
    >
      <g stroke="currentColor" strokeWidth="1.5" strokeLinecap="round">
        <line x1="2" y1="3.5" x2="14" y2="3.5" />
        <line x1="2" y1="8" x2="14" y2="8" />
        <line x1="2" y1="12.5" x2="14" y2="12.5" />
      </g>
      <g fill="currentColor">
        <circle cx="10.5" cy="3.5" r="2" />
        <circle cx="5.5" cy="8" r="2" />
        <circle cx="9" cy="12.5" r="2" />
      </g>
    </svg>
  )
}
