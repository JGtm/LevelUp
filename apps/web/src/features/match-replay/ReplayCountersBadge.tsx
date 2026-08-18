/**
 * ReplayCountersBadge — LES COMPTEURS D'UNE FICHE, et les deux horloges qu'ils peuvent suivre.
 *
 * Extrait de ReplayTeams.tsx quand ce fichier a franchi le seuil de taille du dépôt en
 * recevant les compteurs vivants (même découpage que ReplayWeaponsRow).
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { MatchScoreboardRow } from '@/lib/api/types'

import type { PlayerCounters } from '@/lib/replay/scoreTimeline'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'

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
 * LE SCORE PERSONNEL n'apparaît QUE dans le cas vivant : la base ne le porte pas à cet
 * endroit, et un score de match posé à côté de frags à l'instant lu mélangerait deux
 * horloges sur une même ligne.
 */
export function ReplayCountersBadge({
  board,
  live,
  locale,
}: {
  board?: MatchScoreboardRow
  live: PlayerCounters | null
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  if (!live && !board) return null
  const parts: [number | null | undefined, string][] = live
    ? [
        [live.kills, 'success'],
        [live.deaths, 'destructive'],
        [live.assists, 'info'],
      ]
    : [
        [board?.kills, 'success'],
        [board?.deaths, 'destructive'],
        [board?.assists, 'info'],
      ]
  return (
    <span className="inline-flex shrink-0 items-baseline gap-1 font-mono text-[10px] tabular-nums">
      {live && (
        <span className="font-normal text-muted-foreground" title={t.playerScoreLive}>
          {live.score}
        </span>
      )}
      <span
        className="inline-flex items-baseline gap-0.5"
        title={live ? t.countersLive : t.countersMatch}
      >
        {parts.map(([v, token], i) => (
          <span key={token}>
            {i > 0 && <span className="opacity-50">/</span>}
            <span className="font-semibold" style={{ color: tokenCssVar(token as 'success') }}>
              {v ?? '?'}
            </span>
          </span>
        ))}
      </span>
    </span>
  )
}
