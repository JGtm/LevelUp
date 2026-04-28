/**
 * GamertagSearchInput — champ de recherche mono-sélection avec suggestions priorisées.
 *
 * Utilise le hook partagé useGamertagSuggestions :
 *   1. Joueurs configurés (DB locale)
 *   2. Coéquipiers fréquents (si fournis)
 *   3. Recherche serveur xuid_aliases (fuzzy, debounce 250 ms, ≥ 2 chars)
 */
import { useState, useRef, useEffect } from 'react'
import { Input } from '@/components/ui/input'
import { useGamertagSuggestions } from '@/components/ui/useGamertagSuggestions'
import type { TeammateOption } from '@/lib/api/types'

interface Props {
  onSelect: (gamertag: string) => void
  placeholder?: string
  /** Coéquipiers fréquents à afficher en deuxième groupe (optionnel). */
  frequentOptions?: TeammateOption[]
  /** Gamertags à exclure des suggestions (ex: joueur courant). */
  excludeGamertags?: string[]
}

export function GamertagSearchInput({
  onSelect,
  placeholder = 'Rechercher un joueur…',
  frequentOptions,
  excludeGamertags,
}: Props) {
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  const { configured, frequent, remote, isRemoteLoading, hasAnyResult, remoteAttempted } =
    useGamertagSuggestions({ query, frequentOptions, excludeGamertags })

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
  const showEmpty = trimmed.length > 0 && remoteAttempted && !isRemoteLoading && !hasAnyResult
  const showDropdown =
    open &&
    trimmed.length > 0 &&
    (configured.length > 0 ||
      frequent.length > 0 ||
      remote.length > 0 ||
      isRemoteLoading ||
      showEmpty)

  function pick(gt: string) {
    onSelect(gt)
    setQuery(gt)
    setOpen(false)
  }

  return (
    <div ref={ref} className="relative w-full max-w-md">
      <Input
        value={query}
        onChange={(e) => {
          setQuery(e.target.value)
          setOpen(true)
        }}
        onFocus={() => setOpen(true)}
        placeholder={placeholder}
      />

      {showDropdown && (
        <div className="absolute z-50 mt-1 w-full rounded-md border border-border bg-background shadow-lg max-h-72 overflow-y-auto">
          {configured.length > 0 && (
            <div>
              <div className="sticky top-0 bg-background/95 px-3 py-1.5 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground border-b border-border/50">
                Joueurs configurés
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
              <div className="sticky top-0 bg-background/95 px-3 py-1.5 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground border-b border-border/50">
                Coéquipiers fréquents
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
              <div className="sticky top-0 bg-background/95 px-3 py-1.5 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground border-b border-border/50">
                Autres joueurs
              </div>
              {remote.map((item) => (
                <SuggestionRow
                  key={item.gamertag}
                  gamertag={item.gamertag}
                  badge={item.exact_match ? 'Exact' : undefined}
                  onSelect={() => pick(item.gamertag)}
                />
              ))}
            </div>
          )}

          {isRemoteLoading && (
            <div className="px-4 py-2 text-sm text-muted-foreground">Recherche…</div>
          )}

          {showEmpty && (
            <div className="px-4 py-2 text-sm text-muted-foreground">
              Aucun joueur trouvé pour "{trimmed}"
            </div>
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
