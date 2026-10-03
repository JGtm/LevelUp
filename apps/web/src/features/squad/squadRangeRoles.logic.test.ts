/**
 * Les décisions PURES de la carte « Rôles de portée ».
 *
 * Ce que ces tests cadenassent : l'ordre des matchs et le gabarit des étiquettes « #N ·
 * carte » (le même que les autres graphes par match de la page) ; les seuils de rôle comme
 * TIERS de la période, jamais des mètres en dur ; le point CREUX sous le plancher, exclu de
 * la tendance et sans rôle ; et la fenêtre glissante — les quatre premiers matchs d'un
 * joueur n'ont pas de rôle, et un match atypique ne fait pas basculer l'étiquette.
 */
import { describe, expect, it } from 'vitest'

import type { MatchRangePlayer, MatchRangeProfile } from '@/lib/api/types'

import {
  categoriesMatchs,
  FENETRE_ROLE,
  mesureDeJoueur,
  moyenneGlissante,
  ordonnerProfils,
  PLANCHER_MESURE,
  quantileLineaire,
  roleDeEcart,
  rolesFenetre,
  seriesPortee,
  seuilsRoles,
  taillePoint,
  type PointPortee,
} from './squadRangeRoles.logic'

function profil(
  i: number,
  joueurs: { xuid: string; gamertag: string; delta: number; measured?: number }[],
  map?: string,
): MatchRangeProfile {
  return {
    match_id: `m${i}`,
    played_at: `2026-09-${String(10 + i).padStart(2, '0')}T20:00:00Z`,
    map_name: map,
    lobby_median_m: 20,
    lobby_measured: 60,
    players: joueurs.map((j) => ({
      xuid: j.xuid,
      gamertag: j.gamertag,
      median_m: 20 + j.delta,
      lobby_delta_m: j.delta,
      measured: j.measured ?? 10,
    })),
  }
}

/** Une suite de N matchs pour un joueur unique, écarts donnés. */
function serieDe(deltas: number[], mesures?: number[]) {
  const profils = deltas.map((d, i) =>
    profil(i, [{ xuid: 'x1', gamertag: 'Kaya', delta: d, measured: mesures?.[i] }]),
  )
  return seriesPortee(ordonnerProfils(profils), ['Kaya'])[0]
}

describe('ordonnerProfils / categoriesMatchs', () => {
  it('range les matchs du PLUS ANCIEN au PLUS RÉCENT, quel que soit l’ordre servi', () => {
    const servis = [profil(2, []), profil(0, []), profil(1, [])]
    expect(ordonnerProfils(servis).map((p) => p.match_id)).toEqual(['m0', 'm1', 'm2'])
  })

  it('étiquette « #N · carte » — même gabarit et même troncature que les autres graphes', () => {
    const profils = ordonnerProfils([
      profil(0, [], 'Streets'),
      profil(1, [], 'Fragmentation'),
      profil(2, []),
    ])
    expect(categoriesMatchs(profils)).toEqual(['#1 · Streets', '#2 · Fragment…', '#3'])
  })
})

describe('seriesPortee', () => {
  it('suit l’ordre du roster (joueur principal en tête), casse indifférente', () => {
    const profils = ordonnerProfils([
      profil(0, [
        { xuid: 'x2', gamertag: 'Wisp', delta: -4 },
        { xuid: 'x1', gamertag: 'JGtm', delta: 2 },
      ]),
    ])
    expect(seriesPortee(profils, ['jgtm', 'Wisp']).map((s) => s.gamertag)).toEqual([
      'JGtm',
      'Wisp',
    ])
  })

  it('marque CREUX un point sous le plancher de frags mesurés', () => {
    const serie = serieDe([3, 3], [PLANCHER_MESURE, PLANCHER_MESURE - 1])
    expect(serie.points.map((p) => p.plein)).toEqual([true, false])
  })
})

