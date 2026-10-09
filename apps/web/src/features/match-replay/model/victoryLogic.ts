/**
 * victoryLogic — COMMENT CE MATCH S'EST TERMINÉ POUR CELUI QU'ON REGARDE.
 *
 * POUR QUI, EXACTEMENT : par défaut le joueur de la page, et depuis le 2026-09-06 le POINT DE
 * VUE quand l'appelant en passe un (plan « frise, point de vue », décisions 3 et 12). L'écran
 * de fin et l'export vidéo suivent ce qu'on regarde ; le SON et la voix de fin, eux, restent
 * ancrés sur le joueur de la page — inspecter un adversaire ne doit pas jouer « Défaite » sur
 * un match gagné. Les deux surfaces partagent donc cette lecture, mais lui passent un sujet
 * DIFFÉRENT et explicite : voir `readVictory`.
 *
 * L'ÉCRAN DE FIN EST LE SIEN, PAS CELUI DU VAINQUEUR (amendement utilisateur du 2026-08-26,
 * en cours de lot). Comme dans le jeu, l'écran de fin porte les couleurs et le logo de VOTRE
 * équipe, que vous ayez gagné ou perdu — on ne se fait pas afficher l'emblème de l'adversaire
 * parce qu'il a gagné. Ce module rend donc TROIS choses distinctes : l'ISSUE (victoire, défaite,
 * égalité, du point de vue du joueur de la page), SON équipe (celle qui habille l'écran), et
 * l'équipe qui GAGNE (que l'écran peut nommer). En victoire les deux dernières coïncident ; en
 * défaite elles diffèrent, et c'est tout l'intérêt de les séparer.
 *
 * L'ISSUE SE LIT DANS L'EN-TÊTE, JAMAIS DANS LE SCORE (décision D-B2). L'en-tête publie
 * `outcome_code`, qui est le verdict du joueur DE LA PAGE (2 = victoire, 3 = défaite,
 * 1 = égalité) : c'est une donnée servie, pas une déduction. Déduire le vainqueur du calque de
 * score serait faux au moins deux fois — un mode sans compteur ne publie aucune série, et un
 * match gagné au CHRONO peut finir sur un score à égalité (témoin 64e8adfa, 2-2). Le code de
 * l'en-tête sait dire « gagné 2-2 » ; deux nombres non.
 *
 * DE « MON RÉSULTAT » À « L'ÉQUIPE QUI GAGNE » IL FAUT UN PONT, et c'est le FILM qui le fait
 * (2026-10-06 : l'équipe d'un joueur est celle du film) : l'équipe du film du joueur de la page
 * donne mon camp, et entre exactement deux camps « j'ai perdu » suffit à nommer l'autre. Quand
 * le film ne situe pas le joueur de la page — absent, ou dont il tait l'équipe —, la question n'a
 * pas de réponse sûre : `null`, et l'écran ne se rend pas. Ce pont est aussi ce qui donne à
 * l'écran son habillage : pas de camp connu, pas d'identité à porter.
 *
 * DEUX CAMPS, PAS UN DE PLUS, PAS UN DE MOINS (décision D-B1), y compris pour l'égalité. La
 * même doctrine que le bandeau de score (`scoreBannerLogic.ts`) : un FFA n'a pas d'« équipe
 * gagnante » à nommer, et un mode à trois camps désignerait arbitrairement un adversaire parmi
 * plusieurs. Le panneau d'égalité, lui, ne nomme personne — mais il annonce la fin d'un match
 * OPPOSANT DEUX CAMPS, et l'afficher sur un FFA laisserait croire que les huit joueurs ont fini
 * à égalité. Une lecture absente ne ment pas, une lecture qui invente un cadre si.
 *
 * L'ÉGALITÉ NE REND AUCUNE ÉQUIPE, et c'est délibéré : elle ne désigne personne (décision
 * D-B1), donc le panneau reste neutre. Rendre `null` pour les deux équipes rend cette
 * neutralité IMPOSSIBLE À CONTOURNER par mégarde à l'écran, plutôt que de la confier à une
 * discipline de rendu.
 *
 * LE CODE HORS CONTRAT NE FABRIQUE RIEN. Le mapping code→issue est celui du dépôt,
 * `lib/outcome.ts` (source unique, garde-rail `outcome.guard.test.ts`) : un code absent, nul ou
 * hors 1..4 y rend `null`. Le DNF (code 4) rend bien une issue, mais pas un écran de fin : un
 * match quitté ne se conclut pas (D-B1).
 *
 * Module PUR : ni React, ni DOM, ni couleur, ni libellé.
 */
