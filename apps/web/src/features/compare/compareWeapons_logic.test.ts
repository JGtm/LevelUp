/**
 * compareWeapons_logic.test — LES DÉCISIONS DE LECTURE DU PROFIL D'ARMES.
 *
 * Ce que ces tests verrouillent, dans l'ordre d'importance :
 *
 *  1. L'UNION NE PRIVILÉGIE PERSONNE. Une classe ou un rôle que seul B porte a sa ligne, et
 *     le côté qui n'a rien y vaut ZÉRO — pas « absent ». « Lui n'a jamais tué à la grenade »
 *     est une information de style, exactement ce que la section compare ; l'omettre la perd.
 *  2. AUCUNE CLÉ `frags.` BRUTE N'EST JAMAIS AFFICHÉE (D8). Le repli final rend la clé NUE,
 *     jamais son chemin de manifeste — même exigence que `fragRoleDisplayLabel`.
 *  3. LE TRI EST DÉTERMINISTE et suit la médiane de A, puis celle de B pour les rôles que A
 *     n'a pas : sans ce second critère, ces rôles-là se rangeraient arbitrairement.
 */
import { describe, expect, it } from 'vitest'

import type { CompareWeaponSide, WeaponRangeSide } from '@/lib/api/types'

import {
  fragClassRows,
  hasWeaponProfile,
  roleLabel,
  roleRangeLines,
} from './compareWeapons_logic'

const side = (o: Partial<WeaponRangeSide>): WeaponRangeSide => ({
  measured: 10,
  p10: 5,
  median: 7,
  p90: 10,
  min_m: 2,
  max_m: 14,
  above_pct: 30,
  level_pct: 50,
  below_pct: 20,
  ...o,
})

/** Un côté de profil minimal, complété par l'appelant. */
const profil = (o: Partial<CompareWeaponSide>): CompareWeaponSide => ({
  matches: 10,
  total_kills: 100,
  frag_classes: [],
  top_weapons: [],
  ...o,
})

/** Résolveur de manifeste factice : connaît deux rôles et une classe. */
const resolveManifeste = (key: string) => {
  const connus: Record<string, string> = {
    'frags.role.precision': 'Précision',
    'frags.role.sniper': 'Tir de précision',
    'frags.class.grenade': 'Grenade',
  }
  return connus[key] ?? key
}
const nomDeRole = (key: string) => roleLabel(key, resolveManifeste)

describe('fragClassRows — l’union des classes', () => {
  it('garde l’ordre de A, puis ajoute ce que seul B porte', () => {
    const a = profil({
      frag_classes: [
        { class: 'shoulder', kills: 40, share_pct: 40 },
        { class: 'melee', kills: 10, share_pct: 10 },
      ],
    })
    const b = profil({
      frag_classes: [
        { class: 'melee', kills: 5, share_pct: 5 },
        { class: 'grenade', kills: 20, share_pct: 20 },
      ],
    })
    expect(fragClassRows(a, b).map((r) => r.classKey)).toEqual(['shoulder', 'melee', 'grenade'])
  })

  it('une classe absente d’un côté vaut ZÉRO, jamais « absent »', () => {
    const a = profil({ frag_classes: [{ class: 'shoulder', kills: 40, share_pct: 40 }] })
    const b = profil({ frag_classes: [{ class: 'grenade', kills: 20, share_pct: 20 }] })
    const rows = fragClassRows(a, b)
    const epaule = rows.find((r) => r.classKey === 'shoulder')!
    const grenade = rows.find((r) => r.classKey === 'grenade')!
    expect(epaule.sharePctB).toBe(0)
    expect(epaule.killsB).toBe(0)
    expect(grenade.sharePctA).toBe(0)
    expect(grenade.killsA).toBe(0)
  })

  it('accepte un côté absent (profil non servi)', () => {
    const a = profil({ frag_classes: [{ class: 'shoulder', kills: 40, share_pct: 40 }] })
    expect(fragClassRows(a, null).map((r) => r.classKey)).toEqual(['shoulder'])
    expect(fragClassRows(null, null)).toEqual([])
  })
})

