/**
 * ReplayTeamHeader — LE TITRE D'UNE COLONNE D'ÉQUIPE, dans les fiches du rejeu.
 *
 * CE QU'IL CORRIGE (demande utilisateur du 2026-08-16 : « chaque équipe devra retrouver son
 * nom (Équipe Cobra / Équipe Eagle sur le scoreboard, sans réinventer la roue ») : la colonne
 * affichait `t0` / `t1` bruts, c'est-à-dire l'identifiant de transport du backend. Le libellé
 * vient désormais de `resolveTeamLabel` — LA cascade du dépôt, celle du scoreboard et des
 * objectifs — sans qu'une troisième copie soit écrite ici (règle CLAUDE.md n°6).
 *
 * LA COULEUR EST CELLE DES DEUX AUTRES PANNEAUX (décision D1 amendée) : `team-ally` /
 * `team-enemy`, les tokens que les réglages d'accessibilité peuvent surcharger. Un point bleu
 * sur la carte et un titre rouge pour la même équipe seraient une page cassée.
 *
 * UN GROUPE SANS CAMP CONNU N'EMPRUNTE AUCUNE DES DEUX COULEURS : encre et liseré neutres.
 * Le camp est une information, pas un défaut d'affichage à combler.
 *
 * LE TITRE NE PORTE PLUS AUCUN NOMBRE (demande utilisateur du 2026-08-24 : « pas besoin de
 * mettre le score et le deuxième chiffre à côté du nom de l'équipe ») : le score vivant, la
 * manche et l'effectif sont partis — le score des deux camps vit au BANDEAU au-dessus du
 * terrain (ReplayScoreBanner), le seul endroit où le regard le cherche pendant la lecture.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { XuidMeta } from '@/features/match-view/xuidMeta'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { ReplayPlayer } from './rosterLogic'

/** Part du fond qu'occupe la couleur d'équipe : assez pour teinter, jamais pour crier. */
const TINT_PCT = 14

/**
 * allyOfGroup dit de quel côté est un groupe : `true` allié, `false` adverse, `null` quand
 * on ne sait pas — camp non transmis, ou aucun joueur du groupe reconnu au scoreboard.
 * « Allié » veut dire « du côté du joueur dont on regarde la page » (cf. xuidMeta.ts).
 */
function allyOfGroup(
  players: readonly ReplayPlayer[],
  side: string | null,
  xuidMeta: XuidMeta | undefined,
): boolean | null {
  if (side === null || !xuidMeta) return null
  const known = players.filter((p) => xuidMeta.has(p.xuid))
  if (known.length === 0) return null
  return known.some((p) => xuidMeta.get(p.xuid)?.ally === true)
}

interface Props {
  /** Les joueurs de CE groupe : leurs lignes de scoreboard portent le libellé du backend. */
  players: readonly ReplayPlayer[]
  side: string | null
  xuidMeta?: XuidMeta
  locale: ReplayLocale
}

export function ReplayTeamHeader({ players, side, xuidMeta, locale }: Props) {
  const t = REPLAY_TEXT[locale]
  const rows = players
    .map((p) => p.board)
    .filter((r): r is MatchScoreboardRow => r !== undefined)
  const label = resolveTeamLabel(rows, side, t)
  const ally = allyOfGroup(players, side, xuidMeta)
  const accent = ally === null ? null : tokenCssVar(ally ? 'team-ally' : 'team-enemy')
  return (
    <h3
      className={`mb-2 flex shrink-0 items-baseline rounded-sm px-1.5 py-1 text-[11px] font-semibold uppercase tracking-wide ${
        accent ? 'text-foreground' : 'border-l-[3px] border-border text-muted-foreground'
      }`}
      style={
        accent
          ? {
              borderLeft: `3px solid ${accent}`,
              background: `color-mix(in srgb, ${accent} ${TINT_PCT}%, transparent)`,
            }
          : undefined
      }
    >
      <span className="min-w-0 flex-1 truncate">{label}</span>
    </h3>
  )
}
