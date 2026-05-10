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

// ─── Props ──────────────────────────────────────────────────────────────────────

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
}

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
  onAddAsFriend,
  compact = false,
}: GamertagComboboxProps) {
  const [query, setQuery] = useState('')
  const [isOpen, setIsOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

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

  // ─── Suggestions via hook partagé ───────────────────────────────────────────

  const isAtMax = max != null && selected.length >= max
  const excludeGamertags = excludeGamertag ? [...selected, excludeGamertag] : selected

  const { configured, frequent, remote, isRemoteLoading, hasAnyResult, remoteAttempted } =
    useGamertagSuggestions({ query, frequentOptions, excludeGamertags })

  const trimmed = query.trim()
  const allSuggestedGts = new Set([
    ...configured.map((c) => c.gamertag),
    ...frequent.map((f) => f.gamertag),
    ...remote.map((r) => r.gamertag),
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
    trimmed.length > 0 && remoteAttempted && !isRemoteLoading && !hasAnyResult

  const hasDropdownContent =
    configured.length > 0 ||
    frequent.length > 0 ||
    remote.length > 0 ||
    canAddFree ||
    isRemoteLoading ||
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

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Escape') {
      setIsOpen(false)
      return
    }
    if (e.key === 'Enter') {
      e.preventDefault()
      const first =
        configured[0]?.gamertag ?? frequent[0]?.gamertag ?? remote[0]?.gamertag
      if (first) add(first)
      else if (canAddFree) add(trimmed)
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
        {selected.map((gt, idx) => {
          const color = colors?.[idx % colors.length]
          return (
            <span
              key={gt}
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
                  {gt}
                  <button
                    type="button"
                    onClick={(e) => { e.stopPropagation(); remove(gt) }}
                    className="ml-0.5 text-muted-foreground hover:text-foreground leading-none"
                    aria-label={`Retirer ${gt}`}
                  >
                    ×
                  </button>
                </span>
              )}
              {color && (
                <>
                  {gt}
                  <button
                    type="button"
                    onClick={(e) => { e.stopPropagation(); remove(gt) }}
                    className="ml-0.5 opacity-80 hover:opacity-100 leading-none"
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

      {/* Dropdown */}
      {isOpen && hasDropdownContent && (
        <div className="absolute z-50 mt-1 w-full rounded-md border border-border bg-popover shadow-md max-h-64 overflow-y-auto">
          {/* Groupe 1 : Joueurs configurés */}
          {configured.length > 0 && (
            <div>
              <div className="sticky top-0 bg-popover/95 px-3 py-1.5 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground border-b border-border/50">
                Joueurs configurés
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
              <div className="sticky top-0 bg-popover/95 px-3 py-1.5 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground border-b border-border/50">
                Coéquipiers fréquents
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
              <div className="sticky top-0 bg-popover/95 px-3 py-1.5 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground border-b border-border/50">
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

          {/* Loading recherche serveur */}
          {isRemoteLoading && (
            <div className="px-3 py-2 text-sm text-muted-foreground">Recherche…</div>
          )}

          {/* Message vide */}
          {showEmptyMessage && (
            <div className="px-3 py-2 text-sm text-muted-foreground">
              Aucun joueur trouvé pour "{trimmed}"
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
