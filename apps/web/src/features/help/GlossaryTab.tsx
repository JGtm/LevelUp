import { useState, type ReactNode } from 'react'
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
  return (
    <Card className="overflow-hidden transition-shadow hover:shadow-md">
      <CardHeader className="pb-2 pt-4">
        <CardTitle className="text-sm font-semibold">
          <span className="text-sidebar-primary">{entry.term}</span>
        </CardTitle>
      </CardHeader>
      <CardContent className="pb-4 space-y-3">
        <div className="text-sm text-muted-foreground leading-relaxed whitespace-pre-line">
          {entry.definition}
        </div>

        {entry.example && (
          <CollapsibleSection label="Exemple" sectionName="exemple">
            <p className="text-xs text-muted-foreground leading-relaxed whitespace-pre-line italic">
              {entry.example}
            </p>
          </CollapsibleSection>
        )}

        {entry.formula && (
          <CollapsibleSection label="Formule" sectionName="formule">
            <code className="block rounded bg-muted px-2.5 py-1.5 text-xs font-mono text-foreground/80 whitespace-pre overflow-x-auto">
              {entry.formula}
            </code>
          </CollapsibleSection>
        )}
      </CardContent>
    </Card>
  )
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
