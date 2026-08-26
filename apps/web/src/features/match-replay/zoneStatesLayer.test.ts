/**
 * Tests — zoneStatesLayer (l'état VIVANT des zones, schémas 16-18).
 *
 * CE QU'ILS PROTÈGENT : l'état se lit sur l'intervalle qui couvre la frame (bornes incluses),
 * « personne ne la tient » est une MESURE, le calque n'écrit jamais de texte, il refuse de
 * peindre quand la jointure du catalogue est douteuse — et, depuis le schéma 18, LA PROGRESSION
 * SUIT LA SÉRIE DE LA JAUGE EN DIRECT en escalier : jamais le sommet de l'intervalle (le test
 * échoue si l'on y repasse), AUCUNE progression sans `gauge`, la valeur TENUE jusqu'au point
 * suivant (une capture figée reste affichée) et retour à rien une seconde après le dernier point
 * de la série.
 *
 * LA PROGRESSION EST UN REMPLISSAGE DE LA FORME depuis le 2026-08-25 (item D-R), plus un arc
 * extérieur. Les cas qui la visaient ont donc changé d'INSTRUMENT — hauteur de `fillRect`
 * rapportée à l'emprise, au lieu d'angle d'`arc` — mais pas de PROMESSE : ce sont les mêmes
 * frames, les mêmes valeurs attendues et les mêmes encres. Deux cas neufs tiennent ce que la
 * forme apporte et que l'arc ne pouvait pas donner : le découpage PAR la zone, et
 * l'agnosticité boîte / cylindre.
 *
 * Extraits de `objectivesLayer.test.ts` le 2026-08-18 (lot C-ter volet 3) : le calque a son
 * fichier, ses tests aussi.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayMapObjectives } from '@/lib/api/types'

import { normalizeMapObjectives, OBJECTIVE_TEAM_NEUTRAL } from './objectivesLayer'
import { count, recordingContext, valuesOf, type CanvasOp } from './test/recordingContext'
import type { ReplayZoneStateReady } from './replayNormalize'
import { drawZoneStates, zoneElementsOf, zoneGaugeAt } from './zoneStatesLayer'
// L'échelle d'opacités vit avec la peinture (`zoneStatesPaint.ts`), pas avec la lecture d'état :
// c'est de là qu'elle est importée, plutôt que ré-exportée par le calque pour la commodité d'un
// test — une ré-exportation de complaisance brouillerait la frontière que l'extraction pose.
import { ZONE_ALPHA_ORDER } from './zoneStatesPaint'

const MO: ReplayMapObjectives = {
  zones: [
    {
      role: 'strongholds_zone', team: OBJECTIVE_TEAM_NEUTRAL,
      x: 5, y: 5, z: 1, family: 'box', halfX: 2, halfY: 1, fwdX: 0, fwdY: 1,
    },
    {
      role: 'flag_delivery', team: 1,
      x: 8, y: 2, z: 1, family: 'cylinder', radius: 3, fwdX: 1, fwdY: 0,
    },
  ],
  markers: [{ role: 'flag_spawn', team: 0, x: 1, y: 9, z: 1 }],
}

const VIEW = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 480 + 48, height: 480 + 48, pad: 24 }

/** Un document à 100 ms par frame : la tenue d'une seconde vaut 10 frames. */
const HOLD = 10

/**
 * L'état d'une zone tel que l'artefact le publie (schéma 18), déjà normalisé. La zone 0 porte
 * une RAMPE de jauge aux frames 12..18 (le sommet 0,75 de l'intervalle [10 ; 19] est atteint à
 * la frame 18) FERMÉE par son retour à zéro à 19 ; puis une seconde capture qui monte (30..32),
 * se FIGE 28 frames à 0,2 (zone contestée : aucun point), reprend à 60 et est abandonnée — retour
 * à zéro à 62.
 */
