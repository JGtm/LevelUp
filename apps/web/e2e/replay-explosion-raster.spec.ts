/**
 * GARDE-RAIL DE RASTÉRISATION — L'EXPLOSION PEINT-ELLE RÉELLEMENT DES PIXELS, PHASE PAR PHASE ?
 *
 * MÊME PANNE, MÊME PREUVE. `replay-muzzle-raster.spec.ts` a démasqué un éclair de bouche livré
 * INVISIBLE : composé en additif (`lighter`), il ne changeait aucun pixel du thème CLAIR alors
 * que tous ses tests de primitives étaient verts. L'explosion de grenade portait exactement le
 * même défaut à quatre endroits — mesuré avant correction : sur fond clair, 0 pixel changé par
 * le flash, 0 par la boule de feu, 0 par l'onde de choc, 0 par les éclats. Elle n'était pas
 * absente : il n'en restait que la POUSSIÈRE, seule phase composée normalement.
 *
 * POURQUOI PHASE PAR PHASE, ET PAS L'EFFET ENTIER. C'est le piège que la mesure a révélé :
 * l'explosion entière changeait 209 pixels à 320 ms sur fond clair, ce qui semble sain — mais
 * ces 209 pixels venaient TOUS de la poussière. Un garde-rail qui n'observe que le résultat
 * global laisse repasser une explosion amputée de son coup de flash. Chaque phase est donc
 * mesurée SEULE, à l'âge où elle a un sens, sur les deux fonds de chaque thème et pour les
 * trois types de grenade qui détonent.
 *
 * LA DYNAMO N'EXPLOSE PAS (règle utilisateur, lot 2.3) : elle pose une nappe électrique
 * persistante. Elle est vérifiée ici aussi — par son VRAI chemin de rendu
 * (`drawGrenadeRestLayer`), pas par une réplique — parce qu'une nappe éteinte se constaterait
 * de la même façon : des primitives parfaites et une image vide.
 *
 * POURQUOI ICI ET PAS EN VITEST : jsdom n'a pas de moteur 2D. Seul un Chromium réel dit ce que
 * `oklch(...)` et `globalCompositeOperation` produisent. Ce test ne navigue nulle part et ne
 * demande AUCUN serveur : il pose un canevas dans une page vide.
 */
import { test, expect } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import ts from 'typescript'

const ICI = dirname(fileURLToPath(import.meta.url))
const REPO = resolve(ICI, '..', '..', '..')
const FEATURE = resolve(REPO, 'apps/web/src/features/match-replay')
const CSS = resolve(REPO, 'apps/web/src/styles/globals.css')

/** Écart minimal par composante pour qu'un pixel compte comme réellement changé. */
const SEUIL_ECART = 24

const W = 320
const H = 240

/**
 * LES PHASES DE LA TIMELINE, l'âge où chacune se juge, et le plancher de pixels changés.
 *
 * Les planchers sont posés SOUS le pire cas mesuré après correction (colonne « pire »,
 * minimum sur 2 thèmes × 2 fonds × 3 types) et TRÈS AU-DESSUS du zéro d'avant : ils attrapent
 * la panne (rien de peint, composition qui sature, encre absente), pas une retouche de rendu.
 *
 *   phase          | âge    | avant (fond clair) | pire après | plancher
 *   flash          |  40 ms |                  0 |        820 |      300
 *   boule de feu   | 180 ms |                  0 |        194 |       90
 *   onde de choc   | 180 ms |                  0 |        313 |      120
 *   éclats         | 180 ms |                  0 |         43 |       20
 *   poussière      | 900 ms |                 51 |         40 |       18
 *
 * L'onde de choc se juge à 180 ms et pas plus tard : elle s'ouvre en 380 ms et son opacité
 * est déjà sous le seuil de visibilité à 320 ms — un plancher posé là ne mesurerait rien.
 * La poussière, elle, ne prend son ampleur qu'à la fin (elle vit trois fois plus longtemps
 * que les braises) : c'est la seule phase encore présente à 900 ms.
 */
