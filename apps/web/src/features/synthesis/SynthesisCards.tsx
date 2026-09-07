/**
 * SynthesisCards — LES DEUX BRIQUES DE MISE EN PAGE DE LA SYNTHÈSE, extraites pour être
 * partagées.
 *
 * `AccentCard` (la tuile de KPI à filet coloré) et `SectionSubtitle` (le sous-titre à filet)
 * vivaient dans `SynthesisPage.tsx`, seul consommateur jusqu'au 2026-09-06. La section
 * « Portée des engagements » (lot 5 du plan `.ai/PLAN_DUELS_PORTEE_2026-09-06.md`) est un
 * composant à part et en a besoin : les recopier en aurait fait deux gabarits pour la même
 * chose dans la même page — exactement ce que `SectionCard` a corrigé ailleurs. Elles sont
 * donc DÉPLACÉES ici (aucune modification de rendu), et `SynthesisPage` les importe.
 *
 * L'import inverse (la section important depuis la page qui la monte) aurait fermé un cycle :
 * un module tiers est la seule forme qui n'en crée pas.
 */
import type { ReactNode } from 'react'

import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'

/** Sous-titre de section (type 6 du catalogue) : petit uppercase semibold + filet 1px. */
export function SectionSubtitle({ children }: { children: ReactNode }) {
  return (
    <div className="space-y-2">
      <p className="text-3xs font-semibold uppercase tracking-label-md text-foreground/90">{children}</p>
      <div className="h-px w-full rounded-full bg-border" />
    </div>
  )
}

export interface AccentCardProps {
  label: string
  value: string
  accent: SemanticToken
  /**
   * Ligne de contexte sous la valeur — le DÉNOMINATEUR d'une mesure partielle
   * (« 1 214 frags mesurés sur 1 602 »). Absente = tuile inchangée : un chiffre sans
   * dénominateur se lit comme un total, ce qui est faux dès que la couverture est partielle.
   */
  sub?: string
  onOpenMatch?: () => void
  openMatchLabel?: string
}

export function AccentCard({ label, value, accent, sub, onOpenMatch, openMatchLabel }: AccentCardProps) {
  return (
    <div className="rounded-lg overflow-hidden border border-border bg-card">
      <div className="h-[3px]" style={{ backgroundColor: tokenCssVar(accent) }} />
      <div className="p-3">
        <div className="flex items-center gap-1.5">
          <span className="text-xs text-muted-foreground block">{label}</span>
          {onOpenMatch && (
            <button
              type="button"
              onClick={onOpenMatch}
              aria-label={openMatchLabel}
              className="group flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
            >
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="currentColor" className="h-3 w-3 opacity-50 group-hover:opacity-100 transition-opacity" aria-hidden="true">
                <path d="M6.22 8.72a.75.75 0 0 0 1.06 1.06l5.22-5.22v1.69a.75.75 0 0 0 1.5 0v-3.5a.75.75 0 0 0-.75-.75h-3.5a.75.75 0 0 0 0 1.5h1.69L6.22 8.72Z" />
                <path d="M3.5 6.75c0-.69.56-1.25 1.25-1.25H7A.75.75 0 0 0 7 4H4.75A2.75 2.75 0 0 0 2 6.75v4.5A2.75 2.75 0 0 0 4.75 14h4.5A2.75 2.75 0 0 0 12 11.25V9a.75.75 0 0 0-1.5 0v2.25c0 .69-.56 1.25-1.25 1.25h-4.5c-.69 0-1.25-.56-1.25-1.25v-4.5Z" />
              </svg>
            </button>
          )}
        </div>
        <span className="text-xl font-bold">{value}</span>
        {sub && <span className="mt-0.5 block text-3xs tabular-nums text-muted-foreground">{sub}</span>}
      </div>
    </div>
  )
}
