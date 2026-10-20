/**
 * viewpointOptions — CE QUE LE MENU DE POINT DE VUE PROPOSE, et sous quelle valeur.
 *
 * # POURQUOI UN MODULE PUR POUR UNE LISTE D'OPTIONS (2026-09-07, lot L3)
 *
 * Le menu qui remplace le libellé « Toi » de la frise n'a l'air de rien : un `<select>`, des
 * `<optgroup>`, des noms. Deux règles y sont pourtant invisibles à la relecture et fausses en
 * silence si on les rate — d'où un module à part, testable sans monter un composant.
 *
 * 1. LA VALEUR D'UNE OPTION N'EST PAS LA CLÉ DU JOUEUR. Un bot n'a pas de xuid : le film
 *    l'identifie par la clé synthétique `bot:<nom>` (schéma 36), la base par un xuid `bid(N.0)`.
 *    Or les marques de la frise s'apparient sur les xuid du FIL, qui viennent de la BASE
 *    (`reduceFeed` compare l'acteur au point de vue, et le point de vue est résolu contre le
 *    tableau de score). Un menu qui rendrait la clé film pour un bot changerait bien le point de
 *    vue — vers un joueur que personne ne reconnaît — et rendrait une piste VIDE, sans erreur ni
 *    message. La valeur est donc `board.xuid` quand la jointure a abouti, et la clé du film
 *    seulement à défaut.
 *
 * 2. UN JOUEUR SANS LIGNE DE TABLEAU DE SCORE EST LISTÉ MAIS INERTE (décision 7 bis du plan,
 *    2026-09-07). Sans cette ligne il n'a pas de xuid de base — et `collectKillEvents`
 *    (`match-view/_momentum.ts`) JETTE les kills d'un acteur absent du tableau de score. Sa
 *    piste serait donc vide quoi qu'il arrive, et `resolveViewpoint` retomberait en silence sur
 *    le joueur de la page : le menu afficherait un nom, la frise en montrerait un autre. Une
 *    option qui ne fait rien doit le DIRE (`disabled` + une infobulle qui donne la raison)
 *    plutôt que faire semblant. On ne la retire pas non plus : son absence de la liste se
 *    lirait « ce joueur n'était pas là », ce qui est faux — le film le nomme.
 *
 * # LES SECTIONS SONT LES CAMPS DU FILM (décision du 2026-10-06)
 *
 * Comme les colonnes de fiches : un camp par désignateur du film (`groupByTeam`), nommé par la
 * MÊME cascade (`campLabel`, que l'appelant passe), et jamais une section « sans équipe ». Un
 * joueur sans ligne de tableau de score reste dans le camp que le film lui donne ; un joueur
 * dont le film tait l'équipe n'est dans aucun — c'est un défaut de source, pas une section.
 *
 * # CE QU'IL NE FAIT PAS
 *
 * Il ne traduit rien (les libellés lui arrivent en paramètre) et ne connaît ni React, ni le
 * point de vue courant : choisir est l'affaire de `hooks/useReplayViewpoint`, afficher celle de
 * `ui/ReplayViewpointSelect`.
 */
import { stripBotSuffix } from '@/lib/players/displayName'
import { groupByTeam, playerName, type ReplayPlayer, type ReplayTeamGroup } from '@/lib/replay/rosterLogic'

/** Une entrée du menu : un joueur qu'on peut (ou non) regarder. */
export interface ViewpointOption {
  /** Le xuid à passer au point de vue — celui de la BASE dès que la jointure a abouti. */
  value: string
  /** Le gamertag, sans le suffixe « [bot] » que le film accroche aux siens. */
  label: string
  /** Vrai quand ce joueur n'a pas de ligne de tableau de score (cf. l'en-tête, règle 2). */
  disabled: boolean
  /** Infobulle : le nom entier, ou la raison de l'inertie quand l'option est désactivée. */
  title: string
}

/** Une section du menu : un camp du film. */
export interface ViewpointOptionGroup {
  /** Clé de rendu stable : `camp:<désignateur>`. */
  key: string
  /** Nom du camp, tel que la colonne de fiches l'écrit (`campLabel`). */
  label: string
  options: ViewpointOption[]
}

/** Les deux libellés que ce module ne sait pas produire : ils viennent de l'i18n de la feature. */
export interface ViewpointOptionLabels {
  /** Le nom d'un camp du film — `campLabel`, la cascade des colonnes de fiches. */
  campLabelOf: (camp: ReplayTeamGroup) => string
  /** Ce que dit l'infobulle d'une option inerte. */
  noData: string
}

/**
 * buildViewpointOptions range les joueurs du rejeu en sections de menu.
 *
 * L'ORDRE EST CELUI DE `groupByTeam` — camps dans l'ordre des désignateurs du film, joueurs dans
 * l'ordre du roster. Un ordre stable et reproductible, jamais l'itération d'une table.
 *
 * UN JOUEUR SANS AUCUN NOM N'EST PAS LISTÉ : ni la base ni le film ne le nomment, une option
 * vide ne serait pas cliquable de toute façon. C'est le cas des traces anonymes (caméras,
 * spectateurs de fin de partie), que `buildPlayers` laisse déjà sans propriétaire.
 */
export function buildViewpointOptions(
  players: readonly ReplayPlayer[],
  labels: ViewpointOptionLabels,
): ViewpointOptionGroup[] {
  const groups: ViewpointOptionGroup[] = []
  for (const groupe of groupByTeam(players)) {
    const options: ViewpointOption[] = []
    for (const p of groupe.players) {
      const nom = playerName(p)
      if (!nom) continue
      const label = stripBotSuffix(nom)
      const disabled = !p.board
      options.push({
        // LE PIÈGE DES BOTS EST ICI, EN UNE LIGNE (cf. l'en-tête, règle 1).
        value: p.board?.xuid ?? p.xuid,
        label,
        disabled,
        title: disabled ? labels.noData : label,
      })
    }
    if (options.length === 0) continue
    groups.push({ key: `camp:${groupe.team}`, label: labels.campLabelOf(groupe), options })
  }
  return groups
}
