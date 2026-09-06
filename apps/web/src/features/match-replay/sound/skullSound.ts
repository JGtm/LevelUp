/**
 * skullSound.ts — LES SONS DU CRANE d'Oddball, et les deux canaux qui les datent.
 *
 * POURQUOI UN FICHIER A PART, la meme raison que `zoneSound.ts` : la source n'est pas
 * `doc.objectives`. Le nommage statborg (`objectiveevents/named.go`) ne porte que
 * `ObjectiveTypeFlag`, `ObjectiveTypeZone` et `ObjectiveTypeVip` — `ObjectiveTypeSkull` existe
 * comme constante sans table d'emplacements, donc AUCUNE statistique d'Oddball n'arrive dans
 * `doc.objectives`. Ces sons se datent sur `doc.skullCarries` (schema 23, les PERIODES DE
 * PORTAGE), sur `doc.objectiveObjects` (les VIES du crane libre) et sur `doc.scoreTimeline`.
 *
 * C'est ce qui a change depuis la RE du 2026-08-27
 * (`.ai/V7.5/RE_GESTES_SONORES_2026-08-27.md` §10.6, « les sept sons sont valides, le cablage
 * ne l'est pas encore ») : a l'epoque le moteur sonore ne lisait que `doc.objectives` et le
 * verdict etait « pas cablable ». Le schema 23 a ouvert le canal, et `zoneSound.ts` a etabli
 * le precedent d'un son qui ne vient d'aucune statistique.
 *
 * D'OU VIENNENT CES SONS. Banque `b0c651ea` = `sb_004_mod_mp_oddball` (28 evenements, 53 sons),
 * trouvee le 2026-08-27 par hachage FNV-1 du nom de fichier et jamais inventoriee avant. Les
 * noms d'evenement sont CASSES, pas designes a l'oreille :
 *
 *   play_004_mod_mp_oddball_skull_spawn     event f1dbd79a   1 son       5,47 s
 *   play_004_mod_mp_oddball_skull_despawn   event fab3a00d   1 parmi 2   1,71 s
 *   play_004_mod_mp_oddball_skull_taken     event db39b4c6   1 parmi 2   2,72 s
 *   play_004_mod_mp_oddball_skull_pickup    event f8b484a1   1 parmi 2   2,26 s  (-5 dB)
 *   play_004_mod_mp_oddball_skull_dropped   event 4a27941d   1 parmi 3   2,87 s
 *   play_004_mod_mp_oddball_scoring_team    event 60468d1a   2 couches   3,62 s
 *   play_004_mod_mp_oddball_scoring_enemy   event b72c33bf   2 couches   4,36 s
 *
 * Le rendu a ete refait localement depuis le module du jeu, et les durees obtenues rejouent
 * celles de la planche au centieme — c'est le controle qui valide le mappage evenement -> `.wem`.
 *
 * ## LA MARQUE : le son EXISTE DEJA dans l'arbre, sous un autre nom
 *
 * MESURE DU 2026-08-29, et elle a un temoin negatif. Les deux couches de `scoring_team`
 * portent le `.wem` 578850042 (1,81 s, joue deux fois bout a bout = les 3,62 s annonces) et
 * celles de `scoring_enemy` le `.wem` 444143858 (2,18 s x2 = 4,36 s). Compares aux fichiers
 * DEJA LIVRES `objective_zone_tick_team` / `_enemy`, leurs enveloppes correlent a **+1,000**
 * exactement, la ou deux sons sans rapport de la meme famille correlent a +0,66 / +0,69.
 *
 * C'est LE MEME SON, declare une fois par mode de jeu — exactement ce que la RE avait deja
 * constate pour la capture de zone (« le meme son declare une fois par mode »). On ne livre
 * donc AUCUN fichier de plus : ces deux gestes pointent sur les stems existants. Et le
 * traitement de ces fichiers tombe juste ici sans rien changer — ils sont tronques a 1,2 s
 * avec un fondu et attenues a -12 dBTP parce que le tic de Bastion sonne UNE FOIS PAR
 * SECONDE. En Oddball on marque 1 point par seconde de portage : meme cadence, meme besoin.
 *
 * ## APPARITION ET DISPARITION : une approximation ASSUMEE, et bornee par construction
 *
 * REGLE PRODUIT DE L'UTILISATEUR (2026-08-29) : « c'est quand il apparait sur son socle ou se
 * retrouve hors map. Ce n'est pas encore finalise comme evenements mais peut-etre que tu peux
 * faire comme si pour le son ? » Ces deux gestes ne sont donc PAS mesures — ils sont approches,
 * avec l'accord explicite de l'utilisateur, et la doctrine est de le dire ici plutot que de
 * laisser croire a une mesure.
 *
 * L'APPROXIMATION S'APPUIE SUR LA VERITE TERRAIN de `d9781168`
 * (`.ai/V7.5/replay2d/ODDBALL_VERITE_TERRAIN_d9781168.md`) : « 2:48 meurt en sautant dans le
 * vide avec le crane -> cooldown -> respawn socle 2:53 ». Le crane sorti de la carte ne tombe
 * pas au sol : il DISPARAIT, puis REAPPARAIT sur son socle. D'ou :
 *
 *   HORS MAP  une periode de portage qui se ferme SANS qu'aucune vie ne s'ouvre derriere.
 *             Un lacher normal, lui, rouvre une vie (le crane roule au sol) — les deux cas
 *             sont EXCLUSIFS, et c'est ce qui evite de sonner la chute et la disparition
 *             au meme instant.
 *   SOCLE     la premiere vie du film, puis la premiere vie qui suit chaque disparition.
 *
 * CE QUI BORNE LE COMPTE, et c'est le point important : l'apparition n'est PAS posee sur
 * « chaque debut de vie ». On ne sait toujours pas combien de vies un film Oddball publie, et
 * une vie par rebond ferait sonner une fanfare de 5,47 s a chaque rebond. En la chainant aux
 * disparitions, le nombre d'apparitions vaut AU PLUS 1 + le nombre de disparitions — une
 * quantite qui ne peut pas s'emballer, quelle que soit la granularite reelle des vies.
 *
 * ## Le camp, et ce qui se tait sans lui
 *
 * Les cinq gestes du crane n'ont PAS de variante d'equipe dans le jeu : la banque ne porte ni
 * `_team` ni `_enemy` sur ces evenements, contrairement aux captures de drapeau. Ils sonnent
 * donc pareil pour tout le monde, camp connu ou non — et c'est une mesure sur la banque, pas
 * un raccourci. La MARQUE, elle, a ses deux cotes : sans camp allie resolu (pas de ligne
 * « moi » au tableau de score), elle se TAIT — meme regle que partout dans la chaine sonore.
 */
