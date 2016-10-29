package server

import (
	"log"

	"github.com/clevergo/websocket"
	"github.com/dustinblackman/tidalwave/sqlquery"
	"github.com/labstack/echo"
	fastengine "github.com/labstack/echo/engine/fasthttp"
	uuid "github.com/satori/go.uuid"
	"github.com/tidwall/gjson"
	dry "github.com/ungerik/go-dry"
	"github.com/valyala/fasthttp"
)

var (
	socketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(ctx *fasthttp.RequestCtx) bool { // Allows all origins, limitations should be done by CORS.
			return true
		},
	}
)

// NewLogLine is created during TidalServers WriteLog method and passed through a channel to verify if a socket is listening for it.
type NewLogLine struct {
	appName string
	entry   string
}

// Socket holds information for a single socket
type Socket struct {
	ID    string
	Query *sqlquery.QueryParams
	Ws    *websocket.Conn
}

// SocketsManager holds information for all sockets and running livetails.
type SocketsManager struct {
	NewLinesChannel chan NewLogLine
	LiveTails       map[string]*Socket
}

func (sm *SocketsManager) watchLines() {
	for line := range sm.NewLinesChannel {
		for _, lt := range sm.LiveTails {
			if (dry.StringInSlice(line.appName, lt.Query.From) || dry.StringInSlice("*", lt.Query.From)) && lt.Query.ProcessLine(line.entry) {
				// TODO Verify this is a good idea to run in a goroutine.
				go lt.Ws.WriteMessage(2, []byte(`{"type": "log", "data": `+line.entry+`}`))
			}
		}
	}
}

func (sm *SocketsManager) addTail(id, query string, ws *websocket.Conn) {
	sm.LiveTails[id] = &Socket{id, sqlquery.New(query), ws}
}

func (sm *SocketsManager) removeTail(socketID string) {
	if _, ok := sm.LiveTails[socketID]; ok {
		delete(sm.LiveTails, socketID)
	}
}

// StartConnection upgrades an echo GET request to a websocket connection
func (sm *SocketsManager) StartConnection(ctx echo.Context) error {
	req := ctx.Request().(*fastengine.Request)
	err := socketUpgrader.Upgrade(req.RequestCtx, func(ws *websocket.Conn) {
		id := uuid.NewV4().String()
		for {
			_, data, err := ws.ReadMessage()
			if err != nil {
				break
			}

			value := gjson.GetBytes(data, "type")
			if !value.Exists() {
				continue
			}

			switch value.String() {
			case "query":
				query := gjson.GetBytes(data, "query")
				if !query.Exists() {
					break
				}
				sm.addTail(id, query.String(), ws)
			}

		}

		sm.removeTail(id)
	})

	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// NewSocketsManager creates a new socket manager with a new lines channel already intialized.
func NewSocketsManager() *SocketsManager {
	socketsManger := SocketsManager{NewLinesChannel: make(chan NewLogLine)}
	go socketsManger.watchLines()
	return &socketsManger
}
