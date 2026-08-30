/**
 * zoneSound.ts — LES SONS QUI VIENNENT DE L'ÉTAT DES ZONES, et non d'une action de joueur.
 *
 * POURQUOI UN FICHIER À PART. `objectiveSound.ts` joint sur `doc.objectives`, c'est-à-dire sur
 * les ÉVÉNEMENTS DE STATISTIQUE d'un joueur nommé (« untel a capturé »). Les trois sons d'ici
 * ne viennent d'aucun joueur : ils viennent de l'ÉTAT DE LA ZONE, publié par `doc.zoneStates`
 * — la jauge de capture, le propriétaire, et la bascule de colline. Source différente, clé de
 * jointure différente, doctrine différente : fichier différent.
 *
 * ET C'ÉTAIT LA SEULE RAISON POUR LAQUELLE CES SONS NE SONNAIENT PAS. Ils sont extraits du jeu
 * et nommés depuis le 2026-08-27 ; le moteur, lui, ne lisait que `doc.objectives`. Rien ne
 * manquait côté Go — le calque des zones est publié depuis le schéma 18.
 *
 * ## Les trois règles, et d'où chacune tient
 *
 * **CAPTURE EN COURS.** La jauge ne publie que les RAMPES (montées monotones), fermées par un
 * retour à zéro quand le film le porte — c'est écrit dans `ZoneState.Gauge` côté Go. Une rampe
 * dit qu'une capture est en cours ; elle ne dit pas QUI capture. Le camp se lit à l'arrivée :
 * si la rampe se termine sur un CHANGEMENT DE PROPRIÉTAIRE, le capteur est le nouveau
 * propriétaire. Une rampe qui retombe sans changer le propriétaire reste MUETTE — c'est
 * exactement le cas « contestée », et nous ne savons pas encore nommer celui qui a échoué.
 *
 * **TIC DE SCORE.** Règle produit de l'utilisateur (2026-08-27), et elle n'est pas une mesure :
 * « on pourrait les jouer chaque seconde selon l'équipe, quand l'équipe alliée ou adverse a
 * toutes les zones ». C'est donc UN TIC PAR SECONDE tant qu'un camp tient TOUTES les zones.
 * RESTREINT AUX MODES À ZONES SIMULTANÉES : le son s'appelle `..._strongholds_scoring_tick_*`
 * et l'utilisateur a précisé qu'il parlait de Bastion. Une colline unique (KOTH) tenue en
 * permanence ferait sonner un tic par seconde tout le match — la garde est donc double : au
 * moins deux zones, et aucun intervalle marqué `active` (le marqueur de colline).
 *
 * **NOUVELLE COLLINE.** Chaque début d'intervalle `active` SAUF LE PREMIER : le son est
 * « avant l'apparition d'une NOUVELLE zone », pas l'ouverture du match. Il n'a pas de camp —
 * la colline n'appartient à personne quand elle se déplace.
 *
 * **SÉCURISATION DE LA COLLINE** (2026-08-30). En Roi de la colline il n'y a pas de capture :
 * la colline se prend instantanément et c'est la GARDE qui marque. Le déclencheur est donc
 * l'intervalle `active` POSSÉDÉ, pas une rampe de jauge — laquelle n'existe jamais sur une
 * colline (`ZoneState.Gauge` côté Go). Détail et plancher : `ZONE_SECURING_MIN_MS`.
 *
 * ## Sans camp allié, trois des quatre se taisent
 *
 * Même règle que partout dans la chaîne sonore : sans ligne « moi » au tableau de score, le
 * rejeu ne devine pas un camp. La capture en cours et les tics se taisent ; la nouvelle
 * colline sonne quand même, elle n'affirme rien.
 */
import type { ReplayDocumentReady } from './replayNormalize'
import { frameToMs, msToFrames } from './replayLogic'
import { soundEvent, type ReplaySoundEvent } from './replaySoundVariants'

