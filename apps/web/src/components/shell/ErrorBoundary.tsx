/**
 * ErrorBoundary — limite d'erreur React globale.
 *
 * Intercepte les erreurs non gérées dans l'arborescence de composants enfants
 * et affiche un écran de repli au lieu d'une page blanche.
 * Doit être un class component (React exige componentDidCatch).
 */
import { Component, type ErrorInfo, type ReactNode } from 'react'

interface ErrorBoundaryProps {
  children: ReactNode
  /** Composant de repli personnalisé (optionnel). */
  fallback?: ReactNode
}

interface ErrorBoundaryState {
  hasError: boolean
  error: Error | null
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    // Logging minimal — peut être branché sur Sentry ou autre service.
    console.error('[ErrorBoundary] Erreur non gérée :', error, info.componentStack)
  }

  render(): ReactNode {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback
      }
      return (
        <div className="flex h-screen flex-col items-center justify-center gap-4 bg-muted p-8 text-center">
          <h1 className="text-2xl font-semibold text-foreground">Une erreur est survenue</h1>
          <p className="text-muted-foreground">
            L&apos;application a rencontré un problème inattendu. Rechargez la page pour réessayer.
          </p>
          <pre className="max-w-lg overflow-auto rounded bg-muted p-4 text-left text-xs text-muted-foreground">
            {this.state.error?.message}
          </pre>
          <button
            className="rounded bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-primary/90"
            onClick={() => window.location.reload()}
            type="button"
          >
            Recharger la page
          </button>
        </div>
      )
    }

    return this.props.children
  }
}
