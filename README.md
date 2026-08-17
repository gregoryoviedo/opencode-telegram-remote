# OpenCode Remote

Control remoto para una instancia local de `opencode serve` desde Telegram.
Binario único en Go que lee la configuración desde un archivo `.env`, se
comunica con Telegram por long polling y expone un navegador recursivo de
proyectos limitado al workspace que tú elijas.

La idea es permitirte ejecutar, monitorear y guiar sesiones de OpenCode desde
el teléfono mientras tu MacBook hace el trabajo real, con cero superficie de
ataque pública más allá de la API de bots de Telegram.

## Características

- **Long polling a Telegram** — sin puertos expuestos, sin túneles, sin
  webhooks.
- **Whitelist estricta por usuario** — solo el `ALLOWED_CHAT_ID` configurado
  puede usar el bot; cualquier otro intento se descarta silenciosamente.
- **Navegación limitada al workspace** — selector recursivo de carpetas que
  rechaza salir del workspace raíz, incluso con symlinks o `..`.
- **Estado persistente en SQLite** — workspace, proyecto activo, sesión
  activa y estados efímeros de navegación sobreviven a reinicios.
- **Binario único** — sin daemon, sin GUI, sin firma. Tú lo compilas, lo
  ejecutas y lo paras con `Ctrl+C`.

## Requisitos

- macOS, Linux o Windows.
- Go 1.22 o superior.
- `opencode` instalado localmente (por ejemplo, `brew install anomalyco/tap/opencode`).
- Un token de bot de Telegram desde `@BotFather`.
- Tu ID personal de chat desde `@userinfobot`.

## Inicio rápido

```bash
# 1. Compila el binario
go build -o remote-bot ./cmd/remote-bot

# 2. Crea tu .env
cp .env.example .env
$EDITOR .env

# 3. Ejecuta
./remote-bot
```

El bot lee `.env` desde el directorio actual o cualquier directorio padre.
También puedes apuntar a un archivo específico con `ENV_FILE=/ruta/.env` o
exportar las variables directamente en tu shell.

### Variables obligatorias

| Variable             | Descripción                                            |
|----------------------|--------------------------------------------------------|
| `WORKSPACE_ROOT`     | Ruta absoluta que delimita toda la navegación.         |
| `TELEGRAM_BOT_TOKEN` | Token del bot desde `@BotFather`.                      |
| `ALLOWED_CHAT_ID`    | Tu ID numérico de usuario desde `@userinfobot`.        |

### Variables opcionales

| Variable              | Por defecto                                            | Descripción                                                                                              |
|-----------------------|--------------------------------------------------------|----------------------------------------------------------------------------------------------------------|
| `OPENCODE_PORT`       | `4096`                                                 | Puerto para `opencode serve` local.                                                                      |
| `OPENCODE_BIN`        | `opencode`                                             | Binario a lanzar cuando el bot auto-arranca el servidor. Usa ruta absoluta si no está en `PATH`.         |
| `OPENCODE_AUTOSTART`  | `false`                                                | Si está activo, el bot lanza `opencode serve --port <p>` en `WORKSPACE_ROOT` al iniciarse.               |
| `REMOTE_STATE_PATH`   | `<WORKSPACE_ROOT>/.opencode-remote/state.db`           | Ubicación de la base SQLite.                                                                             |
| `TELEGRAM_API_ROOT`   | (predeterminado de `telebot.v3`)                        | Override del endpoint de la API de Telegram (útil para mirrors o tests).                                |
| `TELEGRAM_PROXY_URL`  | _(vacío)_                                               | URL de proxy HTTP para las llamadas a la API de Telegram (formato `http://host:port`).                  |
| `ENV_FILE`            | `.env` subiendo desde el directorio actual              | Forzar un archivo `.env` específico.                                                                     |

### Auto-arranque del servidor

Por defecto el servidor OpenCode **no** se levanta al iniciar el bot.
Arrancarlo es responsabilidad tuya vía `/init` (atajo manual) o
`/projects → Usar esta carpeta` (cuando seleccionas un proyecto). Esto se
debe a que `opencode serve` queda atado a la carpeta desde la que lo
arrancas, así que el bot prefiere esperar a que le digas qué proyecto
quieres antes de gastar un puerto.

