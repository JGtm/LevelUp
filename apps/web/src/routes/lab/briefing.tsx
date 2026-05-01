import { createFileRoute } from '@tanstack/react-router'

import { SessionBriefingMockPage } from '@/features/lab/SessionBriefingMock'

export const Route = createFileRoute('/lab/briefing')({
  component: SessionBriefingMockPage,
})