/** Le camp d'un état de zone, vu de la page — la même notion que dans `objectiveSound.ts`. */
export type ZoneSide = 'ally' | 'enemy'

/**
 * ZONE_SOUND_STEMS — les fichiers, par geste et par camp.
 *
 * PROVENANCE. Banque `1c609526` (les modes à zones ; son nom de fichier n'est pas cassé). Deux
 * gestes portent un nom Wwise cassé par hachage (`..._strongholds_scoring_tick_team` et
 * `_enemy`), les quatre autres sont DÉSIGNÉS À L'OREILLE par l'utilisateur — une désignation
 * vaut une mesure pour l'usage produit (`RECETTE_SONS_ARMES` §5).
 *
 * LES DEUX PAIRES SONT COMPLÈTES, et c'est neuf : jusqu'au 2026-08-27 le côté adverse de la
 * capture était MUET faute de son désigné.
 */
export const ZONE_SOUND_STEMS = {
  /** La jauge monte : quelqu'un est en train de prendre la zone. Rendu en boucle, 3 s. */
  capturing: {
    ally: 'objective_zone_capturing_team',
    enemy: 'objective_zone_capturing_enemy',
  },
  /** Un camp tient toutes les zones : un tic par seconde. Rendu court (1,2 s) et à -12 dBTP. */
  tick: {
    ally: 'objective_zone_tick_team',
    enemy: 'objective_zone_tick_enemy',
  },
  /**
   * La zone est CONTESTÉE. Définition donnée par l'utilisateur le 2026-08-27, mot pour mot :
   * « c'est quand on va prendre une zone adverse et qu'un adversaire entre dans la zone pour la
   * contester ». Ce que le film en montre : une RAMPE de jauge qui retombe SANS que le
   * propriétaire change — la capture était en cours, elle a été interrompue.
   *
   * AUCUN CAMP, et ce n'est pas une lacune : le jeu n'a qu'UN son de contestation
   * (`play_004_mod_mp_strongholds_contested`, sans jumeau `_team` / `_enemy`), exactement comme
   * le retour de drapeau. Il sonne pareil pour tout le monde.
   */
  contested: 'objective_zone_contested',
  /** La colline se déplace (Roi de la colline). Aucun camp. */
  newZone: 'objective_zone_new',
  /**
   * LA SÉCURISATION DE LA COLLINE, en Roi de la colline. Désigné à l'oreille par l'utilisateur
   * le 2026-08-30 (`93f632c0` allié, `dcf980a5` adverse — deux gestes de forme identique et de
   * médias disjoints, la signature d'un couple `_team`/`_enemy`).
   *
   * POURQUOI CE N'EST PAS `capturing`, ET POURQUOI LES DEUX NE SE MARCHENT PAS DESSUS. En Roi
   * de la colline il n'y a pas de capture : la colline se prend instantanément et c'est la
   * GARDE qui marque (`.ai/V7.5/PLAN_KOTH_GARDE_VIVANTE_2026-08-30.md` § 1.1). Le Go le dit
   * dans le même sens et c'est ce qui rend les deux règles disjointes SANS garde explicite :
   * `ZoneState.Gauge` est TOUJOURS ABSENTE sur une colline (`document_zones.go` — le canal y
   * est un compteur de transfert d'environ une seconde, `coverage.zones.gaugePoints` vaut 0).
   * `capturing` naît d'une rampe de jauge, elle ne peut donc pas se déclencher en KOTH ; la
   * sécurisation naît d'un intervalle `active` possédé, qui n'existe QUE là.
   *
   * LE SON EST SERVI LONG (5,5 s) ET JOUÉ UNE SEULE FOIS. C'est une règle de l'utilisateur, et
   * elle vient de ce qu'il entend : « le son est à prolonger le temps que la sécurisation est
   * en cours ; une boucle relancerait le début du son, qui met du temps à se lancer — c'est
   * comme une sirène ». Le fichier porte donc la boucle DU JEU (une lecture de 1,3 s toutes les
   * 1,8 s, `sLoopCount` = 0, `eTransitionMode` = 3) servie sur 5,5 s ; le lecteur, lui, ne le
   * redéclenche jamais.
   *
   * DEUX ÉCARTS ASSUMÉS, écrits plutôt que masqués. (1) Le délai d'action de 1,5 s est RETIRÉ :
   * il est l'entrée en boucle du jeu, et servi en tête d'un one-shot il n'ajouterait que du
   * silence. (2) 5,5 s ne couvre pas une garde de 40 s ; c'est le plafond des sons d'ÉVÉNEMENT
   * (`LONG_MAX_S` = 6 s dans le garde-rail d'assets), et le tenir vaut mieux que de laisser un
   * son de match grimper vers le plafond des fanfares. Prolonger davantage demanderait de
   * relever ce plafond — décision produit, pas décision de livraison.
   */
  securing: {
    ally: 'objective_zone_securing_team',
    enemy: 'objective_zone_securing_enemy',
  },
} as const

