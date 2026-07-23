/**
 * AscensionLayers — helpers de mise en page partagés par les onglets Ascension.
 *
 * Partagés par les onglets Ascension (Profil / Objectifs / Entraînement) pour
 * éviter la duplication de mise en page entre les tabs.
 *
 * - LayerSection : en-tête de couche (titre + description, barre latérale).
 * - SectionShell : carte de section (titre en capitales + contenu).
 */
import type { ReactNode } from 'react'

// ─── Layer wrapper (Prestige / Coaching) ────────────────────────────────────

interface LayerSectionProps {
  title: string
  description: string
  children: ReactNode
}

export function LayerSection({ title, description, children }: LayerSectionProps) {
  return (
    <section className="space-y-4">
      <header className="space-y-1 border-l-2 border-primary/40 pl-3">
        <h2 className="text-lg font-bold tracking-tight">{title}</h2>
        <p className="text-sm text-muted-foreground">{description}</p>
      </header>
      <div className="space-y-6">{children}</div>
    </section>
  )
}

// ─── Section helper ──────────────────────────────────────────────────────────

export function SectionShell({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="rounded-lg border border-border bg-card p-4">
      <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-muted-foreground">
        {title}
      </h2>
      <div className="space-y-3">{children}</div>
    </section>
  )
}
