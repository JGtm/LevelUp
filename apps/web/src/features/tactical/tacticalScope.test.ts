/**
 * tacticalScope — le contrat d'URL de l'onglet Tactique.
 *
 * Ce que ces tests cadenassent, et ce qui casse à l'écran sans eux :
 *   - l'ALLER-RETOUR complet : un scope encodé puis décodé est le même scope. Sans
 *     lui, un lien partagé rouvre une AUTRE lecture que celle qu'on croyait envoyer ;
 *   - les valeurs par défaut ne s'écrivent PAS dans l'URL (`vue=all`, `exp=all`,
 *     listes vides) : sinon toute visite salit l'URL de paramètres neutres ;
 *   - une URL bricolée est ramenée au vocabulaire (vue/expérience inconnues) et au
 *     plafond de composition — le décodeur est une FRONTIÈRE, pas un cast.
 */
import { describe, expect, it } from 'vitest'

import {
  decodeTacticalScope,
  encodeTacticalScope,
  MAX_COEQUIPIERS,
  TACTICAL_SCOPE_DEFAUT,
  TACTICAL_URL_KEYS,
  type TacticalScope,
} from './tacticalScope'

const scopeComplet: TacticalScope = {
  debut: '2026-01-01',
  fin: '2026-02-01',
  experience: 'ranked',
  playlists: ['Ranked Arena', 'Quick Play'],
  modes: ['Slayer'],
  sessions: ['Session du 3 mars'],
  vue: 'squad',
  coequipiers: ['Ami', 'Autre'],
  carte: 'streets',
}

describe('tacticalScope', () => {
  it('aller-retour : encode puis decode rend le scope de depart', () => {
    expect(decodeTacticalScope(encodeTacticalScope(scopeComplet))).toEqual(scopeComplet)
  })

  it('aller-retour du scope VIDE : rien dans l’URL, et le defaut au retour', () => {
    const encode = encodeTacticalScope(TACTICAL_SCOPE_DEFAUT)
    for (const [cle, valeur] of Object.entries(encode)) {
      expect(valeur, `le param ${cle} ne doit pas etre ecrit a vide`).toBeUndefined()
    }
    expect(decodeTacticalScope(encode)).toEqual(TACTICAL_SCOPE_DEFAUT)
  })

  it('une URL vide rend le scope par defaut', () => {
    expect(decodeTacticalScope({})).toEqual(TACTICAL_SCOPE_DEFAUT)
  })

  it('vue et experience hors vocabulaire retombent sur « all »', () => {
    const s = decodeTacticalScope({ vue: 'tout-le-monde', exp: 'legendaire' })
    expect(s.vue).toBe('all')
    expect(s.experience).toBe('all')
  })

  it('la composition est plafonnee A LA LECTURE — une URL bricolee ne passe pas', () => {
    const s = decodeTacticalScope({ eq: 'A,B,C,D,E' })
    expect(s.coequipiers).toHaveLength(MAX_COEQUIPIERS)
    expect(s.coequipiers).toEqual(['A', 'B', 'C'])
  })

  // W11 — LA LISTE DES CLES D'URL EST COMPAREE A QUELQUE CHOSE.
  //
  // `TACTICAL_URL_KEYS` sert a `usePageScope` pour deux choses : detecter un
  // atterrissage « vierge » (sinon le miroir localStorage ECRASE un lien partage qui ne
  // porte que la cle oubliee) et purger le scope au reset. Une cle qui manque a la liste
  // ne casse aucun test d'aller-retour — elle casse le partage d'URL, en silence.
  it('TACTICAL_URL_KEYS couvre TOUTES les cles encodables', () => {
    // L'encodage d'un scope COMPLET produit exactement les cles du contrat.
    const clesEncodees = Object.keys(encodeTacticalScope(scopeComplet)).sort()
    expect([...TACTICAL_URL_KEYS].sort()).toEqual(clesEncodees)
  })

  it('les listes vides ne laissent aucun param orphelin', () => {
    const encode = encodeTacticalScope({ ...scopeComplet, playlists: [], sessions: [] })
    expect(encode.pl).toBeUndefined()
    expect(encode.ses).toBeUndefined()
    // …et les autres restent.
    expect(encode.md).toBe('Slayer')
  })
})