import type {
  ReplayDocumentReady,
  ReplayObjectiveObjectReady,
  ReplaySkullCarry,
} from '../model/replayNormalize'
import { frameToMs } from '../model/replayLogic'
import { soundEvent, type ReplaySoundEvent } from './replaySoundVariants'
import { scoreTimelineOf } from '@/lib/replay/scoreTimeline'

/**
 * SKULL_SOUND_STEMS — les fichiers, par geste.
 *
 * MANIFESTE au meme titre que les tables de `replaySound.ts` — le garde-rail
 * `replaySoundAssets.guard.test.ts` le rejoue contre le dossier d'assets, dans les DEUX sens.
 *
 * `scoringTeam` / `scoringEnemy` POINTENT SUR LES STEMS DE ZONE, et ce n'est pas un pis-aller :
 * la mesure ci-dessus etablit que c'est le meme fichier (correlation +1,000). Le nom porte
 * « zone » parce que c'est la que le son a ete identifie en premier ; il designe en realite le
 * TIC DE SCORE, que Bastion et Oddball declarent chacun de leur cote.
 */
export const SKULL_SOUND_STEMS = {
  /** Crane qui REAPPARAIT sur son socle. Rendu de l'event `f1dbd79a`, un seul fichier. */
  spawn: 'objective_skull_spawn',
  /** Crane HORS MAP : il disparait au lieu de tomber. Event `fab3a00d`, 2 variantes. */
  despawn: 'objective_skull_despawn',
  /** Crane PRIS sur son socle (il y reposait). Rendu de l'event `db39b4c6`, 2 variantes. */
  taken: 'objective_skull_taken',
  /** Crane RAMASSE au sol. Rendu de l'event `f8b484a1` a -5 dB, 2 variantes. */
  pickup: 'objective_skull_pickup',
  /** Crane LACHE : le porteur meurt ou le pose, et le crane roule au sol. Event `4a27941d`. */
  dropped: 'objective_skull_dropped',
  /** Ticket de score, mon equipe. MEME FICHIER que le tic de Bastion (cf. l'en-tete). */
  scoringTeam: 'objective_zone_tick_team',
  /** Ticket de score, l'adversaire. MEME FICHIER que le tic de Bastion. */
  scoringEnemy: 'objective_zone_tick_enemy',
} as const