Si prefieres el comportamiento antiguo (`/init` implícito en el workspace
_root_), añade a tu `.env`:

```bash
OPENCODE_AUTOSTART=true
```

Sea como sea, el bot siempre apaga el servidor con `SIGTERM` cuando recibe
`Ctrl+C` o una señal de terminación.

## Comandos

| Comando              | Descripción                                             |
|----------------------|---------------------------------------------------------|
| `/start` / `/help`   | Bienvenida y lista de comandos.                         |
| `/status`            | Salud del servidor OpenCode, proyecto activo, sesión.   |
| `/projects`          | Selector recursivo de carpetas; al confirmar, **(re)arranca** el servidor en la carpeta elegida. |
| `/init`              | (Re)arranca el servidor OpenCode en la carpeta activa. Acepta una ruta **relativa al workspace** (`/init work/proyecto`). |
| `/sessions`          | Lista o crea sesiones del proyecto activo. Acepta `new` (crea y activa) o un `id` de sesión para activarla. |
| `/diff` / `/changes` | Archivos modificados por la sesión activa.              |
| `/undo`              | Revierte el último cambio.                              |
| texto libre          | Prompt directo a la sesión activa de OpenCode.          |

Los Inline Keyboards manejan el resto: carpetas, "Atrás", "Inicio", "Usar
esta carpeta", selección de sesión y "Nueva sesión".

## Modelo de servidor

`opencode serve` está pensado como un servidor **por proyecto**: queda atado
al directorio desde el que lo arrancas y todas las sesiones que crea heredan
el `projectID` calculado a partir de esa CWD. El bot opera con un único
servidor a la vez:

- **`/projects` → "Usar esta carpeta"**: el bot mata el servidor actual (si
  lo hay) y arranca uno nuevo con la carpeta recién seleccionada como CWD.
- **`/init [ruta]`**: atajo para reiniciar el servidor sin cambiar de
  carpeta. Si pasas una ruta, se usa como nueva CWD; si no, se reutiliza la
  carpeta activa.
- **`OPENCODE_AUTOSTART` (def. `false`)**: si lo activas, el bot arranca el
  servidor en `WORKSPACE_ROOT` al iniciarse; necesitarás seguir usando
  `/projects` o `/init` para cambiar la carpeta activa.

> Si el servidor está apagado, cualquier comando (`/status`, `/sessions`,
> `/diff`, `/undo`, texto libre) devuelve un mensaje pidiéndote ejecutar
> `/init`.

## Automatización del inicio

Para no abrir una terminal cada vez que quieras lanzar el bot, tienes varias
opciones en macOS.

### Opción 1: Raycast Script Command

1. Compila el binario y déjalo en una ruta estable, por ejemplo
   `~/bin/remote-bot`.
2. Crea un archivo
   `~/.config/raycast/extensions/remote-bot.sh`:

   ```bash
   #!/usr/bin/env bash
   # @raycast.title OpenCode Remote
   # @raycast.mode fullOutput
   # @raycast.icon 🚀
   # @raycast.description Inicia el bot de Telegram para OpenCode

   cd "$HOME/dev/opencode-telegram-remote" || exit 1
   exec ./remote-bot
   ```

3. `chmod +x ~/.config/raycast/extensions/remote-bot.sh`.
4. En Raycast, escribe "OpenCode Remote" y ejecútalo. La ventana de Raycast
   quedará mostrando los logs del bot; ciérrala con `Ctrl+C` para detenerlo.

Para detenerlo, crea un segundo comando:

```bash
#!/usr/bin/env bash
# @raycast.title Stop OpenCode Remote
# @raycast.mode silent
# @raycast.icon 🛑
# @raycast.description Detiene el bot de OpenCode Remote

pkill -f "remote-bot"
```

### Opción 2: Atajos de macOS (Shortcuts)

1. Abre la app **Atajos** (`/System/Applications/Shortcuts.app`).
2. Crea un nuevo atajo llamado "Iniciar OpenCode Remote".
3. Añade la acción **Ejecutar script de shell** con:

   ```bash
   cd "$HOME/dev/opencode-telegram-remote" && exec ./remote-bot
   ```

