package main

import (
	"bytes"
	_ "embed"
	"flag"
	"html/template"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/WillMorrison/JouleQuestCardGame/assets"
	"github.com/WillMorrison/JouleQuestCardGame/core"
	"github.com/WillMorrison/JouleQuestCardGame/engine"
	"github.com/WillMorrison/JouleQuestCardGame/eventlog"
	"github.com/WillMorrison/JouleQuestCardGame/params"
)

type playerAction struct {
	GlobalIndex int
	Type        engine.ActionType
	AssetType   assets.Type
	Cost        int
}

type playerInfo struct {
	Index    int
	Status   core.PlayerStatus
	Reason   core.LossCondition
	Money    int
	AssetMix assets.AssetMix
	Actions  []playerAction
}

type gameInfo struct {
	Status                   core.GameStatus
	Reason                   core.LossCondition
	Round                    int
	CarbonEmissions          int
	LastRoundAssetMix        assets.AssetMix
	LastRoundGridStability   core.GridStability
	LastRoundPriceVolatility core.PriceVolatility
	TakeoverPool             assets.AssetMix
	Players                  []playerInfo
	Params                   params.Params
}

//go:embed assets/game.html.tmpl
var templateSrc []byte

var pageTemplate = template.Must(template.New("page").Parse(string(templateSrc)))

func renderGame(g engine.ProceduralGameState) ([]byte, error) {
	actions := g.PossibleActions()
	gs := g.Game()
	tmplInput := gameInfo{
		Status:                   gs.Status,
		Reason:                   gs.Reason,
		Round:                    gs.Round,
		CarbonEmissions:          gs.CarbonEmissions,
		LastRoundGridStability:   gs.LastSnapshot.GridStability,
		LastRoundPriceVolatility: gs.LastSnapshot.PriceVolatility,
		LastRoundAssetMix:        gs.LastSnapshot.AssetMix,
		TakeoverPool:             gs.TakeoverPool,
		Params:                   gs.Params,
	}
	for pi, p := range gs.Players {
		info := playerInfo{
			Index:    pi,
			Status:   p.Status,
			Reason:   p.Reason,
			Money:    p.Money,
			AssetMix: p.Assets,
		}
		for ai, a := range actions {
			if pi == a.PlayerIndex {
				info.Actions = append(info.Actions, playerAction{GlobalIndex: ai, Type: a.Type, AssetType: a.AssetType, Cost: a.Cost})
			}
		}
		tmplInput.Players = append(tmplInput.Players, info)
	}
	var b bytes.Buffer
	if err := pageTemplate.Execute(&b, tmplInput); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

type handler struct {
	g      engine.ProceduralGameState
	logger eventlog.Logger
}

func (h handler) Mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.render)
	mux.HandleFunc("POST /{$}", h.applyAction)
	mux.HandleFunc("/new", h.newGame())
	return mux
}

func (h handler) render(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "text/html")
	b, err := renderGame(h.g)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(err.Error()))
		log.Printf("Render error: %v", err)
		return
	}
	resp.Write(b)
}

func (h *handler) applyAction(resp http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	actionIndex := req.Form.Get("action")
	if actionIndex == "" {
		resp.WriteHeader(http.StatusBadRequest)
		log.Printf("No action provided")
	} else {
		ai, err := strconv.Atoi(actionIndex)
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			log.Printf("Invalid action index: %v", err)
		} else {
			actions := h.g.PossibleActions()
			if ai < 0 || ai >= len(actions) {
				resp.WriteHeader(http.StatusBadRequest)
				log.Printf("Action index out of range: %d", ai)
			} else {
				h.g.ApplyPlayerAction(actions[ai])
			}
		}
	}
	h.render(resp, req)
}

func (h *handler) newGame() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		gameParams := params.Default
		numPlayers := 4

		req.ParseForm()
		numPlayersStr := req.Form.Get("players")
		if numPlayersStr != "" {
			n, err := strconv.Atoi(numPlayersStr)
			if err != nil {
				resp.WriteHeader(http.StatusBadRequest)
				resp.Write([]byte("Invalid number of players"))
				return
			}
			if _, ok := gameParams.StartingFossilAssetsPerPlayer[n]; !ok {
				resp.WriteHeader(http.StatusBadRequest)
				resp.Write([]byte("Unsupported number of players"))
				return
			}
			numPlayers = n
		}

		g, err := engine.NewProceduralGame(numPlayers, gameParams, h.logger)
		if err != nil {
			log.Fatalf("Unable to create Game: %v", err)
		}
		h.g = *g
		log.Print("Started a new game")

		http.Redirect(resp, req, "/", http.StatusSeeOther)
	}
}

type logWriter struct{}

func (_ logWriter) Write(p []byte) (n int, err error) {
	log.Print(string(p))
	return len(p), nil
}

func NewHandler() handler {
	eventLogger := eventlog.NewJsonLogger(logWriter{})
	g, err := engine.NewProceduralGame(4, params.Default, eventLogger)
	if err != nil {
		log.Fatalf("Unable to create Game: %v", err)
	}
	return handler{g: *g, logger: eventLogger}
}

func main() {
	var netAddr string
	flag.StringVar(&netAddr, "addr", "localhost:0", "Address in host:port format. If the port is 0 or empty, an unused port will be selected.")
	flag.Parse()

	h := NewHandler()
	listener, err := net.Listen("tcp", netAddr)
	if err != nil {
		log.Fatalf("Unable to start TCP listener: %v", err)
	}
	log.Printf("Listening on %s", listener.Addr().String())
	server := http.Server{Handler: h.Mux()}
	server.Serve(listener)
}
