/**
 * Décor PARTAGÉ des tests de l'échange : une section de contrat complète, et les
 * deux fabriques qui la dérivent. Une DDL de test recopiée dans trois fichiers
 * dérive sans que rien ne le dise — celle-ci vit à un seul endroit.
 */
import type { SquadEchange } from '@/lib/api/types'

import { PLANCHER_MORTS } from './squadEchange.logic'

export function couverture(brut: number, n: number, matchs = 12) {
  return {
    taux: n > 0 ? brut / n : 0,
    brut,
    par_match: matchs > 0 ? brut / matchs : 0,
    n,
    echantillon_faible: n < PLANCHER_MORTS,
  }
}

export function echangeDe(over: Partial<SquadEchange> = {}): SquadEchange {
  return {
    joueurs: [
      { xuid: 'x1', gamertag: 'Alice' },
      { xuid: 'x2', gamertag: 'Bob' },
    ],
    cellules: [
      { vengeur_xuid: 'x2', vengeur_gamertag: 'Bob', venge_xuid: 'x1', venge_gamertag: 'Alice', nombre: 6, par_match: 0.5 },
    ],
    delais: [
      { debut_ms: 0, fin_ms: 1000, ouvert: false, hors_fenetre: false, nombre: 2 },
      { debut_ms: 1000, fin_ms: 2000, ouvert: false, hors_fenetre: false, nombre: 4 },
      { debut_ms: 2000, fin_ms: 3000, ouvert: false, hors_fenetre: false, nombre: 0 },
      { debut_ms: 3000, fin_ms: 4000, ouvert: false, hors_fenetre: false, nombre: 0 },
      { debut_ms: 4000, fin_ms: 5000, ouvert: false, hors_fenetre: false, nombre: 0 },
      { debut_ms: 5000, fin_ms: 7000, ouvert: false, hors_fenetre: true, nombre: 3 },
      { debut_ms: 7000, fin_ms: 0, ouvert: true, hors_fenetre: true, nombre: 1 },
    ],
    fenetre_ms: 5000,
    couverture: couverture(18, 45),
    habituel: couverture(40, 100),
    matchs_habituel: 60,
    matchs_mesures: 9,
    matchs_total: 12,
    ...over,
  } as SquadEchange
}

