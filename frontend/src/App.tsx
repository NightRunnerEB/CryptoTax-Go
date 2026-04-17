import { BrowserRouter } from 'react-router-dom'
import { AppRouter } from './app/AppRouter'
import { AuthProvider } from './auth/AuthContext'
import { NotificationProvider } from './components/notifications/NotificationProvider'
import { ThemeProvider } from './components/theme/ThemeProvider'

function App() {
  return (
    <ThemeProvider>
      <NotificationProvider>
        <AuthProvider>
          <BrowserRouter>
            <AppRouter />
          </BrowserRouter>
        </AuthProvider>
      </NotificationProvider>
    </ThemeProvider>
  )
}

export default App
