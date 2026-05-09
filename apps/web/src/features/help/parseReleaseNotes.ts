export interface ReleaseSection {
  title: string
  items: string[]
}

export interface ReleaseEntry {
  version: string
  title: string
  description: string[]
  sections: ReleaseSection[]
}

const VERSION_RE = /^\*\*(v[\d.]+)\s*[—\-]\s*(.+?)\*\*$/
const SECTION_RE = /^\*\*(.+?)\*\*$/
const ITEM_RE = /^-\s+(.+)$/
const TOP_HEADING_RE = /^##\s+/

export function parseReleaseNotes(markdown: string): ReleaseEntry[] {
  const entries: ReleaseEntry[] = []
  let entry: ReleaseEntry | null = null
  let section: ReleaseSection | null = null

  const flushSection = () => {
    if (section && entry) entry.sections.push(section)
    section = null
  }
  const flushEntry = () => {
    flushSection()
    if (entry) entries.push(entry)
    entry = null
  }

  for (const rawLine of markdown.split('\n')) {
    const line = rawLine.trim()
    if (!line) continue
    if (TOP_HEADING_RE.test(line)) continue

    const versionMatch = line.match(VERSION_RE)
    if (versionMatch) {
      flushEntry()
      entry = {
        version: versionMatch[1],
        title: versionMatch[2].trim(),
        description: [],
        sections: [],
      }
      continue
    }

    if (!entry) continue

    const itemMatch = line.match(ITEM_RE)
    if (itemMatch) {
      if (!section) section = { title: '', items: [] }
      section.items.push(itemMatch[1])
      continue
    }

    const sectionMatch = line.match(SECTION_RE)
    if (sectionMatch) {
      flushSection()
      section = { title: sectionMatch[1].trim(), items: [] }
      continue
    }

    if (!section) {
      entry.description.push(line)
    }
  }

  flushEntry()
  return entries
}

export function renderItemMarkdown(text: string): string {
  return text
    .replace(/`([^`]+)`/g, '<code class="font-mono text-xs bg-muted px-1 py-0.5 rounded">$1</code>')
    .replace(
      /\*\*(.+?)\*\*/g,
      '<strong class="font-semibold" style="color: var(--ac-info)">$1</strong>',
    )
}

export function renderInlineMarkdown(text: string): string {
  return text
    .replace(/`([^`]+)`/g, '<code class="font-mono text-xs bg-muted px-1 py-0.5 rounded">$1</code>')
    .replace(/\*\*(.+?)\*\*/g, '<strong class="font-semibold">$1</strong>')
}
