import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenCssVar } from '@/lib/accessibility'
import { type GlossaryEntry, type GlossarySection, type HelpText } from './i18n'

const SECTION_ID_PREFIX = 'glossary-section-'
const ENTRY_ID_PREFIX = 'glossary-entry-'

function buildGlossaryEntryAnchor(term: string): string {
  return ENTRY_ID_PREFIX + slugify(term)
}

function slugify(value: string): string {
  return value
    .toLowerCase()
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '')
}

function normalizeForSearch(value: string): string {
  return value.toLowerCase().normalize('NFD').replace(/[̀-ͯ]/g, '')
}

function entryMatchesQuery(entry: GlossaryEntry, normalizedQuery: string): boolean {
  if (!normalizedQuery) return true
  const haystack = normalizeForSearch(
    [entry.term, entry.definition, entry.example ?? '', entry.formula ?? ''].join('\n'),
  )
  return normalizedQuery
    .split(/\s+/)
    .filter(Boolean)
    .every((token) => haystack.includes(token))
}

interface FilteredSection {
  id: string
  section: GlossarySection
  entries: GlossaryEntry[]
}

interface GlossaryTabProps {
  text: HelpText
}

export function GlossaryTab({ text }: GlossaryTabProps) {
  const [rawQuery, setRawQuery] = useState('')
  const [debouncedQuery, setDebouncedQuery] = useState('')
  const [activeId, setActiveId] = useState<string | null>(null)

  useEffect(() => {
    const handle = setTimeout(() => setDebouncedQuery(rawQuery), 120)
    return () => clearTimeout(handle)
  }, [rawQuery])

  const normalizedQuery = useMemo(
    () => normalizeForSearch(debouncedQuery.trim()),
    [debouncedQuery],
  )

  const queryTokens = useMemo(
    () => normalizedQuery.split(/\s+/).filter(Boolean),
    [normalizedQuery],
  )

  const filteredSections = useMemo<FilteredSection[]>(() => {
    return text.glossary.sections
      .map((section) => ({
        section,
        id: SECTION_ID_PREFIX + slugify(section.title),
        entries: section.entries.filter((entry) =>
          entryMatchesQuery(entry, normalizedQuery),
        ),
      }))
      .filter((s) => s.entries.length > 0)
  }, [text.glossary.sections, normalizedQuery])

  const sectionRefs = useRef<Map<string, HTMLElement>>(new Map())
  const initialAnchorScrollDone = useRef(false)

  useEffect(() => {
    if (initialAnchorScrollDone.current) return
    if (filteredSections.length === 0) return
    if (typeof window === 'undefined') return
    const hash = window.location.hash.slice(1)
    if (!hash || !hash.startsWith(ENTRY_ID_PREFIX)) return
    const el = document.getElementById(hash)
    if (!el) return
    initialAnchorScrollDone.current = true
    requestAnimationFrame(() => {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    })
  }, [filteredSections])

  useEffect(() => {
    if (filteredSections.length === 0) return
    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((e) => e.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)
        if (visible[0]) setActiveId(visible[0].target.id)
      },
      { rootMargin: '-30% 0px -55% 0px', threshold: 0 },
    )
    sectionRefs.current.forEach((el) => observer.observe(el))
    return () => observer.disconnect()
  }, [filteredSections])

  const setSectionRef = useCallback(
    (id: string) => (el: HTMLElement | null) => {
      if (el) sectionRefs.current.set(id, el)
      else sectionRefs.current.delete(id)
    },
    [],
  )

  const handleChipClick = useCallback((id: string) => {
    const el = document.getElementById(id)
    if (!el) return
    setActiveId(id)
    el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }, [])

  return (
    <div className="space-y-6">
      <div className="sticky top-0 z-20 -mx-6 border-b border-border bg-background px-6 py-3">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
          <div className="relative w-full sm:w-72 sm:shrink-0">
            <SearchIcon className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="search"
              value={rawQuery}
              onChange={(e) => setRawQuery(e.target.value)}
              placeholder={text.glossary.search.placeholder}
              aria-label={text.glossary.search.placeholder}
              className="pl-8"
            />
          </div>

          {filteredSections.length > 0 && (
            <nav
              aria-label={text.glossary.search.sectionsLabel}
              className="flex flex-1 min-w-0 flex-wrap gap-2"
            >
              {filteredSections.map(({ section, id }) => {
                const active = activeId === id
                return (
                  <button
                    key={id}
                    type="button"
                    onClick={() => handleChipClick(id)}
                    aria-current={active ? 'location' : undefined}
                    className={[
                      'shrink-0 rounded-full border px-3 py-1 text-xs font-medium transition-colors',
                      active
                        ? 'border-sidebar-primary bg-sidebar-primary/10 text-sidebar-primary'
                        : 'border-border text-muted-foreground hover:border-foreground/30 hover:text-foreground',
                    ].join(' ')}
                  >
                    {section.title}
                  </button>
                )
              })}
            </nav>
          )}
        </div>
      </div>

      {filteredSections.length === 0 ? (
        <EmptyStateNotice
          title={text.glossary.search.emptyTitle}
          description={text.glossary.search.emptyDescription}
        />
      ) : (
        <div className="space-y-8">
          {filteredSections.map(({ section, id, entries }) => (
            <GlossarySectionBlock
              key={id}
              id={id}
              section={section}
              entries={entries}
              setRef={setSectionRef(id)}
              queryTokens={queryTokens}
            />
          ))}
        </div>
      )}
    </div>
  )
}

interface GlossarySectionBlockProps {
  id: string
  section: GlossarySection
  entries: GlossaryEntry[]
  setRef: (el: HTMLElement | null) => void
  queryTokens: string[]
}

