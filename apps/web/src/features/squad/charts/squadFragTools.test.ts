/**
 * squadFragTools.test.ts — « Outils de destruction » Escouade (version multi-joueurs de
 * buildFragDetailBreakdown). Couvre : fusion multi-joueurs, exclusion Spartan/unattributed,
 * anti-double-comptage des sentinels grenade/mêlée, DEUX caps top-N (« Autres armes » côté
 * gun, « Autres frags » côté détail), split Mêlée.
 */
import { describe, it, expect } from 'vitest'
import { buildSquadFragTools, SQUAD_TOOLS_TOP_DETAILS, SQUAD_TOOLS_TOP_GUNS } from './squadFragTools'
import type { FragClassEntry, SquadWeaponKills } from '@/lib/api/types'

// Libellés identité → les labels de sortie sont les clés brutes (guns gardent leur nom).
const roleLabel = (r: string) => r
const classLabel = (c: string) => c
const opts = (topGuns: number, topDetails = 99) => ({
  roleLabel,
  classLabel,
  locale: 'fr' as const,
  otherWeaponsLabel: 'Autres armes',
  otherKillsLabel: 'Autres frags',
  topGuns,
  topDetails,
})

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

  it('« Autres armes » épinglée tout en bas (index 0), le reste ASC par total escouade', () => {
    const res = buildSquadFragTools(input(), fragClasses(), opts(3))!
    const bars = res.bars ?? []
    // Sans inverse yAxis, la 1re catégorie est rendue EN BAS → l'agrégat y est épinglé.
    expect(bars[0]?.label).toBe('Autres armes')
    // Le reste (hors agrégat) reste trié ASC par total escouade (comme le chart existant).
    const rest = bars.slice(1).map((b) => b.total_squad)
    expect([...rest].sort((a, b) => a - b)).toEqual(rest)
  })

  // ── Cap du DÉTAIL non-arme (2026-08-29) ────────────────────────────────────────
  // Le cap top-N ne portait QUE sur les armes : les lignes de détail arrivaient toutes,
  // et le tri ASC les faisait remonter EN HAUT du graphe, noyant les 8 armes.
  describe('cap des lignes de détail non-arme', () => {
    it('garde les N plus grosses lignes de détail, agrège le reste en « Autres frags »', () => {
      const res = buildSquadFragTools(input(), fragClasses(), opts(10, 2))!
      const byLabel = byLabelOf(res)
      // Détail par total : direct_melee(8), grenade(9), assassination(2), ground_pound(2),
      // shoulder_bash(1) → top 2 = grenade(9) et direct_melee(8).
      expect(byLabel.has('grenade')).toBe(true)
      expect(byLabel.has('direct_melee')).toBe(true)
      expect(byLabel.has('assassination')).toBe(false)
      expect(byLabel.has('ground_pound')).toBe(false)
      expect(byLabel.has('shoulder_bash')).toBe(false)
      // Aucune perte silencieuse : 2 + 2 + 1 = 5 frags repliés, ventilés par joueur.
      const autres = byLabel.get('Autres frags')
      expect(autres?.total_squad).toBe(5)
      expect(autres?.kills_by_player).toEqual({ Me: 5 })
    })

    it('les 8 armes du cap restent toutes visibles malgré un détail pléthorique', () => {
      const res = buildSquadFragTools(input(), fragClasses(), opts(SQUAD_TOOLS_TOP_GUNS, SQUAD_TOOLS_TOP_DETAILS))!
      const bars = res.bars ?? []
      const byLabel = byLabelOf(res)
      for (const gun of ['AR', 'BR', 'Sniper', 'Pistol', 'Rocket']) expect(byLabel.has(gun)).toBe(true)
      // Plafond global de catégories : topGuns + topDetails + les 2 agrégats au plus.
      expect(bars.length).toBeLessThanOrEqual(SQUAD_TOOLS_TOP_GUNS + SQUAD_TOOLS_TOP_DETAILS + 2)
    })

    it('les deux agrégats sont épinglés en bas, « Autres armes » en tout dernier', () => {
      const res = buildSquadFragTools(input(), fragClasses(), opts(3, 2))!
      const bars = res.bars ?? []
      expect(bars[0]?.label).toBe('Autres armes')
      expect(bars[1]?.label).toBe('Autres frags')
      const rest = bars.slice(2).map((b) => b.total_squad)
      expect([...rest].sort((a, b) => a - b)).toEqual(rest)
    })

    it('détail sous le cap → aucune ligne « Autres frags » (pas d\'agrégat vide)', () => {
      const res = buildSquadFragTools(input(), fragClasses(), opts(10, SQUAD_TOOLS_TOP_DETAILS))!
      expect(byLabelOf(res).has('Autres frags')).toBe(false)
      expect(SQUAD_TOOLS_TOP_DETAILS).toBeGreaterThan(0)
    })
  })
})
