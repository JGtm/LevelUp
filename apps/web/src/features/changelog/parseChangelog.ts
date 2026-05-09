import type { SemanticToken } from '@/lib/accessibility'

export interface ChangelogSection {
  type: string
  items: string[]
}

export interface ChangelogEntry {
  version: string
  date: string
  isUnreleased: boolean
  sections: ChangelogSection[]
}

const PREFIX_TOKEN: Record<string, SemanticToken> = {
  Added: 'success',
  Changed: 'info',
  Fixed: 'warning',
  Removed: 'destructive',
  Security: 'destructive',
  Performance: 'divergent-pos',
  Architecture: 'info',
}

const PREFIX_ICON: Record<string, string> = {
  Added: '✦',
  Changed: '↺',
  Fixed: '⬡',
  Removed: '✕',
  Security: '⚑',
  Performance: '⚡',
  Architecture: '◈',
}

function sectionPrefix(type: string): string {
  return type.split(' ')[0]
}

export function getSectionToken(type: string): SemanticToken {
  return PREFIX_TOKEN[sectionPrefix(type)] ?? 'info'
}

export function getSectionIcon(type: string): string {
  return PREFIX_ICON[sectionPrefix(type)] ?? '•'
}

export function renderMarkdownInline(text: string): string {
  return text
    .replace(/\*\*(.+?)\*\*/g, '<strong class="font-semibold">$1</strong>')
    .replace(
      /`([^`]+)`/g,
      '<code class="font-mono text-xs bg-muted px-1 py-0.5 rounded">$1</code>',
    )
}

export function parseChangelog(markdown: string): ChangelogEntry[] {
  const entries: ChangelogEntry[] = []
  let currentEntry: ChangelogEntry | null = null
  let currentSection: ChangelogSection | null = null

  for (const rawLine of markdown.split('\n')) {
    const line = rawLine.trimEnd()

    const entryMatch = line.match(/^## \[([^\]]+)\](?:\s*-\s*(\d{4}-\d{2}-\d{2}))?/)
    if (entryMatch) {
      if (currentSection && currentEntry) currentEntry.sections.push(currentSection)
      if (currentEntry) entries.push(currentEntry)
      currentSection = null
      const version = entryMatch[1]
      currentEntry = {
        version,
        date: entryMatch[2] ?? '',
        isUnreleased: version === 'Unreleased',
        sections: [],
      }
      continue
    }

    const sectionMatch = line.match(/^### (.+)/)
    if (sectionMatch && currentEntry) {
      if (currentSection) currentEntry.sections.push(currentSection)
      currentSection = { type: sectionMatch[1], items: [] }
      continue
    }

    const itemMatch = line.match(/^(?:  )?- (.+)/)
    if (itemMatch && currentSection) {
      currentSection.items.push(itemMatch[1])
    }
  }

  if (currentSection && currentEntry) currentEntry.sections.push(currentSection)
  if (currentEntry) entries.push(currentEntry)

  return entries.filter((e) => !e.isUnreleased || e.sections.some((s) => s.items.length > 0))
}
