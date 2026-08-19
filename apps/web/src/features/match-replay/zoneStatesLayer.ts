/**
 * zoneStatesLayer.ts — L'ÉTAT VIVANT DES ZONES (schémas 16-18) : qui tient quoi à l'image
 * courante, et la JAUGE DE CAPTURE qui se remplit sous les yeux.
 *
 * POURQUOI IL VIT À CÔTÉ DU CALQUE STATIQUE, ET PAS DEDANS. `objectivesLayer.ts` porte la
 * GÉOMÉTRIE des objectifs du mode : elle ne dépend ni de l'image ni de la lecture, se cuit une
 * fois hors écran et se recopie. Celui-ci porte l'ÉTAT, qui change à chaque image — même
 * partage que les socles d'arme, et la seule façon de garder les deux fichiers sous le seuil du
 * dépôt. Les deux tracent la MÊME forme : le contour vient de `traceZonePath`, jamais d'une
 * seconde copie de la géométrie.
 *
 * CE QUE LE CALQUE MONTRE : la zone TEINTÉE de l'encre du camp qui la tient, la colline ACTIVE
 * en surbrillance, et L'ARC DE LA JAUGE EN DIRECT (schéma 18) : la VALEUR de la jauge à l'image,
 * lue dans la série `gauge` de la zone en escalier — dernière valeur connue, tenue jusqu'au point
 * suivant (une seconde après le dernier de la série). Une zone sans état à cette frame n'est PAS
 * repeinte : elle garde le trait faible du calque statique, et paraît estompée sous celles qui
 * sont tenues.
 *
 * CE QUE LE CALQUE NE MONTRE PLUS (2026-08-18, lot C-ter volet 3) : le sommet `progress` de
 * l'intervalle. Le schéma 16 le traçait en arc, une valeur tenue pendant toute la durée de la
 * propriété — souvent 1,0, des minutes durant — et il se lisait comme « capture en cours »
 * alors qu'il n'en était que le maximum atteint. Sur un artefact qui ne porte pas `gauge`
 * (schéma <= 17), il n'y a donc PLUS D'ARC DU TOUT : mieux vaut rien qu'un signe qui ment.
 *
 * AUCUN TEXTE, comme le calque statique : la lettre A/B/C affichée en jeu n'existe dans aucune
 * donnée décodée, et le garde-fou du dossier l'interdit.
 */
import {
  traceZonePath,
  type CanvasView,
  type ObjectiveElementReady,
} from './objectivesLayer'
import { canvasScale, worldToCanvas, type XY } from './replayLogic'

import type { ReplayGaugePoint } from '@/lib/api/types'
import type { ReplayZoneStateReady } from './replayNormalize'

/**
 * ZONE_GAUGE_HOLD_MS — combien de temps le DERNIER point de la série reste affiché, en TEMPS RÉEL
 * (converti en frames par `useZoneStates`, une fois par document).
 *
 * ENTRE DEUX POINTS, LA VALEUR TIENT — quel que soit l'écart. La jauge du film ne redescend
 * jamais pas à pas (mesure du lot C-ter volet 3, `echelle_7344d24f.log`) : elle monte tant qu'on
 * capture, se TAIT tant que la capture est figée (zone contestée : 29 s à 0,92 sur `7344d24f`),
 * et revient à zéro d'une seule émission — que le producteur publie comme dernier point de la
 * rampe. Un silence entre deux points est donc une jauge FIGÉE, et l'escalier la garde à l'écran.
 * Seul le dernier point de la série n'a rien après lui pour dire ce qu'il devient : il tient une
 * seconde (la borne du producteur entre deux points d'une rampe), puis l'arc s'efface.
 */
export const ZONE_GAUGE_HOLD_MS = 1_000

/**
 * ZoneStateNow — ce qu'une zone montre à une frame donnée.
 *
 * `owner` vaut `null` quand PERSONNE ne la tient : c'est une mesure (la valeur neutre du canal
 * de propriété), pas une absence de donnée — d'où le champ, plutôt qu'un état omis. Le sommet
 * `progress` de l'intervalle n'y figure plus : le rendu ne le lit plus (cf. l'en-tête).
 */
export interface ZoneStateNow {
  owner: number | null
  active: boolean
}

