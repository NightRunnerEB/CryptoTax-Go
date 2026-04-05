import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { LoadingState } from '../components/states/LoadingState'

interface LocationState {
  from?: string
}

export function ProtectedRoute() {
  const { isAuthenticated, isBootstrapping } = useAuth()
  const location = useLocation()

  if (isBootstrapping) {
    return (
      <main className="auth-page">
        <LoadingState label="Restoring session..." />
      </main>
    )
  }

  if (!isAuthenticated) {
    const state: LocationState = {
      from: `${location.pathname}${location.search}`,
    }

    return <Navigate to="/login" replace state={state} />
  }

  return <Outlet />
}
