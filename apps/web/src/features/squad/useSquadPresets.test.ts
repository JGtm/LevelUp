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

  // Garde anti-disparition : le sous-titre s'était effacé quand le backend
  // renvoyait des listes vides (fenêtre de rebuild → SquadUsualContexts en erreur
  // → usual_* absents). Tant qu'une source existe (ici les playlists), le
  // sous-titre DOIT s'afficher — jamais un blanc total.
  it('modes vides mais playlists présentes -> affiche les playlists (jamais rien)', () => {
    const subtitle = buildUsualSubtitle(['Quick Play', 'Big Team Battle'], [], T_FR)
    expect(subtitle).toBe('surtout Quick Play · Big Team Battle')
  })

  // Depuis V72-10.1, l'API sert le mode déjà résolu en FR canonique
  // (mode_name_tr, « Slayer » -> « Assassin »). Le front ne doit pas le
  // re-mutiler : normalizeModeLabel est idempotent sur un libellé FR propre.
  it('mode FR résolu par l API rendu tel quel (pas de sur-normalisation)', () => {
    const subtitle = buildUsualSubtitle(['Partie rapide'], ['Assassin'], T_FR)
    expect(subtitle).toBe('surtout Partie rapide · Assassin')
  })

  it('réponse complète FR (playlists + mode FR) -> sous-titre complet', () => {
    // Valeurs alignées sur la prod : usual_modes = ["Assassin en équipe","Assassin"]
    // (Team Slayer -> Assassin en équipe, Slayer -> Assassin) après trad mode_name_tr.
    const subtitle = buildUsualSubtitle(
      ['Partie rapide', 'Baston à grande échelle'],
      ['Assassin en équipe', 'Assassin'],
      T_FR,
    )
    expect(subtitle).toBe('surtout Partie rapide · Baston à grande échelle · Assassin en équipe')
  })
})
