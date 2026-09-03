/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) : la table `WALL_PANEL_IDS` du web et le manifeste du titre sont
 * la MÊME liste, rejouée ici depuis le TOML.
 *
 * POURQUOI CETTE TABLE EXISTE CÔTÉ WEB. Le document ne publie par pose que `family` et `id` —
 * jamais le `kind` (`carried` | `deployed`) que le manifeste porte. Or l'arc du mur ne doit se
 * dessiner que sur les PANNEAUX : un mur déployé produit DEUX poses `deployed` (l'appareil qui
 * vole ET ses panneaux), et les dessiner toutes deux ferait deux arcs pour un seul mur. La
 * distinction est donc rejouée côté client à partir des deux identifiants — et sans ce test,
 * elle dériverait EN SILENCE.
 *
 * CE QUE LE TEST FAIT ÉCHOUER, dans les deux sens :
 *  - un TROISIÈME panneau ajouté au manifeste (nouvelle palette de mur, nouvelle saison) que le
 *    web ignorerait : son arc ne se dessinerait jamais ;
 *  - un identifiant du web qui cesserait d'être `kind = deployed` au manifeste : le web
 *    dessinerait un arc sur un objet PORTÉ, c'est-à-dire lâché à la mort de son porteur.
 *
 * LA MESURE QUI SOUTIENT LA RÈGLE (phase G, 2026-08-18) : les deux panneaux sont à 97,9 % et
 * 97,7 % de déploiements (0 lâcher sur 48, 1 sur 43), les deux appareils à 29,4 % et 13,0 %.
 * Rien entre les deux. Même patron que `replaySoundAssets.guard.test.ts` : une table du web
 * rejouée contre sa source Go, parce qu'elle traverse la frontière.
 */
import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { WALL_PANEL_IDS } from './placementWall'

const REPO_ROOT = resolve(__dirname, '..', '..', '..', '..', '..')
const MANIFEST = resolve(
  REPO_ROOT,
  'config/titles/halo_infinite/mappings/replay_labels.toml',
)

/** Un objet d'équipement du manifeste : les trois champs dont ce garde-rail a besoin. */
interface ManifestObject {
  id: string
  family: string
  kind: string
}

/**
 * Les blocs `[[equipment_objects]]` du manifeste.
 *
 * LECTURE À LA MAIN PLUTÔT QU'UN LECTEUR TOML : le web n'en embarque aucun, et en ajouter un
 * pour trois clés serait une dépendance de production payée par un test. Les blocs sont plats
 * (`cle = "valeur"`), la découpe est donc sûre — et un jour où elle ne le serait plus, le test
 * échouerait au lieu de passer à côté (les assertions de cardinalité ci-dessous le garantissent).
 *
 * LA DÉCOUPE EST ANCRÉE SUR LA LIGNE ENTIÈRE, et ce n'est pas un raffinement : une découpe par
 * SOUS-CHAÎNE comptait aussi les mentions du nom de table dans les COMMENTAIRES du manifeste —
 * la cardinalité est passée à 22 le 2026-09-03 quand un commentaire a cité
 * `[[equipment_objects]].family`, et le bloc fantôme en question rendait `id = "famille_a"`,
 * une palette de capacités. Le test a bien parlé, comme son en-tête le promettait.
 */
function manifestObjects(): ManifestObject[] {
  const toml = readFileSync(MANIFEST, 'utf8')
  return toml
    .split(/^\[\[equipment_objects\]\][ \t]*$/m)
    .slice(1)
    .map((bloc) => {
      const champ = (nom: string) =>
        bloc.split('\n').find((l) => l.trimStart().startsWith(nom))?.split('=')[1]?.trim() ?? ''
      const nu = (v: string) => v.replace(/^"|"$/g, '')
      return {
        id: nu(champ('id')),
        family: nu(champ('family')),
        kind: nu(champ('kind')),
      }
    })
    .filter((o) => o.id !== '')
}

describe('garde-rail : panneaux du mur = manifeste du titre', () => {
  const objets = manifestObjects()

  it('le manifeste se lit, et il porte bien ses 21 identifiants', () => {
    // Cardinalité VÉRIFIÉE, pas supposée : si la découpe du TOML cassait, ce test tomberait
    // avant les suivants au lieu de les rendre vides et verts.
    expect(objets).toHaveLength(21)
    for (const o of objets) {
      expect(o.id, JSON.stringify(o)).toMatch(/^0x[0-9a-f]{8}$/)
      expect(['carried', 'deployed'], JSON.stringify(o)).toContain(o.kind)
    }
  })

  it('les identifiants `kind = deployed` du manifeste sont EXACTEMENT ceux du web', () => {
    const deployes = objets
      .filter((o) => o.kind === 'deployed')
      .map((o) => o.id)
      .sort()
    expect(deployes).toEqual([...WALL_PANEL_IDS].sort())
  })

  it('chaque panneau du web est de famille `wall` au manifeste', () => {
    for (const id of WALL_PANEL_IDS) {
      const trouve = objets.filter((o) => o.id === id)
      expect(trouve, `panneau ${id}`).toHaveLength(1)
      expect(trouve[0].family, `panneau ${id}`).toBe('wall')
    }
  })

  it("les APPAREILS du mur restent `carried` : c'est ce qui interdit de leur dessiner l'arc", () => {
    const appareils = objets.filter((o) => o.family === 'wall' && !WALL_PANEL_IDS.includes(o.id))
    expect(appareils.length).toBeGreaterThan(0)
    for (const a of appareils) expect(a.kind, `appareil ${a.id}`).toBe('carried')
  })
})
