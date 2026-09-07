/**
 * fxInk.ts — LES TEINTES D'EFFET DE TIR, lues du thème.
 *
 * CE QUE CE FICHIER TRANCHE, ET POURQUOI CE N'EST PAS UNE ENTORSE À LA RÈGLE DES TOKENS.
 * Le lot 3.2 avait acté « famille = FORME, couleur = TIREUR », et abandonné la palette du
 * POC. L'utilisateur ROUVRE cet arbitrage le 2026-08-15, mot pour mot : « le muzzle flash
 * c'est bien pour les armes CINÉTIQUES ; pour les autres faut des effets de type PLASMA
 * BLEUTÉ OU ROUGEÂTRE pour les armes Banished à énergie, voire d'ÉNERGIE VIOLETTE pour les
 * armes Forerunner », puis, en correction : « les couleurs des effets de tirs ne prennent
 * pas la couleur du tireur [...] elle prend seulement l'ARME en compte ». Ce n'est donc pas
 * un retour en arrière par oubli : la NATURE de la décharge se dit désormais par la couleur,
 * et l'identité du joueur reste portée par son marqueur et son cône de visée.
 *
 * POURQUOI PAS DES TOKENS `--ac-*`. Ceux-là disent un RÔLE DE DONNÉE et sont remappés par
 * les palettes d'accessibilité : sous Okabe-Ito, un plasma bleu deviendrait orange, ce qui
 * détruirait l'information même que la teinte porte. Ces couleurs sont DIÉGÉTIQUES — elles
 * nomment ce que le jeu montre. Elles vivent donc dans le thème (`--replay-fx-*`,
 * globals.css), avec une valeur par thème clair/sombre, et sont lues ici comme
 * `canvasInk.ts` lit les encres de mise en page. La règle est étendue, pas contournée :
 * aucune valeur de couleur n'est écrite dans `features/`.
 *
 * LE VOCABULAIRE EST CELUI DU SERVEUR : `[shot_tints]` du titre (nature de la décharge),
 * publié par arme dans `weaponLabels[id].tint`. Le garde-rail `fxInk.guard.test.ts` vérifie
 * que les deux listes coïncident et que chaque teinte a bien sa variable dans le thème —
 * une teinte ajoutée côté serveur et non stylée retomberait sinon en neutre, en silence.
 */
import { log } from '@/lib/accessibility/_logger'

/** Natures de décharge. `neutral` n'est pas une nature : c'est l'absence de nature connue. */
export type FxTint =
  | 'kinetic'
  | 'plasma_cool'
  | 'plasma_hot'
  | 'forerunner'
  | 'electric'
  | 'needle'
  | 'blast'
  | 'neutral'

/** Les natures que le TITRE peut publier (`neutral` est le repli du client, pas une valeur). */
export const SERVED_FX_TINTS: readonly FxTint[] = [
  'kinetic',
  'plasma_cool',
  'plasma_hot',
  'forerunner',
  'electric',
  'needle',
  'blast',
]

/** Variable de thème d'une teinte. Les tirets suivent la convention CSS, pas le TOML. */
export function fxTintVar(tint: FxTint): string {
  return `--replay-fx-${tint.replace('_', '-')}`
}

/**
 * fxTintOf valide la nature annoncée par le document.
 *
 * INCONNUE OU ABSENTE : `neutral`. On ne rapproche pas d'une teinte voisine — une couleur
 * empruntée affirmerait une nature qu'on ignore, exactement comme le ferait une forme
 * empruntée (même règle que `familyOf`).
 */
export function fxTintOf(raw: string | undefined): FxTint {
  if (!raw) return 'neutral'
  return (SERVED_FX_TINTS as readonly string[]).includes(raw) ? (raw as FxTint) : 'neutral'
}

/** Les encres d'un effet : la teinte de la décharge, et le cœur incandescent commun. */
export interface FxInk {
  tint: Record<FxTint, string>
  /** Le cœur blanc-chaud de l'éclair : la même incandescence quelle que soit la nature. */
  core: string
}

/**
 * readFxInk lit les teintes sur :root. À appeler UNE fois par thème (pas par image) : un
 * `getComputedStyle` par effet et par frame coûterait bien plus que le dessin lui-même.
 *
 * Une variable absente rend une chaîne vide — un `fillStyle` vide laisse le contexte
 * inchangé plutôt que de peindre une couleur inventée (même règle que `readInk`).
 */
export function readFxInk(): FxInk {
  const read = (name: string): string => {
    if (typeof document === 'undefined') return ''
    const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
    if (!value) {
      log.error(`replay-fx:${name}`, `Teinte d'effet "${name}" absente du thème.`)
      return ''
    }
    return value
  }
  const tint = {} as Record<FxTint, string>
  for (const t of [...SERVED_FX_TINTS, 'neutral' as FxTint]) {
    tint[t] = read(fxTintVar(t))
  }
  return { tint, core: read('--replay-fx-core') }
}
