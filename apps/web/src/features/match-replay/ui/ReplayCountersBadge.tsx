/**
 * ReplayCountersBadge — LES COMPTEURS D'UNE FICHE, et les deux horloges qu'ils peuvent suivre.
 *
 * Extrait de ReplayTeams.tsx quand ce fichier a franchi le seuil de taille du dépôt en
 * recevant les compteurs vivants (même découpage que ReplayWeaponsRow).
 */
import { Fragment } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { fdaTone, matchFda, type FdaTone } from '@/lib/fda'
import { formatKDA } from '@/lib/formatters/number'

import type { PlayerCounters } from '@/lib/replay/scoreTimeline'

import type { CardGabarit } from '../model/cardGabarit'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'

/**
 * Part du token d'état dans le fond du triplet, en pour-cent. La même graduation que les
 * voiles de la fiche (`playerCardFx`) : assez pour se lire d'un coup d'oeil sur la tuile,
 * assez peu pour que les trois nombres colorés restent lisibles par-dessus.
 */
const FDA_TINT_PCT = 22

/**
 * LES DEUX GABARITS DE LA GRILLE (demande utilisateur du 2026-08-29, même formulation que
 * celle des armes le 2026-08-24 : « comme sur une grille pour que l'alignement soit le même
 * pour tous les joueurs »).
 *
 * `scoreCellW` — la cellule du score personnel, TOUJOURS rendue même vide SUR LE GABARIT
 * NORMAL : c'est elle qui tient l'alignement. Un joueur non publié n'a pas de score, et si sa
 * cellule disparaissait, son triplet glisserait de 30 px par rapport à celui du voisin publié
 * — exactement le décalage qu'on cherche à supprimer. Vide veut dire « pas de mesure », pas
 * « zéro ». Cette doctrine ne vaut que là où la cellule existe : sur la tuile compacte
 * (`showScore: false`, plan fiches compactes 2026-09-06) il n'y a PAS de cellule de score —
 * le triplet, seul et à droite de sa ligne, s'aligne de lui-même — et le score passe dans
 * l'infobulle du triplet avec sa valeur (`playerScoreLiveFmt`), sur un joueur publié seulement.
 *
 * `countCellW` — la cellule d'UN compteur. Ce sont des `min-width` et non des largeurs
 * fermes : deux chiffres tiennent partout (le cas ordinaire, donc l'alignement est tenu), et
 * un troisième chiffre pousse sa cellule plutôt que d'être rogné. Une valeur juste mais
 * tronquée serait pire qu'une colonne d'un pixel de trop.
 *
 * LES DEUX LARGEURS VIENNENT DU GABARIT DE LA FICHE (`model/cardGabarit.ts`, 2026-09-06 :
 * 30 / 15 en normal, 10 pour le compteur en compact) — ce composant ne porte plus de cote.
 */

/**
 * LA TYPOGRAPHIE DU TRIPLET SUIT SON ÉCHELLE (`countCellW`) par une table FERMÉE de littéraux
 * Tailwind (cf. cardGabarit.ts : une classe interpolée ne produit aucune règle). 15 : le
 * triplet d'aujourd'hui, classe pour classe — mono 10 px, gap 3, fond `px-1`. 10 : la ligne
 * de 12 px de la tuile compacte — mono 9 px, `leading-none`, gap 2, fond `px-[3px]` ; trois
 * cellules de 10, deux séparateurs, gaps et marges : ≈ 54 px.
 */
interface TripletTypo {
  /** La rangée : score et triplet. */
  row: string
  /** Le triplet lui-même, hors son fond. */
  cells: string
  /** Le fond FDA, posé seulement quand il y a un FDA. */
  tint: string
}

const TRIPLET_TYPO: Record<CardGabarit['countCellW'], TripletTypo> = {
  15: {
    row: 'inline-flex shrink-0 items-baseline gap-1 font-mono text-[10px] tabular-nums',
    cells: 'inline-flex items-baseline gap-[3px]',
    tint: 'rounded-[3px] px-1',
  },
  10: {
    row: 'inline-flex shrink-0 items-baseline gap-[2px] font-mono text-[9px] leading-none tabular-nums',
    cells: 'inline-flex items-baseline gap-[2px]',
    tint: 'rounded-[3px] px-[3px]',
  },
}

/** Le fond du triplet : le token du palier, dilué dans le fond de la tuile. */
function fdaTint(tone: FdaTone): string {
  return `color-mix(in srgb, ${tokenCssVar(tone)} ${FDA_TINT_PCT}%, transparent)`
}

/**
 * KdaBadge — FRAGS, MORTS, ASSISTANCES, chacun sa couleur. Trois nombres collés sans
 * distinction se lisent comme un seul nombre à trois chiffres.
 *
 * DEUX SOURCES, ET ELLES NE DISENT PAS LA MÊME CHOSE. Le film publie des compteurs À
 * L'INSTANT LU (schéma 12) : ils tiquent avec la lecture, et c'est ce qu'un rejeu doit
 * montrer. La base, elle, ne connaît que les totaux de FIN DE MATCH. Tant que le film
 * n'apparie pas ce joueur, la fiche garde les totaux de la base — le défaut que ce
 * commentaire signalait depuis l'origine (« ce ne sont pas des compteurs à l'instant lu »)
 * n'est corrigé que là où la mesure existe.
 *
 * UN JOUEUR NON PUBLIÉ N'AFFICHE DONC PAS DE ZÉRO : `live` vaut `null`, et le badge
 * retombe sur la base. Un joueur sans ligne de base NI compteurs n'affiche rien.
 *
 * LE SCORE PERSONNEL n'a de VALEUR que dans le cas vivant : la base ne le porte pas à cet
 * endroit, et un score de match posé à côté de frags à l'instant lu mélangerait deux
 * horloges sur une même ligne. Il vient EN TÊTE depuis le 2026-08-29 (demande utilisateur ;
 * il suivait les compteurs depuis l'option 2a du handoff 2026-08-27) — le score d'abord en
 * encre discrète, les trois nombres colorés ensuite.
 *
 * LA RANGÉE EST UNE GRILLE À CELLULES FIXES, même doctrine que celle des armes : la cellule
 * du score est TOUJOURS rendue, vide quand il n'y a pas de mesure, et chaque compteur a sa
 * largeur minimale. C'est ce qui met les triplets de toutes les fiches sur la même colonne —
 * sans cela, un joueur non publié (pas de score) ou un score à quatre chiffres décalait sa
 * ligne par rapport à ses voisines.
 *
 * LE FOND DU TRIPLET DIT SON FDA (demande utilisateur du 2026-08-29) : le net canonique
 * `frags + assistances/3 − morts` (cf. `lib/fda.ts`), en trois paliers — déficitaire sous
 * zéro, à l'équilibre de 0 à 1, bénéficiaire au-delà. Il suit LA MÊME horloge que les
 * nombres qu'il colore : sur un joueur publié il tique avec la lecture, sinon il vaut
 * celui des totaux de la base. La couleur ne remplace pas la mesure — l'infobulle donne
 * le FDA en toutes lettres, et un triplet incomplet n'a AUCUN fond plutôt qu'un fond
 * calculé sur une lacune.
 */
export function ReplayCountersBadge({
  board,
  live,
  locale,
  gabarit,
}: {
  board?: MatchScoreboardRow
  live: PlayerCounters | null
  locale: ReplayLocale
  /** Les cotes de la fiche : largeurs minimales de la cellule de score et d'un compteur. */
  gabarit: CardGabarit
}) {
  const t = REPLAY_TEXT[locale]
  if (!live && !board) return null
  const counters = live ?? { kills: board?.kills, deaths: board?.deaths, assists: board?.assists }
  const parts: [number | null | undefined, string][] = [
    [counters.kills, 'success'],
    [counters.deaths, 'destructive'],
    [counters.assists, 'info'],
  ]
  // LE FDA SUIT LA SOURCE AFFICHÉE, et ne la mélange jamais : sur un joueur publié il est
  // celui de l'INSTANT LU (il bouge avec la lecture), sinon celui des totaux de la base
  // (il ne bouge pas, et l'infobulle dit déjà lequel des deux on lit). Un compteur non lu
  // ne donne aucun FDA — donc aucun fond : une couleur est une affirmation.
  const fda = matchFda(counters)
  const label = live ? t.countersLive : t.countersMatch
  const typo = TRIPLET_TYPO[gabarit.countCellW]
  const tripletTitle = fda === null ? label : t.fdaTooltipFmt(label, formatKDA(fda, locale))
  // SANS CELLULE DE SCORE, le score vit dans l'infobulle du triplet — avec sa valeur, et sur
  // un joueur publié seulement (la base ne le porte pas à cet endroit).
  const title =
    !gabarit.showScore && live ? `${tripletTitle} · ${t.playerScoreLiveFmt(live.score)}` : tripletTitle
  return (
    <span className={typo.row}>
      {/* LE SCORE PERSONNEL D'ABORD (demande utilisateur du 2026-08-29) : il passe à GAUCHE
          des compteurs. Sa cellule est rendue même sans mesure — vide, sans infobulle — et
          c'est ce qui aligne les triplets d'une colonne à l'autre. */}
      {gabarit.showScore && (
        <span
          className="shrink-0 text-right font-normal text-muted-foreground"
          style={{ minWidth: gabarit.scoreCellW }}
          title={live ? t.playerScoreLive : undefined}
        >
          {live ? live.score : ''}
        </span>
      )}
      <span
        className={`${typo.cells} ${fda === null ? '' : typo.tint}`}
        style={fda === null ? undefined : { background: fdaTint(fdaTone(fda)) }}
        title={title}
      >
        {parts.map(([v, token], i) => (
          <Fragment key={token}>
            {i > 0 && <span className="opacity-[.35]">/</span>}
            <span
              className="inline-block text-center font-bold"
              style={{ color: tokenCssVar(token as 'success'), minWidth: gabarit.countCellW }}
            >
              {v ?? '?'}
            </span>
          </Fragment>
        ))}
      </span>
    </span>
  )
}
