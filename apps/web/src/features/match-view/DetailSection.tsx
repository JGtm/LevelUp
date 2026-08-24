/**
 * DetailSection — titre de section type-1 (catalogue UI d'harmonisation, même
 * format que le Home : titre `text-base font-semibold`) + contenu groupé.
 * Structure les onglets denses (Chronologie, Joueurs) en sections lisibles.
 *
 * Extrait de MatchViewPage.tsx lors du passage à 3 onglets (2026-08-24) : partagé
 * par MatchViewTabChronology et MatchViewTabPlayers, sans cycle d'import vers la page.
 */
import type { ReactNode } from 'react'

export function DetailSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="space-y-4">
      <h3 className="text-base font-semibold text-foreground">{title}</h3>
      {children}
    </section>
  )
}
