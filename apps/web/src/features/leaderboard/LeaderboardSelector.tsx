/**
 * LeaderboardSelector — sélecteur natif compact (catégorie, playlist, saison) de
 * la page Classement, avec libellé accessible.
 *
 * Extrait de LeaderboardBlock.tsx (déjà au-dessus du seuil de 500 lignes) : rendu
 * pur, aucune logique — le filtrage des options vit dans LeaderboardBlock.logic.
 */
export function Selector({
  label,
  value,
  onChange,
  options,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  options: { value: string; label: string }[]
}) {
  return (
    <select
      aria-label={label}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="rounded border border-border bg-transparent px-2 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
    >
      {options.map((o) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
    </select>
  )
}
