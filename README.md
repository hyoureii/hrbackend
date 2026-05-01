### Backend using go + gRPC for a Flutter project me & a bunch of my friends did for uni

---

To-do :

- [X] Write protobuf definitions for the api
- [X] Write go implementations
- [ ] If possible do typesafety checks/validation
- [X] Figure out what im gonna do with db
- [ ] Figure out what im gonna do with file storage (probably S3)
- [X] Setup db and its dependencies (ORM, etc.)
- [X] Get auth working (probably be using JWT)
- [X] Add a REST gateway
- [X] Generate OpenAPI specs
- [ ] Consider adding support for TOTP
- [ ] Dockerize the whole thing
- [ ] Figure out some CI/CD pipeline that just works
- [ ] Ship it

---

### Run the app

get the dependencies

```shell
go mod download
```
install the tools
> this will not install the tools as a binary that you can use from your $PATH, if you wanna do that do `go install tool`

```shell
go get tool
```

\
you can setup your own [Postgres](https://www.postgresql.org/) instance or,\
i included a [docker-compose file](./docker-compose.yaml) that you can use to spin up the db

\
set the environment variables from [the example](./.env.example) (optional)\

then, run the app

```shell
go tool air
```
\
or if you already have [Air](https://github.com/air-verse/air) installed on your $PATH:

```shell
air
```
by default, gRPC server will be served in `localhost:9000`, while the REST gateway in `localhost:9001`

#### OpenAPI Docs

once you run the app, an OpenAPI documentation can be accessed in `localhost:9001/api/v1/docs`
