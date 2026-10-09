/**
 * TipText — le contenu d'une infobulle de l'onglet Emprise : une ligne par `\n`, la première en
 * gras (le sujet de la case, comme dans la maquette : « 21:23 · Starboard », « JGtm · Surbouclier »).
 */
export function TipText({ text }: { text: string }) {
  const [head, ...rest] = text.split('\n')
  return (
    <span className="block">
      <span className="block font-semibold">{head}</span>
      {rest.map((line, i) => (
        <span key={i} className="block">
          {line}
        </span>
      ))}
    </span>
  )
}