const PHASES = [
  { nom: 'flash', fn: 'drawFlash', ageMs: 40, min: 300 },
  { nom: 'boule de feu', fn: 'drawEmbers', ageMs: 180, min: 90 },
  { nom: 'onde de choc', fn: 'drawWave', ageMs: 180, min: 120 },
  { nom: 'éclats', fn: 'drawSparks', ageMs: 180, min: 20 },
  { nom: 'poussière', fn: 'drawDust', ageMs: 900, min: 18 },
] as const

/** Les trois types qui DÉTONENT, avec la teinte que le titre leur donne (`explosionTintOf`). */
const TYPES: { nom: string; teinte: string }[] = [
  { nom: 'frag', teinte: 'blast' },
  { nom: 'plasma', teinte: 'plasma-cool' },
  { nom: 'spike', teinte: 'needle' },
]

/**
 * Un module de `features/match-replay` transpilé pour la page.
 *
 * COMME LE GARDE-RAIL DE L'ÉCLAIR : pas de bundler. Les modules concernés n'ont que des
 * imports de TYPE (vérifié ci-dessous, assertion explicite) sauf `replayDraw.ts`, dont les
 * dépendances de VALEUR sont injectées nommément — ainsi chaque module garde sa portée et
 * aucune fonction privée homonyme n'en écrase une autre.
 */
function moduleSource(fichier: string, importsDeValeurAutorises = 0): string {
  const src = readFileSync(resolve(FEATURE, fichier), 'utf8')
  const valueImports = [...src.matchAll(/^import\s+(?!type\b)/gm)]
  expect(
    valueImports.length,
    `${fichier} : ses imports de VALEUR ont changé — vérifier que les dépendances injectées par harnais() suffisent encore, puis mettre ce compte à jour`,
  ).toBe(importsDeValeurAutorises)
  return ts
    .transpileModule(src, {
      compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.ESNext },
    })
    .outputText.replace(/^import[^\n]*\n/gm, '')
    .replace(/^export /gm, '')
}

/**
 * Le code sous test, chaque module dans sa propre portée, dépendances injectées.
 *
 * `drawExplosion` et ses phases viennent d'`explosionFx.ts` ; la nappe de la Dynamo n'est
 * atteignable que par `drawGrenadeRestLayer`, qui reçoit ici les VRAIES `worldToCanvas` et
 * `restKindOf` — c'est ce qui fait de ce test une preuve et pas une reconstitution.
 *
 * REMISE À JOUR DU 2026-09-06 (constat M1 du registre d'audit du 2026-09-05). Ce harnais
 * chargeait `replayDraw.ts` avec un cliquet à 7 imports de VALEUR ; le fichier n'en porte plus
 * que 6 (`5a666d2bc`, « suppression du sol reconstruit », a retiré `./mapFloor`) et
 * `drawGrenadeRestLayer` en a été EXTRAIT vers `grenadeRestLayer.ts`. Les deux dérives datent
 * du chantier v7.5 et n'ont jamais rougi : le job qui exécute cette spec ne se déclenche que
 * sur `pull_request` vers `main`, et la campagne travaille en branche unique. Le harnais lit
 * désormais `grenadeRestLayer.ts` (3 imports de valeur, tous injectés ici) — et la spec tourne
 * à chaque push, cf. le step « Rasterisation du rejeu » du job `frontend`.
 *
 * L'ORDRE DES PORTÉES SUIT LES DÉPENDANCES : `grenadeFx.ts` importe `EXPLOSION_MS` depuis
 * `explosionFx.ts` (sa fenêtre de rémanence EST la timeline de l'explosion, invariant du
 * 17/08), donc EXPLO se construit avant GREN et lui injecte la constante.
 */
