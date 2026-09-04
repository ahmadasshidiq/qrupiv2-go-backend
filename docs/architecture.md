# QRUPI V2 service architecture

QRUPI V2 has exactly six primary services. Entry points live in `app/`; domain modules remain in `src/`; reusable infrastructure lives in `libs/`.

```text
Client -> Gateway -> Auth / Core / Learning
                         |
                       Kafka
                         |
                 Notification / Reporting
```

| Service | Responsibility | Default port | Data owner |
|---|---|---:|---|
| gateway | Routing, JWT validation, CORS, rate limiting, IDs and access logs | 3000 | None |
| auth | Authentication, credentials, tokens and auth events | 3001 | Auth schema |
| core | Users, roles, institutions and other master data | 3002 | Core schema |
| learning | LMS domains | 3003 | Learning schema |
| notification | Kafka-driven channel delivery | 3004 | Notification schema |
| reporting | Read models, analytics and exports | 3005 | Reporting schema |

Services never query another service's database. Immediate cross-service reads use an API; asynchronous propagation uses Kafka. Institution identity comes from the validated JWT and is forwarded by the gateway as `X-Institution-ID`; request bodies are not an authoritative institution source.

## Kafka

Reusable transport code is in `libs/kafka`. Business publishers and handlers remain in their `src` module or service. The default event topic is `qrupi.events`; exhausted or invalid messages go to `qrupi.events.dlq` after a bounded retry count.

Every message uses this envelope:

```json
{
  "event_id": "uuid",
  "event_type": "student.created",
  "version": 1,
  "timestamp": "2026-08-31T10:00:00Z",
  "source": "core-service",
  "institution_id": "uuid",
  "correlation_id": "uuid",
  "data": {}
}
```

Event names follow `<domain>.<entity>.<action>`. Consumers persist `event_id` in their own `processed_events` table. Duplicates are acknowledged without repeating the handler. Failed handlers retry with bounded backoff and are then published to the DLQ, so there is no infinite retry loop.

The initial end-to-end proof uses a core user created with `context_type=student`. Core publishes `student.created`; notification logs successful consumption; reporting upserts `student_read_models`. The upsert and processed-event check make the reporting projection idempotent.

## Import and export

Large imports stay with the data owner: gateway upload → owning service validation → persisted import job → Kafka worker → batched processing → progress and failed-row report. Jobs require an idempotency key and support CSV/XLSX.

Large exports belong to reporting: request → persisted export job → Kafka → export worker → CSV/XLSX/PDF in MinIO → completed event and download URL. Reporting builds its own read models from events rather than running analytical queries against another service database.

## Observability

Every entry point exposes `/health`, `/ready`, and `/metrics`, emits structured JSON logs, and handles graceful shutdown. The gateway creates or propagates `X-Request-ID` and `X-Correlation-ID`; the latter is copied into Kafka envelopes. The metrics endpoint is Prometheus-compatible and is intentionally minimal pending full OpenTelemetry instrumentation.
