import { useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { useNotifications } from '../components/notifications/NotificationProvider'
import { LoadingState } from '../components/states/LoadingState'
import { toErrorMessages } from '../utils/errors'

interface LocationState {
  from?: string
}

function validateLoginForm(email: string, password: string): string[] {
  const errors: string[] = []

  const emailTrimmed = email.trim()
  if (emailTrimmed.length === 0) {
    errors.push('Email is required.')
  } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(emailTrimmed)) {
    errors.push('Email format is invalid.')
  }

  if (password.trim().length === 0) {
    errors.push('Password is required.')
  }

  return errors
}

export function LoginPage() {
  const { isAuthenticated, isBootstrapping, login } = useAuth()
  const notifications = useNotifications()
  const navigate = useNavigate()
  const location = useLocation()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [errors, setErrors] = useState<string[]>([])
  const [isSubmitting, setIsSubmitting] = useState(false)

  const redirectTo = useMemo(() => {
    const state = location.state as LocationState | null
    return state?.from ?? '/imports'
  }, [location.state])

  if (isBootstrapping) {
    return (
      <main className="auth-page">
        <LoadingState label="Restoring session..." />
      </main>
    )
  }

  if (isAuthenticated) {
    return <Navigate to="/imports" replace />
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>): Promise<void> => {
    event.preventDefault()

    const validationErrors = validateLoginForm(email, password)
    if (validationErrors.length > 0) {
      setErrors(validationErrors)
      return
    }

    setErrors([])
    setIsSubmitting(true)

    try {
      await login({ email: email.trim(), password })
      notifications.success('Welcome back', 'Authentication succeeded.')
      navigate(redirectTo, { replace: true })
    } catch (submitError) {
      setErrors(toErrorMessages(submitError, 'Unable to authenticate.'))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <main className="auth-page">
      <section className="auth-card">
        <header>
          <h1>Sign in</h1>
          <p>Use your CryptoTax credentials to open the demo workspace.</p>
        </header>

        <form onSubmit={handleSubmit} className="form-grid">
          <label>
            Email
            <input
              type="email"
              required
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              autoComplete="email"
            />
          </label>

          <label>
            Password
            <input
              type="password"
              required
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              autoComplete="current-password"
            />
          </label>

          {errors.length > 0 ? (
            <div className="form-error">
              <ul className="form-error-list">
                {errors.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
          ) : null}

          <button type="submit" className="btn-primary" disabled={isSubmitting}>
            {isSubmitting ? 'Signing in...' : 'Sign in'}
          </button>
        </form>

        <footer className="auth-footer">
          <p>No account yet?</p>
          <Link to="/register">Register new account</Link>
        </footer>
      </section>
    </main>
  )
}
