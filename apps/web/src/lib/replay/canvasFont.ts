/**
 * La famille de police du texte peint dans un canvas du rejeu (libellés, étiquettes, légende) : la pile
 * système de l'interface. Un canvas ne lit pas les feuilles de style : la famille s'écrit dans `ctx.font`,
 * et ce module en est la seule source (`canvasFont.guard.test.ts`).
 */
export const CANVAS_FONT_FAMILY = 'ui-sans-serif, system-ui, sans-serif'