/** Période des tics de score, en millisecondes — la seconde demandée par l'utilisateur. */
export const ZONE_TICK_PERIOD_MS = 1000

/**
 * PLAFOND DE TICS PAR INTERVALLE. Une manche entière tenue par un camp produirait des
 * centaines de tics ; le plafond de voix du lecteur (8) les avalerait, mais la piste porterait
 * quand même leur poids. 180 tics = trois minutes de domination continue, au-delà desquelles
 * un tic de plus n'apprend rien.
 */
export const ZONE_TICK_MAX_PAR_INTERVALLE = 180

/**
 * Les trois formes lues, réduites à ce dont ce fichier a besoin (typage STRUCTUREL : il n'a pas
 * à connaître le DTO complet, et le test peut donc construire un état de zone en trois lignes).
 */
interface SpanLike {
  t0: number
  t1: number
  owner?: number | null
  active?: boolean
}

interface GaugeLike {
  t: number
  v: number
}

interface ZoneStateLike {
  spans: readonly SpanLike[]
  gauge?: readonly GaugeLike[] | null
}

/**
 * zoneSoundEvents — les sons d'état de zone, posés sur l'horloge du rejeu.
 *
 * `allyTeam` est l'identifiant d'équipe alliée (celui de la ligne « moi » du tableau de score),
 * ou `null` quand il n'est pas résolu.
 */
export function zoneSoundEvents(
  doc: ReplayDocumentReady,
  allyTeam: number | null,
): ReplaySoundEvent[] {
  const zones: readonly ZoneStateLike[] = doc.zoneStates ?? []
  if (zones.length === 0) return []
  const out: ReplaySoundEvent[] = []
  const cote = (owner: number | null | undefined): ZoneSide | undefined => {
    if (allyTeam === null || owner === null || owner === undefined) return undefined
    return owner === allyTeam ? 'ally' : 'enemy'
  }

  for (const z of zones) {
    for (const r of rampesDeJauge(z.gauge ?? [])) {
      const arrivee = proprietaireApres(z.spans, r.fin)
      if (arrivee === undefined) {
        // LA RAMPE N'A RIEN CHANGÉ : la capture était en cours et a été interrompue. C'est la
        // CONTESTATION, et elle sonne à l'instant où la jauge cesse de monter — pas au début de
        // la rampe, où rien n'était encore contesté. Aucun camp : le jeu n'en a qu'un son.
        out.push(soundEvent(frameToMs(r.fin, doc), ZONE_SOUND_STEMS.contested))
        continue
      }
      const c = cote(arrivee)
      if (!c) continue
      out.push(soundEvent(frameToMs(r.debut, doc), ZONE_SOUND_STEMS.capturing[c]))
    }
  }
  out.push(...ticsDeDomination(zones, allyTeam, doc))
  out.push(...collinesSuivantes(zones, doc))
  out.push(...securisationsDeColline(zones, allyTeam, doc))
  return out.sort((a, b) => a.ms - b.ms)
}

