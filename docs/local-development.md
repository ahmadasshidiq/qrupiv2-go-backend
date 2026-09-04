# Local development

Copy `.env.example` to `.env` and replace secrets. Make sure PostgreSQL, Redis, Kafka, Kafka UI, and MinIO are installed and running on the addresses configured in `.env`.

Create the service-owned PostgreSQL databases manually:

```sql
CREATE DATABASE qrupi_auth;
CREATE DATABASE qrupi_core;
CREATE DATABASE qrupi_learning;
CREATE DATABASE qrupi_notification;
CREATE DATABASE qrupi_reporting;
```

The default local configuration expects Kafka on port 9092, Redis on port 6379, and MinIO on port 9000. Adjust `.env` when the locally installed services use different addresses.

Run entry points independently:

```sh
go run ./app/gateway
go run ./app/auth
go run ./app/core
go run ./app/learning
go run ./app/notification
go run ./app/reporting
```

Air continues to run core by default. All configuration is environment-driven; the complete list and local defaults are documented in `.env.example`.

To exercise the Kafka proof, create a user through core (normally through gateway) with `context_type=student`. Watch notification's structured log for `student.created consumed successfully`, then query `GET /api/v1/reports/students` through gateway. Use a JWT containing `institution_id`; reporting filters the read model by the forwarded institution header.
