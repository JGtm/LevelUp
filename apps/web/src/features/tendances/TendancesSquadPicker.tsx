// cross-feature-allow: le sélecteur de composition de la page Escouade (`SquadCompositionPicker`),
// sa liste légère de coéquipiers (`useCompositionOptions`) et ses libellés — le MÊME sélecteur que
// l'Escouade, sans second jeu de libellés ni de couleurs.
/**
 * TendancesSquadPicker — la sélection de l'escouade de la vue Escouade : le sélecteur de composition
 * de la page Escouade (`SquadCompositionPicker`) et la case « Composition stricte » au gabarit exact
 * de `SquadFilterBar`.
 *
 * L'ÉTAT VIT DANS LA PAGE (`TendancesTab`, via `useTendancesSquadSelection`) : il est lu et écrit
 * dans le stockage par joueur de la page Escouade, la sélection est donc la même des deux côtés.
 *
 * COÉQUIPIERS PROPOSÉS : la page Escouade les tire de sa réponse complète (requête lourde) ; ici
 * `useCompositionOptions` reconstruit la même liste (amis déclarés, bots exclus) à partir de lectures
 * légères. LES XUID viennent de la réponse des tendances (`members` : le joueur principal en
 * premier, puis les coéquipiers) : ils alimentent « Enregistrer » et l'exclusion du joueur dans le
 * roster d'une escouade enregistrée. Tant que la réponse n'est pas là, `members` est vide et
 * « Enregistrer » reste inactif.
 */
import { useMemo } from 'react'

import { getSquadText } from '@/features/squad/i18n'
import { SquadCompositionPicker } from '@/features/squad/SquadCompositionPicker'
import { useCompositionOptions } from '@/features/squad/useCompositionOptions'
import type { TrendsMember } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

export interface TendancesSquadPickerProps {
  playerSlug: string
  locale: Locale
  /** Coéquipiers choisis (sans le joueur lui-même). */
  selected: string[]
  onChange: (next: string[]) => void
  /** Option « composition stricte ». */
  exactComposition: boolean
  onExactCompositionChange: (value: boolean) => void
  /** Membres de la réponse (joueur principal en premier) ; vide tant qu'elle n'est pas là. */
  members: readonly TrendsMember[]
}

export function TendancesSquadPicker({
  playerSlug,
  locale,
  selected,
  onChange,
  exactComposition,
  onExactCompositionChange,
  members,
}: TendancesSquadPickerProps) {
  const t = getSquadText(locale)
  const { options } = useCompositionOptions(playerSlug)
  const selectedRows = useMemo(
    () => members.slice(1).map(({ gamertag, xuid }) => ({ gamertag, xuid })),
    [members],
  )

  return (
    <div className="flex flex-wrap items-center gap-1.5" data-testid="tendances-squad-picker">
      <SquadCompositionPicker
        playerSlug={playerSlug}
        locale={locale}
        selected={selected}
        onChange={onChange}
        options={options}
        selectedRows={selectedRows}
        currentPlayerXuid={members[0]?.xuid ?? ''}
      />

      {selected.length > 0 && (
        <label
          className="shrink-0 inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-input bg-background px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-muted"
          title={t.filter.exactCompositionTitle}
        >
          <input
            type="checkbox"
            className="h-3 w-3 rounded border-input text-primary focus:ring-1 focus:ring-ring"
            checked={exactComposition}
            onChange={(event) => onExactCompositionChange(event.target.checked)}
          />
          {t.filter.exactComposition}
        </label>
      )}
    </div>
  )
}