/**
 * ZONE_SECURING_MIN_MS — en dessous, l'intervalle n'est pas une sécurisation mais un TRANSFERT.
 *
 * Le canal publie le neutre autant que le tenu (50 intervalles neutres pour 50 tenus sur
 * `01e1f945`), et une colline change de mains plusieurs fois par période. Sans plancher, chaque
 * passage d'un pied dans la colline déclencherait une sirène de 11,5 s. Le plancher est calé sur
 * le son lui-même : sa première lecture dure 1,3 s après un délai d'action de 1,5 s, donc en
 * dessous de trois secondes on lancerait un geste que l'intervalle ne laisse pas s'installer.
 *
 * Les gardes qui marquent, elles, durent 36,8 à 50,3 s (mesure sur 11 périodes de 3 films,
 * `PLAN_KOTH_GARDE_VIVANTE_2026-08-30.md` § 1.1) : le plancher ne les touche pas.
 */
export const ZONE_SECURING_MIN_MS = 3000

/**
 * securisationsDeColline — la sirène de garde, une fois par intervalle de possession de la
 * colline ACTIVE.
 *
 * Le déclencheur est `ZoneSpan.active` + `owner` : c'est le seul canal qui parle en Roi de la
 * colline, la jauge y étant toujours absente. Un intervalle neutre (`owner` nul) ne sonne pas —
 * personne ne sécurise. Sans camp allié résolu, tout se tait : même règle que partout ailleurs
 * dans cette chaîne, le rejeu ne devine pas un camp.
 */
function securisationsDeColline(
  zones: readonly ZoneStateLike[],
  allyTeam: number | null,
  doc: ReplayDocumentReady,
): ReplaySoundEvent[] {
  if (allyTeam === null) return []
  const minFrames = msToFrames(ZONE_SECURING_MIN_MS, doc)
  const out: ReplaySoundEvent[] = []
  for (const z of zones) {
    for (const s of z.spans) {
      if (!s.active) continue
      if (s.owner === null || s.owner === undefined) continue
      if (s.t1 - s.t0 < minFrames) continue
      const c: ZoneSide = s.owner === allyTeam ? 'ally' : 'enemy'
      out.push(soundEvent(frameToMs(s.t0, doc), ZONE_SOUND_STEMS.securing[c]))
    }
  }
  return out
}

/**
 * rampesDeJauge découpe la série en montées monotones. Une valeur qui n'augmente pas ferme la
 * rampe en cours — c'est le retour à zéro que le Go publie en fin de rampe, ou simplement le
 * point suivant d'une autre rampe.
 *
 * Une rampe d'un seul point n'en est pas une : le geste cherché est une capture qui DURE.
 */
function rampesDeJauge(g: readonly GaugeLike[]): { debut: number; fin: number }[] {
  const out: { debut: number; fin: number }[] = []
  let i = 0
  while (i < g.length) {
    let j = i
    while (j + 1 < g.length && g[j + 1].v > g[j].v) j++
    if (j > i) out.push({ debut: g[i].t, fin: g[j].t })
    i = j + 1
  }
  return out
}

/**
 * proprietaireApres rend le propriétaire de l'intervalle qui COMMENCE au plus près après la fin
 * d'une rampe — c'est-à-dire celui que la capture a installé. Rend `undefined` quand aucun
 * intervalle ne s'ouvre là : la rampe n'a rien changé, et le rejeu se tait.
 */
function proprietaireApres(spans: readonly SpanLike[], fin: number): number | null | undefined {
  for (const s of spans) {
    if (s.t0 >= fin && s.t0 <= fin + TOLERANCE_FRAMES) return s.owner ?? null
  }
  return undefined
}

/**
 * TOLERANCE_FRAMES — l'écart admis entre la fin d'une rampe et l'ouverture de l'intervalle
 * qu'elle produit. La jauge est ALLÉGÉE à la publication (un point par variation >= 0,02 ou par
 * seconde), donc son dernier point précède la bascule de quelques frames.
 */
const TOLERANCE_FRAMES = 20

