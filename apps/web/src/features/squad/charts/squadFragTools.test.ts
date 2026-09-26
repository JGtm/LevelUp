/**
 * squadFragTools.test.ts — « Outils de destruction » : les lignes `weapon_tools` du serveur
 * nommées et mises à la forme du graphe (D8 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26).
 * Aucun regroupement, aucun plafond : le module nomme, il ne décide rien.
 */
import { describe, it, expect } from 'vitest'
import { buildSquadToolRows, toolLineLabel, type SquadToolKindLabels } from './squadFragTools'
import type { SquadWeaponToolLine, SquadWeaponTools } from '@/lib/api/types'

const LABELS: SquadToolKindLabels = {
  melee: 'Mêlée',
  assassination: 'Assassinat',
  ground_pound: 'Coup au sol',
  shoulder_bash: 'Charge spartane',
  explosive_object: 'Objet explosif (bidon)',
  environment: 'Chute, environnement',
  unattributed: 'Non attribué',
}

function weapon(key: string, fr: string, en: string, cls: string, kills: Record<string, number>): SquadWeaponToolLine {
  const total = Object.values(kills).reduce((a, b) => a + b, 0)
  return { kind: 'weapon', weapon_key: key, label: fr, label_en: en, class: cls, kills_by_player: kills, total_squad: total }
}

function kind(k: string, cls: string, kills: Record<string, number>): SquadWeaponToolLine {
  const total = Object.values(kills).reduce((a, b) => a + b, 0)
  return { kind: k, class: cls, kills_by_player: kills, total_squad: total }
}

/** Seize outils, dans l'ordre du serveur (total décroissant, reliquat en dernier). */
function tools(): SquadWeaponTools {
  const lines: SquadWeaponToolLine[] = [
    weapon('hinf_br75', 'BR75', 'BR75', 'shoulder', { J: 22, C: 22, M: 35 }),
    kind('melee', 'melee', { J: 6, C: 13, M: 16 }),
    weapon('hinf_frag_grenade', 'Grenade frag', 'Frag Grenade', 'grenade', { J: 2, C: 1, M: 4 }),
    kind('explosive_object', 'unattributed', { J: 2, C: 1, M: 2 }),
    kind('environment', 'environmental', { C: 1, M: 1 }),
  ]
  for (let i = 0; i < 10; i++) lines.push(weapon(`hinf_w${i}`, `Arme ${i}`, `Weapon ${i}`, 'shoulder', { J: 1 }))
  lines.push(kind('unattributed', 'unattributed', { M: 1 }))
  return { players: ['J', 'C', 'M'], lines }
}

describe('buildSquadToolRows', () => {
  it('vide → null', () => {
    expect(buildSquadToolRows(null, { locale: 'fr', labels: LABELS })).toBeNull()
    expect(buildSquadToolRows({ players: [], lines: [] }, { locale: 'fr', labels: LABELS })).toBeNull()
  })

  it('AUCUN plafond ni ligne « Autres » : les 16 lignes du serveur, toutes nommées', () => {
    const res = buildSquadToolRows(tools(), { locale: 'fr', labels: LABELS })!
    expect(res.rows).toHaveLength(16)
    expect(res.rows.some((r) => /autres/i.test(r.label))).toBe(false)
    expect(res.rows.every((r) => r.label !== '')).toBe(true)
  })

  it('ordre renversé : la première ligne du serveur en HAUT (dernière catégorie du graphe)', () => {
    const res = buildSquadToolRows(tools(), { locale: 'fr', labels: LABELS })!
    expect(res.rows[res.rows.length - 1].label).toBe('BR75')
    expect(res.rows[0].label).toBe('Non attribué')
  })

  it('comptes, total et classe (pastille) repris tels quels', () => {
    const res = buildSquadToolRows(tools(), { locale: 'fr', labels: LABELS })!
    const br = res.rows.find((r) => r.label === 'BR75')!
    expect(br.killsByPlayer).toEqual({ J: 22, C: 22, M: 35 })
    expect(br.total).toBe(79)
    expect(br.cls).toBe('shoulder')
    const bidon = res.rows.find((r) => r.key === 'explosive_object')!
    expect(bidon.label).toBe('Objet explosif (bidon)')
    expect(bidon.cls).toBe('unattributed')
    expect(res.players).toEqual(['J', 'C', 'M'])
  })
})

describe('toolLineLabel', () => {
  const frag = weapon('hinf_frag_grenade', 'Grenade frag', 'Frag Grenade', 'grenade', { J: 1 })
  it('arme : nom de registre dans la langue de l’interface', () => {
    expect(toolLineLabel(frag, 'fr', LABELS)).toBe('Grenade frag')
    expect(toolLineLabel(frag, 'en', LABELS)).toBe('Frag Grenade')
  })
  it('arme sans nom dans la langue demandée → l’autre langue', () => {
    expect(toolLineLabel({ ...frag, label_en: '' }, 'en', LABELS)).toBe('Grenade frag')
  })
  it('natures nommées par le web ; nature inconnue → sa clé', () => {
    expect(toolLineLabel(kind('melee', 'melee', {}), 'fr', LABELS)).toBe('Mêlée')
    expect(toolLineLabel(kind('environment', 'environmental', {}), 'fr', LABELS)).toBe('Chute, environnement')
    expect(toolLineLabel(kind('nouvelle_nature', '', {}), 'fr', LABELS)).toBe('nouvelle_nature')
  })
})
