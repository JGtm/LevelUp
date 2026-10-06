// cross-feature-allow: le sélecteur d'escouade de la page Escouade (`useSquadPresets`, couleurs
// de joueur, libellés) et la liste légère de coéquipiers de
// l'onglet Tactique (`useCoequipierOptions`) — le MÊME sélecteur que l'Escouade, sans second
// jeu de libellés ni de couleurs.
/**
 * TendancesSquadPicker — la sélection de l'escouade de la vue Escouade : le `GamertagCombobox` de
 * la page Escouade (pastille du joueur en tête, trois coéquipiers au plus, couleurs de joueur,
 * escouades enregistrées par `useSquadPresets`) et la case « Composition stricte » au gabarit
 * exact de `SquadFilterBar`.
 *
 * L'ÉTAT VIT DANS LA PAGE (`TendancesTab`, via `useTendancesSquadSelection`) : il est lu et écrit
 * dans le stockage par joueur de la page Escouade, la sélection est donc la même des deux côtés.
 *
 * COÉQUIPIERS FRÉQUENTS : la page Escouade les tire de sa réponse complète (requête lourde) ;
 * ici on lit la liste légère des joueurs croisés comme coéquipiers (`useCoequipierOptions`).
 * LES XUID viennent de la réponse des tendances (`members` : le joueur principal en premier,
 * puis les coéquipiers) : ils alimentent « Enregistrer » et l'exclusion du joueur dans le roster
 * d'une escouade enregistrée. Tant que la réponse n'est pas là, `members` est vide et
 * « Enregistrer » reste inactif.
 */
import { useMemo } from 'react'

import { GamertagCombobox } from '@/components/ui/GamertagCombobox'
import { SQUAD_MAIN_PLAYER_TOKEN, MAX_SELECTION, getSquadTeammateColors } from '@/features/squad/colors'
import { getSquadText } from '@/features/squad/i18n'
import { useSquadPresets } from '@/features/squad/useSquadPresets'
import { useCoequipierOptions } from '@/features/tactical/queries'
import { tokenCssVar } from '@/lib/accessibility'
import type { TrendsMember } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { useAppShellStore } from '@/stores/appShellStore'

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
  const hasLinkedIdentity = useAppShellStore((s) => !!s.linkedHaloIdentity)
  const { options } = useCoequipierOptions(playerSlug)
  const colors = useMemo(() => getSquadTeammateColors(MAX_SELECTION), [])
  const selectedRows = useMemo(
    () => members.slice(1).map(({ gamertag, xuid }) => ({ gamertag, xuid })),
    [members],
  )
  const {
    presetGroups,
    footer,
    onClose: onPresetClose,
  } = useSquadPresets({
    playerSlug,
    currentPlayerXuid: members[0]?.xuid ?? '',
    hasLinkedIdentity,
    locale,
    selectedRows,
  })

  return (
    <div className="flex flex-wrap items-center gap-1.5" data-testid="tendances-squad-picker">
      <GamertagCombobox
        compact
        leadingPill={{ label: playerSlug, color: tokenCssVar(SQUAD_MAIN_PLAYER_TOKEN) }}
        selected={selected}
        onChange={onChange}
        max={MAX_SELECTION}
        frequentOptions={options}
        colors={colors}
        excludeGamertag={playerSlug}
        placeholder={t.selection.placeholder(options.length)}
        presetGroups={presetGroups}
        onLoadPreset={(gts) =>
          // Un roster ancien peut contenir le joueur lui-même : on le retire avant sélection.
          onChange(
            gts.filter((g) => g.toLowerCase() !== playerSlug.toLowerCase()).slice(0, MAX_SELECTION),
          )
        }
        footer={footer}
        onClose={onPresetClose}
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