const ZONE_STATES: ReplayZoneStateReady[] = [
  {
    zoneRef: 0,
    key: 0x67f43ac3,
    // Rang 0 = la lettre A (fallback d'ordre, cf. `letterRank`). La zone 1, elle, n'en porte
    // pas : c'est la colline du fixture, et une colline n'a pas de lettre.
    letterRank: 0,
    spans: [
      { t0: 0, t1: 9, owner: null, active: false },
      { t0: 10, t1: 19, owner: 0, active: false, progress: 0.75 },
      { t0: 20, t1: 40, owner: 1, active: false },
    ],
    gauge: [
      { t: 12, v: 0 }, { t: 14, v: 0.3 }, { t: 16, v: 0.55 }, { t: 18, v: 0.75 }, { t: 19, v: 0 },
      { t: 30, v: 0 }, { t: 32, v: 0.2 }, { t: 60, v: 0.5 }, { t: 62, v: 0 },
    ],
  },
  { zoneRef: 1, spans: [{ t0: 5, t1: 40, owner: null, active: true, progress: 0.5 }], gauge: [] },
]

describe('zoneGaugeAt — l’escalier de la jauge en direct', () => {
  const gauge = ZONE_STATES[0].gauge

  it('rien AVANT le premier point', () => {
    expect(zoneGaugeAt(gauge, 0, HOLD)).toBeNull()
    expect(zoneGaugeAt(gauge, 11, HOLD)).toBeNull()
  })

  it('la dernière valeur dont l’instant est <= frame — un escalier, pas une pente', () => {
    expect(zoneGaugeAt(gauge, 12, HOLD)).toBe(0)
    expect(zoneGaugeAt(gauge, 13, HOLD)).toBe(0)
    expect(zoneGaugeAt(gauge, 14, HOLD)).toBe(0.3)
    expect(zoneGaugeAt(gauge, 15, HOLD)).toBe(0.3)
    expect(zoneGaugeAt(gauge, 18, HOLD)).toBe(0.75)
  })

  it('le retour à zéro FERME la rampe : la valeur retombe à 0 et y reste', () => {
    expect(zoneGaugeAt(gauge, 19, HOLD)).toBe(0)
    expect(zoneGaugeAt(gauge, 25, HOLD)).toBe(0)
    expect(zoneGaugeAt(gauge, 29, HOLD)).toBe(0)
  })

  it('une capture FIGÉE tient sa valeur jusqu’au point suivant — pas d’expiration entre deux points', () => {
    expect(zoneGaugeAt(gauge, 32, HOLD)).toBe(0.2)
    expect(zoneGaugeAt(gauge, 32 + HOLD + 1, HOLD)).toBe(0.2)
    expect(zoneGaugeAt(gauge, 59, HOLD)).toBe(0.2)
    expect(zoneGaugeAt(gauge, 60, HOLD)).toBe(0.5)
    expect(zoneGaugeAt(gauge, 62, HOLD)).toBe(0)
  })

  it('le DERNIER point de la série tient une seconde, puis plus rien', () => {
    expect(zoneGaugeAt(gauge, 62 + HOLD, HOLD)).toBe(0)
    expect(zoneGaugeAt(gauge, 62 + HOLD + 1, HOLD)).toBeNull()
    // Une série qui s'arrête sur un sommet (fin de film, retour à zéro non lu) : même tenue.
    const sommet = [{ t: 12, v: 0 }, { t: 18, v: 0.75 }]
    expect(zoneGaugeAt(sommet, 18 + HOLD, HOLD)).toBe(0.75)
    expect(zoneGaugeAt(sommet, 18 + HOLD + 1, HOLD)).toBeNull()
  })

  it('une série vide (schéma <= 17, ou zone sans rampe) ne rend jamais de valeur', () => {
    expect(zoneGaugeAt([], 15, HOLD)).toBeNull()
  })
})

describe('zoneElementsOf', () => {
  it('rend les zones SURFACIQUES dans l’ordre servi — celui que zoneRef indexe', () => {
    const zones = zoneElementsOf(normalizeMapObjectives(MO))
    expect(zones).toHaveLength(2)
    expect(zones.every((z) => z.kind === 'zone')).toBe(true)
    expect(zones[0].family).toBe('box')
    expect(zones[1].family).toBe('cylinder')
  })
})

