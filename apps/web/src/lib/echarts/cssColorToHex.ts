/**
 * cssColorToHex — NORMALISE UNE COULEUR CSS EN NOTATION QUE ZRENDER SAIT PARSER.
 *
 * POURQUOI CE FICHIER EXISTE. Les jetons sémantiques du thème sont écrits en `oklch(...)`
 * (`--muted-foreground: oklch(0.708 0 0)`, cf. `styles/globals.css`). Le canvas, lui, accepte
 * `oklch` : une forme peinte avec cette valeur s'affiche correctement. Mais ECharts ne se
 * contente pas de peindre — au survol, `zrender` DÉRIVE la couleur d'emphase en parsant la
 * chaîne (`lift()` -> `color.parse()`), et son parseur ne connaît que `#rgb`, `#rrggbb`,
 * `rgb()`, `rgba()`, `hsl()`, `hsla()` et les noms CSS. Sur `oklch(...)` il rend `undefined`,
 * `lift()` propage `undefined`, et le segment survolé perd son remplissage : il DISPARAÎT.
 *
 * LA CONVERSION EST FAITE PAR LE NAVIGATEUR, JAMAIS À LA MAIN. On assigne la chaîne au
 * `fillStyle` d'un contexte 2D et on relit : le moteur rend la forme canonique (`#9ca3af`, ou
 * `rgba(...)` si la couleur a de l'alpha). Aucune table de conversion oklch->sRGB à maintenir,
 * aucune dérive possible avec le rendu réel. C'est la technique de la maquette validée le
 * 2026-09-06 (`.ai/V7.5/MAQUETTE_PORTEE_ENGAGEMENTS_2026-09-06.html`, fonction `toHex`).
 *
 * REPLI : la valeur d'entrée, inchangée. Sans `document` (SSR) ou sans contexte 2D (jsdom
 * sous vitest rend `null` : « Not implemented: HTMLCanvasElement's getContext() »), on ne
 * peut rien normaliser — et rendre la valeur d'origine laisse le canvas peindre juste, ce qui
 * est exactement l'état d'avant ce module.
 */

/** Le contexte de sonde, créé au premier appel puis conservé. `null` = indisponible. */
let probe: CanvasRenderingContext2D | null | undefined

function getProbe(): CanvasRenderingContext2D | null {
  if (probe !== undefined) return probe
  if (typeof document === 'undefined') {
    probe = null
    return probe
  }
  try {
    probe = document.createElement('canvas').getContext('2d')
  } catch {
    // jsdom peut LEVER au lieu de rendre `null` selon la version : les deux disent la même
    // chose (pas de moteur de rendu), et les deux mènent au repli.
    probe = null
  }
  return probe
}

/**
 * cssColorToHex — « oklch(0.708 0 0) » -> « #9ca3af ». Rend l'entrée telle quelle si elle
 * n'est pas convertible (pas de canvas, chaîne vide, couleur invalide).
 *
 * LA COULEUR INVALIDE EST DÉTECTÉE, PAS DEVINÉE : `fillStyle` refuse silencieusement une
 * valeur qu'il ne comprend pas et GARDE la précédente. On écrit donc deux sentinelles
 * différentes avant chaque lecture ; deux lectures identiques prouvent que c'est bien
 * l'entrée qui a décidé de la valeur, deux lectures différentes prouvent qu'elle a été
 * ignorée. Sans cette double passe, une couleur invalide sortirait en noir — un rendu faux
 * silencieux, pire que la valeur d'origine.
 */
export function cssColorToHex(css: string): string {
  const ctx = getProbe()
  if (!ctx || !css) return css
  ctx.fillStyle = 'black'
  ctx.fillStyle = css
  const first = ctx.fillStyle
  ctx.fillStyle = 'white'
  ctx.fillStyle = css
  const second = ctx.fillStyle
  if (typeof first !== 'string' || first !== second) return css
  return first
}