import { outcomeCodeToValue } from '@/lib/outcome'
import type { FilmAllegiance } from '@/lib/replay/filmAllegiance'
import type { ReplayCamp } from '@/lib/replay/replayCamps'

/**
 * Ce que ce module lit du FILM : les camps du match (désignateurs, nommés par la feuille) et
 * l'équipe d'un joueur désigné par son xuid de base. `FilmAllegiance` s'y conforme — la
 * référence qu'elle porte n'est PAS lue ici : le joueur de la page et le sujet sont passés.
 */
export type VictoryFilm = Pick<FilmAllegiance, 'camps' | 'teamOfXuid'>

/** L'issue du match POUR LE SUJET de la lecture — le joueur de la page à défaut de point de vue. */
export type VictoryOutcome = 'win' | 'loss' | 'tie'

/** Une équipe désignée par la lecture : de quoi la nommer, la teinter et la coiffer. */
export interface VictoryTeam {
  /** Le désignateur d'équipe du film — clé du logo. */
  teamID: number
  /**
   * Le côté de feuille qui NOMME le camp (`ReplayCamp.side`, format `t{N}`) — ce que la cascade de
   * libellé attend ; `null` quand aucun de ses membres n'a de ligne (le libellé retombe alors
   * sur « Équipe N » du désignateur).
   */
  teamSide: string | null
  /** `true` si c'est l'équipe du SUJET (le joueur de la page à défaut de point de vue). */
  ally: boolean
}

/** La lecture de fin de match : l'issue, l'équipe qui habille l'écran, celle qui gagne. */
export interface VictoryReading {
  outcome: VictoryOutcome
  /**
   * L'ÉQUIPE DU SUJET — l'habillage de l'écran. `null` sur une égalité.
   *
   * « Du joueur de la page » jusqu'au 2026-09-06 : c'était vrai tant que `readVictory` devinait
   * son sujet. Depuis qu'il le reçoit en paramètre, c'est l'équipe de CELUI QU'ON REGARDE — le
   * joueur de la page quand aucun sujet n'est passé, ce que fait toujours la fin de partie
   * sonore (décision 3 du plan).
   */
  mine: VictoryTeam | null
  /** L'équipe qui remporte le match. `null` sur une égalité, qui ne désigne personne. */
  winner: VictoryTeam | null
}

/**
 * readVictory rend la lecture de fin de match, ou `null` quand aucun écran ne doit s'afficher :
 * match qui n'oppose pas exactement deux camps du film, résultat non publié ou hors contrat,
 * abandon, ou joueur de la page que le film ne situe pas (cf. l'en-tête du module).
 *
 * `page` : le xuid de base du joueur de la page (`meXUIDOf`) — celui dont `outcomeCode` est le
 * verdict. Le module ne le cherche pas lui-même : il le reçoit, comme le sujet.
 *
 * `subject` (2026-09-06, plan « frise, point de vue ») : PAR LES YEUX DE QUI cette fin se lit.
 *
 * ATTENTION À L'ASYMÉTRIE DES DEUX ENTRÉES, c'est tout le sujet. `outcomeCode` est le verdict
 * DU JOUEUR DE LA PAGE, servi par l'en-tête — il n'en existe pas d'autre, l'API ne publie pas
 * le résultat vu d'un adversaire. Quand `subject` désigne quelqu'un de l'AUTRE camp, la lecture
 * se retourne donc : ce que le joueur de la page a gagné, lui l'a perdu. On calcule la lecture
 * de la page, puis on la permute. Sujet du même camp : rigoureusement identique. Égalité : elle
 * l'est pour tout le monde, rien à permuter. Sujet que le film ne situe pas : `null` — aucun
 * écran plutôt qu'un écran faux.
 *
 * `subject` absent : le joueur de la page, comportement d'origine — c'est ce que passe la fin
 * de partie SONORE (décision 3), qui reste ancrée sur lui.
 */
