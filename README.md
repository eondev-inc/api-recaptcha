# API reCAPTCHA Enterprise

API REST en Go para validar tokens de reCAPTCHA Enterprise de Google. Diseñada para ser consumida desde aplicaciones cliente (como React) con autenticación mediante API Key.

## 🚀 Características

- ✅ Validación de tokens reCAPTCHA Enterprise
- 🔐 Autenticación mediante API Key (header `X-API-Key`)
- 🎯 Soporte para acciones personalizadas (`expectedAction`)
- 📊 Retorna score de riesgo y análisis completo
- ⚙️ Configuración mediante variables de entorno
- 🏗️ Arquitectura limpia y modular

## 📋 Requisitos

- Go 1.22 o superior
- Cuenta de Google Cloud con reCAPTCHA Enterprise habilitado
- API Key de Google reCAPTCHA Enterprise
- Site Key de reCAPTCHA

## 🛠️ Instalación

1. **Clona el repositorio**
```bash
git clone <tu-repositorio>
cd api-recaptcha
```

2. **Instala las dependencias**
```bash
go mod download
```

3. **Configura las variables de entorno**

Crea un archivo `.env` en la raíz del proyecto (puedes usar `.env.example` como plantilla):

```bash
cp .env.example .env
```

Edita el archivo `.env` con tus credenciales:

```properties
# Application API Key - Used by clients to authenticate with this API
APP_API_KEY=tu_api_key_generada

# Google reCAPTCHA Enterprise API Key
GOOGLE_RECAPTCHA_API_KEY=tu_google_api_key

# Google reCAPTCHA Site Key
GOOGLE_RECAPTCHA_SITE_KEY=6LfTUuorAAAAAEYi8wmrchk8zaxcasstljmj-ZZT

# Google reCAPTCHA Enterprise Project ID
GOOGLE_RECAPTCHA_PROJECT_ID=tu_project_id

# Server Port (optional, defaults to 8080)
PORT=8080
```

## 🚀 Uso

### Iniciar el servidor

```bash
go run ./cmd/server/main.go
```

El servidor estará disponible en `http://localhost:8080`

### Endpoint disponible

#### POST `/api/v1/recaptcha/verify`

Valida un token de reCAPTCHA Enterprise.

**Headers requeridos:**
```
Content-Type: application/json
X-API-Key: tu_app_api_key
```

**Body de la solicitud:**
```json
{
  "token": "TOKEN_RECAPTCHA_DEL_CLIENTE",
  "action": "login"
}
```

**Parámetros:**
- `token` (string, requerido): Token generado por `grecaptcha.enterprise.execute()`
- `action` (string, opcional): Acción específica que se está validando

**Respuesta exitosa (200 OK):**
```json
{
  "valid": true,
  "score": 0.9,
  "action": "login",
  "invalidReason": "",
  "reasons": [],
  "createTime": "2025-10-15T10:30:00Z"
}
```

**Respuesta de error:**
```json
{
  "error": "recaptcha verification failed",
  "details": "mensaje de error detallado"
}
```

### Ejemplo con cURL

```bash
curl -X POST http://localhost:8080/api/v1/recaptcha/verify \
  -H "Content-Type: application/json" \
  -H "X-API-Key: tu_app_api_key" \
  -d '{
    "token": "03AL8dmw9q7h...",
    "action": "login"
  }'
```

### Integración con React

```javascript
// 1. Ejecuta reCAPTCHA en el cliente
const token = await window.grecaptcha.enterprise.execute(
  'TU_SITE_KEY', 
  { action: 'login' }
);

// 2. Envía el token a tu API
const response = await fetch('http://localhost:8080/api/v1/recaptcha/verify', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-API-Key': 'tu_app_api_key'
  },
  body: JSON.stringify({
    token: token,
    action: 'login'
  })
});

const result = await response.json();

if (result.valid && result.score >= 0.5) {
  // Usuario validado
  console.log('Usuario válido, score:', result.score);
} else {
  // Posible bot
  console.log('Validación fallida');
}
```

## 📁 Estructura del Proyecto

```
api-recaptcha/
├── cmd/
│   └── server/
│       └── main.go              # Punto de entrada de la aplicación
├── internal/
│   ├── handler/
│   │   └── verify.go            # Handler HTTP para verificación
│   ├── middleware/
│   │   └── apikey.go            # Middleware de autenticación
│   └── service/
│       └── recaptcha.go         # Lógica de negocio reCAPTCHA
├── .env                         # Variables de entorno (no versionado)
├── .env.example                 # Plantilla de variables de entorno
├── .gitignore                   # Archivos ignorados por Git
├── go.mod                       # Dependencias del proyecto
├── go.sum                       # Checksums de dependencias
├── request.json                 # Ejemplo de request para pruebas
└── README.md                    # Este archivo
```

## 🔒 Seguridad

- **API Key**: La aplicación requiere una API Key válida en el header `X-API-Key` para todas las peticiones
- **Variables de entorno**: Las credenciales sensibles se manejan mediante variables de entorno
- **`.env` en .gitignore**: El archivo `.env` está excluido del control de versiones para proteger las credenciales

## 🧪 Testing

Para ejecutar las pruebas:

```bash
go test ./...
```

Para ver cobertura:

```bash
go test -cover ./...
```

## 🏗️ Compilación

Para compilar el binario:

```bash
go build -o bin/api-recaptcha ./cmd/server
```

Ejecutar el binario:

```bash
./bin/api-recaptcha
```

## 🐳 Docker (Opcional)

Crear una imagen Docker:

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o api-recaptcha ./cmd/server

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/api-recaptcha .
EXPOSE 8080
CMD ["./api-recaptcha"]
```

Construir y ejecutar:

```bash
docker build -t api-recaptcha .
docker run -p 8080:8080 --env-file .env api-recaptcha
```

## 📚 Documentación de referencia

- [Google reCAPTCHA Enterprise API](https://cloud.google.com/recaptcha-enterprise/docs)
- [Gin Web Framework](https://gin-gonic.com/)
- [godotenv](https://github.com/joho/godotenv)

## 🤝 Contribuciones

Las contribuciones son bienvenidas. Por favor:

1. Haz fork del proyecto
2. Crea una rama para tu feature (`git checkout -b feature/nueva-funcionalidad`)
3. Commit tus cambios (`git commit -am 'Agrega nueva funcionalidad'`)
4. Push a la rama (`git push origin feature/nueva-funcionalidad`)
5. Abre un Pull Request

## 📝 Licencia

Este proyecto está bajo la licencia MIT. Ver el archivo `LICENSE` para más detalles.

## 👤 Autor

Tu nombre / Tu organización

## 🐛 Problemas conocidos

### Error 403: "Requests to this API method are blocked"

Si recibes este error, verifica:

1. Que la API de reCAPTCHA Enterprise esté habilitada en tu proyecto de Google Cloud
2. Que tu API Key tenga los permisos correctos
3. Considera usar una Service Account en lugar de una API Key simple para mayor seguridad

## 📧 Contacto

Para preguntas o soporte, abre un issue en el repositorio.

---

⭐ Si este proyecto te fue útil, considera darle una estrella en GitHub!
