/**
 * EventGlyph — un pictogramme du fil (`model/feedGlyphs.ts`) : tracé au trait, à l'encre
 * courante du parent, `shrink-0` — une icône rognée ne se lit plus du tout, c'est le NOM qui
 * cède le premier sur une ligne trop étroite.
 */
import { FEED_GLYPH_PATH, type FeedGlyphKey } from '../model/feedGlyphs'

export function EventGlyph({ glyph, size = 14 }: { glyph: FeedGlyphKey; size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      aria-hidden="true"
      className="shrink-0"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d={FEED_GLYPH_PATH[glyph]} />
    </svg>
  )
}