function harnais(): string {
  return `
    const LOGIC = (function(){ ${moduleSource('replayLogic.ts')}
      return { worldToCanvas } })()
    const EXPLO = (function(){ ${moduleSource('explosionFx.ts')}
      return { EXPLOSION_MS, drawExplosion, drawFlash, drawEmbers, drawWave, drawSparks, drawDust } })()
    const GREN = (function(){
      const EXPLOSION_MS = EXPLO.EXPLOSION_MS
      ${moduleSource('grenadeFx.ts', 1)}
      return { restKindOf, explosionTintOf } })()
    const DRAW = (function(){
      const worldToCanvas = LOGIC.worldToCanvas
      const restKindOf = GREN.restKindOf
      const explosionTintOf = GREN.explosionTintOf
      const drawExplosion = EXPLO.drawExplosion
      ${moduleSource('grenadeRestLayer.ts', 3)}
      return { drawGrenadeRestLayer } })()
    return { EXPLO, DRAW }
  `
}

/** Les valeurs de thème dont les effets se servent, lues dans la feuille de style versionnée. */
interface Theme {
  fx: Record<string, string>
  smoke: string
  halo: string
  fonds: string[]
}

/**
 * Les variables de chaque thème, dans l'ORDRE DE CASCADE de la feuille.
 *
 * Ce n'est pas une coquetterie : globals.css déclare ses variables dans PLUSIEURS blocs
 * `:root` (base, teintes d'effet, palette d'accessibilité), et le thème clair ne redéfinit
 * que ce qui change. Une lecture « le premier bloc c'est sombre, le second c'est clair » a
 * d'abord été écrite ici, et elle rendait le garde-rail INOFFENSIF : les deux thèmes s'y
 * mesuraient sur des fonds sombres, où l'additif fonctionne — le test passait sur la version
 * en panne. Les blocs sont donc fusionnés comme le navigateur le fait : tout `:root` non
 * marqué `light` alimente les deux thèmes, un `:root[data-theme='light']` n'alimente que le
 * clair et écrase ce qui précède.
 */
function themes(): Record<'dark' | 'light', Theme> {
  const css = readFileSync(CSS, 'utf8')
  const dark: Record<string, string> = {}
  const light: Record<string, string> = {}
  for (const bloc of css.matchAll(/([^{}]*)\{([^{}]*)\}/g)) {
    const selecteur = bloc[1]
    if (!selecteur.includes(':root')) continue
    const clairSeulement = selecteur.includes("data-theme='light'")
    for (const l of bloc[2].split('\n')) {
      const m = l.match(/(--[a-z-]+):\s*([^;]+);/)
      if (!m) continue
      if (!clairSeulement) dark[m[1]] = m[2].trim()
      light[m[1]] = m[2].trim()
    }
  }
  const vue = (b: Record<string, string>): Theme => ({
    fx: Object.fromEntries(
      Object.entries(b)
        .filter(([k]) => k.startsWith('--replay-fx-'))
        .map(([k, v]) => [k.replace('--replay-fx-', ''), v]),
    ),
    smoke: b['--muted-foreground'],
    // La nappe électrique et le halo de fin de vol portent le token sémantique `info`,
    // résolu en `--ac-info` par la palette d'accessibilité.
    halo: b['--ac-info'],
    // Les DEUX extrêmes de luminance sur lesquels un effet se pose : la page et la carte.
    fonds: [b['--background'], b['--card']],
  })
  const t = { dark: vue(dark), light: vue(light) }
  // Un thème lu de travers (variable absente, blocs déplacés) rendrait le garde-rail muet :
  // un `fillStyle` vide laisse le fond précédent en place, et l'additif y « marcherait ».
  for (const [nom, v] of Object.entries(t)) {
    const manquants = [
      ...Object.entries({ smoke: v.smoke, halo: v.halo, core: v.fx['core'] }),
      ...v.fonds.map((f, i) => [`fond${i}`, f] as const),
      ...TYPES.map((ty) => [ty.teinte, v.fx[ty.teinte]] as const),
    ]
      .filter(([, valeur]) => !valeur)
      .map(([k]) => k)
    expect(manquants, `thème ${nom} : variables illisibles dans globals.css`).toEqual([])
  }
  expect(t.dark.fonds, 'les fonds des deux thèmes sont identiques — la lecture de globals.css est fausse').not.toEqual(
    t.light.fonds,
  )
  return t
}

