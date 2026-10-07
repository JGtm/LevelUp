/**
 * chaleurLissee — la chaleur CONTINUE du plan, tirée de la grille servie : au lieu de peindre chaque
 * cellule en carré, on la peint en zones de chaleur, comme le rejeu 2D.
 *
 * ─── CE QUE LA PEINTURE DIT, ET CE QU'ELLE NE DIT PAS ──────────────────────────────────────────
 *
 * LA VALEUR EN UN POINT EST LA MOYENNE PONDÉRÉE DES CELLULES SERVIES VOISINES (noyau gaussien,
 * régression de Nadaraya-Watson), PAS UNE DENSITÉ. Le rejeu lisse des POINTS bruts, et sa grandeur
 * est une densité ; ici le serveur a déjà agrégé, normalisé par match et appliqué le plancher de
 * matchs par cellule. Étaler la masse d'une cellule (un flou ordinaire) abaisserait ses pics et
 * ferait mentir la légende, qui est dans l'unité de la lecture (morts par match, secondes…). La
 * moyenne pondérée, elle, garde l'unité : au centre d'une cellule isolée, la couleur est sa valeur,
 * entre deux cellules elle passe de l'une à l'autre, et l'échelle servie (p50 / p95 / borne) reste
 * celle de la rampe.
 *
 * LA CHALEUR NE S'ÉTEND PAS AU-DELÀ DE CE QUI EST SERVI. Un point n'est peint que si le poids des
 * cellules servies qui l'entourent atteint celui d'un point à `PORTEE_PAS` pas du centre d'une
 * cellule isolée : la tache d'une cellule seule est un disque à peine plus large qu'elle, et une cellule
 * sous le plancher (non servie) reste vide, jamais « réchauffée » par ses voisines au-delà de ce
 * débord.
 *
 * LA GRILLE RESTE LE MODÈLE DE SÉLECTION : le clic s'adresse toujours à une cellule servie
 * (`choixDuClic`), le lissage ne change que la peinture.
 */
import type { TacticalGrid } from '@/lib/replay/heatPaint'

/** Écart type du noyau, en pas de grille. */
const SIGMA_PAS = 0.75
/** Rayon de troncature du noyau, en pas de grille. */
const RAYON_PAS = 2
/** Distance au centre d'une cellule isolée jusqu'où sa tache est peinte, en pas de grille. */
const PORTEE_PAS = 0.7
/** Sous-cellules par côté de cellule servie, au plus (la finesse du dégradé). */
const SOUS_MAX = 6
/** Plafond de sous-cellules d'une grille lissée (le même ordre que la carte du rejeu). */
const SOUS_CELLULES_MAX = 250_000

/** Le poids minimal d'un point peint : celui d'un point à `PORTEE_PAS` d'une cellule isolée. */
const POIDS_MIN = Math.exp(-(PORTEE_PAS * PORTEE_PAS) / (2 * SIGMA_PAS * SIGMA_PAS))

/** sousDivision — combien de sous-cellules par côté de cellule, sans dépasser le plafond. */
export function sousDivision(nx: number, ny: number): number {
  const n = Math.max(nx * ny, 1)
  return Math.max(1, Math.min(SOUS_MAX, Math.floor(Math.sqrt(SOUS_CELLULES_MAX / n))))
}

/**
 * lisserLaGrille — la grille de peinture en sous-cellules, chaque sous-cellule portant la moyenne
 * pondérée des cellules servies voisines. Même cadre (origine, emprise), même échelle ; seul le pas
 * change. `null` quand rien n'est à peindre.
 */
export function lisserLaGrille(grille: TacticalGrid): TacticalGrid | null {
  if (grille.cells.length === 0 || !(grille.cell > 0)) return null
  const sous = sousDivision(grille.nx, grille.ny)
  const nx = grille.nx * sous
  const ny = grille.ny * sous
  const somme = new Float64Array(nx * ny)
  const poids = new Float64Array(nx * ny)
  const rayon = Math.ceil(RAYON_PAS * sous)
  const deuxSigma2 = 2 * SIGMA_PAS * SIGMA_PAS
  for (const c of grille.cells) {
    // Le centre de la cellule, en sous-cellules ; une sous-cellule se mesure à son propre centre.
    const cx = (c.col + 0.5) * sous
    const cy = (c.row + 0.5) * sous
    const i0 = Math.max(0, Math.floor(cx - rayon))
    const i1 = Math.min(nx - 1, Math.ceil(cx + rayon))
    const j0 = Math.max(0, Math.floor(cy - rayon))
    const j1 = Math.min(ny - 1, Math.ceil(cy + rayon))
    for (let j = j0; j <= j1; j++) {
      const dy = (j + 0.5 - cy) / sous
      for (let i = i0; i <= i1; i++) {
        const dx = (i + 0.5 - cx) / sous
        const d2 = dx * dx + dy * dy
        if (d2 > RAYON_PAS * RAYON_PAS) continue
        const w = Math.exp(-d2 / deuxSigma2)
        somme[j * nx + i] += w * c.value
        poids[j * nx + i] += w
      }
    }
  }
  const cells: TacticalGrid['cells'] = []
  for (let k = 0; k < poids.length; k++) {
    if (poids[k] < POIDS_MIN) continue
    const value = somme[k] / poids[k]
    if (value === 0 || !Number.isFinite(value)) continue
    cells.push({ col: k % nx, row: Math.floor(k / nx), value })
  }
  if (cells.length === 0) return null
  return { ...grille, cell: grille.cell / sous, nx, ny, cells, filled: cells.length }
}
