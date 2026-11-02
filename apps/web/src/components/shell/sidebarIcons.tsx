/**
 * Icônes de la barre latérale de l'application de bureau : une icône Lucide par section de navigation
 * (`L1_SECTIONS`, clé de section), dépliée comme repliée. Une section inconnue n'a pas d'icône : la barre
 * repliée y met l'initiale de son libellé.
 */
import type { ComponentType, ReactNode } from 'react'
import {
  AwardIcon,
  CrosshairIcon,
  FilmIcon,
  FlameIcon,
  GlobeIcon,
  LayoutDashboardIcon,
  SearchIcon,
  SwordsIcon,
  type LucideIconProps,
} from '@/components/icons/lucide'

const SECTION_ICONS: Readonly<Record<string, ComponentType<LucideIconProps>>> = {
  home: LayoutDashboardIcon,
  stats: CrosshairIcon,
  squad: SwordsIcon,
  career: AwardIcon,
  ascension: FlameIcon,
  community: GlobeIcon,
  media: FilmIcon,
  explorer: SearchIcon,
}

/** L'icône d'une section de navigation ; pour une clé inconnue, `fallback` (rien par défaut). */
export function SectionIcon({
  sectionKey,
  className,
  fallback = null,
}: {
  sectionKey: string
  className?: string
  fallback?: ReactNode
}) {
  const Icon = Object.hasOwn(SECTION_ICONS, sectionKey) ? SECTION_ICONS[sectionKey] : null
  return Icon ? <Icon className={className} /> : fallback
}
