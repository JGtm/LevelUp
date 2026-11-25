/**
 * SettingIcons — LES PICTOGRAMMES des réglages que l'on peut épingler au bord de la carte (accès rapide), et la
 * punaise, la poubelle et l'étoile des vues. Des SVG en ligne comme les autres icônes du rejeu (`viewBox` 16, trait
 * `currentColor` de 1,5, extrémités arrondies) : aucune bibliothèque d'icônes dans la feature.
 */
import type { ReactNode } from 'react'

import type { SettingId } from './settingsTable'

const GLYPHS: Partial<Record<SettingId, ReactNode>> = {
  aim: <path d="M8 13.5L3 4a8.5 8.5 0 0 1 10 0z" />,
  trail: (
    <>
      <path d="M2 12c3 0 3-6 6-6s3 4 5 4" />
      <circle cx="13.5" cy="10" r="1.2" fill="currentColor" stroke="none" />
    </>
  ),
  shotFx: <path d="M8 2v3M8 11v3M2 8h3M11 8h3M4 4l2 2M10 10l2 2M12 4l-2 2M6 10l-2 2" />,
  killFx: (
    <>
      <path d="M3 13L13 3" />
      <circle cx="3" cy="13" r="1.3" />
      <circle cx="13" cy="3" r="1.3" />
    </>
  ),
  spawns: (
    <>
      <circle cx="8" cy="9" r="4" />
      <path d="M8 5V2" />
    </>
  ),
  deaths: <path d="M4 4l8 8M12 4l-8 8" />,
  killers: <path d="M8 3l5 9H3z" />,
  forecast: (
    <>
      <circle cx="8" cy="8" r="2" />
      <circle cx="8" cy="8" r="5.3" strokeDasharray="2 2" />
    </>
  ),
  reliefLines: <path d="M2 5c3-2 5 2 12 0M2 9c3-2 5 2 12 0M2 13c3-2 5 2 12 0" />,
  wallPlan: <path d="M2.5 13.5v-10h11v10M6 13.5V8h4v5.5" />,
  zones: <rect x="2.5" y="3.5" width="11" height="9" rx="2" strokeDasharray="2.2 2" />,
  heatmap: (
    <>
      <rect x="2" y="2" width="5" height="5" fill="currentColor" opacity=".25" stroke="none" />
      <rect x="9" y="2" width="5" height="5" fill="currentColor" opacity=".6" stroke="none" />
      <rect x="2" y="9" width="5" height="5" fill="currentColor" opacity=".9" stroke="none" />
      <rect x="9" y="9" width="5" height="5" fill="currentColor" opacity=".4" stroke="none" />
    </>
  ),
  weaponPads: (
    <>
      <rect x="3" y="3" width="10" height="10" rx="2" />
      <circle cx="8" cy="8" r="1.6" fill="currentColor" stroke="none" />
    </>
  ),
  groundWeapons: <path d="M2 7h8l2-2h2v3h-3v3H8V9H2z" />,
  vehicles: <path d="M2 10.5h12M3.2 10.5L4.6 6.5h6.8l1.4 4M5 12.7h.1M11 12.7h.1" />,
  placements: <path d="M8 2l5 6-5 6-5-6z" />,
  objectives: <path d="M4 14V2M4 3h8l-2 3 2 3H4" />,
  flags: <path d="M4 14V2M4 3h8l-2 3 2 3H4" fill="currentColor" fillOpacity=".35" />,
  planFeed: <path d="M2 5h6M2 8h9M2 11h7" />,
  chatterBubbles: <path d="M2.5 3.5h11v7H7l-3 3v-3H2.5z" />,
  chatterIcon: (
    <>
      <path d="M2 6h2.4L8 3v10L4.4 10H2z" fill="currentColor" strokeWidth=".8" />
      <path d="M10.4 5.8a3.2 3.2 0 0 1 0 4.4" />
    </>
  ),
  chatterFeed: (
    <>
      <path d="M2 4.5h12M2 8h5M2 11.5h7" />
      <path d="M10.5 7h3.5v3h-1.8l-1.7 1.6z" />
    </>
  ),
  chatterVoice: (
    <>
      <path d="M2 6h2.4L8 3v10L4.4 10H2z" fill="currentColor" strokeWidth=".8" />
      <path d="M10.4 5.8a3.2 3.2 0 0 1 0 4.4" />
      <path d="M12.4 3.8a6 6 0 0 1 0 8.4" />
    </>
  ),
  sound: (
    <>
      <path d="M1.5 5.6h2.6L8.3 2v12L4.1 10.4H1.5z" fill="currentColor" strokeWidth=".8" />
      <path d="M10.6 5.1a4.1 4.1 0 0 1 0 5.8" />
      <path d="M12.8 2.9a7.2 7.2 0 0 1 0 10.2" />
    </>
  ),
}

/** Une icône à la taille voulue ; faute de pictogramme pour ce réglage, la punaise. */
function Svg({ children, size = 16 }: { children: ReactNode; size?: number }) {
  return (
    <svg
      viewBox="0 0 16 16"
      width={size}
      height={size}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {children}
    </svg>
  )
}

export function SettingIcon({ id }: { id: SettingId }) {
  return <Svg>{GLYPHS[id] ?? <PinPath />}</Svg>
}

function PinPath() {
  return <path d="M9.8 1.8l4.4 4.4-1.6.7-2.3 2.3.3 2.9-.9.9-2.8-2.8-3.7 3.7-.7-.7 3.7-3.7L3.4 7.7l.9-.9 2.9.3 2.3-2.3z" />
}

export function PinIcon() {
  return (
    <Svg size={13}>
      <PinPath />
    </Svg>
  )
}

export function TrashIcon() {
  return (
    <Svg size={13}>
      <path d="M3 4.5h10M6.5 4.5V3h3v1.5M4.5 4.5l.6 8.5h5.8l.6-8.5" />
    </Svg>
  )
}

export function StarIcon({ filled }: { filled: boolean }) {
  return (
    <Svg size={13}>
      <path d="M8 2l1.8 3.7 4 .6-2.9 2.8.7 4L8 11.2 4.4 13.1l.7-4L2.2 6.3l4-.6z" fill={filled ? 'currentColor' : 'none'} />
    </Svg>
  )
}

export function ChevronIcon({ up }: { up?: boolean }) {
  return (
    <svg viewBox="0 0 10 6" width="10" height="6" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d={up ? 'M1 5l4-4 4 4' : 'M1 1l4 4 4-4'} />
    </svg>
  )
}
