/**
 * resolveToken.ts — Résout un SemanticToken vers sa couleur hex active.
 *
 * Lit la CSS custom property `--ac-<token>` sur :root.
 * Usage réservé aux contextes non-CSS : layouts Plotly, canvas, SVG inline.
 *
 * Pour les composants React, préférer `tokenCssVar(token)` directement dans
 * `style={{ color: tokenCssVar('outcome-win') }}` — plus performant, réactif
 * automatiquement au changement de palette sans re-render.
 */
import { log } from './_logger'
import { tokenVar, type SemanticToken } from './semantic-tokens'

/**
 * Retourne la couleur hex résolue pour un token dans la palette active.
 * Doit être appelé après que `applyPalette()` ait été exécuté.
 */
export function resolveToken(token: SemanticToken): string {
  if (typeof document === 'undefined') {
    log.warn(`ssr:${token}`, `resolveToken appelé côté serveur pour "${token}" — retourne chaîne vide`)
    return ''
  }

  const value = getComputedStyle(document.documentElement)
    .getPropertyValue(tokenVar(token))
    .trim()

  if (!value) {
    log.error(
      `unresolved:${token}`,
      `Token "${token}" non résolu (CSS var absente). applyPalette() a-t-il été appelé ?`,
    )
    return ''
  }

  return value
}
