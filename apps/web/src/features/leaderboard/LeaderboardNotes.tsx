/**
 * LeaderboardNotes — bandeaux d'honnêteté affichés sous l'en-tête du classement.
 *
 * Ils disent ce que le relevé NE contient pas : saison archivée (classement CSR
 * seul), stats détaillées indisponibles ou partielles sur les lignes affichées.
 * Un seul endroit pour ces messages : même style (note discrète, tokens
 * sémantiques), même placement, quel que soit le motif.
 *
 * Les entrées falsy sont ignorées — l'appelant passe directement des expressions
 * conditionnelles, sans construire son tableau à la main.
 */
export function LeaderboardNotes({ notes }: { notes: (string | false | null | undefined)[] }) {
  const shown = notes.filter((n): n is string => typeof n === 'string' && n.length > 0)
  if (shown.length === 0) {
    return null
  }
  return (
    <>
      {shown.map((note) => (
        <p key={note} className="mt-1 text-xs italic text-muted-foreground">
          {note}
        </p>
      ))}
    </>
  )
}
