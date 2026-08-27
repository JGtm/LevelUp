/**
 * placementTeleport — LE PASSAGE PAR LA FAILLE : reconnaître, dans les pistes du film,
 * l'instant où un joueur se déplace INSTANTANÉMENT d'un point à un autre.
 *
 * DEUX VERROUS, ET IL FAUT LES DEUX. Ils viennent tous deux de l'utilisateur (2026-08-27) :
 * « les téléportations tu peux sans doute les voir avec un joueur qui se déplace
 * instantanément d'un point à l'autre à une vitesse supérieure à celle de la course », puis
 * « on n'a pas un composant qui permet d'identifier l'équipement ? quand un joueur l'a et
 * quand il l'active ? ». Le second a corrigé la trajectoire de ce module : le film PUBLIE la
 * capacité portée par chaque slot (canal `abilities`, cf. `nearestReading`), et le
 * translocateur quantique y occupe le rang 11.
 *
 *  A. LE PORTEUR — au moment du saut, la dernière lecture de capacité de ce slot vaut 11.
 *     C'est le verrou décisif : il ne repose sur aucune heuristique de mouvement, mais sur un
 *     canal indépendant qui dit qui a l'équipement en main.
 *  B. LA PORTE DE CARTE — le vecteur du saut n'est pas partagé par plusieurs joueurs.
 *
 * CE QUE MESURE LE CORPUS LOCAL (39 films, 2026-08-27), et pourquoi aucun des deux verrous ne
 * suffit seul :
 *  - une frame dure 100 ms. Un Spartan court à ~8 m/s (0,8 m par frame), le grappin culmine
 *    vers 25 m/s (2,5 m). Les 32 sauts relevés font 12 à 47 m en UNE frame, soit 120 à
 *    470 m/s : entre la traversée la plus rapide du jeu et le plus petit saut mesuré, il y a
 *    un facteur cinq. Le seuil n'a donc pas besoin d'être fin ;
 *  - le verrou A SEUL en retient 6. Deux de trop : le slot 742 de `06dfe6d9` franchit deux
 *    fois une porte de la carte pendant qu'il porte le translocateur ;
 *  - le verrou B SEUL en retient 4 — les bons, mais par une heuristique de forme : 28 sauts
 *    portent dans un même film LE MÊME VECTEUR au signe près, ±(38,0 / 12,0) sur l'un,
 *    ±(32,3 / −0,7) sur l'autre, emprunté par plusieurs vies dans les deux sens. Ce sont les
 *    portes de la carte, dont les deux bouches sont fixes ;
 *  - LES DEUX ENSEMBLE en retiennent 4, et les quatre tombent dans une fenêtre où ce slot
 *    précis tient le translocateur (slot 742 : rang 11 lu à t5928, remplacé à t6496, saut à
 *    t6196 ; slot 754 : 11 à t6226, saut à t6517 ; slot 607 : 11 à t5627, sauts à t5863 et
 *    t5899). Deux canaux indépendants qui concordent quatre fois sur quatre.
 *
 * LA FAILLE ELLE-MÊME NE SERT PAS DE PREUVE. Les 7 poses de translocateur du corpus sont
 * toutes d'origine `dropped` — lâchées à la mort du porteur, jamais déployées : aucune faille
 * n'y est jamais dessinée. Une première version exigeait une faille active à l'arrivée et ne
 * rendait donc RIEN. Le champ `viaRift` garde cette corroboration quand elle existe ; il vaut
 * `false` partout dans le corpus local.
 */
import type { ReplayEquipmentPlacement } from '@/lib/api/types'

import type { ReplayDocumentReady, ReplayTrackReady } from './replayNormalize'
import type { XY } from './replayLogic'
import { nearestReading } from './rosterLogic'

/** Famille de la pose, telle que la sert le backend (cf. replay_labels.toml). */
const FAMILLE_FAILLE = 'translocator_beacon'

/**
 * Rang du translocateur quantique dans la palette de capacités du film (famille A). Le rang
 * est établi côté serveur — `internal/analysis/filmdec/translocateur_test.go` en fait foi — et
 * le document le confirme lui-même : `abilityLabels` associe 11 à « translocateur quantique ».
 */
const RANG_TRANSLOCATEUR = 11

/**
 * Distance minimale du saut, en mètres, et fenêtre en frames. À 100 ms la frame, 12 m en 3
 * frames valent déjà 40 m/s — cinq fois la course, et le double du grappin à son maximum.
 */
const SAUT_M_MIN = 12
const SAUT_FRAMES_MAX = 3

/**
 * Tolérance d'appariement de deux vecteurs de saut, en mètres. Les deux bouches d'une porte
 * sont fixes, mais le joueur n'y entre pas deux fois au même centimètre : sur les mesures, un
 * même vecteur varie de ±1 m d'un passage à l'autre. 2 m couvre cette dispersion sans
 * confondre deux portes distinctes (les plus proches mesurées sont à 20 m l'une de l'autre).
 */
const TOLERANCE_PORTE_M = 2
/**
 * Nombre de VIES distinctes qui doivent emprunter le même vecteur pour qu'il soit tenu pour une
 * porte de la carte. Deux, parce qu'un joueur seul peut très bien se téléporter deux fois vers
 * la même faille — c'est la PLURALITÉ DES JOUEURS qui prouve que le passage appartient au
 * terrain et non à un équipement.
 */
