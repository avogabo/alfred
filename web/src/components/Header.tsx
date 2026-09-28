import { Clapperboard, FolderSync, Layers, Server, Settings2, Sparkles } from 'lucide-react'

export type TabType = 'winston' | 'geoffrey' | 'settings'

type HeaderProps = {
  activeTab: TabType
  onTabChange: (tab: TabType) => void
  status: {
    plexConnected: boolean
    pendingReviews: number
    activeCollections: number
  } | null
}

export function Header({ activeTab, onTabChange, status }: HeaderProps) {
  return (
    <header className="top-navbar glass">
      <div className="brand-section">
        <div className="brand-icon">
          <Layers size={22} color="#ffffff" />
        </div>
        <div className="brand-titles">
          <h1>ALFRED</h1>
          <span>The Media Butler</span>
        </div>
      </div>

      <nav className="nav-tabs">
        <button
          className={`nav-tab-btn ${activeTab === 'winston' ? 'active winston' : ''}`}
          onClick={() => onTabChange('winston')}
          id="tab-winston"
        >
          <FolderSync size={16} />
          <span>Winston</span>
          {status && status.pendingReviews > 0 ? (
            <span className="pill needs_review" style={{ padding: '0.1rem 0.4rem', fontSize: '0.65rem' }}>
              {status.pendingReviews}
            </span>
          ) : null}
        </button>

        <button
          className={`nav-tab-btn ${activeTab === 'geoffrey' ? 'active geoffrey' : ''}`}
          onClick={() => onTabChange('geoffrey')}
          id="tab-geoffrey"
        >
          <Clapperboard size={16} />
          <span>Geoffrey</span>
          {status && status.activeCollections > 0 ? (
            <span className="pill approved" style={{ padding: '0.1rem 0.4rem', fontSize: '0.65rem' }}>
              {status.activeCollections}
            </span>
          ) : null}
        </button>

        <button
          className={`nav-tab-btn ${activeTab === 'settings' ? 'active' : ''}`}
          onClick={() => onTabChange('settings')}
          id="tab-settings"
        >
          <Settings2 size={16} />
          <span>Ajustes</span>
        </button>
      </nav>

      <div className="navbar-status">
        <div className="status-pill" title={status?.plexConnected ? 'Plex conectado' : 'Plex no conectado o no configurado'}>
          <Server size={14} />
          <span>Plex</span>
          <div className={`dot ${status?.plexConnected ? 'online' : 'offline'}`} />
        </div>
      </div>
    </header>
  )
}
