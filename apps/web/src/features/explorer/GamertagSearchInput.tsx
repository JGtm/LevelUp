/**
 * GamertagSearchInput — champ de recherche avec suggestions autocomplete.
 */
import { useState, useRef, useEffect } from 'react'
import { Input } from '@/components/ui/input'
import { useGamertagSearch } from './queries'

interface Props {
  onSelect: (gamertag: string) => void
  placeholder?: string
}

export function GamertagSearchInput({ onSelect, placeholder = 'Rechercher un joueur…' }: Props) {
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const { data, isFetching } = useGamertagSearch(query)

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

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

      {open && query.length >= 2 && (
        <div className="absolute z-50 mt-1 w-full rounded-md border border-border bg-background shadow-lg">
          {isFetching && (
            <div className="px-4 py-2 text-sm text-muted-foreground">Recherche…</div>
          )}
          {!isFetching && data?.items.length === 0 && (
            <div className="px-4 py-2 text-sm text-muted-foreground">Aucun résultat</div>
          )}
          {data?.items.map((item) => (
            <button
              key={item.gamertag}
              className="flex w-full items-center gap-2 px-4 py-2 text-sm text-left hover:bg-primary/10 transition-colors"
              onClick={() => {
                onSelect(item.gamertag)
                setQuery(item.gamertag)
                setOpen(false)
              }}
            >
              <span className="font-medium text-foreground">{item.gamertag}</span>
              {item.exact_match && (
                <span className="ml-auto text-xs text-primary">Exact</span>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
