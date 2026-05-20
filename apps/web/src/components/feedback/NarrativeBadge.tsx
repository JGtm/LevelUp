/**
 * NarrativeBadge — rendu unifié des badges narratifs.
 *
 * Sources possibles (cf. canonical/narrative côté Go) :
 *
 *   - DominanceBadge   : flag dominance (1..5) + LabelKey + ColorToken
 *   - EncounterBadge   : kind (ally_plus / tough_enemy / ordinal) + détail
 *   - ImpactRole       : 8 rôles d'impact (silent_hero, top_killer, etc.)
 *
 * Le composant est i18n-naïf : `label` est déjà résolu en amont via le
 * manifest narrative.toml. `colorToken` est résolu via tokenCssVar() côté
 * consommateur (le badge applique juste la couleur en style inline).
 *
 * `inverted` (ImpactRole et LastGroupKill loss) inverse le contraste :
 * background tinté plus sombre + foreground décalé pour signaler le rôle
 * "négatif".
 */
import type { CSSProperties } from 'react'

export interface NarrativeBadgeProps {
  /** Label déjà localisé via formatMessage(narrativeManifest, key, locale). */
  label: string

  /** Variable CSS (ex. "--narrative-dominance-win-strong") résolue par le consommateur. */
  colorVar?: string

  /** Inverse le contraste (rôles négatifs : LastCasualty, FalseBrother…). */
  inverted?: boolean

  /** Détail à afficher en suffixe (ex. "(5×)" pour ordinal). */
  detailSuffix?: string

  /** Taille du badge. */
  size?: 'sm' | 'md' | 'lg'

  /** Tooltip natif (déjà localisé). */
  title?: string

  /** ClassName additionnel. */
  className?: string
}

const SIZE_CLASSES: Record<NonNullable<NarrativeBadgeProps['size']>, string> = {
  sm: 'text-2xs px-1.5 py-0.5',
  md: 'text-xs px-2 py-1',
  lg: 'text-xs px-3 h-9',
}

/**
 * NarrativeBadge — rendu unifié. La couleur est appliquée via une variable
 * CSS pour respecter la règle CLAUDE.md §20 (pas de hex/Tailwind hardcodé
 * dans les composants).
 */
export function NarrativeBadge({
  label,
  colorVar,
  inverted = false,
  detailSuffix,
  size = 'md',
  title,
  className = '',
}: NarrativeBadgeProps) {
  const style: CSSProperties = {}
  if (colorVar) {
    if (inverted) {
      // Mode inversé : background tinté faible + texte couleur pleine.
      style.backgroundColor = `color-mix(in oklab, var(${colorVar}) 18%, transparent)`
      style.color = `var(${colorVar})`
      style.borderColor = `color-mix(in oklab, var(${colorVar}) 35%, transparent)`
    } else {
      // Mode standard : background tinté médian + texte sombre.
      style.backgroundColor = `color-mix(in oklab, var(${colorVar}) 32%, transparent)`
      style.color = `var(${colorVar})`
      style.borderColor = `color-mix(in oklab, var(${colorVar}) 48%, transparent)`
    }
  }
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full border font-medium leading-none ${SIZE_CLASSES[size]} ${className}`}
      style={style}
      data-testid="narrative-badge"
      data-inverted={inverted ? 'true' : 'false'}
      title={title}
    >
      <span data-testid="narrative-badge-label">{label}</span>
      {detailSuffix && (
        <span data-testid="narrative-badge-suffix" className="opacity-80">
          {detailSuffix}
        </span>
      )}
    </span>
  )
}
