/**
 * fragDetailBreakdown.test.ts — contrat de `buildFragDetailBreakdown`, source unique du
 * « Détails des frags » / « Outils de destruction » (Match view, Synthèse, Sessions,
 * Timeseries, Escouade).
 *
 * Cas de tête (2026-08-29, DEC-1) : les classes SANS outil de destruction identifiable
 * — équipement (répulseur, bobines), environnement (chute), UGC, résidu — n'ont RIEN à
 * faire dans un tableau d'armes. Elles restent au sunburst, qui les sert par une
 * provenance à part. Véhicule et tourelle, eux, y restent (V73-3.2 : un engin est un
 * outil réel, nommé par engin).
 */
import { describe, it, expect } from 'vitest'
import { buildFragDetailBreakdown, NON_WEAPON_FRAG_CLASSES } from './fragDetailBreakdown'
import type { FragDistribution, SynthesisWeaponKillEntry } from '@/lib/api/types'

// Libellés identité → les labels de sortie sont les clés brutes (lisibilité des assertions).
const LABELS = { roleLabel: (r: string) => r, classLabel: (c: string) => c, locale: 'fr' as const }

/** Liste per-arme de la surface (armes gun + une ligne parasite non-arme). */
const WEAPONS: SynthesisWeaponKillEntry[] = [
  { label: 'AR', kills: 40, class: 'shoulder' },
  { label: 'Sniper', kills: 12, class: 'heavy' },
  { label: 'Hors registre', kills: 3 }, // classe absente → bénéfice du doute (gardée)
  { label: 'Spartan', kills: 9, class: 'unattributed' }, // parasite → écartée
]

/** Distribution avec les DEUX familles : outils réels (engins) et classes sans outil. */
const DIST: FragDistribution = {
  total_kills: 100,
  classes: [
    { class: 'shoulder', kills: 40, authoritative: false, roles: [{ role: 'automatic', kills: 40 }] },
    { class: 'melee', kills: 6, authoritative: true, roles: [{ role: 'assassination', kills: 2 }, { role: 'direct_melee', kills: 4 }] },
    { class: 'grenade', kills: 5, authoritative: true },
    { class: 'vehicle', kills: 7, authoritative: false, roles: [{ role: 'hinf_warthog', kills: 7, label: 'Warthog' }] },
    { class: 'turret', kills: 4, authoritative: false, roles: [{ role: 'h5_turret_gauss', kills: 4, label: 'Tourelle Gauss' }] },
    // ── Classes SANS outil identifiable (le sujet du cas) ────────────────────────
    { class: 'equipment', kills: 8, authoritative: false, roles: [{ role: 'hinf_repulsor', kills: 8, label: 'Répulseur' }] },
    {
      class: 'environmental',
      kills: 11,
      authoritative: false,
      roles: [
        { role: 'hinf_coil_plasma', kills: 6, label: 'Bobine à plasma' },
        { role: 'hinf_fall', kills: 5, label: 'Chute et environnement' },
      ],
    },
    { class: 'other', kills: 3, authoritative: false },
    { class: 'unattributed', kills: 16, authoritative: false },
  ],
}

const labelsOf = (rows: SynthesisWeaponKillEntry[]) => rows.map((r) => r.label)

describe('buildFragDetailBreakdown — classes sans outil de destruction (DEC-1)', () => {
  it('exclut équipement / environnement / UGC / résidu du breakdown par-arme', () => {
    const labels = labelsOf(buildFragDetailBreakdown(DIST, WEAPONS, LABELS))
    for (const absent of ['Répulseur', 'Bobine à plasma', 'Chute et environnement', 'other', 'unattributed', 'Spartan']) {
      expect(labels, `« ${absent} » n'est pas un outil du joueur`).not.toContain(absent)
    }
    // Aucune ligne ne porte une classe non-arme, quel que soit son libellé.
    const classes = buildFragDetailBreakdown(DIST, WEAPONS, LABELS).map((r) => r.class ?? '')
    expect(classes.filter((c) => NON_WEAPON_FRAG_CLASSES.has(c))).toEqual([])
  })

  it('garde les engins véhicule/tourelle (non-régression V73-3.2)', () => {
    const labels = labelsOf(buildFragDetailBreakdown(DIST, WEAPONS, LABELS))
    expect(labels).toContain('Warthog')
    expect(labels).toContain('Tourelle Gauss')
  })

  it('garde les armes gun et le détail mêlée/grenade/capacités, trié par valeur décroissante', () => {
    const rows = buildFragDetailBreakdown(DIST, WEAPONS, LABELS)
    const labels = labelsOf(rows)
    expect(labels).toContain('AR')
    expect(labels).toContain('Sniper')
    expect(labels).toContain('Hors registre') // classe vide = arme hors registre, gardée
    expect(labels).toContain('assassination')
    expect(labels).toContain('direct_melee')
    expect(labels).toContain('grenade') // classe feuille → libellé de classe
    const kills = rows.map((r) => r.kills)
    expect([...kills].sort((a, b) => b - a)).toEqual(kills)
  })

  it('le total des frags listés ne compte QUE des outils identifiables', () => {
    const total = buildFragDetailBreakdown(DIST, WEAPONS, LABELS).reduce((s, r) => s + r.kills, 0)
    // 40 AR + 12 Sniper + 3 hors registre + 6 mêlée + 5 grenade + 7 véhicule + 4 tourelle.
    expect(total).toBe(77)
  })

  it('miroir du set Go : équipement, environnement, UGC et résidu, jamais les engins', () => {
    expect([...NON_WEAPON_FRAG_CLASSES].sort()).toEqual([
      'environmental',
      'equipment',
      'other',
      'unattributed',
    ])
  })
})
