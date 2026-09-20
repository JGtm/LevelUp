/**
 * SquadFilterBar — barre de filtres unifiée de la section Escouade (sticky) +
 * rail de navigation période/session.
 *
 *   [joueur actif] [coéquipiers▾] [Filtres▾] [Saison▾] [Période▾]
 *   [Sessions escouade▾] [composition stricte] [Analyser] [Réinitialiser]
 *
 * Extraite de `SquadLayout` le 2026-09-20 : la barre est désormais un ENFANT du
 * layout, comme `FilterOmnibar` est un frère du contenu sur les autres pages.
 * Tout l'état transitoire (période et cascade en attente, popover ouvert,
 * preview live) vit ici — cf. `useSquadFilterBarState` pour le pourquoi. Le
 * layout ne re-rend donc plus l'arbre de la page (et ses graphes ECharts) quand
 * on touche un filtre qui n'a pas encore été commité par « Analyser ».
 *
 * Ce qui s'applique EN DIRECT (coéquipiers, sessions pickées, composition
 * stricte) reste piloté par le layout et descend ici par props : c'est lui qui
 * interroge `useTeammates` avec ces valeurs.
 */
import type { ReactNode } from 'react'

import { GamertagCombobox } from '@/components/ui/GamertagCombobox'
import { SessionMultiSelect } from '@/components/ui/SessionMultiSelect'
import { FiltresPill, PeriodePill, SaisonPill, DEFAULT_PERIOD } from '@/components/shell/FilterOmnibar'
import { PeriodSessionRail } from '@/components/shell/PeriodSessionRail'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { tokenCssVar } from '@/lib/accessibility'
import { useSquadFilterStore } from '@/stores/squadFilterStore'
import type { Locale } from '@/lib/i18n/locale'
import type { SessionLabelEntry, TeammateOption, TeammateRow } from '@/lib/api/types'

import { getSquadText } from './i18n'
import { getSquadTeammateColors, MAX_SELECTION, SQUAD_MAIN_PLAYER_TOKEN } from './colors'
import { seasonToPeriod } from './useActiveSeason'
import { useSquadFilterBarState } from './useSquadFilterBarState'
import { useSquadPresets } from './useSquadPresets'

const CHART_COLORS = getSquadTeammateColors(MAX_SELECTION)

export interface SquadFilterBarProps {
  playerSlug: string
  locale: Locale
  hasLinkedIdentity: boolean
  /** Composition sélectionnée (appliquée en direct). */
  selectedGts: string[]
  setSelectedGts: (next: string[] | ((prev: string[]) => string[])) => void
  /** Coéquipiers fréquents proposés par la réponse teammates (`data.options`). */
  availableOptions: TeammateOption[]
  /** Lignes des coéquipiers confirmés — sert aux presets « Mes escouades ». */
  selectedRows: TeammateRow[]
  /** XUID absolu du joueur courant (résolu depuis header.player_cards). */
  currentPlayerXuid: string
  onAddFriendGamertag: (gamertag: string) => void
  /** Sessions de la composition courante + sélection multiple (appliquée en direct). */
  compositionSessions: SessionLabelEntry[]
  pickedSquadSessionLabels: string[]
  applySessionLabels: (labels: string[]) => void
  /** Option « composition stricte » (appliquée en direct). */
  hasTeammates: boolean
  exactComposition: boolean
  setExactComposition: (value: boolean) => void
  /** Bouton « Voir les matchs » construit par le layout (il a la liste des matchs). */
  browseButton: ReactNode
  /** Réinitialisation complète (filtres globaux + sessions pickées). */
  onReset: () => void
}

