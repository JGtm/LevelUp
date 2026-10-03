import type { ReactNode } from 'react'
import { InfoTooltip } from '@/components/ui/info-tooltip'

/**
 * titleWithInfo — LE bandeau de titre d'une carte portant son aide ⓘ (source unique).
 *
 * Contexte (CLAUDE.md n°6, 2026-09-21) : le motif
 * `titleAdornment={(label) => <span className="flex items-center gap-1.5">{label}<InfoTooltip …/></span>}`
 * était recopié dans une douzaine de fichiers (squad, formes, match-view, match-replay,
 * timeseries, synthesis) après le passage des notes de pied en infobulle de titre
 * (lots A1 et D). Trois copies suffisaient à imposer un helper ; le garde-rail
 * `title-with-info.guard.test.ts` interdit le retour du littéral.
 *
 * `content` vaut `null` quand la carte n'a rien à réserver : le libellé se rend seul, sans
 * icône — une icône ⓘ qui n'ouvre rien ment sur la présence d'une réserve.
 */
export interface TitleWithInfoOptions {
  /** Taille de l'icône ⓘ (défaut : celle d'`InfoTooltip`). */
  iconClass?: string
  /** Contenu posé à DROITE du bandeau (compteur, boutons d'équipe) — le titre passe à gauche. */
  trailing?: ReactNode
  /** Alignement vertical du bandeau quand il porte un `trailing` (défaut : `center`). */
  align?: 'center' | 'baseline'
  /** `data-testid` sur le groupe titre+aide, pour les tests qui cherchent le bandeau. */
  testId?: string
}

/**
 * Rend la fonction attendue par `SectionCard.titleAdornment` (et les wrappers qui la
 * propagent) : `(label) => nœud`.
 */
export function titleWithInfo(
  content: ReactNode,
  options: TitleWithInfoOptions = {},
): (label: string) => ReactNode {
  const { iconClass, trailing, align = 'center', testId } = options
  return (label: string) => {
    // Le groupe « libellé + aide ⓘ » est rendu INLINE, pas par un composant local : ce
    // module n'exporte qu'une fabrique, et un composant à côté d'elle casserait le
    // rafraîchissement à chaud (règle `react-refresh/only-export-components`).
    const group = (
      <span className="flex items-center gap-1.5" data-testid={testId}>
        {label}
        {content != null && content !== false && (
          <InfoTooltip content={content} {...(iconClass ? { iconClass } : {})} />
        )}
      </span>
    )
    if (trailing == null) return group
    return (
      <span
        className={`flex flex-wrap ${align === 'baseline' ? 'items-baseline' : 'items-center'} justify-between gap-2`}
      >
        {group}
        {trailing}
      </span>
    )
  }
}
