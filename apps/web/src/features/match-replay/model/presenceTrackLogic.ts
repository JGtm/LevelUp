/**
 * presenceTrackLogic — QUAND LE JOUEUR N'ÉTAIT PAS LÀ, en ratios de frise (2026-09-07, lot L4).
 *
 * # CE QUE CE MODULE RÉPARE
 *
 * La piste du joueur regardé montre ses éliminations et ses morts. Un remplaçant arrivé à la
 * huitième minute y a donc une première moitié VIDE — exactement le dessin qu'aurait un joueur
 * présent qui n'aurait rien fait. Deux faits opposés, un seul rendu : l'ombrage sépare « il
 * n'était pas là » de « il n'a rien fait », et c'est aussi ce qui explique une dominance qui
 * s'effondre quand l'effectif passe de quatre à trois.
 *
 * # AUCUN NOUVEAU CALCUL DE TEMPS, AUCUN NOUVEAU SEUIL
 *
 * Les instants viennent des `PresenceEvent` DÉJÀ fusionnés au fil (`presenceFeed.ts` →
 * `mergeFeedWithPresence`), déjà recalés sur l'axe du rejeu, et ils se posent avec la MÊME
 * fonction que les marques (`ratioOfMs`). Les marges du repli film (10 s à l'arrivée, 20 s au
 * départ) vivent dans `presenceFeed.ts` et n'ont rien à faire ici : ce module ne DÉCIDE de rien,
 * il met en ratios ce que le fil affirme déjà.
 *
 * # LA SOURCE VOYAGE AVEC L'OMBRE
 *
 * `source: 'api' | 'film'` est propagé tel quel jusqu'au rendu : le chemin API date l'entrée à la
 * seconde (horodatage de participation), le repli film la DÉDUIT d'une borne de vie à 10-20 s
 * près et ne distingue pas un départ d'une élimination définitive. Une frontière au pixel sur une
 * déduction serait un mensonge de précision — le rendu dégrade son bord, ce module lui en donne
 * les moyens.
 *
 * # PURETÉ
 *
 * Aucun JSX, aucun token de couleur, aucune hauteur : des ratios et des comptes. Ce qui se décide
 * ici se teste sans monter un composant (`presenceTrackLogic.test.ts`).
 */
import type { ReplayFeedEntry } from './killFeedLogic'
import type { PresenceEvent } from './presenceFeed'
import { ratioOfMs, type TrackScale } from './replayTimelineTracksLogic'

/**
 * UN INTERVALLE OMBRÉ de la piste d'un joueur : le temps de match qu'il n'a pas joué.
 *
 * DEUX FORMES ET DEUX SEULEMENT — une ombre de TÊTE `[0, t]` pour qui rejoint en cours, une
 * ombre de QUEUE `[t, 1]` pour qui s'en va. Un joueur sans événement de présence n'a AUCUNE
 * ombre : il était là du coup d'envoi à la fin, et c'est le cas nominal. Les deux à la fois
 * (arrivé PUIS parti) donnent deux ombres, chacune avec sa frontière et son glyphe.
 */
export interface PresenceShade {
  key: string
  /** Bord gauche de l'ombre, en ratio de frise. */
  from: number
  /** Bord droit de l'ombre, en ratio de frise. */
  to: number
  kind: PresenceEvent['kind']
  /** `api` = horodatage de participation (bord franc) ; `film` = déduit (bord dégradé). */
  source: PresenceEvent['source']
  /**
   * LE BORD QUI TOUCHE LA ZONE JOUÉE, et c'est là que le glyphe se pose (décision 2 bis du plan) :
   * bord DROIT de l'ombre de tête pour un arrivant, bord GAUCHE de l'ombre de queue pour un
   * partant. Redondant avec `from`/`to` par construction, nommé quand même — le rendu ne doit pas
   * avoir à re-choisir lequel des deux selon le sens, c'est exactement là qu'une inversion se
   * glisserait sans qu'aucun test ne rougisse.
   */
  edge: number
  /** L'IMAGE du document où l'événement a lieu : ce que le clic sur le glyphe demande au curseur. */
  frame: number
  /** Instant mis en forme (mm:ss de l'horloge de gameplay), pour le nom accessible du glyphe. */
  clock: string
  xuid: string
}

/**
 * presenceShades — les intervalles ombrés de la piste d'UN joueur.
 *
 * `xuid` à `null` (aucun point de vue résolu) rend une liste vide : ombrer une piste dont on ne
 * sait pas de qui elle parle poserait une affirmation sur personne.
 *
 * SANS HORLOGE ÉTABLIE, LE FIL NE PORTE AUCUNE LIGNE DE PRÉSENCE (`presenceEntries` rend `[]`
 * quand `!playWindow || !clock`) : cette fonction rend alors naturellement une liste vide, et
 * l'ombrage est ABSENT plutôt que FAUX. C'est voulu — cf. le commentaire de la piste appelante.
 *
 * Une ombre de largeur nulle est écartée : un joueur dont l'API date l'arrivée au coup d'envoi
 * exact (ou le départ à la dernière image) n'a rien à ombrer, et une bande invisible n'a rien à
 * faire dans une liste que le rendu parcourt.
 */
