import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import { getLabText, type LabLocale, type LabTab, type LabText } from './i18n'

interface LabNoticeProps {
  eyebrow: string
  title: string
  description: string
  bullets: string[]
  footer?: string
  readOnlyLabel?: string
  readonly?: boolean
}

function LabNotice({
  eyebrow,
  title,
  description,
  bullets,
  footer,
  readOnlyLabel,
  readonly = false,
}: LabNoticeProps) {
  return (
    <Card className="border-sky-200 bg-sky-50/80">
      <CardHeader className="pb-3">
        <div className="flex flex-wrap items-center gap-2">
          <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-sky-700">
            {eyebrow}
          </p>
          {readonly && readOnlyLabel ? <Badge variant="outline">{readOnlyLabel}</Badge> : null}
        </div>
        <CardTitle className="text-base text-slate-950">{title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3 text-sm text-slate-700">
        <p>{description}</p>
        <ul className="space-y-1.5 pl-5 text-sm text-slate-700">
          {bullets.map((item) => (
            <li key={item} className="list-disc">
              {item}
            </li>
          ))}
        </ul>
        {footer ? <p className="text-xs text-slate-500">{footer}</p> : null}
      </CardContent>
    </Card>
  )
}

export function LabIntroNotice({ locale }: { locale: LabLocale }) {
  const text = getLabText(locale)

  return (
    <LabNotice
      eyebrow={text.help.intro.eyebrow}
      title={text.help.intro.title}
      description={text.help.intro.description}
      bullets={text.help.intro.bullets}
      footer={text.help.intro.footer}
      readOnlyLabel={text.common.readOnly}
      readonly
    />
  )
}

function LabToolSectionCard({
  section,
  text,
}: {
  section: LabText['help']['tools'][LabTab]
  text: LabText
}) {
  return (
    <div className="space-y-3 rounded-2xl border border-slate-200 bg-white/80 p-4">
      <p className="text-sm font-semibold text-slate-950">{section.title}</p>
      <div className="space-y-2 text-sm text-slate-700">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">
            {text.help.sections.whatItDoes}
          </p>
          <p className="mt-1">{section.whatItDoes}</p>
        </div>
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">
            {text.help.sections.interest}
          </p>
          <p className="mt-1">{section.interest}</p>
        </div>
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.12em] text-slate-500">
            {text.help.sections.capabilities}
          </p>
          <ul className="mt-1 space-y-1.5 pl-5">
            {section.capabilities.map((item) => (
              <li key={item} className="list-disc">
                {item}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  )
}

export function LabSelectedToolNotice({
  tab,
  locale,
}: {
  tab: LabTab
  locale: LabLocale
}) {
  const text = getLabText(locale)
  const copy = text.help.tools[tab]

  return (
    <Card className="border-violet-200 bg-violet-50/60">
      <CardHeader className="pb-3">
        <div className="flex flex-wrap items-center gap-2">
          <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-violet-700">
            {text.help.selectedToolEyebrow}
          </p>
          <Badge variant="outline">{text.common.readOnly}</Badge>
        </div>
        <CardTitle className="text-base text-slate-950">{copy.title}</CardTitle>
      </CardHeader>
      <CardContent>
        <LabToolSectionCard section={copy} text={text} />
      </CardContent>
    </Card>
  )
}
