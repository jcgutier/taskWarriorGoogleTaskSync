# Taskwarrior to Google Task

Este documento describe el proyecto de sincronización de Taskwarrior entre clientes.

- [Taskwarrior to Google Task](#taskwarrior-to-google-task)
  - [Resumen del progreso de épicos e historias.](#resumen-del-progreso-de-épicos-e-historias)
  - [Objetivo del proyecto](#objetivo-del-proyecto)
  - [Entregables Incluidos (Dentro del Alcance)](#entregables-incluidos-dentro-del-alcance)
  - [Criterios de aceptación](#criterios-de-aceptación)
  - [Exclusiones del Proyecto](#exclusiones-del-proyecto)
  - [Historias de Usuario](#historias-de-usuario)
    - [Roles identificados](#roles-identificados)
    - [Épica 1: Sincronización de tareas entre clientes](#épica-1-sincronización-de-tareas-entre-clientes)
      - [HU-01: Crear tarea en escritorio y verla en otros clientes de escritorio](#hu-01-crear-tarea-en-escritorio-y-verla-en-otros-clientes-de-escritorio)
      - [HU-02: Crear tarea en escritorio y verla en el cliente de Taskwarrior móvil](#hu-02-crear-tarea-en-escritorio-y-verla-en-el-cliente-de-taskwarrior-móvil)
      - [HU-03: Crear tarea en móvil y verla en escritorio](#hu-03-crear-tarea-en-móvil-y-verla-en-escritorio)
      - [HU-04: Completar tarea en cualquier cliente](#hu-04-completar-tarea-en-cualquier-cliente)
      - [HU-04: Sincronización periódica desatendida](#hu-04-sincronización-periódica-desatendida)
    - [Épica 2: Entrada de tareas por voz](#épica-2-entrada-de-tareas-por-voz)
      - [HU-05: Agregar tarea por voz con Google Assistant](#hu-05-agregar-tarea-por-voz-con-google-assistant)
    - [Épica 3: Configuración inicial](#épica-3-configuración-inicial)
      - [HU-06: Autenticar contra Google Tasks la primera vez](#hu-06-autenticar-contra-google-tasks-la-primera-vez)
      - [HU-07: Configurar conexión a PostgreSQL](#hu-07-configurar-conexión-a-postgresql)
      - [HU-08: Modo dry-run para validación](#hu-08-modo-dry-run-para-validación)
    - [Priorización sugerida (MoSCoW)](#priorización-sugerida-moscow)

## Resumen del progreso de épicos e historias.

**Leyenda:** ✅ Completado · ⚠️ Parcial · ⬜ No iniciado

- **Épica 1: Sincronización de tareas entre clientes**: ⚠️ Parcial — HU-01 (sincronización Google→Taskwarrior: en progreso), HU-02 (Taskwarrior→Google: en progreso), HU-03 (completado en ambos clientes: pendiente), HU-04 (daemon periódico: implementado parcialmente)
- **Épica 2: Entrada de tareas por voz**: ⬜ No iniciado — HU-05 depende de integración con Google Assistant
- **Épica 3: Configuración inicial**: ⚠️ Parcial — HU-06 (OAuth2/tokens: disponible), HU-07 (configuración de PostgreSQL: presente en el repositorio), HU-08 (modo `dry-run`: pendiente)

Resumen rápido por historia (selección):

- HU-01: ⚠️ Parcial
- HU-02: ⚠️ Parcial
- HU-03: ⬜ No iniciado
- HU-04: ⚠️ Parcial
- HU-05: ⬜ No iniciado
- HU-06: ✅ Completado
- HU-07: ✅ Completado
- HU-08: ⬜ No iniciado

## Objetivo del proyecto

Desarrollar una solución para sincronizar tareas de taswarrior en la aplicación de escritorio de Ubuntu con algún cliente móvil, como el cliente móvil de Taskwarrior o Google Tasks.
La ventaja de Google Tasks es que se pueden agregar tareas por medio de la voz.
La fuente de información principal quiero que sea la aplicación de escritorio de Taskwarrior.

## Entregables Incluidos (Dentro del Alcance)

Una solución para sincronizar taskwarrior de la aplicación de escritorio de Ubuntu con otros clientes de escritorio y móviles.
Integración con comandos de voz para agregar tareas con el asistente de Google en Android.

## Criterios de aceptación

Habilitar el servidor de taswarrior.
Habilitar el cliente en Android para que muestra las tareas en el servidor.
Las tareas que sean agregadas en un cliente de escritorio Ubuntu o GNU/Linux se sincronicen con los demás clientes.
Agregar tareas por voz con el asistente de Google en Android.

## Exclusiones del Proyecto

No esta dentro del alcance agregar tareas con IA como sugerencias o revisiones automatizadas.
No esta dentro del alcance la sincronización con algún cliente no oficial.
No esta dentro del alcance la sincronización con algún cliente de IOS que no soporte taskwarrior server.

## Historias de Usuario

Los ciclos de trabajo son de 4 semanas (1 mes) por sprint.

### Roles identificados

- **Usuario de escritorio**: persona que usa Taskwarrior en Ubuntu/GNU Linux como herramienta principal de gestión de tareas.
- **Usuario móvil**: la misma persona consultando o creando tareas desde su dispositivo Android.
- **Administrador del sistema**: persona que instala, configura y mantiene el daemon de sincronización (en este proyecto coincide con el usuario final).

---

### Épica 1: Sincronización de tareas entre clientes

#### HU-01: Crear tarea en escritorio y verla en otros clientes de escritorio
**Como** usuario de escritorio,
**quiero** que las tareas que creo en un cliente de escritorio de Taskwarrior en Ubuntu aparezcan automáticamente en otros clientes de escritorio GNU/Linux,
**para** poder consultarlas en movilidad sin tener que duplicarlas manualmente, de manera confiable sin duplicados.

**Criterios de aceptación:**
- Dado que creo una tarea con `task add "comprar leche"` en Ubuntu, la tarea debe de sincronizarse con los demás clientes de manera automática.
- La tarea sincronizada debe conservar título y fecha de vencimiento si la tiene.
- No se deben crear duplicados aunque el ciclo de sync se ejecute varias veces.

Puntos de la historia: 
Sub-tareas:

- Crear

---

#### HU-02: Crear tarea en escritorio y verla en el cliente de Taskwarrior móvil
**Como** usuario de escritorio,
**quiero** que las tareas que creo en Taskwarrior en Ubuntu aparezcan automáticamente en el cliente de Taskwarrior en mi Android,
**para** poder consultarlas en movilidad sin tener que duplicarlas manualmente.

**Criterios de aceptación:**
- Dado que creo una tarea con `task add "comprar leche"` en Ubuntu, cla tarea debe de sincronizarse con los demás clientes de manera automática.
- La tarea sincronizada debe conservar título y fecha de vencimiento si la tiene.
- No se deben crear duplicados aunque el ciclo de sync se ejecute varias veces.

---

#### HU-03: Crear tarea en móvil y verla en escritorio
**Como** usuario móvil,
**quiero** que las tareas que creo en Google Tasks desde mi Android aparezcan en Taskwarrior en mi escritorio Ubuntu,
**para** tener una vista unificada de mis pendientes al regresar a la computadora.

**Criterios de aceptación:**
- Dado que creo una tarea en Google Tasks con estado `needsAction`, cuando se ejecute el ciclo de sync, entonces debe agregarse a Taskwarrior vía `task add`.
- La tarea debe quedar registrada en la base de datos PostgreSQL con su mapping `gid ↔ tid`.

---

#### HU-04: Completar tarea en cualquier cliente
**Como** usuario,
**quiero** que al marcar una tarea como completada en cualquiera de los dos clientes (escritorio o móvil), se refleje en el otro cliente,
**para** no tener tareas “fantasma” que ya terminé.

**Criterios de aceptación:**
- Si completo una tarea en Google Tasks, en el siguiente ciclo Taskwarrior debe marcarla como `completed`.
- Si completo una tarea en Taskwarrior, en el siguiente ciclo Google Tasks debe mostrarla como completada.
- El estado en la tabla `tasks` de PostgreSQL debe actualizarse acorde.

---

#### HU-04: Sincronización periódica desatendida
**Como** administrador del sistema,
**quiero** que el daemon `twgts` se ejecute en segundo plano y sincronice periódicamente sin intervención manual,
**para** olvidarme de la sincronización después de configurarla una vez.

**Criterios de aceptación:**
- El daemon respeta el intervalo definido en `SYNC_INTERVAL_SECONDS` (default 300s).
- Errores transitorios (red, API) no detienen el daemon; se reintenta en el siguiente ciclo.
- Los logs permiten diagnosticar qué se sincronizó en cada ciclo.

---

### Épica 2: Entrada de tareas por voz

#### HU-05: Agregar tarea por voz con Google Assistant
**Como** usuario móvil,
**quiero** poder decirle a Google Assistant “agrega comprar leche a mis tareas”,
**para** capturar pendientes mientras conduzco o tengo las manos ocupadas.

**Criterios de aceptación:**
- La tarea creada por voz aparece en Google Tasks dentro de la lista configurada (`GOOGLE_TASK_LIST_FILTER`).
- En el siguiente ciclo de sync, esa tarea debe llegar a Taskwarrior (cubierto por HU-02).
- No requiere configuración adicional más allá de tener la cuenta Google vinculada.

---

### Épica 3: Configuración inicial

#### HU-06: Autenticar contra Google Tasks la primera vez
**Como** administrador,
**quiero** completar el flujo OAuth2 una sola vez al primer arranque,
**para** que el daemon pueda acceder a mi cuenta de Google Tasks sin pedirme credenciales en cada ejecución.

**Criterios de aceptación:**
- Al arrancar sin `token.json`, el daemon inicia un servidor HTTP en `:8080` e imprime la URL de autorización.
- Tras autorizar, el `token.json` queda persistido en la ruta de `GOOGLE_TASKS_TOKEN_PATH`.
- En arranques posteriores el daemon usa el token cacheado sin requerir intervención.

---

#### HU-07: Configurar conexión a PostgreSQL
**Como** administrador,
**quiero** definir las credenciales y host de PostgreSQL por archivo `config.json` o variables de entorno,
**para** poder desplegar el daemon en distintos entornos (local, Docker, servidor).

**Criterios de aceptación:**
- Las variables `POSTGRES_HOST/USER/PASSWORD/DB` sobreescriben los valores del archivo.
- Si la conexión falla, el daemon registra el error y reintenta sin crashear.
- `docker compose up -d` levanta una instancia funcional para desarrollo local.

---

#### HU-08: Modo dry-run para validación
**Como** administrador,
**quiero** poder ejecutar el daemon con `DRY_RUN=true`,
**para** verificar qué cambios haría sin modificar realmente Taskwarrior ni Google Tasks.

**Criterios de aceptación:**
- Con `DRY_RUN=true`, el daemon registra en logs las acciones que ejecutaría pero no llama a `task add` ni a la API de Google.
- La tabla `tasks` de PostgreSQL no se modifica.

---

### Priorización sugerida (MoSCoW)

| Prioridad | Historia                                                                     |
| --------- | ---------------------------------------------------------------------------- |
| Must      | HU-01, HU-02, HU-04, HU-06, HU-07                                            |
| Should    | HU-03, HU-08                                                                 |
| Could     | HU-05 (depende del soporte de Google Assistant, fuera de control del daemon) |
