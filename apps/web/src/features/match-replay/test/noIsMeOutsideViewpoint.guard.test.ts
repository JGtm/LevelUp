/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6 : à la 3e copie, un helper ET un test qui interdit l'ancien
 * littéral) — LA QUESTION « QUI SUIS-JE ? » A UN SEUL FOYER SUR LA PAGE DE REJEU.
 *
 * # LE DÉFAUT QU'IL PRÉVIENT (2026-09-06, lot L2b du plan « frise, point de vue »)
 *
 * La page de rejeu se regardait implicitement par les yeux du joueur de la page, et cette
 * notion était REDÉCOUVERTE à cinq endroits qui relisaient `is_me` chacun de leur côté : les
 * marques d'identité, le camp de référence des calques, la lecture de fin de match, et deux
 * sections montées ailleurs. Tant que la réponse ne peut pas varier, cinq lectures parallèles
 * ne divergent pas — c'est pour ça que personne ne les avait vues.
 *
 * Le point de vue est maintenant une VALEUR (`model/replayViewpoint.ts`), résolue une fois et
 * passée en paramètre. Un sixième lecteur de `is_me` la contournerait, et le symptôme serait le
 * pire qui soit : la carte suivrait le joueur choisi pendant que ce lecteur-là resterait sur le
 * joueur de la page. Aucun type, aucun test de rendu n'attraperait ça — une couleur de camp est
 * juste une couleur.
 *
 * # CE QU'IL DÉTECTE
 *
 * La LECTURE du drapeau (`x.is_me`, ou `is_me` déstructuré), pas sa mention. Une signature de
 * type (`Pick<MatchScoreboardRow, 'team_side' | 'is_me'>`) et un commentaire ne lisent rien :
 * `endMatchSound.ts` en porte une, et c'est très bien — il reçoit son sujet en paramètre.
 *
 * # CE QU'IL N'INTERDIT PAS
 *
 * Les exemptions ci-dessous, nommées, datées, avec leur raison et leur condition de retrait.
 */
import { describe, expect, it } from 'vitest'

import { fichiersSous, lire, nomDe, racineWeb, tousLesFichiers } from './featureFiles'
import { resolve, sep } from 'node:path'

/**
 * La lecture du drapeau, sous ses deux formes réelles : l'accès par propriété et la
 * déstructuration. Volontairement tolérante aux espaces, pour qu'un reformatage ne la rate pas.
 */
const LECTURE = /\.is_me\b|\bis_me\s*[,}]/

/**
 * Exemptions NOMMÉES, DATÉES, avec leur raison et ce qui les fera disparaître.
 *
 *  - `playerMarks.ts`, `matchSides.ts`, `victoryLogic.ts` (2026-09-06) : le REPLI SANS SUJET.
 *    Les trois prennent désormais le point de vue en paramètre ; sans lui, ils retombent sur la
 *    ligne « moi » — c'est le comportement que la caractérisation L2a a photographié et que la
 *    décision 15 du plan interdit de changer hors de la page de rejeu. RETRAIT CIBLE : quand
 *    plus aucun appel n'omettra le sujet (aujourd'hui `endMatchSound` l'omet exprès, décision 3).
 *
 *  - `MatchPadControlSection.tsx`, `MatchEquipmentUsageSection.tsx` (2026-09-06) : ces deux
 *    composants vivent dans le dossier du rejeu mais sont montés par
 *    `match-view/MatchViewTabChronology.tsx` — l'onglet Chronologie de la page MATCH, où AUCUN
 *    menu de point de vue n'existe. Le point de vue du rejeu ne les concerne pas. RETRAIT
 *    CIBLE : le jour où ils déménageraient dans `match-view/`, ou si un point de vue arrivait
 *    sur la page match — ni l'un ni l'autre n'est prévu.
 *
 * `model/replayViewpoint.ts` N'Y FIGURE PAS, et c'est le résultat du lot : le foyer résout son
 * repli par `meXUIDOf` (`match-view/xuidMeta.ts`, le foyer que le dépôt a déjà pour « qui est le
 * joueur de la page »), plutôt que d'en recopier la lecture une sixième fois.
 */
const EXEMPTIONS = new Map<string, string>([
  ['playerMarks.ts', 'repli sans sujet (2026-09-06)'],
  ['matchSides.ts', 'repli sans sujet (2026-09-06)'],
  ['victoryLogic.ts', 'repli sans sujet (2026-09-06)'],
  ['MatchPadControlSection.tsx', 'monté sur la page match, hors point de vue (2026-09-06)'],
  ['MatchEquipmentUsageSection.tsx', 'monté sur la page match, hors point de vue (2026-09-06)'],
])

/** `lib/replay/` : `playerMarks.ts` y vit, le balayage de la feature ne suffirait donc pas. */
function racineLibReplay(): string {
  return resolve(racineWeb(), 'src', 'lib', 'replay')
}

/** Le chemin relatif à `src/` : le rejeu et `lib/replay/` n'ont pas la même racine. */
function court(chemin: string): string {
  const src = resolve(racineWeb(), 'src')
  return chemin.slice(src.length + 1).split(sep).join('/')
}

/** Les sources balayées : le rejeu ET `lib/replay/`, tests exclus (ils fabriquent des lignes). */
function sourcesBalayees(): string[] {
  return [...tousLesFichiers(), ...fichiersSous(racineLibReplay())].filter(
    (f) => !/\.test\.(ts|tsx)$/.test(f),
  )
}

describe('garde-rail : un seul foyer pour le point de vue de la page de rejeu', () => {
  it('personne ne relit `is_me` hors des exemptions nommées', () => {
    const fautifs = sourcesBalayees()
      .filter((f) => !EXEMPTIONS.has(nomDe(f)))
      .filter((f) => LECTURE.test(lire(f)))
      .map((f) => court(f))
    expect(
      fautifs,
      `ces fichiers relisent \`is_me\` au lieu de recevoir le point de vue en paramètre : ` +
        `[${fautifs.join(', ')}]. Le foyer est \`model/replayViewpoint.ts\` ; ` +
        `voir .ai/PLAN_FRISE_POINT_DE_VUE_2026-09-06.md, lot L2b.`,
    ).toEqual([])
  })

  it('chaque exemption correspond encore à un fichier qui lit vraiment `is_me`', () => {
    // Une exemption périmée est une porte ouverte silencieuse : le jour où le fichier cesse de
    // lire le drapeau, son nom doit sortir de la liste, pas y dormir.
    const perimees = [...EXEMPTIONS.keys()].filter((nom) => {
      const fichiers = sourcesBalayees().filter((f) => nomDe(f) === nom)
      return fichiers.length === 0 || !fichiers.some((f) => LECTURE.test(lire(f)))
    })
    expect(perimees, `exemptions à retirer : [${perimees.join(', ')}]`).toEqual([])
  })

  it('et le balayage voit bien les deux dossiers — sans quoi ce garde ne garderait rien', () => {
    const noms = sourcesBalayees().map((f) => nomDe(f))
    expect(noms).toContain('replayViewpoint.ts')
    expect(noms).toContain('playerMarks.ts')
    expect(sourcesBalayees().length).toBeGreaterThan(200)
  })

  it('le foyer, lui, ne lit PAS `is_me` : il passe par `meXUIDOf`', () => {
    const foyer = sourcesBalayees().find((f) => nomDe(f) === 'replayViewpoint.ts')
    expect(foyer).toBeDefined()
    expect(LECTURE.test(lire(foyer as string))).toBe(false)
    expect(lire(foyer as string)).toContain('meXUIDOf')
  })
})
