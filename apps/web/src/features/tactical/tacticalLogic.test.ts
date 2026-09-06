/**
 * La logique PURE de la grille des cartes.
 *
 * Ce que ces tests cadenassent :
 *   - le tri par nombre de matchs, et son départage STABLE par le nom ;
 *   - l'état sous plancher lu sur le VERDICT DU SERVEUR, aux deux bornes (9 et 10) ;
 *   - la barre victoires / défaites quand V + D est INFÉRIEUR aux matchs (le cas nominal :
 *     nuls, abandons, résultat inconnu) — la barre ne doit jamais remplir le cadre ;
 *   - les deux protections d'affichage (zéro match, contrat incohérent) ;
 *   - le filtre envoyé au serveur, dont la session est retirée (T11).
 */
import { describe, expect, it } from 'vitest'

import type { TacticalMapCard } from '@/lib/api/types'

import {
  barreResultats,
  couvertureGrille,
  estOuvrable,
  nomCarte,
  tacticalFilterQuery,
  trierCartes,
} from './tacticalLogic'

function carte(over: Partial<TacticalMapCard> = {}): TacticalMapCard {
  return {
    map_id: 'm',
    map_name: 'Map',
    map_name_fr: '',
    matchs: 10,
    victoires: 5,
    defaites: 5,
    sous_plancher: false,
    ...over,
  }
}

describe('trierCartes', () => {
  it('classe de la carte la plus jouée à la moins jouée', () => {
    const tri = trierCartes([
      carte({ map_id: 'a', map_name: 'Aquarius', matchs: 4 }),
      carte({ map_id: 'b', map_name: 'Bazaar', matchs: 31 }),
      carte({ map_id: 'c', map_name: 'Catalyst', matchs: 12 }),
    ])
    expect(tri.map((c) => c.map_id)).toEqual(['b', 'c', 'a'])
  })

  it('départage par le NOM à nombre de matchs égal — deux affichages, un seul ordre', () => {
    const tri = trierCartes([
      carte({ map_id: 'z', map_name: 'Streets', matchs: 10 }),
      carte({ map_id: 'a', map_name: 'Aquarius', matchs: 10 }),
      carte({ map_id: 'm', map_name: 'Live Fire', matchs: 10 }),
    ])
    expect(tri.map((c) => c.map_name)).toEqual(['Aquarius', 'Live Fire', 'Streets'])
  })

  it("ne filtre RIEN : une carte sous le plancher reste dans la liste", () => {
    const tri = trierCartes([
      carte({ map_id: 'a', matchs: 20 }),
      carte({ map_id: 'b', matchs: 2, sous_plancher: true }),
    ])
    expect(tri).toHaveLength(2)
    expect(tri[1].map_id).toBe('b')
  })

  it("ne mute pas la liste reçue", () => {
    const source = [carte({ map_id: 'a', matchs: 1 }), carte({ map_id: 'b', matchs: 9 })]
    trierCartes(source)
    expect(source.map((c) => c.map_id)).toEqual(['a', 'b'])
  })
})

describe('estOuvrable — les deux bornes du plancher', () => {
  // Le seuil est décidé PAR LE SERVEUR (`sous_plancher`) : le client ne le recalcule pas.
  // Ces deux cas posent la borne exacte telle que le contrat la sert.
  it('9 matchs, sous le plancher de 10 : la carte ne s’ouvre pas', () => {
    expect(estOuvrable(carte({ matchs: 9, sous_plancher: true }))).toBe(false)
  })

  it('10 matchs pile, plancher atteint : la carte s’ouvre', () => {
    expect(estOuvrable(carte({ matchs: 10, sous_plancher: false }))).toBe(true)
  })
})

