import { useEffect, useState } from 'react'
import {
  Bot,
  CheckCircle,
  FileCode,
  HardDrive,
  Save,
  Server,
  Settings2,
  ShieldCheck,
  Sparkles,
} from 'lucide-react'

type SettingsDTO = {
  plex_base_url: string
  plex_token: string
  plex_default_library: string
  plex_path_from: string
  plex_path_to: string
  source_root: string
  altmount_base_url: string
  altmount_api_key: string
  altmount_path_from: string
  altmount_path_to: string
  altmount_staging_dir: string
  altmount_staging_path: string
  default_mode: string
  sleep_between_imports: string
  auto_import_medium: boolean
  movies_template: string
  series_template: string
  filebot_movie_format: string
  filebot_series_format: string
  filebot_db: string
  filebot_binary: string
  filebot_home: string
  data_dir: string
  tmdb_api_key: string
  telegram_bot_token: string
  time_zone: string
}

type FileBotStatus = {
  enabled: boolean
  available: boolean
  mode: string
  binary: string
  home: string
  db: string
  license_present: boolean
}

type SettingsViewProps = {
  onNotify?: (msg: string, type: 'success' | 'error') => void
  onRefreshGlobalStatus?: () => void
}

const defaultSettings: SettingsDTO = {
  plex_base_url: '',
  plex_token: '',
  plex_default_library: 'Películas',
  plex_path_from: '',
  plex_path_to: '',
  source_root: '',
  altmount_base_url: '',
  altmount_api_key: '',
  altmount_path_from: '',
  altmount_path_to: '',
  altmount_staging_dir: '',
  altmount_staging_path: '',
  default_mode: 'filebot',
  sleep_between_imports: '3s',
  auto_import_medium: true,
  movies_template: 'Peliculas/{quality}/{alpha}/{title} ({year})',
  series_template: 'Series/{alpha}/{series}/Temporada {season}/{series} - {episode}',
  filebot_movie_format: 'Peliculas/{plex}',
  filebot_series_format: 'Series/{plex}',
  filebot_db: 'TheMovieDB',
  filebot_binary: '/usr/local/bin/filebot',
  filebot_home: '/config/filebot',
  data_dir: '/config',
  tmdb_api_key: '',
  telegram_bot_token: '',
  time_zone: 'Europe/Madrid',
}

