/**
 * squadEchangeMaquette.logic.test — les décisions PURES ajoutées le 2026-09-13 pour les six
 * cartes de « L'échange » (maquette 4c520da6) : le compte, le donné/reçu, et le taux par
 * session.
 *
 * Fichier à part de `squadEchange.logic.test.ts` : celui-ci garde la matrice, les délais et
 * le constat du moment, qui n'ont aucune de ces règles en commun.
 */
import { describe, expect, it } from 'vitest'

import type { SquadEchange } from '@/lib/api/types'

import {
  compteEchange,
  donneRecuParJoueur,
  donneRecuSeries,
  libelleCourtSession,
  tauxSessionSeries,
} from './squadEchange.logic'

function couverture(brut: number, n: number, matchs = 10) {
  return {
    taux: n > 0 ? brut / n : 0,
    brut,
    par_match: matchs > 0 ? brut / matchs : 0,
    n,
    echantillon_faible: n < 30,
  }
}

function echangeDe(over: Partial<SquadEchange> = {}): SquadEchange {
  return {
    joueurs: [
      { xuid: 'x1', gamertag: 'Alice' },
      { xuid: 'x2', gamertag: 'Bob' },
    ],
    cellules: [
      {
        vengeur_xuid: 'x1',
        vengeur_gamertag: 'Alice',
        venge_xuid: 'x2',
        venge_gamertag: 'Bob',
        nombre: 7,
        par_match: 0.7,
      },
      {
        vengeur_xuid: 'x2',
        vengeur_gamertag: 'Bob',
        venge_xuid: 'x1',
        venge_gamertag: 'Alice',
        nombre: 3,
        par_match: 0.3,
      },
    ],
    delais: [],
    fenetre_ms: 5000,
    couverture: couverture(40, 100, 10),
    habituel: couverture(30, 100, 10),
    matchs_habituel: 50,
    matchs_mesures: 10,
    matchs_total: 12,
    delai_median_ms: 2400,
    taux_par_session: [],
    ...over,
  } as SquadEchange
}

describe('compteEchange', () => {
  it('les morts SANS RÉPONSE sont une soustraction, jamais un second taux', () => {
    const c = compteEchange(echangeDe())
    expect(c.mortsEquipe).toBe(100)
    expect(c.sansReponse).toBe(60)
  })

  it('la quantité par match se divise par les matchs MESURÉS, pas par ceux du filtre', () => {
    // 60 morts sans réponse sur 10 matchs mesurés (et non 12 matchs filtrés) : diviser
    // par le filtre ferait varier la grandeur avec la couverture de film, pas avec le jeu.
    expect(compteEchange(echangeDe()).sansReponseParMatch).toBe(6)
  })

  it('aucun match mesuré : pas de quantité par match INVENTÉE', () => {
    expect(compteEchange(echangeDe({ matchs_mesures: 0 })).sansReponseParMatch).toBeNull()
  })

  it('le délai médian passe en SECONDES, et vaut null quand aucun échange n’est survenu', () => {
    expect(compteEchange(echangeDe()).delaiMedianS).toBe(2.4)
    expect(compteEchange(echangeDe({ delai_median_ms: 0 })).delaiMedianS).toBeNull()
  })

  it('l’écart se TAIT quand le périmètre couvre tout l’historique', () => {
    // Cardinalités égales = les deux ensembles sont identiques : l'écart est nul par
    // construction, et l'afficher ferait croire à une mesure.
    const plein = compteEchange(echangeDe({ matchs_total: 50, matchs_habituel: 50 }))
    expect(plein.pleinHistorique).toBe(true)
    const filtre = compteEchange(echangeDe())
    expect(filtre.pleinHistorique).toBe(false)
    expect(filtre.ecartPoints).toBe(10)
  })
})

describe('donneRecuParJoueur', () => {
  it('somme la LIGNE (donné) et la COLONNE (reçu) de chaque joueur du roster', () => {
    expect(donneRecuParJoueur(echangeDe())).toEqual([
      { gamertag: 'Alice', donne: 7, recu: 3 },
      { gamertag: 'Bob', donne: 3, recu: 7 },
    ])
  })

  it('un roster sans aucune cellule rend des zéros, pas une liste vide', () => {
    // Le joueur EXISTE au roster : sa barre à zéro est un fait mesuré, son absence non.
    expect(donneRecuParJoueur(echangeDe({ cellules: [] }))).toEqual([
      { gamertag: 'Alice', donne: 0, recu: 0 },
      { gamertag: 'Bob', donne: 0, recu: 0 },
    ])
  })

  it('projette deux composantes par joueur, sous les clés fournies par l’appelant', () => {
    const [serie] = donneRecuSeries(echangeDe(), 'Donné', 'Reçu')
    expect(serie.datapoints[0]).toEqual({
      category: 'Alice',
      components: { 'Donné': 7, 'Reçu': 3 },
    })
  })
})

describe('tauxSessionSeries', () => {
  const sessions = [
    {
      session_label: '02/08/2026 21:09-22:52 (11)',
      couverture: couverture(3, 10, 2),
      matchs_mesures: 2,
    },
    {
      session_label: '05/08/2026 20:15-23:01 (9)',
      couverture: couverture(6, 10, 2),
      matchs_mesures: 2,
    },
  ]

  it('rend UNE série, en POURCENTS, dans l’ordre servi par le serveur', () => {
    const series = tauxSessionSeries(echangeDe({ taux_par_session: sessions }))
    expect(series).toHaveLength(1)
    expect(series[0].datapoints).toEqual([
      { x: '02/08/2026', y: 30 },
      { x: '05/08/2026', y: 60 },
    ])
  })

  it('l’axe ne porte que la DATE : le libellé complet chevauche sur quarante soirées', () => {
    expect(libelleCourtSession('13/10/2025 22:27-22:46 (3)')).toBe('13/10/2025')
    // Sans espace, rien n'est coupé : on ne tronque jamais à l'aveugle.
    expect(libelleCourtSession('13/10/2025')).toBe('13/10/2025')
  })

  it('aucune session : AUCUNE série (et non une série vide, qui tracerait un axe nu)', () => {
    expect(tauxSessionSeries(echangeDe({ taux_par_session: [] }))).toEqual([])
  })
})
