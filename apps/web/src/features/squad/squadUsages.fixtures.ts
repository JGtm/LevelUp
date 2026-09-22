/**
 * Décor PARTAGÉ des tests de l'onglet Usages : la réponse de page qui porte le bloc
 * `equipment_usage`, utilisée par le smoke d'Usages ET par le ratchet de Synergies
 * (« l'équipement n'est plus ici »). Une DDL de test recopiée dans deux fichiers dérive
 * sans que rien ne le dise — celle-ci vit à un seul endroit (même raison que
 * `squadEchange.fixtures.ts`).
 */
import type { TeammatesPageResponse } from '@/lib/api/types'

/** Réponse de page minimale portant un bloc `equipment_usage` mesuré. */
export function pageWithEquipmentUsage(): TeammatesPageResponse {
  return {
    options: [],
    teammates: [],
    total_matches: 42,
    session_labels: { solo: [], squad: [] },
    friends_count: 0,
    tracked_players: [{ xuid: 'f1', gamertag: 'Madina' }],
    equipment_usage: {
      available: true,
      matches_measured: 42,
      matches_total: 42,
      tracked_players: [{ xuid: 'f1', gamertag: 'Madina' }],
      players: [
        { xuid: 'me', taken: 88, used: 55, kept: 9, dropped: 24, pad_pickups: 41 },
        { xuid: 'f1', taken: 74, used: 60, kept: 4, dropped: 10, pad_pickups: 31 },
      ],
    },
  } as TeammatesPageResponse
}

/** Réponse de page SANS aucun usage : ni frags, ni équipement, ni formes retenues. */
export function pageWithoutAnyUsage(): TeammatesPageResponse {
  return {
    options: [],
    teammates: [],
    total_matches: 12,
    session_labels: { solo: [], squad: [] },
    friends_count: 0,
  } as TeammatesPageResponse
}
