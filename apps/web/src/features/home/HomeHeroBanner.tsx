/**
 * HomeHeroBanner — bandeau visuel décoratif de l'accueil.
 */
export function HomeHeroBanner() {
  return (
    <div
      aria-hidden="true"
      data-testid="home-hero-banner"
      className="relative overflow-hidden rounded-lg border border-border/80 bg-card/70 shadow-[0_30px_80px_-48px_rgba(15,23,42,0.95)]"
    >
      <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-primary/70 to-transparent" />
      <div className="pointer-events-none absolute inset-0 bg-gradient-to-r from-background/20 via-transparent to-background/28" />
      <div className="pointer-events-none absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-background/45 to-transparent" />

      <img
        src="/echoes-within-header.webp"
        alt=""
        className="block h-36 w-full object-cover object-center sm:h-48 lg:h-56"
      />
    </div>
  )
}
