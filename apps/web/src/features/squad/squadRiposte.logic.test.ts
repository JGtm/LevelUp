import { describe, it, expect } from 'vitest'

import type { SquadEchange } from '@/lib/api/types'

import { couverture, echangeDe } from './squadRiposte.fixtures'
import {
  ECART_BADGE_RIPOSTES,
  FENETRE_TENDANCE,
  appelRiposte,
  friseRiposte,
  moyenneGlissante,
  PLANCHER_MORTS,
  couvertureParJoueur,
  delaisSeries,
  extremesCouverture,
  matriceSeries,
  matriceVide,
  positionMediane,
  resumeDelais,
} from './squadRiposte.logic'

// Le DÉCOR vit dans `squadRiposte.fixtures` : ce fichier en avait une COPIE, déjà
// divergente (12 matchs par défaut ici, autre valeur là-bas), ce qui rendait deux
// tests du même invariant incomparables. Correction W7 (revue du 2026-09-06).


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

  it('rien sous l’écart minimal : 2 ripostes d’écart ne désignent personne', () => {
    expect(extremesCouverture(avecVengeances(5, 4, 3))).toBeNull()
  })

  it('les deux badges EXACTEMENT à l’écart minimal', () => {
    const ex = extremesCouverture(avecVengeances(5, 4, 5 - ECART_BADGE_RIPOSTES))
    expect(ex).not.toBeNull()
    expect(ex?.plusCouvert.gamertag).toBe('Moi')
    expect(ex?.moinsCouvert.gamertag).toBe('Autre')
  })

  it('compte les ripostes REÇUES (la colonne), pas les ripostes rendues', () => {
    // Moi venge beaucoup (2 lignes) mais n'est vengé qu'une fois.
    const parJoueur = couvertureParJoueur(avecVengeances(1, 9, 9))
    expect(parJoueur.find((j) => j.gamertag === 'Moi')?.ripostes).toBe(1)
    expect(parJoueur.find((j) => j.gamertag === 'Ami')?.ripostes).toBe(9)
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

  it('LIGNE = celui qui riposte (y), COLONNE = celui pour qui (x)', () => {
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

  it('matriceVide dit qu’il n’y a aucune riposte interne à montrer', () => {
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


// ─── LE CHIFFRE D'APPEL (D19) ─────────────────────────────────────────────────

describe('appelRiposte — le chiffre d’appel de la carte « Riposte »', () => {
  it('ne réinvente aucun quotient : taux et brut viennent du serveur', () => {
    const e = echangeDe({ couverture: couverture(99, 511, 128), habituel: couverture(40, 100) })
    const a = appelRiposte(e)
    expect(a.taux).toBeCloseTo(99 / 511, 6)
    expect(a.ripostes).toBe(99)
    expect(a.sansReponse).toBe(511 - 99)
    expect(a.mortsEquipe).toBe(511)
  })

  it('rend `null` sur le délai médian quand aucune riposte n’est survenue', () => {
    const e = echangeDe({ delai_median_ms: 0 } as Partial<SquadEchange>)
    expect(appelRiposte(e).delaiMedianS).toBeNull()
  })

  it('n’expose PLUS d’écart à l’habituel : la phrase sous le donut a disparu', () => {
    // La comparaison à l'habituel se LIT sur la frise (repère tireté), elle n'est plus
    // écrite. Ce test tombe si quelqu'un remet un champ d'écart au chiffre d'appel sans
    // remettre ce que la décision du 2026-09-22 a retiré.
    const a = appelRiposte(echangeDe()) as unknown as Record<string, unknown>
    expect(Object.keys(a)).not.toContain('ecart')
    expect(Object.keys(a)).not.toContain('ecartPoints')
    expect(Object.keys(a)).not.toContain('pleinHistorique')
  })
})

// ─── LA FRISE : UNE SOIRÉE = UN BÂTON, PLUS AUCUN REPLI EN LISTE ──────────────

describe('friseRiposte — la frise soirée par soirée', () => {
  const avecSessions = (taux: number[], morts = 50, dansLeFiltre: boolean[] = []) =>
    echangeDe({
      habituel: couverture(20, 100),
      taux_par_session: taux.map((t, i) => ({
        session_label: `1${i}/09 22:00–23:00 (4)`,
        matchs_mesures: 4,
        dans_le_filtre: dansLeFiltre[i] ?? true,
        couverture: { taux: t, brut: Math.round(t * morts), par_match: 1, n: morts, echantillon_faible: morts < 30 },
      })),
    } as Partial<SquadEchange>)

  it('UNE SEULE SOIRÉE rend UN SEUL BÂTON — il n’y a plus de plancher de tendance', () => {
    const f = friseRiposte(avecSessions([0.3]))
    expect(f.soirees).toHaveLength(1)
    expect(f.soirees[0].tauxPct).toBeCloseTo(30, 6)
  })

  it('réduit le libellé de soirée à sa DATE (un axe ne porte pas la plage horaire)', () => {
    expect(friseRiposte(avecSessions([0.3])).soirees[0].label).toBe('10/09')
  })

  it('porte le VOLUME de chaque soirée — le dénominateur que la courbe ne montrait pas', () => {
    expect(friseRiposte(avecSessions([0.3, 0.1], 84)).soirees.map((s) => s.morts)).toEqual([84, 84])
  })

  it('la couleur porte un VERDICT : au-dessus ou en dessous de l’habituel (20 %)', () => {
    const f = friseRiposte(avecSessions([0.3, 0.1, 0.2]))
    expect(f.soirees.map((s) => s.auDessus)).toEqual([true, false, true])
  })

  it('rend l’habituel en POURCENTS, sur le même axe que les bâtons', () => {
    expect(friseRiposte(avecSessions([0.3])).habituelPct).toBeCloseTo(20, 6)
  })

  it('rend une tendance de MÊME LONGUEUR que les soirées : aucun trou d’amorce', () => {
    const f = friseRiposte(avecSessions([0.3, 0.1, 0.2, 0.4]))
    expect(f.tendancePct).toHaveLength(4)
    expect(f.tendancePct.every((v) => Number.isFinite(v))).toBe(true)
  })

  it('rend une frise VIDE (et non une liste) sans soirée mesurée', () => {
    expect(friseRiposte(echangeDe()).soirees).toEqual([])
  })

  // ─── LE PÉRIMÈTRE EST L'HISTORIQUE, LE FILTRE EST UNE SURBRILLANCE (2026-09-22) ──

  it('PROPAGE la surbrillance servie : aucune soirée n’est écartée par le client', () => {
    const f = friseRiposte(avecSessions([0.3, 0.1, 0.2], 50, [false, true, false]))
    expect(f.soirees).toHaveLength(3)
    expect(f.soirees.map((s) => s.dansLeFiltre)).toEqual([false, true, false])
  })

  it('calcule la TENDANCE sur TOUTES les soirées, pas seulement celles du filtre', () => {
    // Trois soirées à 10 / 20 / 30 %, une seule dans le filtre : la moyenne glissante du
    // dernier point vaut 20 — la moyenne des trois —, et non 30, sa seule soirée filtrée.
    const f = friseRiposte(avecSessions([0.1, 0.2, 0.3], 50, [false, false, true]))
    expect(f.tendancePct[2]).toBeCloseTo(20, 6)
  })
})

describe('moyenneGlissante', () => {
  it('moyenne les `fenetre` dernières valeurs, et s’arrête au début de la série', () => {
    expect(moyenneGlissante([10, 20, 30, 40], FENETRE_TENDANCE)).toEqual([10, 15, 20, 30])
  })

  it('rend une liste vide sur une série vide', () => {
    expect(moyenneGlissante([], FENETRE_TENDANCE)).toEqual([])
  })
})

// ─── LE REPÈRE DE MÉDIANE TOMBE DANS LA BONNE BARRE, À LA BONNE FRACTION ──────
//
// Les intervalles servis n'ont pas tous la même largeur (1 s jusqu'à 5 s, puis 5-7 s, puis
// « 7 s et plus » ouvert). Un repère arrondi à la frontière de barre déplacerait la mesure
// d'une demi-barre : ces cas-ci cadenassent la position FRACTIONNAIRE.

describe('positionMediane', () => {
  it('pose la médiane DANS sa barre, à sa fraction (2,1 s → barre 2, un dixième dedans)', () => {
    // Intervalles du décor : [0-1[ [1-2[ [2-3[ [3-4[ [4-5[ [5-7[ [7-…
    // 2,1 s tombe dans la troisième barre (indice 2), à 10 % de sa largeur.
    expect(positionMediane(echangeDe({ delai_median_ms: 2100 }))).toBeCloseTo(1.6, 6)
  })

  it('pose la médiane sur la FRONTIÈRE quand elle vaut une borne exacte', () => {
    expect(positionMediane(echangeDe({ delai_median_ms: 2000 }))).toBeCloseTo(1.5, 6)
  })

  it('tient compte des intervalles PLUS LARGES (5-7 s : 6 s = la moitié)', () => {
    expect(positionMediane(echangeDe({ delai_median_ms: 6000 }))).toBeCloseTo(5, 6)
  })

  it('pose la médiane au CENTRE d’un intervalle OUVERT — il n’a pas de borne haute', () => {
    expect(positionMediane(echangeDe({ delai_median_ms: 42000 }))).toBe(6)
  })

  it('SE TAIT sans médiane mesurée plutôt que de se poser au hasard', () => {
    expect(positionMediane(echangeDe({ delai_median_ms: 0 }))).toBeNull()
    expect(positionMediane(echangeDe({ delais: [], delai_median_ms: 2100 }))).toBeNull()
  })
})
