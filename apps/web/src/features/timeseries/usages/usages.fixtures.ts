/**
 * usages.fixtures.ts — un périmètre solo témoin au format du contrat, tiré des chiffres
 * d'illustration de la maquette v4 (`.ai/V7.5/MAQUETTE_TIMESERIES_USAGES_2026-10-05.html` ; relevés
 * `.ai/V7.5/MESURES_TIMESERIES_USAGES_2026-10-05.md`) : bilan 111 / 87 bonus et 229 / 231 armes
 * spéciales, vies 1 558 près / 301 seul (1 250 / 262 frags, 59 écartées), mur 52 · 0 · 32 pour moi et
 * 146 · 7 · 151 pour le reste de mon camp, capteur 6 · 1 · 54, grappin 84 et propulseur 65 lâchés.
 * Quatre matchs suffisent à porter ces comptes (le bloc publie des totaux) ; les cartes sont trois
 * colonnes, dont « Autres cartes ». Aucune lecture de base : des données figées.
 */
import type { SoloEmpriseBlock, TimeseriesLivesNearTeammate, TimeseriesMatchRow } from '@/lib/api/types'

export const ME = 'xj'

export const MATCH_ROWS = [
  { match_id: 'm1', start_time: '2026-07-03T19:00:00Z', map_name: 'Carte Alpha', map_name_fr: 'Carte Alpha', outcome: 2, playlist_name: 'Liste témoin', dominance_flag: 1 },
  { match_id: 'm2', start_time: '2026-07-11T19:10:00Z', map_name: 'Carte Alpha', outcome: 3, playlist_name: 'Liste témoin' },
  { match_id: 'm3', start_time: '2026-08-27T20:00:00Z', map_name: 'Carte Bravo', map_name_fr: 'Carte Bravo', outcome: 2, playlist_name: 'Liste témoin' },
  { match_id: 'm4', start_time: '2026-09-22T21:00:00Z', map_name: 'Carte Charlie', map_name_fr: 'Carte Charlie (fr)', outcome: 4, playlist_name: 'Liste témoin' },
] as unknown as TimeseriesMatchRow[]

const count = (us: number, them: number) => ({ us, them })

