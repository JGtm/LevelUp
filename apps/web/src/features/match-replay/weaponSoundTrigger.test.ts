import { describe, expect, it } from 'vitest'

import { MAX_SONS_PAR_PAS, tirsAJouer, weaponFamilyKey } from './weaponSoundTrigger'

describe('weaponFamilyKey', () => {
  it('reduit un identifiant global 64 bits a sa moitie haute', () => {
    // Miroir d'identity.go : famille = high-32 d'un identifiant long.
    expect(weaponFamilyKey('0xd7915565aabbccdd')).toBe('d7915565')
  })

  it('rend telle quelle une famille de 8 chiffres, prefixee ou non', () => {
    expect(weaponFamilyKey('0x00008595')).toBe('00008595')
    expect(weaponFamilyKey('D7915565')).toBe('d7915565')
  })

  it('respecte le seuil de longueur du serveur, prefixe compris', () => {
    // 10 caracteres AVEC prefixe : le serveur lit la valeur basse, pas la moitie haute.
    expect(weaponFamilyKey('0x12345678')).toBe('12345678')
  })

  it('rend null pour tout ce qui ne se lit pas', () => {
    expect(weaponFamilyKey(undefined)).toBeNull()
    expect(weaponFamilyKey('')).toBeNull()
    expect(weaponFamilyKey('pistolet')).toBeNull()
    expect(weaponFamilyKey('0x' + 'f'.repeat(17))).toBeNull()
  })
})

describe('tirsAJouer', () => {
  const tirs = [
    { t: 10, w: '0x00008595' },
    { t: 20, w: '0xd7915565aabbccdd' },
    { t: 30, w: undefined },
    { t: 40, w: '0x00008595' },
  ]

  it('rend les tirs franchis par la fenetre (avant, courant], dans l ordre du film', () => {
    expect(tirsAJouer(tirs, 5, 25, 100)).toEqual(['00008595', 'd7915565'])
  })

  it('exclut la borne basse et inclut la haute', () => {
    expect(tirsAJouer(tirs, 10, 20, 100)).toEqual(['d7915565'])
  })

  it('ignore un tir sans arme lisible plutot que d emprunter un son', () => {
    expect(tirsAJouer(tirs, 25, 35, 100)).toEqual([])
  })

  it('ne joue rien en arriere ni sur place — retour de boucle et curseur couverts', () => {
    expect(tirsAJouer(tirs, 40, 0, 100)).toEqual([])
    expect(tirsAJouer(tirs, 20, 20, 100)).toEqual([])
  })

  it('ne joue rien sur un bond superieur a maxAvance (onglet suspendu, saut de curseur)', () => {
    expect(tirsAJouer(tirs, 0, 50, 15)).toEqual([])
  })

  it('borne le nombre de sons par pas', () => {
    const rafale = Array.from({ length: 20 }, (_, i) => ({ t: i + 1, w: '0x00008595' }))
    expect(tirsAJouer(rafale, 0, 20, 100)).toHaveLength(MAX_SONS_PAR_PAS)
  })
})
