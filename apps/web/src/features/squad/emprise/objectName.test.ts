import { describe, expect, it } from 'vitest'

import { USAGE_TEXT } from '@/features/_shared/usage/usageI18n'
import type { SquadEmpriseObject } from '@/lib/api/types'

import { empriseObjectName } from './objectName'

const obj = (resource: string, key: string, label?: string): SquadEmpriseObject =>
  ({ resource, key, label, taken: { us: 0, them: 0 }, squad: [] }) as SquadEmpriseObject

describe('empriseObjectName — le nom d’un objet de l’Emprise', () => {
  const t = USAGE_TEXT.fr
  it('un bonus est nommé par le web (famille du résumé d’usage)', () => {
    expect(empriseObjectName(obj('powerup', 'powerup_camo'), t, 'Inconnu')).toBe(t.metricCamo)
  })
  it('un véhicule par son libellé de titre, sinon par sa clé, et l’inconnu par le texte fourni', () => {
    expect(empriseObjectName(obj('vehicle', 'turret', 'Tourelle'), t, 'Inconnu')).toBe('Tourelle')
    expect(empriseObjectName(obj('vehicle', 'unknown'), t, 'Inconnu')).toBe('Inconnu')
  })
  it('une arme par son nom du titre, sinon par sa clé', () => {
    expect(empriseObjectName(obj('power_weapon', 'a2b3c4d5', 'Lance-roquettes'), t, 'Inconnu')).toBe('Lance-roquettes')
    expect(empriseObjectName(obj('rack', 'a2b3c4d5'), t, 'Inconnu')).toBe('a2b3c4d5')
  })
})
