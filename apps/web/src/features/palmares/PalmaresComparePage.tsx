import { useEffect } from 'react'
import { useParams, useNavigate } from '@tanstack/react-router'

export function PalmaresComparePage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const navigate = useNavigate()

  useEffect(() => {
    void navigate({
      to: '/players/$playerSlug/compare',
      params: { playerSlug },
      replace: true,
    })
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return null
}