export function readVictory(
  film: VictoryFilm,
  outcomeCode: number | null | undefined,
  page: string | null,
  subject?: string | null,
): VictoryReading | null {
  const camps = film.camps
  if (camps.length !== 2) return null
  const outcome = outcomeCodeToValue(outcomeCode)
  if (outcome === 'tie') return { outcome: 'tie', mine: null, winner: null }
  if (outcome !== 'win' && outcome !== 'loss') return null
  const mineIndex = campIndex(camps, film.teamOfXuid(page))
  if (mineIndex === null) return null
  const vu = subjectCampIndex(film, subject, mineIndex)
  if (vu === null) return null
  const won = vu === mineIndex ? outcome === 'win' : outcome === 'loss'
  const mine = camps[vu]
  const winner = won ? mine : camps[1 - vu]
  return {
    outcome: won ? 'win' : 'loss',
    mine: { teamID: mine.team, teamSide: mine.side, ally: true },
    winner: { teamID: winner.team, teamSide: winner.side, ally: won },
  }
}

/**
 * victoryIsFlipped — LA LECTURE A-T-ELLE ÉTÉ RETOURNÉE ? (2026-09-07)
 *
 * `true` quand le sujet est du camp OPPOSÉ à celui du joueur de la page : ce que la page a
 * gagné, lui l'a perdu, et `readVictory` a permuté en conséquence. `false` partout ailleurs —
 * sans sujet, sujet du même camp, sujet non situable, match qui n'oppose pas deux camps.
 *
 * C'EST LE PRÉDICAT DU SCORE : `finalScoreFromHeader` l'appelle pour savoir s'il doit échanger
 * les deux nombres servis par l'en-tête, qui valent pour le joueur de la page. Une seule
 * définition de « l'autre camp », plutôt que la même triple comparaison sous un autre nom.
 *
 * IL A SERVI UN TEMPS AU MOT DU VERDICT, et n'y sert plus : l'écran de fin et le panneau de
 * l'export choisissaient entre `header.outcome_label` (le mot du backend, valable pour le
 * joueur de la page) et le libellé canonique de l'issue permutée. Depuis que les deux titres
 * viennent d'`outcomes.toml` sur l'issue LUE, il n'y a plus de branche à ouvrir — le mot suit
 * la permutation tout seul. Le score, lui, n'a pas d'équivalent : les nombres de l'en-tête ne
 * portent pas de point de vue, d'où ce prédicat toujours vivant.
 *
 * POURQUOI UNE FONCTION ET NON UN CHAMP DE `VictoryReading` : l'ajouter à l'objet aurait fait
 * rougir les quatorze cas de `victoryLogic.test.ts`, qui fixent la lecture ENTIÈRE par égalité
 * profonde — la caractérisation n'accepte que des ajouts, et un champ de plus n'en est pas un.
 */
export function victoryIsFlipped(
  film: VictoryFilm,
  page: string | null,
  subject?: string | null,
): boolean {
  if (subject == null) return false
  if (film.camps.length !== 2) return false
  const mien = campIndex(film.camps, film.teamOfXuid(page))
  if (mien === null) return false
  const vu = subjectCampIndex(film, subject, mien)
  return vu !== null && vu !== mien
}

/**
 * subjectCampIndex dit dans lequel des deux camps se trouve le point de vue, ou `null` quand le
 * film ne le situe pas (absent du film, équipe tue, camp hors des deux retenus).
 *
 * Sans sujet, la réponse est le camp du joueur de la page, déjà calculé : on ne le recherche
 * pas deux fois, et surtout on ne peut pas diverger de lui.
 */
function subjectCampIndex(
  film: VictoryFilm,
  subject: string | null | undefined,
  defaut: 0 | 1,
): 0 | 1 | null {
  if (subject == null) return defaut
  return campIndex(film.camps, film.teamOfXuid(subject))
}

/**
 * campIndex dit LEQUEL des deux camps du film porte ce désignateur (0 ou 1), ou `null` : équipe
 * inconnue (le film la tait), aucune (`-1`), ou hors des deux camps.
 */
function campIndex(camps: readonly ReplayCamp[], team: number | null): 0 | 1 | null {
  if (team === null) return null
  if (camps[0].team === team) return 0
  if (camps[1].team === team) return 1
  return null
}

/**
 * Le score FINAL du match, dans l'ordre du SUJET : son camp d'abord, l'autre ensuite. Sans
 * sujet, c'est l'ordre du joueur de la page — celui dans lequel l'API le publie.
 */
export interface FinalScoreReading {
  ally: number
  enemy: number
}

/** Ce que l'écran de fin lit de l'en-tête de la vue match pour connaître le score final. */
export interface FinalScoreHeader {
  score_kind?: string
  score_mine?: number
  score_theirs?: number
}

