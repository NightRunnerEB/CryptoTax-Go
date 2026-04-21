import { useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { FileBarChart, FileUp, LogOut, Menu, Moon, Receipt, Settings, Sun, X } from 'lucide-react'
import { useAuth } from '../../auth/AuthContext'
import { useNotifications } from '../notifications/NotificationProvider'
import { useTheme } from '../theme/ThemeProvider'

export function AppShell() {
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const { theme, toggleTheme } = useTheme()
  const { session, logout } = useAuth()
  const notifications = useNotifications()
  const navigate = useNavigate()

  const navItems = [
    { to: '/imports', label: 'CSV Imports', icon: FileUp, end: true },
    { to: '/transactions', label: 'Transactions', icon: Receipt },
    { to: '/reports', label: 'Tax Reports', icon: FileBarChart },
    { to: '/settings', label: 'Settings', icon: Settings },
  ]

  const handleLogout = async (): Promise<void> => {
    await logout()
    notifications.info('Session closed', 'You have been logged out.')
    navigate('/login', { replace: true })
  }

  return (
    <div className="min-h-screen bg-background flex">
      <aside className="w-64 bg-surface border-r border-border flex-col fixed h-full z-20 hidden lg:flex">
        <div className="p-6 border-b border-border">
          <h1 className="text-foreground">CryptoTax</h1>
        </div>

        <nav className="flex-1 p-4 space-y-1">
          {navItems.map((item) => {
            const Icon = item.icon
            return (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.end}
                className={({ isActive }) =>
                  `flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all duration-200 ${
                    isActive
                      ? 'bg-primary text-primary-foreground shadow-sm'
                      : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                  }`
                }
              >
                <Icon className="w-5 h-5" />
                <span className="font-medium text-sm">{item.label}</span>
              </NavLink>
            )
          })}
        </nav>
      </aside>

      {sidebarOpen ? (
        <>
          <div className="fixed inset-0 bg-black/50 z-30 lg:hidden" onClick={() => setSidebarOpen(false)} />
          <aside className="w-64 bg-surface border-r border-border flex flex-col fixed h-full z-40 lg:hidden">
            <div className="p-6 border-b border-border flex items-center justify-between">
              <h1 className="text-foreground">CryptoTax</h1>
              <button onClick={() => setSidebarOpen(false)}>
                <X className="w-5 h-5 text-muted-foreground" />
              </button>
            </div>

            <nav className="flex-1 p-4 space-y-1">
              {navItems.map((item) => {
                const Icon = item.icon
                return (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    end={item.end}
                    onClick={() => setSidebarOpen(false)}
                    className={({ isActive }) =>
                      `flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all duration-200 ${
                        isActive
                          ? 'bg-primary text-primary-foreground shadow-sm'
                          : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                      }`
                    }
                  >
                    <Icon className="w-5 h-5" />
                    <span className="font-medium text-sm">{item.label}</span>
                  </NavLink>
                )
              })}
            </nav>
          </aside>
        </>
      ) : null}

      <div className="flex-1 lg:ml-64 flex flex-col min-h-screen">
        <header className="h-16 bg-surface border-b border-border flex items-center justify-between px-4 lg:px-8 sticky top-0 z-10">
          <div className="flex items-center gap-4">
            <button onClick={() => setSidebarOpen(true)} className="lg:hidden p-2 hover:bg-muted rounded-lg transition-colors">
              <Menu className="w-5 h-5 text-foreground" />
            </button>
            <div className="w-2 h-2 rounded-full bg-primary animate-pulse" />
            <span className="text-sm text-muted-foreground hidden sm:inline">Live sync enabled</span>
          </div>

          <div className="flex items-center gap-4">
            <div className="text-sm hidden md:block">
              <span className="text-muted-foreground">Signed in as</span>
              <span className="ml-2 text-foreground font-medium">{session?.user.email ?? '—'}</span>
            </div>
            <button
              onClick={() => void handleLogout()}
              className="flex items-center gap-2 px-3 py-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
            >
              <LogOut className="w-4 h-4" />
              <span className="hidden sm:inline">Logout</span>
            </button>
            <button
              className="flex items-center gap-2 px-3 py-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
              onClick={toggleTheme}
            >
              {theme === 'dark' ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
              <span className="hidden sm:inline">Toggle Theme</span>
            </button>
          </div>
        </header>

        <main className="flex-1 p-4 sm:p-6 lg:p-8">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
