/**
 * cockpit.logic — la colonne « Cartes jouées » et la carte lue d'office (plan Tactique v2, L4.2).
 *
 * Ce que ces tests cadenassent :
 *   - la recherche ignore la casse ET les accents, sur le nom affiché ET le nom canonique ;
 *   - le verdict de plancher est celui du SERVEUR (`sous_plancher`), jamais recalculé ;
 *   - la carte lue : celle de l'URL si elle est ouvrable, sinon (URL vide) la plus jouée
 *     ouvrable ; une carte d'URL hors du filtre ou sous le plancher reste nommée, sans lecture ;
 *   - le repli des cartes sous le plancher s'ouvre quand seules elles correspondent.
 */
import { describe, expect, it } from 'vitest'

import type { TacticalMapCard } from '@/lib/api/types'

import {
  carteEffective,
  colonneDesCartes,
  filtrerCartes,
  normaliserRecherche,
} from './cockpit.logic'

function carte(map_id: string, matchs: number, sous_plancher: boolean, noms: Partial<TacticalMapCard> = {}): TacticalMapCard {
  return {
    map_id,
    map_name: map_id,
    map_name_fr: '',
    matchs,
    victoires: Math.floor(matchs / 2),
    defaites: matchs - Math.floor(matchs / 2),
    sous_plancher,
    ...noms,
  }
}

const RUELLES = carte('streets', 24, false, { map_name: 'Streets', map_name_fr: 'Ruelles' })
const ECLUSE = carte('lock', 18, false, { map_name: 'Lock', map_name_fr: 'Écluse' })
const AQUARIUS = carte('aquarius', 9, true, { map_name: 'Aquarius', map_name_fr: 'Aquarius' })
const BEHEMOTH = carte('behemoth', 4, true, { map_name: 'Behemoth', map_name_fr: 'Béhémoth' })

describe('normaliserRecherche', () => {
  it('minuscules, sans accents, sans blancs autour', () => {
    expect(normaliserRecherche('  Écluse ')).toBe('ecluse')
    expect(normaliserRecherche('BÉHÉMOTH')).toBe('behemoth')
  })
})

describe('filtrerCartes', () => {
  const cartes = [RUELLES, ECLUSE, AQUARIUS]

  it('recherche vide : toutes les cartes', () => {
    expect(filtrerCartes(cartes, '   ', 'fr')).toEqual(cartes)
  })

  it('sur le nom affiché, sans casse ni accents', () => {
    expect(filtrerCartes(cartes, 'ECL', 'fr').map((c) => c.map_id)).toEqual(['lock'])
    expect(filtrerCartes(cartes, 'écl', 'fr').map((c) => c.map_id)).toEqual(['lock'])
  })

  it('sur le nom canonique aussi (en anglais, le nom affiché EST le canonique)', () => {
    expect(filtrerCartes(cartes, 'stree', 'fr').map((c) => c.map_id)).toEqual(['streets'])
    expect(filtrerCartes(cartes, 'stree', 'en').map((c) => c.map_id)).toEqual(['streets'])
    expect(filtrerCartes(cartes, 'ruel', 'en')).toEqual([])
  })
})

describe('carteEffective (D11)', () => {
  const cartes = [AQUARIUS, ECLUSE, RUELLES]

  it('URL ouvrable dans le filtre : c’est elle', () => {
    expect(carteEffective('lock', cartes)).toEqual({ mapId: 'lock', origine: 'url' })
  })

  it('URL vide : la plus jouée des cartes ouvrables', () => {
    expect(carteEffective('', cartes)).toEqual({ mapId: 'streets', origine: 'defaut' })
  })

  it('le verdict du serveur décide : une carte plus jouée mais sous le plancher n’est jamais prise', () => {
    const plusJoueeSousPlancher = carte('fragmentation', 40, true)
    expect(carteEffective('', [plusJoueeSousPlancher, ECLUSE])).toEqual({ mapId: 'lock', origine: 'defaut' })
  })

  it('URL sous le plancher ou absente du filtre : nommée, sans lecture', () => {
    expect(carteEffective('aquarius', cartes)).toEqual({ mapId: 'aquarius', origine: 'hors_filtre' })
    expect(carteEffective('inconnue', cartes)).toEqual({ mapId: 'inconnue', origine: 'hors_filtre' })
  })

  it('la liste des cartes n’a pas encore répondu : la carte de l’URL (ou aucune), en attente, sans lecture', () => {
    expect(carteEffective('lock', undefined)).toEqual({ mapId: 'lock', origine: 'attente' })
    expect(carteEffective('', undefined)).toEqual({ mapId: '', origine: 'attente' })
  })

  it('la liste des cartes a échoué : la carte de l’URL garde son fond, rien n’est lu', () => {
    expect(carteEffective('lock', null)).toEqual({ mapId: 'lock', origine: 'aucune' })
    expect(carteEffective('', null)).toEqual({ mapId: '', origine: 'aucune' })
  })

  it('aucune carte ouvrable : aucune lecture', () => {
    expect(carteEffective('', [AQUARIUS, BEHEMOTH])).toEqual({ mapId: '', origine: 'aucune' })
  })
})

describe('colonneDesCartes', () => {
  const cartes = [AQUARIUS, ECLUSE, BEHEMOTH, RUELLES]

  it('sans recherche : ouvrables par matchs décroissants, sous le plancher à part, repli fermé', () => {
    const c = colonneDesCartes(cartes, '', 'fr')
    expect(c.ouvrables.map((x) => x.map_id)).toEqual(['streets', 'lock'])
    expect(c.sousPlancher.map((x) => x.map_id)).toEqual(['aquarius', 'behemoth'])
    expect(c.totalSousPlancher).toBe(2)
    expect(c.rechercheActive).toBe(false)
    expect(c.repliOuvert).toBe(false)
    expect(c.listeVide).toBeNull()
  })

  it('seules des cartes sous le plancher correspondent : repli ouvert, liste vide dite', () => {
    const c = colonneDesCartes(cartes, 'behem', 'fr')
    expect(c.ouvrables).toEqual([])
    expect(c.sousPlancher.map((x) => x.map_id)).toEqual(['behemoth'])
    expect(c.totalSousPlancher).toBe(2)
    expect(c.rechercheActive).toBe(true)
    expect(c.repliOuvert).toBe(true)
    expect(c.listeVide).toBe('aucune_correspondance')
  })

  it('une ouvrable correspond : le repli reste fermé', () => {
    const c = colonneDesCartes(cartes, 'u', 'fr')
    expect(c.ouvrables.length).toBeGreaterThan(0)
    expect(c.repliOuvert).toBe(false)
    expect(c.listeVide).toBeNull()
  })

  it('aucune carte ouvrable du tout : la liste le dit et le repli s’ouvre', () => {
    const c = colonneDesCartes([AQUARIUS, BEHEMOTH], '', 'fr')
    expect(c.listeVide).toBe('aucune_ouvrable')
    expect(c.repliOuvert).toBe(true)
  })

  it('aucune carte : ni repli ni message de liste (l’état vide de la page le dit)', () => {
    const c = colonneDesCartes([], '', 'fr')
    expect(c.totalSousPlancher).toBe(0)
    expect(c.listeVide).toBeNull()
    expect(c.repliOuvert).toBe(false)
  })
})
