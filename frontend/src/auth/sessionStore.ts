const STORAGE_KEY = 'cryptotax.demo.session'

export type UserStatus = 'Active' | 'Pending' | 'Blocked'
export type SessionStatus = 'Active' | 'Revoked' | 'Closed'

export interface SessionUser {
  id: string
  email: string
  status: UserStatus
  createdAt: string
}

export interface SessionInfo {
  id: string
  userId: string
  status: SessionStatus
  createdAt: string
  lastSeenAt: string
  ip: string | null
  userAgent: string | null
}

export interface SessionTokens {
  accessToken: string
  refreshToken: string
  accessExpiresAt: number
  refreshExpiresAt: number
}

export interface AuthSessionState {
  user: SessionUser
  session: SessionInfo
  tokens: SessionTokens
  roles: string[]
}

type Listener = () => void

const listeners = new Set<Listener>()
let currentSession: AuthSessionState | null = readSessionFromStorage()

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object'
}

function readSessionFromStorage(): AuthSessionState | null {
  if (typeof window === 'undefined') {
    return null
  }

  const raw = window.localStorage.getItem(STORAGE_KEY)
  if (!raw) {
    return null
  }

  try {
    const parsed = JSON.parse(raw) as unknown
    if (!isRecord(parsed)) {
      return null
    }

    const user = parsed.user
    const session = parsed.session
    const tokens = parsed.tokens

    if (!isRecord(user) || !isRecord(session) || !isRecord(tokens)) {
      return null
    }

    if (
      typeof user.id !== 'string' ||
      typeof user.email !== 'string' ||
      typeof user.status !== 'string' ||
      typeof user.createdAt !== 'string'
    ) {
      return null
    }

    if (
      typeof session.id !== 'string' ||
      typeof session.userId !== 'string' ||
      typeof session.status !== 'string' ||
      typeof session.createdAt !== 'string' ||
      typeof session.lastSeenAt !== 'string'
    ) {
      return null
    }

    if (
      typeof tokens.accessToken !== 'string' ||
      typeof tokens.refreshToken !== 'string' ||
      typeof tokens.accessExpiresAt !== 'number' ||
      typeof tokens.refreshExpiresAt !== 'number'
    ) {
      return null
    }

    return {
      user: {
        id: user.id,
        email: user.email,
        status: user.status as UserStatus,
        createdAt: user.createdAt,
      },
      session: {
        id: session.id,
        userId: session.userId,
        status: session.status as SessionStatus,
        createdAt: session.createdAt,
        lastSeenAt: session.lastSeenAt,
        ip: typeof session.ip === 'string' ? session.ip : null,
        userAgent: typeof session.userAgent === 'string' ? session.userAgent : null,
      },
      tokens: {
        accessToken: tokens.accessToken,
        refreshToken: tokens.refreshToken,
        accessExpiresAt: tokens.accessExpiresAt,
        refreshExpiresAt: tokens.refreshExpiresAt,
      },
      roles: Array.isArray(parsed.roles) ? parsed.roles.filter((role): role is string => typeof role === 'string') : [],
    }
  } catch {
    return null
  }
}

function persistSession(session: AuthSessionState | null): void {
  if (typeof window === 'undefined') {
    return
  }

  if (!session) {
    window.localStorage.removeItem(STORAGE_KEY)
    return
  }

  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(session))
}

function notify(): void {
  listeners.forEach((listener) => {
    listener()
  })
}

export function getSession(): AuthSessionState | null {
  return currentSession
}

export function setSession(session: AuthSessionState): void {
  currentSession = session
  persistSession(currentSession)
  notify()
}

export function updateSessionTokens(tokens: SessionTokens): void {
  if (!currentSession) {
    return
  }

  currentSession = {
    ...currentSession,
    tokens,
  }

  persistSession(currentSession)
  notify()
}

export function clearSession(): void {
  currentSession = null
  persistSession(currentSession)
  notify()
}

export function subscribeSession(listener: Listener): () => void {
  listeners.add(listener)

  return () => {
    listeners.delete(listener)
  }
}

export function getSessionSnapshot(): AuthSessionState | null {
  return getSession()
}
