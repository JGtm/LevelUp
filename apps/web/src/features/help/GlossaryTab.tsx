import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { type GlossaryEntry, type GlossarySection, type HelpText } from './i18n'

interface GlossaryTabProps {
  text: HelpText
}

export function GlossaryTab({ text }: GlossaryTabProps) {
  return (
    <div className="space-y-8">
      {text.glossary.sections.map((section) => (
        <GlossarySectionBlock key={section.title} section={section} />
      ))}
    </div>
  )
}

function GlossarySectionBlock({ section }: { section: GlossarySection }) {
  return (
    <section>
      <h2 className="mb-4 text-base font-semibold text-foreground/70 uppercase tracking-wider">
        {section.title}
      </h2>
      <div className="grid gap-3 sm:grid-cols-2">
        {section.entries.map((entry) => (
          <GlossaryCard key={entry.term} entry={entry} />
        ))}
      </div>
    </section>
  )
}

function GlossaryCard({ entry }: { entry: GlossaryEntry }) {
  const [expanded, setExpanded] = useState(false)
  const hasDetails = !!(entry.formula || entry.example)

  return (
    <Card className="overflow-hidden transition-shadow hover:shadow-md">
      <CardHeader className="pb-2 pt-4">
        <CardTitle className="flex items-center justify-between text-sm font-semibold">
          <span className="text-sidebar-primary">{entry.term}</span>
          {hasDetails && (
            <button
              type="button"
              onClick={() => setExpanded((v) => !v)}
              className="ml-2 shrink-0 rounded p-0.5 text-muted-foreground hover:text-foreground transition-colors"
              aria-label={expanded ? 'Réduire' : 'Détails'}
              aria-expanded={expanded}
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                className={`h-4 w-4 transition-transform ${expanded ? 'rotate-180' : ''}`}
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
          )}
        </CardTitle>
      </CardHeader>
      <CardContent className="pb-4 space-y-3">
        <p className="text-sm text-muted-foreground leading-relaxed">{entry.definition}</p>

        {hasDetails && expanded && (
          <div className="space-y-2 border-t border-border pt-3">
            {entry.formula && (
              <div>
                <span className="text-xs font-medium uppercase tracking-wide text-foreground/50 block mb-1">
                  Formule
                </span>
                <code className="block rounded bg-muted px-2.5 py-1.5 text-xs font-mono text-foreground/80 whitespace-pre-wrap">
                  {entry.formula}
                </code>
              </div>
            )}
            {entry.example && (
              <div>
                <span className="text-xs font-medium uppercase tracking-wide text-foreground/50 block mb-1">
                  Exemple
                </span>
                <p className="text-xs text-muted-foreground leading-relaxed whitespace-pre-line italic">
                  {entry.example}
                </p>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
