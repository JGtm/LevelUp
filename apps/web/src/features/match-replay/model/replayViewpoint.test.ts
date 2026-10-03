/**
 * Tests — resolveViewpoint : par les yeux de qui la page de rejeu se regarde.
 *
 * CE QU'ILS PROTÈGENT. La règle tient en trois lignes, et chacune évite une panne muette :
 * une sélection qui ne correspond à personne rendrait TOUTE la page neutre (aucun camp, aucune
 * marque, aucun écran de fin) sans lever la moindre erreur ; un défaut mal placé ferait ouvrir
 * un match par les yeux de quelqu'un d'autre. Le repli est donc explicite et testé.
 */
import { describe, expect, it } from 'vitest'

import { resolveViewpoint, type ViewpointRow } from './replayViewpoint'

/** Le lobby témoin : une ligne « moi », un coéquipier, un adversaire, un bot. */
const BOARD: ViewpointRow[] = [
  { xuid: 'me-1', is_me: true },
  { xuid: 'ally-2', is_me: false },
  { xuid: 'foe-1', is_me: false },
  { xuid: 'bid(1.0)', is_me: false },
]

/** Le même lobby sans ligne « moi » : plus aucun défaut à offrir. */
const BOARD_SANS_MOI: ViewpointRow[] = BOARD.map((r) => ({ ...r, is_me: false }))

describe('resolveViewpoint', () => {
  it('sans sélection : le joueur de la page (décision 8 — le défaut, à chaque montage)', () => {
    expect(resolveViewpoint(BOARD, null)).toBe('me-1')
  })

  it('sélection présente au tableau de score : c’est elle, y compris un adversaire', () => {
    expect(resolveViewpoint(BOARD, 'foe-1')).toBe('foe-1')
  })

  it('un bot est un point de vue comme un autre (décision 7)', () => {
    expect(resolveViewpoint(BOARD, 'bid(1.0)')).toBe('bid(1.0)')
  })

  it('sélection ABSENTE du tableau de score : repli sur le joueur de la page', () => {
    // Le cas d'un tableau rechargé pendant qu'une sélection survit au re-rendu. Sans ce repli
    // le point de vue vaudrait un joueur inexistant : aucun camp allié nulle part, en silence.
    expect(resolveViewpoint(BOARD, 'xuid-jamais-vu')).toBe('me-1')
  })

  it('sélection absente ET aucune ligne « moi » : null — personne n’est deviné', () => {
    expect(resolveViewpoint(BOARD_SANS_MOI, 'xuid-jamais-vu')).toBeNull()
  })

  it('aucune ligne « moi », aucune sélection : null', () => {
    expect(resolveViewpoint(BOARD_SANS_MOI, null)).toBeNull()
  })

  it('sélection valide alors qu’aucune ligne « moi » n’existe : la sélection l’emporte', () => {
    expect(resolveViewpoint(BOARD_SANS_MOI, 'foe-1')).toBe('foe-1')
  })

  it('tableau de score absent : null, jamais une exception', () => {
    expect(resolveViewpoint(null, null)).toBeNull()
    expect(resolveViewpoint(undefined, 'me-1')).toBeNull()
    expect(resolveViewpoint([], 'me-1')).toBeNull()
  })

  it('une sélection vide vaut absence de sélection', () => {
    expect(resolveViewpoint(BOARD, '')).toBe('me-1')
  })
})
