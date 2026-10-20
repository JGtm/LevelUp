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
 * sur la carte et un titre rouge pour la même équipe seraient une page cassée. ELLE VIENT DU
 * FILM (2026-10-06) : l'allégeance du CAMP — son désignateur comparé à l'équipe du film du
 * joueur regardé (`FilmAllegiance.ofTeam`) —, la même que celle des pions de ses joueurs. Elle
 * se lisait sur les occupants présents reconnus à la feuille : un camp de bots, que la feuille
 * ne reconnaissait pas, restait neutre.
 *
 * UN CAMP DONT L'ENCRE N'EST PAS CONNUE N'EMPRUNTE AUCUNE DES DEUX COULEURS : liseré `border`,
 * fond à l'encre du thème, texte `muted-foreground` — quand le joueur regardé n'a pas d'équipe
 * du film, ou que le camp n'en est pas un (`-1`, mode sans camps). L'allégeance est une
 * information, pas un défaut d'affichage à combler.
 *
 * LE TITRE NE PORTE PLUS AUCUN NOMBRE (demande utilisateur du 2026-08-24 : « pas besoin de
 * mettre le score et le deuxième chiffre à côté du nom de l'équipe ») : le score vivant, la
 * manche et l'effectif sont partis — le score des deux camps vit au BANDEAU au-dessus du
 * terrain (ReplayScoreBanner), le seul endroit où le regard le cherche pendant la lecture.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { Allegiance } from '@/lib/replay/filmAllegiance'

import { HUD_BAND_CLASS, hudBandStyle } from '../model/hudBand'

interface Props {
  /** Le nom du camp, déjà résolu par la colonne (`campLabel`). */
  label: string
  /** L'allégeance du camp vue du joueur regardé (`FilmAllegiance.ofTeam`) ; `null` = encre neutre. */
  ally: Allegiance
}

export function ReplayTeamHeader({ label, ally }: Props) {
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
