import { tokenCssVar } from '@/lib/accessibility'
import { useChangelog } from './queries'
import {
  parseChangelog,
  getSectionToken,
  getSectionIcon,
  renderMarkdownInline,
  type ChangelogEntry,
  type ChangelogSection,
} from './parseChangelog'

export function ChangelogPage() {
  const { data, isLoading, error } = useChangelog()

  if (isLoading) return null
  if (error || !data) {
    return (
      <div className="p-6">
        <p className="text-destructive">Impossible de charger le changelog.</p>
      </div>
    )
  }

  const entries = parseChangelog(data.content)

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-8">
      <div>
        <h1 className="text-2xl font-bold">Changelog</h1>
        <p className="text-sm text-muted-foreground mt-1">Journal des modifications du projet.</p>
      </div>

      {entries.length === 0 ? (
        <p className="text-sm text-muted-foreground">Aucune entrée disponible.</p>
      ) : (
        <ol className="relative border-l border-border space-y-10 pl-8">
          {entries.map((entry) => (
            <ChangelogItem key={entry.version} entry={entry} />
          ))}
        </ol>
      )}
    </div>
  )
}

function ChangelogItem({ entry }: { entry: ChangelogEntry }) {
  return (
    <li className="relative">
      <span
        className="absolute -left-[2.1rem] top-1.5 w-3 h-3 rounded-full border-2 bg-background"
        style={{ borderColor: tokenCssVar('info') }}
      />

      <div className="flex items-baseline gap-3 mb-3 flex-wrap">
        {entry.isUnreleased ? (
          <span className="text-xs font-bold uppercase tracking-widest px-2.5 py-0.5 rounded bg-muted text-muted-foreground border border-border">
            À venir
          </span>
        ) : (
          <span
            className="text-xs font-bold uppercase tracking-widest px-2.5 py-0.5 rounded border"
            style={{
              background: `color-mix(in srgb, ${tokenCssVar('info')} 10%, transparent)`,
              borderColor: `color-mix(in srgb, ${tokenCssVar('info')} 30%, transparent)`,
              color: tokenCssVar('info'),
            }}
          >
            {entry.version}
          </span>
        )}
        {entry.date && <time className="text-xs text-muted-foreground">{entry.date}</time>}
      </div>

      <div className="space-y-4">
        {entry.sections.map((section) => (
          <SectionBlock key={section.type} section={section} />
        ))}
      </div>
    </li>
  )
}

function SectionBlock({ section }: { section: ChangelogSection }) {
  return (
    <div>
      <h3
        className="text-xs font-semibold uppercase tracking-wider mb-2 flex items-center gap-1.5"
        style={{ color: tokenCssVar(getSectionToken(section.type)) }}
      >
        <span aria-hidden="true">{getSectionIcon(section.type)}</span>
        {section.type}
      </h3>
      <ul className="space-y-1.5">
        {section.items.map((item, i) => (
          <li key={i} className="text-sm text-foreground flex gap-2">
            <span className="text-muted-foreground select-none mt-0.5 shrink-0">—</span>
            <span dangerouslySetInnerHTML={{ __html: renderMarkdownInline(item) }} />
          </li>
        ))}
      </ul>
    </div>
  )
}
