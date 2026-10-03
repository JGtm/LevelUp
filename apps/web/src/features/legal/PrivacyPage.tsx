/**
 * PrivacyPage — politique de confidentialité, route /privacy.
 *
 * Accessible SANS être connecté (cf. la liste des chemins anonymes de
 * `routes/__root.tsx`) : le pied de page de l'écran de connexion y renvoie, et
 * un visiteur qui hésite à connecter son compte Microsoft doit pouvoir lire
 * cette page avant de le faire.
 *
 * Le contenu vit dans `./i18n` — s'y référer avant toute modification de fond.
 */
import type { ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { privacyContactEmail } from '@/lib/appLinks'
import { getPrivacyText, PRIVACY_UPDATED_AT, CONTACT_TOKEN, type PrivacySection } from './i18n'

/**
 * Remplace le jeton de contact par un lien `mailto:` vivant. L'adresse n'est
 * assemblée qu'ici, à l'affichage — cf. la note anti-moissonnage de
 * `lib/appLinks`.
 */
function withContactLink(paragraph: string): ReactNode {
  if (!paragraph.includes(CONTACT_TOKEN)) return paragraph
  const email = privacyContactEmail()
  const [before, after] = paragraph.split(CONTACT_TOKEN)
  return (
    <>
      {before}
      <a
        href={`mailto:${email}`}
        className="text-foreground underline underline-offset-2 transition-colors hover:text-primary"
      >
        {email}
      </a>
      {after}
    </>
  )
}

function Section({ section }: { section: PrivacySection }) {
  return (
    <section className="space-y-3">
      <h2 className="text-base font-semibold text-foreground">{section.heading}</h2>
      {section.paragraphs?.map((p) => (
        <p key={p} className="text-sm leading-relaxed text-muted-foreground">
          {withContactLink(p)}
        </p>
      ))}
      {section.bullets && (
        <ul className="list-disc space-y-1 pl-5 text-sm leading-relaxed text-muted-foreground">
          {section.bullets.map((b) => (
            <li key={b}>{b}</li>
          ))}
        </ul>
      )}
    </section>
  )
}

export function PrivacyPage() {
  const locale = useAppShellStore((s) => s.locale)
  const text = getPrivacyText(locale)

  return (
    <div className="mx-auto w-full max-w-3xl px-4 py-8">
      <header className="space-y-2">
        <h1 className="text-2xl font-semibold text-foreground">{text.title}</h1>
        <p className="text-xs text-muted-foreground">
          {text.updatedLabel} : {PRIVACY_UPDATED_AT[locale]}
        </p>
      </header>

      <div className="mt-6 space-y-4">
        {text.intro.map((p) => (
          <p key={p} className="text-sm leading-relaxed text-muted-foreground">
            {p}
          </p>
        ))}
      </div>

      <div className="mt-8 space-y-8">
        {text.sections.map((section) => (
          <Section key={section.heading} section={section} />
        ))}
      </div>

      <p className="mt-10 border-t border-border pt-6 text-sm">
        <Link
          to="/"
          className="text-muted-foreground underline-offset-2 transition-colors hover:text-foreground hover:underline"
        >
          {text.backToApp}
        </Link>
      </p>
    </div>
  )
}
