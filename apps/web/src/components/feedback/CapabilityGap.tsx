/**
 * CapabilityGap — affichage standardisé d'une fonctionnalité non disponible.
 *
 * Conformément au PLAN_META_FOUNDATIONS_GO § 3.4.5 : 3 modes possibles selon
 * le contexte UX (la décision est tracée dans `lib/capability/gap_modes.ts`) :
 *
 *   hide        : retourne `null`, la section disparaît du DOM. À utiliser
 *                 quand la section n'a aucun sens dans le contexte (ex.
 *                 battlepass pour un titre sans matchmaking).
 *   placeholder : carte avec icône + label localisé + note discrète. Utilisé
 *                 quand la donnée est attendue mais absente.
 *   cta         : placeholder + bouton d'action (lien vers doc / route de
 *                 résolution). Utilisé quand l'utilisateur peut résoudre le
 *                 manque (ex. "lancer un sync des awards").
 *
 * Le composant est i18n-naïf : les `reasonLabel` et `cta.label` sont déjà
 * résolus en amont par le consommateur (qui connaît la locale courante).
 */
import type { ReactNode } from 'react'

export type CapabilityGapMode = 'hide' | 'placeholder' | 'cta'

export interface CapabilityGapProps {
  /** Mode de rendu (cf. lib/capability/gap_modes.ts pour le mapping). */
  mode: CapabilityGapMode

  /** Label déjà localisé décrivant pourquoi la section est indisponible. */
  reasonLabel: string

  /** Note discrète optionnelle (déjà localisée). */
  hintLabel?: string

  /** Action proposée pour les modes "cta". */
  cta?: {
    /** Lien interne (`/...`) ou externe (`https://...`). */
    href: string
    /** Libellé du bouton, déjà localisé. */
    label: string
    /** True si le lien doit s'ouvrir dans un nouvel onglet. */
    external?: boolean
  }

  /** Icône optionnelle injectée dans le placeholder (ex. emoji ou SVG). */
  icon?: ReactNode

  /** ClassName additionnel pour la card racine (mode placeholder/cta). */
  className?: string
}

/**
 * CapabilityGap rend une carte standardisée pour signaler qu'une section
 * est indisponible. Mode `hide` retourne `null` (la section disparaît).
 */
export function CapabilityGap({
  mode,
  reasonLabel,
  hintLabel,
  cta,
  icon,
  className = '',
}: CapabilityGapProps) {
  if (mode === 'hide') {
    return null
  }
  return (
    <div
      className={`flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-border bg-card/40 px-6 py-8 text-center ${className}`}
      role="status"
      aria-live="polite"
      data-testid="capability-gap"
      data-mode={mode}
    >
      {icon && (
        <div className="text-muted-foreground" aria-hidden="true" data-testid="capability-gap-icon">
          {icon}
        </div>
      )}
      <div className="text-sm font-medium text-foreground" data-testid="capability-gap-reason">
        {reasonLabel}
      </div>
      {hintLabel && (
        <div className="text-xs text-muted-foreground" data-testid="capability-gap-hint">
          {hintLabel}
        </div>
      )}
      {mode === 'cta' && cta && (
        <a
          href={cta.href}
          target={cta.external ? '_blank' : undefined}
          rel={cta.external ? 'noopener noreferrer' : undefined}
          className="mt-1 inline-flex items-center rounded-md border border-border bg-card px-3 py-1.5 text-sm font-medium hover:bg-accent hover:text-accent-foreground"
          data-testid="capability-gap-cta"
        >
          {cta.label}
        </a>
      )}
    </div>
  )
}