export function soloEmprise(): SoloEmpriseBlock {
  return {
    matches_total: 4,
    matches_measured: 3,
    players: [{ xuid: ME, gamertag: 'JGtm' }],
    resources: [
      { resource: 'powerup', taken: count(111, 87), matches_measured: 3, outcomes: { us: { taken: 111, used: 99, kept: 4, dropped: 8 }, them: { taken: 87, used: 80, kept: 2, dropped: 5 } } },
      { resource: 'power_weapon', taken: count(229, 231), matches_measured: 3 },
      { resource: 'rack', taken: count(5, 3), matches_measured: 3 },
    ],
    objects: [
      { resource: 'powerup', key: 'powerup_camo', taken: count(60, 50), squad: [{ xuid: ME, taken: 20, kept: 1, dropped: 2 }, { taken: 40 }] },
      { resource: 'powerup', key: 'powerup_overshield', taken: count(51, 37), squad: [{ xuid: ME, taken: 11 }, { taken: 40 }] },
      { resource: 'power_weapon', key: 'spnkr', label: 'Lance-roquettes', taken: count(129, 131), squad: [{ xuid: ME, taken: 30 }, { taken: 99 }] },
      { resource: 'power_weapon', key: 'sniper', label: 'Fusil de précision', taken: count(100, 100), squad: [{ taken: 100 }] },
      { resource: 'rack', key: 'br', label: 'Fusil de combat', taken: count(5, 3), squad: [{ xuid: ME, taken: 2 }, { taken: 3 }] },
    ],
    matches: [
      { match_id: 'm1', has_film: true, team_known: true, tiers: 'measured', resources: [{ resource: 'powerup', taken: count(60, 40), objects: [] }], power_weapon_kills: count(8, 4) },
      { match_id: 'm2', has_film: true, team_known: true, tiers: 'measured', resources: [{ resource: 'powerup', taken: count(30, 30), objects: [] }], power_weapon_kills: count(3, 6) },
      { match_id: 'm3', has_film: true, team_known: true, tiers: 'measured', resources: [{ resource: 'powerup', taken: count(21, 17), objects: [] }], power_weapon_kills: count(5, 5) },
      { match_id: 'm4', has_film: false, team_known: false, resources: [], power_weapon_kills: count(2, 1) },
    ],
    production: [
      { resource: 'powerup', kills: count(180, 150), exposure: { kind: 'effect_ms', value: count(600000, 500000), kills: count(180, 150) }, yield_us: 18, yield_them: 18, relative_gap: 0 },
      { resource: 'power_weapon', kills: count(150, 160) },
    ],
    maps: [
      {
        map_key: 'aq', map_label: 'Carte Alpha', matches: 2, matches_filmed: 2, matches_measured: 2, matches_tiers: 2, vehicles_measured: 0,
        wins: 1, losses: 1, others: 0,
        resources: [
          { resource: 'powerup', taken: count(90, 70), objects: [{ resource: 'powerup', key: 'powerup_camo', taken: count(50, 40), squad: [{ xuid: ME, taken: 15 }, { taken: 35 }] }] },
          { resource: 'power_weapon', taken: count(120, 100), objects: [] },
        ],
        power_weapon_kills: count(11, 10),
      },
      {
        map_key: 'rc', map_label: 'Carte Bravo', matches: 1, matches_filmed: 1, matches_measured: 1, matches_tiers: 0, vehicles_measured: 0,
        wins: 1, losses: 0, others: 0,
        resources: [{ resource: 'powerup', taken: count(21, 17), objects: [] }],
        power_weapon_kills: count(5, 5),
      },
      {
        matches: 1, matches_filmed: 0, matches_measured: 0, matches_tiers: 0, vehicles_measured: 0, other_maps: 3,
        wins: 0, losses: 0, others: 1, resources: [], power_weapon_kills: count(2, 1),
      },
    ],
    equipment: {
      matches_measured: 3,
      families: [
        { family: 'grapple', measured: false, dropped_me: 84, dropped_lobby: 130 },
        {
          family: 'wall', measured: true, me: { taken: 23, used: 52, kept: 0, dropped: 32 }, rest: { taken: 60, used: 146, kept: 7, dropped: 151 },
          lobby: { taken: 120, used: 400, kept: 10, dropped: 300 },
        },
        {
          family: 'sensor', measured: true, me: { taken: 12, used: 6, kept: 1, dropped: 54 }, rest: { taken: 30, used: 16, kept: 4, dropped: 183 },
          lobby: { taken: 50, used: 30, kept: 6, dropped: 300 },
        },
        // Tenue par le seul adversaire : gardée, mes comptes et ceux de mon camp à zéro.
        {
          family: 'shroud_screen', measured: true, me: { taken: 0, used: 0, kept: 0, dropped: 0 }, rest: { taken: 0, used: 0, kept: 0, dropped: 0 },
          lobby: { taken: 2, used: 3, kept: 0, dropped: 1 },
        },
        // Tenue par personne dans le lobby : retirée.
        {
          family: 'repair_field', measured: true, me: { taken: 0, used: 0, kept: 0, dropped: 0 }, rest: { taken: 0, used: 0, kept: 0, dropped: 0 },
          lobby: { taken: 0, used: 0, kept: 0, dropped: 0 },
        },
        { family: 'thruster', measured: false, dropped_me: 65, dropped_lobby: 90 },
      ],
    },
  } as SoloEmpriseBlock
}

/** Halo 5 (sans film) : la feuille de match seule — les frags aux armes spéciales. */
export function soloEmpriseSansFilm(): SoloEmpriseBlock {
  return {
    matches_total: 4,
    matches_measured: 0,
    film_unavailable: 'film_unsupported',
    players: [{ xuid: ME, gamertag: 'JGtm' }],
    resources: [],
    objects: [],
    matches: soloEmprise().matches!.map((m) => ({ match_id: m.match_id, has_film: false, team_known: false, resources: [], power_weapon_kills: m.power_weapon_kills })),
    production: [{ resource: 'power_weapon', kills: count(18, 16) }],
    maps: [],
  } as SoloEmpriseBlock
}

export function lives(): TimeseriesLivesNearTeammate {
  return {
    near: { lives: 1558, kills: 1250 },
    alone: { lives: 301, kills: 262 },
    excluded_unlocated: 59,
    excluded_no_radar: 0,
    excluded_unpublishable: 0,
    matches_read: 4,
    matches_without_radar: 0,
  }
}
