/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) : LES FAMILLES DE POSE SONT ÉNUMÉRÉES EN QUATRE ENDROITS.
 *
 * Une famille d'objet d'équipement existe à quatre endroits, et il n'y a pas de compilateur
 * entre eux :
 *   1. le VALIDEUR du serveur — `equipmentFamilies` (loader_replay_labels_equipment.go),
 *      liste FERMÉE des familles qu'un manifeste de titre a le droit de nommer ;
 *   2. la TABLE du titre      — `[[equipment_objects]]` de replay_labels.toml (`family = …`) ;
 *   3. le RENDU du client     — `PLACEMENT_RENDER` (equipmentPlacementsLayer.ts) ;
 *   4. les LIBELLÉS du client — `placementFamily` (i18n.ts), FR et EN.
 *
 * CE QUI CASSE SANS CE TEST, ET SILENCIEUSEMENT. Une famille ajoutée au manifeste et acceptée
 * par le valideur, mais absente de la table de rendu, ne dessine RIEN : la pose est publiée,
 * décodée, transmise, et l'écran reste vide sans qu'aucune erreur ne soit levée. Une règle de
 * rendu sans libellé rendrait l'infobulle muette. Les deux dérives sont invisibles à
 * l'exécution — c'est exactement ce qu'un garde-rail doit attraper. Modèle : fxInk.guard.test.ts.
 *
 * LES DEUX NIVEAUX NE SE CONFONDENT PAS, et c'est le point délicat de ce fichier :
 * `PLACEMENT_RENDER` est indexée par FAMILLE et rend une RÈGLE DE RENDU (`PlacementKind`) —
 * deux familles peuvent partager un tracé — tandis que `placementFamily` (i18n) est indexée
 * par RÈGLE. Le test relie donc famille → règle → libellé, jamais famille → libellé.
 *
 * LES ABSENCES SONT DES DÉCISIONS, et elles sont NOMMÉES ici. `null` dans la table = famille
 * connue qu'on ne dessine pas (objets portés). Absente de la table = décision assumée, et la
 * seule aujourd'hui porte sur les power-ups de carte (registre des reports, 2026-08-18 : un
 * seul membre mesuré chacun, tous deux `dropped`). Un ajout au valideur qui ne serait ni
 * dessiné ni listé ici fait ROUGIR ce test — c'est voulu.
 *
 * CETTE TABLE NE DIT RIEN DES LÂCHERS depuis le 2026-08-19 : un power-up LÂCHÉ se dessine, et
 * il reste pourtant hors table. La règle est ORTHOGONALE (une ORIGINE croisée avec une liste
 * de familles) et son garde-rail est `placementDropped.guard.test.ts` — ne pas « corriger »
 * l'absence des power-ups ci-dessous en les ajoutant à `PLACEMENT_RENDER`.
 */
import { describe, expect, it } from 'vitest'

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { PLACEMENT_RENDER, type PlacementKind } from './equipmentPlacementsLayer'
import { REPLAY_TEXT } from './i18n'

const REPO = resolve(__dirname, '..', '..', '..', '..', '..')
const GO_LOADER = resolve(
  REPO,
  'apps/go-api/internal/games/mappings/loader_replay_labels_equipment.go',
)
const TOML = resolve(REPO, 'config/titles/halo_infinite/mappings/replay_labels.toml')

/** La famille par défaut du serveur : admise et dessinée, jamais nommée. */
const DEFAULT_FAMILY = 'other'

/** La règle de rendu du défaut : dessinée, sans libellé de famille. */
const DEFAULT_KIND = 'unnamed'

/**
 * Les familles que le valideur admet MAIS que le rendu laisse dehors, par décision écrite.
 * Toute autre absence est un oubli, et ce test la signale.
 */
const HORS_TABLE_ASSUME = ['powerup_camo', 'powerup_overshield']

/** Les familles admises par le VALIDEUR Go (la liste fermée d'`equipmentFamilies`). */
function goFamilies(): string[] {
  const src = readFileSync(GO_LOADER, 'utf8')
  const start = src.indexOf('var equipmentFamilies = map[string]bool{')
  expect(start, 'equipmentFamilies introuvable dans le loader Go').toBeGreaterThan(-1)
  const body = src.slice(start, src.indexOf('\n}', start))
  const noms = [...body.matchAll(/"(\w+)":\s*true/g)].map((m) => m[1])
  // `equipFamilyOther` est servi par une CONSTANTE Go, pas par un littéral : la regex ne peut
  // pas l'attraper, et l'oublier ferait croire que le défaut n'est pas admis.
  if (body.includes('equipFamilyOther:')) noms.push(DEFAULT_FAMILY)
  return [...new Set(noms)].sort()
}

/**
 * Les familles que la table du titre emploie réellement (`family = "…"`).
 *
 * LE BLOC EST BORNÉ PAR LA TABLE SUIVANTE, et ce n'est pas une précaution théorique : le
 * manifeste porte depuis le 2026-08-18 une table `[[objective_objects]]` qui emploie AUSSI une
 * clé `family` — d'un autre vocabulaire (le drapeau de CTF, archétype `ti=42`), validé par une
 * autre liste fermée. Sans cette borne, la lecture allait jusqu'à la fin du fichier et
 * réclamait au valideur des poses une famille qui ne lui appartient pas.
 */