describe('seuilsRoles / roleDeEcart', () => {
  it('rend les TIERS des points pleins, pas des mètres en dur', () => {
    const serie = serieDe([0, 3, 6, 9])
    const seuils = seuilsRoles([serie])
    expect(seuils).toEqual({ bas: quantileLineaire([0, 3, 6, 9], 1 / 3), haut: quantileLineaire([0, 3, 6, 9], 2 / 3) })
    // Une escouade entièrement « longue » garde TROIS rôles internes.
    const longue = serieDe([20, 24, 28, 32])
    const s2 = seuilsRoles([longue])!
    expect(roleDeEcart(20, s2)).toBe('front')
    expect(roleDeEcart(32, s2)).toBe('sniper')
  })

  it('ignore les points creux dans le calcul des seuils', () => {
    const serie = serieDe([0, 3, 6, 9, 999], [10, 10, 10, 10, 1])
    expect(seuilsRoles([serie])!.haut).toBeLessThan(10)
  })

  it('rend `null` quand rien n’est mesuré — pas de bandes fabriquées', () => {
    expect(seuilsRoles([serieDe([5, 5], [1, 2])])).toBeNull()
    expect(seuilsRoles([])).toBeNull()
  })
})

describe('moyenneGlissante', () => {
  it('se tait sur les premiers matchs : la fenêtre n’est pas pleine', () => {
    const serie = serieDe([1, 2, 3, 4, 5, 6])
    const m = moyenneGlissante(serie.points)
    expect(m.slice(0, FENETRE_ROLE - 1)).toEqual([null, null, null, null])
    expect(m[4]).toBeCloseTo(3, 6)
    expect(m[5]).toBeCloseTo(4, 6)
  })

  it('exclut les points creux du numérateur ET du dénominateur', () => {
    const serie = serieDe([2, 2, 2, 2, 100], [10, 10, 10, 10, PLANCHER_MESURE - 1])
    expect(moyenneGlissante(serie.points)[4]).toBeCloseTo(2, 6)
  })
})

describe('rolesFenetre', () => {
  const seuils = { bas: -2, haut: 2 }

  it('ne rend AUCUN rôle sur les quatre premiers matchs', () => {
    const serie = serieDe([6, 6, 6, 6, 6])
    const roles = rolesFenetre(serie.points, seuils)
    expect(roles.slice(0, 4)).toEqual([null, null, null, null])
    expect(roles[4]).toBe('sniper')
  })

  it('un match ATYPIQUE ne fait pas basculer l’étiquette (fenêtre de 5)', () => {
    const serie = serieDe([6, 6, 6, 6, -6])
    // Moyenne de la fenêtre = (6+6+6+6-6)/5 = 3,6 → toujours au-dessus du tiers haut.
    expect(rolesFenetre(serie.points, seuils)[4]).toBe('sniper')
  })

  it('ne rend pas de rôle sur un match CREUX, même fenêtre pleine', () => {
    const serie = serieDe([6, 6, 6, 6, 6], [10, 10, 10, 10, PLANCHER_MESURE - 1])
    expect(rolesFenetre(serie.points, seuils)[4]).toBeNull()
  })

  it('ne rend aucun rôle sans seuils (rien de mesuré sur la période)', () => {
    const serie = serieDe([6, 6, 6, 6, 6])
    expect(rolesFenetre(serie.points, null)).toEqual(Array(5).fill(null))
  })
})

describe('taillePoint', () => {
  it('projette les frags mesurés sur la plage réelle, milieu si elle est plate', () => {
    expect(taillePoint(5, 5, 25)).toBeLessThan(taillePoint(25, 5, 25))
    expect(taillePoint(7, 7, 7)).toBe(taillePoint(99, 7, 7))
  })
})

describe('quantileLineaire', () => {
  it('interpole entre les deux rangs encadrants', () => {
    expect(quantileLineaire([0, 10], 0.5)).toBeCloseTo(5, 6)
    expect(quantileLineaire([4], 0.9)).toBe(4)
    expect(quantileLineaire([], 0.5)).toBe(0)
  })
})

