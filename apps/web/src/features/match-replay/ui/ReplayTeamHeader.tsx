/**
 * ReplayTeamHeader — LE TITRE D'UNE COLONNE D'ÉQUIPE, dans les fiches du rejeu.
 *
 * UN BANDEAU HUD AU-DESSUS DE LA COLONNE, PLUS UN EN-TÊTE DANS LA CARTE (option 2a du
 * handoff 2026-08-27) : la carte de colonne n'existe plus — les fiches sont des tuiles
 * autonomes — et le nom d'équipe est SORTI du conteneur : un liseré de camp, un dégradé qui
 * s'éteint vers la droite, une typo mono espacée. Le STYLE du bandeau vit dans `hudBand.ts`,
 * partagé avec le titre du fil des éliminations : même objet visuel, une seule écriture
 * (règle CLAUDE.md n°6).
 *
 * CE QU'IL CORRIGE (demande utilisateur du 2026-08-16 : « chaque équipe devra retrouver son
 * nom (Équipe Cobra / Équipe Eagle sur le scoreboard, sans réinventer la roue ») : la colonne
 * affichait `t0` / `t1` bruts, c'est-à-dire l'identifiant de transport du backend. Le libellé
 * lui arrive tout fait : c'est le nom du CAMP DU FILM (`campLabel`, `lib/replay/replayCamps.ts`),
 * la cascade du scoreboard sur la feuille de ses occupants, « Équipe N » de son désignateur quand
 * la feuille se tait — jamais « sans équipe » : une colonne est toujours un camp du film.
 *
 * LA COULEUR EST CELLE DES DEUX AUTRES PANNEAUX (décision D1 amendée) : `team-ally` /
 * `team-enemy`, les tokens que les réglages d'accessibilité peuvent surcharger. Un point bleu
 * sur la carte et un titre rouge pour la même équipe seraient une page cassée.
 *
 * UN CAMP DONT L'ENCRE N'EST PAS CONNUE N'EMPRUNTE AUCUNE DES DEUX COULEURS : liseré `border`,
 * fond à l'encre du thème, texte `muted-foreground` — quand aucun de ses occupants n'a de côté
 * de feuille, ou qu'aucun n'est reconnu au scoreboard. L'allégeance est une information, pas un
 * défaut d'affichage à combler.
 *
 * LE TITRE NE PORTE PLUS AUCUN NOMBRE (demande utilisateur du 2026-08-24 : « pas besoin de
 * mettre le score et le deuxième chiffre à côté du nom de l'équipe ») : le score vivant, la
 * manche et l'effectif sont partis — le score des deux camps vit au BANDEAU au-dessus du
 * terrain (ReplayScoreBanner), le seul endroit où le regard le cherche pendant la lecture.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { XuidMeta } from '@/features/match-view/xuidMeta'

import { HUD_BAND_CLASS, hudBandStyle } from '../model/hudBand'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'

/**
 * allyOfGroup dit de quel côté est un groupe : `true` allié, `false` adverse, `null` quand
 * on ne sait pas — camp sans côté de feuille, ou aucun joueur du groupe reconnu au scoreboard.
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
  /** Le nom du camp, déjà résolu par la colonne (`campLabel`). */
  label: string
  /** Les joueurs qui tiennent une place de CE camp à l'image lue : ils disent son allégeance. */
  players: readonly ReplayPlayer[]
  /** Le côté de feuille du camp (`ReplayCamp.side`) — l'encre seulement, jamais l'appartenance. */
  side: string | null
  xuidMeta?: XuidMeta
}

export function ReplayTeamHeader({ label, players, side, xuidMeta }: Props) {
  const ally = allyOfGroup(players, side, xuidMeta)
  const accent = ally === null ? null : tokenCssVar(ally ? 'team-ally' : 'team-enemy')
  return (
    <h3
      className={`${HUD_BAND_CLASS} flex items-baseline ${
        accent ? 'text-foreground' : 'text-muted-foreground'
      }`}
      style={hudBandStyle(accent)}
    >
      <span className="min-w-0 flex-1 truncate">{label}</span>
    </h3>
  )
}
