# redir

redir is a Go media upload and redirect service. It provides a backend API for auth, product-scoped API keys, multipart media uploads, S3-compatible object storage, public asset redirects, Redis-backed sessions/cache, Postgres persistence, and redirect metrics.

The repository also includes a Go SDK in `pkg/redir` for client applications that need to upload media through the client API.

## Project Layout

- `cmd/api`: server entry point.
- `internal/routes`: Gin route registration.
- `internal/handlers`: HTTP handlers for auth, users, products, client uploads, and assets.
- `internal/middlewares`: auth, API-key validation, ownership checks, asset visibility checks, request logging, recovery, and rate limiting.
- `internal/store`: orchestration layer over Redis cache helpers and Postgres repositories.
- `internal/cache`: Redis sessions, verification tokens, product/media cache, and presigned URL cache.
- `internal/repository`: Postgres persistence using `pgx`.
- `internal/configs`: environment loading, S3-compatible storage setup, and user-agent parser setup.
- `internal/models` and `internal/domain`: persistence models and request/domain types.
- `internal/utils`: key generation, cookies, hashing, IDs, and Gin context helpers.
- `pkg/redir`: Go SDK for product-key client uploads.
- `migrations`: SQL migrations.
- `scripts`: PowerShell and Bash helpers for applying and managing migrations.
- `Dockerfile` and `docker-compose.yml`: container image and local development stack.

## Backend

The server starts from `cmd/api/main.go`, which calls `internal.Listen()`. Startup loads `.env` when present (and otherwise uses system environment variables), initializes Redis, Postgres, S3-compatible storage, a user-agent parser, repositories, and the store layer, then mounts routes under `/api/v1`.

### Readiness

- `GET /api/v1/readyz`

The readiness endpoint checks Redis, Postgres, and the configured storage bucket. It returns `200` when every dependency is available; otherwise it returns `503` with the failed dependency check.

### Auth

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `GET /api/v1/auth/verify`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/oauth/google`
- `GET /api/v1/auth/oauth/google/callback`
- `GET /api/v1/auth/oauth/github`

GitHub OAuth redirect scaffolding exists, but the callback route is currently not mounted.

### Users

- `GET /api/v1/users/me`

### Products

- `POST /api/v1/product`
- `POST /api/v1/product/:id`
- `PUT /api/v1/product/:id`
- `PUT /api/v1/product/:id/assets/:assetId`

Product routes require a valid user session cookie. Product mutation routes also check that the authenticated user owns the product.

### Client API

- `GET /api/v1/client/ping`
- `POST /api/v1/client/upload`
- `PUT /api/v1/client/commit/:batchId`

Client routes are intended for SDK or external application usage. They require:

```http
X-Product: <product-id>
Authorization: Bearer <private-key>
```

Uploads also require:

```http
X-Batch-ID: <uuid>
```

Each multipart file part should include:

```http
X-Sequential-ID: <stable-file-sequence-id>
Content-Type: <mime-type>
```

The backend skips already-uploaded pending files with the same batch ID and sequence ID, which allows the SDK to retry missing files without duplicating successful files.

### Assets

- `GET /api/v1/assets/:assetId`

Asset requests validate the public key, load the completed media row, enforce public visibility, generate or reuse a cached presigned storage URL, save access metrics, and redirect to object storage.

## Go SDK

The SDK lives in `pkg/redir` and can be imported as:

```go
import "github.com/ajaka/redir/pkg/redir"
```

Create a client with the backend URL, product ID, and generated private key:

```go
client, err := redir.New(redir.Config{
	BaseURL:   "http://localhost:5000",
	ProductID: 123,
	APIKey:    "rp_live_xxx",
})
```

`Auto` defaults to `true`. `MaxRetries` defaults to `3`.

```go
client, err := redir.New(redir.Config{
	BaseURL:    "http://localhost:5000",
	ProductID: 123,
	APIKey:    "rp_live_xxx",
	Auto:      redir.Bool(false),
	MaxRetries: 1,
})
```

### SDK Upload Flow

`Upload` assigns stable sequence IDs to the provided files, sends them as a batch, compares the returned media rows with the sent sequence IDs, and commits only when every file is confirmed.

In auto mode:

- Upload the batch.
- Compare sent sequence IDs with returned media sequence IDs.
- Retry only missing files with the same batch ID and original sequence IDs.
- Stop after all files succeed or `MaxRetries` is reached.
- Commit automatically only after all files succeed.

In non-auto mode:

- Upload once.
- If every file succeeds, commit automatically.
- If any sequence ID is missing, return an incomplete batch error and do not commit.

Example:

```go
files := []redir.File{
	redir.FileFromBytes("avatar.png", "image/png", data),
}

