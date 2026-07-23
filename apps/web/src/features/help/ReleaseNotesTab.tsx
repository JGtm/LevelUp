import Markdown from 'react-markdown'
import { Card, CardContent } from '@/components/ui/card'
import { tokenCssVar } from '@/lib/accessibility'
import type { Locale } from '@/lib/i18n/locale'
import { useReleaseNotes } from './queries'
import type { SemanticToken } from '@/lib/accessibility'
import {
  parseReleaseNotes,
  getSectionToken,
  renderItemMarkdown,
  renderInlineMarkdown,
  type ReleaseEntry,
  type ReleaseSection,
} from './parseReleaseNotes'

interface ReleaseNotesTabProps {
  locale: Locale
  errorMessage: string
  loadingMessage: string
}

export function ReleaseNotesTab({ locale, errorMessage, loadingMessage }: ReleaseNotesTabProps) {
  const { data, isLoading, error } = useReleaseNotes(locale)

  if (isLoading) {
    return <p className="py-8 text-center text-sm text-muted-foreground">{loadingMessage}</p>
  }

  if (error || !data) {
    return <p className="py-8 text-center text-sm text-destructive">{errorMessage}</p>
  }

  const entries = parseReleaseNotes(data.content)

  if (entries.length === 0) {
    return (
      <Card>
        <CardContent className="prose prose-sm max-w-none pt-6 dark:prose-invert">
          <Markdown>{data.content}</Markdown>
        </CardContent>
      </Card>
    )
  }

  return (
    <ol className="relative border-l border-border space-y-12 pl-8 mt-2">
      {entries.map((entry) => (
        <ReleaseItem key={entry.version} entry={entry} />
      ))}
    </ol>
  )
}

function ReleaseItem({ entry }: { entry: ReleaseEntry }) {
  return (
    <li className="relative">
      <span
        className="absolute top-1.5 w-3 h-3 rounded-full border-2 bg-background"
        style={{ left: 'calc(-2rem - 6px)', borderColor: tokenCssVar('info') }}
      />

      <div className="flex items-baseline gap-3 mb-3 flex-wrap">
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
        <h2 className="text-base font-semibold text-foreground">{entry.title}</h2>
      </div>

      {entry.description.length > 0 && (
        <div className="space-y-2 mb-5 text-sm text-muted-foreground">
          {entry.description.map((p, i) => (
            <p key={i} dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(p) }} />
          ))}
        </div>
      )}

      <div className="space-y-5">
        {entry.sections.map((section, i) => (
          <SectionBlock
            key={`${section.title}-${i}`}
            section={section}
            accentToken={getSectionToken(i)}
          />
        ))}
      </div>
    </li>
  )
}

function SectionBlock({
  section,
  accentToken,
}: {
  section: ReleaseSection
  accentToken: SemanticToken
}) {
  return (
    <div>
      {section.title && (
        <h3
          className="text-xs font-semibold uppercase tracking-wider mb-2 flex items-center gap-1.5"
          style={{ color: tokenCssVar(accentToken) }}
        >
          <span aria-hidden="true">◈</span>
          {section.title}
        </h3>
      )}
      <ul className="space-y-1.5">
        {section.items.map((item, i) => (
          <li key={i} className="text-sm text-foreground flex gap-2">
            <span className="text-muted-foreground select-none mt-0.5 shrink-0">—</span>
            <span dangerouslySetInnerHTML={{ __html: renderItemMarkdown(item, accentToken) }} />
          </li>
        ))}
      </ul>
    </div>
  )
}
