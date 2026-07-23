/**
 * Route /players/$playerSlug/ascension — index (onglet "Profil").
 *
 * Restructuration 4 onglets (2026-07, DEC-3) : l'index rend l'onglet Profil
 * (identité/style/performance + patterns). La couche Prestige (objectifs/arcs)
 * a migré vers /ascension/objectifs.
 */
import { createFileRoute } from '@tanstack/react-router'
import { AscensionProfilTab } from '@/features/ascension/AscensionProfilTab'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/ascension/')({
  component: AscensionProfilTab,
})