4. Marca "Ejecutar como acción rápida" si quieres lanzarlo desde Finder o
   desde la barra de menús.
5. Opcional: añade un atajo de teclado (por ejemplo `⌥⌘R`) en los ajustes
   del atajo.
6. Crea un segundo atajo "Detener OpenCode Remote" con `pkill -f remote-bot`.

### Opción 3: launchd para inicio silencioso

Si prefieres que el bot corra en segundo plano sin ventana, crea
`~/Library/LaunchAgents/ai.opencode.remote.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>ai.opencode.remote</string>
  <key>ProgramArguments</key>
  <array>
    <string>/Users/tu-usuario/bin/remote-bot</string>
  </array>
  <key>WorkingDirectory</key>
  <string>/Users/tu-usuario/dev/opencode-telegram-remote</string>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><false/>
  <key>StandardOutPath</key><string>/tmp/opencode-remote.log</string>
  <key>StandardErrorPath</key><string>/tmp/opencode-remote.err</string>
</dict>
</plist>
```

```bash
launchctl load ~/Library/LaunchAgents/ai.opencode.remote.plist
launchctl unload ~/Library/LaunchAgents/ai.opencode.remote.plist
```

## Seguridad

- El bot solo responde al `ALLOWED_CHAT_ID` configurado. El resto se descarta
  silenciosamente sin llegar al handler.
- Los callbacks de los Inline Keyboards se validan contra un registro de
  navegación de corta vida por chat. Otros chats reciben "menú expirado".
- Todas las rutas que vienen de Telegram son relativas; se resuelven, se
  evaluan los symlinks y la ruta final debe permanecer dentro de
  `WORKSPACE_ROOT`.
- El token se lee del `.env` al arrancar. SQLite solo guarda proyecto
  activo, sesión activa y estado efímero — nunca contenido de prompts.

Consulta `SECURITY.md` para el modelo de confianza completo.

## Arquitectura

Hexagonal / Clean Architecture en Go, con tres capas concéntricas:

```text
internal/
  domain/      entidades (Project, Session, RuntimeState, NavigationState)
               y puertos (WorkspaceFS, StateRepository, OpenCodeClient)
  usecase/     navegador del workspace, navegación, bot handler
  adapter/
    opencode/  cliente REST + SSE
    telegram/  long polling con telebot.v3, whitelist, callbacks
    storage/   repositorio SQLite
    workspace/ adaptador de filesystem
    config/    cargador de .env
cmd/remote-bot/  composition root
```

La capa de dominio no tiene dependencias externas; todo lo demás va detrás
de interfaces declaradas por el dominio. Esto permite que los tests
sustituyan SQLite por un store en memoria y OpenCode por un servidor
`httptest`.

Consulta `docs/DESIGN.md` para el documento de diseño completo.

## Producto

Consulta `docs/PRODUCT.md` para el alcance actual del producto, lo que
está dentro y fuera de alcance, y la hoja de ruta planificada.

## Desarrollo

```bash
go test ./...
go vet ./...
go build -o remote-bot ./cmd/remote-bot
```

El repositorio conserva el binario `remote-bot` compilado en la raíz por
comodidad (es el único artefacto que produce el proyecto). Bórralo antes de
publicar una rama si no quieres enviar el binario compilado junto con el
código fuente.

## Estructura

```text
.
├── cmd/remote-bot/        punto de entrada
├── internal/
│   ├── adapter/           OpenCode, Telegram, SQLite, filesystem
│   ├── config/            cargador de .env
│   ├── domain/            entidades y puertos
│   └── usecase/           navegador del workspace, navegación, bot handler
├── docs/
│   ├── DESIGN.md          arquitectura y decisiones
│   └── PRODUCT.md         alcance y hoja de ruta
├── .env.example
├── CONTRIBUTING.md
├── LICENSE
├── README.md
├── SECURITY.md
├── go.mod
├── go.sum
└── remote-bot             binario compilado (arm64)
```

## Licencia

MIT. Consulta `LICENSE`.
