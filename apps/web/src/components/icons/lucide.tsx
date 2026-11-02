/**
 * Icônes Lucide (https://lucide.dev), copiées en SVG : aucune dépendance npm.
 *
 * Lucide est sous licence ISC :
 *
 *   ISC License
 *
 *   Copyright (c) for portions of Lucide are held by Cole Bemis 2013-2022 as part of Feather (MIT).
 *   All other copyright (c) for Lucide are held by Lucide Contributors 2022.
 *
 *   Permission to use, copy, modify, and/or distribute this software for any purpose with or without
 *   fee is hereby granted, provided that the above copyright notice and this permission notice appear
 *   in all copies.
 *
 *   THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH REGARD TO THIS
 *   SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE
 *   AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 *   WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT,
 *   NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE
 *   OF THIS SOFTWARE.
 *
 * Glyphes copiés de lucide-static 0.552.0 : layout-dashboard, crosshair, swords, award, flame, globe,
 * film, search, book-open, bug, book-open-text, settings, sun-moon, info, panel-left-close,
 * panel-left-open. Un composant par glyphe, décoratif (`aria-hidden` : le libellé vit sur le bouton ou le
 * lien), couleur `currentColor`, trait de 2 px. Réservées à l'interface de l'application : les icônes du
 * jeu restent réservées au contenu Halo (règles de contenu de Microsoft). La mention de licence est
 * aussi dans `apps/desktop/THIRD_PARTY_NOTICES.md`.
 */
import type { ReactNode, SVGProps } from 'react'

export type LucideIconProps = Omit<SVGProps<SVGSVGElement>, 'children'>

function LucideSvg({ children, ...props }: LucideIconProps & { children: ReactNode }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      {...props}
    >
      {children}
    </svg>
  )
}

export function LayoutDashboardIcon(props: LucideIconProps) {
  return (
    <LucideSvg {...props}>
      <rect width="7" height="9" x="3" y="3" rx="1" />
      <rect width="7" height="5" x="14" y="3" rx="1" />
      <rect width="7" height="9" x="14" y="12" rx="1" />
      <rect width="7" height="5" x="3" y="16" rx="1" />
    </LucideSvg>
  )
}

export function CrosshairIcon(props: LucideIconProps) {
  return (
    <LucideSvg {...props}>
      <circle cx="12" cy="12" r="10" />
      <line x1="22" x2="18" y1="12" y2="12" />
      <line x1="6" x2="2" y1="12" y2="12" />
      <line x1="12" x2="12" y1="6" y2="2" />
      <line x1="12" x2="12" y1="22" y2="18" />
    </LucideSvg>
  )
}

export function SwordsIcon(props: LucideIconProps) {
  return (
    <LucideSvg {...props}>
      <polyline points="14.5 17.5 3 6 3 3 6 3 17.5 14.5" />
      <line x1="13" x2="19" y1="19" y2="13" />
      <line x1="16" x2="20" y1="16" y2="20" />
      <line x1="19" x2="21" y1="21" y2="19" />
      <polyline points="14.5 6.5 18 3 21 3 21 6 17.5 9.5" />
      <line x1="5" x2="9" y1="14" y2="18" />
      <line x1="7" x2="4" y1="17" y2="20" />
      <line x1="3" x2="5" y1="19" y2="21" />
    </LucideSvg>
  )
}

export function AwardIcon(props: LucideIconProps) {
  return (
    <LucideSvg {...props}>
      <path d="m15.477 12.89 1.515 8.526a.5.5 0 0 1-.81.47l-3.58-2.687a1 1 0 0 0-1.197 0l-3.586 2.686a.5.5 0 0 1-.81-.469l1.514-8.526" />
      <circle cx="12" cy="8" r="6" />
    </LucideSvg>
  )
}

export function FlameIcon(props: LucideIconProps) {
  return (
    <LucideSvg {...props}>
      <path d="M12 3q1 4 4 6.5t3 5.5a1 1 0 0 1-14 0 5 5 0 0 1 1-3 1 1 0 0 0 5 0c0-2-1.5-3-1.5-5q0-2 2.5-4" />
    </LucideSvg>
  )
}

export function GlobeIcon(props: LucideIconProps) {
  return (
    <LucideSvg {...props}>
      <circle cx="12" cy="12" r="10" />
      <path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20" />
      <path d="M2 12h20" />
    </LucideSvg>
  )
}

export function FilmIcon(props: LucideIconProps) {
  return (
    <LucideSvg {...props}>
      <rect width="18" height="18" x="3" y="3" rx="2" />
      <path d="M7 3v18" />
      <path d="M3 7.5h4" />
      <path d="M3 12h18" />
      <path d="M3 16.5h4" />
      <path d="M17 3v18" />
      <path d="M17 7.5h4" />
      <path d="M17 16.5h4" />
    </LucideSvg>
  )
}