function GlossarySectionBlock({ id, section, entries, setRef, queryTokens }: GlossarySectionBlockProps) {
  return (
    <section id={id} ref={setRef} className="scroll-mt-24">
      <h2 className="mb-4 text-base font-semibold text-foreground/70 uppercase tracking-wider">
        {section.title}
      </h2>
      <div className="grid gap-3 sm:grid-cols-2">
        {entries.map((entry) => (
          <GlossaryCard key={entry.term} entry={entry} queryTokens={queryTokens} />
        ))}
      </div>
    </section>
  )
}

function GlossaryCard({ entry, queryTokens }: { entry: GlossaryEntry; queryTokens: string[] }) {
  return (
    <Card
      id={buildGlossaryEntryAnchor(entry.term)}
      className="overflow-hidden scroll-mt-32 transition-shadow hover:shadow-md"
    >
      <CardHeader className="pb-2 pt-4">
        <CardTitle className="text-sm font-semibold">
          <span className="text-sidebar-primary">
            <HighlightedText text={entry.term} tokens={queryTokens} />
          </span>
        </CardTitle>
      </CardHeader>
      <CardContent className="pb-4 space-y-3">
        <div className="text-sm text-muted-foreground leading-relaxed whitespace-pre-line">
          <HighlightedText text={entry.definition} tokens={queryTokens} />
        </div>

        {entry.example && (
          <CollapsibleSection label="Exemple" sectionName="exemple">
            <p className="text-xs text-muted-foreground leading-relaxed whitespace-pre-line italic">
              <HighlightedText text={entry.example} tokens={queryTokens} />
            </p>
          </CollapsibleSection>
        )}

        {entry.formula && (
          <CollapsibleSection label="Formule" sectionName="formule">
            <code className="block rounded bg-muted px-2.5 py-1.5 text-xs font-mono text-foreground/80 whitespace-pre overflow-x-auto">
              <HighlightedText text={entry.formula} tokens={queryTokens} />
            </code>
          </CollapsibleSection>
        )}
      </CardContent>
    </Card>
  )
}

interface HighlightRange {
  start: number
  end: number
}

function buildNormalizedMap(text: string): { normalized: string; indices: number[] } {
  let normalized = ''
  const indices: number[] = []
  for (let i = 0; i < text.length; i++) {
    const nfd = text[i].normalize('NFD').toLowerCase()
    for (const nc of nfd) {
      if (nc.charCodeAt(0) >= 0x0300 && nc.charCodeAt(0) <= 0x036f) continue
      normalized += nc
      indices.push(i)
    }
  }
  return { normalized, indices }
}

function findMatchRanges(text: string, tokens: string[]): HighlightRange[] {
  if (!tokens.length) return []
  const { normalized, indices } = buildNormalizedMap(text)
  const ranges: HighlightRange[] = []
  for (const token of tokens) {
    if (!token) continue
    let from = 0
    while (true) {
      const idx = normalized.indexOf(token, from)
      if (idx === -1) break
      const start = indices[idx]
      const end = indices[idx + token.length - 1] + 1
      ranges.push({ start, end })
      from = idx + token.length
    }
  }
  ranges.sort((a, b) => a.start - b.start || a.end - b.end)
  const merged: HighlightRange[] = []
  for (const r of ranges) {
    const last = merged[merged.length - 1]
    if (last && r.start <= last.end) {
      last.end = Math.max(last.end, r.end)
    } else {
      merged.push({ ...r })
    }
  }
  return merged
}

function HighlightedText({ text, tokens }: { text: string; tokens: string[] }) {
  const ranges = useMemo(() => findMatchRanges(text, tokens), [text, tokens])
  if (ranges.length === 0) return <>{text}</>

  const segments: ReactNode[] = []
  let pos = 0
  for (const r of ranges) {
    if (r.start > pos) segments.push(text.slice(pos, r.start))
    segments.push(
      <mark
        key={r.start}
        className="rounded-sm px-0.5"
        style={{
          backgroundColor: `color-mix(in srgb, ${tokenCssVar('warning')} 35%, transparent)`,
          color: 'inherit',
        }}
      >
        {text.slice(r.start, r.end)}
      </mark>,
    )
    pos = r.end
  }
  if (pos < text.length) segments.push(text.slice(pos))
  return <>{segments}</>
}

interface CollapsibleSectionProps {
  label: string
  sectionName: string
  children: ReactNode
}

function CollapsibleSection({ label, sectionName, children }: CollapsibleSectionProps) {
  const [open, setOpen] = useState(true)
  return (
    <div className="border-t border-border pt-3">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center justify-between gap-2 text-xs font-medium uppercase tracking-wide text-foreground/50 hover:text-foreground transition-colors mb-1"
        aria-label={open ? `Réduire la section ${sectionName}` : `Afficher la section ${sectionName}`}
        aria-expanded={open}
      >
        <span>{label}</span>
        <svg
          xmlns="http://www.w3.org/2000/svg"
          className={`h-4 w-4 transition-transform ${open ? 'rotate-180' : ''}`}
          viewBox="0 0 20 20"
          fill="currentColor"
          aria-hidden="true"
        >
          <path
            fillRule="evenodd"
            d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z"
            clipRule="evenodd"
          />
        </svg>
      </button>
      {open && children}
    </div>
  )
}

function SearchIcon({ className = '' }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 20 20"
      fill="currentColor"
      aria-hidden="true"
      className={className}
    >
      <path
        fillRule="evenodd"
        d="M9 3.5a5.5 5.5 0 100 11 5.5 5.5 0 000-11zM2 9a7 7 0 1112.452 4.391l3.328 3.329a.75.75 0 11-1.06 1.06l-3.329-3.328A7 7 0 012 9z"
        clipRule="evenodd"
      />
    </svg>
  )
}
