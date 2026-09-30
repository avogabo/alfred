import { useEffect, useMemo, useState } from 'react'
import {
  Calendar,
  Check,
  Clapperboard,
  Clock3,
  ImagePlus,
  Library,
  LoaderCircle,
  Plus,
  Search,
  Sparkles,
  Trash2,
  Wand2,
} from 'lucide-react'

type LibraryItem = { key: string; title: string; type: string }
type CollectionItem = {
  ratingKey: string
  title: string
  type: string
  childCount: number
  temporary: boolean
  expiresAt?: string
  thumbUrl?: string
  artUrl?: string
}
type SearchItem = {
  ratingKey: string
  title: string
  type: string
  year: number
  thumb?: string
  art?: string
  libraryKey?: string
  libraryTitle?: string
}

type PosterSuggestion = {
  url: string
  source: string
  label: string
}

type RecipeItem = {
  id: string
  name: string
  promptAliases: string[]
  inclusionRules: string[]
  exclusionRules: string[]
  orderingRules: string[]
  temporaryByDefault: boolean
}
type IdeaSuggestion = {
  recipeId?: string
  recipeName?: string
  matchedAliases?: string[]
  searchTerms: string[]
  suggestedTitles: SearchItem[]
}

type FormState = {
  name: string
  query: string
  sourcePrompt: string
  expiresAt: string
  temporary: boolean
  posterUrl: string
  posterBase64: string
}

const emptyForm: FormState = {
  name: '',
  query: '',
  sourcePrompt: '',
  expiresAt: '',
  temporary: false,
  posterUrl: '',
  posterBase64: '',
}

type GeoffreyViewProps = {
  onNotify?: (msg: string, type: 'success' | 'error') => void
  onRefreshGlobalStatus?: () => void
}