/**
 * TOLERANCE_FRAMES — l'ecart admis entre la fin d'une vie et la prise qu'elle precede, pour
 * reconnaitre « le porteur arrive SUR le crane au repos ». Meme ordre de grandeur que la
 * tolerance de `zoneSound.ts`, et meme raison : les series publiees sont ALLEGEES, donc le
 * dernier point d'une vie precede la bascule de quelques images.
 */
const TOLERANCE_FRAMES = 12

/**
 * RETOMBEE_FRAMES — la fenetre dans laquelle une vie doit s'ouvrir apres un lacher pour que ce
 * lacher soit une CHUTE AU SOL. Au-dela, le crane n'est pas retombe : il est sorti de la carte.
 * La verite terrain donne l'ordre de grandeur de l'autre cote (5 s entre la mort dans le vide
 * et le respawn au socle) ; une seconde suffit largement a couvrir une retombee reelle.
 */
const RETOMBEE_FRAMES = 60

/**
 * estAuRepos — la vie designe-t-elle un crane POSE ? `t0 === t1` : un seul instant emis, ce que
 * `skullPresence.ts` a etabli comme la signature du socle.
 */
function estAuRepos(vie: ReplayObjectiveObjectReady): boolean {
  return vie.t0 === vie.t1
}

/**
 * priseSurRepos dit si la prise `t0` suit immediatement un crane au repos. Elle cherche la vie
 * de plus grand `t1` qui se termine avant la prise, et ne repond vrai que si cette vie est un
 * repos ET qu'elle touche la prise a la tolerance pres — un repos vieux de dix secondes ne
 * decrit pas la prise en cours.
 */
function priseSurRepos(lives: readonly ReplayObjectiveObjectReady[], t0: number): boolean {
  let derniere: ReplayObjectiveObjectReady | undefined
  for (const vie of lives) {
    if (vie.t1 > t0) continue
    if (!derniere || vie.t1 > derniere.t1) derniere = vie
  }
  if (!derniere) return false
  return estAuRepos(derniere) && t0 - derniere.t1 <= TOLERANCE_FRAMES
}

/**
 * retombeApres dit si une vie s'ouvre dans la fenetre qui suit un lacher — c'est-a-dire si le
 * crane est retombe au sol.
 *
 * UN ARTEFACT SANS AUCUNE VIE REPOND VRAI, et cette degradation est le coeur de la fonction :
 * « aucune vie apres le lacher » et « aucune vie du tout » ne sont pas la meme chose. Le second
 * cas est un artefact qui ne publie pas `objectiveObjects` — on n'y sait RIEN de la trajectoire
 * du crane, et en conclure une sortie de carte serait affirmer l'exceptionnel a partir d'une
 * absence de donnee. Sans information, le rejeu joue le geste ORDINAIRE.
 */
function retombeApres(lives: readonly ReplayObjectiveObjectReady[], t1: number): boolean {
  if (lives.length === 0) return true
  return lives.some((v) => v.t0 >= t1 && v.t0 - t1 <= RETOMBEE_FRAMES)
}

/**
 * premiereVieApres rend l'instant de la premiere vie qui s'ouvre strictement apres `t`, ou
 * `undefined`. C'est la REAPPARITION du crane sur son socle.
 */
function premiereVieApres(
  lives: readonly ReplayObjectiveObjectReady[],
  t: number,
): number | undefined {
  let best: number | undefined
  for (const v of lives) {
    if (v.t0 > t && (best === undefined || v.t0 < best)) best = v.t0
  }
  return best
}

/** Le camp d'un ticket de score, vu de la page — meme notion que dans `objectiveSound.ts`. */
type MarqueSide = 'ally' | 'enemy'

/**
 * ticketsDeMarque — un evenement par PALIER DE SCORE, dans le camp de l'equipe qui marque.
 *
 * C'EST LE MECANISME DE L'AFFICHAGE AU-DESSUS DU CANVAS, et pas un second calcul : le film ne
 * transmet pas un score echantillonne mais ses CHANGEMENTS (`scoreTimeline.ts` : « une serie
 * est une suite de PALIERS — {t, v} veut dire a partir de la frame t la valeur est v »). Chaque
 * palier montant EST un ticket de score, celui-la meme que la banniere lit. On lit `total` et
 * non `rounds[]` : la grandeur qui repart de zero a chaque manche ferait passer une remise a
 * zero pour une marque.
 *
 * SANS CAMP ALLIE, RIEN NE SONNE : jouer l'un des deux cotes serait affirmer un camp.
 */
