/**
 * SquadGroupLoader — charge les membres d'un groupe dans la sélection Escouade.
 *
 * Pont optionnel entre les groupes (accès mutuel) et l'affichage Escouade : sélectionner
 * un groupe peuple la liste de coéquipiers (selectedGts) avec ses membres. NE touche
 * PAS au mécanisme index→couleur des pills (on remplace juste le contenu de la sélection
 * ordonnée existante). Monté uniquement quand une identité Halo est liée (sinon /groups
 * répondrait 401).
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { useMyGroups } from '@/features/groups/queries'

interface SquadGroupLoaderProps {
  /** Reçoit les gamertags des membres du groupe (hors joueur actif). */
  onLoad: (gamertags: string[]) => void
  /** Gamertag du joueur actif, exclu de la sélection. */
  exclude: string
}

export function SquadGroupLoader({ onLoad, exclude }: SquadGroupLoaderProps) {
  const locale = useAppShellStore((s) => s.locale)
  const { data: groups } = useMyGroups()
  if (!groups?.length) return null

  return (
    <select
      aria-label={locale === 'en' ? 'Load a group' : 'Charger un groupe'}
      value=""
      onChange={(e) => {
        const g = groups.find((x) => x.id === e.target.value)
        if (!g) return
        const gts = g.members
          .map((m) => m.gamertag)
          .filter((gt) => gt && gt.toLowerCase() !== exclude.toLowerCase())
        onLoad(gts)
      }}
      className="h-7 shrink-0 rounded-md border border-input bg-background px-2 text-xs text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
    >
      <option value="">{locale === 'en' ? 'Load a group…' : 'Charger un groupe…'}</option>
      {groups.map((g) => (
        <option key={g.id} value={g.id}>
          {g.name}
        </option>
      ))}
    </select>
  )
}
