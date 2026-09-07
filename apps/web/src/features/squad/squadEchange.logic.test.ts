import { describe, it, expect } from 'vitest'

import type { SquadEchange } from '@/lib/api/types'

import { couverture, echangeDe } from './squadEchange.fixtures'
import {
  ECART_BADGE_VENGEANCES,
  ECART_CONSTAT_POINTS,
  PLANCHER_MORTS,
  constatDuMoment,
  couvertureParJoueur,
  delaisSeries,
  ecartEchange,
  extremesCouverture,
  matriceSeries,
  matriceVide,
  resumeDelais,
  trendEcart,
} from './squadEchange.logic'

// Le DÉCOR vit dans `squadEchange.fixtures` : ce fichier en avait une COPIE, déjà
// divergente (12 matchs par défaut ici, autre valeur là-bas), ce qui rendait deux
// tests du même invariant incomparables. Correction W7 (revue du 2026-09-06).

// ─── LE CONSTAT DU MOMENT : DEUX SEUILS, TOUS LES DEUX NÉCESSAIRES ────────────────

describe('constatDuMoment — la règle de seuil du plan (§1, 2026-09-06)', () => {
  it('N’EST PAS rendu sous le plancher de morts, même avec un écart énorme', () => {
    // 29 morts, 50 points d'écart : la carte reste absente. Sous le plancher,
    // l'écart n'est pas un signal, c'est un tirage.
    const e = echangeDe({
      couverture: couverture(20, PLANCHER_MORTS - 1),
      habituel: couverture(4, 40),
    })
    expect(constatDuMoment(e)).toBeNull()
  })

  it('N’EST PAS rendu sous le seuil d’écart, même avec un gros échantillon', () => {
    // 400 morts mesurées, 4 points d'écart : rien à dire.
    const e = echangeDe({
      couverture: couverture(176, 400), // 44,0 %
      habituel: couverture(160, 400), // 40,0 %
    })
    expect(constatDuMoment(e)).toBeNull()
  })

  it('EST rendu EXACTEMENT au plancher (30 morts) et à l’écart minimal (5 points)', () => {
    const e = echangeDe({
      couverture: couverture(15, PLANCHER_MORTS), // 50,0 %
      habituel: couverture(45, 100), // 45,0 %
    })
    const cap = constatDuMoment(e)
    expect(cap).not.toBeNull()
    expect(cap?.ecartPoints).toBe(ECART_CONSTAT_POINTS)
    expect(cap?.ton).toBe('consolide')
    expect(cap?.morts).toBe(PLANCHER_MORTS)
  })

  it('parle d’ATTENTION quand l’écart est négatif, jamais d’alerte', () => {
    const e = echangeDe({
      couverture: couverture(12, 40), // 30,0 %
      habituel: couverture(40, 100), // 40,0 %
    })
    const cap = constatDuMoment(e)
    expect(cap?.ton).toBe('attention')
    expect(cap?.ecartPoints).toBe(-10)
  })

  it('N’EST PAS rendu sans référence mesurée (un écart contre zéro n’est pas un écart)', () => {
    const e = echangeDe({
      couverture: couverture(20, 40),
      habituel: couverture(0, 0),
    })
    expect(constatDuMoment(e)).toBeNull()
  })

  it('N’EST PAS rendu quand la section est absente du contrat', () => {
    expect(constatDuMoment(null)).toBeNull()
    expect(constatDuMoment(undefined)).toBeNull()
  })
})

// ─── ANTI-BIAIS : 8 MORTS À 100 % NE CLASSE PERSONNE ──────────────────────────

describe('anti-biais — un petit échantillon ne classe personne', () => {
  const petit = echangeDe({
    couverture: couverture(8, 8, 3),
    habituel: couverture(30, 100),
    matchs_total: 3,
    matchs_habituel: 40,
    cellules: [
      { vengeur_xuid: 'x2', vengeur_gamertag: 'Ami', venge_xuid: 'x1', venge_gamertag: 'Moi', nombre: 8, par_match: 8 / 3 },
    ],
  })

  it('le serveur pose le drapeau « échantillon faible » à 8 morts', () => {
    expect(petit.couverture.echantillon_faible).toBe(true)
  })

  it('aucun constat du moment, malgré 100 % contre 30 % (60 points d’écart)', () => {
    expect(constatDuMoment(petit)).toBeNull()
  })

  it('aucun badge « le plus / le moins couvert » : rien ne classe qui que ce soit', () => {
    expect(extremesCouverture(petit)).toBeNull()
  })
})

// ─── LES BADGES : SEULEMENT À ÉCART RÉEL ──────────────────────────────────────

