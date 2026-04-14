### Backend using go + gRPC for a Flutter project me & a bunch of my friends did for uni

---

To-do :

- [X] Write protobuf definitions for the api
- [ ] Write go implementations
- [ ] If possible do typesafety checks/validation
- [X] Figure out what im gonna do with db
- [ ] Figure out what im gonna do with file storage (probably S3)
- [ ] Setup db and its dependencies (ORM, etc.)
- [ ] Get auth working (probably be using JWT)
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
optionally, set the environment variables from [the example](./.env.example)\
then, run the app

```shell
go run ./cmd
```
or if you have [Air](https://github.com/air-verse/air):

```shell
air
```
by default, gRPC server will be served in `localhost:9000`, while the REST gateway in `localhost:9001`

#### OpenAPI Docs

once you run the app, an OpenAPI documentation can be accessed in `localhost:9001/api/v1/docs`
