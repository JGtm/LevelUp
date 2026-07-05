import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'

interface ChangelogResponse {
  content: string
}

export function useChangelog() {
  return useQuery({
    queryKey: queryKeys.changelog,
    queryFn: () => api.get<ChangelogResponse>('/changelog'),
    staleTime: 10 * 60 * 1000, // 10 min
  })
}
