import { useEffect, useRef, useState } from 'react'

interface AssetSearchProps {
  value: string
  placeholder: string
  onChange: (q: string) => void
  debounceMs?: number
}

export function AssetSearch({ value, placeholder, onChange, debounceMs = 300 }: AssetSearchProps) {
  const [draft, setDraft] = useState(value)
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)

  // sync si value réinitialisé par le store (ex: changement d'onglet) — ajustement
  // pendant le rendu (pattern React « prop précédente ») au lieu d'un effet.
  const [prevValue, setPrevValue] = useState(value)
  if (prevValue !== value) {
    setPrevValue(value)
    setDraft(value)
  }

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    const q = e.target.value
    setDraft(q)
    if (timer.current) clearTimeout(timer.current)
    timer.current = setTimeout(() => onChange(q), debounceMs)
  }

  // cleanup au démontage
  useEffect(() => () => { if (timer.current) clearTimeout(timer.current) }, [])

  return (
    <input
      type="search"
      value={draft}
      placeholder={placeholder}
      onChange={handleChange}
      className="w-full rounded border border-border bg-background px-2.5 py-1.5 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
      aria-label={placeholder}
    />
  )
}
