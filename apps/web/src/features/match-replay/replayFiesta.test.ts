/**
 * Tests — replayFiesta : LA GARDE DE MODE, et surtout ce qu'elle ne sait pas.
 *
 * La règle est CONSERVATRICE : on ne dessine les objets lâchés que si le match ne porte AUCUN
 * indice de Fiesta. Ces tests défendent les trois valeurs (`fiesta`, `clear`, `unknown`) et,
 * plus important, ils FIGENT LE TROU MESURÉ — un `pair_name` « Fiesta:Slayer on … » arrive
 * ici sous la forme du sous-mode (« Assassin »), et la garde le laisse passer. Le jour où
 * l'en-tête publiera la catégorie de mode, c'est ce test-là qui devra changer, et il dit
 * pourquoi.
 */
import { describe, expect, it } from 'vitest'

import type { MatchViewHeader } from '@/lib/api/types'

import { FIESTA_TOKENS, matchFiestaGuard } from './replayFiesta'

/**
 * Un en-tete minimal : SEULS `mode_ui` et `playlist_label` sont lus par la garde. Les deux
 * arrivent en arguments plutot qu'en `Partial` : `exactOptionalPropertyTypes` interdit de
 * repandre un `string | undefined` dans un champ declare optionnel sans `undefined`, et un
 * en-tete de test bati par etalement finirait par mentir sur ce que le serveur publie.
 */
function header(modeUi = '', playlistLabel = 'Quick Play'): MatchViewHeader {
  return {
    dominance_flag: false,
    had_bot_teammate: false,
    is_excluded: false,
    is_favorite: false,
    is_ranked: false,
    map_ui: 'Carte',
    match_id: 'm',
    outcome_color: '',
    outcome_label: '',
    performance_display: '',
    playlist_label: playlistLabel,
    replay_available: true,
    start_time_label: '',
    mode_ui: modeUi,
  }
}

describe('matchFiestaGuard — les trois verdicts', () => {
  it('un en-tête ABSENT rend `unknown` : une absence d’indice n’est pas une preuve', () => {
    expect(matchFiestaGuard(undefined)).toBe('unknown')
  })

  it('un en-tête au libellé de mode VIDE rend `clear` — il est là, il ne dit rien', () => {
    // `mode_ui` est REQUIS au contrat : il ne manque jamais, il peut être vide.
    expect(matchFiestaGuard(header(''))).toBe('clear')
  })

  it('« Super Fiesta » est reconnu — le cas de 428 matchs sur 1 855 du corpus', () => {
    expect(matchFiestaGuard(header('Super Fiesta'))).toBe('fiesta')
  })

  it('« Fiesta » nu et « Castle Wars » aussi — les deux autres préfixes de la catégorie', () => {
    expect(matchFiestaGuard(header('Fiesta'))).toBe('fiesta')
    expect(matchFiestaGuard(header('Castle Wars'))).toBe('fiesta')
  })

  it('la casse n’entre pas en jeu : les `pair_name` arrivent avec une casse inconstante', () => {
    expect(matchFiestaGuard(header('SUPER FIESTA'))).toBe('fiesta')
  })

  it('une PLAYLIST nommée Fiesta suffit, même si le mode ne dit rien', () => {
    expect(matchFiestaGuard(header('Assassin', 'Fiesta'))).toBe('fiesta')
  })

  it('un mode ordinaire rend `clear` — c’est le seul verdict qui dessine', () => {
    expect(matchFiestaGuard(header('Roi de la colline'))).toBe('clear')
    expect(matchFiestaGuard(header('Capture du drapeau'))).toBe('clear')
  })

  it('HUSKY RAID n’est PAS une Fiesta : le Go le promeut en catégorie propre', () => {
    // Ce n'est pas un mode à équipement aléatoire — ses lâchers ont le sens habituel.
    expect(matchFiestaGuard(header('Husky Raid'))).toBe('clear')
    expect(matchFiestaGuard(header('Super Husky Raid'))).toBe('clear')
  })

  it('LE TROU MESURÉ, figé ici : « Fiesta:Slayer » arrive en « Assassin » et passe', () => {
    // `NormalizeModeLabel` conserve « Super Fiesta » (identité de playlist) mais extrait le
    // SOUS-mode d'un « Fiesta:Slayer on … » — 3 matchs sur les 432 de catégorie Fiesta du
    // corpus (0,7 %). Fermer le trou demande que l'en-tête publie la catégorie de mode ou le
    // `pair_name`, côté serveur : report écrit au registre, hors périmètre de ce lot web.
    expect(matchFiestaGuard(header('Assassin'))).toBe('clear')
  })
})

describe('FIESTA_TOKENS — le vocabulaire, et sa raison d’être', () => {
  it('« fiesta » attrape « Super Fiesta » par inclusion : deux jetons suffisent', () => {
    expect(FIESTA_TOKENS).toEqual(['fiesta', 'castle wars'])
  })

  it('les jetons sont en minuscules : la comparaison abaisse la casse du libellé', () => {
    for (const t of FIESTA_TOKENS) expect(t).toBe(t.toLowerCase())
  })
})
