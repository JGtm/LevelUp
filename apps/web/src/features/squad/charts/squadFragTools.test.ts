/**
 * squadFragTools.test.ts — « Outils de destruction » Escouade (version multi-joueurs de
 * buildFragDetailBreakdown). Couvre : fusion multi-joueurs, exclusion Spartan/unattributed,
 * anti-double-comptage des sentinels grenade/mêlée, cap top-N + « Autres armes », split Mêlée.
 */
import { describe, it, expect } from 'vitest'
import { buildSquadFragTools, SQUAD_TOOLS_TOP_GUNS } from './squadFragTools'
import type { FragClassEntry, SquadWeaponKills } from '@/lib/api/types'

// Libellés identité → les labels de sortie sont les clés brutes (guns gardent leur nom).
const roleLabel = (r: string) => r
const classLabel = (c: string) => c
const opts = (topGuns: number) => ({ roleLabel, classLabel, otherWeaponsLabel: 'Autres armes', topGuns })

function input(): SquadWeaponKills {
  return {
    players: ['Me', 'F1'],
    bars: [
      { weapon_id: 10, label: 'AR', class: 'shoulder', kills_by_player: { Me: 50, F1: 40 }, total_squad: 90 },
      { weapon_id: 11, label: 'BR', class: 'shoulder', kills_by_player: { Me: 30, F1: 20 }, total_squad: 50 },
      { weapon_id: 12, label: 'Sniper', class: 'heavy', kills_by_player: { Me: 10, F1: 5 }, total_squad: 15 },
      { weapon_id: 13, label: 'Pistol', class: 'sidearm', kills_by_player: { Me: 8, F1: 2 }, total_squad: 10 },
      { weapon_id: 14, label: 'Rocket', class: 'heavy', kills_by_player: { Me: 3, F1: 1 }, total_squad: 4 },
      // Résidu « Spartan » (classe unattributed) — DOIT disparaître.
      { weapon_id: 3168248199, label: 'Spartan', class: 'unattributed', kills_by_player: { Me: 7, F1: 3 }, total_squad: 10 },
      // Sentinels grenade/mêlée (is_grenade_melee, classe vide) — DOIVENT être écartés du
      // volet per-arme (leur compte fait doublon avec la distribution).
      { weapon_id: 0, label: 'Grenade', class: '', is_grenade_melee: true, kills_by_player: { Me: 5, F1: 4 }, total_squad: 9 },
      { weapon_id: 1, label: 'Mêlée', class: '', is_grenade_melee: true, kills_by_player: { Me: 6, F1: 2 }, total_squad: 8 },
    ],
  }
}

// FragDistribution par joueur (niveau classe). Les classes gun sont IGNORÉES par
// buildFragDetailBreakdown (le détail gun vient du volet per-arme) ; seules melee/grenade/
// spartan_ability alimentent le détail non-arme.
function fragClasses(): Record<string, FragClassEntry[]> {
  return {
    Me: [
      { authoritative: false, class: 'shoulder', kills: 80 },
      { authoritative: false, class: 'heavy', kills: 13 },
      { authoritative: false, class: 'sidearm', kills: 8 },
      { authoritative: true, class: 'melee', kills: 8, roles: [{ role: 'assassination', kills: 2 }, { role: 'direct_melee', kills: 6 }] },
      { authoritative: true, class: 'grenade', kills: 5 },
      { authoritative: true, class: 'spartan_ability', kills: 3, roles: [{ role: 'ground_pound', kills: 2 }, { role: 'shoulder_bash', kills: 1 }] },
      { authoritative: false, class: 'unattributed', kills: 7 },
    ],
    F1: [
      { authoritative: false, class: 'shoulder', kills: 60 },
      { authoritative: false, class: 'heavy', kills: 6 },
      { authoritative: true, class: 'melee', kills: 2, roles: [{ role: 'direct_melee', kills: 2 }] },
      { authoritative: true, class: 'grenade', kills: 4 },
      { authoritative: false, class: 'unattributed', kills: 3 },
    ],
  }
}

function byLabelOf(res: SquadWeaponKills) {
  return new Map((res.bars ?? []).map((b) => [b.label, b]))
}

