import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../../auth/AuthContext'
import { useNotifications } from '../notifications/NotificationProvider'

interface NavItem {
  to: string
  label: string
  subtitle: string
}

const NAV_ITEMS: NavItem[] = [
  { to: '/imports', label: 'CSV Imports', subtitle: 'Ledger upload' },
  { to: '/transactions', label: 'Transactions', subtitle: 'Aggregated records' },
  { to: '/reports', label: 'Tax Reports', subtitle: 'Calculation jobs' },
  { to: '/settings', label: 'Settings', subtitle: 'Tax profile' },
]

const PAGE_META: Record<string, { title: string; description: string }> = {
  '/imports': {
    title: 'CSV Imports',
    description: 'Upload exchange statements and trigger parsing.',
  },
  '/transactions': {
    title: 'Transactions',
    description: 'Review aggregated transaction data by import identifier.',
  },
  '/reports': {
    title: 'Tax Reports',
    description: 'Start and monitor asynchronous tax report jobs.',
  },
  '/settings': {
    title: 'Settings',
    description: 'Maintain tax profile details used by tax calculations.',
  },
}

function resolveMeta(pathname: string): { title: string; description: string } {
  const exact = PAGE_META[pathname]
  if (exact) {
    return exact
  }

  const fallback = Object.entries(PAGE_META).find(([prefix]) => pathname.startsWith(prefix))
  if (fallback) {
    return fallback[1]
  }

  return {
    title: 'Workspace',
    description: 'Demo workflow for CryptoTax backend services.',
  }
}

export function AppShell() {
  const { session, logout } = useAuth()
  const notifications = useNotifications()
  const navigate = useNavigate()
  const location = useLocation()

  const pageMeta = resolveMeta(location.pathname)

  const handleLogout = async (): Promise<void> => {
    await logout()
    notifications.info('Session closed', 'You have been logged out.')
    navigate('/login', { replace: true })
  }

  return (
    <div className="shell">
      <header className="shell-header">
        <div>
          <p className="shell-brand">CryptoTax</p>
          <p className="shell-caption">Backend workflow demo</p>
        </div>
        <div className="shell-header-right">
          <div className="shell-user-block">
            <p>{session?.user.email}</p>
            <p className="shell-muted">{session?.user.status ?? '—'}</p>
          </div>
          <button type="button" className="btn-secondary" onClick={handleLogout}>
            Logout
          </button>
        </div>
      </header>

      <div className="shell-body">
        <aside className="shell-sidebar" aria-label="Primary navigation">
          <nav>
            <ul>
              {NAV_ITEMS.map((item) => (
                <li key={item.to}>
                  <NavLink to={item.to} className={({ isActive }) => `nav-link${isActive ? ' nav-link-active' : ''}`}>
                    <span>{item.label}</span>
                    <small>{item.subtitle}</small>
                  </NavLink>
                </li>
              ))}
            </ul>
          </nav>
        </aside>

        <main className="shell-main">
          <section className="page-title-bar">
            <h1>{pageMeta.title}</h1>
            <p>{pageMeta.description}</p>
          </section>
          <section className="page-content">
            <Outlet />
          </section>
        </main>
      </div>
    </div>
  )
}
