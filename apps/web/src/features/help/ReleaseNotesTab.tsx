import Markdown from 'react-markdown'
import { Card, CardContent } from '@/components/ui/card'
import { PageLoader } from '@/components/ui/spinner'
import { type HelpLocale } from './i18n'
import { useReleaseNotes } from './queries'

interface ReleaseNotesTabProps {
  locale: HelpLocale
  errorMessage: string
  loadingMessage: string
}

export function ReleaseNotesTab({ locale, errorMessage, loadingMessage }: ReleaseNotesTabProps) {
  const { data, isLoading, error } = useReleaseNotes(locale)

  if (isLoading) {
    return (
      <div className="flex flex-col items-center gap-3 py-12 text-muted-foreground">
        <PageLoader />
        <span className="text-sm">{loadingMessage}</span>
      </div>
    )
  }

  if (error || !data) {
    return (
      <p className="py-8 text-center text-sm text-destructive">{errorMessage}</p>
    )
  }

  return (
    <Card>
      <CardContent className="prose prose-sm max-w-none pt-6 dark:prose-invert">
        <Markdown>{data.content}</Markdown>
      </CardContent>
    </Card>
  )
}
