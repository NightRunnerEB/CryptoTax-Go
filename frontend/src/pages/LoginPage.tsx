import { useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { ArrowRight, Lock, Mail, Moon, Sun } from 'lucide-react'
import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { useNotifications } from '../components/notifications/NotificationProvider'
import { useTheme } from '../components/theme/ThemeProvider'
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
  const { theme, toggleTheme } = useTheme()

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
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="w-6 h-6 border-2 border-primary/30 border-t-primary rounded-full animate-spin" />
      </div>
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
    <div className="min-h-screen bg-background flex relative">
      <button
        type="button"
        onClick={toggleTheme}
        className="absolute top-6 right-6 p-3 rounded-lg bg-surface border border-border hover:bg-muted transition-colors z-50"
        aria-label="Toggle theme"
      >
        {theme === 'dark' ? <Sun className="w-5 h-5 text-foreground" /> : <Moon className="w-5 h-5 text-foreground" />}
      </button>

      <div className="hidden lg:flex lg:w-1/2 bg-primary p-12 flex-col justify-between relative overflow-hidden">
        <div className="relative z-10">
          <h1 className="text-primary-foreground mb-4">CryptoTax</h1>
          <p className="text-primary-foreground/80 text-lg max-w-md">
            Professional cryptocurrency tax workflow and transaction operations platform
          </p>
        </div>

        <div className="relative z-10 space-y-6">
          <div className="bg-primary-foreground/10 backdrop-blur-sm rounded-xl p-6 border border-primary-foreground/20">
            <h3 className="text-primary-foreground mb-2">Trusted by professionals</h3>
            <p className="text-primary-foreground/70 text-sm">
              Import statements, normalize transactions, and generate comprehensive tax reports with confidence.
            </p>
          </div>

          <div className="grid grid-cols-3 gap-4 text-center">
            <div className="bg-primary-foreground/10 backdrop-blur-sm rounded-lg p-4 border border-primary-foreground/20">
              <div className="text-2xl font-bold text-primary-foreground mb-1">12+</div>
              <div className="text-xs text-primary-foreground/70">Exchanges</div>
            </div>
            <div className="bg-primary-foreground/10 backdrop-blur-sm rounded-lg p-4 border border-primary-foreground/20">
              <div className="text-2xl font-bold text-primary-foreground mb-1">100K+</div>
              <div className="text-xs text-primary-foreground/70">Transactions</div>
            </div>
            <div className="bg-primary-foreground/10 backdrop-blur-sm rounded-lg p-4 border border-primary-foreground/20">
              <div className="text-2xl font-bold text-primary-foreground mb-1">99.9%</div>
              <div className="text-xs text-primary-foreground/70">Accuracy</div>
            </div>
          </div>
        </div>

        <div className="absolute inset-0 bg-gradient-to-br from-primary-light/20 to-transparent" />
      </div>

      <div className="flex-1 flex items-center justify-center p-8">
        <div className="w-full max-w-md">
          <div className="mb-8">
            <h2 className="text-foreground mb-2">Welcome back</h2>
            <p className="text-muted-foreground">Sign in to your account to continue</p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-6">
            <div>
              <label className="block text-foreground mb-2">Email address</label>
              <div className="relative">
                <Mail className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-muted-foreground pointer-events-none" />
                <input
                  type="email"
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                  placeholder="you@example.com"
                  className="w-full pl-12 pr-4 py-3 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                  style={{ paddingLeft: '3rem' }}
                  autoComplete="email"
                  required
                />
              </div>
            </div>

            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="block text-foreground">Password</label>
                <button type="button" className="text-sm text-primary hover:text-primary-dark transition-colors">
                  Forgot password?
                </button>
              </div>
              <div className="relative">
                <Lock className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-muted-foreground pointer-events-none" />
                <input
                  type="password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  placeholder="Enter your password"
                  className="w-full pl-12 pr-4 py-3 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                  style={{ paddingLeft: '3rem' }}
                  autoComplete="current-password"
                  required
                />
              </div>
            </div>

            {errors.length > 0 ? (
              <div className="p-4 rounded-lg border border-[var(--status-failed)]/30 bg-[var(--status-failed-bg)]">
                <ul className="text-sm text-[var(--status-failed)] space-y-1 list-disc pl-5">
                  {errors.map((item) => (
                    <li key={item}>{item}</li>
                  ))}
                </ul>
              </div>
            ) : null}

            <button
              type="submit"
              disabled={isSubmitting}
              className="w-full mt-8 flex items-center justify-center gap-2 py-3 bg-primary hover:bg-primary-dark text-primary-foreground rounded-lg font-medium transition-all disabled:opacity-50"
            >
              {isSubmitting ? (
                <>
                  <div className="w-5 h-5 border-2 border-primary-foreground/30 border-t-primary-foreground rounded-full animate-spin" />
                  Signing in...
                </>
              ) : (
                <>
                  Sign in
                  <ArrowRight className="w-5 h-5" />
                </>
              )}
            </button>
          </form>

          <div className="mt-8 text-center">
            <p className="text-muted-foreground">
              Don&apos;t have an account?{' '}
              <Link to="/register" className="text-primary hover:text-primary-dark font-medium transition-colors">
                Create account
              </Link>
            </p>
          </div>

          <div className="mt-8 pt-8 border-t border-border">
            <p className="text-xs text-muted-foreground text-center">
              By signing in, you agree to our Terms of Service and Privacy Policy
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
