import { Link } from 'react-router-dom'

export function NotFoundPage() {
  return (
    <main className="auth-page">
      <section className="auth-card">
        <h1>Page not found</h1>
        <p>The requested route is not available in this demo.</p>
        <Link className="btn-primary" to="/imports">
          Go to dashboard
        </Link>
      </section>
    </main>
  )
}
