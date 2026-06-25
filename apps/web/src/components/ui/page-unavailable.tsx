import { Card, CardContent } from './card'
import { Button } from './button'

export interface PageUnavailableAction {
  label: string
  onClick: () => void
  variant?: 'default' | 'outline'
}

interface PageUnavailableProps {
  title: string
  description: string
  actions?: PageUnavailableAction[]
}

/**
 * État "page indisponible" réutilisable (ADR 0029) — affiché quand une ressource
 * existe mais n'est pas accessible au joueur courant (match non-participé, session
 * inexistante, accès refusé). Boutons de navigation explicites plutôt qu'une
 * redirection automatique (l'utilisateur garde le contrôle).
 */
export function PageUnavailable({ title, description, actions = [] }: PageUnavailableProps) {
  return (
    <div className="p-6">
      <Card>
        <CardContent className="py-10 text-center">
          <p className="text-base font-semibold text-foreground">{title}</p>
          <p className="mx-auto mt-2 max-w-md text-sm text-muted-foreground">{description}</p>
          {actions.length > 0 && (
            <div className="mt-6 flex flex-wrap justify-center gap-3">
              {actions.map((a) => (
                <Button
                  key={a.label}
                  variant={a.variant ?? 'outline'}
                  size="sm"
                  onClick={a.onClick}
                >
                  {a.label}
                </Button>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
