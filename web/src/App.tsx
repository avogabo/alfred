import { useEffect, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { AlertCircle, CheckCircle2, X } from 'lucide-react'
import { Header, TabType } from './components/Header'
import { WinstonView } from './components/WinstonView'
import { GeoffreyView } from './components/GeoffreyView'
import { SettingsView } from './components/SettingsView'

type Notification = {
  id: string
  message: string
  type: 'success' | 'error'
}

type GlobalStatus = {
  plexConnected: boolean
  pendingReviews: number
  activeCollections: number
}

export default function App() {
  const [activeTab, setActiveTab] = useState<TabType>('winston')
  const [status, setStatus] = useState<GlobalStatus | null>(null)
  const [notifications, setNotifications] = useState<Notification[]>([])

  useEffect(() => {
    void fetchStatus()
    const timer = setInterval(() => {
      void fetchStatus()
    }, 15000)
    return () => clearInterval(timer)
  }, [])

  async function fetchStatus() {
    try {
      const res = await fetch('/api/status')
      if (res.ok) {
        const data = await res.json()
        setStatus({
          plexConnected: Boolean(data.plex_connected),
          pendingReviews: Number(data.pending_reviews || 0),
          activeCollections: Number(data.active_collections || 0),
        })
      }
    } catch {
      // offline or server starting
    }
  }

  function addNotification(message: string, type: 'success' | 'error') {
    const id = Math.random().toString(36).substring(2, 9)
    setNotifications((prev) => [...prev, { id, message, type }])
    setTimeout(() => {
      setNotifications((prev) => prev.filter((n) => n.id !== id))
    }, 4000)
  }

  function removeNotification(id: string) {
    setNotifications((prev) => prev.filter((n) => n.id !== id))
  }

  return (
    <div className="app-container">
      <Header activeTab={activeTab} onTabChange={setActiveTab} status={status} />

      {/* Floating Toasts */}
      <div style={{ position: 'fixed', top: '1.5rem', right: '1.5rem', zIndex: 1000, display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
        <AnimatePresence>
          {notifications.map((n) => (
            <motion.div
              key={n.id}
              initial={{ opacity: 0, x: 20, scale: 0.95 }}
              animate={{ opacity: 1, x: 0, scale: 1 }}
              exit={{ opacity: 0, x: 20, scale: 0.95 }}
              className={`banner ${n.type}`}
              style={{
                boxShadow: '0 8px 24px rgba(0,0,0,0.4)',
                minWidth: '280px',
                justifyContent: 'space-between',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                {n.type === 'success' ? (
                  <CheckCircle2 size={16} color="#34d399" />
                ) : (
                  <AlertCircle size={16} color="#fb7185" />
                )}
                <span>{n.message}</span>
              </div>
              <button
                onClick={() => removeNotification(n.id)}
                style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'inherit', display: 'flex' }}
              >
                <X size={14} />
              </button>
            </motion.div>
          ))}
        </AnimatePresence>
      </div>

      {/* Tab Panels */}
      <main style={{ marginTop: '0.5rem' }}>
        <AnimatePresence mode="wait">
          {activeTab === 'winston' && (
            <motion.div
              key="winston"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              transition={{ duration: 0.2 }}
            >
              <WinstonView onNotify={addNotification} onRefreshGlobalStatus={fetchStatus} />
            </motion.div>
          )}

          {activeTab === 'geoffrey' && (
            <motion.div
              key="geoffrey"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              transition={{ duration: 0.2 }}
            >
              <GeoffreyView onNotify={addNotification} onRefreshGlobalStatus={fetchStatus} />
            </motion.div>
          )}

          {activeTab === 'settings' && (
            <motion.div
              key="settings"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              transition={{ duration: 0.2 }}
            >
              <SettingsView onNotify={addNotification} onRefreshGlobalStatus={fetchStatus} />
            </motion.div>
          )}
        </AnimatePresence>
      </main>
    </div>
  )
}
