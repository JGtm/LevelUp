/**
 * Tests — weaponPadsLayer : LA PILE, LES ÉTATS DESSINÉS, LA TAILLE, LE SURVOL.
 *
 * CE QUE CE FICHIER VERROUILLE, et chaque point correspond à une clause écrite du plan :
 *  - le LOSANGE est TOUJOURS PLEIN à l'encre de sa nature (retour utilisateur du 2026-08-27) :
 *    c'est son OPACITÉ qui dit l'état, et le seul état incertain gagne un HALO POINTILLÉ ;
 *  - l'incertitude ne se MASQUE pas : la vignette reste, en fantôme ;
 *  - LA PILE, de bas en haut : losange au lieu mesuré, vignette AU-DESSUS, compteur au sommet.
 *    L'anneau-bordure qui enfermait la vignette n'existe plus — c'est lui que l'utilisateur
 *    lisait comme « l'icône dans le losange » ;
 *  - la TAILLE suit l'arme : puissance grande, classique petite ;
 *  - le SURVOL couvre la PILE ENTIÈRE (vignette comprise) et se rejoue sur la donnée, y compris
 *    sur un socle VIDE (c'est là qu'on inspecte).
 *
 * LA LECTURE TEMPORELLE (états, occupations, compte à rebours) EST TESTÉE À PART, dans
 * `weaponPadTime.test.ts` : elle a quitté ce calque le 2026-08-27 avec son module.
 *
 * Le contexte enregistreur observe la GÉOMÉTRIE ÉMISE, jamais un pixel (cf. recordingContext).
 */
import { describe, expect, it } from 'vitest'

import { count, diamondCentres, recordingContext, valuesOf } from '../test/recordingContext'
import type { ReplayWeaponPadReady } from '../../../lib/replay/replayNormalize'
import { worldToCanvas } from '../../../lib/replay/replayLogic'
import { padStateAt } from '../model/weaponPadTime'
import { drawWeaponPadsLayer, padAt, padRadiusPx, type PadStyle } from './weaponPadsLayer'

/** 10 m de côté sur 100 px : 10 px par mètre — le même cadrage que les tests de poses. */
const VIEW = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 100, height: 100, pad: 0 }

/** Une image de 100 ms : 10 images = 1 s, ce qui rend les comptes lisibles à l'œil nu. */
const FRAME_MS = 100

/** Le S7 Sniper est une arme de PUISSANCE ; le BR75 une classique (weaponPadFamilies). */
const SNIPER = '0x0A1992BC'
const BR75 = '0x2B1824D5'
const KEYS: Record<string, string> = { [SNIPER]: 'hinf_s7_sniper', [BR75]: 'hinf_br75' }

const IMAGE = { width: 40, height: 16 } as unknown as CanvasImageSource
/** Une vignette prête à poser : son corps et son liseré (cf. PadIcon). */
const ICON = { fill: IMAGE, outline: IMAGE }

/** Le cycle des témoins : 40 s pile, pour que les comptes se lisent à l'œil nu. */
const CYCLE = { medianS: 40, p10S: 40, p90S: 40, gaps: 2, missing: 0 }

function style(over: Partial<PadStyle> = {}): PadStyle {
  return {
    ink: 'encre',
    fill: 'remplissage',
    outline: 'contour',
    iconOf: () => ICON,
    scaleOf: (weapon) => (KEYS[weapon] === 'hinf_s7_sniper' ? 'power' : 'classic'),
    // A13 : l'encre de la NATURE du socle, résolue par l'appelant comme toutes les autres.
    inkOf: () => 'nature',
    countdownLabel: (s) => `${Math.ceil(s)} s`,
    ...over,
  }
}

function pad(over: Partial<ReplayWeaponPadReady> = {}): ReplayWeaponPadReady {
  return {
    x: 5,
    y: 5,
    weapon: SNIPER,
    spawns: [0],
    presence: [{ t0: 0, tLow: 100, tHigh: 120 }],
    ...over,
  }
}