result, err := client.Upload(ctx, files, nil)
if err != nil {
	if errors.Is(err, redir.ErrIncompleteBatch) {
		// result.MissingSeqIDs contains the files that did not upload.
	}
	return err
}

_ = result.Media
```

Files opened with `FileFromPath` should be closed when the caller is done:

```go
file, err := redir.FileFromPath("avatar.png")
if err != nil {
	return err
}
defer file.Close()
```

The SDK accepts `io.ReadSeeker` file bodies so it can rewind and retry missing files.

## Configuration

The backend loads configuration from `.env` using `godotenv` when the file is present, and otherwise uses system environment variables. For local development, start with the included defaults:

```sh
cp .env.example .env
```

The sample configuration targets a local Postgres instance, Redis, and MinIO. Set the OAuth variables when those flows are needed.

Supported environment variables include:

- `DATABASE_URL`
- `PORT`
- `SERVER_ADDRESS`
- `ENVIRONMENT`
- `GOOGLE_CLIENT_ID`
- `GOOGLE_CLIENT_SECRET`
- `GOOGLE_REDIRECT_URL`
- `GITHUB_CLIENT_ID`
- `GITHUB_CLIENT_SECRET`
- `GITHUB_REDIRECT_URL`
- `CLIENT_DASHBOARD`
- `CLIENT_LOGIN_URL`
- `STORAGE_SERVICE_ACCESS_KEY_ID`
- `STORAGE_SERVICE_SECRET_ACCESS_KEY`
- `STORAGE_SERVICE_ENDPOINT`
- `BUCKET_NAME`
- `BUCKET_ROOT`
- `DOMAIN`
- `DATA_GET_PATH`
- `REDIS_ADDR`
- `REDIS_PASSWORD`

`ENVIRONMENT=production` enables production behavior where the code checks `cfg.PRODUCTION`, including Gin release mode and secure cookies.

## Local Development

### Docker Compose

Docker Compose provides Postgres, Redis, MinIO (with its console), a one-off migration service, and the API. It creates the storage bucket at application startup.

```sh
make docker-up
```

The API is available at `http://localhost:5000`, the readiness endpoint is `http://localhost:5000/api/v1/readyz`, and the MinIO console is available at `http://localhost:9001`.

Stop the stack with:

```sh
make docker-down
```

`make docker-down` removes the Compose volumes, including local Postgres and MinIO data.

### Run services yourself

Run the server:

```sh
go run cmd/api/main.go
```

Or use the make target:

```sh
make server
```

Run module cleanup when dependency metadata changes:

```sh
go mod tidy
```

Tests in this repository include integration coverage that expects configured external services such as Postgres and Redis. Avoid treating `go test ./...` as a lightweight unit-only check unless those services are available.

## Database

SQL migrations live in `migrations/` and cover users, media, metrics, products, encryption helpers, and later media/metrics schema changes.

Run migrations against `DATABASE_URL` with the migration tooling you use locally. Both helpers expect the `migrate` CLI to be installed; use `.env` to provide `DATABASE_URL` for local runs.

On Linux or macOS:

```sh
bash scripts/migrations.sh up
bash scripts/migrations.sh down 1
bash scripts/migrations.sh create add_example_table
bash scripts/migrations.sh force 9
```

On Windows PowerShell:

```powershell
.\scripts\migrations.ps1 up
.\scripts\migrations.ps1 down 1
.\scripts\migrations.ps1 create add_example_table
.\scripts\migrations.ps1 force 9
```

## Current Limitations

- Response bodies are not fully standardized across all handlers yet.
- Product/media cache invalidation after mutations needs tightening.
- GitHub OAuth callback handling is present as commented code but not currently mounted.
- Some names and error messages still need Go/API polish.