describe('extremesCouverture — badges « le plus / le moins couvert »', () => {
  const troisJoueurs = [
    { xuid: 'x1', gamertag: 'Moi' },
    { xuid: 'x2', gamertag: 'Ami' },
    { xuid: 'x3', gamertag: 'Autre' },
  ]

  function avecVengeances(pourMoi: number, pourAmi: number, pourAutre: number): SquadEchange {
    return echangeDe({
      joueurs: troisJoueurs,
      couverture: couverture(20, 60),
      cellules: [
        { vengeur_xuid: 'x2', vengeur_gamertag: 'Ami', venge_xuid: 'x1', venge_gamertag: 'Moi', nombre: pourMoi, par_match: 0 },
        { vengeur_xuid: 'x1', vengeur_gamertag: 'Moi', venge_xuid: 'x2', venge_gamertag: 'Ami', nombre: pourAmi, par_match: 0 },
        { vengeur_xuid: 'x1', vengeur_gamertag: 'Moi', venge_xuid: 'x3', venge_gamertag: 'Autre', nombre: pourAutre, par_match: 0 },
      ],
    })
  }

  it('rien sous l’écart minimal : 2 vengeances d’écart ne désignent personne', () => {
    expect(extremesCouverture(avecVengeances(5, 4, 3))).toBeNull()
  })

  it('les deux badges EXACTEMENT à l’écart minimal', () => {
    const ex = extremesCouverture(avecVengeances(5, 4, 5 - ECART_BADGE_VENGEANCES))
    expect(ex).not.toBeNull()
    expect(ex?.plusCouvert.gamertag).toBe('Moi')
    expect(ex?.moinsCouvert.gamertag).toBe('Autre')
  })

  it('compte les vengeances REÇUES (la colonne du vengé), pas les vengeances rendues', () => {
    // Moi venge beaucoup (2 lignes) mais n'est vengé qu'une fois.
    const parJoueur = couvertureParJoueur(avecVengeances(1, 9, 9))
    expect(parJoueur.find((j) => j.gamertag === 'Moi')?.vengeances).toBe(1)
    expect(parJoueur.find((j) => j.gamertag === 'Ami')?.vengeances).toBe(9)
  })

  it('pose les badges EXACTEMENT au plancher d’échantillon (30 morts)', () => {
    // W9 (revue du 2026-09-06) : la borne n'était testée que par le dessous. Un
    // `<` mué en `<=` dans `extremesCouverture` passait inaperçu.
    const auPlancher = echangeDe({
      joueurs: troisJoueurs,
      couverture: couverture(10, PLANCHER_MORTS),
      cellules: [
        { vengeur_xuid: 'x2', vengeur_gamertag: 'Ami', venge_xuid: 'x1', venge_gamertag: 'Moi', nombre: 5, par_match: 0 },
        { vengeur_xuid: 'x1', vengeur_gamertag: 'Moi', venge_xuid: 'x3', venge_gamertag: 'Autre', nombre: 1, par_match: 0 },
      ],
    })
    expect(extremesCouverture(auPlancher)).not.toBeNull()

    const justeEnDessous = echangeDe({
      joueurs: troisJoueurs,
      couverture: couverture(10, PLANCHER_MORTS - 1),
      cellules: (auPlancher.cellules ?? []).slice(),
    })
    expect(extremesCouverture(justeEnDessous)).toBeNull()
  })

  it('rien avec un seul joueur au roster : « le plus couvert » d’un seul ne veut rien dire', () => {
    const solo = echangeDe({
      joueurs: [{ xuid: 'x1', gamertag: 'Moi' }],
      couverture: couverture(20, 60),
      cellules: [],
    })
    expect(extremesCouverture(solo)).toBeNull()
  })
})

// ─── LA MATRICE ───────────────────────────────────────────────────────────────

describe('matriceSeries — orientation, complétude, et AXES', () => {
  const quatre = [
    { xuid: 'x1', gamertag: 'A' },
    { xuid: 'x2', gamertag: 'B' },
    { xuid: 'x3', gamertag: 'C' },
    { xuid: 'x4', gamertag: 'D' },
  ]
  // Le décor partagé : roster [Alice, Bob], une seule case (Bob venge Alice).
  const e = echangeDe()

  it('LIGNE = vengeur (y), COLONNE = vengé (x) — l’orientation de SquadAssistPairsTable', () => {
    const dp = matriceSeries(e)[0].datapoints
    const case42 = dp.find((d) => d.y === 'Bob' && d.x === 'Alice')
    expect(case42?.value).toBe(6)
  })

  // W1 (revue ronde 1, 2026-09-06) — LA MATRICE SE LISAIT DE TRAVERS.
  //
  // Le wrapper déduit ses catégories d'axe de l'ORDRE D'APPARITION des points. Tant
  // que `matriceSeries` sautait la diagonale, la première COLONNE rencontrée était le
  // DEUXIÈME joueur : lignes [A,B,C,D] mais colonnes [B,C,D,A]. Sur un duo, l'axe X
  // sortait exactement inversé par rapport à l'axe Y. Ces deux tests dérivent les axes
  // COMME LE WRAPPER, et tombent si la diagonale disparaît à nouveau.
  const axesDerives = (dp: { x: string; y: string }[]) => ({
    xs: [...new Set(dp.map((d) => d.x))],
    ys: [...new Set(dp.map((d) => d.y))],
  })

  it('rend des axes IDENTIQUES et dans l’ordre du roster (4 joueurs)', () => {
    const { xs, ys } = axesDerives(matriceSeries(echangeDe({ joueurs: quatre }))[0].datapoints)
    expect(ys).toEqual(['A', 'B', 'C', 'D'])
    expect(xs).toEqual(ys)
  })

  it('rend des axes identiques sur un DUO (le cas où l’inversion était totale)', () => {
    const { xs, ys } = axesDerives(matriceSeries(e)[0].datapoints)
    expect(ys).toEqual(['Alice', 'Bob'])
    expect(xs).toEqual(ys)
  })

  it('émet TOUTES les cases, diagonale COMPRISE, celle-ci VIDE', () => {
    const dp = matriceSeries(echangeDe({ joueurs: quatre }))[0].datapoints
    expect(dp).toHaveLength(16) // 4 x 4, rien d'omis
    const diagonale = dp.filter((d) => d.x === d.y)
    expect(diagonale).toHaveLength(4)
    expect(diagonale.every((d) => d.value === null)).toBe(true)
  })

  it('émet les cases hors diagonale à zéro : un 0 mesuré n’est pas une case absente', () => {
    const dp = matriceSeries(e)[0].datapoints
    expect(dp.find((d) => d.y === 'Alice' && d.x === 'Bob')?.value).toBe(0)
  })

  it('matriceVide dit qu’il n’y a aucune vengeance interne à montrer', () => {
    expect(matriceVide(echangeDe({ cellules: [] }))).toBe(true)
    expect(matriceVide(e)).toBe(false)
  })
})

