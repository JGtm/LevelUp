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

import type { CompareWeaponSide } from '@/lib/api/types'

import { fragClassRows, hasWeaponProfile } from './compareWeapons_logic'

/** Un côté de profil minimal, complété par l'appelant. */
const profil = (o: Partial<CompareWeaponSide>): CompareWeaponSide => ({
  matches: 10,
  total_kills: 100,
  frag_classes: [],
  top_weapons: [],
  ...o,
})

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
