# Tidalwave

_JSON log file parsing with Postgres SQL_

<a href="https://travis-ci.org/busbud/tidalwave"><img src="https://img.shields.io/travis/busbud/tidalwave.svg" alt="Build Status"></a> <a href="https://goreportcard.com/report/github.com/busbud/tidalwave"><img src="https://goreportcard.com/badge/github.com/busbud/tidalwave"></a>

__New Home!__ Tidalwave has moved to Busbud, where it should get a lot more love than it was getting before on [github.com/dustinblackman](https://github.com/dustinblackman).

Tidalwave is an awesomely fast command line, and server for parsing JSON logs. It's meant to be an alternative to application suites like ELK which can be rather resource hungry, where Tidalwave only consumes resources when a search is in progress. It's recorded at being 8 times faster than grep with more in depth parsing than simple regex matching.

Tidalwave works best with logging modules such as [logrus](https://github.com/Sirupsen/logrus), [bunyan](https://github.com/trentm/node-bunyan), [slf4j](https://github.com/savoirtech/slf4j-json-logger), [python-json-logger](https://github.com/madzak/python-json-logger), [json_logger](https://github.com/rsolomo/json_logger) or anything else that outputs JSON logs. It uses Postgres' SQL parser for handling queryies and using them logs.

This project is in it's early stages where it's littered with TODOs, possible bugs, outdated docs, and all the other nifty things that come with early development.

## [Features / Roadmap](./ROADMAP.md)

## How it Works

Products like ELK work by having multiple layers process' to manage and query logs. Elastic search can get quite hungry, and 3rd party services that do something similar is just too expensive for small applications. Tidalwave works by having a folder and file structure that acts as an index, then matching those files to the given query. It only takes up resources on search by taking advantage of multi core systems to quickly parse large log files. Tidalwave is meant to be CPU intensive on queries, but remains on very low resources when idle.

The SQL parser can do basic math (`==`, `!=`, `<=`, `>`, ect) that works with strings, numbers, and date. Parsing multiple applications is as simple as (`SELECT * FROM serverapp, clientapp`). It can also truncate logs to reduce response size (`SELECT time, line.cmd FROM serverapp`).

`date` is a special work as you'll find in more time series applications that's used for. You can either pass a date (`SELECT * FROM serverapp WHERE date = '2016-01-01'`), or pass a full timestamp (`SELECT * FROM serverapp WHERE date = '2016-01-01T01:30:00'`).

### Example

Folder structure is sorted by application name, folder with date, then file names with datetime split by hour.

__Folder Structure__
```
.
+-- serverapp
|   +-- 2016-10-01
|   |   +-- 2016-10-01T01_00_00.log
|   |   +-- 2016-10-01T02_00_00.log
|   +-- 2016-10-02
|   |   +-- 2016-10-02T01_00_00.log
|   |   +-- 2016-10-02T02_00_00.log
|   |   +-- 2016-10-02T03_00_00.log
|   |   +-- 2016-10-02T04_00_00.log
|   |   +-- 2016-10-02T05_00_00.log
|   +-- 2016-10-03
+-- clientapp
```

`2016-10-02T01_00_00.log` was created by the Docker client logger, where the application was using Bunyan to output it's logs.

```
...
{"v":3,"id":"49aa6ad41125","image":"docker-image","name":"server","line":{"name":"server","hostname":"49aa6ad41125","pid":14,"level":30,"cmd":"lol","suffix":"status","msg":"cmd","time":"2016-10-02T00:04:25.172Z","v":0},"host":"a2197bfa39c7"}
{"v":0,"id":"49aa6ad41125","image":"docker-image","name":"server","line":{"name":"server","hostname":"49aa6ad41125","pid":14,"level":30,"cmd":"chat","suffix":"What time is it?","msg":"cmd","time":"2016-10-02T00:04:25.629Z","v":0},"host":"a2197bfa39c7"}
{"v":0,"id":"49aa6ad41125","image":"docker-image","name":"server","line":{"name":"server","hostname":"49aa6ad41125","pid":14,"level":30,"cmd":"chat","suffix":"Pizza.","msg":"cmd","time":"2016-10-02T00:04:33.164Z","v":0},"host":"a2197bfa39c7"}
{"v":0,"id":"49aa6ad41125","image":"docker-image","name":"server","line":{"name":"server","hostname":"49aa6ad41125","pid":14,"level":30,"cmd":"meme","suffix":"fry1 \"meme\"","msg":"cmd","time":"2016-10-02T00:04:35.811Z","v":0},"host":"a2197bfa39c7"}
{"v":0,"id":"49aa6ad41125","image":"docker-image","name":"server","line":{"name":"server","hostname":"49aa6ad41125","pid":14,"level":30,"cmd":"lol","suffix":"status","msg":"cmd","time":"2016-10-02T00:04:36.066Z","v":0},"host":"a2197bfa39c7"}
...
```

Querying all the lines where `cmd` equals `chat` within a set timeframe is as simple as querying a SQL database!

__Query:__
```
SELECT * FROM serverapp WHERE line.cmd = 'chat' and date <= '2016-10-02' and date > '2016-10-02T02:00:00'
```

__Result:__

```
{"v":0,"id":"49aa6ad41125","image":"docker-image","name":"server","line":{"name":"server","hostname":"49aa6ad41125","pid":14,"level":30,"cmd":"chat","suffix":"What time is it?","msg":"cmd","time":"2016-10-02T00:04:25.629Z","v":0},"host":"a2197bfa39c7"}
{"v":0,"id":"49aa6ad41125","image":"docker-image","name":"server","line":{"name":"server","hostname":"49aa6ad41125","pid":14,"level":30,"cmd":"chat","suffix":"Pizza.","msg":"cmd","time":"2016-10-02T00:04:33.164Z","v":0},"host":"a2197bfa39c7"}
```

## Install

Grab the latest release from the [releases](https://github.com/busbud/tidalwave/releases) page, or build from source and install directly from master. Tidalwave is currently built and tested against Go 1.11. A [docker image](https://hub.docker.com/r/busbud/tidalwave/) is also available.

__Quick install for Linux:__
```
curl -Ls "https://github.com/busbud/tidalwave/releases/download/1.0.0/tidalwave-linux-amd64-1.0.0.tar.gz" | tar xz -C /usr/local/bin/
```

__Build From Source:__

A makefile exists to handle all things needed to build and install from source.

```
git pull https://github.com/busbud/tidalwave
cd tidalwave
make install
```


## Usage/Configuration

Configuration can be done either by command line parameters, environment variables, or a JSON file. Please see all available flags with `tidalwave --help`.

To set a configuration, you can take the flag name and export it in your environment or save in one of the three locations for config files.

### Examples

__Flag:__
```
tidalwave --client --max-parallelism 2
```

__Environment:__
```
export TIDALWAVE_CLIENT=true
export TIDALWAVE_MAX_PARALLELISM=2
```

__JSON File:__

Configuration files can be stored in one of the three locations

```sh
./tidalwave.json
/etc/tidalwave.json
$HOME/.tidalwave/tidalwave.json
```
```json
{
  "client": true,
  "max-parallelism": 2
}
```
