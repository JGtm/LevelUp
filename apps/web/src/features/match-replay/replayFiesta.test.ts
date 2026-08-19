/**
 * Tests — replayFiesta : LA GARDE DE MODE, et sur quoi elle corrèle.
 *
 * La règle est CONSERVATRICE : on ne dessine les objets lâchés que si le match ne porte AUCUN
 * indice de Fiesta. Ces tests défendent les trois valeurs (`fiesta`, `clear`, `unknown`), la
 * PRIORITÉ de `mode_category` (résolution canonique, Go) sur les libellés devinés
 * (`mode_ui`/`playlist_label`, repli), et FERMENT LE TROU AUTREFOIS MESURÉ : un `pair_name`
 * « Fiesta:Slayer on … » vaut mode_ui=« Assassin » (sous-mode) ET mode_category=« Fiesta »
 * (catégorie conservée) — la garde reconnaît désormais ces 3 matchs sur 432 (0,7 %).
 */
import { describe, expect, it } from 'vitest'

import type { MatchViewHeader } from '@/lib/api/types'

import { FIESTA_TOKENS, matchFiestaGuard } from './replayFiesta'

/**
 * Un en-tete minimal : `mode_ui`/`playlist_label` (repli) et `mode_category` (primaire,
 * optionnel) sont les seuls champs lus par la garde. `mode_category` est etale
 * CONDITIONNELLEMENT (jamais assigne a `undefined`) : `exactOptionalPropertyTypes` interdit de
 * repandre un `string | undefined` dans un champ declare optionnel sans `undefined`, et un
 * en-tete de test bati par etalement inconditionnel finirait par mentir sur ce que le serveur
 * publie (absent et vide ne sont pas la meme chose).
 */
function header(modeUi = '', playlistLabel = 'Quick Play', modeCategory?: string): MatchViewHeader {
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
    ...(modeCategory !== undefined ? { mode_category: modeCategory } : {}),
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

  it('SANS mode_category (repli pur), le sous-mode seul ne suffit pas — limite assumée', () => {
    // `NormalizeModeLabel` conserve « Super Fiesta » (identité de playlist) mais extrait le
    // SOUS-mode d'un « Fiesta:Slayer on … » — c'était LE TROU MESURÉ (3/432 matchs Fiesta,
    // 0,7 %) tant que l'en-tête ne publiait que mode_ui/playlist_label. Fermé pour les
    // en-têtes réels par mode_category (cf. describe suivant) ; ce cas ne survit qu'en repli
    // pur (mode_category absente), qui n'est plus le chemin nominal.
    expect(matchFiestaGuard(header('Assassin'))).toBe('clear')
  })
})

describe('matchFiestaGuard — mode_category, la corrélation canonique (pas un libellé deviné)', () => {
  it('LE TROU MESURÉ, FERMÉ : « Fiesta:Slayer » vaut mode_ui=Assassin ET mode_category=Fiesta', () => {
    // Cas réel du corpus (3/432 matchs Fiesta, 0,7 %) : NormalizeModeLabel extrait le sous-mode
    // affiché (« Assassin ») et perd l'identité playlist, mais InferModeCategoryFromPairName
    // (Go, même résolution que le filtre Mode de la galerie média) la CONSERVE dans
    // mode_category. La garde corrèle sur mode_category en premier.
    expect(matchFiestaGuard(header('Assassin', 'Quick Play', 'Fiesta'))).toBe('fiesta')
  })

  it('« Super Fiesta » en catégorie est reconnu au même titre que « Fiesta »', () => {
    expect(matchFiestaGuard(header('Assassin', 'Quick Play', 'Super Fiesta'))).toBe('fiesta')
  })

  it('mode_category PRÉSENTE et non-Fiesta décide seule : les libellés ne sont plus consultés', () => {
    // playlist_label="Fiesta" aurait suffi sous l'ancienne logique (OR pur sur les libellés) ;
    // mode_category présente tranche seule désormais — Husky Raid n'est pas une Fiesta.
    expect(matchFiestaGuard(header('Assassin', 'Fiesta', 'Husky Raid'))).toBe('clear')
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