export function presenceShades(
  entries: readonly ReplayFeedEntry[],
  xuid: string | null,
  frameIntervalMs: number,
  scale: TrackScale,
  clockOf: (replayMs: number) => string,
): PresenceShade[] {
  const out: PresenceShade[] = []
  if (xuid == null || scale.span <= 0 || !frameIntervalMs) return out
  for (const entry of entries) {
    const p = entry.presence
    if (!p || p.xuid !== xuid) continue
    const edge = clampRatio(ratioOfMs(entry.replayMs, frameIntervalMs, scale))
    const from = p.kind === 'joined' ? 0 : edge
    const to = p.kind === 'joined' ? edge : 1
    if (to <= from) continue
    out.push({
      key: entry.key,
      from,
      to,
      kind: p.kind,
      source: p.source,
      edge,
      frame: Math.round(entry.replayMs / frameIntervalMs),
      clock: clockOf(entry.replayMs),
      xuid: p.xuid,
    })
  }
  return out
}

/**
 * UN PALIER D'ABSENCE de la piste des coéquipiers : sur `[from, to]`, `absent` d'entre eux sur
 * `total` manquaient. C'est la définition de l'ombrage « agrégé » que le plan laissait ouverte
 * (précisée en revue le 2026-09-06) : l'opacité de l'ombre est la PART DE L'EFFECTIF ABSENTE,
 * par paliers — un absent sur trois donne un tiers.
 */
export interface AbsenceStep {
  key: string
  from: number
  to: number
  /** Combien de coéquipiers manquaient sur cet intervalle. Jamais 0 (cf. `teammatesAbsence`). */
  absent: number
  /** L'effectif de référence : le nombre de coéquipiers du point de vue, lui-même exclu. */
  total: number
}

/**
 * teammatesAbsence — la piste des coéquipiers, en PALIERS d'effectif manquant.
 *
 * POURQUOI DES PALIERS ET NON DES GLYPHES (décision du plan, item « aucun glyphe sur la piste
 * Coéquipiers ») : à quatre coéquipiers, quatre portes empilées sur quatorze pixels ne se lisent
 * pas, et la question que cette piste pose n'est pas « qui est parti » mais « combien
 * manquaient ». Le fil, lui, nomme chacun.
 *
 * L'ALGORITHME EST UNE FONCTION EN ESCALIER, pas une somme d'intervalles : on découpe la frise
 * aux frontières de TOUTES les absences, on compte les absents au MILIEU de chaque tranche (le
 * milieu évite d'avoir à décider si une borne appartient à l'intervalle qu'elle ouvre ou à celui
 * qu'elle ferme), puis on fond les tranches voisines de même compte. Les tranches à ZÉRO absent
 * sortent de la liste : l'effectif complet n'est pas un état à peindre, c'est le cas normal.
 *
 * `total` est l'effectif de RÉFÉRENCE et il ne bouge pas d'un palier à l'autre : la part
 * manquante se lit toujours sur le même dénominateur, sans quoi « deux tiers » puis « un demi »
 * décriraient le même effectif avec deux échelles.
 */
export function teammatesAbsence(
  entries: readonly ReplayFeedEntry[],
  teammates: readonly string[],
  frameIntervalMs: number,
  scale: TrackScale,
): AbsenceStep[] {
  const total = teammates.length
  if (total === 0 || scale.span <= 0 || !frameIntervalMs) return []
  const concernes = new Set(teammates)
  const absences: { from: number; to: number }[] = []
  for (const entry of entries) {
    const p = entry.presence
    if (!p || !concernes.has(p.xuid)) continue
    const edge = clampRatio(ratioOfMs(entry.replayMs, frameIntervalMs, scale))
    const from = p.kind === 'joined' ? 0 : edge
    const to = p.kind === 'joined' ? edge : 1
    if (to > from) absences.push({ from, to })
  }
  if (absences.length === 0) return []
  return fondrePaliers(decouper(absences), absences, total)
}

/** Les frontières de la frise, dédoublonnées et triées : les bords de tranche de l'escalier. */
function decouper(absences: readonly { from: number; to: number }[]): number[] {
  const bornes = new Set<number>([0, 1])
  for (const a of absences) {
    bornes.add(a.from)
    bornes.add(a.to)
  }
  return [...bornes].sort((x, y) => x - y)
}

/**
 * Compte les absents tranche par tranche et fond les voisines de même compte. Les tranches à
 * zéro ferment le palier courant sans en ouvrir un : elles ne sont jamais émises.
 */
function fondrePaliers(
  bornes: readonly number[],
  absences: readonly { from: number; to: number }[],
  total: number,
): AbsenceStep[] {
  const out: AbsenceStep[] = []
  for (let i = 0; i + 1 < bornes.length; i += 1) {
    const from = bornes[i]
    const to = bornes[i + 1]
    if (to <= from) continue
    const milieu = (from + to) / 2
    const absent = absences.filter((a) => a.from < milieu && milieu < a.to).length
    if (absent === 0) continue
    const dernier = out[out.length - 1]
    // Deux tranches voisines de même compte sont UN palier : la frontière qui les sépare est
    // celle d'un autre joueur, dont l'absence commence là où une autre s'achève.
    if (dernier && dernier.absent === absent && dernier.to === from) {
      dernier.to = to
      continue
    }
    out.push({ key: `a${from}-${to}`, from, to, absent, total })
  }
  return out
}

/** Un ratio rabattu sur la frise : une présence hors fenêtre s'ombre jusqu'au bord, pas au-delà. */
function clampRatio(ratio: number): number {
  return Math.min(1, Math.max(0, ratio))
}
