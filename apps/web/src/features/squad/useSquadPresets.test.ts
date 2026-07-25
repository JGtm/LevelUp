/**
 * useSquadPresets.test.ts — non-régression du sous-titre « surtout … » de la
 * liste des compositions enregistrées (V72-10).
 *
 * Cible étroite : `buildUsualSubtitle` (fonction pure exportée), pas le hook
 * complet (chaîne de requêtes useMySquads/useMyGroups hors périmètre ici).
 */
import { describe, expect, it } from 'vitest'

import { buildUsualSubtitle } from './useSquadPresets'
import { SQUAD_PRESETS_STRINGS } from './squadPresets.i18n'

const T_FR = SQUAD_PRESETS_STRINGS.fr

describe('buildUsualSubtitle — indice « surtout … » (V72-10)', () => {
  it('retire la carte collée au mode brut ("Slayer on Bazaar" -> "Slayer")', () => {
    const subtitle = buildUsualSubtitle(['Ranked Arena'], ['Slayer on Bazaar'], T_FR)
    expect(subtitle).toBe('surtout Ranked Arena · Slayer')
    expect(subtitle).not.toContain('Bazaar')
  })

  it('retire la carte cross-langue (mode EN + carte FR collée, "Slayer sur Forêt")', () => {
    const subtitle = buildUsualSubtitle(undefined, ['Slayer sur Forêt'], T_FR)
    expect(subtitle).toBe('surtout Slayer')
    expect(subtitle).not.toContain('Forêt')
  })

  it('extrait le sous-mode du préfixe technique ("Arena:Slayer on Bazaar" -> "Slayer")', () => {
    const subtitle = buildUsualSubtitle([], ['Arena:Slayer on Bazaar'], T_FR)
    expect(subtitle).toBe('surtout Slayer')
  })

  it('playlists conservées telles quelles (pas de suffixe carte à retirer)', () => {
    const subtitle = buildUsualSubtitle(['Ranked Arena', 'Quick Play'], [], T_FR)
    expect(subtitle).toBe('surtout Ranked Arena · Quick Play')
  })

  it('aucune donnée -> undefined (pas de sous-titre affiché)', () => {
    expect(buildUsualSubtitle(undefined, undefined, T_FR)).toBeUndefined()
    expect(buildUsualSubtitle([], [], T_FR)).toBeUndefined()
  })

  it('seulement 1 mode gardé (slice top-1) même si plusieurs modes fournis', () => {
    const subtitle = buildUsualSubtitle([], ['Slayer on Bazaar', 'Oddball on Forge'], T_FR)
    expect(subtitle).toBe('surtout Slayer')
  })
})
