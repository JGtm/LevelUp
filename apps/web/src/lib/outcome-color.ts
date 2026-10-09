/**
 * outcome-color.ts — Wrappers de compatibilité vers la couche accessibilité.
 *
 * Seuils canoniques : instances.ts (outcomeScale)
 */
import { outcomeScale } from './accessibility/scales'
import { tokenCssVar } from './accessibility'
import type { SemanticToken } from './accessibility/semantic-tokens'

export const OUTCOME_COLORS = {
  win:  tokenCssVar('outcome-win'),
  loss: tokenCssVar('outcome-loss'),
  draw: tokenCssVar('outcome-draw'),
  dnf:  tokenCssVar('outcome-dnf'),
} as const

export type OutcomeKey = 'win' | 'loss' | 'draw' | 'dnf'

const OUTCOME_KEY: Record<number, OutcomeKey> = {
  2: 'win',
  1: 'draw',
  3: 'loss',
}

/**
 * Mappe un code outcome numérique Halo (2=win, 3=loss, 1=draw, autre/4=dnf) vers sa
 * clé canonique. Source UNIQUE — ne pas redéfinir localement (revue 2026-06-02).
 */
export function outcomeKey(outcome: number): OutcomeKey {
  return OUTCOME_KEY[outcome] ?? 'dnf'
}

/** Retourne la CSS var correspondant à un outcome numérique Halo. */
export function getOutcomeColor(outcome: number): string {
  const key = OUTCOME_KEY[outcome] ?? 'dnf'
  return tokenCssVar(outcomeScale(key) ?? 'outcome-dnf')
}

/**
 * Clé CANONIQUE d'issue (`canonical.Outcome` côté Go, clés d'`outcomes.toml` :
 * win | loss | tie | dnf) -> jeton de couleur sémantique.
 *
 * POURQUOI ELLE NE SE CONFOND PAS AVEC `outcomeKey` : les échelles d'accessibilité
 * nomment l'égalité `draw`, le contrat canonique la nomme `tie` (cf. l'en-tête de
 * `lib/outcome.ts`). La traduction vit ICI, une seule fois — un composant qui la
 * recopierait re-diverge au premier jeu de clés oublié.
 */
const CANONICAL_TO_TOKEN: Readonly<Record<string, SemanticToken>> = {
  win: 'outcome-win',
  loss: 'outcome-loss',
  tie: 'outcome-draw',
  dnf: 'outcome-dnf',
}

/** Jeton de couleur d'une issue canonique, ou `null` si la clé n'est pas au contrat. */
export function outcomeTokenFromCanonical(key: string): SemanticToken | null {
  return CANONICAL_TO_TOKEN[key] ?? null
}
