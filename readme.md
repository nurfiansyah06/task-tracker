
## About Task Tracker

This is a simple project from (https://roadmap.sh/projects/task-tracker)

## Run the API

Requires Go 1.22 or later:

```sh
go run .
```

The server listens on all interfaces at port `3000` (locally, `http://localhost:3000`). Tasks persist in `task.json`
in the working directory. A missing file starts an empty task list.
This local API has no authentication.

## Endpoints

### GET /tasks

Returns `200 OK` with an array of tasks, or `[]` when empty.
Each task has `id`, `name`, and `status`.

```sh
curl http://localhost:3000/tasks
```

### POST /tasks

Accepts a JSON object with a required nonblank `name` and optional `status`
(`Progress` or `Done`, default `Progress`). IDs are assigned automatically
using the highest existing ID plus one. Unknown fields are rejected.
Returns `201 Created` with the saved task.

```sh
curl -X POST http://localhost:3000/tasks \
  -H 'Content-Type: application/json' \
  -d '{"name":"Buy groceries","status":"Progress"}'
```

Errors are JSON objects such as `{"error":"invalid JSON body"}`.
Invalid input returns `400`, unsupported methods return `405` with
`Allow: GET, POST`, and storage failures return `500`.

### PUT /tasks/{id}

Replaces a task's name and status, preserving its ID. Both fields are required;
send only `name` and `status`, not `id`. Returns `200` with the updated task.

```sh
curl -X PUT http://localhost:3000/tasks/8 \
  -H 'Content-Type: application/json' \
  -d '{"name":"Buy Car","status":"Done"}'
```

### DELETE /tasks/{id}

Deletes a task and returns `204 No Content` with no response body.

```sh
curl -X DELETE http://localhost:3000/tasks/8
```

Both item endpoints return `404` if the task does not exist, `400` for invalid
IDs or input, and `500` for storage failures. Unsupported item methods return
`405` with `Allow: PUT, DELETE`.

## Deployment health check

Configure the ECS container and target group to use port `3000`, with
health-check path `/health` and expected status `200`.
`GET /health` returns `{"status":"ok"}` without accessing task storage.
It checks that the HTTP server responds, not that storage is writable.

Container-local `task.json` is ephemeral and is not shared across replicas.
Use durable shared storage before relying on this API for persistent deployed data.