/**
 * spanStateAt rend l'état porté par l'intervalle qui couvre la frame (bornes INCLUSES), ou `null`
 * quand aucun ne la couvre — avant la première émission du film, la zone n'a pas d'état CONNU, et
 * la dessiner « neutre » affirmerait quelque chose que l'artefact ne dit pas.
 *
 * FONCTION PURE, et la SEULE lecture d'état du calque : c'est elle que le rendu appelle à chaque
 * image, pour chaque zone (cf. drawZoneStates). Elle est privée parce que rien hors du calque ne
 * lit l'état d'une zone ; un jour où quelque chose le lira, c'est elle qu'on exportera — pas une
 * seconde façon de faire la même lecture.
 */
function spanStateAt(spans: ReplayZoneStateReady['spans'], frame: number): ZoneStateNow | null {
  for (const sp of spans) {
    if (frame < sp.t0 || frame > sp.t1) continue
    return { owner: sp.owner ?? null, active: sp.active }
  }
  return null
}

/**
 * zoneGaugeAt rend la VALEUR de la jauge à la frame demandée, lue EN ESCALIER dans la série
 * publiée : la dernière valeur dont l'instant est <= frame, tenue jusqu'au point suivant. Rend
 * `null` AVANT le premier point, et une fois le DERNIER point de la série plus vieux que
 * `holdFrames` (une seconde : la borne du producteur entre deux points d'une rampe — au-delà, plus
 * rien ne viendra dire ce que la jauge est devenue).
 *
 * FONCTION PURE, SANS INTERPOLATION LINÉAIRE ET SANS EXPIRATION ENTRE DEUX POINTS : entre deux
 * points la vraie jauge a bougé de moins de 0,02, ou n'a pas bougé du tout (capture figée) —
 * inventer une pente lisserait ce que le producteur a quantifié, effacer l'arc cacherait un
 * blocage que le film montre. Une jauge qui retombe à zéro le dit par un point (cf.
 * ZONE_GAUGE_HOLD_MS).
 */
export function zoneGaugeAt(
  gauge: readonly ReplayGaugePoint[],
  frame: number,
  holdFrames: number,
): number | null {
  let lo = 0
  let hi = gauge.length - 1
  let idx = -1
  while (lo <= hi) {
    const mid = (lo + hi) >> 1
    if (gauge[mid].t <= frame) {
      idx = mid
      lo = mid + 1
    } else {
      hi = mid - 1
    }
  }
  if (idx < 0) return null
  const p = gauge[idx]
  const last = idx === gauge.length - 1
  return last && frame - p.t > holdFrames ? null : p.v
}

/**
 * zoneElementsOf rend les éléments SURFACIQUES dans l'ordre servi — celui que `zoneRef` indexe.
 *
 * L'ORDRE EST LE CONTRAT : `normalizeMapObjectives` empile les zones puis les marqueurs, en
 * conservant l'ordre du serveur, et `zoneStates[].zoneRef` est l'index dans `mapObjectives.zones`.
 * Filtrer sur `kind` restitue donc exactement cette liste.
 */
export function zoneElementsOf(
  elements: readonly ObjectiveElementReady[],
): ObjectiveElementReady[] {
  return elements.filter((e) => e.kind === 'zone')
}

/**
 * zoneCatalogMatches dit si l'index `zoneRef` de l'artefact est JOIGNABLE à la liste servie.
 *
 * POURQUOI CETTE VÉRIFICATION EXISTE (revue R1, 2026-08-18). `zoneRef` est un INDEX dans
 * `mapObjectives.zones` — figé à la CUISSON de l'artefact — alors que `mapObjectives` est
 * reconstruit À CHAQUE REQUÊTE par le service, depuis le catalogue de formes de la carte et la
 * table de rôles du titre. Ces deux listes ne sont les mêmes que tant que ni le catalogue ni la
 * table ne bougent. Le jour où l'un des deux change sans que les artefacts soient recuits, la
 * teinte du camp se poserait sur une AUTRE zone : une erreur invisible et parfaitement crédible.
 *
 * `coverage.zones.catalog` est le nombre de zones que l'artefact avait sous les yeux. Il ne
 * prouve pas que les listes coïncident — deux listes différentes peuvent avoir la même
 * longueur — mais il attrape le cas qui arrive vraiment : une zone ajoutée ou retirée du
 * catalogue. Dans le doute, le calque vivant se TAIT et le calque statique reste seul.
 *
 * UN CATALOGUE ABSENT NE PASSE PAS : un artefact qui publie des états sans publier sa couverture
 * ne permet aucune vérification, et « pas vérifiable » se traite comme « pas joignable ».
 */
