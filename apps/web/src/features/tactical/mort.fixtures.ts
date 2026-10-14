/**
 * Fixtures partagées des tests de la lecture des coéquipiers à la mort : une mort complète (équipe,
 * riposte), un frag, un bilan de zone et une séquence de huit étapes.
 */
import type { TacticalBilanZone, TacticalContribution, TacticalSequence } from '@/lib/api/types'

/** Une mort : instant du match 462 000 ms (countdown compris), horloge AFFICHÉE du rejeu 440 000 ms (7:20). */
export const MORT_COMPLETE: TacticalContribution = {
  match_id: 'm1',
  instant_ms: 462_000,
  instant_affiche_ms: 440_000,
  clock: 'match',
  xuid: '111',
  match_started_at: '2026-10-12T20:00:00Z',
  replay_available: true,
  face: 'mort',
  autre_gamertag: 'Hollow Vex',
  arme_label: 'BR75',
  point: { x: 10, y: 10 },
  autre_point: { x: 25, y: 10 },
  equipe: {
    statut: 'accompagnee',
    rayon_m: 18,
    coequipiers: [
      { xuid: '222', gamertag: 'Kestrel 77', statut: 'visible', dans_le_rayon: true, point: { x: -2, y: 10 }, distance_mort_m: 12, distance_tueur_m: 27 },
      { xuid: '333', gamertag: 'NovaDrift', statut: 'visible', dans_le_rayon: false, point: { x: 10, y: -10 }, distance_mort_m: 20, distance_tueur_m: 25 },
      { xuid: '444', gamertag: 'Brisk Otter', statut: 'waiting', dans_le_rayon: false },
    ],
  },
  riposte: { type: 'tueur_abattu', delai_ms: 1800, par_gamertag: 'Kestrel 77', par_xuid: '222', point: { x: -2, y: 12 } },
}

export const FRAG_ABATTU_EN_RETOUR: TacticalContribution = {
  match_id: 'm1',
  instant_ms: 470_000,
  instant_affiche_ms: 448_000,
  clock: 'match',
  xuid: '111',
  match_started_at: '2026-10-12T20:00:00Z',
  replay_available: true,
  face: 'frag',
  autre_gamertag: 'Cible',
  point: { x: 10, y: 10 },
  autre_point: { x: 30, y: 10 },
  riposte: { type: 'abattu_en_retour', delai_ms: 1500, par_gamertag: 'Bot Elite' },
}

export const BILAN: TacticalBilanZone = {
  morts: {
    total: 14,
    examinees: 14,
    accompagnees: 4,
    visibles_hors_rayon: 6,
    aucun_visible: 1,
    equipe_a_terre: 3,
    rose: [1, 4, 5, 2, 0, 0, 1, 1],
    mediane_tueur_m: 16,
    mediane_plus_proche_m: 21,
    tueur_abattu: { n: 5, m: 14 },
    fenetre_riposte_ms: 3000,
  },
  frags: { abattu_en_retour: { n: 2, m: 9 }, fenetre_riposte_ms: 3000 },
}

/** Séquence : la victime court vers +x et meurt en (10, 10) ; un coéquipier la suit ; le tueur est immobile. */
export const SEQUENCE: TacticalSequence = {
  rayon_m: 18,
  traine_ms: 4000,
  pas_ms: 200,
  mort: { x: 10, y: 10 },
  acteurs: [
    {
      role: 'victime',
      xuid: '111',
      gamertag: 'JGtm',
      points: [-4000, -3800, -3600, -3400, -3200, -3000, -2000, -1000, 0].map((offset_ms) => ({
        offset_ms,
        x: 10 + offset_ms / 400,
        y: 10,
      })),
    },
    {
      role: 'tueur',
      xuid: '900',
      gamertag: 'Hollow Vex',
      points: [-4000, -3000, -2000, -1000, 0, 1000].map((offset_ms) => ({ offset_ms, x: 25, y: 10, coupure: offset_ms === -1000 })),
    },
    {
      role: 'coequipier',
      xuid: '222',
      gamertag: 'Kestrel 77',
      points: [-4000, -3000, -2000, -1000, 0, 1000, 2000, 3000].map((offset_ms) => ({ offset_ms, x: -2 + offset_ms / 1000, y: 10 })),
    },
  ],
  etapes: [-4000, -3000, -2000, -1000, 0, 1000, 2000, 3000].map((offset_ms) => ({
    offset_ms,
    instant_affiche_ms: 440_000 + offset_ms,
    centre: offset_ms < 0 ? { x: 10 + offset_ms / 400, y: 10 } : { x: 10, y: 10 },
    dans_le_rayon: offset_ms < 0 ? 0 : 1,
    dans_le_rayon_xuids: offset_ms < 0 ? [] : ['222'],
    positionnes: 1,
  })),
}
