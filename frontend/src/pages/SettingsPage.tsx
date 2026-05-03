import { useCallback, useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { AlertCircle, Edit2, Save, User, X } from 'lucide-react'
import { ApiError } from '../api/httpClient'
import { getTaxProfile, upsertTaxProfile, type TaxProfile, type TaxProfileInput } from '../api/taxService'
import { useAuth } from '../auth/AuthContext'
import { useNotifications } from '../components/notifications/NotificationProvider'
import { toErrorMessage } from '../utils/errors'

interface ProfileFormState {
  inn: string
  oktmo: string
  lastName: string
  firstName: string
  middleName: string
  timezone: string
  phone: string
  wallets: string[]
  taxResidencyStatus: string
  taxpayerType: string
}

const INITIAL_PROFILE_FORM: ProfileFormState = {
  inn: '',
  oktmo: '',
  lastName: '',
  firstName: '',
  middleName: '',
  timezone: 'Europe/Moscow',
  phone: '',
  wallets: [],
  taxResidencyStatus: 'RESIDENT',
  taxpayerType: 'INDIVIDUAL',
}

const TAX_RESIDENCY_OPTIONS = ['RESIDENT', 'NON_RESIDENT'] as const
const TAXPAYER_TYPE_OPTIONS = ['INDIVIDUAL', 'SOLE_PROPRIETOR'] as const

function normalizeWallets(wallets: string[]): string[] {
  return wallets.map((wallet) => wallet.trim()).filter((wallet) => wallet.length > 0)
}

function toProfileForm(profile: TaxProfile): ProfileFormState {
  return {
    inn: profile.inn,
    oktmo: profile.oktmo,
    lastName: profile.lastName,
    firstName: profile.firstName,
    middleName: profile.middleName,
    timezone: profile.timezone,
    phone: profile.phone,
    wallets: profile.wallets,
    taxResidencyStatus: profile.taxResidencyStatus,
    taxpayerType: profile.taxpayerType,
  }
}

function toTaxProfileInput(form: ProfileFormState): TaxProfileInput {
  return {
    inn: form.inn.trim(),
    oktmo: form.oktmo.trim(),
    lastName: form.lastName.trim(),
    firstName: form.firstName.trim(),
    middleName: form.middleName.trim(),
    timezone: form.timezone.trim(),
    phone: form.phone.trim(),
    wallets: normalizeWallets(form.wallets),
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
  if (form.oktmo.trim() === '') {
    errors.push('OKTMO is required.')
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

function formatProfileLabel(value: string): string {
  if (!value) {
    return '—'
  }

  return value
    .toLowerCase()
    .replace(/_/g, ' ')
    .replace(/-/g, ' ')
    .replace(/(^|\s)\S/g, (match) => match.toUpperCase())
}

function fullName(profile: TaxProfile): string {
  return [profile.lastName, profile.firstName, profile.middleName].filter(Boolean).join(' ').trim() || '—'
}

export function SettingsPage() {
  const { session } = useAuth()
  const notifications = useNotifications()

  const [profile, setProfile] = useState<TaxProfile | null>(null)
  const [profileForm, setProfileForm] = useState<ProfileFormState>(INITIAL_PROFILE_FORM)
  const [isEditing, setIsEditing] = useState(false)

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
      setIsEditing(false)
    } catch (error) {
      if (error instanceof ApiError && error.status === 404) {
        setProfile(null)
        setProfileForm(INITIAL_PROFILE_FORM)
        setIsEditing(false)
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

  const handleEdit = (): void => {
    setFormErrors([])
    setSaveError(null)
    setProfileForm(profile ? toProfileForm(profile) : INITIAL_PROFILE_FORM)
    setIsEditing(true)
  }

  const handleCancel = (): void => {
    setFormErrors([])
    setSaveError(null)
    setProfileForm(profile ? toProfileForm(profile) : INITIAL_PROFILE_FORM)
    setIsEditing(false)
  }

  const handleWalletChange = (index: number, value: string): void => {
    setProfileForm((prev) => ({
      ...prev,
      wallets: prev.wallets.map((wallet, walletIndex) => (walletIndex === index ? value : wallet)),
    }))
  }

  const addWallet = (): void => {
    setProfileForm((prev) => ({
      ...prev,
      wallets: [...prev.wallets, ''],
    }))
  }

  const removeWallet = (index: number): void => {
    setProfileForm((prev) => ({
      ...prev,
      wallets: prev.wallets.filter((_, walletIndex) => walletIndex !== index),
    }))
  }

  const handleCreate = (): void => {
    setFormErrors([])
    setSaveError(null)
    setProfileForm(INITIAL_PROFILE_FORM)
    setIsEditing(true)
  }

  const handleSave = async (event: FormEvent<HTMLFormElement>): Promise<void> => {
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
      setIsEditing(false)
      notifications.success('Tax profile saved')
    } catch (error) {
      const message = toErrorMessage(error, 'Unable to save tax profile.')
      setSaveError(message)
      notifications.error('Tax profile save failed', message)
    } finally {
      setIsSaving(false)
    }
  }

  if (isLoading) {
    return (
      <div className="max-w-4xl">
        <div
          className="bg-surface rounded-xl border border-border p-8 flex items-center justify-center gap-3"
          style={{ boxShadow: 'var(--shadow-md)' }}
        >
          <div className="w-5 h-5 border-2 border-primary/30 border-t-primary rounded-full animate-spin" />
          <span className="text-sm text-muted-foreground">Loading tax profile...</span>
        </div>
      </div>
    )
  }

  if (loadError) {
    return (
      <div className="max-w-4xl">
        <div
          className="bg-surface rounded-xl border border-[var(--status-failed)]/30 p-5"
          style={{ boxShadow: 'var(--shadow-md)' }}
        >
          <p className="text-sm text-[var(--status-failed)]">{loadError}</p>
          <button
            type="button"
            onClick={() => void loadProfile()}
            className="mt-4 px-4 py-2 border border-border hover:bg-muted rounded-lg font-medium transition-all"
          >
            Retry
          </button>
        </div>
      </div>
    )
  }

  if (!profile && !isEditing) {
    return (
      <div className="max-w-4xl">
      <div className="mb-8 flex flex-col gap-6">
        <h2 className="text-foreground">Settings</h2>
        <p className="text-muted-foreground text-sm">Manage your tax profile and calculation preferences</p>
      </div>

        <div
          className="bg-surface rounded-xl border border-border p-12"
          style={{ boxShadow: 'var(--shadow-md)' }}
        >
          <div className="mx-auto flex max-w-md flex-col items-center text-center gap-4">
            <div className="w-16 h-16 bg-muted rounded-full flex items-center justify-center">
              <User className="w-8 h-8 text-muted-foreground" />
            </div>
            <h3 className="text-foreground">No Tax Profile Configured</h3>
            <p className="text-muted-foreground text-center max-w-md">
              Set up your tax profile to enable accurate tax calculations and reporting
            </p>
            <button
              type="button"
              onClick={handleCreate}
              className="mt-2 px-6 py-2.5 bg-primary hover:bg-primary-dark text-primary-foreground rounded-lg font-medium transition-all"
            >
              Create Tax Profile
            </button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="max-w-4xl">
      <div className="mb-8 flex items-start justify-between gap-4">
        <div className="flex flex-col gap-3">
          <h2 className="text-foreground">Settings</h2>
          <p className="text-muted-foreground text-sm">Manage your tax profile and calculation preferences</p>
        </div>

        {!isEditing ? (
          <button
            type="button"
            onClick={handleEdit}
            className="inline-flex items-center gap-2 px-4 py-2 bg-primary hover:bg-primary-dark text-primary-foreground rounded-lg text-sm font-medium transition-all whitespace-nowrap"
          >
            <Edit2 className="w-4 h-4" />
            Edit Profile
          </button>
        ) : null}
      </div>

      <div className="bg-surface rounded-xl border border-border overflow-hidden" style={{ boxShadow: 'var(--shadow-md)' }}>
        <div className="p-6 border-b border-border bg-surface-secondary/30">
          <h3 className="text-foreground">Tax Profile</h3>
        </div>

        <div className="p-8">
          {isEditing ? (
            <form className="space-y-6" onSubmit={handleSave}>
              {formErrors.length > 0 ? (
                <div className="p-4 rounded-lg border border-[var(--status-failed)]/30 bg-[var(--status-failed-bg)]">
                  <ul className="text-sm text-[var(--status-failed)] space-y-1 list-disc pl-5">
                    {formErrors.map((item) => (
                      <li key={item}>{item}</li>
                    ))}
                  </ul>
                </div>
              ) : null}

              {saveError ? <p className="text-sm text-[var(--status-failed)]">{saveError}</p> : null}

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
                <div>
                  <label className="block text-foreground mb-2">INN (Tax Identification Number)</label>
                  <input
                    type="text"
                    value={profileForm.inn}
                    onChange={(event) => setProfileForm((prev) => ({ ...prev, inn: event.target.value }))}
                    placeholder="7730123456789"
                    className="w-full px-4 py-2.5 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>

                <div>
                  <label className="block text-foreground mb-2">OKTMO</label>
                  <input
                    type="text"
                    value={profileForm.oktmo}
                    onChange={(event) => setProfileForm((prev) => ({ ...prev, oktmo: event.target.value }))}
                    placeholder="45382000"
                    className="w-full px-4 py-2.5 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>

                <div>
                  <label className="block text-foreground mb-2">Phone Number</label>
                  <input
                    type="tel"
                    value={profileForm.phone}
                    onChange={(event) => setProfileForm((prev) => ({ ...prev, phone: event.target.value }))}
                    placeholder="+7 (495) 123-45-67"
                    className="w-full px-4 py-2.5 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-3 gap-6">
                <div>
                  <label className="block text-foreground mb-2">Last Name</label>
                  <input
                    type="text"
                    value={profileForm.lastName}
                    onChange={(event) => setProfileForm((prev) => ({ ...prev, lastName: event.target.value }))}
                    placeholder="Ivanov"
                    className="w-full px-4 py-2.5 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
                <div>
                  <label className="block text-foreground mb-2">First Name</label>
                  <input
                    type="text"
                    value={profileForm.firstName}
                    onChange={(event) => setProfileForm((prev) => ({ ...prev, firstName: event.target.value }))}
                    placeholder="Alexey"
                    className="w-full px-4 py-2.5 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
                <div>
                  <label className="block text-foreground mb-2">Middle Name</label>
                  <input
                    type="text"
                    value={profileForm.middleName}
                    onChange={(event) => setProfileForm((prev) => ({ ...prev, middleName: event.target.value }))}
                    placeholder="Sergeevich"
                    className="w-full px-4 py-2.5 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-3 gap-6">
                <div>
                  <label className="block text-foreground mb-2">Timezone</label>
                  <input
                    type="text"
                    value={profileForm.timezone}
                    onChange={(event) => setProfileForm((prev) => ({ ...prev, timezone: event.target.value }))}
                    placeholder="Europe/Moscow"
                    className="w-full px-4 py-2.5 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>

                <div>
                  <label className="block text-foreground mb-2">Residency Status</label>
                  <select
                    value={profileForm.taxResidencyStatus}
                    onChange={(event) => setProfileForm((prev) => ({ ...prev, taxResidencyStatus: event.target.value }))}
                    className="w-full px-4 py-2.5 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  >
                    {TAX_RESIDENCY_OPTIONS.map((status) => (
                      <option key={status} value={status}>
                        {formatProfileLabel(status)}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-foreground mb-2">Taxpayer Type</label>
                  <select
                    value={profileForm.taxpayerType}
                    onChange={(event) => setProfileForm((prev) => ({ ...prev, taxpayerType: event.target.value }))}
                    className="w-full px-4 py-2.5 bg-input-background border border-input-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  >
                    {TAXPAYER_TYPE_OPTIONS.map((type) => (
                      <option key={type} value={type}>
                        {formatProfileLabel(type)}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="block text-foreground">Wallet Addresses</label>
                  <button type="button" onClick={addWallet} className="text-sm text-primary hover:text-primary-dark font-medium">
                    + Add Wallet
                  </button>
                </div>
                <div className="space-y-3">
                  {profileForm.wallets.map((wallet, index) => (
                    <div key={`wallet-${index}`} className="flex items-center gap-3">
                      <input
                        type="text"
                        value={wallet}
                        onChange={(event) => handleWalletChange(index, event.target.value)}
                        placeholder="0x... or bc1..."
                        className="flex-1 px-4 py-2.5 bg-input-background border border-input-border rounded-lg text-foreground font-mono text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                      />
                      <button
                        type="button"
                        onClick={() => removeWallet(index)}
                        className="p-2 hover:bg-muted rounded-lg transition-colors"
                        aria-label="Remove wallet"
                      >
                        <X className="w-4 h-4 text-muted-foreground" />
                      </button>
                    </div>
                  ))}
                  {profileForm.wallets.length === 0 ? (
                    <div className="p-4 bg-surface-secondary rounded-lg border border-dashed border-border text-center">
                      <p className="text-sm text-muted-foreground">No wallets added yet</p>
                    </div>
                  ) : null}
                </div>
              </div>

              <div className="pt-6 border-t border-border flex items-center gap-3">
                <button
                  type="submit"
                  className="flex items-center gap-2 px-6 py-2.5 bg-primary hover:bg-primary-dark text-primary-foreground rounded-lg font-medium transition-all disabled:opacity-50"
                  disabled={isSaving}
                >
                  <Save className="w-4 h-4" />
                  {isSaving ? 'Saving...' : 'Save Changes'}
                </button>
                <button
                  type="button"
                  onClick={handleCancel}
                  className="px-6 py-2.5 border border-border hover:bg-muted rounded-lg font-medium transition-all"
                  disabled={isSaving}
                >
                  Cancel
                </button>
              </div>
            </form>
          ) : profile ? (
            <div className="space-y-8">
              <div>
                <h4 className="text-muted-foreground text-xs uppercase tracking-wider mb-0">Personal Information</h4>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-12 gap-y-4 mt-4">
                <div>
                  <div className="text-sm text-muted-foreground mb-1">INN</div>
                  <div className="text-foreground font-medium font-mono">{profile.inn || '—'}</div>
                </div>
                <div>
                  <div className="text-sm text-muted-foreground mb-1">OKTMO</div>
                  <div className="text-foreground font-medium font-mono">{profile.oktmo || '—'}</div>
                </div>
                <div>
                  <div className="text-sm text-muted-foreground mb-1">Phone</div>
                  <div className="text-foreground font-medium">{profile.phone || '—'}</div>
                </div>
                  <div>
                    <div className="text-sm text-muted-foreground mb-1">Full Name</div>
                    <div className="text-foreground font-medium">{fullName(profile)}</div>
                  </div>
                  <div>
                    <div className="text-sm text-muted-foreground mb-1">Timezone</div>
                    <div className="text-foreground font-medium">{profile.timezone || '—'}</div>
                  </div>
                </div>
              </div>

              <div className="pt-8 border-t border-border">
                <h4 className="text-muted-foreground text-xs uppercase tracking-wider mb-0">Tax Configuration</h4>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-12 gap-y-4 mt-4">
                  <div>
                    <div className="text-sm text-muted-foreground mb-1">Residency Status</div>
                    <div className="text-foreground font-medium">{formatProfileLabel(profile.taxResidencyStatus)}</div>
                  </div>
                  <div>
                    <div className="text-sm text-muted-foreground mb-1">Taxpayer Type</div>
                    <div className="text-foreground font-medium">{formatProfileLabel(profile.taxpayerType)}</div>
                  </div>
                </div>
              </div>

              <div className="pt-8 border-t border-border">
                <h4 className="text-muted-foreground text-xs uppercase tracking-wider mb-0">Registered Wallets</h4>
                <div className="space-y-2 mt-8">
                  {profile.wallets.length > 0 ? (
                    profile.wallets.map((wallet, index) => (
                      <div key={`${wallet}-${index}`} className="p-3 bg-surface-secondary rounded-lg border border-border">
                        <div className="text-sm font-mono text-foreground break-all">{wallet}</div>
                      </div>
                    ))
                  ) : (
                    <div className="p-4 bg-surface-secondary rounded-lg border border-dashed border-border text-center">
                      <p className="text-sm text-muted-foreground">No wallets added yet</p>
                    </div>
                  )}
                </div>
              </div>
            </div>
          ) : null}
        </div>
      </div>

      <div className="mt-6 p-4 bg-[var(--status-running-bg)] border border-[var(--status-running)] rounded-lg flex items-start gap-3">
        <AlertCircle className="w-5 h-5 text-[var(--status-running)] flex-shrink-0 mt-0.5" />
        <div>
          <h4 className="text-sm font-medium mb-1" style={{ color: '#111111' }}>Profile Used for Tax Calculations</h4>
          <p className="text-sm" style={{ color: '#111111' }}>
            Changes to your tax profile will be captured in the policy snapshot when you create new tax reports.
            Existing reports will continue to use the profile settings from when they were created.
          </p>
        </div>
      </div>
    </div>
  )
}
