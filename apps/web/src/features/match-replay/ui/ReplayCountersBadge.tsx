/**
 * ReplayCountersBadge — LES COMPTEURS D'UNE FICHE, et les deux horloges qu'ils peuvent suivre.
 */
import { Fragment } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { fdaTone, matchFda, type FdaTone } from '@/lib/fda'
import { formatKDA } from '@/lib/formatters/number'

import type { PlayerCounters } from '@/lib/replay/scoreTimeline'

import type { CardGabarit } from '../model/cardGabarit'
import { countersShown } from '../model/playerCardReadings'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'

/**
 * Part du token d'état dans le fond du triplet, en pour-cent. La même graduation que les
 * voiles de la fiche (`playerCardFx`) : assez pour se lire d'un coup d'oeil sur la tuile,
 * assez peu pour que les trois nombres colorés restent lisibles par-dessus.
 */
const FDA_TINT_PCT = 22

/**
 * LA CELLULE D'UN COMPTEUR (demande utilisateur du 2026-08-29, même formulation que celle des armes le
 * 2026-08-24 : « comme sur une grille pour que l'alignement soit le même pour tous les joueurs »).
 *
 * `countCellW` — la cellule d'UN compteur, venue du gabarit de la tuile (`model/cardGabarit.ts`). Ce sont des
 * `min-width` et non des largeurs fermes : deux chiffres tiennent partout (le cas ordinaire, donc l'alignement
 * est tenu), et un troisième chiffre pousse sa cellule plutôt que d'être rogné. Une valeur juste mais tronquée
 * serait pire qu'une colonne d'un pixel de trop.
 *
 * LA TYPOGRAPHIE DU TRIPLET est écrite en littéraux Tailwind (cf. cardGabarit.ts : une classe interpolée ne
 * produit aucune règle) : la ligne de 12 px de la tuile compacte — mono 9 px, `leading-none`, gap 2, fond
 * `px-[3px]` ; trois cellules de 10, deux séparateurs, gaps et marges : environ 54 px.
 */
const TRIPLET_TYPO = {
  /** La rangée du triplet. */
  row: 'inline-flex shrink-0 items-baseline gap-[2px] font-mono text-[9px] leading-none tabular-nums',
  /** Le triplet lui-même, hors son fond. */
  cells: 'inline-flex items-baseline gap-[2px]',
  /** Le fond FDA, posé seulement quand il y a un FDA. */
  tint: 'rounded-[3px] px-[3px]',
} as const

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
 * LE SCORE PERSONNEL vit dans l'infobulle du triplet, avec sa valeur, sur un joueur publié seulement : la
 * base ne le porte pas à cet endroit, et un score de match posé à côté de frags à l'instant lu mélangerait
 * deux horloges. C'est LE score de la fiche du joueur (`score`, lu par `playerScoreAt` sous ses trois gardes) :
 * quand la garde refuse, la ligne du score n'est pas écrite, ni zéro ni tiret.
 *
 * LA RANGÉE EST UNE GRILLE À CELLULES FIXES, même doctrine que celle des armes : chaque compteur a sa largeur
 * minimale, ce qui met les triplets de toutes les fiches sur la même colonne.
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
  score,
  locale,
  gabarit,
}: {
  board?: MatchScoreboardRow
  live: PlayerCounters | null
  /** Le score personnel à l'instant lu, tel que la fiche l'écrit ; `null` = garde refusée, rien n'est écrit. */
  score: number | null
  locale: ReplayLocale
  /** Les cotes de la fiche : largeur minimale d'un compteur. */
  gabarit: CardGabarit
}) {
  const t = REPLAY_TEXT[locale]
  const counters = countersShown(live, board)
  if (!counters) return null
  const parts: [number | null | undefined, string][] = [
    [counters.kills, 'stat-kills'],
    [counters.deaths, 'stat-deaths'],
    [counters.assists, 'stat-assists'],
  ]
  // LE FDA SUIT LA SOURCE AFFICHÉE, et ne la mélange jamais : sur un joueur publié il est
  // celui de l'INSTANT LU (il bouge avec la lecture), sinon celui des totaux de la base
  // (il ne bouge pas, et l'infobulle dit déjà lequel des deux on lit). Un compteur non lu
  // ne donne aucun FDA — donc aucun fond : une couleur est une affirmation.
  const fda = matchFda(counters)
  const label = live ? t.countersLive : t.countersMatch
  const tripletTitle = fda === null ? label : t.fdaTooltipFmt(label, formatKDA(fda, locale))
  // LE SCORE vit dans l'infobulle du triplet, sur un joueur publié seulement ; garde refusée : aucune ligne.
  const title = live && score !== null ? `${tripletTitle} · ${t.playerScoreLiveFmt(score)}` : tripletTitle
  return (
    <span className={TRIPLET_TYPO.row}>
      <span
        className={`${TRIPLET_TYPO.cells} ${fda === null ? '' : TRIPLET_TYPO.tint}`}
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
