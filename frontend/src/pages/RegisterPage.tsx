import { useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import type { RegisterRequest } from '../api/authService'
import { useAuth } from '../auth/AuthContext'
import { useNotifications } from '../components/notifications/NotificationProvider'
import { LoadingState } from '../components/states/LoadingState'
import { toErrorMessages } from '../utils/errors'

interface RegistrationFormState {
  email: string
  password: string
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

  const [form, setForm] = useState<RegistrationFormState>(INITIAL_FORM)
  const [errors, setErrors] = useState<string[]>([])
  const [isSubmitting, setIsSubmitting] = useState(false)

  const payload = useMemo<RegisterRequest>(() => toRegisterPayload(form), [form])

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

  return (
    <main className="auth-page auth-page-register">
      <section className="auth-card auth-card-register">
        <header>
          <h1>Create account</h1>
          <p>Register a tenant account and initial tax profile for the demo flow.</p>
        </header>

        <form onSubmit={handleSubmit} className="form-grid two-columns">
          <label>
            Email
            <input
              type="email"
              required
              value={form.email}
              onChange={(event) => setField('email', event.target.value)}
              autoComplete="email"
            />
          </label>

          <label>
            Password
            <input
              type="password"
              required
              value={form.password}
              onChange={(event) => setField('password', event.target.value)}
              autoComplete="new-password"
            />
          </label>

          <label>
            INN
            <input required value={form.inn} onChange={(event) => setField('inn', event.target.value)} />
          </label>

          <label>
            Jurisdiction
            <select value={form.jurisdiction} onChange={(event) => setField('jurisdiction', event.target.value)}>
              <option value="RU">RU</option>
              <option value="KZ">KZ</option>
            </select>
          </label>

          <label>
            Last name
            <input required value={form.lastName} onChange={(event) => setField('lastName', event.target.value)} />
          </label>

          <label>
            First name
            <input required value={form.firstName} onChange={(event) => setField('firstName', event.target.value)} />
          </label>

          <label>
            Middle name
            <input required value={form.middleName} onChange={(event) => setField('middleName', event.target.value)} />
          </label>

          <label>
            Timezone
            <input required value={form.timezone} onChange={(event) => setField('timezone', event.target.value)} />
          </label>

          <label>
            Tax residency
            <select value={form.taxResidencyStatus} onChange={(event) => setField('taxResidencyStatus', event.target.value)}>
              <option value="RESIDENT">RESIDENT</option>
              <option value="NON_RESIDENT">NON_RESIDENT</option>
            </select>
          </label>

          <label>
            Taxpayer type
            <select value={form.taxpayerType} onChange={(event) => setField('taxpayerType', event.target.value)}>
              <option value="INDIVIDUAL">INDIVIDUAL</option>
              <option value="SOLE_PROPRIETOR">SOLE_PROPRIETOR</option>
              <option value="LEGAL_ENTITY">LEGAL_ENTITY</option>
            </select>
          </label>

          <label className="column-full">
            Phone
            <input value={form.phone} onChange={(event) => setField('phone', event.target.value)} />
          </label>

          <label className="column-full">
            Wallets (comma or new line separated)
            <textarea
              rows={3}
              value={form.wallets}
              onChange={(event) => setField('wallets', event.target.value)}
              placeholder="0xabc..., 0xdef..."
            />
          </label>

          {errors.length > 0 ? (
            <div className="form-error column-full">
              <ul className="form-error-list">
                {errors.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
          ) : null}

          <div className="column-full actions-row">
            <button type="submit" className="btn-primary" disabled={isSubmitting}>
              {isSubmitting ? 'Submitting...' : 'Register'}
            </button>
          </div>
        </form>

        <footer className="auth-footer">
          <p>Already registered?</p>
          <Link to="/login">Back to login</Link>
        </footer>
      </section>
    </main>
  )
}