// ─── LA DISTRIBUTION DU DÉLAI ─────────────────────────────────────────────────

describe('délais — les deux barres hors fenêtre sont montrées et jamais comptées', () => {
  const e = echangeDe({
    delais: [
      { debut_ms: 0, fin_ms: 1000, ouvert: false, hors_fenetre: false, nombre: 3 },
      { debut_ms: 1000, fin_ms: 2000, ouvert: false, hors_fenetre: false, nombre: 5 },
      { debut_ms: 2000, fin_ms: 3000, ouvert: false, hors_fenetre: false, nombre: 2 },
      { debut_ms: 3000, fin_ms: 4000, ouvert: false, hors_fenetre: false, nombre: 1 },
      { debut_ms: 4000, fin_ms: 5000, ouvert: false, hors_fenetre: false, nombre: 1 },
      { debut_ms: 5000, fin_ms: 7000, ouvert: false, hors_fenetre: true, nombre: 4 },
      { debut_ms: 7000, fin_ms: 0, ouvert: true, hors_fenetre: true, nombre: 6 },
    ],
  })

  it('les bornes passent en SECONDES, sans re-binning', () => {
    const dp = delaisSeries(e)[0].datapoints
    expect(dp).toHaveLength(7)
    expect(dp[0]).toEqual({ binStart: 0, binEnd: 1, count: 3 })
    expect(dp[5]).toEqual({ binStart: 5, binEnd: 7, count: 4 })
  })

  it('le dernier intervalle est ouvert : sa borne haute vaut sa borne basse', () => {
    const dp = delaisSeries(e)[0].datapoints
    expect(dp[6]).toEqual({ binStart: 7, binEnd: 7, count: 6 })
  })

  it('résume les deux populations séparément — jamais leur somme comme dénominateur', () => {
    expect(resumeDelais(e)).toEqual({ dansLaFenetre: 12, horsFenetre: 10, total: 22 })
  })

  it('rend zéro partout sur une section sans riposte', () => {
    expect(resumeDelais(echangeDe({ delais: [] }))).toEqual({
      dansLaFenetre: 0,
      horsFenetre: 0,
      total: 0,
    })
  })
})

// ─── LA FLÈCHE ────────────────────────────────────────────────────────────────

describe('trendEcart', () => {
  it('rend une flèche explicite pour le zéro (une flèche absente = pas de mesure)', () => {
    expect(trendEcart(7)).toBe('above')
    expect(trendEcart(-7)).toBe('below')
    expect(trendEcart(0)).toBe('near')
  })
})

// ─── L'ÉCART À L'HABITUEL (correction W3) ─────────────────────────────────────

describe('ecartEchange', () => {
  it('rend l’écart signé et son arrondi en points', () => {
    const e = echangeDe({ couverture: couverture(27, 45), habituel: couverture(40, 100) })
    const r = ecartEchange(e)
    expect(r.ecartPoints).toBe(20) // 60,0 % − 40,0 %
    expect(r.ecart).toBeCloseTo(0.2, 6)
    expect(r.pleinHistorique).toBe(false)
  })

  it('rend un écart NÉGATIF quand le périmètre est sous son habituel', () => {
    // Inverser le signe de la soustraction fait tomber ce test — c'est ce qui était
    // impossible tant que le calcul vivait inliné dans le composant.
    const e = echangeDe({ couverture: couverture(9, 45), habituel: couverture(40, 100) })
    expect(ecartEchange(e).ecartPoints).toBe(-20)
  })

  it('signale le plein historique : périmètre == référence', () => {
    const e = echangeDe({ matchs_total: 60, matchs_habituel: 60 })
    expect(ecartEchange(e).pleinHistorique).toBe(true)
  })
})
