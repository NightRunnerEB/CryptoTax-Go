import { BrowserRouter } from 'react-router-dom'
import { AppRouter } from './app/AppRouter'
import { AuthProvider } from './auth/AuthContext'
import { NotificationProvider } from './components/notifications/NotificationProvider'

function App() {
  return (
    <NotificationProvider>
      <AuthProvider>
        <BrowserRouter>
          <AppRouter />
        </BrowserRouter>
      </AuthProvider>
    </NotificationProvider>
  )
}

export default App
