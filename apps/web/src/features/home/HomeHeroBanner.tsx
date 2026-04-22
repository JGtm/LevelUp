import { useEffect, useState } from 'react'

const HEADER_IMAGES = [
  '/echoes-within-header.webp',
  '/Infinite.png',
  '/Infinite-2.png',
  '/LoneWolves.png',
]

const ROTATION_INTERVAL_MS = 45_000

function pickOther(current: string): string {
  const others = HEADER_IMAGES.filter((img) => img !== current)
  return others[Math.floor(Math.random() * others.length)]
}

/**
 * HomeHeroBanner — bandeau visuel décoratif de l'accueil.
 * Effectue une rotation aléatoire entre les images toutes les 45 secondes.
 */
export function HomeHeroBanner() {
  const [src, setSrc] = useState(
    () => HEADER_IMAGES[Math.floor(Math.random() * HEADER_IMAGES.length)],
  )
  const [visible, setVisible] = useState(true)

  useEffect(() => {
    const timer = setInterval(() => {
      setVisible(false)
      setTimeout(() => {
        setSrc((prev) => pickOther(prev))
        setVisible(true)
      }, 600)
    }, ROTATION_INTERVAL_MS)

    return () => clearInterval(timer)
  }, [])

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
        src={src}
        alt=""
        className="block h-36 w-full object-cover object-center sm:h-48 lg:h-56"
        style={{ transition: 'opacity 0.6s ease', opacity: visible ? 1 : 0 }}
      />
    </div>
  )
}