export function GeoffreyView({ onNotify, onRefreshGlobalStatus }: GeoffreyViewProps) {
  const [libraries, setLibraries] = useState<LibraryItem[]>([])
  const [selectedLibrary, setSelectedLibrary] = useState<string>('')
  const [collections, setCollections] = useState<CollectionItem[]>([])
  const [recipes, setRecipes] = useState<RecipeItem[]>([])
  const [searchResults, setSearchResults] = useState<SearchItem[]>([])
  const [selectedTitles, setSelectedTitles] = useState<SearchItem[]>([])
  const [posterSuggestions, setPosterSuggestions] = useState<PosterSuggestion[]>([])
  const [loadingPosters, setLoadingPosters] = useState(false)
  const [showPosterPicker, setShowPosterPicker] = useState(false)
  const [form, setForm] = useState<FormState>(emptyForm)
  const [loading, setLoading] = useState(true)
  const [working, setWorking] = useState(false)
  const [ideaSuggestion, setIdeaSuggestion] = useState<IdeaSuggestion | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    void bootstrap()
  }, [])

  useEffect(() => {
    if (!selectedLibrary) return
    void loadCollections(selectedLibrary)
  }, [selectedLibrary])

  const selectedLibraryMeta = libraries.find((item) => item.key === selectedLibrary)
  const posterPreview = form.posterBase64 || form.posterUrl
  const selectedCount = selectedTitles.length
  const canCreate = Boolean(selectedLibrary && form.name.trim() && selectedCount)

  const stats = useMemo(() => {
    const temporary = collections.filter((item) => item.temporary).length
    return {
      libraries: libraries.length,
      collections: collections.length,
      temporary,
    }
  }, [libraries, collections])

  async function bootstrap() {
    try {
      setLoading(true)
      const [librariesRes, recipesRes] = await Promise.all([
        fetchJSON<{ items: LibraryItem[] }>('/api/geoffrey/libraries'),
        fetchJSON<{ items: RecipeItem[] }>('/api/geoffrey/recipes'),
      ])
      const rawLibs = librariesRes.items || []
      const allLibs: LibraryItem[] = rawLibs.length > 1
        ? [{ key: 'all', title: '🌟 Todas las bibliotecas (Películas + Series)', type: 'combinadas' }, ...rawLibs]
        : rawLibs
      setLibraries(allLibs)
      setRecipes(recipesRes.items || [])
      if (allLibs.length > 0) {
        setSelectedLibrary(allLibs[0].key)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error conectando con Plex')
    } finally {
      setLoading(false)
    }
  }

  async function loadCollections(libraryKey: string) {
    try {
      const res = await fetchJSON<{ items: CollectionItem[] }>(
        `/api/geoffrey/collections?library=${encodeURIComponent(libraryKey)}`,
      )
      setCollections(res.items || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error cargando colecciones')
    }
  }

  async function runSearch() {
    if (!form.query.trim() || !selectedLibrary) return
    try {
      setWorking(true)
      setError('')
      const res = await fetchJSON<{ items: SearchItem[] }>(
        `/api/geoffrey/search?library=${encodeURIComponent(selectedLibrary)}&q=${encodeURIComponent(form.query)}`,
      )
      setSearchResults(res.items || [])
      if (!res.items?.length) {
        onNotify?.('No se encontraron títulos con esa búsqueda en Plex.', 'error')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error en la búsqueda')
    } finally {
      setWorking(false)
    }
  }

  async function runIdeaSearch() {
    if (!form.sourcePrompt.trim() || !selectedLibrary) return
    try {
      setWorking(true)
      setError('')
      const suggestion = await fetchJSON<IdeaSuggestion>(
        `/api/geoffrey/ideas?library=${encodeURIComponent(selectedLibrary)}&idea=${encodeURIComponent(form.sourcePrompt)}`,
      )
      setIdeaSuggestion(suggestion)
      setSearchResults(suggestion.suggestedTitles || [])
      setSelectedTitles(suggestion.suggestedTitles || [])
      if (!form.name.trim()) {
        setForm((current) => ({
          ...current,
          name: suggestion.recipeName || current.sourcePrompt,
        }))
      }
      onNotify?.(
        `Se encontraron ${suggestion.suggestedTitles?.length || 0} candidatos para "${form.sourcePrompt}".`,
        'success',
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error generando idea')
    } finally {
      setWorking(false)
    }
  }

  function toggleTitle(item: SearchItem) {
    setSelectedTitles((current) =>
      current.some((entry) => entry.ratingKey === item.ratingKey)
        ? current.filter((entry) => entry.ratingKey !== item.ratingKey)
        : [...current, item],
    )
  }

  function applyRecipe(recipe: RecipeItem) {
    setForm((current) => ({
      ...current,
      name: recipe.name,
      sourcePrompt: recipe.promptAliases[0] ?? recipe.name,
      temporary: recipe.temporaryByDefault,
    }))
    onNotify?.(`Receta seleccionada: ${recipe.name}`, 'success')
  }

  async function uploadPoster(file: File) {
    const body = new FormData()
    body.append('file', file)
    try {
      const res = await fetch('/api/geoffrey/poster/upload', { method: 'POST', body })
      const payload = await res.json()
      if (!res.ok) throw new Error(payload.error ?? 'No se pudo subir el póster')
      setForm((current) => ({ ...current, posterBase64: payload.dataUrl, posterUrl: '' }))
      onNotify?.(`Póster cargado: ${payload.filename}`, 'success')
    } catch (err) {
      onNotify?.(err instanceof Error ? err.message : 'Error subiendo póster', 'error')
    }
  }

  async function createCollection() {
    if (!selectedLibrary || !form.name.trim() || !selectedTitles.length) return
    try {
      setWorking(true)
      setError('')
      await fetchJSON('/api/geoffrey/collections', {
        method: 'POST',
        body: JSON.stringify({
          libraryKey: selectedLibrary,
          name: form.name,
          titles: selectedTitles.map((item) => item.title),
          items: selectedTitles.map((item) => ({
            title: item.title,
            ratingKey: item.ratingKey,
            libraryKey: item.libraryKey,
          })),
          sourcePrompt: form.sourcePrompt,
          temporary: form.temporary,
          expiresAt: form.expiresAt,
          posterUrl: form.posterUrl,
          posterBase64: form.posterBase64,
        }),
      })
      onNotify?.(`Colección creada en Plex: ${form.name}`, 'success')
      setForm(emptyForm)
      setSearchResults([])
      setSelectedTitles([])
      setPosterSuggestions([])
      setShowPosterPicker(false)
      await loadCollections(selectedLibrary)
      onRefreshGlobalStatus?.()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Error creando colección')
    } finally {
      setWorking(false)
    }
  }

  async function fetchPosterSuggestions() {
    const query = form.name.trim() || form.sourcePrompt.trim()
    if (!query) {
      onNotify?.('Escribe primero el nombre de la colección o un tema para sugerir pósters.', 'error')
      return
    }
    try {
      setLoadingPosters(true)
      setShowPosterPicker(true)
      const res = await fetchJSON<{ items: PosterSuggestion[] }>(
        `/api/geoffrey/poster/suggestions?title=${encodeURIComponent(form.name)}&prompt=${encodeURIComponent(form.sourcePrompt)}`,
      )
      setPosterSuggestions(res.items || [])
      if (!res.items || res.items.length === 0) {
        onNotify?.('No se encontraron pósters automáticos para esta búsqueda.', 'error')
      }
    } catch (err) {
      onNotify?.(err instanceof Error ? err.message : 'Error obteniendo pósters sugeridos', 'error')
    } finally {
      setLoadingPosters(false)
    }
  }

  async function deleteCollection(item: CollectionItem) {
    if (!selectedLibrary) return
    if (!window.confirm(`¿Seguro que deseas borrar la colección "${item.title}" de Plex?`)) return
    try {
      setWorking(true)
      await fetchJSON(
        `/api/geoffrey/collections/${encodeURIComponent(selectedLibrary)}/${encodeURIComponent(item.title)}`,
        { method: 'DELETE' },
      )
      onNotify?.(`Colección borrada: ${item.title}`, 'success')
      await loadCollections(selectedLibrary)
      onRefreshGlobalStatus?.()
    } catch (err) {
      onNotify?.(err instanceof Error ? err.message : 'Error borrando colección', 'error')
    } finally {
      setWorking(false)
    }
  }

  return (
    <div className="tab-view-content" style={{ display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
      <section className="hero-banner glass">
        <div className="hero-info">
          <span className="hero-tag geoffrey">
            <Sparkles size={14} /> Subsección Geoffrey
          </span>
          <h2>Curador de Colecciones de Plex</h2>
          <p>
            Crea colecciones inteligentes, permanentes o temporales (con fecha de caducidad automática), enriquecidas con sugerencias temáticas de TMDb y pósters personalizados.
          </p>
        </div>
        <div className="hero-stats">
          <div className="stat-item glass-soft">
            <span>
              <Library size={15} color="#fbbf24" /> Bibliotecas
            </span>
            <strong>{stats.libraries}</strong>
          </div>
          <div className="stat-item glass-soft">
            <span>
              <Clapperboard size={15} color="#818cf8" /> Colecciones
            </span>
            <strong>{stats.collections}</strong>
          </div>
          <div className="stat-item glass-soft">
            <span>
              <Clock3 size={15} color="#34d399" /> Temporales
            </span>
            <strong>{stats.temporary}</strong>
          </div>
        </div>
      </section>

      {error && <div className="banner error">{error}</div>}

      <div className="split-layout">
        {/* Left Column: Libraries & Recipes */}
        <aside className="glass sidebar-panel">
          <div>
            <h3 style={{ fontSize: '0.95rem', fontWeight: 600, marginBottom: '0.75rem' }}>Bibliotecas Plex</h3>
            <div className="library-selector-list">
              {libraries.map((lib) => (
                <button
                  key={lib.key}
                  className={`library-selector-btn ${selectedLibrary === lib.key ? 'active' : ''}`}
                  onClick={() => setSelectedLibrary(lib.key)}
                >
                  <strong style={{ fontSize: '0.85rem' }}>{lib.title}</strong>
                  <span style={{ fontSize: '0.75rem', color: '#94a3b8' }}>{lib.type}</span>
                </button>
              ))}
              {libraries.length === 0 && (
                <span style={{ fontSize: '0.8rem', color: '#64748b' }}>
                  No se detectaron bibliotecas. Verifica tu conexión a Plex en Ajustes.
                </span>
              )}
            </div>
          </div>

          <div>
            <h3 style={{ fontSize: '0.95rem', fontWeight: 600, marginBottom: '0.75rem' }}>Recetas temáticas</h3>
            <div className="recipe-cards-grid">
              {recipes.map((r) => (
                <button key={r.id} className="recipe-btn" onClick={() => applyRecipe(r)}>
                  <strong>{r.name}</strong>
                  <small>{r.promptAliases.slice(0, 2).join(' · ')}</small>
                </button>
              ))}
            </div>
          </div>

          <div className="glass-soft" style={{ padding: '0.85rem' }}>
            <h4 style={{ fontSize: '0.8rem', color: '#94a3b8', textTransform: 'uppercase', marginBottom: '0.5rem' }}>
              Checklist de creación
            </h4>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem', fontSize: '0.8rem' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', color: selectedLibrary ? '#34d399' : '#64748b' }}>
                <Check size={14} />
                <span>Biblioteca seleccionada</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', color: form.name.trim() ? '#34d399' : '#64748b' }}>
                <Check size={14} />
                <span>Nombre definido</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', color: selectedCount > 0 ? '#34d399' : '#64748b' }}>
                <Check size={14} />
                <span>{selectedCount} títulos añadidos</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', color: !form.temporary || form.expiresAt ? '#34d399' : '#64748b' }}>
                <Check size={14} />
                <span>{form.temporary ? 'Temporal con fecha' : 'Colección permanente'}</span>
              </div>
            </div>
          </div>
        </aside>

        {/* Right Column: Composer and Collections */}
        <main style={{ display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
          {/* Builder Panel */}
          <section className="glass" style={{ padding: '1.5rem', display: 'flex', flexDirection: 'column', gap: '1.25rem' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <h3 style={{ fontSize: '1.1rem', fontWeight: 600 }}>Componer Colección</h3>
              <span style={{ fontSize: '0.8rem', color: '#94a3b8' }}>
                {selectedLibraryMeta ? `${selectedLibraryMeta.title} (${selectedLibraryMeta.type})` : 'Elige biblioteca'}
              </span>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 260px', gap: '1.25rem' }}>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '0.85rem' }}>
                <div className="field">
                  <span>Nombre de la colección</span>
                  <input
                    value={form.name}
                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                    placeholder="ej. Halloween de risa, Sagas Marvel..."
                    id="collection-name-input"
                  />
                </div>

                <div className="field">
                  <span>Idea / Prompt temático (TMDb)</span>
                  <div style={{ display: 'flex', gap: '0.5rem' }}>
                    <input
                      value={form.sourcePrompt}
                      onChange={(e) => setForm({ ...form, sourcePrompt: e.target.value })}
                      placeholder="ej. Películas de nieve familiares como Frozen"
                      style={{ flex: 1 }}
                      id="collection-idea-input"
                    />
                    <button
                      className="btn btn-secondary"
                      onClick={() => void runIdeaSearch()}
                      disabled={working || !form.sourcePrompt.trim()}
                    >
                      <Wand2 size={16} />
                      <span>Proponer</span>
                    </button>
                  </div>
                </div>

                <div className="field">
                  <span>Búsqueda manual en Plex</span>
                  <div style={{ display: 'flex', gap: '0.5rem' }}>
                    <input
                      value={form.query}
                      onChange={(e) => setForm({ ...form, query: e.target.value })}
                      placeholder="ej. Gremlins, Shrek..."
                      style={{ flex: 1 }}
                      id="collection-search-input"
                    />
                    <button
                      className="btn btn-primary"
                      onClick={() => void runSearch()}
                      disabled={working || !form.query.trim()}
                    >
                      <Search size={16} />
                      <span>Buscar</span>
                    </button>
                  </div>
                </div>

                <div style={{ display: 'flex', alignItems: 'center', gap: '1.5rem', marginTop: '0.25rem' }}>
                  <label style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', cursor: 'pointer', fontSize: '0.875rem' }}>
                    <input
                      type="checkbox"
                      checked={form.temporary}
                      onChange={(e) => setForm({ ...form, temporary: e.target.checked })}
                    />
                    <span>Colección Temporal</span>
                  </label>

                  {form.temporary && (
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                      <Calendar size={16} color="#fbbf24" />
                      <input
                        type="date"
                        value={form.expiresAt}
                        onChange={(e) => setForm({ ...form, expiresAt: e.target.value })}
                        style={{ padding: '0.4rem 0.6rem', fontSize: '0.8rem' }}
                      />
                    </div>
                  )}
                </div>
              </div>

              {/* Poster Preview and Quick Upload */}
              <div className="glass-soft" style={{ padding: '1rem', display: 'flex', flexDirection: 'column', gap: '0.75rem', alignItems: 'center' }}>
                <span style={{ fontSize: '0.8rem', color: '#94a3b8' }}>Póster de la colección</span>
                <div className="poster-box">
                  {posterPreview ? (
                    <img src={posterPreview} alt="Poster" />
                  ) : (
                    <div className="poster-box empty">
                      <Clapperboard size={32} />
                    </div>
                  )}
                </div>
                <div style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: '0.4rem' }}>
                  <input
                    value={form.posterUrl}
                    onChange={(e) => setForm({ ...form, posterUrl: e.target.value, posterBase64: '' })}
                    placeholder="URL del póster..."
                    style={{ fontSize: '0.75rem', padding: '0.4rem' }}
                  />
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.4rem' }}>
                    <label className="btn btn-secondary" style={{ fontSize: '0.75rem', padding: '0.4rem', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '0.3rem' }}>
                      <ImagePlus size={14} />
                      <span>Subir</span>
                      <input
                        type="file"
                        accept="image/*"
                        style={{ display: 'none' }}
                        onChange={(e) => {
                          const file = e.target.files?.[0]
                          if (file) void uploadPoster(file)
                        }}
                      />
                    </label>
                    <button
                      type="button"
                      className="btn btn-secondary"
                      style={{ fontSize: '0.75rem', padding: '0.4rem', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '0.3rem' }}
                      onClick={() => void fetchPosterSuggestions()}
                      disabled={loadingPosters || (!form.name.trim() && !form.sourcePrompt.trim())}
                      title="Generar opciones de póster con IA y TMDb"
                    >
                      {loadingPosters ? <LoaderCircle size={14} className="spin" /> : <Sparkles size={14} color="#38bdf8" />}
                      <span>IA / TMDb</span>
                    </button>
                  </div>
                </div>
              </div>
            </div>

            {/* Poster Suggestion Picker */}
            {showPosterPicker && (
              <div
                className="glass-soft"
                style={{
                  padding: '1rem',
                  borderRadius: '10px',
                  border: '1px solid rgba(56, 189, 248, 0.3)',
                  background: 'rgba(15, 23, 42, 0.7)',
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '0.75rem' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                    <Sparkles size={16} color="#38bdf8" />
                    <strong style={{ fontSize: '0.9rem', color: '#f8fafc' }}>
                      Pósters sugeridos para "{form.name || form.sourcePrompt}"
                    </strong>
                  </div>
                  <button
                    type="button"
                    className="btn btn-ghost"
                    style={{ fontSize: '0.75rem', padding: '0.2rem 0.6rem' }}
                    onClick={() => setShowPosterPicker(false)}
                  >
                    Cerrar
                  </button>
                </div>

                {loadingPosters && (
                  <div style={{ display: 'flex', alignItems: 'center', gap: '0.6rem', color: '#94a3b8', fontSize: '0.85rem', padding: '1.25rem 0', justifyContent: 'center' }}>
                    <LoaderCircle size={18} className="spin" color="#38bdf8" />
                    <span>Buscando sagas en TMDb y generando arte temático con IA...</span>
                  </div>
                )}

                {!loadingPosters && posterSuggestions.length > 0 && (
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))', gap: '0.75rem' }}>
                    {posterSuggestions.map((sug, idx) => {
                      const isSelected = form.posterUrl === sug.url
                      return (
                        <div
                          key={idx}
                          onClick={() => {
                            setForm((prev) => ({ ...prev, posterUrl: sug.url, posterBase64: '' }))
                            onNotify?.(`Póster seleccionado: ${sug.label}`, 'success')
                          }}
                          style={{
                            cursor: 'pointer',
                            borderRadius: '8px',
                            overflow: 'hidden',
                            border: isSelected ? '2px solid #38bdf8' : '1px solid rgba(255, 255, 255, 0.12)',
                            background: isSelected ? 'rgba(56, 189, 248, 0.15)' : 'rgba(0, 0, 0, 0.5)',
                            display: 'flex',
                            flexDirection: 'column',
                            boxShadow: isSelected ? '0 0 12px rgba(56, 189, 248, 0.4)' : undefined,
                            transition: 'all 0.15s ease',
                          }}
                        >
                          <div style={{ height: '190px', width: '100%', overflow: 'hidden', background: '#090d16' }}>
                            <img
                              src={sug.url}
                              alt={sug.label}
                              style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                              loading="lazy"
                            />
                          </div>
                          <div style={{ padding: '0.5rem', display: 'flex', flexDirection: 'column', gap: '0.2rem' }}>
                            <span
                              style={{
                                fontSize: '0.65rem',
                                padding: '0.1rem 0.35rem',
                                borderRadius: '4px',
                                background: sug.source.startsWith('tmdb') ? 'rgba(16, 185, 129, 0.25)' : 'rgba(168, 85, 247, 0.25)',
                                color: sug.source.startsWith('tmdb') ? '#34d399' : '#d8b4fe',
                                width: 'fit-content',
                                fontWeight: 600,
                              }}
                            >
                              {sug.source.startsWith('tmdb') ? 'TMDb Oficial' : 'IA Pollinations'}
                            </span>
                            <span style={{ fontSize: '0.72rem', color: '#cbd5e1', lineHeight: '1.2' }}>
                              {sug.label}
                            </span>
                          </div>
                        </div>
                      )
                    })}
                  </div>
                )}

                {!loadingPosters && posterSuggestions.length === 0 && (
                  <div style={{ textAlign: 'center', color: '#94a3b8', fontSize: '0.8rem', padding: '1rem' }}>
                    No se encontraron sugerencias. Prueba con otro nombre o palabra clave.
                  </div>
                )}
              </div>
            )}

            {ideaSuggestion && (
              <div className="banner" style={{ background: 'rgba(245, 158, 11, 0.12)', border: '1px solid rgba(245, 158, 11, 0.3)' }}>
                <Sparkles size={16} color="#fbbf24" />
                <span style={{ fontSize: '0.85rem' }}>
                  Términos sugeridos por IA: {ideaSuggestion.searchTerms.join(' · ')}
                </span>
              </div>
            )}

            {/* Create Collection CTA */}
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderTop: '1px solid var(--border-subtle)', paddingTop: '1rem' }}>
              <span style={{ fontSize: '0.85rem', color: '#94a3b8' }}>
                {selectedCount} títulos seleccionados
              </span>
              <button
                className="btn btn-primary"
                onClick={() => void createCollection()}
                disabled={working || !canCreate}
                id="btn-create-collection"
              >
                <Plus size={16} />
                <span>Crear colección en Plex</span>
              </button>
            </div>

            {/* Search Results Grid */}
            {searchResults.length > 0 && (
              <div style={{ marginTop: '0.5rem' }}>
                <h4 style={{ fontSize: '0.9rem', color: '#cbd5e1', marginBottom: '0.5rem' }}>
                  Resultados encontrados ({searchResults.length})
                </h4>
                <div className="results-grid">
                  {searchResults.map((item) => {
                    const isPicked = selectedTitles.some((t) => t.ratingKey === item.ratingKey)
                    return (
                      <button
                        key={item.ratingKey}
                        className={`result-card ${isPicked ? 'selected' : ''}`}
                        onClick={() => toggleTitle(item)}
                      >
                        <div className="poster-box" style={{ height: '140px' }}>
                          {item.thumb ? (
                            <img
                              src={`/api/geoffrey/plex/image?path=${encodeURIComponent(item.thumb)}`}
                              alt={item.title}
                              loading="lazy"
                            />
                          ) : (
                            <Clapperboard size={24} />
                          )}
                        </div>
                        <strong style={{ fontSize: '0.8rem', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {item.title}
                        </strong>
                        <span style={{ fontSize: '0.7rem', color: '#94a3b8' }}>
                          {item.year || 's/f'} · {isPicked ? '✓ Seleccionado' : '+ Añadir'}
                        </span>
                        {(item.libraryTitle || item.type) && (
                          <span
                            style={{
                              fontSize: '0.65rem',
                              padding: '0.1rem 0.35rem',
                              borderRadius: '4px',
                              background: item.type === 'show' ? 'rgba(168, 85, 247, 0.25)' : 'rgba(59, 130, 246, 0.25)',
                              color: item.type === 'show' ? '#d8b4fe' : '#93c5fd',
                              alignSelf: 'flex-start',
                              marginTop: '0.2rem',
                            }}
                          >
                            {item.libraryTitle || (item.type === 'show' ? 'Serie' : 'Película')}
                          </span>
                        )}
                      </button>
                    )
                  })}
                </div>
              </div>
            )}
          </section>

          {/* Existing Collections Section */}
          <section className="glass" style={{ padding: '1.5rem' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '1rem' }}>
              <h3 style={{ fontSize: '1.1rem', fontWeight: 600 }}>Colecciones en esta biblioteca</h3>
              <span style={{ fontSize: '0.8rem', color: '#94a3b8' }}>{collections.length} colecciones</span>
            </div>

            <div className="collections-grid">
              {collections.map((col) => (
                <div key={col.ratingKey} className="collection-card">
                  {col.thumbUrl || col.artUrl ? (
                    <img
                      src={col.thumbUrl || col.artUrl}
                      alt={col.title}
                      className="collection-thumb"
                      loading="lazy"
                    />
                  ) : (
                    <div className="collection-thumb" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                      <Clapperboard size={20} color="#94a3b8" />
                    </div>
                  )}
                  <div className="collection-meta">
                    <strong>{col.title}</strong>
                    <span>{col.childCount} elementos</span>
                    {col.temporary && (
                      <span className="pill needs_review" style={{ width: 'fit-content', fontSize: '0.65rem' }}>
                        Caduca {col.expiresAt ? col.expiresAt.slice(0, 10) : ''}
                      </span>
                    )}
                  </div>
                  <button
                    className="btn btn-ghost btn-danger"
                    style={{ padding: '0.4rem', borderRadius: '8px' }}
                    onClick={() => void deleteCollection(col)}
                    title="Borrar colección de Plex"
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
              ))}
              {collections.length === 0 && (
                <div className="empty-state" style={{ gridColumn: '1 / -1' }}>
                  <span>No se encontraron colecciones en esta biblioteca de Plex.</span>
                </div>
              )}
            </div>
          </section>
        </main>
      </div>
    </div>
  )
}

async function fetchJSON<T>(input: RequestInfo | URL, init?: RequestInit): Promise<T> {
  const response = await fetch(input, {
    headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
    ...init,
  })
  const payload = await response.json().catch(() => ({}))
  if (!response.ok) throw new Error(payload.error ?? 'Petición fallida')
  return payload as T
}
