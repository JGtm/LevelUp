/**
 * oneLifeDamageGradient — repère commun « une vie » des surfaces Rendement &
 * Résistance (escouade + solo), et son cadre de lecture PARTAGÉ.
 *
 * Le pivot est le même partout : **une vie de Spartan**, exprimée à 100 % dans
 * l'unité de TOUTES les surfaces (escouade et solo tracent le taux rapporté à
 * une vie). Le barème en dégâts bruts du titre ({@link ONE_LIFE_DAMAGE} pour
 * Halo Infinite) ne sert plus qu'à CONVERTIR une valeur brute en taux
 * ({@link oneLifeOffensiveRatePct}, {@link oneLifeDefensiveRatePct}).
 *
 * Deux invariants tiennent l'alignement des surfaces :
 *   1. **Fenêtre FIXE** autour du pivot — {@link oneLifeWindowBounds} : demi-pivot
 *      … double pivot, soit 50…200 % sur le pivot 100. Elle ne dépend JAMAIS des
 *      données affichées, sinon la même valeur change de position (et de
 *      couleur) d'une session à l'autre.
 *   2. **Zones de lecture** — {@link oneLifeZonesMarkArea} : `divergent-pos`
 *      au-dessus du pivot, `divergent-neg` en dessous, à des opacités distinctes
 *      (le rouge pèse plus que le vert à opacité égale). Elles sont posées en
 *      coordonnées d'axe, donc exactes, là où un dégradé est ancré sur la boîte
 *      englobante du tracé.
 *
 * Polarité UNIQUE : toutes les surfaces « une vie » montent quand le joueur va
 * mieux, donc le vert est toujours au-dessus du pivot. Les dégradés de trait
 * (qui servaient l'ancienne courbe solo en dégâts bruts, de polarité inverse)
 * n'existent plus : ECharts ancre un gradient sur la boîte de la série et non
 * sur l'axe, ce qui en faisait un rappel décoratif inexact.
 */
import { resolveToken } from '@/lib/accessibility'

/**
 * Dégâts d'une vie complète de Spartan — barème de conversion brut → taux.
 * DÉFAUT = Halo Infinite (90 vie + 135 bouclier). Les fonctions ci-dessous
 * acceptent un `oneLife` par titre (115 Halo 5…) ; sans argument elles gardent
 * ce barème Infinite.
 */
export const ONE_LIFE_DAMAGE = 225

/** Taux d'UNE VIE, en % — pivot commun de toutes les surfaces. */
export const ONE_LIFE_RATE_PCT = 100

/**
 * Demi-amplitude multiplicative de la fenêtre de lecture : de `pivot / 2` à
 * `pivot × 2`. Sur le pivot 100 % cela donne la fenêtre 50…200 % des surfaces
 * Rendement / Résistance. Une seule constante pour toutes les surfaces.
 */
export const ONE_LIFE_WINDOW_FACTOR = 2

/** Opacités des deux zones de lecture. Le rouge est volontairement PLUS discret
 *  que le vert : à opacité égale il pèse davantage à l'œil. */
export const ONE_LIFE_ZONE_OPACITY = { pos: 0.08, neg: 0.06 } as const

/** Bornes de la fenêtre FIXE encadrant le pivot (jamais dérivées des données). */
export function oneLifeWindowBounds(pivot: number = ONE_LIFE_DAMAGE): {
  min: number
  max: number
} {
  return { min: pivot / ONE_LIFE_WINDOW_FACTOR, max: pivot * ONE_LIFE_WINDOW_FACTOR }
}

/** Fenêtre FIXE en % : 50…200 % autour du pivot « une vie ». Bornes d'axe de
 *  TOUTES les surfaces Rendement / Résistance (escouade et solo). */
export const ONE_LIFE_RATE_BOUNDS = oneLifeWindowBounds(ONE_LIFE_RATE_PCT)

/** Dégâts/mort bruts d'un match : ΣDT / morts. null si non calculable. */
export function damagePerDeath(damageTaken?: number | null, deaths?: number | null): number | null {
  if (damageTaken == null || deaths == null || deaths <= 0) return null
  return Math.round(damageTaken / deaths)
}

