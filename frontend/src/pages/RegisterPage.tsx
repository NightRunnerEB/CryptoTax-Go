import { useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { ArrowRight, Building2, Lock, Mail, Moon, Sun, User } from 'lucide-react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import type { RegisterRequest } from '../api/authService'
import { useAuth } from '../auth/AuthContext'
import { useNotifications } from '../components/notifications/NotificationProvider'
import { useTheme } from '../components/theme/ThemeProvider'
import { toErrorMessages } from '../utils/errors'

interface RegistrationFormState {
  email: string
  password: string
  confirmPassword: string
  inn: string
  lastName: string
  firstName: string
  middleName: string
  jurisdiction: string
  timezone: string
  phone: string
  wallets: string
  taxResidencyStatus: string
  taxpayerType: string
}

const INITIAL_FORM: RegistrationFormState = {
  email: '',
  password: '',
  confirmPassword: '',
  inn: '',
  lastName: '',
  firstName: '',
  middleName: '',
  jurisdiction: 'RU',
  timezone: 'Europe/Moscow',
  phone: '',
  wallets: '',
  taxResidencyStatus: 'RESIDENT',
  taxpayerType: 'INDIVIDUAL',
}

function splitWallets(raw: string): string[] {
  return raw
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
}

const JURISDICTIONS = new Set(['RU', 'KZ'])
const TAX_RESIDENCY_STATUSES = new Set(['RESIDENT', 'NON_RESIDENT'])
const TAXPAYER_TYPES = new Set(['INDIVIDUAL', 'SOLE_PROPRIETOR', 'LEGAL_ENTITY'])

function validateRegistrationForm(form: RegistrationFormState): string[] {
  const errors: string[] = []

  if (form.email.trim().length === 0) {
    errors.push('Email is required.')
  } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email.trim())) {
    errors.push('Email format is invalid.')
  }

  if (form.password.trim().length === 0) {
    errors.push('Password is required.')
  } else if (form.password.length < 8) {
    errors.push('Password must contain at least 8 characters.')
  }

  if (form.confirmPassword.trim().length === 0) {
    errors.push('Password confirmation is required.')
  } else if (form.password !== form.confirmPassword) {
    errors.push('Passwords do not match.')
  }

  if (form.inn.trim().length === 0) {
    errors.push('INN is required.')
  }
  if (form.lastName.trim().length === 0) {
    errors.push('Last name is required.')
  }
  if (form.firstName.trim().length === 0) {
    errors.push('First name is required.')
  }
  if (form.middleName.trim().length === 0) {
    errors.push('Middle name is required.')
  }
  if (form.timezone.trim().length === 0) {
    errors.push('Timezone is required.')
  }

  if (!JURISDICTIONS.has(form.jurisdiction)) {
    errors.push('Jurisdiction must be RU or KZ.')
  }
  if (!TAX_RESIDENCY_STATUSES.has(form.taxResidencyStatus)) {
    errors.push('Tax residency status is invalid.')
  }
  if (!TAXPAYER_TYPES.has(form.taxpayerType)) {
    errors.push('Taxpayer type is invalid.')
  }

  return errors
}

function toRegisterPayload(form: RegistrationFormState): RegisterRequest {
  return {
    email: form.email.trim(),
    password: form.password,
    tax_profile: {
      inn: form.inn.trim(),
      last_name: form.lastName.trim(),
      first_name: form.firstName.trim(),
      middle_name: form.middleName.trim(),
      jurisdiction: form.jurisdiction,
      timezone: form.timezone.trim(),
      phone: form.phone.trim(),
      wallets: splitWallets(form.wallets),
      tax_residency_status: form.taxResidencyStatus,
      taxpayer_type: form.taxpayerType,
    },
  }
}

