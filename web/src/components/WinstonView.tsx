import { useEffect, useMemo, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import {
  AlertTriangle,
  CheckCircle2,
  Film,
  FolderTree,
  LoaderCircle,
  RefreshCw,
  Search,
  Sparkles,
  Trash2,
  Tv,
  Upload,
  Wand2,
} from 'lucide-react'

type ItemMetadata = {
  tmdb_id: number
  tvdb_id: number
  imdb_id: string
  kind: string
  title: string
  year: number
  season: number
  episode: number
  quality: string
  relative_path_override: string
}

type CandidateMatch = {
  label: string
  kind: string
  tmdb_id: number
  tvdb_id: number
  imdb_id: string
  year: number
  reason: string
  score: number
}

type ReviewItem = {
  source_nzb_path: string
  state: 'needs_review' | 'approved' | 'imported' | 'corrected' | 'failed' | 'importing' | 'detected'
  confidence: 'high' | 'medium' | 'low'
  metadata: ItemMetadata
  proposed_path: string
  reason: string
  candidates: CandidateMatch[]
  status?: string
}

type WinstonViewProps = {
  onNotify?: (msg: string, type: 'success' | 'error') => void
  onRefreshGlobalStatus?: () => void
}

export function WinstonView({ onNotify, onRefreshGlobalStatus }: WinstonViewProps) {
  const [items, setItems] = useState<ReviewItem[]>([])
  const [query, setQuery] = useState('')
  const [selectedSource, setSelectedSource] = useState('')
  const [selected, setSelected] = useState<ReviewItem | null>(null)
  const [loading, setLoading] = useState(true)
  const [reloading, setReloading] = useState(false)
  const [rescanning, setRescanning] = useState(false)
  const [saving, setSaving] = useState(false)
  const [importing, setImporting] = useState(false)
  const [error, setError] = useState('')

  const [tmdbId, setTmdbId] = useState('')
  const [pathOverride, setPathOverride] = useState('')

  useEffect(() => {
    void loadItems()
  }, [])

  useEffect(() => {
    if (!selectedSource && items.length > 0) {
      setSelectedSource(items[0].source_nzb_path)
    }
  }, [items, selectedSource])

  useEffect(() => {
    if (!selectedSource) {
      setSelected(null)
      return
    }
    void loadItem(selectedSource)
  }, [selectedSource])

  useEffect(() => {
    if (!selected) return
    setTmdbId(selected.metadata.tmdb_id ? String(selected.metadata.tmdb_id) : '')
    setPathOverride(selected.metadata.relative_path_override || '')
  }, [selected])

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return items
    return items.filter((item) =>
      [
        item.metadata?.title,
        item.source_nzb_path,
        item.proposed_path,
        item.reason,
        item.state,
        item.confidence,
      ]
        .filter(Boolean)
        .join(' ')
        .toLowerCase()
        .includes(q),
    )
  }, [items, query])

  async function loadItems() {
    setLoading(true)
    setError('')
    try {
      const res = await fetch('/api/winston/review/items')
      if (!res.ok) throw new Error('No se pudo cargar la lista de items')
      const data = await res.json()
      setItems(data.items || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error cargando items')
    } finally {
      setLoading(false)
    }
  }

  async function loadItem(source: string) {
    try {
      const res = await fetch(`/api/winston/review/item?source=${encodeURIComponent(source)}`)
      if (!res.ok) throw new Error('No se pudo cargar el detalle del item')
      const data: ReviewItem = await res.json()
      setSelected(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error cargando detalle')
    }
  }

  async function rescanItems() {
    setRescanning(true)
    setError('')
    try {
      const res = await fetch('/api/winston/review/rescan', { method: 'POST' })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      setItems(data.items || [])
      if (!selectedSource && data.items?.length > 0) {
        setSelectedSource(data.items[0].source_nzb_path)
      }
      onNotify?.('Escaneo completado. Se han descubierto nuevos NZB.', 'success')
      onRefreshGlobalStatus?.()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Error al buscar nuevos NZB'
      setError(msg)
      onNotify?.(msg, 'error')
    } finally {
      setRescanning(false)
    }
  }

  async function clearAndRescan() {
    setRescanning(true)
    setError('')
    try {
      const res = await fetch('/api/winston/review/rescan?clear=true', { method: 'POST' })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      setItems(data.items || [])
      setSelectedSource(data.items?.[0]?.source_nzb_path || '')
      onNotify?.('Lista reseteada y carpeta reanalizada con éxito.', 'success')
      onRefreshGlobalStatus?.()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Error al limpiar y reanalizar'
      setError(msg)
      onNotify?.(msg, 'error')
    } finally {
      setRescanning(false)
    }
  }

  async function clearAllItems() {
    if (!window.confirm('¿Estás seguro de que deseas vaciar toda la lista de Winston?')) return
    setLoading(true)
    setError('')
    try {
      const res = await fetch('/api/winston/review/clear', { method: 'POST' })
      if (!res.ok) throw new Error(await res.text())
      setItems([])
      setSelectedSource('')
      onNotify?.('Lista de Winston vaciada correctamente.', 'success')
      onRefreshGlobalStatus?.()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Error al vaciar la lista'
      setError(msg)
      onNotify?.(msg, 'error')
    } finally {
      setLoading(false)
    }
  }

  async function applyCorrection(payload: Record<string, unknown>) {
    if (!selected) return
    setSaving(true)
    setError('')
    try {
      const res = await fetch(
        `/api/winston/review/correct?source=${encodeURIComponent(selected.source_nzb_path)}`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        },
      )
      if (!res.ok) throw new Error(await res.text())
      await loadItems()
      await loadItem(selected.source_nzb_path)
      onNotify?.('Corrección aplicada correctamente.', 'success')
      onRefreshGlobalStatus?.()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Error al aplicar corrección'
      setError(msg)
      onNotify?.(msg, 'error')
    } finally {
      setSaving(false)
    }
  }

  async function approveSelected() {
    if (!selected) return
    setSaving(true)
    setError('')
    try {
      const res = await fetch(
        `/api/winston/review/approve?source=${encodeURIComponent(selected.source_nzb_path)}`,
        { method: 'POST' },
      )
      if (!res.ok) throw new Error(await res.text())
      await loadItems()
      await loadItem(selected.source_nzb_path)
      onNotify?.('Item aprobado para importación.', 'success')
      onRefreshGlobalStatus?.()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Error al aprobar item'
      setError(msg)
      onNotify?.(msg, 'error')
    } finally {
      setSaving(false)
    }
  }

  async function approveAndImportSelected() {
    if (!selected) return
    setImporting(true)
    setError('')
    try {
      await fetch(
        `/api/winston/review/approve?source=${encodeURIComponent(selected.source_nzb_path)}`,
        { method: 'POST' },
      )
      const res = await fetch(
        `/api/winston/review/import?source=${encodeURIComponent(selected.source_nzb_path)}`,
        { method: 'POST' },
      )
      if (!res.ok) {
        const body = await res.text()
        throw new Error(body)
      }
      await loadItems()
      await loadItem(selected.source_nzb_path)
      onNotify?.('Aprobado y enviado a AltMount con éxito.', 'success')
      onRefreshGlobalStatus?.()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Error al procesar e importar item'
      setError(msg)
      onNotify?.(msg, 'error')
      await loadItems()
      await loadItem(selected.source_nzb_path)
    } finally {
      setImporting(false)
    }
  }

  async function importSelected() {
    if (!selected) return
    setImporting(true)
    setError('')
    try {
      const res = await fetch(
        `/api/winston/review/import?source=${encodeURIComponent(selected.source_nzb_path)}`,
        { method: 'POST' },
      )
      if (!res.ok) throw new Error(await res.text())
      await loadItems()
      await loadItem(selected.source_nzb_path)
      onNotify?.('Importación enviada a AltMount con éxito.', 'success')
      onRefreshGlobalStatus?.()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Error al importar item'
      setError(msg)
      onNotify?.(msg, 'error')
    } finally {
      setImporting(false)
    }
  }

  async function reloadSelected() {
    if (!selected) return
    setReloading(true)
    try {
      await loadItems()
      await loadItem(selected.source_nzb_path)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error al recargar')
    } finally {
      setReloading(false)
    }
  }

  async function resetSelected() {
    if (!selected) return
    try {
      const res = await fetch(
        `/api/winston/review/reset?source=${encodeURIComponent(selected.source_nzb_path)}`,
        { method: 'POST' },
      )
      if (!res.ok) throw new Error(await res.text())
      setSelectedSource('')
      await loadItems()
      onNotify?.('Item descartado de la lista de revisión.', 'success')
      onRefreshGlobalStatus?.()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Error al descartar item'
      setError(msg)
      onNotify?.(msg, 'error')
    }
  }

  const reviewCount = items.filter(
    (item) => item.state === 'needs_review' || item.state === 'detected',
  ).length
  const approvedCount = items.filter(
    (item) => item.state === 'approved' || item.state === 'corrected',
  ).length
  const importedCount = items.filter((item) => item.state === 'imported').length

  return (
    <div className="tab-view-content" style={{ display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
      <section className="hero-banner glass">
        <div className="hero-info">
          <span className="hero-tag winston">
            <Sparkles size={14} /> Subsección Winston
          </span>
          <h2>Ingesta, Staging y FileBot</h2>
          <p>
            Winston procesa y normaliza tus NZBs, resuelve metadatos con FileBot / TMDb y te permite revisar o corregir antes de delegar la importación a AltMount.
          </p>
        </div>
        <div className="hero-stats">
          <div className="stat-item glass-soft">
            <span>
              <AlertTriangle size={15} color="#fbbf24" /> En revisión
            </span>
            <strong>{reviewCount}</strong>
          </div>
          <div className="stat-item glass-soft">
            <span>
              <CheckCircle2 size={15} color="#34d399" /> Aprobados
            </span>
            <strong>{approvedCount}</strong>
          </div>
          <div className="stat-item glass-soft">
            <span>
              <Upload size={15} color="#818cf8" /> Importados
            </span>
            <strong>{importedCount}</strong>
          </div>
        </div>
      </section>

      {error && <div className="banner error">{error}</div>}

      <div className="toolbar-row">
        <div className="search-field">
          <Search size={18} color="#94a3b8" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Buscar por título, ruta de archivo o motivo..."
            id="winston-search-input"
          />
        </div>
        <button
          className="btn btn-secondary"
          onClick={() => void rescanItems()}
          disabled={loading || rescanning || saving || importing || reloading}
          id="winston-rescan-btn"
          title="Prunea archivos borrados y detecta nuevos NZBs presentes en la carpeta"
        >
          <RefreshCw size={15} className={rescanning ? 'spin' : ''} />
          <span>{rescanning ? 'Reanalizando...' : 'Volver a analizar'}</span>
        </button>
        <button
          className="btn btn-secondary"
          onClick={() => void clearAndRescan()}
          disabled={loading || rescanning || saving || importing || reloading}
          id="winston-clear-rescan-btn"
          title="Elimina todos los items antiguos/fallidos y reanaliza la carpeta desde cero"
          style={{ borderColor: 'rgba(239, 68, 68, 0.4)', color: '#fca5a5' }}
        >
          <Trash2 size={15} />
          <span>Limpiar y reanalizar</span>
        </button>
      </div>

      <div className="split-layout">
        {/* Left Column: Items List */}
        <aside className="glass" style={{ padding: '1.25rem' }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '1rem' }}>
            <h3 style={{ fontSize: '1rem', fontWeight: 600 }}>NZBs ({filtered.length})</h3>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
              {items.length > 0 && (
                <button
                  className="btn btn-ghost"
                  style={{ fontSize: '0.72rem', padding: '0.15rem 0.45rem', color: '#f87171' }}
                  onClick={() => void clearAllItems()}
                  title="Vaciar la lista completa de Winston"
                >
                  Vaciar lista
                </button>
              )}
              <span style={{ fontSize: '0.75rem', color: '#94a3b8' }}>
                {loading ? 'Cargando...' : `${filtered.length} visibles`}
              </span>
            </div>
          </div>

          <div className="item-list-container">
            {loading ? (
              <div className="empty-state">
                <LoaderCircle className="spin" size={20} />
                <span>Cargando items en revisión...</span>
              </div>
            ) : filtered.length === 0 ? (
              <div className="empty-state">
                <span>No se encontraron NZBs en el estado de Winston.</span>
              </div>
            ) : (
              filtered.map((item) => {
                const isActive = selectedSource === item.source_nzb_path
                const title = item.metadata?.title || item.source_nzb_path.split('/').pop()
                const isSeries = item.metadata?.kind === 'series'

                return (
                  <button
                    key={item.source_nzb_path}
                    className={`item-row ${isActive ? 'active' : ''}`}
                    onClick={() => setSelectedSource(item.source_nzb_path)}
                  >
                    <div className="item-row-header">
                      <span className={`pill ${item.state}`}>{item.state}</span>
                      <span className={`pill confidence ${item.confidence}`}>{item.confidence}</span>
                    </div>
                    <div className="item-row-title">
                      {isSeries ? <Tv size={16} color="#818cf8" /> : <Film size={16} color="#fbbf24" />}
                      <span>{title}</span>
                    </div>
                    <div className="item-row-source">
                      <FolderTree size={13} />
                      <span>{item.source_nzb_path}</span>
                    </div>
                  </button>
                )
              })
            )}
          </div>
        </aside>

        {/* Right Column: Selected Item Detail */}
        <main>
          <AnimatePresence mode="wait">
            {selected ? (
              <motion.div
                key={selected.source_nzb_path}
                initial={{ opacity: 0, y: 12 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -12 }}
                className="glass detail-container"
              >
                <div className="detail-header">
                  <div>
                    <h2 style={{ fontSize: '1.25rem', fontWeight: 700 }}>
                      {selected.metadata?.title || selected.source_nzb_path.split('/').pop()}
                    </h2>
                    <p style={{ fontSize: '0.85rem', color: '#94a3b8', marginTop: '0.2rem' }}>
                      {selected.reason || 'Sin motivo de clasificación'}
                    </p>
                  </div>
                  <div className="detail-actions">
                    <button
                      className="btn btn-secondary"
                      onClick={() => void approveSelected()}
                      disabled={saving || importing || reloading}
                      id="btn-approve"
                      title="Marca como aprobado en Winston sin enviar de inmediato a AltMount"
                    >
                      Solo Aprobar
                    </button>
                    <button
                      className="btn btn-winston"
                      onClick={() => void approveAndImportSelected()}
                      disabled={saving || importing || reloading}
                      id="btn-import"
                      title="Aprobar y enviar inmediatamente la orden a AltMount"
                    >
                      <Upload size={16} />
                      <span>{importing ? 'Enviando a AltMount...' : 'Aprobar e Importar'}</span>
                    </button>
                    <button
                      className="btn btn-ghost"
                      onClick={() => void reloadSelected()}
                      disabled={saving || importing || reloading}
                      title="Recargar item"
                    >
                      <RefreshCw size={16} className={reloading ? 'spin' : ''} />
                    </button>
                    <button
                      className="btn btn-ghost"
                      onClick={() => void resetSelected()}
                      disabled={saving || importing || reloading}
                      title="Eliminar de la lista de revisión"
                    >
                      <Trash2 size={16} color="#f87171" />
                    </button>
                  </div>
                </div>

                {/* AltMount or Processing Notice Banner */}
                {(selected.state === 'failed' || selected.reason.toLowerCase().includes('altmount') || selected.reason.toLowerCase().includes('failed') || selected.reason.toLowerCase().includes('error')) && (
                  <div
                    className="glass-soft"
                    style={{
                      padding: '0.85rem 1rem',
                      borderRadius: '8px',
                      borderLeft: '4px solid #ef4444',
                      background: 'rgba(239, 68, 68, 0.12)',
                      display: 'flex',
                      flexDirection: 'column',
                      gap: '0.4rem',
                      marginBottom: '1rem',
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', color: '#f87171', fontWeight: 600, fontSize: '0.85rem' }}>
                      <AlertTriangle size={16} />
                      <span>Estado de Importación en AltMount</span>
                    </div>
                    <div style={{ fontSize: '0.8rem', color: '#fca5a5', lineHeight: '1.4' }}>
                      {selected.reason}
                    </div>
                    {selected.reason.includes('File does not exist') && (
                      <div style={{ fontSize: '0.75rem', color: '#cbd5e1', marginTop: '0.2rem', lineHeight: '1.4' }}>
                        💡 <strong>Diagnóstico de ruta remota:</strong> AltMount está en tu servidor (<code>altmount.gabypozo.com.es</code>) y busca el archivo dentro de <code>/config/.nzbs/</code>. Al probar en local desde tu ordenador, el archivo NZB no está subido físicamente al disco del servidor. En producción con Docker compartiendo volúmenes o con NZBs en el servidor, la ingesta es 100% directa.
                      </div>
                    )}
                  </div>
                )}

                {/* Two cards: Current Preview & Quick Correction */}
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '1rem' }}>
                  <div className="glass-soft" style={{ padding: '1.25rem' }}>
                    <h3 style={{ fontSize: '0.95rem', fontWeight: 600, marginBottom: '0.75rem' }}>
                      Vista previa de resolución
                    </h3>
                    <div className="preview-table">
                      <div className="preview-row">
                        <span>Origen</span>
                        <code title={selected.source_nzb_path}>{selected.source_nzb_path}</code>
                      </div>
                      <div className="preview-row">
                        <span>Tipo</span>
                        <code>{selected.metadata?.kind || 'desconocido'}</code>
                      </div>
                      <div className="preview-row">
                        <span>Calidad detectada</span>
                        <code>{selected.metadata?.quality || 'auto'}</code>
                      </div>
                      <div className="preview-row">
                        <span>Confianza</span>
                        <code>{selected.confidence}</code>
                      </div>
                      <div className="preview-row">
                        <span>Ruta propuesta</span>
                        <code title={selected.proposed_path}>{selected.proposed_path || '-'}</code>
                      </div>
                    </div>
                  </div>

                  <div className="glass-soft" style={{ padding: '1.25rem', display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
                    <h3 style={{ fontSize: '0.95rem', fontWeight: 600 }}>Corrección manual</h3>
                    <div className="field">
                      <span>TMDB ID</span>
                      <div style={{ display: 'flex', gap: '0.5rem' }}>
                        <input
                          value={tmdbId}
                          onChange={(e) => setTmdbId(e.target.value)}
                          placeholder="ej. 37952"
                          style={{ flex: 1 }}
                        />
                        <button
                          className="btn btn-secondary"
                          disabled={saving || importing || !tmdbId.trim()}
                          onClick={() => void applyCorrection({ tmdb_id: Number(tmdbId) })}
                        >
                          Aplicar ID
                        </button>
                      </div>
                    </div>

                    <div className="field" style={{ marginTop: '0.25rem' }}>
                      <span>Ruta relativa (Override)</span>
                      <div style={{ display: 'flex', gap: '0.5rem' }}>
                        <input
                          value={pathOverride}
                          onChange={(e) => setPathOverride(e.target.value)}
                          placeholder="Peliculas/1080/M/Matrix (1999)/..."
                          style={{ flex: 1 }}
                        />
                        <button
                          className="btn btn-secondary"
                          disabled={saving || importing || !pathOverride.trim()}
                          onClick={() => void applyCorrection({ relative_path_override: pathOverride })}
                        >
                          Aplicar Ruta
                        </button>
                      </div>
                    </div>
                  </div>
                </div>

                {/* Candidate Matches */}
                <div className="glass-soft" style={{ padding: '1.25rem' }}>
                  <h3 style={{ fontSize: '0.95rem', fontWeight: 600, marginBottom: '0.75rem' }}>
                    Coincidencias alternativas sugeridas
                  </h3>
                  {selected.candidates && selected.candidates.length > 0 ? (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
                      {selected.candidates.map((cand) => (
                        <div
                          key={`${cand.label}-${cand.tmdb_id}-${cand.year}`}
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'space-between',
                            padding: '0.65rem 0.85rem',
                            borderRadius: '8px',
                            background: 'rgba(0, 0, 0, 0.25)',
                          }}
                        >
                          <div>
                            <strong style={{ fontSize: '0.9rem' }}>{cand.label}</strong>
                            <div style={{ fontSize: '0.75rem', color: '#94a3b8' }}>
                              {cand.year ? `${cand.year} · ` : ''}TMDB {cand.tmdb_id || '-'} · {cand.reason}
                            </div>
                          </div>
                          <button
                            className="btn btn-secondary"
                            style={{ padding: '0.35rem 0.75rem', fontSize: '0.75rem' }}
                            onClick={() => cand.tmdb_id && void applyCorrection({ tmdb_id: cand.tmdb_id })}
                            disabled={saving || importing}
                          >
                            Seleccionar
                          </button>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <span style={{ fontSize: '0.85rem', color: '#64748b' }}>
                      No se encontraron candidatos alternativos para este item.
                    </span>
                  )}
                </div>
              </motion.div>
            ) : (
              <div className="glass detail-container" style={{ alignItems: 'center', justifyContent: 'center', minHeight: '350px' }}>
                <span style={{ color: '#94a3b8' }}>Selecciona un NZB de la lista para ver su detalle y resolución.</span>
              </div>
            )}
          </AnimatePresence>
        </main>
      </div>
    </div>
  )
}