const VIES_POUR_UNE_PORTE = 2

/** Tolérance à l'arrivée pour corroborer un passage par une faille du même joueur, en mètres. */
const ARRIVEE_M = 6

/** Un passage mesuré : qui, quand, d'où, vers où. */
export interface RiftTeleport {
  /** Slot de la vie qui franchit. */
  slot: number
  /** Frame d'ARRIVÉE — l'instant que le calque date pour effacer le lien. */
  frame: number
  /** Position quittée (dernier point avant le saut). */
  from: XY
  /** Position atteinte. */
  to: XY
  /**
   * Vrai quand une faille ACTIVE du même joueur se trouve à l'arrivée : le passage est alors
   * corroboré par un second canal. Faux ne veut pas dire « douteux », seulement « la pose n'a
   * pas été enregistrée » — cas de tout le corpus local.
   */
  viaRift: boolean
}

interface SautBrut {
  slot: number
  frame: number
  from: XY
  to: XY
  /** Vecteur du saut, ramené à un sens unique : une porte et son retour se répondent. */
  dx: number
  dy: number
}

function distance(a: XY, b: XY): number {
  return Math.hypot(a.x - b.x, a.y - b.y)
}

/**
 * Ramène un vecteur à un sens canonique (dx > 0, ou dy > 0 quand dx est nul), pour qu'un aller
 * et son retour tombent sur la même clé.
 */
function sensUnique(dx: number, dy: number): { dx: number; dy: number } {
  if (dx < 0 || (dx === 0 && dy < 0)) return { dx: -dx, dy: -dy }
  return { dx, dy }
}

/** Tous les sauts instantanés du film, sans jugement sur leur cause. */
function sautsBruts(lives: readonly ReplayTrackReady[]): SautBrut[] {
  const out: SautBrut[] = []
  for (const vie of lives) {
    const pts = vie.points
    for (let i = 1; i < pts.length; i++) {
      const a = pts[i - 1]
      const b = pts[i]
      if (b.t - a.t > SAUT_FRAMES_MAX) continue
      if (distance(a, b) < SAUT_M_MIN) continue
      const v = sensUnique(b.x - a.x, b.y - a.y)
      out.push({ slot: vie.slot, frame: b.t, from: { x: a.x, y: a.y }, to: { x: b.x, y: b.y }, ...v })
    }
  }
  return out
}

/**
 * estUnePorte — le saut emprunte-t-il un vecteur que PLUSIEURS VIES ont emprunté ?
 *
 * On compte les VIES et non les sauts : un même joueur qui passe quatre fois par la même porte
 * ne prouve rien de plus qu'un joueur qui se téléporte quatre fois vers sa faille. C'est le
 * partage entre joueurs qui distingue le terrain de l'équipement.
 */
function estUnePorte(saut: SautBrut, tous: readonly SautBrut[]): boolean {
  const vies = new Set<number>()
  for (const autre of tous) {
    if (Math.abs(autre.dx - saut.dx) > TOLERANCE_PORTE_M) continue
    if (Math.abs(autre.dy - saut.dy) > TOLERANCE_PORTE_M) continue
    vies.add(autre.slot)
  }
  return vies.size >= VIES_POUR_UNE_PORTE
}

/**
 * portaitLeTranslocateur — VERROU A : ce slot avait-il l'équipement en main à cet instant ?
 *
 * UNE LECTURE À VENIR NE PROUVE RIEN. `nearestReading` sait rendre la plus proche lecture
 * FUTURE quand une vie n'en a pas encore eu, et c'est le bon service pour afficher un
 * inventaire ; ici ce serait un contresens — « il portera le translocateur dans dix secondes »
 * ne dit pas qu'il le portait au saut. D'où l'exigence d'un âge positif ou nul.
 */
function portaitLeTranslocateur(
  saut: SautBrut,
  abilities: ReplayDocumentReady["abilities"],
): boolean {
  const lecture = nearestReading(abilities, saut.slot, saut.frame)
  return lecture !== null && lecture.age >= 0 && lecture.value.r === RANG_TRANSLOCATEUR
}

/**
 * riftTeleports — tous les passages du film, dans l'ordre des frames d'arrivée.
 *
 * Pure et sans état : le calque la fait passer par un mémo, jamais par image.
 */
export function riftTeleports(
  placements: readonly ReplayEquipmentPlacement[],
  lives: readonly ReplayTrackReady[],
  abilities: ReplayDocumentReady["abilities"],
): RiftTeleport[] {
  const bruts = sautsBruts(lives)
  if (bruts.length === 0) return []
  const failles = placements.filter((p) => p.family === FAMILLE_FAILLE)
  const out: RiftTeleport[] = []
  for (const s of bruts) {
    if (!portaitLeTranslocateur(s, abilities)) continue
    if (estUnePorte(s, bruts)) continue
    const corroboree = failles.some(
      (f) => f.owner === s.slot && f.t0 <= s.frame && s.frame <= f.t1 && distance(s.to, f) <= ARRIVEE_M,
    )
    out.push({ slot: s.slot, frame: s.frame, from: s.from, to: s.to, viaRift: corroboree })
  }
  out.sort((a, b) => a.frame - b.frame)
  return out
}
