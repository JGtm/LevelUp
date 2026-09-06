/**
 * GamertagCombobox — sélecteur multi-gamertag avec fuzzy search.
 *
 * Affiche trois groupes priorisés :
 *  1. « Joueurs configurés »      — profils dans available_players (db_profiles)
 *  2. « Coéquipiers fréquents »   — options depuis l'API (encounter_count)
 *  3. « Autres joueurs »          — recherche serveur (xuid_aliases, fuzzy Jaro-Winkler)
 *
 * Logique de suggestion centralisée dans useGamertagSuggestions.
 * Autorise aussi la saisie libre d'un gamertag absent des suggestions.
 */
import { useRef, useState, useEffect } from 'react'
import type { TeammateOption } from '@/lib/api/types'
import { useGamertagSuggestions } from './useGamertagSuggestions'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

// ─── Props ──────────────────────────────────────────────────────────────────────

/**
 * Groupe de presets affiché en tête du popover (ex. « MES ESCOUADES »).
 *
 * Chaque option est un roster nommé chargé d'un clic dans la sélection (≠ ajout
 * d'un joueur) ; `trailing` permet au parent d'injecter des actions (renommer/
 * supprimer) sans coupler de logique métier dans ce composant générique.
 */
export interface ComboboxPresetGroup {
  key: string
  label: string
  options: Array<{
    id: string
    name: string
    subtitle?: string
    gamertags: string[]
    trailing?: React.ReactNode
  }>
}

export interface GamertagComboboxProps {
  /** Gamertags actuellement sélectionnés */
  selected: string[]
  onChange: (v: string[]) => void
  /** Nombre max de sélections (défaut : pas de limite) */
  max?: number
  /** Coéquipiers fréquents depuis l'API */
  frequentOptions?: TeammateOption[]
  /** Couleurs des pills (si non fourni, couleur neutre) */
  colors?: string[]
  /** Gamertag du joueur courant à exclure des suggestions */
  excludeGamertag?: string
  placeholder?: string
  /** Autoriser la saisie libre (gamertag hors listes) */
  allowFreeInput?: boolean
  /**
   * Optionnel — quand fourni, affiche un CTA secondaire "Ajouter X comme ami"
   * sous l'option saisie libre. Utilisé par SquadLayout pour déclencher
   * l'AddFriendModal sur saisie d'un gamertag hors top dropdown (§3 plan
   * Squad/Sessions overhaul).
   */
  onAddAsFriend?: (gamertag: string) => void
  /**
   * Mode compact — supprime le conteneur bordé pour un usage inline dans une
   * barre de filtres. Les pills sélectionnées et le champ de saisie flottent
   * directement dans le flux sans padding/border externes.
   */
  compact?: boolean
  /**
   * Presets (rosters nommés) affichés en tête du popover quand la recherche est
   * vide. Cliquer un preset REMPLACE la sélection (via onLoadPreset). Sert au
   * câblage « charger une escouade / un groupe » sans logique métier ici.
   */
  presetGroups?: ComboboxPresetGroup[]
  /** Charge un roster entier (remplace la sélection courante). */
  onLoadPreset?: (gamertags: string[]) => void
  /** Contenu rendu en pied de popover (actions de gestion : enregistrer…). */
  footer?: React.ReactNode
  /** Appelé quand le popover se ferme (clic-extérieur, Échap, chargement preset).
   *  Permet au parent de réinitialiser un état de gestion transitoire. */
  onClose?: () => void
  /**
   * Pill de tête non-supprimable (joueur actif), rendue dans la même ligne flex
   * que les pills sélectionnées → alignement vertical garanti. `color` est une
   * chaîne CSS prête à l'emploi (ex. tokenCssVar('compare-a')).
   */
  leadingPill?: { label: string; color: string }
  /**
   * Sources de suggestion PROPOSÉES. Par défaut : les quatre.
   *
   * Un appelant qui ne sait pas TRADUIRE toutes les sources doit pouvoir n'offrir que
   * celles qu'il traite : l'onglet Tactique envoie au serveur des XUIDs qu'il ne connaît
   * que pour ses coéquipiers fréquents, et proposer un joueur de l'annuaire l'aurait
   * conduit à refuser ensuite ce qu'il venait d'accepter. Accepter puis refuser est
   * toujours pire que ne pas proposer.
   *
   * Rétro-compatible : omettre la prop garde le comportement complet (SquadLayout,
   * Compare, Settings, Admin sont inchangés).
   */
  sources?: readonly GamertagSuggestionSource[]
}

