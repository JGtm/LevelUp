/**
 * applyPalette.ts — Injecte une palette sur :root via des CSS custom properties.
 *
 * Chaque token SemanticToken est écrit sous la forme `--ac-<token>`.
 * Appelé par le ThemeProvider au mount et à chaque changement de palette.
 */
import { log } from './_logger'
import { tokenVar, type Palette, type SemanticToken, ALL_TOKENS } from './semantic-tokens'

let _activePaletteKey: string | null = null

/**
 * Applique une palette sur document.documentElement.
 * Idempotent : si la même clé est appliquée deux fois, le second appel est ignoré.
 */
export function applyPalette(palette: Palette, paletteKey: string): void {
  if (_activePaletteKey === paletteKey) return

  const root = document.documentElement
  for (const token of ALL_TOKENS) {
    const value = palette[token]
    if (!value) {
      log.warn(
        `missing-token:${paletteKey}:${token}`,
        `Token manquant dans la palette "${paletteKey}" : ${token}`,
      )
      continue
    }
    if (!/^#[0-9a-fA-F]{3,8}$/.test(value)) {
      log.warn(
        `invalid-hex:${paletteKey}:${token}`,
        `Valeur invalide pour "${token}" dans "${paletteKey}" : ${value}`,
      )
    }
    root.style.setProperty(tokenVar(token as SemanticToken), value)
  }

  log.info(`Palette appliquée : ${_activePaletteKey ?? 'none'} → ${paletteKey}`)
  _activePaletteKey = paletteKey
}

/** Réinitialise l'état de déduplication (usage test uniquement). */
export function _resetActivePalette(): void {
  _activePaletteKey = null
}
