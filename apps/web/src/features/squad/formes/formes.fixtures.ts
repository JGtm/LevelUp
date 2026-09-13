/**
 * formes.fixtures.ts — UNE SOIRÉE MINIATURE au format du bloc `formes_retenues`,
 * calquée sur la session que l'artefact 2ec1b8eb met en scène : un match sans
 * film décodé au milieu d'un scope mesuré, un lobby à deux camps, des socles
 * dont certaines occupations n'ont pas de ramasseur nommé, et un seul match à
 * objectif.
 *
 * Sert aux tests unitaires ET à la vérification visuelle (le harnais Playwright
 * intercepte la réponse de page et y injecte ce bloc) : une seule fixture, donc
 * un seul jeu de chiffres à vérifier à l'écran comme dans les tests.
 */
import type { SquadFormesBlock } from '@/lib/api/types'

/** Les trois joueurs de l'escouade, plus leurs voisins de lobby. */
export const FORMES_MAIN_XUID = 'x-jgtm'
export const FORMES_MATE_A = 'x-madina'
export const FORMES_MATE_B = 'x-choco'

/** Le bloc complet : trois matchs, dont un sans film décodé. */
export function formesFixture(): SquadFormesBlock {
  return {
    available: true,
    matches_total: 3,
    matches_measured: 2,
    main_xuid: FORMES_MAIN_XUID,
    squad: [
      { xuid: FORMES_MAIN_XUID, gamertag: 'JGtm' },
      { xuid: FORMES_MATE_A, gamertag: 'Madina97294' },
      { xuid: FORMES_MATE_B, gamertag: 'Chocoboflor' },
    ],
    weapons: [
      { key: '0a1992bc', label: 'Fusil de précision S7', weapon_key: 'hinf_s7', class: 'heavy' },
      { key: '230447b1', label: 'Carabine Vestige', weapon_key: 'hinf_vk78', class: 'precision' },
      { key: '71ab0a2c', label: 'SPNKr', weapon_key: 'hinf_m41', class: 'heavy' },
      { key: 'a1b2c3d4', label: '', weapon_key: '', class: 'other' },
    ],
    matches: [
      {
        match_id: 'm-no-film',
        start_time: '2026-07-31T17:22:00Z',
        mode_label: 'Bastion',
        map_label: 'Perilous',
        measured: false,
        player_team: 0,
        team_size: 4,
        lobby_size: 8,
      },
      {
        match_id: 'm-slayer',
        start_time: '2026-07-31T17:32:00Z',
        mode_label: 'Assassin',
        map_label: 'Curfew',
        duration_seconds: 564,
        measured: true,
        player_team: 0,
        team_size: 4,
        lobby_size: 8,
        pad_named: 6,
        pad_unnamed: 4,
        weapon_pads: [
          { weapon: '71ab0a2c', occupations: 4, named: 3 },
          { weapon: '0a1992bc', occupations: 3, named: 2 },
          { weapon: 'a1b2c3d4', occupations: 2, named: 1 },
        ],
        lobby: [
          {
            xuid: FORMES_MAIN_XUID,
            gamertag: 'JGtm',
            team_id: 0,
            camo: 2,
            overshield: 1,
            wall: 0,
            grapple: 1,
            dropped: 3,
            pad_pickups: 2,
            pads_by_weapon: { '71ab0a2c': 2 },
          },
          {
            xuid: FORMES_MATE_A,
            gamertag: 'Madina97294',
            team_id: 0,
            camo: 5,
            overshield: 0,
            wall: 1,
            grapple: 0,
            dropped: 2,
            pad_pickups: 1,
            pads_by_weapon: { '0a1992bc': 1 },
          },
          {
            xuid: FORMES_MATE_B,
            gamertag: 'Chocoboflor',
            team_id: 0,
            camo: 0,
            overshield: 0,
            wall: 0,
            grapple: 2,
            dropped: 2,
            pad_pickups: 0,
          },
          {
            xuid: 'x-ally-4',
            gamertag: 'DectroPK',
            team_id: 0,
            camo: 1,
            overshield: 0,
            wall: 0,
            grapple: 0,
            dropped: 1,
            pad_pickups: 1,
            pads_by_weapon: { a1b2c3d4: 1 },
          },
          {
            xuid: 'x-foe-1',
            gamertag: 'PhoenixVC89',
            team_id: 1,
            camo: 1,
            overshield: 2,
            wall: 0,
            grapple: 0,
            dropped: 4,
            pad_pickups: 2,
            pads_by_weapon: { '71ab0a2c': 1, '0a1992bc': 1 },
          },
          {
            xuid: 'x-foe-2',
            gamertag: 'SLAM ME RAW',
            team_id: 1,
            camo: 0,
            overshield: 0,
            wall: 2,
            grapple: 0,
            dropped: 1,
            pad_pickups: 0,
          },
        ],
      },
      {
        match_id: 'm-flag',
        start_time: '2026-07-31T17:56:00Z',
        mode_label: 'Drapeau',
        map_label: 'Aquarius',
        duration_seconds: 420,
        measured: true,
        player_team: 1,
        team_size: 4,
        lobby_size: 8,
        pad_named: 3,
        pad_unnamed: 1,
        weapon_pads: [{ weapon: '230447b1', occupations: 4, named: 3 }],
        lobby: [
          {
            xuid: FORMES_MAIN_XUID,
            gamertag: 'JGtm',
            team_id: 1,
            camo: 0,
            overshield: 0,
            wall: 2,
            grapple: 3,
            dropped: 1,
            pad_pickups: 1,
            pads_by_weapon: { '230447b1': 1 },
          },
          {
            xuid: FORMES_MATE_A,
            gamertag: 'Madina97294',
            team_id: 1,
            camo: 3,
            overshield: 1,
            wall: 0,
            grapple: 1,
            dropped: 0,
            pad_pickups: 1,
            pads_by_weapon: { '230447b1': 1 },
          },
          {
            xuid: 'x-foe-3',
            gamertag: 'Baggsville',
            team_id: 0,
            camo: 0,
            overshield: 0,
            wall: 0,
            grapple: 0,
            dropped: 2,
            pad_pickups: 1,
            pads_by_weapon: { '230447b1': 1 },
          },
        ],
        objective: {
          family: 'ctf',
          columns: [
            { key: 'flag_captures', role: 'take' },
            { key: 'flag_returns', role: 'defend' },
            { key: 'time_as_flag_carrier_seconds', role: 'hold', duration: true },
          ],
          players: [
            {
              xuid: FORMES_MAIN_XUID,
              team_id: 1,
              values: {
                flag_captures: 0,
                flag_returns: 2,
                time_as_flag_carrier_seconds: 0,
              },
            },
            {
              xuid: FORMES_MATE_A,
              team_id: 1,
              values: {
                flag_captures: 3,
                flag_returns: 0,
                time_as_flag_carrier_seconds: 31.2,
              },
            },
            {
              xuid: 'x-foe-3',
              team_id: 0,
              values: {
                flag_captures: 1,
                flag_returns: 1,
                time_as_flag_carrier_seconds: 46.2,
              },
            },
          ],
        },
      },
    ],
  }
}