export function SquadFilterBar({
  playerSlug,
  locale,
  hasLinkedIdentity,
  selectedGts,
  setSelectedGts,
  availableOptions,
  selectedRows,
  currentPlayerXuid,
  onAddFriendGamertag,
  compositionSessions,
  pickedSquadSessionLabels,
  applySessionLabels,
  hasTeammates,
  exactComposition,
  setExactComposition,
  browseButton,
  onReset,
}: SquadFilterBarProps) {
  const t = getSquadText(locale)
  // Pas d'alias local `tCommon` ici : il en existe déjà deux dans le dépôt
  // (FilterOmnibar, SquadLayout) et la règle « ≤ 2 copies » interdit la 3e —
  // `formatMessage` est déjà le helper canonique, on l'appelle directement.
  const common = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  const {
    pendingCascade,
    pendingPeriod,
    setPendingCascade,
    setPendingPeriod,
    activePopover,
    togglePopover,
    closeAllPopovers,
    available,
    presetCounts,
    seasons,
    activeSeason,
    seasonCounts,
    cascadeCount,
    isDirty,
    analyser,
    activeContextLabels,
    getSessionCount,
    getSessionShownCount,
  } = useSquadFilterBarState({ playerSlug, locale, pickedSquadSessionLabels, compositionSessions })

  // Presets du combobox : escouades sauvegardées + groupes d'accès (charger un
  // roster), + footer de gestion (enregistrer / renommer / supprimer).
  const {
    presetGroups: squadPresetGroups,
    footer: squadPresetFooter,
    onClose: squadPresetOnClose,
  } = useSquadPresets({
    playerSlug,
    currentPlayerXuid,
    hasLinkedIdentity,
    locale,
    selectedRows,
    activeContextLabels,
  })

  return (
    <div className="sticky top-0 z-30 px-6" style={{ background: 'var(--background)' }}>
      <div className="flex min-h-10 items-center gap-1.5 overflow-visible border-b border-border py-1.5">

        {/* Joueur actif (pill de tête non-supprimable) + coéquipiers (multi-select
            compact inline, jusqu'à 3). La pill du joueur actif est rendue DANS le
            combobox (leadingPill) → même ligne flex que les pills coéquipiers, donc
            alignement vertical garanti. Le popover intègre les presets « Mes
            escouades » (charger/gérer une compo) et « Mes groupes ». */}
        <GamertagCombobox
          compact
          leadingPill={{ label: playerSlug, color: tokenCssVar(SQUAD_MAIN_PLAYER_TOKEN) }}
          selected={selectedGts}
          onChange={setSelectedGts}
          max={MAX_SELECTION}
          frequentOptions={availableOptions}
          colors={CHART_COLORS}
          excludeGamertag={playerSlug}
          placeholder={t.selection.placeholder(availableOptions.length)}
          onAddAsFriend={onAddFriendGamertag}
          presetGroups={squadPresetGroups}
          onLoadPreset={(gts) =>
            // Dédup vs la pill de tête (leadingPill = joueur courant) : un roster
            // legacy peut encore contenir le viewer → on le retire avant sélection.
            setSelectedGts(
              gts.filter((g) => g.toLowerCase() !== playerSlug.toLowerCase()).slice(0, MAX_SELECTION),
            )
          }
          footer={squadPresetFooter}
          onClose={squadPresetOnClose}
        />

        {/* Séparateur */}
        <div className="mx-0.5 h-5 w-px shrink-0 bg-border" aria-hidden />

        {/* Filtres cascade (playlists / modes / cartes / expérience) */}
        <FiltresPill
          open={activePopover === 'filtres'}
          onToggle={() => togglePopover('filtres')}
          onClose={closeAllPopovers}
          available={available ?? { playlists: [], modes: [], maps: [], experience_types: [] }}
          cascade={pendingCascade}
          cascadeCount={cascadeCount}
          onSetCascade={setPendingCascade}
        />

        {/* Saison (catalog TOML kind="season" — applique la fenêtre via setPendingPeriod) */}
        {seasons.length > 0 && (
          <SaisonPill
            open={activePopover === 'saison'}
            onToggle={() => togglePopover('saison')}
            onClose={closeAllPopovers}
            seasons={seasons}
            activeSeason={activeSeason}
            seasonCounts={seasonCounts}
            onSelectSeason={(s) => setPendingPeriod(seasonToPeriod(s))}
            onClear={() => setPendingPeriod(DEFAULT_PERIOD)}
          />
        )}

        {/* Période */}
        <PeriodePill
          open={activePopover === 'periode'}
          onToggle={() => togglePopover('periode')}
          onClose={closeAllPopovers}
          period={pendingPeriod}
          onSetPeriod={setPendingPeriod}
          presetCounts={presetCounts}
        />

        {/* Sessions escouade (multi-select par label) — composition-aware */}
        {compositionSessions.length > 0 && (
          <SessionMultiSelect
            sessions={compositionSessions}
            selected={pickedSquadSessionLabels}
            onChange={applySessionLabels}
            locale={locale}
            triggerClassName="flex items-center gap-1.5 rounded-md border border-input bg-background px-2.5 py-1 text-xs font-medium hover:bg-muted whitespace-nowrap transition-colors"
            getMatchCount={getSessionShownCount}
          />
        )}

        {/* Composition stricte — option cochée par défaut : la règle affichée est
            « exactement cette composition » ; la décocher élargit aux matchs
            commencés ensemble. Appliquée en direct (pas d'Analyser). */}
        {hasTeammates && (
          <label
            className="shrink-0 inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-input bg-background px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-muted"
            title={t.filter.exactCompositionTitle}
          >
            <input
              type="checkbox"
              className="h-3 w-3 rounded border-input text-primary focus:ring-1 focus:ring-ring"
              checked={exactComposition}
              onChange={(e) => setExactComposition(e.target.checked)}
            />
            {t.filter.exactComposition}
          </label>
        )}

        <div className="flex-1" />

        {/* Compteur de matchs + « Voir les matchs » : rendus dans le rail
            ci-dessous (zone centrale) pour éviter le doublon avec son compteur
            et décharger la barre. */}

        {/* Analyser — applique les filtres en attente (cascade + période).
            Les coéquipiers et sessions, eux, s'appliquent en direct. */}
        <button
          type="button"
          onClick={analyser}
          title={common('common.filters.apply_pending_title')}
          className={[
            'shrink-0 rounded-md px-2.5 py-1 text-xs font-medium transition-colors',
            isDirty
              ? 'bg-primary text-primary-foreground hover:bg-primary/90'
              : 'border border-input bg-background text-muted-foreground hover:bg-muted',
          ].join(' ')}
        >
          Analyser
        </button>

        {/* Réinitialiser — bouton (aligné visuellement sur Analyser). */}
        <button
          type="button"
          onClick={onReset}
          className="shrink-0 rounded-md border border-input bg-background px-2.5 py-1 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-destructive"
          title={common('common.filters.reset_title')}
        >
          {common('common.filters.reset_label')}
        </button>
      </div>
      {/* Rail de navigation période/session — placé DANS la barre sticky pour
          apparaître toujours juste sous les filtres Squad au scroll. Reçoit le
          bouton « Voir les matchs » + le compte composition (source unique,
          ADR 0033) en mode session unique via sessionCount. `matchCount`
          (tous les AUTRES modes : période/multi-session/all-time) ne
          reprend PLUS `totalAfter` — c'était la population du JOUEUR
          PRINCIPAL (/filters/resolve), pas celle de l'escouade affichée :
          exactement le défaut mesuré (7 vs 4) que l'option composition
          exacte pouvait aggraver. Aucun total composition fiable pour ces
          modes dans ce lot (composition_sessions n'est pas borné par la
          période/le multi-select) : on affiche 0 plutôt qu'un nombre faux. */}
      <PeriodSessionRail
        filterStore={useSquadFilterStore}
        trailing={browseButton}
        sessionCount={getSessionCount}
      />
    </div>
  )
}