describe('typage', () => {
  it('un point porte sa médiane BRUTE à côté de son écart — l’infobulle dit les deux', () => {
    const p: PointPortee = serieDe([3])!.points[0]
    expect(p.medianeM).toBe(23)
    expect(p.ecartM).toBe(3)
  })
})

// ─── La GRANDEUR (E1, D24 du 2026-09-22) : le même nuage lit la hauteur ───────────────
describe('grandeur hauteur', () => {
  const profilHauteur = (
    i: number,
    joueurs: Array<Partial<MatchRangePlayer> & { xuid: string }>,
  ): MatchRangeProfile => ({
    match_id: `h${i}`,
    played_at: `2026-09-${String(10 + i).padStart(2, '0')}T20:00:00Z`,
    lobby_median_m: 20,
    lobby_measured: 80,
    lobby_elevation_median_m: 0.4,
    players: joueurs.map((j) => ({
      gamertag: j.xuid,
      median_m: 20,
      lobby_delta_m: 0,
      measured: 12,
      ...j,
    })) as MatchRangePlayer[],
  })

  it('projette le dénivelé, pas la distance', () => {
    const profils = ordonnerProfils([
      profilHauteur(1, [
        { xuid: 'A', median_m: 30, lobby_delta_m: 10, elevation_median_m: 2.5, elevation_lobby_delta_m: 2.1 },
      ]),
    ])
    const [serie] = seriesPortee(profils, ['A'], 'hauteur')
    expect(serie.points[0].medianeM).toBeCloseTo(2.5, 6)
    expect(serie.points[0].ecartM).toBeCloseTo(2.1, 6)
    // La même entrée en portée rend la distance — la grandeur est le seul commutateur.
    expect(seriesPortee(profils, ['A'])[0].points[0].ecartM).toBe(10)
  })

  it('un (match, joueur) SANS dénivelé servi n’est pas un point creux : il n’existe pas', () => {
    const profils = ordonnerProfils([
      profilHauteur(1, [
        { xuid: 'A', elevation_median_m: 1, elevation_lobby_delta_m: 1 },
        { xuid: 'B' },
      ]),
      profilHauteur(2, [{ xuid: 'A', elevation_median_m: 0, elevation_lobby_delta_m: 0 }]),
    ])
    const series = seriesPortee(profils, ['A', 'B'], 'hauteur')
    expect(series.map((s) => s.gamertag)).toEqual(['A'])
    // 0 m EST une mesure (« à plat ») : le deuxième point existe bel et bien.
    expect(series[0].points).toHaveLength(2)
    expect(series[0].points[1].ecartM).toBe(0)
  })

  it('mesureDeJoueur rend null dès qu’une des deux valeurs de hauteur manque', () => {
    const base: MatchRangePlayer = { xuid: 'A', median_m: 20, lobby_delta_m: 0, measured: 9 }
    expect(mesureDeJoueur(base, 'hauteur')).toBeNull()
    expect(mesureDeJoueur({ ...base, elevation_median_m: 1 }, 'hauteur')).toBeNull()
    expect(mesureDeJoueur({ ...base, elevation_lobby_delta_m: 1 }, 'hauteur')).toBeNull()
    expect(mesureDeJoueur(base, 'portee')).toEqual({ medianeM: 20, ecartM: 0 })
  })

  it('les seuils de rôle restent les TIERS des points pleins de la grandeur lue', () => {
    const profils = ordonnerProfils(
      [-3, -1, 0, 1, 3].map((dz, i) =>
        profilHauteur(i, [
          { xuid: 'A', elevation_median_m: dz, elevation_lobby_delta_m: dz },
        ]),
      ),
    )
    const seuils = seuilsRoles(seriesPortee(profils, ['A'], 'hauteur'))!
    expect(seuils.bas).toBeCloseTo(quantileLineaire([-3, -1, 0, 1, 3], 1 / 3), 6)
    expect(seuils.haut).toBeCloseTo(quantileLineaire([-3, -1, 0, 1, 3], 2 / 3), 6)
  })
})