describe('drawZoneStates', () => {
  const style = {
    colorOfOwner: (team: number) => (team === 0 ? '#allié' : '#adverse'),
    colorOfCapturer: (owner: number) => (owner === 0 ? '#adverse' : '#allié'),
    neutral: '#neutre',
  }
  const zones = () => zoneElementsOf(normalizeMapObjectives(MO))
  /** L'entrée du calque telle que `useZoneStates` la rend : jointure ACCORDÉE sauf dit autrement. */
  const layer = (zoneElements = zones(), joinable = true) => ({ zoneElements, joinable, style, gaugeHoldFrames: HOLD })
  /** Combien de remplissages de PROGRESSION ce rendu a émis (un `fillRect` = une capture). */
  const progressions = (ops: CanvasOp[]) => count(ops, 'fillRect')

  /**
   * LA FRACTION LUE SUR LE DERNIER REMPLISSAGE DE PROGRESSION : sa hauteur rapportée à celle de
   * l'emprise de la zone.
   *
   * L'EMPRISE SE DÉDUIT DU RECTANGLE LUI-MÊME, et c'est ce qui rend ce contrôle non
   * tautologique : le rendu émet `fillRect(x, bottom - h, w, h)`, donc `y + h` vaut le BAS de
   * l'emprise quelle que soit la valeur — et le HAUT se retrouve en ajoutant la hauteur totale.
   * On la reconstruit ici par une seconde lecture à valeur pleine, jamais en recopiant la
   * formule de géométrie du calque.
   */
  const fractionDe = (ops: CanvasOp[], hauteurPleine: number) => {
    const rects = ops.filter((o) => o.op === 'fillRect')
    return (rects[rects.length - 1].args[3] as number) / hauteurPleine
  }

  /** La hauteur d'emprise d'une zone : la progression à valeur 1, mesurée une fois. */
  const hauteurPleineDe = (etat: ReplayZoneStateReady, elems = [zones()[0]]) => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer(elems), [{ ...etat, gauge: [{ t: 0, v: 1 }] }], VIEW, 1)
    const rects = ops.filter((o) => o.op === 'fillRect')
    return rects[rects.length - 1].args[3] as number
  }

  /** Les textes écrits par un rendu, dans l'ordre d'émission. */
  const textesDe = (ops: { op: string; args: unknown[] }[]) =>
    ops.filter((o) => o.op === 'fillText' || o.op === 'strokeText').map((o) => String(o.args[0]))

  /**
   * L'encre EN VIGUEUR AU MOMENT DU DERNIER REMPLISSAGE DE PROGRESSION.
   *
   * POURQUOI PAS « LA DERNIÈRE `fillStyle` DU RENDU » : le calque écrit la lettre de la zone
   * après la passe géométrique (2026-08-24), et son remplissage est une encre STRUCTURELLE qui
   * ne dit aucun camp. Viser l'instant du remplissage dit ce que ces cas veulent réellement
   * dire, et ne dépend pas de ce qui est peint après. (Même technique que l'ancien
   * `encreDeLArc`, transposée du trait au remplissage.)
   */
  const encreDeLaProgression = (ops: CanvasOp[]): string | undefined => {
    const dernier = ops.map((o) => o.op).lastIndexOf('fillRect')
    if (dernier < 0) return undefined
    const avant = ops.slice(0, dernier).filter((o) => o.op === 'set fillStyle')
    return avant.length > 0 ? String(avant[avant.length - 1].args[0]) : undefined
  }

  /** L'opacité en vigueur au moment du DERNIER remplissage de progression. */
  const opaciteDeLaProgression = (ops: CanvasOp[]): number | undefined => {
    const dernier = ops.map((o) => o.op).lastIndexOf('fillRect')
    if (dernier < 0) return undefined
    const avant = ops.slice(0, dernier).filter((o) => o.op === 'set globalAlpha')
    return avant.length > 0 ? Number(avant[avant.length - 1].args[0]) : undefined
  }

  /** L'opacité en vigueur au PREMIER `fill` — la teinte d'appartenance. */
  const opaciteDeLaTeinte = (ops: CanvasOp[]): number | undefined => {
    const premier = ops.findIndex((o) => o.op === 'fill')
    if (premier < 0) return undefined
    const avant = ops.slice(0, premier).filter((o) => o.op === 'set globalAlpha')
    return avant.length > 0 ? Number(avant[avant.length - 1].args[0]) : undefined
  }

  /**
   * L'OPACITÉ EN VIGUEUR AU PREMIER TRAIT — celui du contour de la zone.
   *
   * POURQUOI PAS « la plus grande opacité du rendu » : le calque remet `globalAlpha` à 1 avant
   * d'écrire les lettres, donc TOUS les rendus finissent à 1 et la comparaison ne dirait plus
   * rien. Viser l'instant du trait dit ce que ces cas veulent réellement dire — même technique
   * que `encreDeLArc` ci-dessus.
   */
  const opaciteDuTrait = (ops: { op: string; args: unknown[] }[]): number | undefined => {
    const premierTrait = ops.findIndex((o) => o.op === 'stroke')
    if (premierTrait < 0) return undefined
    const avant = ops.slice(0, premierTrait).filter((o) => o.op === 'set globalAlpha')
    return avant.length > 0 ? Number(avant[avant.length - 1].args[0]) : undefined
  }

  /**
   * Les opérations de la passe GÉOMÉTRIQUE : tout ce qui précède la passe des lettres.
   *
   * LA COUPURE EST `set font`, PAS LE PREMIER GLYPHE : la passe des lettres pose ses encres
   * AVANT d'écrire, donc couper au premier `strokeText` laisserait le noir du cerne dans la
   * passe géométrique — ce qu'un premier essai a montré en rougissant.
   */
  const avantTexte = (ops: { op: string; args: unknown[] }[]) => {
    const premier = ops.findIndex((o) => o.op === 'set font')
    return premier < 0 ? ops : ops.slice(0, premier)
  }

  // GARDE AMENDÉE LE 2026-08-24 (décision utilisateur, plan PLAN_LETTRES_BASES_FALLBACK).
  // Ce calque n'écrivait RIEN, parce que la lettre A/B/C du HUD n'existait dans aucune donnée
  // décodée. Elle n'y est toujours pas : ce que l'artefact publie est un FALLBACK D'ORDRE
  // (`letterRank`), mesuré reproductible sur 8 cartes, et le relevé Theater de l'utilisateur
  // reste le juge de « ce sont bien les lettres du jeu ». La garde autorise donc LE glyphe
  // d'une lettre de base — une seule chaîne d'UN caractère, dans A-C — et continue d'interdire
  // tout AUTRE texte : un libellé de zone, un chiffre de jauge, un nom de camp échoueraient ici.
  it("n'écrit AUCUN texte hors le glyphe d'une lettre de base A-C", () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer(), ZONE_STATES, VIEW, 10)
    const textes = textesDe(ops)
    expect(textes.length).toBeGreaterThan(0)
    for (const t of textes) expect(t).toMatch(/^[ABC]$/)
  })

  it('sans `letterRank`, le calque redevient MUET — aucun texte du tout', () => {
    const sansLettre = ZONE_STATES.map((s) => ({ ...s, letterRank: undefined }))
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer(), sansLettre, VIEW, 10)
    expect(count(ops, 'fillText') + count(ops, 'strokeText')).toBe(0)
  })

  // LA LETTRE EST UNE IDENTITÉ, PAS UN ÉTAT : le HUD l'affiche en permanence. Elle se dessine
  // donc même à une frame qu'aucun intervalle ne couvre — là où la teinte, elle, se tait.
  it("la lettre est écrite CERNÉE (strokeText puis fillText) et tient hors des intervalles", () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 80)
    expect(count(ops, 'fill') + count(ops, 'stroke')).toBe(0)
    expect(textesDe(ops)).toEqual(['A', 'A'])
    expect(count(ops, 'strokeText')).toBe(1)
    expect(count(ops, 'fillText')).toBe(1)
  })

  it("la lettre suit le RANG publié : rang 1 rend « B », rang 2 rend « C »", () => {
    for (const [rank, lettre] of [
      [1, 'B'],
      [2, 'C'],
    ] as const) {
      const { ctx, ops } = recordingContext()
      drawZoneStates(ctx, layer([zones()[0]]), [{ ...ZONE_STATES[0], letterRank: rank }], VIEW, 10)
      expect(textesDe(ops)).toEqual([lettre, lettre])
    }
  })

  // Le producteur ne publie jamais au-delà de C, mais le client ne le suppose pas : une donnée
  // hors bornes se tait plutôt que d'écrire « undefined » sur la carte.
  it('un rang hors alphabet ne dessine aucune lettre', () => {
    for (const rank of [3, -1, 1.5]) {
      const { ctx, ops } = recordingContext()
      drawZoneStates(ctx, layer([zones()[0]]), [{ ...ZONE_STATES[0], letterRank: rank }], VIEW, 10)
      expect(textesDe(ops)).toEqual([])
    }
  })

  // La fonction ne LIT pas l'état trouvé : elle sort sur les valeurs PAR DÉFAUT du canvas.
  // C'est ce que ce cas vérifie — pas une restauration, qui n'a jamais eu lieu.
  it("le tracé sort sur les réglages de texte PAR DÉFAUT du canvas (left / alphabetic)", () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 10)
    const aligns = ops.filter((o) => o.op === 'set textAlign').map((o) => o.args[0])
    const baselines = ops.filter((o) => o.op === 'set textBaseline').map((o) => o.args[0])
    expect(aligns[aligns.length - 1]).toBe('left')
    expect(baselines[baselines.length - 1]).toBe('alphabetic')
  })

  // CE CAS N'EST PAS UNE GARDE COLLINE, et le dire importe : le client ne connaît pas la
  // notion de colline. Il vérifie qu'une zone SANS `letterRank` reste muette — la zone active
  // du fixture en est une. L'invariant « une colline ne porte jamais de lettre » est tenu
  // ailleurs, côté Go (`zoneLetterRanks`, porte `hill`), et c'est là qu'il est testé.
  it('une zone sans `letterRank` — dont toute colline, jamais publiée avec — reste muette', () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer(), [ZONE_STATES[1]], VIEW, 30)
    expect(ZONE_STATES[1].letterRank).toBeUndefined()
    expect(textesDe(ops)).toEqual([])
  })

  it('une zone TENUE est remplie ET cerclée à l’encre de son camp', () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 10)
    expect(count(ops, 'fill')).toBe(1)
    expect(count(ops, 'stroke')).toBeGreaterThanOrEqual(1)
  })

  it('une zone que PERSONNE ne tient garde le liseré seul — aucun remplissage', () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 3)
    expect(count(ops, 'fill')).toBe(0)
    expect(count(ops, 'stroke')).toBe(1)
  })

  // DEMANDE UTILISATEUR DU 2026-08-25 : « bases non prises : contour grisé ». L'encre était
  // déjà le neutre ; c'est le TRAIT qui s'affirmait autant que celui d'une base gagnée. Ces
  // cas tiennent le retrait, et surtout ce qui le DÉCLENCHE.
  it('zone NON PRISE : contour à l’encre neutre, plus fin et plus transparent qu’une zone tenue', () => {
    const libre = recordingContext()
    drawZoneStates(libre.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 3)
    const tenue = recordingContext()
    drawZoneStates(tenue.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 14)
    const traitLibre = valuesOf(avantTexte(libre.ops), 'lineWidth') as unknown as number[]
    const traitTenu = valuesOf(avantTexte(tenue.ops), 'lineWidth') as unknown as number[]
    expect(traitLibre[0]).toBeLessThan(traitTenu[0])
    expect(opaciteDuTrait(libre.ops)!).toBeLessThan(opaciteDuTrait(tenue.ops)!)
    // L'encre reste le neutre du thème — jamais une couleur de camp devinée.
    const encres = valuesOf(avantTexte(libre.ops), 'strokeStyle') as unknown as string[]
    expect(encres.every((i) => i === '#neutre')).toBe(true)
  })

  // LE SEUIL EST `owner === null`, PAS `held`. Une zone TENUE par un camp que la page ne sait
  // pas situer (aucune ligne « moi ») n'est PAS libre : la griser dirait « à prendre » d'une
  // base qui est gagnée. Elle garde le trait plein, à l'encre neutre.
  it('camp inconnu ≠ zone libre : le trait reste PLEIN, seulement neutre', () => {
    const aveugle = { colorOfOwner: () => null, colorOfCapturer: () => null, neutral: '#neutre' }
    const inconnu = recordingContext()
    drawZoneStates(
      inconnu.ctx,
      { ...layer([zones()[0]]), style: aveugle },
      [{ ...ZONE_STATES[0], gauge: [] }],
      VIEW,
      14,
    )
    const libre = recordingContext()
    drawZoneStates(libre.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 3)
    const traitInconnu = valuesOf(avantTexte(inconnu.ops), 'lineWidth') as unknown as number[]
    const traitLibre = valuesOf(avantTexte(libre.ops), 'lineWidth') as unknown as number[]
    expect(traitInconnu[0]).toBeGreaterThan(traitLibre[0])
  })

  // La colline ACTIVE n'a pas de propriétaire non plus, et elle ne doit surtout PAS être
  // grisée : la surbrillance dit « c'est ici que ça se joue ». Le retrait ne l'atteint pas.
  it('la colline ACTIVE sans propriétaire garde sa surbrillance, jamais le retrait', () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer(), [ZONE_STATES[1]], VIEW, 30)
    expect(count(ops, 'fill')).toBe(1)
    expect(valuesOf(ops, 'lineWidth')).toContain(3.5)
  })

  // La zone ACTIVE est le SEUL cas où le calque remplit sans propriétaire : la surbrillance dit
  // « c'est ici que ça se joue » là où le liseré seul dirait « personne ne la tient ». Ce cas
  // porte la lecture d'`active` depuis la revue du 2026-08-19 : elle était vérifiée sur la
  // fonction de lecture exportée, que plus rien n'appelait — c'est au RENDU qu'elle se voit.
  it('la zone ACTIVE est en SURBRILLANCE : remplie sans propriétaire, liseré renforcé', () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer(), [ZONE_STATES[1]], VIEW, 30)
    expect(count(ops, 'fill')).toBe(1)
    expect(valuesOf(ops, 'lineWidth')).toContain(3.5)
  })

  it('une zone sans état à cette frame n’est PAS repeinte : elle reste au trait faible', () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer(), ZONE_STATES, VIEW, 80)
    expect(count(ops, 'fill') + count(ops, 'stroke')).toBe(0)
  })

  // LE VERROU DU SCHÉMA 17 : la progression se remplit avec la VALEUR de la jauge à l'image. À
  // la frame 14 la série dit 0,3 alors que le sommet de l'intervalle dit 0,75 — repasser au
  // sommet fait échouer ce cas, exactement comme dessiner 0,55 à la frame 15 (l'escalier tient
  // 0,3).
  it('la progression SUIT la série de la jauge, en escalier — jamais le sommet de l’intervalle', () => {
    const H = hauteurPleineDe(ZONE_STATES[0])
    const a14 = recordingContext()
    drawZoneStates(a14.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 14)
    expect(progressions(a14.ops)).toBe(1)
    expect(fractionDe(a14.ops, H)).toBeCloseTo(0.3, 6)
    const a15 = recordingContext()
    drawZoneStates(a15.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 15)
    expect(fractionDe(a15.ops, H)).toBeCloseTo(0.3, 6)
    const a18 = recordingContext()
    drawZoneStates(a18.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 18)
    expect(fractionDe(a18.ops, H)).toBeCloseTo(0.75, 6)
  })

  // LE REMPLISSAGE PART DU BAS : à valeur croissante, le rectangle grandit ET son bord haut
  // MONTE, le bas restant fixe. Sans ce cas, un remplissage par le haut passerait les fractions.
  it('la progression monte du BAS vers le haut : le bord bas ne bouge pas', () => {
    const bas = (ops: CanvasOp[]) => {
      const r = ops.filter((o) => o.op === 'fillRect')
      const a = r[r.length - 1].args
      return (a[1] as number) + (a[3] as number)
    }
    const haut = (ops: CanvasOp[]) => {
      const r = ops.filter((o) => o.op === 'fillRect')
      return r[r.length - 1].args[1] as number
    }
    const petit = recordingContext()
    drawZoneStates(petit.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 14)
    const grand = recordingContext()
    drawZoneStates(grand.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 18)
    expect(bas(grand.ops)).toBeCloseTo(bas(petit.ops), 6)
    // L'axe Y du canvas descend : « plus haut » = ordonnée plus PETITE.
    expect(haut(grand.ops)).toBeLessThan(haut(petit.ops))
  })

  // CE QUE LA FORME APPORTE, ET QUE L'ARC EXTÉRIEUR NE POUVAIT PAS DONNER : le remplissage est
  // DÉCOUPÉ PAR LA ZONE. Sans `clip`, le rectangle déborderait sur la carte.
  it('la progression est DÉCOUPÉE par la forme de la zone (clip), et rendue sous save/restore', () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 14)
    const noms = ops.map((o) => o.op)
    const clip = noms.indexOf('clip')
    expect(clip).toBeGreaterThan(-1)
    // Le découpage suit un tracé de forme, et précède le remplissage.
    expect(noms.lastIndexOf('beginPath', clip)).toBeGreaterThan(-1)
    expect(noms.indexOf('fillRect')).toBeGreaterThan(clip)
    // Le découpage est REFERMÉ : sans restore, tout ce qui suit resterait clippé à la zone.
    expect(noms.lastIndexOf('save')).toBeLessThan(clip)
    expect(noms.indexOf('restore')).toBeGreaterThan(noms.indexOf('fillRect'))
  })

  // AGNOSTICITÉ DE FORME : c'est l'argument qui a fait abandonner l'arc (une fraction d'ANGLE
  // n'est pas une fraction d'AIRE sur une boîte orientée). La MÊME valeur doit rendre la MÊME
  // fraction d'emprise sur une boîte et sur un cylindre.
  it('la fraction est la même sur une BOÎTE et sur un CYLINDRE', () => {
    const boite = zones()[0]
    const cylindre = zones()[1]
    expect(boite.family).toBe('box')
    expect(cylindre.family).toBe('cylinder')
    for (const [elem, etat] of [
      [boite, ZONE_STATES[0]],
      [cylindre, ZONE_STATES[0]],
    ] as const) {
      const H = hauteurPleineDe(etat, [elem])
      const { ctx, ops } = recordingContext()
      drawZoneStates(ctx, layer([elem]), [etat], VIEW, 18)
      expect(fractionDe(ops, H)).toBeCloseTo(0.75, 6)
    }
  })

  it('aucune progression AVANT la rampe ni APRÈS son retour à zéro ; elle TIENT pendant un blocage', () => {
    const H = hauteurPleineDe(ZONE_STATES[0])
    const avant = recordingContext()
    drawZoneStates(avant.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 11)
    expect(progressions(avant.ops)).toBe(0)
    // Frame 25 : la jauge est revenue à zéro (point de la frame 19) — rien à remplir.
    const retombe = recordingContext()
    drawZoneStates(retombe.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 25)
    expect(progressions(retombe.ops)).toBe(0)
    // Frame 45 : la capture est FIGÉE à 0,2 depuis la frame 32 — le remplissage reste, à 0,2.
    const fige = recordingContext()
    drawZoneStates(fige.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 45)
    expect(progressions(fige.ops)).toBe(1)
    expect(fractionDe(fige.ops, H)).toBeCloseTo(0.2, 6)
    // Une seconde après le DERNIER point de la série (62, retour à zéro), plus rien.
    const eteint = recordingContext()
    drawZoneStates(eteint.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 62 + HOLD + 1)
    expect(progressions(eteint.ops)).toBe(0)
  })

  // LA DÉCISION DU PLAN, INCHANGÉE PAR LE CHANGEMENT DE FORME : sur un artefact qui ne porte pas
  // `gauge` (schéma <= 17), il n'y a AUCUNE PROGRESSION — même quand l'intervalle publie un
  // sommet. Le sommet statique se lisait comme une jauge ; mieux vaut rien.
  it('sans `gauge`, AUCUNE progression — le sommet `progress` de l’intervalle ne la remplace pas', () => {
    const sansJauge = [{ ...ZONE_STATES[0], gauge: [] }]
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer([zones()[0]]), sansJauge, VIEW, 14)
    expect(progressions(ops)).toBe(0)
    // La colline active de la zone 1 publie un sommet (0,5) et aucune série : rien non plus.
    // C'est aussi le cas de TOUTE colline en production — le producteur ne publie jamais de
    // série de jauge en KOTH (`zone_states_gauge.go`, « EN KOTH, RIEN »).
    const colline = recordingContext()
    drawZoneStates(colline.ctx, layer([zones()[0]]), [{ ...ZONE_STATES[1], zoneRef: 0 }], VIEW, 30)
    expect(progressions(colline.ops)).toBe(0)
  })

  it('la progression prend l’encre du camp QUI CAPTURE (le camp d’en face du propriétaire)', () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 14)
    // Le propriétaire à la frame 14 est le camp 0 (allié) : la progression est ADVERSE.
    expect(encreDeLaProgression(ops)).toBe('#adverse')
  })

  // LA HIÉRARCHIE DEMANDÉE : progression FRANCHE de l'attaquant, teinte du propriétaire
  // AFFAIBLIE. Ce cas oppose la MÊME zone tenue avec et sans capture en cours.
  it('pendant une capture, la teinte du propriétaire S’EFFACE sous une progression plus franche', () => {
    const sous = recordingContext()
    drawZoneStates(sous.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 14)
    const tranquille = recordingContext()
    drawZoneStates(tranquille.ctx, layer([zones()[0]]), [{ ...ZONE_STATES[0], gauge: [] }], VIEW, 14)
    // La teinte d'appartenance recule pendant la capture...
    expect(opaciteDeLaTeinte(sous.ops)!).toBeLessThan(opaciteDeLaTeinte(tranquille.ops)!)
    // ...et la progression passe au-dessus d'elle, plus franche que toute appartenance.
    expect(opaciteDeLaProgression(sous.ops)!).toBeGreaterThan(opaciteDeLaTeinte(tranquille.ops)!)
  })

  // L'ÉCHELLE COMPLÈTE. Ces opacités ne valent que les unes par rapport aux autres : c'est leur
  // ORDRE qui porte la lecture, et le gate visuel peut toutes les bouger.
  it('l’échelle d’appartenance est STRICTEMENT croissante : libre < en perte < tenue < active < progression', () => {
    for (let i = 1; i < ZONE_ALPHA_ORDER.length; i++) {
      expect(ZONE_ALPHA_ORDER[i]).toBeGreaterThan(ZONE_ALPHA_ORDER[i - 1])
    }
  })

  it('propriétaire inconnu (zone neutre) : la progression est NEUTRE, jamais une couleur devinée', () => {
    const neutre = [{ ...ZONE_STATES[0], gauge: [{ t: 2, v: 0.4 }] }]
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer([zones()[0]]), neutre, VIEW, 3)
    expect(progressions(ops)).toBe(1)
    expect(encreDeLaProgression(ops)).toBe('#neutre')
  })

  it('une rampe AVANT le premier intervalle se dessine quand même, à l’encre neutre et sans contour', () => {
    const tot = [{ ...ZONE_STATES[0], spans: ZONE_STATES[0].spans.slice(1), gauge: [{ t: 2, v: 0.4 }] }]
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer([zones()[0]]), tot, VIEW, 3)
    expect(progressions(ops)).toBe(1)
    // Aucune appartenance à affirmer : ni teinte, ni liseré.
    expect(count(ops, 'fill')).toBe(0)
    expect(count(ops, 'stroke')).toBe(0)
    expect(encreDeLaProgression(ops)).toBe('#neutre')
  })

  it('camp inconnu (aucune ligne « moi ») : encre NEUTRE, jamais une couleur devinée', () => {
    const { ctx, ops } = recordingContext()
    const aveugle = { colorOfOwner: () => null, colorOfCapturer: () => null, neutral: '#neutre' }
    drawZoneStates(ctx, { ...layer([zones()[0]]), style: aveugle }, [ZONE_STATES[0]], VIEW, 14)
    // Aucune TEINTE d'appartenance : une zone TENUE par un camp qu'on ne sait pas situer garde
    // le liseré seul — un unique trait, à l'encre neutre. La progression de capture, elle, est
    // bien peinte (c'est une mesure de la zone, pas une appartenance) et elle est neutre aussi.
    expect(count(ops, 'fill')).toBe(0)
    expect(count(ops, 'stroke')).toBe(1)
    expect(encreDeLaProgression(ops)).toBe('#neutre')
    // La passe GÉOMÉTRIQUE n'emploie que l'encre neutre. Le cerne de la lettre, posé après, est
    // une encre structurelle hors thème et ne dit aucun camp : il n'entre pas dans ce compte.
    const encres = valuesOf(avantTexte(ops), 'strokeStyle') as unknown as string[]
    expect(encres.length).toBeGreaterThan(0)
    expect(encres.every((i) => i === '#neutre')).toBe(true)
  })

  it('sans état publié, le calque ne dessine rien du tout', () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer(), [], VIEW, 10)
    expect(ops).toHaveLength(0)
  })

  // VERROU DE LA REVUE R1-7 : `zoneRef` est un index figé à la cuisson, la liste servie est
  // reconstruite à la requête. Quand le catalogue de l'artefact ne joint pas la liste servie,
  // le calque VIVANT ne touche PAS au contexte — pas un trait, pas même un `beginPath`.
  it('jointure REFUSÉE (catalogue différent de la liste servie) : le calque ne peint rien', () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer([zones()[0]], false), [ZONE_STATES[0]], VIEW, 14)
    expect(ops).toHaveLength(0)
  })
})
