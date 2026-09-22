/**
 * Parité des libellés du nuage des rôles : UN PRÉFIXE PAR GRANDEUR, LES MÊMES SUFFIXES.
 *
 * Ce que ce test cadenasse : ajouter une grandeur (ou un libellé) sans écrire son bloc de
 * clés dans `manifests/squad.toml` rendrait la CLÉ BRUTE à l'écran (`formatMessage` dégrade
 * sans casser). Il vérifie donc que chaque suffixe existe sous chaque préfixe, dans les deux
 * locales, et que les deux grandeurs ne se retrouvent pas avec les mêmes noms de bandes —
 * « Tireur d'élite » sur un axe de hauteur serait un contresens.
 */
import { describe, expect, it } from 'vitest'

import { squadManifest } from '@/lib/i18n/generated/squad'

import type { GrandeurProfil } from './squadRangeRoles.logic'
import {
  getSquadRangeRolesText,
  PREFIXE_GRANDEUR,
  SUFFIXES_ROLES,
} from './squadRangeRolesStrings'

const GRANDEURS: GrandeurProfil[] = ['portee', 'hauteur']

describe('squadRangeRolesStrings', () => {
  it.each(GRANDEURS)('la grandeur %s porte tous les suffixes, FR et EN', (grandeur) => {
    for (const suffixe of SUFFIXES_ROLES) {
      const cle = `${PREFIXE_GRANDEUR[grandeur]}.${suffixe}`
      const entree = (squadManifest as Record<string, { fr: string; en: string }>)[cle]
      expect(entree, cle).toBeDefined()
      expect(entree.fr.length, `${cle} fr`).toBeGreaterThan(0)
      expect(entree.en.length, `${cle} en`).toBeGreaterThan(0)
    }
  })

  it('les bandes de hauteur sont nommées Contrebas / À niveau / Hauteurs', () => {
    const t = getSquadRangeRolesText('fr', 'hauteur')
    expect(t.bandes).toEqual({ front: 'Contrebas', polyvalent: 'À niveau', sniper: 'Hauteurs' })
    const en = getSquadRangeRolesText('en', 'hauteur')
    expect(en.bandes).toEqual({
      front: 'Low ground',
      polyvalent: 'Level',
      sniper: 'High ground',
    })
  })

  it('les deux grandeurs ne partagent ni titre ni noms de bandes', () => {
    const portee = getSquadRangeRolesText('fr')
    const hauteur = getSquadRangeRolesText('fr', 'hauteur')
    expect(hauteur.cardTitle).toBe('Rôles de hauteur')
    expect(hauteur.cardTitle).not.toBe(portee.cardTitle)
    for (const role of ['front', 'polyvalent', 'sniper'] as const) {
      expect(hauteur.bandes[role]).not.toBe(portee.bandes[role])
    }
  })

  it('l’infobulle de hauteur dit la grandeur, le signe et le plancher', () => {
    const aide = getSquadRangeRolesText('fr', 'hauteur').help(5)
    expect(aide).toContain('tueur')
    expect(aide).toContain('positif')
    expect(aide).toContain('5')
    expect(aide.split('.').filter((p) => p.trim().length > 0)).toHaveLength(3)
  })
})