/** Compte, dans la page, les pixels que le dessin a réellement changés. */
const COMPTEUR = `
  function mesurer(ctx, W, H, seuil, fond, dessin) {
    ctx.setTransform(1, 0, 0, 1, 0, 0)
    ctx.globalCompositeOperation = 'source-over'
    ctx.globalAlpha = 1
    ctx.fillStyle = fond
    ctx.fillRect(0, 0, W, H)
    const avant = ctx.getImageData(0, 0, W, H).data
    let erreur = ''
    ctx.save()
    try { dessin() } catch (e) { erreur = String(e) }
    ctx.restore()
    const apres = ctx.getImageData(0, 0, W, H).data
    let changes = 0
    for (let i = 0; i < avant.length; i += 4) {
      const d = Math.max(
        Math.abs(apres[i] - avant[i]),
        Math.abs(apres[i + 1] - avant[i + 1]),
        Math.abs(apres[i + 2] - avant[i + 2]),
      )
      if (d >= seuil) changes++
    }
    return { changes, erreur }
  }
`

interface Mesure {
  cle: string
  changes: number
  min: number
  erreur: string
}

function verdict(mesures: Mesure[]): void {
  expect(mesures.length, 'aucune combinaison mesurée').toBeGreaterThan(0)
  expect(mesures.filter((m) => m.erreur).map((m) => `${m.cle} : ${m.erreur}`)).toEqual([])
  const invisibles = mesures
    .filter((m) => m.changes < m.min)
    .map((m) => `${m.cle} = ${m.changes} px (plancher ${m.min})`)
  expect(invisibles, 'phases illisibles — la scène ne change pas assez pour être vue').toEqual([])
}

