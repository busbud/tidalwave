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
- [x] `SELECT *`, `SELECT line.cmd`, `SELECT * FROM foo, bar`
- [x] `SELECT COUNT(*)`
- [x] `SELECT DISTINCT(*)`
- [x] `SELECT COUNT(DISTINCT(*))`
- [x] `SELECT * FROM app WHERE key IN ('a', 'b')`
- [x] `SELECT * FROM app WHERE key LIKE '%val%'`
- [x] `SELECT * FROM app WHERE key ILIKE '%vAl%'`
- [x] `SELECT * FROM app WHERE date BETWEEN '2017-04-10' AND '2017-04-12';`
- [ ] `SELECT * FROM app LIMIT 1`
- [ ] `GROUP BY`

#### Dev
- [x] Verbose parameter
- [x] Linter
- [x] Bash autocomplete
- [ ] Tests
- [ ] Benchmarks
