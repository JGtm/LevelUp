/**
 * i18n.test.ts — Parité FR / EN du dictionnaire Escouade.
 *
 * Test load-bearing : tout ajout d'une clé en FR doit avoir son équivalent
 * EN, et inversement. Aucune autre feature n'a ce test aujourd'hui — on
 * l'introduit ici comme pattern réutilisable.
 *
 * On compare la structure profonde des deux objets (clés uniquement, pas
 * les valeurs). Les valeurs string vs function sont validées par leur
 * type pour rester strictes.
 */
import { describe, it, expect } from 'vitest'
import { FR_TEXT, EN_TEXT, getSquadText } from './i18n'

type Shape = { [k: string]: 'string' | 'function' | Shape }

function shapeOf(obj: unknown): Shape | 'string' | 'function' {
  if (typeof obj === 'string') return 'string'
  if (typeof obj === 'function') return 'function'
  if (obj === null || typeof obj !== 'object') {
    throw new Error(`Type non supporté dans le dict i18n : ${typeof obj}`)
  }
  const out: Shape = {}
  for (const [k, v] of Object.entries(obj as Record<string, unknown>)) {
    out[k] = shapeOf(v) as Shape | 'string' | 'function'
  }
  return out
}

describe('SquadText FR/EN parity', () => {
  it('a exactement la même forme structurelle (clés + types) en FR et EN', () => {
    const fr = shapeOf(FR_TEXT)
    const en = shapeOf(EN_TEXT)
    expect(en).toEqual(fr)
  })

  it('n\'a aucune chaîne vide en FR', () => {
    const flat = JSON.stringify(FR_TEXT)
    expect(flat).not.toContain('""')
  })

  it('n\'a aucune chaîne vide en EN', () => {
    const flat = JSON.stringify(EN_TEXT)
    expect(flat).not.toContain('""')
  })
})

describe('getSquadText', () => {
  it('retourne FR par défaut', () => {
    expect(getSquadText('fr')).toBe(FR_TEXT)
  })

  it('retourne EN pour locale "en"', () => {
    expect(getSquadText('en')).toBe(EN_TEXT)
  })

  it('fallback FR pour une locale inconnue', () => {
    expect(getSquadText('xx')).toBe(FR_TEXT)
    expect(getSquadText(undefined)).toBe(FR_TEXT)
  })

  it('compose correctement les fonctions paramétrées (FR)', () => {
    const t = getSquadText('fr')
    expect(t.selection.placeholder(5)).toBe('Rechercher parmi 5 coéquipiers…')
    expect(t.table.withTeammate('Foo')).toBe('Avec Foo')
    expect(t.errors.loadError('boom')).toBe('Erreur : boom')
  })

  it('compose correctement les fonctions paramétrées (EN)', () => {
    const t = getSquadText('en')
    expect(t.selection.placeholder(5)).toBe('Search among 5 teammates…')
    expect(t.table.withTeammate('Foo')).toBe('With Foo')
    expect(t.errors.loadError('boom')).toBe('Error: boom')
  })
})