/** Les quatre sources du popover, dans leur ordre d'affichage. */
export type GamertagSuggestionSource = 'configured' | 'frequent' | 'remote' | 'live'

const TOUTES_SOURCES: readonly GamertagSuggestionSource[] = [
  'configured',
  'frequent',
  'remote',
  'live',
]

// ─── Composant ──────────────────────────────────────────────────────────────────

export function GamertagCombobox({
  selected,
  onChange,
  max,
  frequentOptions = [],
  colors,
  excludeGamertag,
  placeholder = 'Rechercher un gamertag…',
  allowFreeInput = true,
  sources,
  onAddAsFriend,
  compact = false,
  presetGroups,
  onLoadPreset,
  footer,
  onClose,
  leadingPill,
}: GamertagComboboxProps) {
  const [query, setQuery] = useState('')
  const [isOpen, setIsOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  // ─── Fermeture click-outside ────────────────────────────────────────────────
  useEffect(() => {
    function handleMouseDown(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false)
      }
    }
    document.addEventListener('mousedown', handleMouseDown)
    return () => document.removeEventListener('mousedown', handleMouseDown)
  }, [])

  // Notifie le parent à chaque fermeture du popover (transition open → closed),
  // quel que soit le chemin (clic-extérieur, Échap, chargement preset). Permet
  // au parent (useSquadPresets) de réinitialiser son état de gestion transitoire.
  const onCloseRef = useRef(onClose)
  useEffect(() => {
    onCloseRef.current = onClose
  }, [onClose])
  const prevOpenRef = useRef(isOpen)
  useEffect(() => {
    if (prevOpenRef.current && !isOpen) onCloseRef.current?.()
    prevOpenRef.current = isOpen
  }, [isOpen])

  // ─── Suggestions via hook partagé ───────────────────────────────────────────

  const isAtMax = max != null && selected.length >= max
  const excludeGamertags = excludeGamertag ? [...selected, excludeGamertag] : selected

  const {
    configured: configuredTous,
    frequent: frequentTous,
    remote: remoteTous,
    isRemoteLoading: isRemoteLoadingBrut,
    hasAnyResult: hasAnyResultBrut,
    remoteAttempted,
    liveResults: liveResultsTous,
    isLiveLoading,
    liveAttempted,
    liveEmpty: liveEmptyBrut,
    triggerLiveSearch,
  } = useGamertagSuggestions({ query, frequentOptions, excludeGamertags })

  // Filtrage par SOURCE : une source non proposée n'apparaît pas, ne compte pas dans
  // « a-t-on trouvé quelque chose », et ne déclenche ni le repli Xbox ni ses messages.
  const sourcesActives = sources ?? TOUTES_SOURCES
  const configured = sourcesActives.includes('configured') ? configuredTous : []
  const frequent = sourcesActives.includes('frequent') ? frequentTous : []
  const remote = sourcesActives.includes('remote') ? remoteTous : []
  const liveResults = sourcesActives.includes('live') ? liveResultsTous : []
  const rechercheServeurProposee = sourcesActives.includes('remote')
  const repliXboxPropose = sourcesActives.includes('live')
  const isRemoteLoading = rechercheServeurProposee && isRemoteLoadingBrut
  const liveEmpty = repliXboxPropose && liveEmptyBrut
  const hasAnyResult = rechercheServeurProposee
    ? hasAnyResultBrut
    : configured.length > 0 || frequent.length > 0

  const trimmed = query.trim()
  const allSuggestedGts = new Set([
    ...configured.map((c) => c.gamertag),
    ...frequent.map((f) => f.gamertag),
    ...remote.map((r) => r.gamertag),
    ...liveResults.map((r) => r.gamertag),
  ])

  // L'option "ajouter en libre" n'apparaît que si :
  // - allowFreeInput actif
  // - query non-vide
  // - pas déjà sélectionné
  // - pas déjà dans les suggestions
  const canAddFree =
    allowFreeInput &&
    trimmed.length > 0 &&
    !selected.includes(trimmed) &&
    !allSuggestedGts.has(trimmed)

  const showEmptyMessage =
    trimmed.length > 0 &&
    (rechercheServeurProposee ? remoteAttempted : true) &&
    !isRemoteLoading &&
    !hasAnyResult

  // Bouton « Rechercher sur Xbox » (V72-24) : proposé quand la recherche locale a
  // répondu sans surfacer le joueur tapé, tant que le repli live n'a pas été lancé.
  const canSearchLive =
    repliXboxPropose &&
    trimmed.length > 0 &&
    !selected.includes(trimmed) &&
    !allSuggestedGts.has(trimmed) &&
    remoteAttempted &&
    !isRemoteLoading &&
    !liveAttempted

  // Presets (escouades / groupes) : affichés en tête quand la recherche est vide.
  const showPresets =
    trimmed.length === 0 && !!presetGroups && presetGroups.some((g) => g.options.length > 0)

  const hasDropdownContent =
    showPresets ||
    !!footer ||
    configured.length > 0 ||
    frequent.length > 0 ||
    remote.length > 0 ||
    liveResults.length > 0 ||
    canAddFree ||
    canSearchLive ||
    isRemoteLoading ||
    isLiveLoading ||
    liveEmpty ||
    showEmptyMessage

  // ─── Actions ────────────────────────────────────────────────────────────────

  function add(gamertag: string) {
    if (isAtMax) return
    if (selected.includes(gamertag)) return
    onChange([...selected, gamertag])
    setQuery('')
    inputRef.current?.focus()
  }

  function remove(gamertag: string) {
    onChange(selected.filter((g) => g !== gamertag))
  }

  // Charge un roster nommé (escouade/groupe) : REMPLACE la sélection.
  function loadPreset(gamertags: string[]) {
    onLoadPreset?.(gamertags)
    setQuery('')
    setIsOpen(false)
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Escape') {
      setIsOpen(false)
      return
    }
    if (e.key === 'Enter') {
      e.preventDefault()
      // Texte tapé hors suggestions → ajoute le texte exact (saisie libre).
      // Sinon, prend la 1re suggestion priorisée (configured > frequent > remote).
      if (canAddFree) {
        add(trimmed)
      } else {
        const first =
          configured[0]?.gamertag ?? frequent[0]?.gamertag ?? remote[0]?.gamertag
        if (first) add(first)
      }
      return
    }
    if (e.key === 'Backspace' && query === '' && selected.length > 0) {
      remove(selected[selected.length - 1])
    }
  }

  // ─── Rendu ──────────────────────────────────────────────────────────────────

  return (
    <div ref={containerRef} className="relative">
      {/* Zone input + pills */}
      <div
        className={compact
          ? 'flex flex-wrap items-center gap-1.5'
          : 'relative flex min-h-[42px] flex-wrap items-center gap-1.5 rounded-md border border-input bg-background px-2 py-1.5 cursor-text focus-within:ring-1 focus-within:ring-ring'
        }
        onClick={() => { inputRef.current?.focus(); setIsOpen(true) }}
      >
        {leadingPill && (
          <span
            title={leadingPill.label}
            onClick={(e) => e.stopPropagation()}
            className="inline-flex shrink-0 items-center rounded-full px-2.5 py-0.5 text-sm font-medium"
            style={{ backgroundColor: leadingPill.color, color: '#fff', border: 'none' }} // color-allow: blanc structurel pour contraste sur fond coloré du pill
          >
            <span className="max-w-[7rem] truncate">{leadingPill.label}</span>
          </span>
        )}
        {selected.map((gt, idx) => {
          const color = colors?.[idx % colors.length]
          return (
            <span
              key={gt}
              title={gt}
              className="inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-sm font-medium"
              style={
                color
                  ? { backgroundColor: color, color: '#fff', border: 'none' } // color-allow: blanc structurel pour contraste sur fond coloré du pill
                  : undefined
              }
              data-pill={!color ? 'neutral' : undefined}
            >
              {!color && (
                <span className="inline-flex items-center gap-1 rounded-full border border-border bg-muted px-2.5 py-0.5 text-sm text-foreground">
                  <span className="max-w-[7rem] truncate">{gt}</span>
                  <button
                    type="button"
                    onClick={(e) => { e.stopPropagation(); remove(gt) }}
                    className="ml-0.5 shrink-0 text-muted-foreground hover:text-foreground leading-none"
                    aria-label={`Retirer ${gt}`}
                  >
                    ×
                  </button>
                </span>
              )}
              {color && (
                <>
                  <span className="max-w-[7rem] truncate">{gt}</span>
                  <button
                    type="button"
                    onClick={(e) => { e.stopPropagation(); remove(gt) }}
                    className="ml-0.5 shrink-0 opacity-80 hover:opacity-100 leading-none"
                    aria-label={`Retirer ${gt}`}
                  >
                    ×
                  </button>
                </>
              )}
            </span>
          )
        })}

        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={(e) => { setQuery(e.target.value); setIsOpen(true) }}
          onFocus={() => setIsOpen(true)}
          onKeyDown={handleKeyDown}
          placeholder={selected.length === 0 ? placeholder : ''}
          disabled={isAtMax}
          className={`flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50 ${compact ? 'min-w-[80px]' : 'min-w-[120px]'} ${max != null && !compact ? 'pr-8' : ''}`}
        />

        {max != null && !compact && (
          <span className="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-muted-foreground pointer-events-none">
            {selected.length}/{max}
          </span>
        )}
      </div>

      {/* Dropdown — largeur mini indépendante du déclencheur (en mode compact, le
          conteneur peut être très étroit avant toute sélection : sans plancher, le
          popover hérite de cette largeur et tronque les sous-titres de presets). */}
      {isOpen && hasDropdownContent && (
        <div className="absolute z-50 mt-1 w-full min-w-[260px] rounded-md border border-border bg-popover shadow-md max-h-64 overflow-y-auto">
          {/* Groupes de presets (escouades / groupes) — chargent un roster entier */}
          {showPresets &&
            presetGroups!.map((group) =>
              group.options.length === 0 ? null : (
                <div key={group.key}>
                  <div className="sticky top-0 bg-popover/95 px-3 py-1.5 text-3xs font-semibold uppercase tracking-wide text-muted-foreground border-b border-border/50">
                    {group.label}
                  </div>
                  {group.options.map((opt) => (
                    <div
                      key={opt.id}
                      className="flex w-full items-center justify-between gap-2 px-3 py-2 hover:bg-accent"
                    >
                      <button
                        type="button"
                        onClick={() => loadPreset(opt.gamertags)}
                        className="flex min-w-0 flex-1 flex-col items-start text-left"
                      >
                        <span className="truncate text-sm">{opt.name}</span>
                        {opt.subtitle && (
                          <span className="line-clamp-2 text-2xs text-muted-foreground">{opt.subtitle}</span>
                        )}
                      </button>
                      {opt.trailing && <span className="flex shrink-0 items-center gap-1">{opt.trailing}</span>}
                    </div>
                  ))}
                </div>
              ),
            )}

          {/* Groupe 1 : Joueurs configurés */}
          {configured.length > 0 && (
            <div>
              <div className="sticky top-0 bg-popover/95 px-3 py-1.5 text-3xs font-semibold uppercase tracking-wide text-muted-foreground border-b border-border/50">
                {t('common.gamertag.players_configured')}
              </div>
              {configured.map((item) => (
                <DropdownItem
                  key={item.gamertag}
                  gamertag={item.gamertag}
                  icon="⭐"
                  disabled={isAtMax}
                  onSelect={() => add(item.gamertag)}
                />
              ))}
            </div>
          )}

          {/* Groupe 2 : Coéquipiers fréquents */}
          {frequent.length > 0 && (
            <div>
              <div className="sticky top-0 bg-popover/95 px-3 py-1.5 text-3xs font-semibold uppercase tracking-wide text-muted-foreground border-b border-border/50">
                {t('common.gamertag.frequent_teammates')}
              </div>
              {frequent.map((item) => (
                <DropdownItem
                  key={item.gamertag}
                  gamertag={item.gamertag}
                  badge={item.encounter_count ? `${item.encounter_count}×` : undefined}
                  disabled={isAtMax}
                  onSelect={() => add(item.gamertag)}
                />
              ))}
            </div>
          )}

          {/* Groupe 3 : Autres joueurs (recherche serveur) */}
          {remote.length > 0 && (
            <div>
              <div className="sticky top-0 bg-popover/95 px-3 py-1.5 text-3xs font-semibold uppercase tracking-wide text-muted-foreground border-b border-border/50">
                Autres joueurs
              </div>
              {remote.map((item) => (
                <DropdownItem
                  key={item.gamertag}
                  gamertag={item.gamertag}
                  badge={item.exact_match ? 'Exact' : undefined}
                  disabled={isAtMax}
                  onSelect={() => add(item.gamertag)}
                />
              ))}
            </div>
          )}

          {/* Groupe 4 : Repli live Xbox (résolution explicite d'un joueur jamais croisé) */}
          {liveResults.length > 0 && (
            <div>
              <div className="sticky top-0 bg-popover/95 px-3 py-1.5 text-3xs font-semibold uppercase tracking-wide text-muted-foreground border-b border-border/50">
                {t('common.gamertag.xbox_badge')}
              </div>
              {liveResults.map((item) => (
                <DropdownItem
                  key={item.gamertag}
                  gamertag={item.gamertag}
                  badge={t('common.gamertag.xbox_badge')}
                  disabled={isAtMax}
                  onSelect={() => add(item.gamertag)}
                />
              ))}
            </div>
          )}

          {/* Loading recherche serveur */}
          {isRemoteLoading && (
            <div className="px-3 py-2 text-sm text-muted-foreground">Recherche…</div>
          )}

          {/* Loading repli Xbox */}
          {isLiveLoading && (
            <div className="px-3 py-2 text-sm text-muted-foreground">{t('common.gamertag.searching_xbox')}</div>
          )}

          {/* Repli Xbox terminé SANS aucun résultat — sans ce retour, le bouton
              « Rechercher sur Xbox » disparaît et rien ne s'affiche : l'utilisateur
              croit que la recherche n'a pas fonctionné. */}
          {liveEmpty && (
            <div className="px-3 py-2 text-sm text-muted-foreground">
              {t('common.gamertag.no_xbox_result')}
            </div>
          )}

          {/* Message vide (recherche locale) — masqué quand le message Xbox ci-dessus
              couvre déjà le cas, pour ne pas empiler deux « rien trouvé ». */}
          {showEmptyMessage && !isLiveLoading && !liveEmpty && liveResults.length === 0 && (
            <div className="px-3 py-2 text-sm text-muted-foreground">
              {t('common.gamertag.no_player_found_prefix')}{trimmed}"
            </div>
          )}

          {/* Option saisie libre */}
          {canAddFree && (
            <button
              type="button"
              onClick={() => add(trimmed)}
              disabled={isAtMax}
              className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent disabled:opacity-50 border-t border-border/50"
            >
              <span className="text-muted-foreground">+</span>
              <span>
                Ajouter <span className="font-medium">"{trimmed}"</span>
              </span>
            </button>
          )}

          {/* CTA "Ajouter comme ami" — §3 plan Squad/Sessions */}
          {canAddFree && onAddAsFriend && (
            <button
              type="button"
              onClick={() => {
                onAddAsFriend(trimmed)
                setQuery('')
                setIsOpen(false)
              }}
              className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent border-t border-border/50 text-primary"
            >
              <span>👥</span>
              <span>
                Ajouter <span className="font-medium">"{trimmed}"</span> comme ami
              </span>
            </button>
          )}

          {/* CTA « Rechercher sur Xbox » — repli live explicite (V72-24) */}
          {canSearchLive && (
            <button
              type="button"
              onClick={triggerLiveSearch}
              className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent border-t border-border/50 text-primary"
            >
              <span>{t('common.gamertag.search_on_xbox')}</span>
            </button>
          )}

          {/* Footer — actions de gestion fournies par le parent (enregistrer…). */}
          {footer && <div className="border-t border-border/50">{footer}</div>}
        </div>
      )}
    </div>
  )
}

// ─── Sous-composant item dropdown ──────────────────────────────────────────────

interface DropdownItemProps {
  gamertag: string
  badge?: string
  icon?: string
  disabled: boolean
  onSelect: () => void
}

function DropdownItem({ gamertag, badge, icon, disabled, onSelect }: DropdownItemProps) {
  return (
    <button
      type="button"
      onClick={onSelect}
      disabled={disabled}
      className="flex w-full items-center justify-between px-3 py-2 text-sm hover:bg-accent disabled:opacity-40 disabled:cursor-not-allowed"
    >
      <span className="flex items-center gap-2">
        {icon && <span className="text-xs">{icon}</span>}
        <span>{gamertag}</span>
      </span>
      {badge && (
        <span className="text-xs text-muted-foreground">{badge}</span>
      )}
    </button>
  )
}