test.describe('garde-rail : l’explosion de grenade peint des pixels', () => {
  test('chaque phase change la scène, dans les deux thèmes et sur les deux fonds', async ({
    page,
  }) => {
    await page.setContent(`<canvas id="c" width="${W}" height="${H}"></canvas>`)
    const mesures = await page.evaluate(
      ({ js, compteur, th, phases, types, W, H, SEUIL }) => {
        const mod = new Function(js)() as {
          EXPLO: Record<string, (...a: unknown[]) => void>
        }
        const mesurer = new Function(`${compteur}\nreturn mesurer`)() as (
          ctx: CanvasRenderingContext2D,
          w: number,
          h: number,
          seuil: number,
          fond: string,
          dessin: () => void,
        ) => { changes: number; erreur: string }
        const ctx = (document.getElementById('c') as HTMLCanvasElement).getContext('2d')
        if (!ctx) throw new Error('pas de contexte 2D')
        const STEP = 1000 / 60
        const out: { cle: string; changes: number; min: number; erreur: string }[] = []
        for (const theme of ['dark', 'light'] as const) {
          const t = th[theme]
          for (let f = 0; f < t.fonds.length; f++) {
            for (const type of types) {
              const ink = { fire: t.fx[type.teinte], core: t.fx['core'], smoke: t.smoke }
              for (const ph of phases) {
                const shape = {
                  x: W / 2,
                  y: H / 2,
                  ageMs: ph.ageMs,
                  seed: 4242,
                  k: 1,
                  reduced: false,
                }
                const steps = Math.floor(ph.ageMs / STEP)
                const fn = mod.EXPLO[ph.fn]
                if (!fn) {
                  out.push({
                    cle: `${theme}/${ph.nom}`,
                    changes: 0,
                    min: ph.min,
                    erreur: `phase ${ph.fn} introuvable dans explosionFx.ts (renommée ?)`,
                  })
                  continue
                }
                const r = mesurer(ctx, W, H, SEUIL, t.fonds[f], () => fn(ctx, shape, ink, steps))
                out.push({
                  cle: `${theme}/fond${f}/${type.nom}/${ph.nom}@${ph.ageMs}ms`,
                  changes: r.changes,
                  min: ph.min,
                  erreur: r.erreur,
                })
              }
            }
          }
        }
        return out
      },
      { js: harnais(), compteur: COMPTEUR, th: themes(), phases: PHASES, types: TYPES, W, H, SEUIL: SEUIL_ECART },
    )
    verdict(mesures)
  })

  /**
   * LA DYNAMO N'EXPLOSE PAS : sa nappe électrique doit se voir elle aussi sur les deux fonds,
   * du premier instant à la fin de sa rémanence (~2,5 s). Elle n'a jamais été composée en
   * additif — ce test la GARDE : c'est une nappe de traits fins, la première à disparaître
   * si quelqu'un la repasse un jour en `lighter` « pour l'harmoniser » avec les explosions.
   */
  test('la nappe électrique de la Dynamo reste visible dans les deux thèmes', async ({ page }) => {
    /**
     * Plancher. Mesuré : 153 px à l'instant du dépôt, 122 à mi-rémanence, et le même chiffre
     * dans les deux thèmes — la nappe porte le token `info` de la palette, qui ne change pas
     * avec le thème, et elle n'a JAMAIS été composée en additif. C'est bien un constat, pas
     * une correction : rien n'a été touché ici.
     */
    const MIN_NAPPE = 50
    await page.setContent(`<canvas id="c" width="${W}" height="${H}"></canvas>`)
    const mesures = await page.evaluate(
      ({ js, compteur, th, W, H, SEUIL, MIN }) => {
        const mod = new Function(js)() as {
          DRAW: { drawGrenadeRestLayer: (...a: unknown[]) => void }
        }
        const mesurer = new Function(`${compteur}\nreturn mesurer`)() as (
          ctx: CanvasRenderingContext2D,
          w: number,
          h: number,
          seuil: number,
          fond: string,
          dessin: () => void,
        ) => { changes: number; erreur: string }
        const ctx = (document.getElementById('c') as HTMLCanvasElement).getContext('2d')
        if (!ctx) throw new Error('pas de contexte 2D')
        // Un cadrage 1 m = 1 px centré sur l'origine : la nappe se pose au milieu du canevas.
        const view = { bounds: { minX: -W / 2, maxX: W / 2, minY: -H / 2, maxY: H / 2 }, width: W, height: H, pad: 0 }
        // Rang 2 = Shock/Dynamo. C'est `restKindOf` (le vrai) qui en fait une nappe.
        const fx = [{ frame: 0, x: 0, y: 0, rank: 2, rest: false, seed: 991 }]
        const out: { cle: string; changes: number; min: number; erreur: string }[] = []
        for (const theme of ['dark', 'light'] as const) {
          const t = th[theme]
          for (let f = 0; f < t.fonds.length; f++) {
            // Début et MILIEU de rémanence. Pas la toute fin : la nappe s'éteint en fondu,
            // et exiger qu'elle change encore la scène à son dernier souffle interdirait le
            // fondu lui-même. Ce qui se garde, c'est qu'elle EXISTE tant qu'elle est là.
            for (const age of [0, 12]) {
              const style = {
                ink: { tint: t.fx, core: t.fx['core'] },
                smoke: t.smoke,
                halo: t.halo,
                k: 1,
                reducedMotion: false,
              }
              const win = { frame: age, holdHalo: 14, holdDynamo: 25, frameMs: 100 }
              const r = mesurer(ctx, W, H, SEUIL, t.fonds[f], () =>
                mod.DRAW.drawGrenadeRestLayer(ctx, fx, view, win, style),
              )
              out.push({
                cle: `${theme}/fond${f}/nappe@${age}f`,
                changes: r.changes,
                min: MIN,
                erreur: r.erreur,
              })
            }
          }
        }
        return out
      },
      { js: harnais(), compteur: COMPTEUR, th: themes(), W, H, SEUIL: SEUIL_ECART, MIN: MIN_NAPPE },
    )
    verdict(mesures)
  })
})
