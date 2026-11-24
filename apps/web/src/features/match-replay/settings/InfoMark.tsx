/**
 * InfoMark — la marque « (i) » d'une explication, en SVG et jamais en caractère : le « i cerclé » typographique dépend
 * de la police installée et se rend en carré vide sur les machines qui ne l'ont pas. Elle ne prend pas le focus et
 * n'ouvre rien — tout ce qu'elle a à dire tient dans son infobulle, et le lecteur d'écran lit la même phrase.
 */
export function InfoMark({ text }: { text: string }) {
  return (
    <span role="img" aria-label={text} title={text} className="inline-flex shrink-0 items-center text-muted-foreground">
      <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round" aria-hidden="true">
        <circle cx="6" cy="6" r="4.6" />
        <path d="M6 5.4v3" />
        <path d="M6 3.6v.2" />
      </svg>
    </span>
  )
}