export function SearchIcon(props: LucideIconProps) {
  return (
    <LucideSvg {...props}>
      <path d="m21 21-4.34-4.34" />
      <circle cx="11" cy="11" r="8" />
    </LucideSvg>
  )
}

export function BookOpenIcon(props: LucideIconProps) {
  return (
    <LucideSvg {...props}>
      <path d="M12 7v14" />
      <path d="M3 18a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h5a4 4 0 0 1 4 4 4 4 0 0 1 4-4h5a1 1 0 0 1 1 1v13a1 1 0 0 1-1 1h-6a3 3 0 0 0-3 3 3 3 0 0 0-3-3z" />
    </LucideSvg>
  )
}

export function BugIcon(props: LucideIconProps) {
  return (
    <LucideSvg {...props}>
      <path d="M12 20v-9" />
      <path d="M14 7a4 4 0 0 1 4 4v3a6 6 0 0 1-12 0v-3a4 4 0 0 1 4-4z" />
      <path d="M14.12 3.88 16 2" />
      <path d="M21 21a4 4 0 0 0-3.81-4" />
      <path d="M21 5a4 4 0 0 1-3.55 3.97" />
      <path d="M22 13h-4" />
      <path d="M3 21a4 4 0 0 1 3.81-4" />
      <path d="M3 5a4 4 0 0 0 3.55 3.97" />
      <path d="M6 13H2" />
      <path d="m8 2 1.88 1.88" />
      <path d="M9 7.13V6a3 3 0 1 1 6 0v1.13" />
    </LucideSvg>
  )
}

export function BookOpenTextIcon(props: LucideIconProps) {
  return (
    <LucideSvg {...props}>
      <path d="M12 7v14" />
      <path d="M16 12h2" />
      <path d="M16 8h2" />
      <path d="M3 18a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h5a4 4 0 0 1 4 4 4 4 0 0 1 4-4h5a1 1 0 0 1 1 1v13a1 1 0 0 1-1 1h-6a3 3 0 0 0-3 3 3 3 0 0 0-3-3z" />
      <path d="M6 12h2" />
      <path d="M6 8h2" />
    </LucideSvg>
  )
}

export function SettingsIcon(props: LucideIconProps) {
  return (
    <LucideSvg {...props}>
      <path d="M9.671 4.136a2.34 2.34 0 0 1 4.659 0 2.34 2.34 0 0 0 3.319 1.915 2.34 2.34 0 0 1 2.33 4.033 2.34 2.34 0 0 0 0 3.831 2.34 2.34 0 0 1-2.33 4.033 2.34 2.34 0 0 0-3.319 1.915 2.34 2.34 0 0 1-4.659 0 2.34 2.34 0 0 0-3.32-1.915 2.34 2.34 0 0 1-2.33-4.033 2.34 2.34 0 0 0 0-3.831A2.34 2.34 0 0 1 6.35 6.051a2.34 2.34 0 0 0 3.319-1.915" />
      <circle cx="12" cy="12" r="3" />
    </LucideSvg>
  )
}

export function SunMoonIcon(props: LucideIconProps) {
  return (
    <LucideSvg {...props}>
      <path d="M12 2v2" />
      <path d="M14.837 16.385a6 6 0 1 1-7.223-7.222c.624-.147.97.66.715 1.248a4 4 0 0 0 5.26 5.259c.589-.255 1.396.09 1.248.715" />
      <path d="M16 12a4 4 0 0 0-4-4" />
      <path d="m19 5-1.256 1.256" />
      <path d="M20 12h2" />
    </LucideSvg>
  )
}

export function InfoIcon(props: LucideIconProps) {
  return (
    <LucideSvg {...props}>
      <circle cx="12" cy="12" r="10" />
      <path d="M12 16v-4" />
      <path d="M12 8h.01" />
    </LucideSvg>
  )
}

export function PanelLeftCloseIcon(props: LucideIconProps) {
  return (
    <LucideSvg {...props}>
      <rect width="18" height="18" x="3" y="3" rx="2" />
      <path d="M9 3v18" />
      <path d="m16 15-3-3 3-3" />
    </LucideSvg>
  )
}

export function PanelLeftOpenIcon(props: LucideIconProps) {
  return (
    <LucideSvg {...props}>
      <rect width="18" height="18" x="3" y="3" rx="2" />
      <path d="M9 3v18" />
      <path d="m14 9 3 3-3 3" />
    </LucideSvg>
  )
}
