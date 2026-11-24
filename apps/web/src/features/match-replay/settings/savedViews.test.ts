/**
 * savedViews.test.ts — « MES VUES » : jusqu'à trois vues nommées, renommables, supprimables, une vue d'ouverture,
 * un rattachement facultatif à un mode de jeu. Les réducteurs sont purs ; le stockage est testé à part.
 */
import { describe, expect, it } from 'vitest'

import { factoryValues } from './settingsTable'
import {
  attachMode, cleanName, deleteView, EMPTY_VIEWS, openingViewId, readViews, renameView, saveView, setActiveView,
  setDefaultView, updateView, valuesOfView, viewExists, VIEW_LIMIT, VIEW_NAME_MAX, type ViewsState,
} from './savedViews'
import { presetValues, snapshotValues } from './viewPresets'

const SNAP = snapshotValues(factoryValues())

function withViews(n: number): ViewsState {
  let s = EMPTY_VIEWS
  for (let i = 0; i < n; i++) {
    const r = saveView(s, `Vue ${i + 1}`, SNAP, `v${i + 1}`)
    if (!r.ok) throw new Error('attendu : ok')
    s = r.state
  }
  return s
}

describe('enregistrer une vue', () => {
  it('ajoute la vue et la rend active', () => {
    const r = saveView(EMPTY_VIEWS, '  Mon combat  ', SNAP, 'v1')
    expect(r.ok).toBe(true)
    if (!r.ok) return
    expect(r.state.views).toEqual([{ id: 'v1', name: 'Mon combat', values: SNAP, mode: null }])
    expect(r.state.activeId).toBe('v1')
  })

  it('refuse un nom vide et une quatrième vue, sans rien changer', () => {
    expect(saveView(EMPTY_VIEWS, '   ', SNAP, 'v1')).toEqual({ ok: false, reason: 'name' })
    expect(saveView(withViews(VIEW_LIMIT), 'Une de plus', SNAP, 'v9')).toEqual({ ok: false, reason: 'limit' })
  })

  it('coupe un nom trop long', () => {
    expect(cleanName('x'.repeat(40))).toHaveLength(VIEW_NAME_MAX)
    expect(cleanName('')).toBeNull()
  })
})

describe('renommer, mettre à jour, supprimer', () => {
  it('renomme en place ; un nom vide laisse l’ancien', () => {
    const s = withViews(2)
    expect(renameView(s, 'v1', 'Autre').views[0].name).toBe('Autre')
    expect(renameView(s, 'v1', '  ')).toBe(s)
  })

  it('met à jour la photographie d’une vue', () => {
    const s = updateView(withViews(1), 'v1', { ...SNAP, aim: false })
    expect(s.views[0].values.aim).toBe(false)
  })

  it('supprimer la vue active ou la vue d’ouverture rend ces rôles ; les autres vues restent', () => {
    let s = withViews(2)
    s = setDefaultView(s, 'v1')
    s = setActiveView(s, 'v1')
    const after = deleteView(s, 'v1')
    expect(after.views.map((v) => v.id)).toEqual(['v2'])
    expect(after.defaultId).toBeNull()
    expect(after.activeId).toBe('complet')
    expect(deleteView(setActiveView(s, 'v2'), 'v1').activeId).toBe('v2')
  })
})

describe('la vue d’ouverture et le rattachement à un mode', () => {
  it('sans choix, rien ne s’applique à l’ouverture : les réglages restent comme laissés', () => {
    expect(openingViewId(withViews(2), 'flag')).toBeNull()
  })

  it('la vue d’ouverture est un préréglage ou une vue enregistrée', () => {
    expect(openingViewId(setDefaultView(withViews(1), 'combat'), 'open')).toBe('combat')
    expect(openingViewId(setDefaultView(withViews(1), 'v1'), 'open')).toBe('v1')
  })

  it('une vue rattachée à un mode passe avant la vue d’ouverture, pour ce mode seulement', () => {
    let s = setDefaultView(withViews(2), 'combat')
    s = attachMode(s, 'v2', 'flag')
    expect(openingViewId(s, 'flag')).toBe('v2')
    expect(openingViewId(s, 'open')).toBe('combat')
    expect(openingViewId(attachMode(s, 'v2', null), 'flag')).toBe('combat')
  })

  it('une vue d’ouverture qui n’existe pas n’est pas retenue', () => {
    expect(setDefaultView(withViews(1), 'fantome')).toEqual(withViews(1))
    expect(setDefaultView(setDefaultView(withViews(1), 'combat'), null).defaultId).toBeNull()
  })

  it('la vue active inconnue retombe sur Complet', () => {
    expect(setActiveView(withViews(1), 'fantome').activeId).toBe('complet')
  })
})

describe('retrouver une vue', () => {
  it('lit la table d’un préréglage ou d’une vue enregistrée', () => {
    const s = withViews(1)
    expect(valuesOfView(s, 'combat')).toBe(presetValues('combat'))
    expect(valuesOfView(s, 'v1')).toBe(s.views[0].values)
    expect(valuesOfView(s, 'fantome')).toBeNull()
    expect(viewExists(s, 'objectif')).toBe(true)
    expect(viewExists(s, 'v1')).toBe(true)
  })
})

describe('relire les vues du stockage', () => {
  it('écarte l’invalide, l’inconnu et le doublon ; plafonne à trois', () => {
    const good = (id: string) => ({ id, name: id, values: { aim: false, nimporte: true, trail: { x: 1 } }, mode: 'flag' })
    const read = readViews([good('a'), good('a'), { id: 'b', name: '', values: {} }, null, 'x', good('c'), good('d'), good('e')])
    expect(read.map((v) => v.id)).toEqual(['a', 'c', 'd'])
    expect(read[0].values).toEqual({ aim: false })
    expect(read[0].mode).toBe('flag')
  })

  it('un mode inconnu se lit comme aucun rattachement', () => {
    expect(readViews([{ id: 'a', name: 'A', values: {}, mode: 'bataille' }])[0].mode).toBeNull()
  })

  it('un stockage qui n’est pas une liste donne aucune vue', () => {
    expect(readViews({ id: 'a' })).toEqual([])
    expect(readViews(undefined)).toEqual([])
  })
})
