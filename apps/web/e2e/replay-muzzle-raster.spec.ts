/**
 * GARDE-RAIL DE RASTÉRISATION — L'ÉCLAIR DE BOUCHE PEINT-IL RÉELLEMENT DES PIXELS ?
 *
 * CE QUI MANQUAIT, ET CE QUE ÇA A COÛTÉ. Le calque des tirs était testé par un contexte
 * ENREGISTREUR (`test/recordingContext.ts`) : il vérifie les primitives émises — un
 * `createRadialGradient`, un `fill`, la bonne couleur passée. Il ne rastérise rien. Or
 * l'éclair de bouche a été livré INVISIBLE sans qu'un seul test rougisse : en composition
 * additive, la teinte ambrée posée sur le sol quasi blanc du thème CLAIR saturait
 * immédiatement — écart maximal de 3 valeurs sur 255, zéro pixel modifié. Les primitives
 * étaient parfaites ; l'image était vide. Aucun test de primitives ne peut voir ça.
 *
 * CE QUE CE FICHIER VÉRIFIE. Pour CHAQUE famille de rendu servie par le titre et CHAQUE
 * teinte servie, dans les DEUX thèmes et sur DEUX fonds (la page et le sol de la carte),
 * à l'instant du tir : l'éclair peint des pixels, et ces pixels CHANGENT réellement la
 * scène. C'est la seule propriété qui compte pour l'utilisateur.
 *
 * POURQUOI ICI ET PAS EN VITEST. Vitest tourne sous jsdom, qui n'a pas de moteur 2D :
 * `getContext('2d')` n'y peint rien. Seul un Chromium réel peut dire si `oklch(...)` est
 * accepté par `addColorStop`, et à quoi ressemble le résultat. Ce test ne navigue nulle
 * part et ne demande AUCUN serveur : il pose un canevas dans une page vide.
 *
 * COMMENT LE MODULE ARRIVE DANS LA PAGE. `muzzleFlash.ts` n'a que des imports de TYPE : il
 * se transpile donc seul, sans bundler. Si un import de VALEUR y entrait un jour, le test
 * échouerait ici (assertion explicite) plutôt que de mesurer un module amputé.
 */
import { test, expect } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import ts from 'typescript'

import { REPO, sourceDuRejeu } from './_helpers/replaySource'

const CSS = resolve(REPO, 'apps/web/src/styles/globals.css')
const TOML = resolve(REPO, 'config/titles/halo_infinite/mappings/replay_labels.toml')

/**
 * Pixels de la scène que l'éclair doit changer d'au moins 24/255 sur une composante, à
 * l'instant du tir. Le pire cas mesuré après correction est de 121 (thème sombre, gerbe
 * d'aiguilles en plasma froid sur un marqueur) ; le pire cas AVANT était 0. Le seuil est
 * posé bien au-dessus de zéro et bien au-dessous du pire cas réel : il attrape la panne
 * (rien de peint, couleur qui sature, teinte absente), pas une retouche de rendu.
 */
const MIN_PIXELS_CHANGES = 60
/** Écart minimal par composante pour qu'un pixel compte comme réellement changé. */
const SEUIL_ECART = 24

const W = 260
const H = 180

/** Le code de l'éclair, transpilé pour la page. */
function moduleSource(): string {
  const src = sourceDuRejeu('muzzleFlash.ts')
  const valueImports = [...src.matchAll(/^import\s+(?!type\b)/gm)]
  expect(
    valueImports,
    'muzzleFlash.ts a gagné un import de VALEUR : ce garde-rail ne peut plus le transpiler seul (bundler requis)',
  ).toHaveLength(0)
  return ts
    .transpileModule(src, {
      compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.ESNext },
    })
    .outputText.replace(/^import[^\n]*\n/gm, '')
    .replace(/^export /gm, '')
}

/** Les teintes `--replay-fx-*` d'un thème, lues dans la feuille de style versionnée. */
function fxTokens(): { dark: Record<string, string>; light: Record<string, string> } {
  const css = readFileSync(CSS, 'utf8')
  const bloc = (depuis: number) => {
    const i = css.indexOf('--replay-fx-core', depuis)
    expect(i, 'teintes --replay-fx-* introuvables dans globals.css').toBeGreaterThan(-1)
    const corps = css.slice(css.lastIndexOf('{', i), css.indexOf('}', i))
    const out: Record<string, string> = {}
    for (const l of corps.split('\n')) {
      const m = l.match(/--replay-fx-([a-z-]+):\s*([^;]+);/)
      if (m) out[m[1]] = m[2].trim()
    }
    return { out, i }
  }
  const d = bloc(0)
  const l = bloc(d.i + 1)
  return { dark: d.out, light: l.out }
}

/**
 * Le vocabulaire vient du TITRE, pas d'une liste recopiée ici.
 *
 * POURQUOI PAS UN IMPORT DE `SERVED_FX_TINTS` / `familyOf`. Le projet TypeScript des specs
 * E2E est volontairement indépendant de celui de l'application (tsconfig.e2e.json, sans
 * alias `@/`) : une spec qui importerait `src/` ferait échouer `tsc -b`. Et la source est
 * la même de toute façon — `fxInk.guard.test.ts` verrouille déjà l'égalité entre la table
 * du titre, le valideur Go et la liste du client. Ce fichier-ci ne juge pas le
 * vocabulaire, il juge les PIXELS.
 *
 * `plain` s'ajoute aux familles du titre : c'est le repli des armes hors catalogue, il doit
 * peindre lui aussi. La MÊLÉE est exclue — elle n'a pas d'éclair, le filtrage vit en amont
 * (`buildShotFx`).
 */
