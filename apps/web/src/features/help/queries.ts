import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import type { Locale } from '@/lib/i18n/locale'

interface ReleaseNotesResponse {
  content: string
}

export function useReleaseNotes(lang: Locale) {
  return useQuery({
    queryKey: queryKeys.releaseNotes(lang),
    queryFn: () => api.get<ReleaseNotesResponse>(`/help/release-notes?lang=${lang}`),
    staleTime: 10 * 60 * 1000, // 10 min
  })
}