describe('barreResultats', () => {
  it('V + D INFÉRIEUR aux matchs : le reste est du neutre, jamais du V ni du D', () => {
    const parts = barreResultats(carte({ matchs: 10, victoires: 6, defaites: 3 }))
    expect(parts.victoires).toBeCloseTo(0.6)
    expect(parts.defaites).toBeCloseTo(0.3)
    expect(parts.autres).toBeCloseTo(0.1)
  })

  it('V + D égal aux matchs : aucun reste', () => {
    const parts = barreResultats(carte({ matchs: 8, victoires: 5, defaites: 3 }))
    expect(parts.victoires + parts.defaites).toBeCloseTo(1)
    expect(parts.autres).toBeCloseTo(0)
  })

  it('aucun match : barre vide, jamais une division par zéro', () => {
    const parts = barreResultats(carte({ matchs: 0, victoires: 0, defaites: 0 }))
    expect(parts).toEqual({ victoires: 0, defaites: 0, autres: 0 })
    expect(Number.isNaN(parts.victoires)).toBe(false)
  })

  it('contrat incohérent (V + D > matchs) : la barre ne déborde pas du cadre', () => {
    const parts = barreResultats(carte({ matchs: 4, victoires: 5, defaites: 3 }))
    expect(parts.victoires + parts.defaites + parts.autres).toBeLessThanOrEqual(1.0000001)
    expect(parts.victoires).toBeCloseTo(5 / 8)
    expect(parts.defaites).toBeCloseTo(3 / 8)
  })

  it('valeurs négatives : ramenées à zéro plutôt que peintes à l’envers', () => {
    const parts = barreResultats(carte({ matchs: 10, victoires: -3, defaites: 4 }))
    expect(parts.victoires).toBe(0)
    expect(parts.defaites).toBeCloseTo(0.4)
  })
})

describe('couvertureGrille', () => {
  it('compte les cartes ET la somme de leurs matchs', () => {
    expect(
      couvertureGrille([carte({ matchs: 12 }), carte({ matchs: 3 }), carte({ matchs: 30 })]),
    ).toEqual({ cartes: 3, matchs: 45 })
  })

  it('grille vide : deux zéros', () => {
    expect(couvertureGrille([])).toEqual({ cartes: 0, matchs: 0 })
  })
})

describe('nomCarte', () => {
  it('le français vient du contrat', () => {
    expect(nomCarte(carte({ map_name: 'Streets', map_name_fr: 'Ruelles' }), 'fr')).toBe('Ruelles')
  })

  it('nom FR vide : on retombe sur le nom canonique plutôt que sur du blanc', () => {
    expect(nomCarte(carte({ map_name: 'Streets', map_name_fr: '   ' }), 'fr')).toBe('Streets')
  })

  it('en anglais, le nom canonique', () => {
    expect(nomCarte(carte({ map_name: 'Streets', map_name_fr: 'Ruelles' }), 'en')).toBe('Streets')
  })

  it('aucun nom : le map_id plutôt qu’une vignette anonyme', () => {
    expect(nomCarte(carte({ map_id: 'asset-42', map_name: '', map_name_fr: '' }), 'fr')).toBe(
      'asset-42',
    )
  })
})

describe('tacticalFilterQuery', () => {
  it('traduit le contexte global dans le vocabulaire de l’Explorateur', () => {
    const q = tacticalFilterQuery({
      cascade: { playlists: ['Ranked Arena'], modes: ['Slayer'] },
      period: { start_date: '2026-01-01', end_date: '2026-02-01' },
    } as never)
    const params = new URLSearchParams(q)
    expect(params.get('playlist')).toBe('Ranked Arena')
    expect(params.get('mode')).toBe('Slayer')
    expect(params.get('from')).toBe('2026-01-01T00:00:00Z')
    expect(params.get('to')).toBe('2026-02-01T23:59:59Z')
  })

  it('RETIRE la session : l’onglet ne l’honore pas, donc ne la demande pas (T11)', () => {
    const q = tacticalFilterQuery({
      filter_mode: 'sessions',
      sessions: { picked_session_label: 'Session du 3 mars' },
      cascade: { playlists: ['Ranked Arena'] },
    } as never)
    expect(q).not.toContain('session')
    expect(new URLSearchParams(q).get('playlist')).toBe('Ranked Arena')
  })

  it('aucun filtre : chaîne vide (et donc aucune interrogation superflue)', () => {
    expect(tacticalFilterQuery(null)).toBe('')
    expect(tacticalFilterQuery({} as never)).toBe('')
  })
})
