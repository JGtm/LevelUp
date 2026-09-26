/**
 * usageCardTitle.tsx — LE BANDEAU DE TITRE D'UNE CARTE D'USAGE, et il n'y en a qu'un.
 *
 * UN SEUL MÉCANISME D'AIDE (ajustement du 2026-09-21, D7/§7). Le bloc en portait DEUX :
 * `HeaderLabelTooltip` sur le libellé (Sessions, Synthèse, Escouade) — une aide invisible,
 * qui ne s'ouvre que si l'on survole un titre sans rien qui l'annonce — et, sous les
 * grilles, des notes et des pieds de carte en toutes lettres. Les deux fusionnent ici dans
 * UNE infobulle (i) VISIBLE, posée à droite du titre, qui prend plusieurs paragraphes :
 * l'aide de lecture, les notes de mesure et la couverture de la carte.
 *
 * Source unique des DEUX appelants (`session-detail/SessionUsageShared.tsx` et
 * `EquipmentUsageSection.tsx`) — CLAUDE.md n°6. Le rendu lui-même est délégué au helper
 * transverse `titleWithInfo` (réconciliation des lots A2 et G, 2026-09-21) : ce module ne
 * fait que mettre en paragraphes.
 */
import type { ReactNode } from 'react'

import { titleWithInfo } from '@/components/ui/title-with-info'

/**
 * Rend l'habillage de titre attendu par `SectionCard.titleAdornment` : le libellé, puis
 * l'icône (i) quand il reste au moins un paragraphe à dire. Les entrées vides
 * (`null`/`undefined`/chaîne vide) sont écartées — une infobulle vide ne se pose pas.
 */
export function usageCardTitle(
  ...paragraphs: (string | null | undefined)[]
): (label: string) => ReactNode {
  const kept = paragraphs.filter((p): p is string => typeof p === 'string' && p.length > 0)
  const content =
    kept.length > 0 ? (
      <div className="space-y-2">
        {kept.map((p) => (
          <p key={p}>{p}</p>
        ))}
      </div>
    ) : null
  return titleWithInfo(content, { iconClass: 'w-3.5 h-3.5' })
}
