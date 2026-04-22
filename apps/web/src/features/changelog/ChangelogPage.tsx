/**
 * ChangelogPage — affiche le journal des modifications (CHANGELOG.md).
 */
import Markdown from 'react-markdown'
import { useChangelog } from './queries'
import { PageHeader } from '@/components/shell/PageHeader'
import { PageLoader } from '@/components/ui/spinner'
import { Card, CardContent } from '@/components/ui/card'

export function ChangelogPage() {
  const { data, isLoading, error } = useChangelog()

  if (isLoading) return <PageLoader />
  if (error || !data) {
    return (
      <div className="p-6">
        <PageHeader title="Changelog" inset={false} />
        <p className="text-destructive mt-4">Impossible de charger le changelog.</p>
      </div>
    )
  }

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-4">
      <PageHeader title="Changelog" inset={false} />
      <Card>
        <CardContent className="prose prose-sm max-w-none pt-6">
          <Markdown>{data.content}</Markdown>
        </CardContent>
      </Card>
    </div>
  )
}
