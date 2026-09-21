/**
 * SquadRangeRolesTape — la « bande des rôles » (forme 2 de la maquette du 2026-09-21), le
 * repli fermé de la carte « Rôles de portée ».
 *
 * Une ligne par joueur, un jeton par match, coloré par le rôle LU SUR LA FENÊTRE GLISSANTE
 * de `FENETRE_ROLE` matchs — jamais par l'écart du match seul. Les premiers matchs d'un
 * joueur (fenêtre incomplète) et les matchs creux portent un jeton neutre : un rôle deviné
 * n'est pas un rôle. Aucune valeur écrite : la bande dit la STABILITÉ, le nuage dit
 * l'amplitude.
 *
 * TROIS TEINTES D'UNE SEULE ENCRE, du plus clair au plus marqué (`tokenTone`) : le rôle est
 * ORDINAL (du plus près au plus loin), et une rampe ordonnée se lit sans réapprendre la
 * légende à chaque bloc. Aucune autre encre n'est empruntée (skill color-tokens).
 *
 * PAS `OutcomeSequenceTape` : son contrat est soudé à l'issue de match (union
 * `OutcomeValue`, regroupement RLE par issue, crochets au-dessus pour les victoires et
 * en-dessous pour les défaites, encoches de dominance, quatre libellés fixes). Lui passer
 * une échelle ordinale quelconque, c'est le réécrire pour ses cinq consommateurs — hors
 * périmètre et risqué. Ici : ni RLE, ni crochet, ni encoche, une ligne PAR JOUEUR, et donc
 * du DOM, sans moteur de graphe.
 */
import { resolveToken, tokenTone } from '@/lib/accessibility'

import { ROLES_ORDONNES, type RoleDePortee, type SeriePortee } from './squadRangeRoles.logic'
import type { SquadRangeRolesText } from './squadRangeRolesStrings'

/** Écart de clarté de chaque rôle par rapport à l'encre de base — la rampe ordinale. */
const PAS_DE_TON: Record<RoleDePortee, number> = { front: 0, polyvalent: 0.12, sniper: 0.24 }

/** L'encre unique dont les trois rôles sont des tons. */
const ENCRE_ROLES = 'info' as const

function roleTone(role: RoleDePortee): string {
  return tokenTone(resolveToken(ENCRE_ROLES), PAS_DE_TON[role])
}

export interface SquadRangeRolesTapeProps {
  series: SeriePortee[]
  /** Rôle par joueur (xuid) et par point, dans l'ordre des points de la série. */
  roles: Map<string, (RoleDePortee | null)[]>
  /** Étiquettes des matchs (« #N · carte »), indexées par ordre du match. */
  categories: string[]
  t: SquadRangeRolesText
}

export function SquadRangeRolesTape({ series, roles, categories, t }: SquadRangeRolesTapeProps) {
  return (
    <div className="space-y-2" data-testid="squad-portee-bande">
      {series.map((serie) => (
        <div key={serie.xuid} className="flex items-center gap-2">
          <span className="w-28 shrink-0 truncate text-xs font-medium text-foreground">
            {serie.gamertag}
          </span>
          <div className="flex flex-1 flex-wrap gap-0.5">
            {serie.points.map((point, i) => {
              const role = roles.get(serie.xuid)?.[i] ?? null
              const libelle = role ? t.bandes[role] : t.tapeNoRole
              return (
                <span
                  key={point.matchId}
                  className={`h-4 w-5 rounded-sm${role ? '' : ' bg-muted'}`}
                  data-testid="squad-portee-jeton"
                  data-role={role ?? 'aucun'}
                  title={`${categories[point.ordre] ?? ''} — ${libelle}`}
                  style={role ? { backgroundColor: roleTone(role) } : undefined}
                />
              )
            })}
          </div>
        </div>
      ))}
      <div className="flex flex-wrap items-center gap-3 pt-1 text-2xs text-muted-foreground">
        {ROLES_ORDONNES.map((role) => (
          <span key={role} className="flex items-center gap-1.5">
            <span
              className="inline-block h-3 w-3 rounded-sm"
              style={{ backgroundColor: roleTone(role) }}
            />
            {t.bandes[role]}
          </span>
        ))}
        <span className="flex items-center gap-1.5">
          <span className="inline-block h-3 w-3 rounded-sm bg-muted" />
          {t.tapeNoRole}
        </span>
      </div>
    </div>
  )
}
