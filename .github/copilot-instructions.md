# Instrucciones para Copilot Agent Chat - Proyecto Go Gin

## Buenas Prácticas

- Utiliza siempre la convención de nombres de Go (camelCase para variables y funciones, PascalCase para structs).
- Organiza el código en paquetes claros y reutilizables.
- Maneja los errores explícitamente, nunca ignores los errores.
- Usa contextos (`context.Context`) en handlers y servicios para controlar tiempos de espera y cancelaciones.
- Implementa middlewares para autenticación, logging y manejo de errores.
- Documenta las funciones y métodos importantes usando comentarios en español.
- Utiliza variables de entorno para configuraciones sensibles (por ejemplo, claves API, cadenas de conexión).
- Escribe pruebas unitarias para los controladores y servicios principales.
- Sigue el formato estándar de Go usando `gofmt` o `goimports`.

## Formato de Respuestas

- Responde siempre en español.
- Explica brevemente el propósito del código antes de mostrarlo.
- Usa bloques de código para mostrar ejemplos.
- Si el usuario solicita una explicación, sé claro y conciso.
- Si el usuario pide ayuda con errores, solicita el mensaje de error completo y el fragmento de código relevante.
- Si el usuario pide recomendaciones, sugiere siempre las mejores prácticas de Go y Gin.

## Ejemplo de Respuesta

> Para crear un endpoint básico en Gin que responda "Hola Mundo", puedes usar el siguiente código:
