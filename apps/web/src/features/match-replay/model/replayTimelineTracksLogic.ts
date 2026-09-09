/**
 * replayTimelineTracksLogic — CE QUE LA FRISE MONTRE, en dehors du temps : les marques des
 * pistes (celles du joueur REGARDÉ, celles de ses COÉQUIPIERS — cf. `buildEventTracks`), les
 * segments de DOMINANCE, et les médias posés sur le match. Logique pure, testable, sans JSX —
 * les composants n'en font que le rendu.
 *
 * TROIS PISTES PLUTÔT QU'UNE FRISE : une frise seule donne la position dans le temps et rien
 * d'autre — c'est le constat qui avait donné les marques de retournement (`ReplayLeadMarks`,
 * supprimé le 2026-08-28 : la piste DOMINANCE dit la même chose en DURÉES). Empilées, les
 * pistes disent la FORME du match avant qu'on l'ait lu — où tu as marqué, où tu es tombé,
 * quand ton équipe a pris l'ascendant.
 *
 * L'ÉCHELLE EST CELLE DE LA FRISE, donc celle de la FENÊTRE DE GAMEPLAY (`replayWindow.ts`) :
 * une marque calculée sur le film entier se poserait à côté de l'instant qu'elle désigne. Et
 * comme les pistes s'alignent sur un `input[type=range]`, elles héritent de sa géométrie : la
 * piste utile court de THUMB_PX / 2 à largeur − THUMB_PX / 2 — le navigateur ne publie pas la
 * largeur du curseur.
 *
 * LE SUFFIXE `Logic` LÈVE UNE COLLISION DE NOMS, il n'est pas décoratif : Windows ne distingue
 * pas ce fichier de `ReplayTimelineTracks.tsx`, et TypeScript refuse alors les deux dans le
 * même programme (TS1149). C'est aussi le patron du dépôt (killFeedLogic, victoryLogic).
 */
import type { PlayerMarkKind } from '../../../lib/replay/playerMarks'
import type { MedalEvent } from './killFeedLogic'
import type { ReplayWindowBounds } from './replayWindow'

/**
 * Largeur supposée du curseur natif, en px. Mesure héritée des marques de retournement : le
 * navigateur ne publie pas cette valeur, 16 px est celle des thèmes par défaut, et l'écart
 * résiduel se compte en pixels sur une frise qui en fait plusieurs centaines.
 */
export const THUMB_PX = 16

/** Position CSS d'un ratio [0..1] sur la piste, curseur compris. */
export function trackLeft(ratio: number): string {
  const r = Math.min(1, Math.max(0, ratio))
  return `calc(${THUMB_PX / 2}px + (100% - ${THUMB_PX}px) * ${r})`
}

/**
 * LA MÊME POSITION, MAIS SUR UNE VARIABLE CSS que le navigateur résout tout seul (2026-09-06).
 *
 * POURQUOI UNE SECONDE FORME PLUTÔT QU'UN SECOND CALCUL. Ce qui suit la LECTURE — la bulle de
 * temps, le trait de lecture — ne connaît pas son ratio au rendu : il change soixante fois par
 * seconde, et le passer en prop coûterait un rendu de la frise entière par image. Ces
 * objets-là sont donc positionnés par une variable écrite en impératif
 * (`useReplayPlayback.writeCursor`, `--played-r`). Ils ont pourtant besoin de la MÊME géométrie
 * que les marques, sans quoi le trait de lecture passe jusqu'à 8 px à côté du kill qu'il
 * désigne — c'est le défaut exact que la bulle de temps portait, invisible faute de repère.
 *
 * La formule ne vit donc qu'ICI, sous deux formes : `trackLeft` pour une valeur connue au
 * rendu, `trackLeftVar` pour une position qui vit dans une variable. Le garde-rail
 * `ui/timelineGeometry.guard.test.ts` interdit d'en recopier les littéraux ailleurs.
 *
 * LE REPLI À ZÉRO N'EST PAS DÉCORATIF : tant que la lecture n'a pas posé le curseur une
 * première fois, la variable n'existe pas — et un `var()` sans repli rend tout le `calc()`
 * invalide, donc la position `auto`. L'objet sauterait de l'origine à sa place au premier pas.
 */
export function trackLeftVar(cssVar: string): string {
  return `calc(${THUMB_PX / 2}px + (100% - ${THUMB_PX}px) * var(${cssVar}, 0))`
}

/** Largeur CSS d'un intervalle [a..b] de ratios sur la même piste. */
export function trackWidth(from: number, to: number): string {
  const span = Math.min(1, Math.max(0, to)) - Math.min(1, Math.max(0, from))
  return `calc((100% - ${THUMB_PX}px) * ${Math.max(0, span)})`
}

/** Bornes de la frise, en IMAGES : les mêmes que celles du curseur. */
export interface TrackScale {
  from: number
  span: number
}

/**
 * L'échelle des pistes, lue de la fenêtre de gameplay comme la frise. `span <= 0` = rien à
 * placer (film d'une image, fenêtre dégénérée) : les appelants ne rendent alors aucune piste.
 */
export function trackScale(playWindow: ReplayWindowBounds | null, frameCount: number): TrackScale {
  const from = playWindow?.startFrame ?? 0
  const span = (playWindow?.endFrame ?? frameCount - 1) - from
  return { from, span }
}

/** Ratio [0..1] d'un instant du rejeu (ms) sur l'échelle des pistes. */
export function ratioOfMs(replayMs: number, frameIntervalMs: number, scale: TrackScale): number {
  if (!frameIntervalMs || scale.span <= 0) return 0
  return (replayMs / frameIntervalMs - scale.from) / scale.span
}

/** Une marque d'événement sur une piste. `kind` décide de sa couleur, jamais sa piste. */
export interface TrackMark {
  key: string
  ratio: number
  /**
   * TROIS ESPÈCES DE MARQUE, DEUX ENCRES ET UN ANNEAU. `kill` et `death` sont les deux encres du
   * rejeu ; `medal` est une médaille d'OBJECTIF, décrochée sans élimination (2026-09-07, L4) —
   * elle n'a pas de camp à dire et se dessine en ANNEAU CREUX, la même signature que l'anneau
   * qui décore un kill médaillé. Un seul code à apprendre : anneau = médaille.
   */
  kind: 'kill' | 'death' | 'medal'
  /** Instant, pour l'infobulle (mm:ss déjà mis en forme par l'appelant). */
  clock: string
  /**
   * LES MÉDAILLES DE CETTE MARQUE, identité complète (cf. `MedalEvent`) — vide quand il n'y en a
   * pas. EN IMAGES DEPUIS LE 2026-09-09 (décision D7, plan « vague C ») : la marque ne porte plus
   * qu'un libellé, elle porte de quoi dessiner le badge du jeu (`imageUrl`) et nourrir son
   * infobulle (`label`, `description`) — cf. `ui/MedalBadges.tsx`, déjà employé par le fil.
   *
   * Elles ne prennent PAS de repère à elles quand un kill les porte (décision 9 du plan) : la
   * majorité des médailles tombe à moins de 500 ms d'un kill déjà dessiné, et un second glyphe
   * au même endroit doublerait la frise sans rien ajouter. La marque existante reçoit le badge
   * en SURIMPRESSION.
   *
   * UN LIBELLÉ VIDE N'ENTRE PAS (matchs d'avant le backfill des médailles du fil) : la
   * décoration dirait « il s'est passé quelque chose » sans pouvoir dire quoi.
   */
  medals: readonly MedalEvent[]
  /**
   * L'acteur est-il un AMI du compte connecté (marque `friend` de `playerMarks`) ?
   *
   * LA FORME DIT L'IDENTITÉ, LA COULEUR DIT LE CAMP (décision D5 du plan d'habillage, reprise
   * en décision 4 du plan « frise, point de vue ») : un ami prend le LOSANGE, comme son point
   * sur la carte (`layers/replayMarkers.ts`, `MarkerShape = 'diamond'`). Sur cette frise la
   * couleur est déjà prise deux fois — kill et mort — et les deux encres reprennent les
   * couleurs d'équipe choisies par l'utilisateur en jeu : il n'en reste aucune de libre.
   */
  friend: boolean
}

/** Les deux pistes d'événements : celle du point de vue, celle de ses coéquipiers. */
export interface EventTracks {
  own: TrackMark[]
  teammates: TrackMark[]
}

/**
 * À QUI LA FRISE S'INTÉRESSE — les trois lectures d'identité dont les pistes ont besoin,
 * groupées pour ne pas allonger la liste d'arguments de `buildEventTracks`.
 *
 * ELLES NE SE DÉDUISENT PAS L'UNE DE L'AUTRE, et c'est pourquoi les trois sont là : le POINT DE
 * VUE dit à qui appartient la piste du haut, le CAMP (relatif à ce point de vue) dit qui va sur
 * celle des coéquipiers, et les MARQUES disent qui est un ami — une propriété du compte
 * connecté, indépendante du match et du camp.
 */
export interface TrackAudience {
  /** Le joueur regardé : sa piste porte ses kills ET ses morts. `null` = aucune piste propre. */
  viewpoint: string | null
  /** Camp de chaque xuid, RELATIF au point de vue (`XuidMeta` de la vue match). */
  identity: ReadonlyMap<string, { ally: boolean }>
  /** Marques d'identité du compte connecté : seul `friend` compte ici (cf. `TrackMark.friend`). */
  marks: ReadonlyMap<string, PlayerMarkKind>
}

/** Un kill, réduit à ce dont les pistes ont besoin. */
export interface TrackKill {
  key: string
  replayMs: number
  /** xuid du tueur (celui à qui la marque appartient). */
  xuid: string
  /** Médailles décrochées SUR ce kill, identité complète (cf. `TrackMark.medals`). */
  medals: readonly MedalEvent[]
}

/** Une mort, réduite de même. `xuid` est celui du défunt. */
export interface TrackDeath {
  key: string
  replayMs: number
  xuid: string
}

/** Une médaille ORPHELINE : décrochée sans kill à moins de 500 ms — un objectif, presque toujours. */
export interface TrackMedal {
  key: string
  replayMs: number
  xuid: string
  /** La médaille elle-même, identité complète et libellé non vide par construction (cf. `reduceFeed`). */
  medal: MedalEvent
}

/**
 * CE QUE LE FIL DONNE AUX PISTES : trois listes qui voyagent ensemble parce qu'elles sortent du
 * même parcours (`reduceFeed`) et se posent sur la même échelle. Groupées, elles tiennent
 * `buildEventTracks` sous les cinq paramètres du dépôt — la liste en comptait déjà six avant que
 * les médailles n'arrivent.
 */
export interface TrackEvents {
  kills: readonly TrackKill[]
  deaths: readonly TrackDeath[]
  medals: readonly TrackMedal[]
}

/**
 * buildEventTracks range les kills et les morts sur DEUX pistes : celle du POINT DE VUE, et
 * celle de ses COÉQUIPIERS.
 *
 * LA RÈGLE A CHANGÉ LE 2026-09-07 (décision 5 du plan « frise, point de vue »), et le
 * changement de comportement est assumé. La seconde piste montrait les joueurs marqués AMIS —
 * `playerMarks` marque un ami quel que soit son camp — et elle montre désormais les joueurs du
 * MÊME CAMP que le point de vue. Conséquence à connaître pour qu'aucune relecture n'y voie une
 * régression : un ami de l'équipe ADVERSE disparaît de la frise. Il reste losange sur la carte
 * et nommé dans le fil ; sa place n'était pas sur une piste qui répond à « comment mon équipe
 * s'en sortait-elle ». Les amis ne sont plus une PISTE, ils sont une FORME (cf. `TrackMark.friend`).
 *
 * Un acteur qui n'est ni le point de vue ni son coéquipier n'est sur aucune piste : la frise ne
 * parle pas de la salle entière. Un acteur sans camp connu (absent du tableau de score) non
 * plus — `identity` ne le porte pas, et deviner un camp le poserait sur une piste au hasard.
 *
 * Les morts ne vont QUE sur la piste du point de vue : « où je suis tombé » est une lecture de
 * soi. Une piste de coéquipiers mêlant leurs kills et leurs morts serait illisible à huit joueurs.
 *
 * LES MÉDAILLES NE DÉCORENT QUE LA PISTE DU POINT DE VUE (2026-09-07, lot L4). Le kill d'un
 * coéquipier garde sa marque nue même s'il a valu une médaille : la piste des coéquipiers est
 * atténuée par nature — elle répond à « comment mon équipe s'en sortait-elle », pas à « qui a
 * fait un doublé ». Y semer des anneaux la rendrait aussi dense que celle du dessus, qu'elle est
 * faite pour ne pas concurrencer. Le fil, lui, nomme la médaille de chacun.
 */
export function buildEventTracks(
  events: TrackEvents,
  audience: TrackAudience,
  frameIntervalMs: number,
  scale: TrackScale,
  clockOf: (replayMs: number) => string,
): EventTracks {
  const own: TrackMark[] = []
  const teammates: TrackMark[] = []
  if (scale.span <= 0) return { own, teammates }
  // UNE SEULE FABRIQUE POUR LES TROIS BOUCLES : la règle du losange (« un ami, quel que soit son
  // camp ») n'est écrite qu'ici. Recopiée dans chaque boucle, elle aurait fini par ne valoir que
  // sur les kills — et une mort d'ami perdrait sa forme sans que rien ne le dise.
  const marque = (
    key: string,
    replayMs: number,
    kind: TrackMark['kind'],
    xuid: string,
    medals: readonly MedalEvent[] = [],
  ): TrackMark => ({
    key,
    ratio: ratioOfMs(replayMs, frameIntervalMs, scale),
    kind,
    clock: clockOf(replayMs),
    medals,
    friend: audience.marks.get(xuid) === 'friend',
  })
  for (const k of events.kills) {
    const sien = audience.viewpoint != null && k.xuid === audience.viewpoint
    if (!sien && audience.identity.get(k.xuid)?.ally !== true) continue
    const at = marque(k.key, k.replayMs, 'kill', k.xuid, sien ? k.medals : [])
    if (at.ratio < 0 || at.ratio > 1) continue
    ;(sien ? own : teammates).push(at)
  }
  for (const d of events.deaths) {
    if (audience.viewpoint == null || d.xuid !== audience.viewpoint) continue
    const at = marque(d.key, d.replayMs, 'death', d.xuid)
    if (at.ratio < 0 || at.ratio > 1) continue
    own.push(at)
  }
  for (const m of events.medals) {
    if (audience.viewpoint == null || m.xuid !== audience.viewpoint) continue
    const at = marque(m.key, m.replayMs, 'medal', m.xuid, [m.medal])
    if (at.ratio < 0 || at.ratio > 1) continue
    own.push(at)
  }
  return { own, teammates }
}

/**
 * Un segment de DOMINANCE : qui menait AUX FRAGS, de quand à quand (en ratios de frise).
 *
 * `teamId = null` VEUT DIRE ÉGALITÉ, et c'est un fait mesuré comme un autre — pas un trou :
 * la piste le peint à l'encre d'égalité du dépôt (`outcome-draw`). Les deux camps à égalité
 * de frags est l'état LE PLUS FRÉQUENT d'un début de match (0-0), et le laisser vide se
 * lisait « on ne sait pas ».
 */
export interface DominanceSegment {
  key: string
  from: number
  to: number
  teamId: number | null
}

/**
 * Un FRAG posé sur l'axe du rejeu, réduit à ce que la dominance demande : quand, et par quel
 * camp. `teamId` est celui du TUEUR (`KillEvent.teamID`) — un kill dont le camp n'est pas
 * résolu n'entre pas dans le compte (cf. `buildFragDominance`).
 */
export interface TrackFrag {
  replayMs: number
  teamId: number
}

/**
 * buildFragDominance dit, à chaque instant du match, QUEL CAMP A LE PLUS DE FRAGS.
 *
 * POURQUOI LES FRAGS ET NON LE SCORE (demande utilisateur du 2026-08-28). La piste lisait le
 * calque de score (`leadChanges`), c'est-à-dire le compteur DU MODE : des captures en CTF, des
 * secondes de balle en Oddball. Deux ennuis. D'abord elle ne disait pas ce que son nom promet
 * — une équipe peut mener au score en ne tenant qu'un objectif, sans dominer un seul duel.
 * Ensuite le calque de score joueur est absent de modes entiers (mesure de la phase 0 :
 * Oddball 0 joueur sur 32 avec compteurs), là où le FIL DES ÉLIMINATIONS existe sur tous les
 * matchs. Les frags viennent donc du fil déjà aligné (`buildFeedEntries`), pas d'un second
 * calque : la bande et la ligne qu'on lit à côté parlent du même événement.
 *
 * LE MENEUR EST L'ARGMAX UNIQUE, comme partout dans le dépôt (`leaderAt`) : une ÉGALITÉ n'a
 * pas de meneur, et elle sort ici comme un segment `teamId: null` — la piste la peint en
 * BLEU (`outcome-draw`, demande utilisateur du 2026-08-28). Elle ne garde donc JAMAIS la
 * couleur du dernier meneur, et elle ne laisse pas non plus un vide qui se lirait « on ne
 * sait pas ». Généralise à plus de deux camps sans rien changer.
 *
 * AVANT LE PREMIER FRAG, personne ne mène : 0-0 EST une égalité, et la piste s'ouvre donc sur
 * une bande bleue. Contrairement au calque de score, rien n'est ici « non publié » — le compte
 * se fait de zéro, et le premier camp à frapper prend la tête à cet instant précis.
 *
 * AUCUN FRAG DU TOUT = AUCUNE BANDE, et c'est la seule sortie vide. Un match sans frag apparié
 * (aucun camp résolu au fil) n'est pas « une égalité de bout en bout » : c'est une absence de
 * mesure, et la peindre en bleu serait une affirmation.
 *
 * Un frag antérieur à la fenêtre de gameplay est BORNÉ à l'origine de la frise, pas rejeté :
 * il compte dans le total, et sa bande commence au bord.
 */
export function buildFragDominance(
  frags: readonly TrackFrag[],
  frameIntervalMs: number,
  scale: TrackScale,
): DominanceSegment[] {
  if (scale.span <= 0 || !frameIntervalMs || frags.length === 0) return []
  const counts = new Map<number, number>()
  // Le fil sort trié (`buildFeedEntries`), mais un compte cumulé lu à l'envers donnerait des
  // meneurs faux sans rien casser à l'écran : cette fonction ne dépend pas de son appelant.
  const ordered = [...frags].sort((a, b) => a.replayMs - b.replayMs)
  // LE COUP D'ENVOI EST UNE ÉGALITÉ (0-0) : c'est le premier état, pas un préambule.
  const states: DominanceState[] = [{ teamId: null, at: null, from: 0 }]
  for (const frag of ordered) {
    counts.set(frag.teamId, (counts.get(frag.teamId) ?? 0) + 1)
    const leader = soleLeader(counts)
    if (leader === states[states.length - 1].teamId) continue
    states.push({
      teamId: leader,
      at: frag.replayMs,
      from: clampRatio(frag.replayMs, frameIntervalMs, scale),
    })
  }
  return toSegments(states)
}

/** Un état de la piste : qui mène (ou l'égalité), et depuis quand — instant brut et ratio. */
interface DominanceState {
  teamId: number | null
  /** Instant du frag qui ouvre l'état, en ms ; `null` pour le coup d'envoi (clé stable). */
  at: number | null
  from: number
}

/**
 * toSegments ferme chaque état sur l'ouverture du suivant, le dernier sur la fin de la frise.
 * Les segments de LARGEUR NULLE sont écartés : deux frags dans la même image (ou avant le coup
 * d'envoi) donnent le même ratio, et une bande invisible n'a rien à faire dans une liste.
 */
function toSegments(states: readonly DominanceState[]): DominanceSegment[] {
  const out: DominanceSegment[] = []
  for (let i = 0; i < states.length; i += 1) {
    const s = states[i]
    const to = states[i + 1]?.from ?? 1
    if (to <= s.from) continue
    out.push({ key: `${s.at ?? 'start'}-${s.teamId ?? 'tie'}`, from: s.from, to, teamId: s.teamId })
  }
  return out
}

/** Ratio d'un instant, RABATTU sur la frise : hors fenêtre, une bande se borne (cf. ci-dessus). */
function clampRatio(replayMs: number, frameIntervalMs: number, scale: TrackScale): number {
  return Math.min(1, Math.max(0, ratioOfMs(replayMs, frameIntervalMs, scale)))
}

/** Le même rabattement, pour une donnée déjà datée en IMAGES (le calque de score l'est). */
function clampFrameRatio(frame: number, scale: TrackScale): number {
  return Math.min(1, Math.max(0, (frame - scale.from) / scale.span))
}

/**
 * buildScoreDominance — LA MÊME LECTURE QUE LA DOMINANCE, MAIS SUR LE SCORE DU MODE.
 *
 * POURQUOI UNE SECONDE PISTE (demande utilisateur du 2026-08-28). Les frags disent qui gagne
 * les duels ; le score dit qui gagne LE MATCH, et les deux se séparent exactement dans les
 * modes qui ont un objectif — une équipe peut dominer les duels en perdant les captures. Les
 * superposer répondrait à deux questions à la fois ; les empiler les fait comparer d'un coup
 * d'œil, ce que ni le bandeau (une seule image) ni la courbe de la vue match (un autre écran)
 * ne permettent.
 *
 * ELLE NE S'AFFICHE PAS EN SLAYER, et c'est mesuré, pas supposé : « le score du Slayer EST le
 * compte de frags » (état de l'art des modes, témoin `000d5950` : score API 43-50 = somme des
 * frags par équipe). La piste y répéterait celle du dessus. Le tri se fait sur le RENDU, pas
 * sur un libellé de mode — cf. `sameLeadSegments`.
 *
 * `states` vient de `leaderStates` (lib/replay/scoreTimeline) : un état par changement,
 * ÉGALITÉS COMPRISES, daté en IMAGES du document. Le reste est la règle de la dominance —
 * bande d'ouverture à égalité (0-0), fermeture sur l'ouverture de la suivante, rabattement sur
 * la frise.
 */
export function buildScoreDominance(
  states: readonly LeadStateLike[],
  scale: TrackScale,
): DominanceSegment[] {
  if (scale.span <= 0 || states.length === 0) return []
  const out: DominanceState[] = [{ teamId: null, at: null, from: 0 }]
  for (const s of states) {
    out.push({ teamId: s.teamId, at: s.frame, from: clampFrameRatio(s.frame, scale) })
  }
  return toSegments(out)
}

/** Un état de la course au score, tel que `leaderStates` le publie (daté en IMAGES). */
export interface LeadStateLike {
  frame: number
  teamId: number | null
}

/** Un séparateur de manche posé sur la frise : où la manche `endedIndex` s'est terminée. */
export interface RoundSeparator {
  key: string
  endedIndex: number
  ratio: number
}

/**
 * LA PISTE SCORE, ou son absence. `null` (côté appelant) veut dire « ce match n'a pas de piste
 * score » — Slayer, calque absent, camps non identifiés — et NON « la piste est vide ».
 */
export interface ReplayScoreTrack {
  segments: readonly DominanceSegment[]
  rounds: readonly RoundSeparator[]
}

/**
 * roundSeparators pose un repère à chaque BASCULE DE MANCHE, en ratios de frise.
 *
 * Sans eux, une piste de score multi-manche est illisible : le compteur repart de zéro et le
 * meneur peut changer sans qu'aucune action ne l'explique. Les bascules viennent de
 * `roundTransitions` (roundsLogic), le même foyer que les pastilles du bandeau et l'écran
 * inter-manche — trois lectures, une seule définition de « où les manches se touchent ».
 *
 * Un repère hors fenêtre de gameplay est ÉCARTÉ, pas rabattu : collé au bord, il se lirait
 * comme une manche qui commence au coup d'envoi.
 */
export function roundSeparators(
  transitions: readonly { endedIndex: number; frame: number }[],
  scale: TrackScale,
): RoundSeparator[] {
  if (scale.span <= 0) return []
  const out: RoundSeparator[] = []
  for (const tr of transitions) {
    const ratio = (tr.frame - scale.from) / scale.span
    if (ratio < 0 || ratio > 1) continue
    out.push({ key: `r${tr.endedIndex}-${tr.frame}`, endedIndex: tr.endedIndex, ratio })
  }
  return out
}

/**
 * sameLeadSegments — LES DEUX PISTES DESSINERAIENT-ELLES LA MÊME CHOSE ?
 *
 * C'est le tri qui décide de montrer la piste SCORE, et il se fait sur LE RENDU plutôt que
 * sur le nom du mode : comparer un libellé (« Slayer ») serait faux au premier mode dérivé
 * (Super Fiesta est un Slayer, Attrition ne l'est pas) et illisible sur un second titre.
 *
 * HISTOIRE DE LA GARDE (2026-09-02, décision user D1). La première version
 * (`scoreMirrorsFrags`) comparait les TOTAUX finaux score/frags à l'égalité stricte : un seul
 * kill au camp non résolu (suicide, environnemental, acteur hors scoreboard) suffisait à
 * réafficher le doublon — c'était le cas nominal en Assassin, pas l'exception. La garde
 * compare désormais ce que l'utilisateur voit : la SUITE DES MENEURS et les FRONTIÈRES des
 * bandes. Deux pistes identiques à l'écran = un doublon, on n'en montre qu'une.
 *
 * LA TOLÉRANCE DE FRONTIÈRE est une affaire de PIXEL, pas de donnée : les deux pistes datent
 * la même bascule sur deux horloges (le fil des éliminations en ms, le calque de score en
 * frames), et ces horloges se recalent à mieux qu'une frame. En ratio de frise,
 * SAME_TRACK_EPS couvre cet écart de quantification — en dessous d'un demi-pour-cent de la
 * largeur, aucune différence n'est lisible. Deux suites de meneurs DIFFÉRENTES, elles, ne
 * passent jamais : le premier segment divergent rend `false`.
 */
export const SAME_TRACK_EPS = 0.005

export function sameLeadSegments(
  a: readonly DominanceSegment[],
  b: readonly DominanceSegment[],
): boolean {
  if (a.length === 0 || a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    if (a[i].teamId !== b[i].teamId) return false
    if (Math.abs(a[i].from - b[i].from) > SAME_TRACK_EPS) return false
    if (Math.abs(a[i].to - b[i].to) > SAME_TRACK_EPS) return false
  }
  return true
}

/** Le camp SEUL en tête au compte donné, ou `null` (égalité) — même règle que `leaderAt`. */
function soleLeader(counts: ReadonlyMap<number, number>): number | null {
  let best = -1
  let bestTeam: number | null = null
  let tied = false
  for (const [teamId, n] of counts) {
    if (n > best) {
      best = n
      bestTeam = teamId
      tied = false
    } else if (n === best) {
      tied = true
    }
  }
  return tied ? null : bestTeam
}

/**
 * UN MÉDIA DU JOUEUR posé sur le match : une capture (instant) ou un clip (durée). La source
 * est le `media_tab` de la vue match, réduit à ce format par `buildReplayMedia`
 * (replayMediaLogic.ts) — c'est LÀ que vivent la règle « début = fin − durée » et le recalage
 * sur l'axe du rejeu ; ici, `replayMs` est déjà sur cet axe.
 */
export interface ReplayMediaItem {
  id: string
  kind: 'image' | 'clip'
  /** Instant du média sur l'axe du rejeu, en ms. */
  replayMs: number
  /** Durée du clip, en ms. Absente ou 0 pour une capture. */
  durationMs?: number
  /** Vignette (petite) et média plein, tels que l'API les publiera. */
  thumbUrl: string
  url: string
  label?: string
}

/** Un média placé sur la piste : ratio de début, et largeur pour un clip. */
export interface PlacedMedia {
  item: ReplayMediaItem
  from: number
  to: number
}

/**
 * placeMedia pose les médias sur l'échelle de la frise. UN CLIP OCCUPE SA DURÉE (c'est ce qui
 * le distingue d'une capture à l'œil, sans légende) ; une capture reçoit une largeur nulle et
 * la piste lui donne sa taille de vignette. Un média hors fenêtre de gameplay est écarté :
 * la frise ne montre pas le countdown, elle ne montrera pas ce qui s'y est passé.
 */
export function placeMedia(
  media: readonly ReplayMediaItem[],
  frameIntervalMs: number,
  scale: TrackScale,
): PlacedMedia[] {
  if (scale.span <= 0) return []
  const out: PlacedMedia[] = []
  for (const item of media) {
    const from = ratioOfMs(item.replayMs, frameIntervalMs, scale)
    if (from < 0 || from > 1) continue
    const to =
      item.kind === 'clip' && item.durationMs
        ? Math.min(1, ratioOfMs(item.replayMs + item.durationMs, frameIntervalMs, scale))
        : from
    out.push({ item, from, to })
  }
  return out.sort((a, b) => a.from - b.from)
}

/**
 * COMBIEN D'IMAGES MONTRER D'UN CLIP dans la lightbox : une image toutes les trois secondes,
 * bornée à [4..12]. La borne basse évite une bande d'une seule vignette (elle ne dirait pas
 * qu'il s'agit d'une durée), la borne haute évite une bande illisible sur un long clip.
 */
export function clipFrameCount(durationMs: number): number {
  return Math.min(12, Math.max(4, Math.round(durationMs / 3_000)))
}
