## Features / Roadmap

#### Server
- [x] API Server
- [x] Websockets support for live tail
- [ ] Remote logging endpoints

#### Command Line
- [x] Querying by command line
- [ ] Parse specific file rather then parsing folder index

#### Clients
- [x] File watch client
- [x] Docker container client
- [x] PID file watch client
- [ ] Syslog parsing
- [ ] Submitting logs to remote Tidalwave server

#### Queries
- [x] `SELECT *`, `SELECT line.cmd`
- [x] `SELECT COUNT(*)`
- [x] `SELECT DISTINCT(*)`
- [x] `SELECT COUNT(DISTINCT(*))`
- [ ] `SELECT * FROM app WHERE key LIKE '%val%'`

#### Dev
- [x] Verbose parameter
- [x] Linter
- [x] Bash autocomplete
- [ ] Tests
- [ ] Benchmarks
