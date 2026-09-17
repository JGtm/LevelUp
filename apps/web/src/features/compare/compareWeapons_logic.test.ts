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
  roleAxis,
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

describe('fragClassRows — l’union des classes, sur N côtés', () => {
  it('garde l’ordre du premier côté, puis ajoute ce que les suivants portent', () => {
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
    expect(epaule.parts[1]).toEqual({ kills: 0, sharePct: 0 })
    expect(grenade.parts[0]).toEqual({ kills: 0, sharePct: 0 })
  })

  /**
   * LE CAS DU GATE VISUEL (2026-09-17) : un seul des trois joueurs porte « Environnement ».
   * Avec deux unions à deux appliquées séparément, la ligne n'existait que d'un côté et les
   * colonnes se décalaient. Ici l'union est faite UNE fois sur les trois.
   */
  it('trois côtés : une classe que seul C porte a sa ligne, et les trois parts existent', () => {
    const a = profil({ frag_classes: [{ class: 'shoulder', kills: 40, share_pct: 40 }] })
    const b = profil({ frag_classes: [{ class: 'shoulder', kills: 30, share_pct: 30 }] })
    const c = profil({ frag_classes: [{ class: 'environmental', kills: 5, share_pct: 5 }] })
    const rows = fragClassRows(a, b, c)
    expect(rows.map((r) => r.classKey)).toEqual(['shoulder', 'environmental'])
    for (const r of rows) expect(r.parts).toHaveLength(3)
    const env = rows.find((r) => r.classKey === 'environmental')!
    expect(env.parts[0].sharePct).toBe(0)
    expect(env.parts[1].sharePct).toBe(0)
    expect(env.parts[2].sharePct).toBe(5)
  })

  it('accepte un côté absent (profil non servi)', () => {
    const a = profil({ frag_classes: [{ class: 'shoulder', kills: 40, share_pct: 40 }] })
    expect(fragClassRows(a, null).map((r) => r.classKey)).toEqual(['shoulder'])
    expect(fragClassRows(null, null)).toEqual([])
  })
})

describe('roleAxis / roleRangeLines — un axe unique pour tous les graphes', () => {
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
  /** C porte un rôle qu'aucun des deux autres n'a — le cas du gate visuel. */
  const c = profil({
    range: {
      weapons: [{ weapon_key: 'environmental', kills: side({ median: 8 }) }],
      median_kills_m: 8,
      median_deaths_m: 0,
      measured_kills: 10,
      total_kills: 12,
      measured_deaths: 0,
      total_deaths: 0,
    },
  })

  it('trie par médiane de la RÉFÉRENCE côté frags, les rôles qu’elle n’a pas à la fin', () => {
    // A mesure melee (2) et precision (20) ; sniper (B) et environmental (C) lui sont
    // inconnus → à la fin, par libellé (« Environnement » avant « Tir de précision »).
    expect(roleAxis([a, b, c], nomDeRole).map((e) => e.weaponKey)).toEqual([
      'melee',
      'precision',
      'environmental',
      'sniper',
    ])
  })

  it('résout le libellé et n’expose JAMAIS une clé de manifeste brute', () => {
    for (const e of roleAxis([a, b, c], nomDeRole)) {
      expect(e.label).not.toContain('frags.role.')
      expect(e.label).not.toContain('frags.class.')
    }
    expect(roleAxis([a, b], nomDeRole).find((e) => e.weaponKey === 'precision')!.label).toBe(
      'Précision',
    )
  })

  it('superpose A en haut et B en bas, sur le côté demandé', () => {
    const axis = roleAxis([a, b], nomDeRole)
    const precision = roleRangeLines(axis, a, b, 'kills').find((l) => l.weaponKey === 'precision')!
    expect(precision.top?.median).toBe(20)
    expect(precision.bottom?.median).toBe(25)
  })

  /**
   * LE TÉMOIN DU GATE VISUEL : les deux paires du mode miroir rendent EXACTEMENT le même axe,
   * de la même longueur, dans le même ordre — y compris les lignes où le couple n'a rien.
   */
  it('les deux paires du miroir portent des axes identiques, lignes vides comprises', () => {
    const axis = roleAxis([a, b, c], nomDeRole)
    const paires = [
      roleRangeLines(axis, a, b, 'kills'),
      roleRangeLines(axis, a, c, 'kills'),
      roleRangeLines(axis, a, b, 'deaths'),
      roleRangeLines(axis, a, c, 'deaths'),
    ]
    const attendu = axis.map((e) => e.weaponKey)
    for (const lignes of paires) {
      expect(lignes).toHaveLength(axis.length)
      expect(lignes.map((l) => l.weaponKey)).toEqual(attendu)
    }
    // A vs B ne mesure rien sur « environmental » : la ligne existe quand même, vide.
    const env = paires[0].find((l) => l.weaponKey === 'environmental')!
    expect(env.top).toBeNull()
    expect(env.bottom).toBeNull()
    // A vs C la porte côté C.
    expect(paires[1].find((l) => l.weaponKey === 'environmental')!.bottom?.median).toBe(8)
  })

  it('une ligne sans mesure de ce côté est rendue vide, jamais filtrée', () => {
    const axis = roleAxis([a, b], nomDeRole)
    const morts = roleRangeLines(axis, a, b, 'deaths')
    expect(morts.map((l) => l.weaponKey)).toEqual(axis.map((e) => e.weaponKey))
    const precision = morts.find((l) => l.weaponKey === 'precision')!
    expect(precision.top).toBeNull()
    expect(precision.bottom?.median).toBe(18)
    const melee = morts.find((l) => l.weaponKey === 'melee')!
    expect(melee.top).toBeNull()
    expect(melee.bottom).toBeNull()
  })

  it('rend une liste vide sans bloc de portée', () => {
    expect(roleAxis([profil({}), profil({})], nomDeRole)).toEqual([])
    expect(roleRangeLines([], profil({}), profil({}), 'kills')).toEqual([])
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
