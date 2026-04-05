import { useCallback, useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { ApiError } from '../api/httpClient'
import { getTaxProfile, upsertTaxProfile, type TaxProfile, type TaxProfileInput } from '../api/taxService'
import { useAuth } from '../auth/AuthContext'
import { PageHeader } from '../components/layout/PageHeader'
import { useNotifications } from '../components/notifications/NotificationProvider'
import { EmptyState } from '../components/states/EmptyState'
import { ErrorState } from '../components/states/ErrorState'
import { LoadingState } from '../components/states/LoadingState'
import { toErrorMessage } from '../utils/errors'

interface ProfileFormState {
  inn: string
  lastName: string
  firstName: string
  middleName: string
  timezone: string
  phone: string
  walletsRaw: string
  taxResidencyStatus: string
  taxpayerType: string
}

const INITIAL_PROFILE_FORM: ProfileFormState = {
  inn: '',
  lastName: '',
  firstName: '',
  middleName: '',
  timezone: 'Europe/Moscow',
  phone: '',
  walletsRaw: '',
  taxResidencyStatus: 'RESIDENT',
  taxpayerType: 'INDIVIDUAL',
}

const TAX_RESIDENCY_OPTIONS = ['RESIDENT', 'NON_RESIDENT'] as const
const TAXPAYER_TYPE_OPTIONS = ['INDIVIDUAL', 'SOLE_PROPRIETOR', 'LEGAL_ENTITY'] as const

function splitWallets(raw: string): string[] {
  return raw
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
}

function toProfileForm(profile: TaxProfile): ProfileFormState {
  return {
    inn: profile.inn,
    lastName: profile.lastName,
    firstName: profile.firstName,
    middleName: profile.middleName,
    timezone: profile.timezone,
    phone: profile.phone,
    walletsRaw: profile.wallets.join('\n'),
    taxResidencyStatus: profile.taxResidencyStatus,
    taxpayerType: profile.taxpayerType,
  }
}

function toTaxProfileInput(form: ProfileFormState): TaxProfileInput {
  return {
    inn: form.inn.trim(),
    lastName: form.lastName.trim(),
    firstName: form.firstName.trim(),
    middleName: form.middleName.trim(),
    timezone: form.timezone.trim(),
    phone: form.phone.trim(),
    wallets: splitWallets(form.walletsRaw),
    taxResidencyStatus: form.taxResidencyStatus,
    taxpayerType: form.taxpayerType,
  }
}

function isValidTimeZone(value: string): boolean {
  try {
    Intl.DateTimeFormat(undefined, { timeZone: value })
    return true
  } catch {
    return false
  }
}

function validateForm(form: ProfileFormState): string[] {
  const errors: string[] = []

  if (form.inn.trim() === '') {
    errors.push('INN is required.')
  }

  if (form.lastName.trim() === '') {
    errors.push('Last name is required.')
  }

  if (form.firstName.trim() === '') {
    errors.push('First name is required.')
  }

  if (form.timezone.trim() === '') {
    errors.push('Timezone is required.')
  } else if (!isValidTimeZone(form.timezone.trim())) {
    errors.push('Timezone must be a valid IANA timezone (for example: Europe/Moscow).')
  }

  if (!TAX_RESIDENCY_OPTIONS.includes(form.taxResidencyStatus as (typeof TAX_RESIDENCY_OPTIONS)[number])) {
    errors.push('Tax residency status is invalid.')
  }

  if (!TAXPAYER_TYPE_OPTIONS.includes(form.taxpayerType as (typeof TAXPAYER_TYPE_OPTIONS)[number])) {
    errors.push('Taxpayer type is invalid.')
  }

  return errors
}

export function SettingsPage() {
  const { session } = useAuth()
  const notifications = useNotifications()

  const [profile, setProfile] = useState<TaxProfile | null>(null)
  const [profileForm, setProfileForm] = useState<ProfileFormState>(INITIAL_PROFILE_FORM)
  const [mode, setMode] = useState<'view' | 'edit'>('view')

  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [formErrors, setFormErrors] = useState<string[]>([])
  const [saveError, setSaveError] = useState<string | null>(null)

  const loadProfile = useCallback(async (): Promise<void> => {
    if (!session) {
      return
    }

    setIsLoading(true)
    setLoadError(null)

    try {
      const currentProfile = await getTaxProfile()
      setProfile(currentProfile)
      setProfileForm(toProfileForm(currentProfile))
      setMode('view')
    } catch (error) {
      if (error instanceof ApiError && error.status === 404) {
        setProfile(null)
        setProfileForm(INITIAL_PROFILE_FORM)
        setMode('view')
      } else {
        setLoadError(toErrorMessage(error, 'Unable to load tax profile.'))
      }
    } finally {
      setIsLoading(false)
    }
  }, [session])

  useEffect(() => {
    void loadProfile()
  }, [loadProfile])

  const startEditing = (): void => {
    setFormErrors([])
    setSaveError(null)

    if (profile) {
      setProfileForm(toProfileForm(profile))
    } else {
      setProfileForm(INITIAL_PROFILE_FORM)
    }

    setMode('edit')
  }

  const cancelEditing = (): void => {
    setFormErrors([])
    setSaveError(null)

    if (profile) {
      setProfileForm(toProfileForm(profile))
    } else {
      setProfileForm(INITIAL_PROFILE_FORM)
    }

    setMode('view')
  }

  const handleSaveProfile = async (event: FormEvent<HTMLFormElement>): Promise<void> => {
    event.preventDefault()

    if (!session) {
      return
    }

    const errors = validateForm(profileForm)
    if (errors.length > 0) {
      setFormErrors(errors)
      return
    }

    setIsSaving(true)
    setFormErrors([])
    setSaveError(null)

    try {
      const updated = await upsertTaxProfile(toTaxProfileInput(profileForm))
      setProfile(updated)
      setProfileForm(toProfileForm(updated))
      setMode('view')
      notifications.success('Tax profile saved')
    } catch (error) {
      setSaveError(toErrorMessage(error, 'Unable to save tax profile.'))
      notifications.error('Tax profile save failed', toErrorMessage(error))
    } finally {
      setIsSaving(false)
    }
  }

  if (isLoading) {
    return <LoadingState label="Loading tax profile..." />
  }

  return (
    <section className="stack-lg">
      <PageHeader
        title="Settings"
        description="Tax profile settings used by tax calculations and report generation."
      />

      {loadError ? <ErrorState message={loadError} actionLabel="Retry" onAction={() => void loadProfile()} /> : null}

      {!loadError && mode === 'view' && !profile ? (
        <article className="card">
          <EmptyState
            title="Tax profile is not configured"
            description="Create your profile to enable tax report calculations."
          />
          <div className="actions-row">
            <button type="button" className="btn-primary" onClick={startEditing}>
              Create profile
            </button>
          </div>
        </article>
      ) : null}

      {!loadError && mode === 'view' && profile ? (
        <article className="card">
          <div className="content-header">
            <div>
              <h3>Tax profile</h3>
              <p>Current profile values used by tax-svc.</p>
            </div>
            <div className="actions-row">
              <button type="button" className="btn-secondary" onClick={startEditing}>
                Edit profile
              </button>
            </div>
          </div>

          <dl className="details-grid">
            <dt>INN</dt>
            <dd>{profile.inn || '—'}</dd>
            <dt>Last name</dt>
            <dd>{profile.lastName || '—'}</dd>
            <dt>First name</dt>
            <dd>{profile.firstName || '—'}</dd>
            <dt>Middle name</dt>
            <dd>{profile.middleName || '—'}</dd>
            <dt>Timezone</dt>
            <dd>{profile.timezone || '—'}</dd>
            <dt>Phone</dt>
            <dd>{profile.phone || '—'}</dd>
            <dt>Tax residency status</dt>
            <dd>{profile.taxResidencyStatus || '—'}</dd>
            <dt>Taxpayer type</dt>
            <dd>{profile.taxpayerType || '—'}</dd>
            <dt>Wallets</dt>
            <dd>
              {profile.wallets.length > 0 ? (
                <ul className="profile-wallets-list">
                  {profile.wallets.map((wallet) => (
                    <li key={wallet} className="mono-text">
                      {wallet}
                    </li>
                  ))}
                </ul>
              ) : (
                '—'
              )}
            </dd>
          </dl>
        </article>
      ) : null}

      {!loadError && mode === 'edit' ? (
        <article className="card">
          <h3>{profile ? 'Edit tax profile' : 'Create tax profile'}</h3>

          {formErrors.length > 0 ? (
            <div className="form-error" role="alert">
              <ul className="form-error-list">
                {formErrors.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
          ) : null}

          {saveError ? <ErrorState message={saveError} /> : null}

          <form className="form-grid two-columns" onSubmit={handleSaveProfile}>
            <label>
              INN
              <input
                required
                value={profileForm.inn}
                onChange={(event) =>
                  setProfileForm((prev) => ({
                    ...prev,
                    inn: event.target.value,
                  }))
                }
              />
            </label>

            <label>
              Timezone
              <input
                required
                value={profileForm.timezone}
                onChange={(event) =>
                  setProfileForm((prev) => ({
                    ...prev,
                    timezone: event.target.value,
                  }))
                }
              />
            </label>

            <label>
              Last name
              <input
                required
                value={profileForm.lastName}
                onChange={(event) =>
                  setProfileForm((prev) => ({
                    ...prev,
                    lastName: event.target.value,
                  }))
                }
              />
            </label>

            <label>
              First name
              <input
                required
                value={profileForm.firstName}
                onChange={(event) =>
                  setProfileForm((prev) => ({
                    ...prev,
                    firstName: event.target.value,
                  }))
                }
              />
            </label>

            <label>
              Middle name
              <input
                value={profileForm.middleName}
                onChange={(event) =>
                  setProfileForm((prev) => ({
                    ...prev,
                    middleName: event.target.value,
                  }))
                }
              />
            </label>

            <label>
              Phone
              <input
                value={profileForm.phone}
                onChange={(event) =>
                  setProfileForm((prev) => ({
                    ...prev,
                    phone: event.target.value,
                  }))
                }
              />
            </label>

            <label>
              Tax residency status
              <select
                value={profileForm.taxResidencyStatus}
                onChange={(event) =>
                  setProfileForm((prev) => ({
                    ...prev,
                    taxResidencyStatus: event.target.value,
                  }))
                }
              >
                {TAX_RESIDENCY_OPTIONS.map((status) => (
                  <option key={status} value={status}>
                    {status}
                  </option>
                ))}
              </select>
            </label>

            <label>
              Taxpayer type
              <select
                value={profileForm.taxpayerType}
                onChange={(event) =>
                  setProfileForm((prev) => ({
                    ...prev,
                    taxpayerType: event.target.value,
                  }))
                }
              >
                {TAXPAYER_TYPE_OPTIONS.map((type) => (
                  <option key={type} value={type}>
                    {type}
                  </option>
                ))}
              </select>
            </label>

            <label className="column-full">
              Wallets
              <textarea
                rows={4}
                value={profileForm.walletsRaw}
                onChange={(event) =>
                  setProfileForm((prev) => ({
                    ...prev,
                    walletsRaw: event.target.value,
                  }))
                }
              />
              <span className="hint-text">One wallet per line or comma-separated.</span>
            </label>

            <div className="column-full modal-actions">
              <button type="button" className="btn-secondary" onClick={cancelEditing} disabled={isSaving}>
                Cancel
              </button>
              <button type="submit" className="btn-primary" disabled={isSaving}>
                {isSaving ? 'Saving...' : 'Save profile'}
              </button>
            </div>
          </form>
        </article>
      ) : null}
    </section>
  )
}
