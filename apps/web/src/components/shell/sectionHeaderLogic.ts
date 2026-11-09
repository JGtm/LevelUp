/**
 * Logique pure de `SectionHeader` : quelle section de `L1_SECTIONS` pose son en-tête pour un chemin.
 */
import { hasCapabilityIn } from '@/lib/capabilities/capabilities'
import { L1_SECTIONS, type L1Section } from './navL1Sections'

/**
 * La section dont l'en-tête est rendu ici, ou `null` quand le layout joueur n'en pose pas.
 * `titleCaps` : capacités du titre courant (`null` : pas encore chargées, tout est permis).
 */
export function sectionWithHeader(
  sections: readonly L1Section[],
  pathname: string,
  titleCaps: readonly string[] | null,
): L1Section | null {
  // Réglage `show_progression` coupé : la section est absente de la navigation mais son URL reste
  // ouvrable, avec son en-tête. Le repli respecte la capacité de la section : un titre qui ne la
  // déclare pas n'affiche ni titre ni onglets au-dessus de « fonction indisponible ».
  const section =
    sections.find((s) => s.matchPathname(pathname)) ??
    L1_SECTIONS.find(
      (s) => s.matchPathname(pathname) && (s.capability == null || hasCapabilityIn(titleCaps, s.capability)),
    )
  if (!section || section.leadKey == null || section.headerInLayout) return null
  return section
}
