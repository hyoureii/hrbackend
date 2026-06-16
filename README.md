### Backend using go + gRPC for a Flutter project me & a bunch of my friends did for uni

---

To-do :

- [X] Write protobuf definitions for the api
- [X] Write go implementations
- [X] If possible do typesafety checks/validation
- [X] Figure out what im gonna do with db
- [ ] Figure out what im gonna do with file storage (probably S3)
    - [ ] Implement user avatar storing in file storage
- [X] Setup db and its dependencies (ORM, etc.)
- [X] Get auth working (probably be using JWT)
- [ ] Add query params for a bunch of filtering
- [X] Add a REST gateway
- [X] Generate OpenAPI specs
- [ ] Consider adding support for TOTP
- [ ] Add support for universal link
- [X] Dockerize the whole thing
- [X] Figure out some CI/CD pipeline that just works
- [X] Ship it

---

### Dependencies Installation

in order to develop you'd need these installed:
- `go >= 1.26.1`
- `buf >= 1.70`

get the go dependencies

```shell
go mod download
```
install the tools
> [!TIP]
> this will not install the tools as a binary that you can use from your $PATH, if you wanna do that do `go install tool`

```shell
go get tool
```

### Database & Cache

you can setup your own [Postgres](https://www.postgresql.org/) and [Redis](https://redis.io/) instance or,\
i included a [docker-compose file](./docker-compose.yaml) that you can use to spin up the db

```shell
docker compose up -d
```

### Running the app

set the environment variables from [the example](./.env.example)\

then, run the app

```shell
go tool air
```
> [!TIP]
> or if you already have [Air](https://github.com/air-verse/air) installed on your $PATH, just run `air`

\
at this point the app is running, but no data will be available to work with. for running the seeding and migration script see [Migrating and seeding the db](#migrating-and-seeding-the-db) down below\
by default, gRPC server will be served in `localhost:9000`, while the REST gateway in `localhost:9001`

#### OpenAPI Docs

once you run the app, an OpenAPI documentation can be accessed in `localhost:9001/api/v1/docs`

### Migrating and seeding the db

> [!NOTE]
> if you had problem connecting to the database, its likely because the script doesnt read your .env file as im using air to do that.\
> in that case, you can manually set the environment variables before running the script, for example:
> ```shell
> export POSTGRES_PORT=9001
> go run ./scripts/seed.go
> ```

once you got the app up and running you can run the migration as such

```shell
go run ./scripts/migrate.go push
```
after the tables are created, you can fill it up with your own data, or use the dummy data from the seeding script

```shell
go run ./scripts/seed.go
```
