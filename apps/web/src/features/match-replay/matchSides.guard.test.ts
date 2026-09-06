/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) — LES DEUX LECTURES DU TABLEAU DE SCORE ONT UN SEUL FOYER.
 *
 * POURQUOI CE GARDE (registre 2026-09-05, K2). « Quelle est MON équipe » était écrite quatre
 * fois — trois copies dans des hooks de calque, plus une canonique cachée dans le module des
 * SONS d'objectif, un foyer que personne ne va chercher pour peindre une onde. « L'équipe de
 * chaque xuid » l'était deux fois, byte pour byte. Une divergence entre deux de ces lectures
 * ne se voit pas : elle donne la couleur de l'ennemi à un allié, sur un seul calque, sans
 * qu'aucun type ni aucun test ne s'en aperçoive.
 *
 * CE QU'IL DÉTECTE : la formule elle-même, sous ses deux formes — la ligne « moi » cherchée à
 * la main, et la table `xuid -> équipe` reconstruite à la main.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'

/**
 * « MON équipe » : la FORMULE entière — la ligne « moi » cherchée à la main, puis son camp
 * parsé. Chercher la ligne `is_me` reste permis pour en lire autre chose (l'issue du match,
 * la ligne complète d'un tableau) : ce qui est interdit, c'est la seconde lecture du CAMP.
 */
const CAMP_ALLIE = /parseTeamSideID\([^)]*\.find\(\([^)]*\) => \w+\.is_me\)/

/**
 * La table `xuid -> équipe` reconstruite à la main : c'est la LIGNE DE REMPLISSAGE qui la
 * signe, pas le type de la carte — d'autres tables `string -> number` existent (l'indice de
 * film d'un siège, la taille d'un véhicule) et n'ont rien à voir.
 */
const TABLE_XUID = /map\.set\(\w+\.xuid, team\)/

const AUTORISES = new Set(['matchSides.ts', 'matchSides.guard.test.ts'])

function fautifs(motif: RegExp): string[] {
  return readdirSync(__dirname)
    .filter((n) => /\.(ts|tsx)$/.test(n) && !AUTORISES.has(n))
    .filter((n) => motif.test(readFileSync(join(__dirname, n), 'utf8')))
}

describe('garde-rail : un seul foyer pour les camps du match', () => {
  it('personne ne recherche la ligne « moi » du tableau de score', () => {
    expect(fautifs(CAMP_ALLIE)).toEqual([])
  })

  it('personne ne rebâtit la table xuid -> équipe', () => {
    expect(fautifs(TABLE_XUID)).toEqual([])
  })

  it('et `matchSides` porte bien les deux — sans quoi ce test ne garderait rien', () => {
    const src = readFileSync(join(__dirname, 'matchSides.ts'), 'utf8')
    expect(CAMP_ALLIE.test(src)).toBe(true)
    expect(TABLE_XUID.test(src)).toBe(true)
  })
})
