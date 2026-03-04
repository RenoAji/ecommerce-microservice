# TESTING PLAN

This document defines a concrete, incremental testing strategy for this ecommerce microservices repository.

## Goals

- Catch regressions early with fast unit tests.
- Validate service-to-service behavior (gRPC + Redis events + Postgres).
- Prove critical checkout flow works end-to-end through gateway.
- Make all tests runnable locally and in CI with consistent commands.

## Scope

Services:
- user-service
- product-service
- cart-service
- order-service
- payment-service
- delivery-service

Infra dependencies:
- Postgres (per service)
- Redis
- Consul
- Nginx gateway

## Test Pyramid for This Repo

1. Unit tests (majority)
   - Pure service logic, validation, mapper/converter functions, repository behavior with mocks.
2. Integration tests (medium)
   - Real DB/Redis/Consul dependencies with service handlers/repositories/workers.
3. Contract tests (targeted)
   - gRPC compatibility and protobuf message expectations between producer/consumer.
4. End-to-end tests (small, high value)
   - User journey through gateway and async event outcomes.

## Prerequisites

- Go 1.25+ installed.
- Docker + Docker Compose.
- `.env` file configured (see README).
- Midtrans keys optional for local E2E (mock in CI).

## Repository Pre-check (Do Once)

1. Ensure `go.work` includes all service modules used for workspace-level runs.
   - Current file does not include `./user-service`.
2. Ensure each service can run tests independently:
   - `cd <service> && go test ./...`
3. Ensure compose health checks pass before integration/E2E tests.

## Folder and Naming Convention

Use this convention across all services:

- Unit tests
  - Co-located with source files as `*_test.go`.
- Integration tests
  - Prefer `internal/.../*_integration_test.go`.
  - Guard with build tag when needed:
    - `//go:build integration`
- E2E tests
  - Create root folder: `tests/e2e`.
  - Test files: `*_e2e_test.go`.

## Phase 1: Unit Test Baseline (Week 1)

Target: add minimum baseline tests in every service.

Per service, add tests for:
- `internal/service`: happy path + error path.
- `internal/repository`: query/result mapping with mocked DB interface where possible.
- `internal/handler`: request validation and response code behavior using `httptest`.
- `internal/worker` (where present): event parsing and branch handling.

Minimum required count per service:
- 8–12 unit tests per service module.
- Mandatory negative test cases for invalid input.

Command:

```bash
# Run per service (example)
cd order-service && go test ./... -cover
```

Coverage target (initial):
- `internal/service` >= 60%
- repository/handler/worker combined >= 40%

## Phase 2: Integration Tests with Compose (Week 2)

Target: validate real dependency interaction.

### Start environment

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d --build
```

### Run integration tests

Use tag-based execution:

```bash
# Example from each service root
go test ./... -tags=integration -count=1
```

Required integration scenarios:

1. user-service
- user registration persists to user-db.
- login returns JWT for valid credential.

2. product-service
- create/update product persists to product-db.
- stock reservation/update emits expected Redis event.

3. cart-service
- add/remove item persists in Redis.
- product lookup via gRPC failure is handled.

4. order-service
- order creation reads cart + product data and writes order-db.
- out-of-stock path marks order failed.

5. payment-service
- order-created event triggers payment record creation.
- webhook handler updates payment state idempotently.

6. delivery-service
- paid-order event triggers delivery creation in delivery-db.
- failure path emits expected failure event.

## Phase 3: Contract Tests (Week 2–3)

Target: avoid gRPC/proto drift between services.

Contract checks:
- Regenerate stubs and ensure clean diff:
  - `make proto`
- For each consumer, verify:
  - required fields are present in responses.
  - expected gRPC status codes on error.

Add one contract test per integration edge:
- cart -> product
- order -> cart
- order -> product
- order -> payment
- order -> delivery

## Phase 4: End-to-End Flow Tests (Week 3)

Create `tests/e2e` and cover these critical journeys through gateway (`:8080`):

1. Checkout success flow
- Register/login user
- Create product (admin)
- Add to cart
- Create order
- Simulate payment success webhook (or sandbox)
- Verify order status = PAID/DELIVERING (based on current implementation)

2. Stock failure flow
- Product stock low
- Create order with excessive quantity
- Verify order status = FAILED / STOCK_INSUFFICIENT

3. Payment failure flow
- Create order
- Simulate payment failure webhook
- Verify order status updated to payment failure state

Execution:

```bash
go test ./tests/e2e -count=1 -v
```

## CI Pipeline Plan

Run in this order:

1. Static checks
- `go vet ./...` per module
- optional: `staticcheck ./...`

2. Unit tests
- `go test ./... -race -coverprofile=coverage.out`

3. Integration tests
- bring up compose dependencies
- `go test ./... -tags=integration -count=1`

4. E2E smoke
- `go test ./tests/e2e -run TestCheckoutSuccess -count=1`

5. Proto drift check
- run `make proto`
- fail pipeline if generated files changed unexpectedly

## Makefile Additions (Recommended)

Add targets to simplify execution:

- `test-unit`
- `test-integration`
- `test-e2e`
- `test-all`
- `test-coverage`

Suggested behavior:
- `test-unit`: loops all service directories and runs unit tests.
- `test-integration`: ensures compose up first.
- `test-all`: `test-unit` + `test-integration` + `test-e2e`.

## Definition of Done

This testing rollout is complete when:

- Every service has baseline unit tests and passes locally.
- Integration tests exist for DB/Redis + gRPC edges per service.
- At least 2 E2E flows are automated and stable.
- CI enforces unit + integration + e2e smoke + proto drift checks.
- Team can run all tests using a single command (`make test-all`).

## Immediate Next Actions

1. Add `user-service` to `go.work` (if you want workspace-level test runs).
2. Implement baseline unit tests in `order-service` first (orchestration risk is highest).
3. Add Makefile test targets.
4. Add `tests/e2e` with one checkout success test.