describe('roleRangeLines — l’union des rôles, un côté de mesure à la fois', () => {
  const a = profil({
    range: {
      weapons: [
        { weapon_key: 'precision', kills: side({ median: 20 }) },
        { weapon_key: 'melee', kills: side({ median: 2 }) },
      ],
      median_kills_m: 12,
      median_deaths_m: 14,
      measured_kills: 100,
      total_kills: 130,
      measured_deaths: 80,
      total_deaths: 95,
    },
  })
  const b = profil({
    range: {
      weapons: [
        { weapon_key: 'precision', kills: side({ median: 25 }), deaths: side({ median: 18 }) },
        { weapon_key: 'sniper', kills: side({ median: 40 }) },
      ],
      median_kills_m: 30,
      median_deaths_m: 18,
      measured_kills: 60,
      total_kills: 70,
      measured_deaths: 50,
      total_deaths: 60,
    },
  })

  it('superpose A en haut et B en bas, sur le côté demandé', () => {
    const lignes = roleRangeLines(a, b, 'kills', nomDeRole)
    const precision = lignes.find((l) => l.weaponKey === 'precision')!
    expect(precision.top?.median).toBe(20)
    expect(precision.bottom?.median).toBe(25)
  })

  it('trie par médiane de A croissante, puis par celle de B pour les rôles que A n’a pas', () => {
    // Mêlée (A, 2) · Précision (A, 20) · Sniper (B seul, 40).
    expect(roleRangeLines(a, b, 'kills', nomDeRole).map((l) => l.weaponKey)).toEqual([
      'melee',
      'precision',
      'sniper',
    ])
  })

  it('écarte un rôle qu’aucun des deux n’a mesuré de ce côté', () => {
    // Côté morts : seul B mesure « precision ». Mêlée et sniper n'ont pas de côté morts.
    const lignes = roleRangeLines(a, b, 'deaths', nomDeRole)
    expect(lignes.map((l) => l.weaponKey)).toEqual(['precision'])
    expect(lignes[0].top).toBeNull()
    expect(lignes[0].bottom?.median).toBe(18)
  })

  it('résout le libellé et n’affiche JAMAIS une clé de manifeste brute', () => {
    const lignes = roleRangeLines(a, b, 'kills', nomDeRole)
    for (const l of lignes) {
      expect(l.label).not.toContain('frags.role.')
      expect(l.label).not.toContain('frags.class.')
    }
    expect(lignes.find((l) => l.weaponKey === 'precision')!.label).toBe('Précision')
  })

  it('rend une liste vide sans bloc de portée', () => {
    expect(roleRangeLines(profil({}), profil({}), 'kills', nomDeRole)).toEqual([])
  })
})

describe('roleLabel — rôle, puis classe, puis la clé NUE', () => {
  it('résout un rôle canonique', () => {
    expect(roleLabel('precision', resolveManifeste)).toBe('Précision')
  })

  it('retombe sur la CLASSE pour les clés qui sont leur propre rôle', () => {
    // Le registre pose class == role pour grenade/melee/sidearm/equipment/vehicle/turret/
    // environmental : le manifeste ne les déclare que sous `frags.class.*`.
    expect(roleLabel('grenade', resolveManifeste)).toBe('Grenade')
  })

  it('retombe sur la clé NUE, jamais sur le chemin de manifeste', () => {
    const label = roleLabel('role_inconnu', resolveManifeste)
    expect(label).toBe('role_inconnu')
    expect(label).not.toContain('frags.')
  })
})

describe('hasWeaponProfile — la section a-t-elle quoi que ce soit à montrer', () => {
  it('suffit qu’UN des deux joueurs porte un bloc', () => {
    const vide = profil({})
    const avecClasses = profil({ frag_classes: [{ class: 'shoulder', kills: 1, share_pct: 100 }] })
    expect(hasWeaponProfile(vide, avecClasses)).toBe(true)
    expect(hasWeaponProfile(avecClasses, vide)).toBe(true)
  })

  it('faux quand les deux côtés sont vides ou absents', () => {
    expect(hasWeaponProfile(profil({}), profil({}))).toBe(false)
    expect(hasWeaponProfile(null, undefined)).toBe(false)
  })
})