export function SettingsView({ onNotify, onRefreshGlobalStatus }: SettingsViewProps) {
  const [settings, setSettings] = useState<SettingsDTO>(defaultSettings)
  const [filebotStatus, setFilebotStatus] = useState<FileBotStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    void loadData()
  }, [])

  async function loadData() {
    setLoading(true)
    setError('')
    try {
      const [settingsRes, fbRes] = await Promise.all([
        fetch('/api/settings').then((r) => r.json()),
        fetch('/api/winston/filebot/status')
          .then((r) => r.json())
          .catch(() => null),
      ])
      setSettings(settingsRes)
      if (fbRes) setFilebotStatus(fbRes)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error cargando ajustes')
    } finally {
      setLoading(false)
    }
  }

  async function saveSettings() {
    setSaving(true)
    setError('')
    try {
      const res = await fetch('/api/settings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(settings),
      })
      if (!res.ok) throw new Error(await res.text())
      const updated = await res.json()
      setSettings(updated)
      onNotify?.('Configuración persistida y aplicada en el servidor.', 'success')
      onRefreshGlobalStatus?.()

      // Refresh Filebot status
      const fb = await fetch('/api/winston/filebot/status')
        .then((r) => r.json())
        .catch(() => null)
      if (fb) setFilebotStatus(fb)
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Error al guardar ajustes'
      setError(msg)
      onNotify?.(msg, 'error')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="tab-view-content" style={{ display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
      <section className="hero-banner glass">
        <div className="hero-info">
          <span className="hero-tag settings">
            <Settings2 size={14} /> Ajustes Unificados
          </span>
          <h2>Configuración de Alfred</h2>
          <p>
            Gestiona la conexión compartida a Plex, los parámetros de ingesta para Winston (AltMount & FileBot) y la configuración de curación de Geoffrey.
          </p>
        </div>
        <div>
          <button
            className="btn btn-primary"
            onClick={() => void saveSettings()}
            disabled={loading || saving}
            id="btn-save-settings"
          >
            <Save size={16} />
            <span>{saving ? 'Guardando...' : 'Guardar Ajustes'}</span>
          </button>
        </div>
      </section>

      {error && <div className="banner error">{error}</div>}

      <div className="settings-grid">
        {/* Plex Card (Shared) */}
        <div className="glass settings-card">
          <div className="settings-card-head">
            <h3>
              <Server size={18} color="#f59e0b" /> Plex Media Server
            </h3>
            <span className="pill approved">Compartido</span>
          </div>

          <div className="field">
            <span>Base URL de Plex</span>
            <input
              value={settings.plex_base_url}
              onChange={(e) => setSettings({ ...settings, plex_base_url: e.target.value })}
              placeholder="http://192.168.1.100:32400"
              id="settings-plex-url"
            />
          </div>

          <div className="field">
            <span>Plex Token</span>
            <input
              type="password"
              value={settings.plex_token}
              onChange={(e) => setSettings({ ...settings, plex_token: e.target.value })}
              placeholder="Token X-Plex-Token"
              id="settings-plex-token"
            />
          </div>

          <div className="field">
            <span>Biblioteca de películas por defecto</span>
            <input
              value={settings.plex_default_library}
              onChange={(e) => setSettings({ ...settings, plex_default_library: e.target.value })}
              placeholder="Películas"
            />
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.4rem' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.75rem' }}>
              <div className="field">
                <span>Ruta en Alfred (Path From)</span>
                <input
                  value={settings.plex_path_from}
                  onChange={(e) => setSettings({ ...settings, plex_path_from: e.target.value })}
                  placeholder="/media (o vacío)"
                />
                <small>Cómo ve Alfred tu carpeta multimedia.</small>
              </div>
              <div className="field">
                <span>Ruta en Plex (Path To)</span>
                <input
                  value={settings.plex_path_to}
                  onChange={(e) => setSettings({ ...settings, plex_path_to: e.target.value })}
                  placeholder="/data/movies (o vacío)"
                />
                <small>Cómo ve Plex esa misma carpeta en su servidor.</small>
              </div>
            </div>
            <small style={{ color: 'var(--text-dim)', fontSize: '0.75rem', marginTop: '0.15rem' }}>
              💡 <em>Traducción para refrescar contenido al instante en Plex. Déjalo vacío si Alfred y Plex usan las mismas rutas.</em>
            </small>
          </div>
        </div>

        {/* AltMount Card (Winston) */}
        <div className="glass settings-card">
          <div className="settings-card-head">
            <h3>
              <HardDrive size={18} color="#10b981" /> AltMount (Winston)
            </h3>
            <span className="pill" style={{ background: 'rgba(16, 185, 129, 0.15)', color: '#34d399' }}>Ingesta</span>
          </div>

          <div className="field">
            <span>AltMount Base URL</span>
            <input
              value={settings.altmount_base_url}
              onChange={(e) => setSettings({ ...settings, altmount_base_url: e.target.value })}
              placeholder="http://192.168.1.100:8989"
              id="settings-altmount-url"
            />
          </div>

          <div className="field">
            <span>AltMount API Key</span>
            <input
              type="password"
              value={settings.altmount_api_key}
              onChange={(e) => setSettings({ ...settings, altmount_api_key: e.target.value })}
              placeholder="API Key de AltMount"
              id="settings-altmount-key"
            />
          </div>

          <div className="field">
            <span>Ruta origen NZB (Source Root)</span>
            <input
              value={settings.source_root}
              onChange={(e) => setSettings({ ...settings, source_root: e.target.value })}
              placeholder="/data/nzb"
              id="settings-source-root"
            />
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.4rem' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.75rem' }}>
              <div className="field">
                <span>Ruta NZB en Alfred (Path From)</span>
                <input
                  value={settings.altmount_path_from}
                  onChange={(e) => setSettings({ ...settings, altmount_path_from: e.target.value })}
                  placeholder="/data/nzb (o vacío)"
                />
                <small>Carpeta donde Alfred lee los archivos .nzb.</small>
              </div>
              <div className="field">
                <span>Ruta NZB en AltMount (Path To)</span>
                <input
                  value={settings.altmount_path_to}
                  onChange={(e) => setSettings({ ...settings, altmount_path_to: e.target.value })}
                  placeholder="/config/.nzbs (o vacío)"
                />
                <small>Ruta donde AltMount accede a esa misma carpeta.</small>
              </div>
            </div>
            <small style={{ color: 'var(--text-dim)', fontSize: '0.75rem', marginTop: '0.15rem' }}>
              💡 <em>Permite que AltMount encuentre los NZB en su contenedor/máquina. Déjalo vacío si comparten la misma ruta.</em>
            </small>
          </div>

          <div className="field">
            <span>Espera entre imports</span>
            <input
              value={settings.sleep_between_imports}
              onChange={(e) => setSettings({ ...settings, sleep_between_imports: e.target.value })}
              placeholder="3s"
            />
          </div>
        </div>

        {/* FileBot Card (Winston) */}
        <div className="glass settings-card">
          <div className="settings-card-head">
            <h3>
              <FileCode size={18} color="#6366f1" /> FileBot (Renombrado)
            </h3>
            {filebotStatus ? (
              <span className={`pill ${filebotStatus.available ? 'approved' : 'failed'}`}>
                {filebotStatus.available ? 'Binario listo' : 'No disponible'}
              </span>
            ) : null}
          </div>

          <div className="field">
            <span>Ruta del binario FileBot</span>
            <input
              value={settings.filebot_binary}
              onChange={(e) => setSettings({ ...settings, filebot_binary: e.target.value })}
              placeholder="/usr/local/bin/filebot"
            />
          </div>

          <div className="field">
            <span>Directorio de datos (Licencia .psm)</span>
            <input
              value={settings.filebot_home}
              onChange={(e) => setSettings({ ...settings, filebot_home: e.target.value })}
              placeholder="/config/filebot"
            />
          </div>

          <div className="field">
            <span>Formato para Películas</span>
            <input
              value={settings.filebot_movie_format}
              onChange={(e) => setSettings({ ...settings, filebot_movie_format: e.target.value })}
              placeholder="Peliculas/{plex}"
            />
          </div>

          <div className="field">
            <span>Formato para Series</span>
            <input
              value={settings.filebot_series_format}
              onChange={(e) => setSettings({ ...settings, filebot_series_format: e.target.value })}
              placeholder="Series/{plex}"
            />
          </div>

          <div className="glass-soft" style={{ padding: '0.85rem', fontSize: '0.8rem', display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', color: filebotStatus?.license_present ? '#34d399' : '#fbbf24' }}>
              <ShieldCheck size={16} />
              <span>
                {filebotStatus?.license_present
                  ? 'Licencia de FileBot detectada en almacenamiento persistente'
                  : 'Licencia no detectada. Coloca license.psm en /config/filebot/'}
              </span>
            </div>
          </div>
        </div>

        {/* Geoffrey & Automation Card */}
        <div className="glass settings-card">
          <div className="settings-card-head">
            <h3>
              <Bot size={18} color="#f43f5e" /> Geoffrey & TMDb
            </h3>
            <span className="pill geoffrey" style={{ background: 'rgba(244, 63, 94, 0.15)', color: '#fb7185' }}>Curación</span>
          </div>

          <div className="field">
            <span>TheMovieDB (TMDb) API Key</span>
            <input
              type="password"
              value={settings.tmdb_api_key}
              onChange={(e) => setSettings({ ...settings, tmdb_api_key: e.target.value })}
              placeholder="API Key de TMDb v3"
              id="settings-tmdb-key"
            />
          </div>

          <div className="field">
            <span>Telegram Bot Token (Opcional)</span>
            <input
              type="password"
              value={settings.telegram_bot_token}
              onChange={(e) => setSettings({ ...settings, telegram_bot_token: e.target.value })}
              placeholder="123456:ABC-DEF..."
            />
          </div>

          <div className="field">
            <span>Zona horaria (TZ)</span>
            <input
              value={settings.time_zone}
              onChange={(e) => setSettings({ ...settings, time_zone: e.target.value })}
              placeholder="Europe/Madrid"
            />
          </div>

          <div className="field">
            <span>Directorio de datos (Memoria Geoffrey)</span>
            <input
              value={settings.data_dir}
              onChange={(e) => setSettings({ ...settings, data_dir: e.target.value })}
              placeholder="/config"
            />
          </div>

          <div className="glass-soft" style={{ padding: '0.85rem' }}>
            <label style={{ display: 'flex', alignItems: 'center', gap: '0.6rem', cursor: 'pointer', fontSize: '0.85rem' }}>
              <input
                type="checkbox"
                checked={settings.auto_import_medium}
                onChange={(e) => setSettings({ ...settings, auto_import_medium: e.target.checked })}
              />
              <span>Autoimportar elementos de confianza 'medium' sin conflicto</span>
            </label>
          </div>
        </div>
      </div>
    </div>
  )
}
