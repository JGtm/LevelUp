import { tokenCssVar } from '@/lib/accessibility'

interface OutcomeBarProps {
  wins: number
  draws: number
  losses: number
  dnfs?: number
}

export function OutcomeBar({ wins, draws, losses, dnfs = 0 }: OutcomeBarProps) {
  const total = wins + draws + losses + dnfs
  if (total === 0) return null
  const pct = (n: number) => `${(n / total) * 100}%`
  return (
    <div className="flex h-1.5 w-full overflow-hidden rounded-full gap-px">
      {wins > 0 && <div style={{ width: pct(wins), backgroundColor: tokenCssVar('outcome-win') }} />}
      {draws > 0 && <div style={{ width: pct(draws), backgroundColor: tokenCssVar('outcome-draw') }} />}
      {dnfs > 0 && <div style={{ width: pct(dnfs), backgroundColor: tokenCssVar('outcome-dnf') }} />}
      {losses > 0 && <div style={{ width: pct(losses), backgroundColor: tokenCssVar('outcome-loss') }} />}
    </div>
  )
}
