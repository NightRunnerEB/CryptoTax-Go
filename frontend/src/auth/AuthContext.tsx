/* eslint-disable react-refresh/only-export-components */
import {
  createContext,
  type PropsWithChildren,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  useSyncExternalStore,
} from 'react'
import type { LoginRequest, RegisterRequest } from '../api/authService'
import { loginUser, logoutUser, refreshAuthTokens, registerUser } from '../api/authService'
import { setRefreshHandler } from './refreshCoordinator'
import {
  clearSession,
  getSession,
  getSessionSnapshot,
  setSession,
  subscribeSession,
  type AuthSessionState,
} from './sessionStore'

interface AuthContextValue {
  session: AuthSessionState | null
  isAuthenticated: boolean
  isBootstrapping: boolean
  login: (input: Pick<LoginRequest, 'email' | 'password'>) => Promise<void>
  register: (payload: RegisterRequest) => Promise<void>
  logout: (options?: { remote?: boolean }) => Promise<void>
  refreshSession: () => Promise<boolean>
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined)

function toSessionState(result: Awaited<ReturnType<typeof loginUser>>): AuthSessionState {
  const now = Date.now()

  return {
    roles: [],
    user: {
      id: result.user.id,
      email: result.user.email,
      status: result.user.status,
      createdAt: result.user.created_at,
    },
    session: {
      id: result.session.id,
      userId: result.session.user_id,
      status: result.session.status,
      createdAt: result.session.created_at,
      lastSeenAt: result.session.last_seen_at,
      ip: result.session.ip,
      userAgent: result.session.user_agent,
    },
    tokens: {
      accessToken: result.tokens.access_token,
      refreshToken: result.tokens.refresh_token,
      accessExpiresAt: now + result.tokens.access_expires_in * 1000,
      refreshExpiresAt: now + result.tokens.refresh_expires_in * 1000,
    },
  }
}

export function AuthProvider({ children }: PropsWithChildren) {
  const session = useSyncExternalStore(subscribeSession, getSessionSnapshot, getSession)
  const [isBootstrapping, setIsBootstrapping] = useState(true)

  const refreshSession = useCallback(async (): Promise<boolean> => {
    const current = getSession()
    if (!current) {
      return false
    }

    if (Date.now() >= current.tokens.refreshExpiresAt) {
      clearSession()
      return false
    }

    try {
      const refreshed = await refreshAuthTokens({
        refresh_token: current.tokens.refreshToken,
      })

      setSession({
        ...current,
        tokens: {
          accessToken: refreshed.access_token,
          refreshToken: refreshed.refresh_token,
          accessExpiresAt: Date.now() + refreshed.access_expires_in * 1000,
          refreshExpiresAt: Date.now() + refreshed.refresh_expires_in * 1000,
        },
      })

      return true
    } catch {
      clearSession()
      return false
    }
  }, [])

  useEffect(() => {
    setRefreshHandler(refreshSession)
    return () => {
      setRefreshHandler(null)
    }
  }, [refreshSession])

  useEffect(() => {
    let cancelled = false

    const bootstrap = async (): Promise<void> => {
      const current = getSession()
      if (!current) {
        if (!cancelled) {
          setIsBootstrapping(false)
        }
        return
      }

      if (Date.now() >= current.tokens.refreshExpiresAt) {
        clearSession()
        if (!cancelled) {
          setIsBootstrapping(false)
        }
        return
      }

      const shouldRefresh = current.tokens.accessExpiresAt - Date.now() <= 30_000
      if (shouldRefresh) {
        await refreshSession()
      }

      if (!cancelled) {
        setIsBootstrapping(false)
      }
    }

    void bootstrap()

    return () => {
      cancelled = true
    }
  }, [refreshSession])

  const login = useCallback(async (input: Pick<LoginRequest, 'email' | 'password'>): Promise<void> => {
    const result = await loginUser({
      ...input,
      ip: null,
      ua: typeof navigator !== 'undefined' ? navigator.userAgent : null,
    })

    setSession(toSessionState(result))
  }, [])

  const register = useCallback(async (payload: RegisterRequest): Promise<void> => {
    await registerUser(payload)
  }, [])

  const logout = useCallback(async (options?: { remote?: boolean }): Promise<void> => {
    const shouldCallRemote = options?.remote ?? true

    if (shouldCallRemote) {
      try {
        await logoutUser()
      } catch {
        // Keep local logout deterministic even when remote call fails.
      }
    }

    clearSession()
  }, [])

  const value = useMemo<AuthContextValue>(() => {
    return {
      session,
      isAuthenticated: Boolean(session?.tokens.accessToken),
      isBootstrapping,
      login,
      register,
      logout,
      refreshSession,
    }
  }, [isBootstrapping, login, logout, refreshSession, register, session])

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext)

  if (!context) {
    throw new Error('useAuth must be used within AuthProvider')
  }

  return context
}
