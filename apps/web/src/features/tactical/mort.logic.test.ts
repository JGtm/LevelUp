/**
 * mort.logic — la logique pure de la lecture des coéquipiers : barre des statuts, rose, phrases, fiche
 * d'une mort, scènes du plan (nuage, instantané, séquence).
 */
import { describe, expect, it } from 'vitest'

import { getTacticalText } from './i18n'
import {
  batonsDeLaRose,
  directionDominante,
  distancesDEtape,
  libelleDeDecalage,
  lignesDeLaFiche,
  partsDeBarre,
  phraseDeLaRose,
  resumeDeLEquipe,
  sceneDeMort,
  sceneDeZone,
} from './mort.logic'
import { BILAN, FRAG, MORT_COMPLETE, SEQUENCE } from './mort.fixtures'
import { sceneDeSequence } from './sequenceLecture.logic'
import { sceneDeLaVue, sequenceDemandee } from './vueDeMort'

const t = getTacticalText('fr')

describe('barre des quatre statuts', () => {
  it('comptes du serveur, parts sur les morts examinées, dans l’ordre de la barre', () => {
    const parts = partsDeBarre(BILAN.morts!)
    expect(parts.map((p) => [p.statut, p.n])).toEqual([
      ['accompagnee', 4],
      ['visible_hors_rayon', 6],
      ['aucun_visible', 1],
      ['equipe_a_terre', 3],
    ])
    expect(parts[1].part).toBeCloseTo(6 / 14)
  })
})

describe('rose', () => {
  it('un bâton par secteur non vide, le plus long ramené à 1, secteur k à 45 k degrés', () => {
    const batons = batonsDeLaRose([1, 4, 5, 2, 0, 0, 1, 1])
    expect(batons.map((b) => b.secteur)).toEqual([0, 1, 2, 3, 6, 7])
    expect(batons.find((b) => b.secteur === 2)).toMatchObject({ longueur: 1, angleDeg: 90 })
    expect(batonsDeLaRose([0, 0, 0, 0, 0, 0, 0, 0])).toEqual([])
  })

  it('direction dominante : le secteur le plus fourni (à égalité, le premier de la rose)', () => {
    expect(directionDominante([1, 4, 5, 2, 0, 0, 1, 1])).toEqual({ n: 5, secteur: 2 })
    expect(directionDominante([3, 0, 0, 0, 0, 0, 0, 3])).toEqual({ n: 3, secteur: 0 })
    expect(directionDominante([0, 0, 0, 0, 0, 0, 0, 0])).toBeNull()
  })

  it('la phrase dit le secteur dominant du plan, sans « non mesuré »', () => {
    expect(phraseDeLaRose(t, BILAN.morts!)).toBe('5 sur 14 par la droite')
  })
})

describe('fiche d’une mort', () => {
  it('le tueur d’abord, puis les coéquipiers visibles du plus proche au plus loin, puis en attente', () => {
    const l = lignesDeLaFiche(t, MORT_COMPLETE, 'fr')
    expect(l.map((x) => x.role)).toEqual(['tueur', 'coequipier', 'coequipier', 'coequipier'])
    expect(l[0]).toMatchObject({ texte: 'Hollow Vex · BR75', distanceMort: '15' })
    expect(l[1]).toMatchObject({ texte: 'Kestrel 77', marque: 'plein', distanceMort: '12', distanceTueur: '27' })
    expect(l[2]).toMatchObject({ texte: 'NovaDrift', marque: 'creux', distanceMort: '20' })
    expect(l[3]).toMatchObject({ texte: 'Brisk Otter · en attente de réapparition', marque: 'aucune', distanceMort: null })
  })

  it('résumé : dans le rayon, au-delà, hors jeu', () => {
    expect(resumeDeLEquipe(t, MORT_COMPLETE, 'fr')).toBe('1 coéquipier dans le rayon de 18 m, 1 au-delà, 1 hors jeu.')
    expect(resumeDeLEquipe(t, { ...MORT_COMPLETE, equipe: undefined }, 'fr')).toBeNull()
  })
})
describe('scènes du plan', () => {
  it('nuage de la zone : les coéquipiers visibles de chaque mort, rien d’autre', () => {
    const s = sceneDeZone([MORT_COMPLETE, FRAG])
    expect(s.marques).toHaveLength(2)
    expect(s.marques.every((m) => m.role === 'nuage')).toBe(true)
    expect(s.cercle).toBeNull()
  })

  it('instantané : cercle du rayon au lieu de la mort, mort, coéquipiers pleins dans le rayon et creux au-delà', () => {
    const s = sceneDeMort(MORT_COMPLETE)
    expect(s.cercle).toEqual({ centre: { x: 10, y: 10 }, rayonM: 18 })
    expect(s.marques.map((m) => [m.role, m.plein])).toEqual([
      ['mort', true],
      ['coequipier', true],
      ['coequipier', false],
    ])
  })

  it('un frag n’a pas de scène d’équipe', () => {
    expect(sceneDeMort(FRAG).marques).toEqual([])
  })

  it('séquence : le cercle suit la victime avant la mort puis reste au lieu de la mort', () => {
    const avant = sceneDeSequence(SEQUENCE, 0) // −4 s
    expect(avant.cercle?.centre).toEqual({ x: 0, y: 10 })
    expect(avant.marques.find((m) => m.role === 'mort')?.pt).toEqual({ x: 0, y: 10 })
    const apres = sceneDeSequence(SEQUENCE, 6) // +2 s : la victime n'a plus de piste
    expect(apres.cercle?.centre).toEqual({ x: 10, y: 10 })
    expect(apres.marques.find((m) => m.role === 'mort')?.pt).toEqual({ x: 10, y: 10 })
    // Le tueur n'a plus d'échantillon à +2 s : il n'est pas peint.
    expect(apres.marques.map((m) => m.role).sort()).toEqual(['coequipier', 'mort'])
  })

  it('séquence : la traîne couvre les 4 s précédentes et se coupe à une lacune du film', () => {
    // À l'étape 0 s : le tueur a une lacune avant son échantillon à −1 s ; sa traîne se coupe en deux segments.
    const s = sceneDeSequence(SEQUENCE, 4)
    const tueur = s.traines.find((x) => x.role === 'tueur')
    expect(tueur?.segments.map((seg) => seg.length)).toEqual([3, 2])
    const victime = s.traines.find((x) => x.role === 'mort')
    expect(victime?.segments[0].length).toBeGreaterThan(2)
  })

  it('séquence : un acteur sans position à l’instant n’est pas peint', () => {
    const s = sceneDeSequence(SEQUENCE, 7) // +3 s : le tueur n'a plus d'échantillon
    expect(s.marques.some((m) => m.role === 'tueur')).toBe(false)
  })

  it('la scène de la vue : étape > instantané > nuage', () => {
    const vue = { index: 0, contribution: MORT_COMPLETE, sequence: SEQUENCE, etape: 4, onEtape: () => {} }
    expect(sceneDeLaVue([MORT_COMPLETE], vue).traines.length).toBeGreaterThan(0)
    expect(sceneDeLaVue([MORT_COMPLETE], { ...vue, etape: null }).traines).toEqual([])
    expect(sceneDeLaVue([MORT_COMPLETE], null).marques.every((m) => m.role === 'nuage')).toBe(true)
  })

  it('distances d’étape : au centre du cercle, « — » sans position', () => {
    const d = distancesDEtape(t, SEQUENCE, 4, 'fr')
    expect(d.find((x) => x.role === 'tueur')?.distance).toBe('15')
    expect(d.find((x) => x.role === 'coequipier')?.distance).toBe('12')
    expect(distancesDEtape(t, SEQUENCE, 7, 'fr').find((x) => x.role === 'tueur')?.distance).toBeNull()
  })
})

