# 🎩 ALFRED · The Media Butler

**Alfred** es la plataforma unificada de mayordomos para la gestión, ingesta y curación de tu biblioteca multimedia en **Plex**, **Usenet** y **Unraid / Docker**.

Unifica en una única aplicación (un solo binario en Go, una interfaz web moderna en React y un solo contenedor Docker) los dos proyectos previamente independientes:

- 📦 **Winston**: Subsección de ingesta, normalización, resolución FileBot y delegación a AltMount.
- 🎬 **Geoffrey**: Subsección de curación visual de colecciones en Plex, recetas temáticas, integración TMDb, colecciones temporales auto-expirables y gestión de pósters.
- ⚙️ **Ajustes Compartidos**: Configuración centralizada y persistente que comparte la conexión con Plex y simplifica la administración.

---

## 🌟 Características Principales

### 1. 📦 Subsección Winston (Ingesta & Revisión)
- **Staging y Normalización:** Escanea carpetas de NZBs y detecta series, temporadas, episodios y películas.
- **Integración con FileBot 5.1.6:** Resolución nativa de rutas organizadas según estándares de Plex mediante TheMovieDB y TheTVDB.
- **Revisión en Vivo:** Interfaz interactiva para inspeccionar candidatos, corregir IDs de TMDb al vuelo o forzar rutas relativas.
- **Importación a AltMount:** Aprobación e importación secuencial sin colisiones ni duplicados.

### 2. 🎬 Subsección Geoffrey (Colecciones & Curación)
- **Gestión Visual de Plex:** Explora todas las bibliotecas de tu servidor Plex (Películas, Series) y sus colecciones activas.
- **Recetas Temáticas & TMDb:** Ideas inteligentes ("Halloween de risa", "Navidad familiar", "Sagas Marvel") con propuestas automáticas de títulos.
- **Colecciones Temporales:** Crea colecciones que caducan en una fecha determinada (ej. tras las navidades o la noche de Halloween) y deja que Alfred las elimine automáticamente sin tocar tus archivos.
- **Gestor de Pósters:** Asigna carátulas personalizadas a las colecciones por URL directa o subiendo un archivo desde el navegador.

### 3. ⚙️ Ajustes Unificados
- **Plex Compartido:** Se configura una única vez (URL, Token, Mappings de rutas) y sirve tanto para Winston como para Geoffrey.
- **Persistencia Transparente:** Los ajustes se guardan en `/config/.alfred-settings.json` y se pueden editar desde la propia interfaz web o mediante variables de entorno.
- **Persistencia de Licencia FileBot:** Detecta y activa automáticamente `license.psm` sin consumir nuevas activaciones en cada recreate del contenedor.

---

## 🚀 Despliegue Rápido con Docker

### 1. Estructura de carpetas recomendada

```bash
mkdir -p /opt/alfred/config/filebot
mkdir -p /opt/alfred/nzb
```

> **Nota:** Si utilizas FileBot, copia tu archivo de licencia en:
> `/opt/alfred/config/filebot/license.psm`

### 2. Docker Run

```bash
docker run -d \
  --name alfred \
  --restart unless-stopped \
  -p 8091:8091 \
  -e PLEX_BASE_URL=http://TU_PLEX_IP:32400 \
  -e PLEX_TOKEN=TU_PLEX_TOKEN \
  -e WINSTON_SOURCE_ROOT=/data/nzb \
  -e WINSTON_ALTMOUNT_BASE_URL=http://TU_ALTMOUNT_IP:8989 \
  -e WINSTON_ALTMOUNT_API_KEY=TU_API_KEY \
  -e TMDB_API_KEY=TU_API_KEY_TMDB \
  -v /opt/alfred/config:/config \
  -v /opt/alfred/nzb:/data/nzb \
  ghcr.io/avogabo/alfred:latest
```

### 3. Docker Compose

```yaml
services:
  alfred:
    image: ghcr.io/avogabo/alfred:latest
    container_name: alfred
    restart: unless-stopped
    ports:
      - "8091:8091"
    environment:
      ALFRED_HTTP_LISTEN_ADDR: ":8091"
      ALFRED_DATA_DIR: /config
      PLEX_BASE_URL: http://192.168.1.100:32400
      PLEX_TOKEN: tu_token_plex
      PLEX_DEFAULT_LIBRARY: Películas
      WINSTON_SOURCE_ROOT: /data/nzb
      WINSTON_ALTMOUNT_BASE_URL: http://192.168.1.100:8989
      WINSTON_ALTMOUNT_API_KEY: tu_api_key_altmount
      TMDB_API_KEY: tu_tmdb_key
      TZ: Europe/Madrid
    volumes:
      - /opt/alfred/config:/config
      - /opt/alfred/nzb:/data/nzb
```

Abre tu navegador en `http://TU_HOST:8091`.

---

## 🛠️ Desarrollo Local

Requisitos:
- **Go 1.24+**
- **Node.js 22+**

```bash
# 1. Compilar frontend React
cd web
npm install
npm run build
cd ..

# 2. Configurar variables de entorno de prueba
cp .env.example .env

# 3. Arrancar backend en Go
go run .
```

O para desarrollar en caliente con Vite:

```bash
# Terminal 1: Backend
go run .

# Terminal 2: Frontend Vite dev server (con proxy a :8091)
cd web
npm run dev
```

---

## 📡 API REST Unificada

### Sistema & Ajustes
- `GET /api/health` — Comprobación de salud de los servicios.
- `GET /api/status` — Estado de conexión a Plex, FileBot, pendientes de Winston y colecciones de Geoffrey.
- `GET /api/settings` — Obtener ajustes actuales.
- `POST /api/settings` — Guardar y aplicar ajustes en tiempo real.

### Winston
- `GET /api/winston/review/items` — Listar NZBs y estado de resolución.
- `GET /api/winston/review/item?source=...` — Detalle de un item específico.
- `POST /api/winston/review/correct` — Aplicar corrección de TMDb ID o ruta.
- `POST /api/winston/review/approve` — Aprobar item para importación.
- `POST /api/winston/review/import` — Delegar importación a AltMount.
- `POST /api/winston/review/rescan` — Forzar escaneo de la carpeta de NZBs.
- `GET /api/winston/filebot/status` — Comprobar estado de FileBot y licencia.

### Geoffrey
- `GET /api/geoffrey/libraries` — Bibliotecas disponibles en Plex.
- `GET /api/geoffrey/collections?library=...` — Colecciones de una biblioteca.
- `POST /api/geoffrey/collections` — Crear colección (con títulos, póster y fecha de expiración).
- `DELETE /api/geoffrey/collections/{libraryKey}/{name}` — Eliminar colección.
- `GET /api/geoffrey/search?library=...&q=...` — Buscar títulos dentro de una biblioteca Plex.
- `GET /api/geoffrey/ideas?library=...&idea=...` — Sugerir títulos temáticos con TMDb.
- `GET /api/geoffrey/recipes` — Listar recetas de colecciones predefinidas.
- `POST /api/geoffrey/poster/upload` — Subir póster personalizado para una colección.
- `GET /api/geoffrey/plex/image?path=...` — Proxy de miniaturas de Plex.

*(Todas las rutas mantienen retrocompatibilidad con las URLs originales de Winston `/api/review/...` y Geoffrey `/api/collections...`).*
