/**
 * GamertagCombobox — sélecteur multi-gamertag avec fuzzy search.
 *
 * Affiche deux groupes priorisés :
 *  1. « Joueurs configurés » — profils dans available_players (db_profiles)
 *  2. « Coéquipiers fréquents » — options depuis l'API (encounter_count)
 *
 * Autorise aussi la saisie libre d'un gamertag absent des suggestions.
 */
import { useRef, useState, useEffect, useCallback } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import type { TeammateOption } from '@/lib/api/types'

// ─── Fuzzy scoring ─────────────────────────────────────────────────────────────

function fuzzyScore(query: string, target: string): number {
  const q = query.toLowerCase()
  const t = target.toLowerCase()
  if (t === q) return 4
  if (t.startsWith(q)) return 3
  if (t.includes(q)) return 2
  // correspondance caractère par caractère (dans l'ordre)
  let qi = 0
  for (let i = 0; i < t.length && qi < q.length; i++) {
    if (t[i] === q[qi]) qi++
  }
  return qi === q.length ? 1 : 0
}

// ─── Types internes ─────────────────────────────────────────────────────────────

interface SuggestionItem {
  gamertag: string
  badge?: string
  isConfigured: boolean
}

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
}: GamertagComboboxProps) {
  const [query, setQuery] = useState('')
  const [isOpen, setIsOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const availablePlayers = useAppShellStore((s) => s.availablePlayers)

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

  // ─── Calcul des suggestions ─────────────────────────────────────────────────

  const isAtMax = max != null && selected.length >= max

  const suggestions = useCallback((): { configured: SuggestionItem[]; frequent: SuggestionItem[] } => {
    const q = query.trim()

    // Gamertags à exclure : sélectionnés + joueur courant
    const excluded = new Set([...selected, ...(excludeGamertag ? [excludeGamertag] : [])])

    // Groupe 1 : available_players (db_profiles)
    const configuredGts = new Set(availablePlayers.map((p) => p.gamertag))
    const configured: SuggestionItem[] = availablePlayers
      .filter((p) => !excluded.has(p.gamertag))
      .filter((p) => !q || fuzzyScore(q, p.gamertag) > 0)
      .sort((a, b) => fuzzyScore(q, b.gamertag) - fuzzyScore(q, a.gamertag))
      .map((p) => ({ gamertag: p.gamertag, isConfigured: true }))

    // Groupe 2 : coéquipiers fréquents (non-dupliqués du groupe 1)
    const frequent: SuggestionItem[] = frequentOptions
      .filter((o) => !excluded.has(o.gamertag) && !configuredGts.has(o.gamertag))
      .filter((o) => !q || fuzzyScore(q, o.gamertag) > 0)
      .sort((a, b) => {
        const scoreDiff = fuzzyScore(q, b.gamertag) - fuzzyScore(q, a.gamertag)
        return scoreDiff !== 0 ? scoreDiff : (b.encounter_count ?? 0) - (a.encounter_count ?? 0)
      })
      .map((o) => ({
        gamertag: o.gamertag,
        badge: o.encounter_count ? `${o.encounter_count}×` : undefined,
        isConfigured: false,
      }))

    return { configured, frequent }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query, selected, availablePlayers, frequentOptions, excludeGamertag])

  const { configured, frequent } = suggestions()

  // L'option "ajouter en libre" n'apparaît que si :
  // - allowFreeInput actif
  // - query non-vide
  // - pas déjà sélectionné
  // - pas déjà dans les suggestions
  const allSuggestedGts = new Set([...configured, ...frequent].map((s) => s.gamertag))
  const canAddFree =
    allowFreeInput &&
    query.trim().length > 0 &&
    !selected.includes(query.trim()) &&
    !allSuggestedGts.has(query.trim())

  const hasDropdownContent = configured.length > 0 || frequent.length > 0 || canAddFree

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
      const first = configured[0]?.gamertag ?? frequent[0]?.gamertag
      if (first) add(first)
      else if (canAddFree) add(query.trim())
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
        className="flex min-h-[42px] flex-wrap items-center gap-1.5 rounded-md border border-input bg-background px-2 py-1.5 cursor-text focus-within:ring-1 focus-within:ring-ring"
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
                  ? { backgroundColor: color, color: '#fff', border: 'none' }
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
          className="flex-1 min-w-[120px] bg-transparent text-sm outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50"
        />

        {max != null && (
          <span className="ml-auto shrink-0 text-xs text-muted-foreground">
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
                  badge={item.badge}
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
                  badge={item.badge}
                  disabled={isAtMax}
                  onSelect={() => add(item.gamertag)}
                />
              ))}
            </div>
          )}

          {/* Option saisie libre */}
          {canAddFree && (
            <button
              type="button"
              onClick={() => add(query.trim())}
              disabled={isAtMax}
              className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent disabled:opacity-50 border-t border-border/50"
            >
              <span className="text-muted-foreground">+</span>
              <span>
                Ajouter <span className="font-medium">"{query.trim()}"</span>
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