function bloc(source: string, debut: string, fin?: string): string[] {
  const i = source.indexOf(debut)
  expect(i, `${debut} introuvable dans replay_labels.toml`).toBeGreaterThan(-1)
  const corps = fin ? source.slice(i, source.indexOf(fin)) : source.slice(i)
  return [...new Set([...corps.matchAll(/^\w+\s*=\s*"(\w+)"/gm)].map((m) => m[1]))]
}

function vocabulaireDuTitre(): { familles: string[]; teintes: string[] } {
  const src = readFileSync(TOML, 'utf8')
  const familles = bloc(src, '[shot_effects]', '[shot_tints]').filter((f) => f !== 'melee')
  const teintes = bloc(src, '[shot_tints]')
  expect(familles.length, 'aucun effet de tir lu dans le titre').toBeGreaterThan(0)
  expect(teintes.length, 'aucune teinte lue dans le titre').toBeGreaterThan(0)
  return { familles: [...familles, 'plain'], teintes: [...teintes, 'neutral'] }
}

/** Fonds de scène : la page et le sol de la carte, dans chaque thème. */
const FONDS: Record<'dark' | 'light', string[]> = {
  // `--background` et `--card` de globals.css : les deux extrêmes de luminance sur
  // lesquels l'éclair se pose. Le second (quasi blanc en clair) est celui qui a démasqué
  // la panne de composition.
  dark: ['oklch(0.145 0 0)', 'oklch(0.205 0 0)'],
  light: ['oklch(1 0 0)', 'oklch(0.995 0.002 255)'],
}

test.describe('garde-rail : l’éclair de bouche peint des pixels', () => {
  test('chaque famille et chaque teinte servies changent la scène, dans les deux thèmes', async ({
    page,
  }) => {
    const { familles, teintes } = vocabulaireDuTitre()
    await page.setContent(`<canvas id="c" width="${W}" height="${H}"></canvas>`)

    const mesures = await page.evaluate(
      ({ js, tokens, familles, teintes, fonds, W, H, SEUIL }) => {
        const mod = new Function(`${js}\nreturn { drawMuzzleFlash }`)() as {
          drawMuzzleFlash: (
            ctx: CanvasRenderingContext2D,
            fam: string,
            tint: string,
            shape: Record<string, unknown>,
            ink: { tint: Record<string, string>; core: string },
          ) => void
        }
        const cv = document.getElementById('c') as HTMLCanvasElement
        const ctx = cv.getContext('2d')
        if (!ctx) throw new Error('pas de contexte 2D')
        const encre = (t: Record<string, string>) => ({
          tint: Object.fromEntries(teintes.map((k) => [k, t[k.replace('_', '-')]])) as Record<string, string>,
          core: t['core'],
        })
        const out: { theme: string; fam: string; tint: string; fond: number; changes: number; erreur: string }[] = []
        for (const theme of ['dark', 'light'] as const) {
          const ink = encre(tokens[theme])
          for (let f = 0; f < fonds[theme].length; f++) {
            for (const fam of familles) {
              for (const tint of teintes) {
                ctx.setTransform(1, 0, 0, 1, 0, 0)
                ctx.globalCompositeOperation = 'source-over'
                ctx.globalAlpha = 1
                ctx.fillStyle = fonds[theme][f]
                ctx.fillRect(0, 0, W, H)
                const avant = ctx.getImageData(0, 0, W, H).data
                let erreur = ''
                try {
                  mod.drawMuzzleFlash(
                    ctx,
                    fam,
                    tint,
                    { x: W / 2, y: H / 2, angle: 0, fade: 1, reduced: false, seed: 3, k: 1 },
                    ink,
                  )
                } catch (e) {
                  erreur = String(e)
                }
                const apres = ctx.getImageData(0, 0, W, H).data
                let changes = 0
                for (let i = 0; i < avant.length; i += 4) {
                  const d = Math.max(
                    Math.abs(apres[i] - avant[i]),
                    Math.abs(apres[i + 1] - avant[i + 1]),
                    Math.abs(apres[i + 2] - avant[i + 2]),
                  )
                  if (d >= SEUIL) changes++
                }
                out.push({ theme, fam, tint, fond: f, changes, erreur })
              }
            }
          }
        }
        return out
      },
      {
        js: moduleSource(),
        tokens: fxTokens(),
        familles,
        teintes,
        fonds: FONDS,
        W,
        H,
        SEUIL: SEUIL_ECART,
      },
    )

    expect(mesures.length, 'aucune combinaison mesurée').toBeGreaterThan(0)
    const enErreur = mesures.filter((m) => m.erreur)
    expect(enErreur.map((m) => `${m.theme}/${m.fam}/${m.tint} : ${m.erreur}`)).toEqual([])
    const invisibles = mesures
      .filter((m) => m.changes < MIN_PIXELS_CHANGES)
      .map((m) => `${m.theme}/fond${m.fond}/${m.fam}/${m.tint} = ${m.changes} px`)
    expect(invisibles, `éclairs illisibles (< ${MIN_PIXELS_CHANGES} px changés)`).toEqual([])
  })
})
