import { ArrowRight, Search } from 'lucide-react'
import { Link } from 'react-router-dom'

export function NotFoundPage() {
  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-8">
      <div className="w-full max-w-xl bg-surface rounded-2xl border border-border p-10 text-center" style={{ boxShadow: 'var(--shadow-md)' }}>
        <div className="w-16 h-16 mx-auto mb-6 rounded-full bg-muted flex items-center justify-center">
          <Search className="w-8 h-8 text-muted-foreground" />
        </div>
        <p className="text-sm text-primary font-medium mb-3">404</p>
        <h1 className="text-foreground mb-3">Page not found</h1>
        <p className="text-muted-foreground mb-8">
          The requested route is not available in this workspace.
        </p>
        <Link
          to="/imports"
          className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-primary hover:bg-primary-dark text-primary-foreground rounded-lg font-medium transition-all"
        >
          Go to dashboard
          <ArrowRight className="w-5 h-5" />
        </Link>
      </div>
    </div>
  )
}
