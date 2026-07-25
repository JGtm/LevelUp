/**
 * useSquadPresets.test.ts — non-régression du sous-titre « surtout … » de la
 * liste des compositions enregistrées (V72-10).
 *
 * Cible étroite : `buildUsualSubtitle` (fonction pure exportée), pas le hook
 * complet (chaîne de requêtes useMySquads/useMyGroups hors périmètre ici).
 */
import { describe, expect, it } from 'vitest'

import {
  buildUsualSubtitle,
  buildActiveContextKeys,
  scoreSquadContext,
} from './useSquadPresets'
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

/**
 * Tri souple des compositions : le mode usuel FR servi par l'API doit matcher le
 * libellé du filtre actif. Constat 2026-07-25 : la comparaison se faisait sur les
 * chaînes BRUTES (simple toLowerCase), alors que `buildUsualSubtitle` normalisait
 * déjà — tout suffixe de carte ou préfixe technique résiduel d'un seul côté
 * cassait le match en silence.
 */
describe('scoreSquadContext — matching contexte actif ↔ contextes habituels', () => {
  it('mode usuel FR identique au filtre actif -> matche', () => {
    const keys = buildActiveContextKeys(['Assassin en équipe'])
    expect(scoreSquadContext({ usual_modes: ['Assassin en équipe'] }, keys)).toBe(1)
  })

  it('mode usuel FR avec carte collée -> matche quand même (normalisation)', () => {
    const keys = buildActiveContextKeys(['Assassin en équipe'])
    expect(scoreSquadContext({ usual_modes: ['Assassin en équipe sur Bazaar'] }, keys)).toBe(1)
  })

  it('préfixe technique côté filtre actif -> matche le mode usuel propre', () => {
    const keys = buildActiveContextKeys(['Arena:Slayer on Bazaar'])
    expect(scoreSquadContext({ usual_modes: ['Slayer'] }, keys)).toBe(1)
  })

  it('casse différente -> matche (comparaison insensible à la casse)', () => {
    const keys = buildActiveContextKeys(['ASSASSIN'])
    expect(scoreSquadContext({ usual_modes: ['Assassin'] }, keys)).toBe(1)
  })

  it('playlists et modes cumulent le score', () => {
    const keys = buildActiveContextKeys(['Partie rapide', 'Assassin en équipe'])
    const score = scoreSquadContext(
      { usual_playlists: ['Partie rapide'], usual_modes: ['Assassin en équipe'] },
      keys,
    )
    expect(score).toBe(2)
  })

  it('aucun filtre actif -> score 0 (ordre d origine préservé)', () => {
    const keys = buildActiveContextKeys([])
    expect(scoreSquadContext({ usual_modes: ['Assassin'] }, keys)).toBe(0)
    expect(buildActiveContextKeys(undefined).size).toBe(0)
  })

  it('contexte sans rapport -> score 0 (pas de faux positif)', () => {
    const keys = buildActiveContextKeys(['Capture du drapeau'])
    expect(scoreSquadContext({ usual_modes: ['Assassin en équipe'] }, keys)).toBe(0)
  })

  it('composition sans contextes habituels -> score 0', () => {
    const keys = buildActiveContextKeys(['Assassin'])
    expect(scoreSquadContext({}, keys)).toBe(0)
  })
})