export function RegisterPage() {
  const { isAuthenticated, isBootstrapping, register } = useAuth()
  const navigate = useNavigate()
  const notifications = useNotifications()
  const { theme, toggleTheme } = useTheme()

  const [form, setForm] = useState<RegistrationFormState>(INITIAL_FORM)
  const [errors, setErrors] = useState<string[]>([])
  const [isSubmitting, setIsSubmitting] = useState(false)

  const payload = useMemo<RegisterRequest>(() => toRegisterPayload(form), [form])

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

    const validationErrors = validateRegistrationForm(form)
    if (validationErrors.length > 0) {
      setErrors(validationErrors)
      return
    }

    setErrors([])
    setIsSubmitting(true)

    try {
      await register(payload)
      notifications.success('Registration accepted', 'Check email and verify before login.')
      navigate('/login', { replace: true })
    } catch (submitError) {
      setErrors(toErrorMessages(submitError, 'Registration failed.'))
    } finally {
      setIsSubmitting(false)
    }
  }

  const setField = <TKey extends keyof RegistrationFormState>(field: TKey, value: RegistrationFormState[TKey]): void => {
    setForm((prev) => ({
      ...prev,
      [field]: value,
    }))
  }

  const passwordsMatch =
    form.confirmPassword.length === 0 || form.password.length === 0 || form.password === form.confirmPassword

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
            Join thousands of professionals who trust CryptoTax for accurate cryptocurrency tax reporting
          </p>
        </div>

        <div className="relative z-10 space-y-4">
          <div className="flex items-start gap-4">
            <div className="w-10 h-10 rounded-lg bg-primary-foreground/10 flex items-center justify-center flex-shrink-0 border border-primary-foreground/20">
              <User className="w-5 h-5 text-primary-foreground" />
            </div>
            <div>
              <h4 className="text-primary-foreground font-medium mb-1">Multi-exchange support</h4>
              <p className="text-primary-foreground/70 text-sm">Import and normalize data from 12+ major exchanges</p>
            </div>
          </div>

          <div className="flex items-start gap-4">
            <div className="w-10 h-10 rounded-lg bg-primary-foreground/10 flex items-center justify-center flex-shrink-0 border border-primary-foreground/20">
              <Building2 className="w-5 h-5 text-primary-foreground" />
            </div>
            <div>
              <h4 className="text-primary-foreground font-medium mb-1">Automated calculations</h4>
              <p className="text-primary-foreground/70 text-sm">FIFO, LIFO, and other methods with full audit trails</p>
            </div>
          </div>

          <div className="flex items-start gap-4">
            <div className="w-10 h-10 rounded-lg bg-primary-foreground/10 flex items-center justify-center flex-shrink-0 border border-primary-foreground/20">
              <ArrowRight className="w-5 h-5 text-primary-foreground" />
            </div>
            <div>
              <h4 className="text-primary-foreground font-medium mb-1">Professional reports</h4>
              <p className="text-primary-foreground/70 text-sm">Export-ready tax reports and comprehensive documentation</p>
            </div>
          </div>
        </div>

        <div className="absolute inset-0 bg-gradient-to-br from-primary-light/20 to-transparent" />
      </div>

      <div className="flex-1 flex items-center justify-center p-8 lg:py-10">
        <div className="w-full max-w-2xl">
          <div className="mb-8">
            <h2 className="text-foreground mb-2">Create your account</h2>
            <p className="text-muted-foreground">Start managing your crypto taxes professionally</p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-5">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-foreground mb-2">First name</label>
                <div className="relative">
                  <User className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-muted-foreground pointer-events-none" />
                  <input
                    type="text"
                    value={form.firstName}
                    onChange={(event) => setField('firstName', event.target.value)}
                    placeholder="John"
                    className="w-full pl-12 pr-4 py-3 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                    style={{ paddingLeft: '3rem' }}
                    required
                  />
                </div>
              </div>

              <div>
                <label className="block text-foreground mb-2">Last name</label>
                <input
                  type="text"
                  value={form.lastName}
                  onChange={(event) => setField('lastName', event.target.value)}
                  placeholder="Doe"
                  className="w-full px-4 py-3 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                  required
                />
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-foreground mb-2">Middle name</label>
                <input
                  type="text"
                  value={form.middleName}
                  onChange={(event) => setField('middleName', event.target.value)}
                  placeholder="Sergeevich"
                  className="w-full px-4 py-3 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                  required
                />
              </div>

              <div>
                <label className="block text-foreground mb-2">INN</label>
                <input
                  type="text"
                  value={form.inn}
                  onChange={(event) => setField('inn', event.target.value)}
                  placeholder="7730123456789"
                  className="w-full px-4 py-3 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                  required
                />
              </div>
            </div>

            <div>
              <label className="block text-foreground mb-2">Email address</label>
              <div className="relative">
                <Mail className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-muted-foreground pointer-events-none" />
                <input
                  type="email"
                  value={form.email}
                  onChange={(event) => setField('email', event.target.value)}
                  placeholder="you@example.com"
                  className="w-full pl-12 pr-4 py-3 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                  style={{ paddingLeft: '3rem' }}
                  autoComplete="email"
                  required
                />
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-foreground mb-2">Password</label>
                <div className="relative">
                  <Lock className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-muted-foreground pointer-events-none" />
                  <input
                    type="password"
                    value={form.password}
                    onChange={(event) => setField('password', event.target.value)}
                    placeholder="Create a strong password"
                    className={`w-full pl-12 pr-4 py-3 bg-input-background border rounded-lg text-foreground focus:outline-none focus:ring-2 transition-all ${
                      !passwordsMatch
                        ? 'border-[var(--status-failed)] focus:ring-[var(--status-failed)]'
                        : 'border-input-border focus:ring-primary focus:border-transparent'
                    }`}
                    style={{ paddingLeft: '3rem' }}
                    autoComplete="new-password"
                    required
                  />
                </div>
              </div>

              <div>
                <label className="block text-foreground mb-2">Confirm password</label>
                <div className="relative">
                  <Lock className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-muted-foreground pointer-events-none" />
                  <input
                    type="password"
                    value={form.confirmPassword}
                    onChange={(event) => setField('confirmPassword', event.target.value)}
                    placeholder="Confirm your password"
                    className={`w-full pl-12 pr-4 py-3 bg-input-background border rounded-lg text-foreground focus:outline-none focus:ring-2 transition-all ${
                      !passwordsMatch
                        ? 'border-[var(--status-failed)] focus:ring-[var(--status-failed)]'
                        : 'border-input-border focus:ring-primary focus:border-transparent'
                    }`}
                    style={{ paddingLeft: '3rem' }}
                    autoComplete="new-password"
                    required
                  />
                </div>
              </div>
            </div>

            {!passwordsMatch ? <p className="text-sm text-[var(--status-failed)]">Passwords do not match</p> : null}

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-foreground mb-2">Jurisdiction</label>
                <div className="relative">
                  <Building2 className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-muted-foreground pointer-events-none" />
                  <select
                    value={form.jurisdiction}
                    onChange={(event) => setField('jurisdiction', event.target.value)}
                    className="w-full pl-12 pr-4 py-3 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                    style={{ paddingLeft: '3rem' }}
                  >
                    <option value="RU">RU</option>
                    <option value="KZ">KZ</option>
                  </select>
                </div>
              </div>

              <div>
                <label className="block text-foreground mb-2">Timezone</label>
                <input
                  type="text"
                  value={form.timezone}
                  onChange={(event) => setField('timezone', event.target.value)}
                  placeholder="Europe/Moscow"
                  className="w-full px-4 py-3 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                  required
                />
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-foreground mb-2">Tax residency</label>
                <select
                  value={form.taxResidencyStatus}
                  onChange={(event) => setField('taxResidencyStatus', event.target.value)}
                  className="w-full px-4 py-3 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                >
                  <option value="RESIDENT">RESIDENT</option>
                  <option value="NON_RESIDENT">NON_RESIDENT</option>
                </select>
              </div>

              <div>
                <label className="block text-foreground mb-2">Taxpayer type</label>
                <select
                  value={form.taxpayerType}
                  onChange={(event) => setField('taxpayerType', event.target.value)}
                  className="w-full px-4 py-3 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
                >
                  <option value="INDIVIDUAL">INDIVIDUAL</option>
                  <option value="SOLE_PROPRIETOR">SOLE_PROPRIETOR</option>
                  <option value="LEGAL_ENTITY">LEGAL_ENTITY</option>
                </select>
              </div>
            </div>

            <div>
              <label className="block text-foreground mb-2">Phone</label>
              <input
                type="text"
                value={form.phone}
                onChange={(event) => setField('phone', event.target.value)}
                placeholder="+7 (495) 123-45-67"
                className="w-full px-4 py-3 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all"
              />
            </div>

            <div>
              <label className="block text-foreground mb-2">Wallets</label>
              <textarea
                rows={4}
                value={form.wallets}
                onChange={(event) => setField('wallets', event.target.value)}
                placeholder="0xabc..., 0xdef..."
                className="w-full px-4 py-3 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all resize-y"
              />
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
              className="w-full flex items-center justify-center gap-2 py-3 bg-primary hover:bg-primary-dark text-primary-foreground rounded-lg font-medium transition-all disabled:opacity-50"
            >
              {isSubmitting ? (
                <>
                  <div className="w-5 h-5 border-2 border-primary-foreground/30 border-t-primary-foreground rounded-full animate-spin" />
                  Creating account...
                </>
              ) : (
                <>
                  Create account
                  <ArrowRight className="w-5 h-5" />
                </>
              )}
            </button>
          </form>

          <div className="mt-8 text-center">
            <p className="text-muted-foreground">
              Already have an account?{' '}
              <Link to="/login" className="text-primary hover:text-primary-dark font-medium transition-colors">
                Sign in
              </Link>
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