/**
 * Taux « une vie » OFFENSIF en %, converti depuis les dégâts par frag effectif :
 * `hp / dmgPerEffectiveFrag × 100`. Miroir front exact de l'indicateur canonique
 * `rendement_offensif` servi par l'API (Go : `hp × (frags + assists/3) / ΣDD`,
 * dont `dmgPerEffectiveFrag` — cf. `effectiveDmgPerFrag` — est l'inverse). À
 * n'employer que sur les payloads qui ne servent PAS l'indicateur natif.
 * 100 % = une vie de dégâts dépensée par frag effectif ; au-dessus = mieux.
 */
export function oneLifeOffensiveRatePct(
  dmgPerEffectiveFrag: number | null | undefined,
  oneLife: number = ONE_LIFE_DAMAGE,
): number | null {
  if (dmgPerEffectiveFrag == null || !Number.isFinite(dmgPerEffectiveFrag)) return null
  if (dmgPerEffectiveFrag <= 0) return null
  return (oneLife / dmgPerEffectiveFrag) * ONE_LIFE_RATE_PCT
}

/**
 * Taux « une vie » DÉFENSIF en %, converti depuis les dégâts encaissés par mort :
 * `dmgPerDeath / hp × 100`. Miroir front exact de l'indicateur canonique
 * `resistance_defensive` servi par l'API (Go : `ΣDT / (hp × morts)`). À
 * n'employer que sur les payloads qui ne servent PAS l'indicateur natif.
 * 100 % = une vie encaissée par mort ; au-dessus = mieux.
 */
export function oneLifeDefensiveRatePct(
  dmgPerDeath: number | null | undefined,
  oneLife: number = ONE_LIFE_DAMAGE,
): number | null {
  if (dmgPerDeath == null || !Number.isFinite(dmgPerDeath)) return null
  if (oneLife <= 0) return null
  return (dmgPerDeath / oneLife) * ONE_LIFE_RATE_PCT
}

/** Une zone `markArea` ECharts : paire [borne basse, borne haute] sur l'axe Y. */
type MarkAreaZone = [
  { yAxis: number; itemStyle: { color: string; opacity: number } },
  { yAxis: number },
]

/**
 * markArea des DEUX zones de lecture d'une surface « une vie », en coordonnées
 * d'axe : zone « favorable » (`divergent-pos`) AU-DESSUS du pivot, zone
 * « défavorable » (`divergent-neg`) en dessous. Source unique de toutes les
 * surfaces — ne jamais ré-écrire les deux zones à la main.
 *
 * Aucune orientation à choisir : toutes les grandeurs tracées sur ce cadre sont
 * des TAUX rapportés à une vie, qui montent quand le joueur va mieux. Une
 * grandeur de polarité inverse (dégâts bruts dépensés par frag) se convertit en
 * taux ({@link oneLifeOffensiveRatePct}) au lieu de retourner les zones.
 *
 * L'opacité suit le TOKEN, pas la position : le rouge reste le plus discret (il
 * pèse davantage à l'œil à opacité égale).
 *
 * Ordre de sortie STABLE : zone haute (pivot → `bounds.max`) puis zone basse
 * (`bounds.min` → pivot).
 */
export function oneLifeZonesMarkArea(
  pivot: number,
  bounds: { min: number; max: number } = oneLifeWindowBounds(pivot),
): { silent: true; data: MarkAreaZone[] } {
  const good = {
    color: resolveToken('divergent-pos'),
    opacity: ONE_LIFE_ZONE_OPACITY.pos,
  }
  const bad = {
    color: resolveToken('divergent-neg'),
    opacity: ONE_LIFE_ZONE_OPACITY.neg,
  }
  return {
    silent: true,
    data: [
      [{ yAxis: pivot, itemStyle: good }, { yAxis: bounds.max }],
      [{ yAxis: bounds.min, itemStyle: bad }, { yAxis: pivot }],
    ],
  }
}
