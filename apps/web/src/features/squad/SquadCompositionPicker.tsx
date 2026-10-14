/**
 * SquadCompositionPicker — LE sélecteur de composition d'escouade, partagé par toutes les pages qui
 * choisissent des coéquipiers : Escouade (`SquadFilterBar`), Tendances (`TendancesSquadPicker`),
 * Ascension › Tactique (`TacticalFilterBar`).
 *
 *   [joueur consulté] [coéquipiers choisis…] [champ de recherche]
 *
 * Le `GamertagCombobox` compact, avec la pastille du joueur consulté en tête (non supprimable),
 * trois coéquipiers au plus aux couleurs de joueur de l'Escouade, et en tête du popover les
 * escouades enregistrées et les groupes (`useSquadPresets` : charger, enregistrer, gérer).
 *
 * CE QUE L'APPELANT FOURNIT : la liste proposée (`options`) — la page Escouade a la sienne dans sa
 * réponse, les autres passent par `useCompositionOptions` — et les lignes des coéquipiers choisis
 * avec leur xuid (pour « Enregistrer la compo »). Un appelant qui ne sait traduire que certaines
 * sources les restreint (`sources`, `allowFreeInput`).
 *
 * Garde-rail : `squadCompositionPicker.guard.test.ts` — aucun autre fichier ne monte les escouades
 * enregistrées dans un `GamertagCombobox`.
 */
import {
  GamertagCombobox,
  type GamertagSuggestionSource,
} from '@/components/ui/GamertagCombobox'
import { tokenCssVar } from '@/lib/accessibility'
import type { TeammateOption } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { useAppShellStore } from '@/stores/appShellStore'

import { getSquadTeammateColors, MAX_SELECTION, SQUAD_MAIN_PLAYER_TOKEN } from './colors'
import { getSquadText } from './i18n'
import { useSquadPresets, type SquadPresetRow } from './useSquadPresets'

const COULEURS = getSquadTeammateColors(MAX_SELECTION)

export interface SquadCompositionPickerProps {
  playerSlug: string
  locale: Locale
  /** Gamertags des coéquipiers choisis (sans le joueur consulté). */
  selected: string[]
  onChange: (next: string[]) => void
  /** Coéquipiers proposés (« Coéquipiers fréquents » du popover). */
  options: TeammateOption[]
  /** Coéquipiers choisis avec leur xuid — alimente « Enregistrer la compo ». */
  selectedRows: readonly SquadPresetRow[]
  /** Xuid du joueur consulté — l'exclut du roster d'une escouade enregistrée. */
  currentPlayerXuid: string
  /** Playlists et modes du filtre courant : remontent les escouades qui y jouent. */
  activeContextLabels?: string[]
  onAddAsFriend?: (gamertag: string) => void
  sources?: readonly GamertagSuggestionSource[]
  allowFreeInput?: boolean
  /** Demande d'ouverture du popover (cf. `GamertagCombobox.openRequest`). */
  openRequest?: number
}

export function SquadCompositionPicker({
  playerSlug,
  locale,
  selected,
  onChange,
  options,
  selectedRows,
  currentPlayerXuid,
  activeContextLabels,
  onAddAsFriend,
  sources,
  allowFreeInput,
  openRequest,
}: SquadCompositionPickerProps) {
  const t = getSquadText(locale)
  const hasLinkedIdentity = useAppShellStore((s) => !!s.linkedHaloIdentity)
  const { presetGroups, footer, onClose } = useSquadPresets({
    playerSlug,
    currentPlayerXuid,
    hasLinkedIdentity,
    locale,
    selectedRows,
    activeContextLabels,
  })

  return (
    <GamertagCombobox
      compact
      leadingPill={{ label: playerSlug, color: tokenCssVar(SQUAD_MAIN_PLAYER_TOKEN) }}
      selected={selected}
      onChange={onChange}
      max={MAX_SELECTION}
      frequentOptions={options}
      colors={COULEURS}
      excludeGamertag={playerSlug}
      placeholder={t.selection.placeholder(options.length)}
      onAddAsFriend={onAddAsFriend}
      sources={sources}
      allowFreeInput={allowFreeInput}
      presetGroups={presetGroups}
      onLoadPreset={(gts) =>
        // Un roster ancien peut contenir le joueur consulté : il est déjà la pastille de tête.
        onChange(
          gts.filter((g) => g.toLowerCase() !== playerSlug.toLowerCase()).slice(0, MAX_SELECTION),
        )
      }
      footer={footer}
      onClose={onClose}
      openRequest={openRequest}
    />
  )
}