function tomlFamilies(): string[] {
  const src = readFileSync(TOML, 'utf8')
  const start = src.indexOf('[[equipment_objects]]')
  expect(start, '[[equipment_objects]] introuvable dans le manifeste du titre').toBeGreaterThan(-1)
  const block = blocDeTable(src.slice(start), '[[equipment_objects]]')
  return [...new Set([...block.matchAll(/^family\s*=\s*"(\w+)"/gm)].map((m) => m[1]))].sort()
}

/** Rend le préfixe de `src` qui n'appartient qu'à la table `entete` (répétée ou non). */
function blocDeTable(src: string, entete: string): string {
  const lignes = src.split('\n')
  const fin = lignes.findIndex((l) => l.trimStart().startsWith('[[') && l.trim() !== entete)
  return fin < 0 ? src : lignes.slice(0, fin).join('\n')
}

/** Les règles de rendu réellement employées par la table, sans le défaut. */
function kindsNommes(): string[] {
  const kinds = Object.values(PLACEMENT_RENDER).filter(
    (k): k is Exclude<PlacementKind, typeof DEFAULT_KIND> => k !== null && k !== DEFAULT_KIND,
  )
  return [...new Set<string>(kinds)].sort()
}

describe('garde-rail : le vocabulaire des familles de pose', () => {
  it('toute famille de la table de RENDU est admise par le valideur Go', () => {
    for (const f of Object.keys(PLACEMENT_RENDER)) {
      expect(goFamilies(), `${f} dessine, mais le valideur Go la refuserait`).toContain(f)
    }
  })

  it('toute famille admise par le Go est dessinée, ou hors table PAR DÉCISION ÉCRITE', () => {
    const dessinees = Object.keys(PLACEMENT_RENDER)
    for (const f of goFamilies()) {
      if (dessinees.includes(f)) continue
      expect(HORS_TABLE_ASSUME, `${f} est admise par le Go et ne dessine rien`).toContain(f)
    }
  })

  it('chaque famille que le titre emploie est admise ET connue du rendu', () => {
    for (const f of tomlFamilies()) {
      expect(goFamilies(), `${f} n'est pas admise par le valideur Go`).toContain(f)
      if (HORS_TABLE_ASSUME.includes(f)) continue
      expect(Object.keys(PLACEMENT_RENDER), `${f} ne dessine rien côté client`).toContain(f)
    }
  })

  it('chaque RÈGLE DE RENDU nommée a son libellé en FR et en EN', () => {
    for (const locale of ['fr', 'en'] as const) {
      const labels = REPLAY_TEXT[locale].placementFamily as Record<string, string>
      expect(Object.keys(labels).sort(), `libellés ${locale} désynchronisés`).toEqual(kindsNommes())
      for (const k of kindsNommes()) expect(labels[k], `${k} sans libellé ${locale}`).toBeTruthy()
    }
  })

  it('la famille par défaut est admise et dessinée, mais jamais nommée', () => {
    expect(goFamilies()).toContain(DEFAULT_FAMILY)
    expect(PLACEMENT_RENDER[DEFAULT_FAMILY]).toBe(DEFAULT_KIND)
    expect(Object.keys(REPLAY_TEXT.fr.placementFamily)).not.toContain(DEFAULT_KIND)
  })
})

/**
 * Garde-rail de TAILLE — le cliquet du registre des reports (2026-08-16 : « prochaine addition
 * sur l'un d'eux : extraire d'abord »). Le canvas du rejeu porte une dette de taille gelée ;
 * le lot des poses l'avait fait passer de 861 à 942 lignes sans extraction préalable, et c'est
 * ce que la revue a relevé. Le plafond n'est pas un idéal, c'est un CLIQUET : il ne remonte
 * jamais. Le franchir se corrige en extrayant, pas en relevant le nombre.
 *
 * 861 -> 858 le 2026-08-18 (lot des SOCLES D'ARME) : le calque, son survol et son infobulle
 * pesaient une soixantaine de lignes, et le canvas était PILE à son plafond. Les huit encres
 * qui y recopiaient huit fois le même corps sont donc parties dans `useReplayInks` AVANT
 * l'ajout — c'est exactement la manœuvre que ce cliquet existe pour imposer, et le nombre
 * descend d'autant.
 *
 * 858 -> 812 le 2026-08-18 (lot R3, item R3.1) : la croix de mort change de taille, d'encre et
 * de duree, et la duree devait ecrire POURQUOI. Plutot que d'allonger le canvas d'un
 * paragraphe, les reglages temporels et leur conversion sont partis dans `useReplayTiming` —
 * quatrieme extraction imposee par ce cliquet, et la plus grosse.
 *
 * 812 -> 808 le 2026-08-19 (lot des objets LACHES) : le canvas etait PILE a son plafond et le
 * lot y ajoutait une bascule, un comptage, un argument de survol et une prop de garde de mode.
 * Les trois morceaux des poses — comptes, axe de temps, survol — sont donc partis dans
 * `useReplayPlacements` AVANT l'ajout, cinquieme extraction imposee par ce cliquet.
 */
describe('garde-rail : la taille du canvas du rejeu ne remonte pas', () => {
  it('ReplayCanvas.tsx reste sous son plafond', () => {
    const src = readFileSync(resolve(__dirname, 'ReplayCanvas.tsx'), 'utf8')
    expect(src.split('\n').length - 1).toBeLessThanOrEqual(808)
  })
})
