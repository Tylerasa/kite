import Login from './routes/Login'
import Signup from './routes/Signup'
import Dashboard from './routes/Dashboard'
import Deposit from './routes/Deposit'
import Convert from './routes/Convert'
import Payout from './routes/Payout'
import Transactions from './routes/Transactions'
import TransactionDetail from './routes/TransactionDetail'

// Simple client-side router based on pathname
function getPage() {
  const path = window.location.pathname

  // Auth guard — redirect to login if no token
  const publicPaths = ['/login', '/signup', '/']
  const isLoggedIn = !!localStorage.getItem('kite_token')

  if (!isLoggedIn && !publicPaths.includes(path)) {
    window.location.href = '/login'
    return null
  }

  if (path === '/' || path === '/login') return <Login />
  if (path === '/signup') return <Signup />
  if (path === '/dashboard') return <Dashboard />
  if (path === '/deposit') return <Deposit />
  if (path === '/convert') return <Convert />
  if (path === '/payout') return <Payout />
  if (path === '/transactions') return <Transactions />
  if (path.startsWith('/transactions/')) return <TransactionDetail />

  // 404 fallback
  return (
    <div className="min-h-screen bg-white flex flex-col items-center justify-center font-sans text-[#1b1b1b]">
      <h2 className="text-2xl font-bold mb-3">Page not found</h2>
      <a href="/dashboard" className="text-sm font-medium text-[#1b1b1b] underline underline-offset-2">
        Go to dashboard →
      </a>
    </div>
  )
}

export default function App() {
  return getPage()
}