describe('buildSquadFragTools', () => {
  it('vide / null → null', () => {
    expect(buildSquadFragTools(null, {}, opts(8))).toBeNull()
    expect(buildSquadFragTools({ players: [], bars: [] }, {}, opts(8))).toBeNull()
    expect(SQUAD_TOOLS_TOP_GUNS).toBeGreaterThan(0)
  })

  it('fusionne les armes gun par joueur (kills_by_player + total_squad)', () => {
    const res = buildSquadFragTools(input(), fragClasses(), opts(10))!
    const byLabel = byLabelOf(res)
    expect(byLabel.get('AR')?.kills_by_player).toEqual({ Me: 50, F1: 40 })
    expect(byLabel.get('AR')?.total_squad).toBe(90)
    expect(byLabel.get('Sniper')?.total_squad).toBe(15)
  })

  it('exclut « Spartan » / unattributed (jamais dans la sortie)', () => {
    const res = buildSquadFragTools(input(), fragClasses(), opts(10))!
    const byLabel = byLabelOf(res)
    expect(byLabel.has('Spartan')).toBe(false)
    expect((res.bars ?? []).every((b) => b.class !== 'unattributed')).toBe(true)
  })

  it('écarte les sentinels grenade/mêlée : détail depuis la distribution, sans doublon', () => {
    const res = buildSquadFragTools(input(), fragClasses(), opts(10))!
    const byLabel = byLabelOf(res)
    // Les libellés bruts des sentinels (Grenade/Mêlée majuscules) ne surfacent PAS comme gun.
    expect(byLabel.has('Grenade')).toBe(false)
    expect(byLabel.has('Mêlée')).toBe(false)
    // Grenade vient uniquement de la distribution (Me 5 + F1 4 = 9), pas ×2.
    expect(byLabel.get('grenade')?.total_squad).toBe(9)
    expect(byLabel.get('grenade')?.kills_by_player).toEqual({ Me: 5, F1: 4 })
  })

  it('split Mêlée + capacités spartanes présents (Assassinat / Corps-à-corps / Coup au sol / Charge)', () => {
    const res = buildSquadFragTools(input(), fragClasses(), opts(10))!
    const byLabel = byLabelOf(res)
    expect(byLabel.get('assassination')?.kills_by_player).toEqual({ Me: 2 })
    expect(byLabel.get('direct_melee')?.total_squad).toBe(8) // Me 6 + F1 2
    expect(byLabel.get('ground_pound')?.total_squad).toBe(2)
    expect(byLabel.get('shoulder_bash')?.total_squad).toBe(1)
    // Pas de feuille « melee » (la classe a des rôles → pas de niveau feuille).
    expect(byLabel.has('melee')).toBe(false)
  })

  it('cap top-N : garde les N plus grosses armes, agrège le reste en « Autres armes »', () => {
    const res = buildSquadFragTools(input(), fragClasses(), opts(3))!
    const byLabel = byLabelOf(res)
    // Top 3 guns par total escouade : AR(90), BR(50), Sniper(15).
    expect(byLabel.has('AR')).toBe(true)
    expect(byLabel.has('BR')).toBe(true)
    expect(byLabel.has('Sniper')).toBe(true)
    // Pistol(10) + Rocket(4) repliés.
    expect(byLabel.has('Pistol')).toBe(false)
    expect(byLabel.has('Rocket')).toBe(false)
    const autres = byLabel.get('Autres armes')
    expect(autres).toBeDefined()
    expect(autres?.total_squad).toBe(14)
    expect(autres?.kills_by_player).toEqual({ Me: 11, F1: 3 })
    // Les lignes non-arme restent toutes présentes malgré le cap.
    expect(byLabel.has('grenade')).toBe(true)
    expect(byLabel.has('assassination')).toBe(true)
  })

  it('ordonne ASC par total escouade (comme le chart existant)', () => {
    const res = buildSquadFragTools(input(), fragClasses(), opts(3))!
    const totals = (res.bars ?? []).map((b) => b.total_squad)
    expect([...totals].sort((a, b) => a - b)).toEqual(totals)
  })
})