export function zoneCatalogMatches(catalog: number | null | undefined, served: number): boolean {
  return catalog != null && catalog === served
}

/** Style du calque VIVANT : les encres sont RÉSOLUES par l'appelant (règle color-tokens). */
export interface ZoneStateStyle {
  /** Encre d'un camp ; `null` = camp inconnu (aucune ligne « moi ») — le liseré reste neutre. */
  colorOfOwner: (team: number) => string | null
  /**
   * Encre du camp QUI CAPTURE une zone tenue par `owner` : le camp d'en face. `null` = camp
   * inconnu (aucune ligne « moi ») — l'arc reste neutre.
   *
   * POURQUOI C'EST UNE DÉDUCTION ET PAS UNE INVENTION. Le film ne dit pas qui pousse la jauge
   * (les slots de rampe ne portent aucun propriétaire — mesure du lot C-bis) ; mais dans un mode
   * à zones à DEUX camps, une zone TENUE ne se capture que par l'adversaire : le sien n'a rien à
   * y capturer. Une zone que PERSONNE ne tient, elle, se prend par n'importe qui — l'arc y reste
   * neutre, comme sur une colline de KOTH dont le propriétaire n'est jamais publié.
   */
  colorOfCapturer: (owner: number) => string | null
  /** Encre neutre : zone que personne ne tient, et arc de jauge sans camp connu. */
  neutral: string
}

/**
 * ZoneStatesLayerInput — ce que le calque vivant reçoit de l'appelant (`useZoneStates`) : les
 * zones dans l'ordre servi, le verdict de jointure, les encres résolues, la tenue de la jauge.
 */
export interface ZoneStatesLayerInput {
  /** Les zones SURFACIQUES dans l'ordre servi : celui que `zoneStates[].zoneRef` indexe. */
  zoneElements: readonly ObjectiveElementReady[]
  /**
   * `zoneRef` est-il joignable à `zoneElements` (cf. zoneCatalogMatches) ? Faux : le calque ne
   * peint RIEN — l'index de l'artefact est figé à la cuisson, la liste servie est reconstruite à
   * la requête, et teinter la mauvaise zone serait une erreur invisible.
   */
  joinable: boolean
  style: ZoneStateStyle
  /** ZONE_GAUGE_HOLD_MS converti en frames pour ce document (cf. zoneGaugeAt). */
  gaugeHoldFrames: number
}

// Réglages du calque vivant. Plus francs que le calque statique (qui reste dessous) : c'est le
// contraste entre les deux qui fait lire la bascule, et les zones sans état courant gardent
// leur trait faible — elles paraissent estompées sans qu'on ait à les repeindre.
const ZONE_HELD_FILL_ALPHA = 0.22
const ZONE_HELD_STROKE_ALPHA = 0.95
const ZONE_HELD_STROKE_WIDTH = 2.5
const ZONE_ACTIVE_FILL_ALPHA = 0.3
const ZONE_ACTIVE_STROKE_WIDTH = 3.5
const ZONE_GAUGE_ALPHA = 0.9
const ZONE_GAUGE_WIDTH = 3
const ZONE_GAUGE_MIN_RADIUS = 10

/**
 * drawZoneStates peint, PAR-DESSUS le calque statique, ce que le film dit de chaque zone à
 * l'image courante : teinte du propriétaire, surbrillance de la zone ACTIVE, et l'arc de la
 * JAUGE EN DIRECT quand la série en porte une valeur à cette frame.
 *
 * CE CALQUE N'EST PAS STATIQUE, et c'est tout le point : sa géométrie ne bouge pas, son ÉTAT
 * change à chaque image — comme les socles d'arme. Il se peint donc dans la boucle, pas dans un
 * canvas cuit une fois.
 *
 * L'ARC NE DÉPEND PAS DE L'INTERVALLE COURANT : la jauge est une mesure de la ZONE, publiée par
 * frame ; une rampe qui précède la première émission du canal de propriété se dessine quand
 * même — à l'encre neutre, faute de savoir qui tient la zone. Quand un intervalle la couvre et
 * que son propriétaire est connu, l'arc prend l'encre du camp d'en face : celui qui capture.
 *
 * AUCUN TEXTE, comme le calque statique : la lettre A/B/C affichée en jeu n'existe dans aucune
 * donnée décodée, et le garde-fou du fichier l'interdit.
 *
 * LA GARDE DE JOINTURE EST ICI, dans le calque, et pas seulement chez l'appelant : quand
 * `zones.joinable` est faux, la fonction rend AVANT le premier trait. C'est le calque qui
 * refuse de peindre — un appelant ne peut pas l'oublier, et le test le vérifie sur le contexte.
 */