/**
 * ticsDeDomination rend un tic par seconde tant qu'un camp tient TOUTES les zones.
 *
 * LA GARDE EST DOUBLE, et chacune de ses deux moitiés a sa raison : au moins deux zones (une
 * zone unique n'est pas une domination), et aucun intervalle `active` (ce marqueur ne vit qu'en
 * Roi de la colline, où une colline tenue ferait sonner tout le match).
 */
function ticsDeDomination(
  zones: readonly ZoneStateLike[],
  allyTeam: number | null,
  doc: ReplayDocumentReady,
): ReplaySoundEvent[] {
  if (allyTeam === null || zones.length < 2) return []
  if (zones.some((z) => z.spans.some((s) => s.active))) return []
  const out: ReplaySoundEvent[] = []
  for (const iv of intervallesDeDomination(zones)) {
    const stem = iv.owner === allyTeam ? ZONE_SOUND_STEMS.tick.ally : ZONE_SOUND_STEMS.tick.enemy
    const debut = frameToMs(iv.t0, doc)
    const fin = frameToMs(iv.t1, doc)
    for (let n = 0; n < ZONE_TICK_MAX_PAR_INTERVALLE; n++) {
      const ms = debut + n * ZONE_TICK_PERIOD_MS
      if (ms > fin) break
      out.push(soundEvent(ms, stem))
    }
  }
  return out
}

/**
 * intervallesDeDomination rend les tranches de temps où TOUTES les zones ont le même
 * propriétaire non nul. Le balayage se fait sur les BORNES des intervalles de toutes les zones
 * — la seule grille où l'état peut changer.
 */
function intervallesDeDomination(
  zones: readonly ZoneStateLike[],
): { t0: number; t1: number; owner: number }[] {
  const bornes = new Set<number>()
  for (const z of zones) for (const s of z.spans) bornes.add(s.t0)
  const grille = [...bornes].sort((a, b) => a - b)
  const out: { t0: number; t1: number; owner: number }[] = []
  for (const t of grille) {
    const owner = proprietaireCommun(zones, t)
    if (owner === null) continue
    const dernier = out[out.length - 1]
    const fin = finDeDomination(zones, t, owner)
    if (dernier && dernier.owner === owner && t <= dernier.t1) {
      dernier.t1 = Math.max(dernier.t1, fin)
      continue
    }
    out.push({ t0: t, t1: fin, owner })
  }
  return out
}

/** proprietaireCommun rend le camp qui tient TOUTES les zones à cette frame, ou `null`. */
function proprietaireCommun(zones: readonly ZoneStateLike[], t: number): number | null {
  let owner: number | null = null
  for (const z of zones) {
    const s = z.spans.find((x) => x.t0 <= t && t <= x.t1)
    if (!s || s.owner === null || s.owner === undefined) return null
    if (owner === null) owner = s.owner
    else if (owner !== s.owner) return null
  }
  return owner
}

/** finDeDomination rend la dernière frame où le camp tient encore toutes les zones. */
function finDeDomination(zones: readonly ZoneStateLike[], t: number, owner: number): number {
  let fin = Number.MAX_SAFE_INTEGER
  for (const z of zones) {
    const s = z.spans.find((x) => x.t0 <= t && t <= x.t1)
    if (!s || s.owner !== owner) return t
    fin = Math.min(fin, s.t1)
  }
  return fin
}

/**
 * collinesSuivantes rend un son par DÉPLACEMENT de la colline : chaque début d'intervalle
 * `active` sauf le premier de la partie.
 */
function collinesSuivantes(
  zones: readonly ZoneStateLike[],
  doc: ReplayDocumentReady,
): ReplaySoundEvent[] {
  const debuts: number[] = []
  for (const z of zones) for (const s of z.spans) if (s.active) debuts.push(s.t0)
  debuts.sort((a, b) => a - b)
  return debuts
    .slice(1)
    .map((t) => soundEvent(frameToMs(t, doc), ZONE_SOUND_STEMS.newZone))
}
