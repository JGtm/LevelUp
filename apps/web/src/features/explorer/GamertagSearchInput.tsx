/**
 * GamertagSearchInput — champ de recherche mono-sélection avec suggestions priorisées.
 *
 * Utilise le hook partagé useGamertagSuggestions :
 *   1. Joueurs configurés (DB locale)
 *   2. Coéquipiers fréquents (si fournis)
 *   3. Recherche serveur xuid_aliases (fuzzy, debounce 250 ms, ≥ 2 chars)
 */
import React, { useState, useRef, useEffect } from 'react'
import { Input } from '@/components/ui/input'
import { useGamertagSuggestions } from '@/components/ui/useGamertagSuggestions'
import type { TeammateOption } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { useAppShellStore } from '@/stores/appShellStore'

interface Props {
  onSelect: (gamertag: string) => void
  placeholder?: string
  /** Valeur initiale du champ (ex: gamertag venant d'un paramètre URL). */
  initialValue?: string
  /** Coéquipiers fréquents à afficher en deuxième groupe (optionnel). */
  frequentOptions?: TeammateOption[]
  /** Gamertags à exclure des suggestions (ex: joueur courant). */
  excludeGamertags?: string[]
}

export function GamertagSearchInput({
  onSelect,
  placeholder,
  initialValue,
  frequentOptions,
  excludeGamertags,
}: Props) {
  const [query, setQuery] = useState(initialValue ?? '')
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: ExplorerManifestKey, vars?: Record<string, unknown>) =>
    formatMessage(explorerManifest, key, locale, vars)
  const tc = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const resolvedPlaceholder = placeholder ?? t('explorer.search.placeholder')

  const {
    configured,
    frequent,
    remote,
    isRemoteLoading,
    hasAnyResult,
    remoteAttempted,
    liveResults,
    isLiveLoading,
    liveAttempted,
    triggerLiveSearch,
  } = useGamertagSuggestions({ query, frequentOptions, excludeGamertags })

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  const trimmed = query.trim()
  // Gamertag tapé absent de toutes les suggestions → propose la recherche libre
  // (même logique que canAddFree dans GamertagCombobox).
  const allSuggestedGts = new Set([
    ...configured.map((c) => c.gamertag),
    ...frequent.map((f) => f.gamertag),
    ...remote.map((r) => r.gamertag),
    ...liveResults.map((r) => r.gamertag),
  ])
  const canSearchFree = trimmed.length > 0 && !allSuggestedGts.has(trimmed)
  const showEmpty = trimmed.length > 0 && remoteAttempted && !isRemoteLoading && !hasAnyResult
  // Bouton « Rechercher sur Xbox » (V72-24) : proposé quand la recherche locale a
  // répondu sans surfacer le joueur tapé, tant que le repli live n'a pas déjà été lancé.
  const canSearchLive = canSearchFree && remoteAttempted && !isRemoteLoading && !liveAttempted
  const showDropdown =
    open &&
    trimmed.length > 0 &&
    (configured.length > 0 ||
      frequent.length > 0 ||
      remote.length > 0 ||
      liveResults.length > 0 ||
      isRemoteLoading ||
      isLiveLoading ||
      showEmpty ||
      canSearchFree ||
      canSearchLive)

  function pick(gt: string) {
    onSelect(gt)
    setQuery(gt)
    setOpen(false)
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter') {
      e.preventDefault()
      // Texte tapé hors suggestions → recherche le texte exact (joueur inconnu).
      // Sinon, prend la 1re suggestion priorisée (configured > frequent > remote).
      if (canSearchFree) {
        pick(trimmed)
      } else {
        const first =
          configured[0]?.gamertag ?? frequent[0]?.gamertag ?? remote[0]?.gamertag
        if (first) pick(first)
      }
    } else if (e.key === 'Escape') {
      setOpen(false)
    }
  }

  return (
    <div ref={ref} className="relative w-[22ch]">
      <Input
        value={query}
        onChange={(e) => {
          setQuery(e.target.value)
          setOpen(true)
        }}
        onFocus={() => setOpen(true)}
        onKeyDown={handleKeyDown}
        placeholder={resolvedPlaceholder}
      />

      {showDropdown && (
        <div className="absolute z-50 mt-1 w-full rounded-md border border-border bg-background shadow-lg max-h-72 overflow-y-auto">
          {configured.length > 0 && (
            <div>
              <div className="sticky top-0 bg-background/95 px-3 py-1.5 text-3xs font-semibold uppercase tracking-wide text-muted-foreground border-b border-border/50">
                {t('explorer.search.group_configured')}
              </div>
              {configured.map((item) => (
                <SuggestionRow
                  key={item.gamertag}
                  gamertag={item.gamertag}
                  icon="⭐"
                  onSelect={() => pick(item.gamertag)}
                />
              ))}
            </div>
          )}

          {frequent.length > 0 && (
            <div>
              <div className="sticky top-0 bg-background/95 px-3 py-1.5 text-3xs font-semibold uppercase tracking-wide text-muted-foreground border-b border-border/50">
                {t('explorer.search.group_frequent')}
              </div>
              {frequent.map((item) => (
                <SuggestionRow
                  key={item.gamertag}
                  gamertag={item.gamertag}
                  badge={item.encounter_count ? `${item.encounter_count}×` : undefined}
                  onSelect={() => pick(item.gamertag)}
                />
              ))}
            </div>
          )}

          {remote.length > 0 && (
            <div>
              <div className="sticky top-0 bg-background/95 px-3 py-1.5 text-3xs font-semibold uppercase tracking-wide text-muted-foreground border-b border-border/50">
                {t('explorer.search.group_others')}
              </div>
              {remote.map((item) => (
                <SuggestionRow
                  key={item.gamertag}
                  gamertag={item.gamertag}
                  badge={item.exact_match ? t('explorer.search.badge_exact') : undefined}
                  onSelect={() => pick(item.gamertag)}
                />
              ))}
            </div>
          )}

          {liveResults.length > 0 && (
            <div>
              <div className="sticky top-0 bg-background/95 px-3 py-1.5 text-3xs font-semibold uppercase tracking-wide text-muted-foreground border-b border-border/50">
                {tc('common.gamertag.xbox_badge')}
              </div>
              {liveResults.map((item) => (
                <SuggestionRow
                  key={item.gamertag}
                  gamertag={item.gamertag}
                  badge={tc('common.gamertag.xbox_badge')}
                  onSelect={() => pick(item.gamertag)}
                />
              ))}
            </div>
          )}

          {isRemoteLoading && (
            <div className="px-4 py-2 text-sm text-muted-foreground">{t('explorer.search.loading')}</div>
          )}

          {isLiveLoading && (
            <div className="px-4 py-2 text-sm text-muted-foreground">{tc('common.gamertag.searching_xbox')}</div>
          )}

          {showEmpty && !isLiveLoading && liveResults.length === 0 && (
            <div className="px-4 py-2 text-sm text-muted-foreground">
              {t('explorer.search.no_results', { query: trimmed })}
            </div>
          )}

          {canSearchFree && (
            <button
              type="button"
              onClick={() => pick(trimmed)}
              className="flex w-full items-center gap-2 px-4 py-2 text-sm text-left hover:bg-primary/10 transition-colors border-t border-border/50"
            >
              <span className="text-muted-foreground">+</span>
              <span>{t('explorer.search.search_for', { query: trimmed })}</span>
            </button>
          )}

          {canSearchLive && (
            <button
              type="button"
              onClick={triggerLiveSearch}
              className="flex w-full items-center gap-2 px-4 py-2 text-sm text-left text-primary hover:bg-primary/10 transition-colors border-t border-border/50"
            >
              <span>{tc('common.gamertag.search_on_xbox')}</span>
            </button>
          )}
        </div>
      )}
    </div>
  )
}

interface SuggestionRowProps {
  gamertag: string
  icon?: string
  badge?: string
  onSelect: () => void
}

function SuggestionRow({ gamertag, icon, badge, onSelect }: SuggestionRowProps) {
  return (
    <button
      type="button"
      onClick={onSelect}
      className="flex w-full items-center justify-between px-4 py-2 text-sm text-left hover:bg-primary/10 transition-colors"
    >
      <span className="flex items-center gap-2">
        {icon && <span className="text-xs">{icon}</span>}
        <span className="font-medium text-foreground">{gamertag}</span>
      </span>
      {badge && (
        <span className="ml-auto text-xs text-primary">{badge}</span>
      )}
    </button>
  )
}