function ticketsDeMarque(doc: ReplayDocumentReady, allyTeam: number | null): ReplaySoundEvent[] {
  if (allyTeam === null) return []
  const timeline = scoreTimelineOf(doc)
  if (!timeline) return []
  const out: ReplaySoundEvent[] = []
  for (const team of timeline.teams) {
    const cote: MarqueSide = team.teamId === allyTeam ? 'ally' : 'enemy'
    const stem = cote === 'ally' ? SKULL_SOUND_STEMS.scoringTeam : SKULL_SOUND_STEMS.scoringEnemy
    let precedent: number | undefined
    for (const p of team.total) {
      if (precedent !== undefined && p.v > precedent) out.push(soundEvent(frameToMs(p.t, doc), stem))
      precedent = p.v
    }
  }
  return out
}

/**
 * skullSoundEvents — les sons du crane, poses sur l'horloge du rejeu.
 *
 * L'HORLOGE NE DEMANDE AUCUN RECALAGE : `t0` et `t1` d'une periode de portage sont deja des
 * index d'image, sur le meme axe que les tirs et les positions. La conversion est la meme
 * `frameToMs` que partout.
 *
 * LA PORTE D'ENTREE EST `skullCarries` : sans periode de portage, le film n'est pas un Oddball
 * et RIEN ne sonne — y compris les tickets de score, qui autrement se declencheraient sur les
 * paliers de n'importe quel mode.
 *
 * UNE PERIODE NON FERMEE NE LACHE RIEN. `closed: false` signale un portage que le FILM
 * interrompt (fin de match, fin d'enregistrement) : personne n'a lache le crane, et le rejeu ne
 * joue ni la chute ni la disparition. Sonner la fin de l'enregistrement serait inventer une
 * action que le match n'a pas eue.
 */
export function skullSoundEvents(
  doc: ReplayDocumentReady,
  allyTeam: number | null = null,
): ReplaySoundEvent[] {
  const carries: readonly ReplaySkullCarry[] = doc.skullCarries ?? []
  if (carries.length === 0) return []
  const lives = doc.objectiveObjects ?? []
  const out: ReplaySoundEvent[] = []

  // LES APPARITIONS SONT UN ENSEMBLE D'INSTANTS, pas une suite de `push` : l'apparition
  // d'ouverture et celle qui suit une disparition DESIGNENT LA MEME VIE quand le crane sort de
  // la carte avant d'avoir jamais touche le sol. Deux `push` y feraient sonner deux fanfares de
  // 5,47 s au meme instant ; l'ensemble les confond, ce qui est exactement le sens voulu.
  const apparitions = new Set<number>()
  const ouverture = premiereVieApres(lives, -1)
  if (ouverture !== undefined) apparitions.add(ouverture)

  for (const c of carries) {
    const prise = priseSurRepos(lives, c.t0) ? SKULL_SOUND_STEMS.taken : SKULL_SOUND_STEMS.pickup
    out.push(soundEvent(frameToMs(c.t0, doc), prise))
    if (!c.closed) continue
    // CHUTE ou HORS MAP, jamais les deux : le crane est retombe au sol, ou il a disparu.
    if (retombeApres(lives, c.t1)) {
      out.push(soundEvent(frameToMs(c.t1, doc), SKULL_SOUND_STEMS.dropped))
      continue
    }
    out.push(soundEvent(frameToMs(c.t1, doc), SKULL_SOUND_STEMS.despawn))
    // ... et le crane REAPPARAIT sur son socle a la vie suivante. C'est ce chainage qui borne
    // le nombre d'apparitions (cf. l'en-tete).
    const socle = premiereVieApres(lives, c.t1)
    if (socle !== undefined) apparitions.add(socle)
  }

  for (const t of apparitions) out.push(soundEvent(frameToMs(t, doc), SKULL_SOUND_STEMS.spawn))
  out.push(...ticketsDeMarque(doc, allyTeam))
  return out.sort((a, b) => a.ms - b.ms)
}