describe('étapes et requête', () => {
  it('libellés de décalage', () => {
    expect([-4000, 0, 3000].map((ms) => libelleDeDecalage(ms, 'fr'))).toEqual(['−4,0 s', '0,0 s', '+3,0 s'])
    expect(libelleDeDecalage(-1500, 'en')).toBe('−1.5 s')
  })

  it('la séquence est demandée pour toute mort dont la victime est nommée, rejeu construit ou non', () => {
    expect(sequenceDemandee(MORT_COMPLETE)).toEqual({ match_id: 'm1', victim_xuid: '111', time_ms: 462_000 })
    // Seul le serveur sait dire pourquoi une séquence manque : la liste n'écarte rien.
    expect(sequenceDemandee({ ...MORT_COMPLETE, replay_available: false })).toEqual({ match_id: 'm1', victim_xuid: '111', time_ms: 462_000 })
    expect(sequenceDemandee({ ...MORT_COMPLETE, xuid: '' })).toBeNull()
    expect(sequenceDemandee(FRAG)).toBeNull()
    expect(sequenceDemandee(undefined)).toBeNull()
  })
})

describe('les mots de la lecture des coéquipiers', () => {
  it('« instant » (moment du chrono), jamais « heure » ; aucun « non mesuré » ; aucun possessif de personne', async () => {
    const { tacticalManifest } = await import('@/lib/i18n/generated/tactical')
    const textes = Object.entries(tacticalManifest as Record<string, { fr: string }>)
      .filter(([cle]) => cle.startsWith('tactical.mort.'))
      .map(([, v]) => v.fr)
    expect(textes.length).toBeGreaterThan(40)
    for (const texte of textes) {
      expect(texte).not.toMatch(/non mesur|\bheure\b|\b(ma|mes|mon|moi|notre)\b/i)
    }
  })

  it('les états vides de la séquence sont des constats : aucune action de l’utilisateur, aucun vocabulaire interne', async () => {
    const { tacticalManifest } = await import('@/lib/i18n/generated/tactical')
    const textes = Object.entries(tacticalManifest as Record<string, { fr: string; en: string }>)
      .filter(([cle]) => /^tactical\.mort\.seq_(missing_|only_deaths_|pending|loading)/.test(cle))
      .flatMap(([, v]) => [v.fr, v.en])
    expect(textes.length).toBeGreaterThanOrEqual(18)
    for (const texte of textes) {
      expect(texte).not.toMatch(/r[ée]analys|re-?analys|analyse[a-z ]* again|cuisson|re-?cuir|bake|artefact|artifact|reconstru|rebuild|relanc|\bretry\b|\bà (refaire|lancer)\b/i)
    }
  })
})

describe('séquence : plein / creux selon le rayon, publié par le serveur', () => {
  it('un coéquipier est plein quand il figure parmi ceux du rayon à l’étape, creux sinon', () => {
    const avant = sceneDeSequence(SEQUENCE, 0) // −4 s : personne dans le rayon
    expect(avant.marques.find((m) => m.role === 'coequipier')?.plein).toBe(false)
    const pendant = sceneDeSequence(SEQUENCE, 4) // 0 s : 222 dans le rayon
    expect(pendant.marques.find((m) => m.role === 'coequipier')?.plein).toBe(true)
    expect(distancesDEtape(t, SEQUENCE, 0, 'fr').find((d) => d.role === 'coequipier')?.marque).toBe('creux')
    expect(distancesDEtape(t, SEQUENCE, 4, 'fr').find((d) => d.role === 'coequipier')?.marque).toBe('plein')
  })
})