export function drawZoneStates(
  ctx: CanvasRenderingContext2D,
  zones: ZoneStatesLayerInput,
  states: readonly ReplayZoneStateReady[],
  view: CanvasView,
  frame: number,
): void {
  if (!zones.joinable || states.length === 0) return
  const { style } = zones
  const px = (p: XY) => worldToCanvas(p, view.bounds, view.width, view.height, view.pad)
  const scale = canvasScale(view.bounds, view.width, view.height, view.pad)
  zones.zoneElements.forEach((e, ref) => {
    const st = states.find((s) => s.zoneRef === ref)
    if (!st) return
    const now = spanStateAt(st.spans, frame)
    const ownerInk = now && now.owner !== null ? style.colorOfOwner(now.owner) : null
    if (now) {
      paintZoneState(ctx, e, { px, scale, ink: ownerInk ?? style.neutral, held: ownerInk !== null, now })
    }
    // Une jauge à ZÉRO (au repos, ou revenue à zéro) n'a pas d'arc : un arc d'angle nul ne trace
    // rien, autant ne pas l'émettre.
    const value = zoneGaugeAt(st.gauge, frame, zones.gaugeHoldFrames)
    if (value !== null && value > 0) {
      const capturer = now && now.owner !== null ? style.colorOfCapturer(now.owner) : null
      drawGaugeArc(ctx, e, { px, scale, value, ink: capturer ?? style.neutral })
    }
  })
  ctx.globalAlpha = 1
}

/** Ce que le tracé d'une zone vivante a besoin de savoir (règle des 5 paramètres). */
interface ZonePaint {
  px: (p: XY) => XY
  scale: number
  ink: string
  /** `false` = personne ne tient la zone : LISERÉ SEUL, aucun remplissage. */
  held: boolean
  now: ZoneStateNow
}

/** Zone tenue : remplissage + liseré à l'encre du camp. Zone active : les deux, renforcés. */
function paintZoneState(
  ctx: CanvasRenderingContext2D,
  e: ObjectiveElementReady,
  p: ZonePaint,
): void {
  traceZonePath(ctx, e, p.px, p.scale)
  if (p.held || p.now.active) {
    ctx.globalAlpha = p.now.active ? ZONE_ACTIVE_FILL_ALPHA : ZONE_HELD_FILL_ALPHA
    ctx.fillStyle = p.ink
    ctx.fill()
  }
  ctx.globalAlpha = ZONE_HELD_STROKE_ALPHA
  ctx.strokeStyle = p.ink
  ctx.lineWidth = p.now.active ? ZONE_ACTIVE_STROKE_WIDTH : ZONE_HELD_STROKE_WIDTH
  ctx.stroke()
}

/** Ce que l'arc de jauge a besoin de savoir (règle des 5 paramètres — regroupé le 2026-08-18). */
interface ZoneGaugeArc {
  px: (p: XY) => XY
  scale: number
  /** La valeur de la jauge à l'image, dans [0, 1] (cf. zoneGaugeAt). */
  value: number
  ink: string
}

/**
 * drawGaugeArc dessine LA JAUGE EN DIRECT : un arc qui part du haut et se referme à mesure que la
 * capture avance — à la valeur lue à cette image, jamais au sommet de l'intervalle. Il est tracé
 * HORS de la forme (rayon un peu plus grand) pour rester lisible sur une zone déjà teintée.
 */
function drawGaugeArc(ctx: CanvasRenderingContext2D, e: ObjectiveElementReady, a: ZoneGaugeArc): void {
  const c = a.px(e)
  const half = e.family === 'cylinder' ? e.radius : Math.max(e.halfX, e.halfY)
  const r = Math.max(half * a.scale + ZONE_GAUGE_WIDTH, ZONE_GAUGE_MIN_RADIUS)
  const start = -Math.PI / 2
  ctx.globalAlpha = ZONE_GAUGE_ALPHA
  ctx.strokeStyle = a.ink
  ctx.lineWidth = ZONE_GAUGE_WIDTH
  ctx.beginPath()
  ctx.arc(c.x, c.y, r, start, start + 2 * Math.PI * Math.min(Math.max(a.value, 0), 1))
  ctx.stroke()
}