/**
 * finalScoreFromHeader rend le score final SERVI PAR L'API, et `null` quand l'en-tête n'en
 * publie pas — auquel cas l'écran de fin garde sa lecture du calque du film.
 *
 * POURQUOI L'API ICI, ALORS QUE TOUT LE RESTE DU REJEU VIENT DU FILM. La lecture du calque à
 * la borne de fin rend, sur un mode à manches, les points de la DERNIÈRE MANCHE (Oddball :
 * « 100 - 43 ») présentés comme le score du match. C'est faux : le match s'est joué en deux
 * manches à une. L'écran de fin annonce un RÉSULTAT, et le résultat est celui que toute
 * l'app affiche par ailleurs — le prendre de l'en-tête garantit que les deux surfaces ne
 * peuvent pas dire deux nombres différents du même match.
 *
 * IL NE FAUT PAS LE RESTREINDRE AUX SEULES LECTURES EN MANCHES, et c'est un piège qui a
 * réellement été posé puis retiré : sur une variante à manches dont les deux camps finissent
 * à ÉGALITÉ de manches (témoin `adb93fb7` : 1 partout, plus une nulle), le serveur retombe
 * volontairement sur les points et publie `score_kind = "points"`. Filtrer sur « rounds »
 * renvoyait alors l'écran de fin vers le calque, donc vers les points de la dernière manche,
 * pendant que la vue match affichait le total. Le critère est donc la PRÉSENCE des deux
 * nombres, jamais leur nature.
 *
 * Le pendant vivant — le compte de manches qui monte pendant la lecture — reste dérivé du
 * film : il doit suivre la position de lecture, ce que l'en-tête ne sait pas faire
 * (cf. `roundsTally`).
 *
 * # LE SUJET, ET POURQUOI IL A FALLU L'AJOUTER (2026-09-07, lot L3)
 *
 * `score_mine` / `score_theirs` sont ANCRÉS SUR LE JOUEUR DE LA PAGE — l'API ne publie pas le
 * score vu d'un adversaire, exactement comme elle ne publie qu'un seul `outcome_code`. Tant que
 * la page n'avait qu'un point de vue, l'ordre était juste par construction. Depuis L2b il ne
 * l'est plus : `ReplayVictoryOverlay` donne à cette lecture la PRIORITÉ sur celle du calque
 * (`readScoreBanner`), qui, elle, suit le point de vue. Vu depuis un adversaire, l'écran
 * annonçait donc l'issue permutée (`readVictory` le fait) et le score dans l'ordre du joueur de
 * la page : « Défaite, 3 - 1 ». Faux, en plein cadre, et parfaitement silencieux.
 *
 * La règle est celle de `readVictory` : sujet de l'AUTRE camp, on permute. Sujet du même camp,
 * rigoureusement identique.
 *
 * SUJET NON SITUABLE (que le film ne range dans aucun des deux camps, match qui n'oppose pas
 * exactement deux camps) : on rend l'ordre du JOUEUR DE LA PAGE, pas `null`. Deux raisons. D'abord il n'y
 * a rien à permuter — sans camp, « l'autre camp » n'existe pas, et l'ordre de l'API est le seul
 * sens que ces deux nombres aient. Ensuite aucun score faux ne peut atteindre l'écran par ce
 * chemin : les deux seules surfaces qui l'affichent (l'écran de fin et le panneau de l'export)
 * ne se rendent qu'à condition que `readVictory` rende une lecture — et celui-ci rend `null`
 * dans exactement les mêmes cas. Rendre `null` ici n'effacerait donc rien de visible, mais
 * priverait un futur lecteur d'un score que l'API publie bel et bien.
 *
 * `subject` absent : comportement d'origine, à la ligne près (cf. `victoryLogic.test.ts`).
 */
export function finalScoreFromHeader(
  header: FinalScoreHeader | undefined,
  film?: VictoryFilm,
  page?: string | null,
  subject?: string | null,
): FinalScoreReading | null {
  if (!header || header.score_mine == null || header.score_theirs == null) return null
  const parLaPage = { ally: header.score_mine, enemy: header.score_theirs }
  if (!film) return parLaPage
  // MÊME DÉFINITION DE « L'AUTRE CAMP » QUE L'ISSUE (2026-09-07) : ce test était écrit ici en
  // quatre gardes, recopiées de `readVictory`. Elles vivent désormais dans `victoryIsFlipped`,
  // que l'écran de fin appelle aussi pour choisir son MOT — les trois ne peuvent plus diverger.
  return victoryIsFlipped(film, page ?? null, subject)
    ? { ally: parLaPage.enemy, enemy: parLaPage.ally }
    : parLaPage
}
