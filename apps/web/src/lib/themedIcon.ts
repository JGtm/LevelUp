/**
 * themedIcon.ts — le chemin d'une ICÔNE RASTER QUI SUIT LE THÈME, et il n'y en a qu'un.
 *
 * Convention du dépôt : `public/icons/{name}-black.png` pour le thème clair,
 * `public/icons/{name}-white.png` pour le thème sombre (halowaypoint, nemesis, victim,
 * replay). Le suffixe se calculait à la main dans `lib/match-nav/waypointUrl.ts` et dans
 * `features/match-view/MatchNemesisCards.tsx` ; le logo du rejeu 2D en aurait été la
 * troisième copie — règle CLAUDE.md n°6 : à la troisième, on centralise ET on pose un
 * garde-rail (`themedIcon.guard.test.ts`).
 *
 * `UiTheme` n'a que deux valeurs (`dark` | `light`) : pas de « système » à résoudre ici,
 * le store a déjà tranché.
 */
import type { UiTheme } from '@/stores/settingsDraftStore'

/** Chemin public de l'icône `name` dans la variante lisible sur le thème courant. */
export function themedIconSrc(name: string, theme: UiTheme): string {
  return `/icons/${name}-${theme === 'light' ? 'black' : 'white'}.png`
}