function draw(pads: ReplayWeaponPadReady[], frame: number, over: Partial<PadStyle> = {}) {
  const { ops, ctx } = recordingContext()
  drawWeaponPadsLayer(ctx, pads, VIEW, { frame, frameMs: FRAME_MS, k: 1 }, style(over))
  return ops
}

/** La demi-diagonale d'un losange se lit sur son sommet : centre - sommet (cf. traceDiamond). */
function demiDiagonales(ops: ReturnType<typeof draw>): number[] {
  const centres = diamondCentres(ops)
  const sommets = ops.filter((o) => o.op === 'moveTo').map((o) => (o.args as number[])[1])
  return centres.map((c, i) => c.y - sommets[i])
}

/**
 * LE TRACÉ, EN PILE (retour utilisateur du 2026-08-27 : « l'icône doit être au-dessus du petit
 * losange, pas dedans ») : le LOSANGE au lieu mesuré, la VIGNETTE au-dessus de lui, le COMPTE À
 * REBOURS au sommet. Le losange est TOUJOURS PLEIN à l'encre de sa nature ; l'anneau-bordure
 * d'A13, qui enfermait la vignette, a été supprimé avec ses constantes.
 */
describe('le tracé — losange plein, vignette au-dessus, compte à rebours au sommet', () => {
  it('PLEIN : UN SEUL losange, REMPLI, et la vignette à pleine encre', () => {
    const ops = draw([pad()], 50)
    // UNE marque, et une seule : plus d'anneau-bordure autour de la vignette.
    expect(diamondCentres(ops)).toHaveLength(1)
    expect(count(ops, 'arc')).toBe(0)
    expect(count(ops, 'drawImage')).toBeGreaterThan(0)
    // Le plein se suffit : ni contour, ni pointillé, aucun trait tracé.
    expect(count(ops, 'setLineDash')).toBe(0)
    expect(count(ops, 'stroke')).toBe(0)
    expect(count(ops, 'fill')).toBe(1)
    // LA REMISE À 1 FINALE EST EXCLUE (revue adversariale du 2026-08-27) : `drawWeaponPadsLayer`
    // termine par un `globalAlpha = 1` inconditionnel, si bien que le maximum brut valait
    // TOUJOURS 1 et que ce cas ne mordait sur rien — il serait resté vert avec une marque
    // quasi transparente. Même patron que le cas INCERTAIN ci-dessous.
    expect(Math.max(...valuesOf(ops, 'globalAlpha').slice(0, -1))).toBeGreaterThan(0.9)
  })

  it('INCERTAIN : le losange reste PLEIN (atténué) et gagne un HALO POINTILLÉ concentrique', () => {
    const ops = draw([pad()], 110)
    expect(count(ops, 'drawImage'), "l'incertitude ne se masque pas").toBeGreaterThan(0)
    // LE PLEIN N'A PAS DISPARU : c'est l'opacité qui dit l'incertitude, plus l'absence d'aplat.
    expect(count(ops, 'fill')).toBe(1)
    const dashes = ops.filter((o) => o.op === 'setLineDash').map((o) => o.args[0] as number[])
    expect(dashes[0]?.length).toBeGreaterThan(0)
    expect(count(ops, 'stroke')).toBe(1)
    // Deux losanges CONCENTRIQUES — la marque et son halo — et le halo est le plus grand.
    const centres = diamondCentres(ops)
    expect(centres).toHaveLength(2)
    expect(centres[0]).toEqual(centres[1])
    const [marque, halo] = demiDiagonales(ops)
    expect(halo).toBeGreaterThan(marque)
    // Aucune opacité pleine : ni la marque ni la vignette n'affirment une présence.
    expect(Math.max(...valuesOf(ops, 'globalAlpha').slice(0, -1))).toBeLessThan(0.6)
  })

  it('VIDE : plus de vignette, mais le losange PLEIN reste — le lieu ne disparaît pas', () => {
    const ops = draw([pad()], 200)
    expect(count(ops, 'drawImage')).toBe(0)
    expect(diamondCentres(ops)).toHaveLength(1)
    expect(count(ops, 'fill')).toBe(1)
    // Atténué, jamais éteint : c'est la plus basse des trois opacités, et elle reste visible.
    const alpha = valuesOf(ops, 'globalAlpha')[0]
    expect(alpha).toBeGreaterThan(0.3)
    expect(alpha).toBeLessThan(0.55)
  })

  it('VIDE avec cycle : le compte à rebours s’écrit, cerné pour rester lisible', () => {
    const ops = draw([pad({ cycle: CYCLE })], 320)
    expect(ops.filter((o) => o.op === 'fillText').map((o) => o.args[0])).toEqual(['20 s'])
    expect(count(ops, 'strokeText')).toBe(1)
  })

  // LE GAIN VISIBLE DE D3 (2026-08-27) : un trou MÉDIAN — refermé par une apparition que le film
  // montre — porte désormais son compte à rebours, cycle ou pas. C'était le défaut signalé
  // (« compteur pas toujours visible ») : 24 socles sur 57 n'ont aucun cycle établi et
  // n'affichaient donc jamais rien, alors même que le rejeu connaissait la suite.
  it('VIDE sans cycle mais avec une apparition SUIVANTE : le compte s’écrit quand même', () => {
    const deux = pad({
      spawns: [0, 300],
      presence: [
        { t0: 0, tLow: 100, tHigh: 120 },
        { t0: 300, tLow: 400, tHigh: 420 },
      ],
    })
    expect(deux.cycle).toBeUndefined()
    const ops = draw([deux], 200)
    expect(ops.filter((o) => o.op === 'fillText').map((o) => o.args[0])).toEqual(['10 s'])
  })

  it('VIDE sans cycle NI apparition suivante : AUCUN texte — ni chiffre, ni tiret', () => {
    const ops = draw([pad()], 320)
    expect(count(ops, 'fillText')).toBe(0)
    expect(count(ops, 'strokeText')).toBe(0)
  })

  it('sans contour de thème, le compte à rebours s’écrit quand même (sans cerne)', () => {
    const ops = draw([pad({ cycle: CYCLE })], 320, { outline: '' })
    expect(count(ops, 'fillText')).toBe(1)
    expect(count(ops, 'strokeText')).toBe(0)
  })

  // AMENDÉ LE 2026-08-26 (retour utilisateur : « j'ai l'impression qu'il y en a deux »). Ce cas
  // verrouillait le GLYPHE DE REPLI : sans vignette, le calque posait un disque plein du même
  // rayon que le point, une dizaine de pixels sous lui. Deux ronds empilés pour UN socle — et
  // c'est exactement ce que l'œil comptait deux fois. Le repli est supprimé : le losange porte
  // déjà la position ET l'état. Ce qui reste garanti, et qui était le vrai sujet du cas :
  // aucune image n'est empruntée à une arme voisine.
  it('SANS VIGNETTE : le losange SEUL, et surtout jamais l’icône d’une arme voisine', () => {
    const ops = draw([pad()], 50, { iconOf: () => null })
    expect(count(ops, 'drawImage')).toBe(0)
    expect(diamondCentres(ops)).toHaveLength(1)
    expect(count(ops, 'fill')).toBe(1)
  })

  /**
   * LE LISERÉ : la même forme reposée tout autour, à l'encre du FOND. C'est lui qui détache la
   * vignette d'un fond de carte clair comme d'un fond sombre — le « contour noir » demandé.
   *
   * IL EST DÛ À TOUTES LES IMAGES depuis le 2026-08-27 (retour utilisateur : « icône AVEC
   * contour »). Une image FINIE du jeu se posait telle quelle, au motif qu'on ne peut pas la
   * reteindre : vrai pour son CORPS, faux pour son contour — cerner ne demande que la
   * SILHOUETTE, que `tintedIconCanvas` rend de n'importe quelle image à alpha. Le calque n'a
   * donc plus de branche « sans liseré », et le TYPE l'interdit désormais (`PadIcon.outline`
   * n'est plus nullable) : c'est le compilateur, et non ce cas, qui tient l'invariant.
   */
  it('la vignette est CERNÉE : son liseré est reposé huit fois autour du corps', () => {
    const ops = draw([pad()], 50)
    expect(count(ops, 'drawImage')).toBe(9)
  })

  it('la TAILLE suit l’arme : une arme de puissance est plus grande qu’une classique', () => {
    const puissance = draw([pad()], 50)
    const classique = draw([pad({ weapon: BR75 })], 50)
    expect(demiDiagonales(puissance)[0]).toBeGreaterThan(demiDiagonales(classique)[0])
    const hauteur = (ops: ReturnType<typeof draw>) =>
      (ops.find((o) => o.op === 'drawImage')?.args[4] as number) ?? 0
    expect(hauteur(puissance)).toBeGreaterThan(hauteur(classique))
  })

  /**
   * L'ORDRE DE LA PILE EST LA RÈGLE (2026-08-27) : le losange à la position monde du socle, la
   * vignette ENTIÈREMENT au-dessus de lui — son bas strictement plus haut que le sommet du
   * losange — et le compteur au-dessus de tout.
   */
  it('LA PILE : losange au socle, vignette entièrement AU-DESSUS de lui', () => {
    const ops = draw([pad({ x: 2, y: 8 })], 50)
    const c = worldToCanvas({ x: 2, y: 8 }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
    const marque = diamondCentres(ops)[0]
    expect(marque.x).toBeCloseTo(c.x, 6)
    expect(marque.y).toBeCloseTo(c.y, 6)
    const sommet = c.y - demiDiagonales(ops)[0]
    // Le CORPS est le dernier posé (le liseré vient d'abord, tout autour) ; son BAS est au-dessus
    // du sommet du losange, écart compris — la vignette ne le chevauche jamais.
    const images = ops.filter((o) => o.op === 'drawImage')
    const [, dx, dy, w, h] = images[images.length - 1].args as number[]
    expect(dx + w / 2).toBeCloseTo(c.x, 6)
    expect(dy + h).toBeLessThan(sommet)
    expect(dy + h / 2).toBeLessThan(c.y)
  })

  /**
   * LE COMPTEUR S'ANCRE SUR LE LOSANGE, ET C'EST UNE CONSÉQUENCE (revue adversariale du
   * 2026-08-27) : un compte à rebours n'existe que sur un socle VIDE, et un socle vide n'a PAS
   * de vignette. Le sommet de la pile EST donc toujours celui du losange quand un chiffre
   * s'écrit — le calque portait un `icon ? … : …` dont la branche « sous la vignette » était
   * inatteignable. Ce cas vérifie les deux moitiés : rien n'est dessiné au-dessus du losange, et
   * le chiffre est bien plus haut que lui.
   */
  it('le COMPTEUR s’ancre au sommet du LOSANGE, seule chose dessinée sous lui', () => {
    const ops = draw([pad({ x: 2, y: 8, cycle: CYCLE })], 320)
    const c = worldToCanvas({ x: 2, y: 8 }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
    expect(count(ops, 'drawImage'), 'un socle vide ne porte aucune vignette').toBe(0)
    const sommet = c.y - demiDiagonales(ops)[0]
    const [, , ty] = ops.find((o) => o.op === 'fillText')!.args as number[]
    expect(ty).toBeLessThan(sommet)
  })

  it('aucune COULEUR n’est écrite ici : les encres viennent toutes de l’appelant', () => {
    const ops = draw([pad({ cycle: CYCLE })], 320)
    const encres = ops
      .filter((o) => o.op === 'set fillStyle' || o.op === 'set strokeStyle')
      .map((o) => o.args[0])
    expect(encres.length).toBeGreaterThan(0)
    // `nature` rejoint la liste depuis A13 : c'est l'encre de la FAMILLE du socle, et elle
    // vient de l'appelant exactement comme les trois autres — ce cas garde son sujet entier.
    for (const e of encres) expect(['encre', 'remplissage', 'contour', 'nature']).toContain(e)
  })

  it('aucun socle : rien n’est dessiné, pas même un cadre vide (témoin 000d5950)', () => {
    expect(draw([], 50)).toHaveLength(0)
  })
})

describe('padAt — le survol se rejoue sur la donnée, sur toute la pile', () => {
  const s = style()
  const at = (x: number, y: number) =>
    worldToCanvas({ x, y }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)

  it('trouve le socle sous le pointeur', () => {
    const p = pad()
    expect(padAt([p], VIEW, s, 1, at(5, 5))).toBe(p)
  })

  it('ne trouve rien loin du socle', () => {
    expect(padAt([pad()], VIEW, s, 1, at(1, 1))).toBeNull()
  })

  /**
   * LA VIGNETTE EST SURVOLABLE (2026-08-27), et c'est la contrepartie de la pile : viser l'image
   * — la seule chose lisible d'un socle à 10 px — doit ouvrir l'infobulle. La zone couvrait le
   * seul losange, dix pixels plus bas : le pointeur sur la vignette ne trouvait rien.
   */
  it('le survol attrape la VIGNETTE, posée au-dessus du losange', () => {
    const p = pad({ x: 5, y: 5 })
    const ops = draw([p], 50)
    const images = ops.filter((o) => o.op === 'drawImage')
    const [, , dy, , h] = images[images.length - 1].args as number[]
    const centreVignette = { x: 50, y: dy + h / 2 }
    expect(centreVignette.y, 'la vignette doit être au-dessus du losange').toBeLessThan(50 - 5)
    expect(padAt([p], VIEW, s, 1, centreVignette)).toBe(p)
    // Et jusqu'à son bord haut : c'est toute la vignette qui se vise, pas son seul centre.
    expect(padAt([p], VIEW, s, 1, { x: 50, y: dy })).toBe(p)
  })

  it('un socle VIDE se survole aussi : c’est là qu’on veut savoir ce qui manque', () => {
    const p = pad()
    expect(padStateAt(p, 300)).toBe('empty')
    expect(padAt([p], VIEW, s, 1, at(5, 5))).toBe(p)
  })

  /**
   * L'ARBITRAGE SE FAIT SUR LE LIEU, pas sur la pile : deux socles de tailles différentes ont
   * des piles de hauteurs différentes, et départager sur elles ferait gagner le plus petit
   * voisin alors que le pointeur est pile sur le losange de l'autre.
   */
  it('le PLUS PROCHE l’emporte quand deux socles se recouvrent', () => {
    const a = pad({ x: 5, y: 5 })
    const b = pad({ x: 5.4, y: 5, weapon: BR75 })
    expect(padAt([a, b], VIEW, s, 1, at(5.4, 5))).toBe(b)
    expect(padAt([a, b], VIEW, s, 1, at(5, 5))).toBe(a)
  })

  /**
   * LA PORTÉE EST CELLE DE LA PILE, ET RIEN D'AUTRE (revue adversariale du 2026-08-27).
   *
   * Ce cas annonçait un « plancher visable » de 9 px et passait pour une TOUTE AUTRE raison : la
   * zone d'une arme classique vaut 12,25 px (4 + 5,25 + 3), le plancher ne pouvait plus jamais
   * l'emporter, et il a été supprimé avec son `Math.max`. Ce qui restait vrai — la pile donne
   * d'elle-même une cible trois fois plus large que son losange — est désormais vérifié SUR LA
   * FORMULE RÉELLE, aux deux pixels qui l'encadrent.
   */
  it('la portée du survol est celle de la PILE : 12,25 px pour une arme classique', () => {
    const classique = pad({ weapon: BR75 })
    // Le losange seul ne se viserait pas : 3,2 px de rayon, 4 px de demi-diagonale après
    // compensation d'aire.
    expect(padRadiusPx(classique, s, 1)).toBeCloseTo(3.2, 6)
    // Le centre de la zone est relevé de (8 + 2,5) / 2 = 5,25 px au-dessus du socle.
    const cy = 50 - 5.25
    expect(padAt([classique], VIEW, s, 1, { x: 50 + 12, y: cy })).not.toBeNull()
    expect(padAt([classique], VIEW, s, 1, { x: 50 + 13, y: cy })).toBeNull()
  })
})
