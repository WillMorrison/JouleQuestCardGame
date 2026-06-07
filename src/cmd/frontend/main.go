package main

import (
	"bytes"
	_ "embed"
	"flag"
	"html/template"
	"log"
	"net"
	"net/http"

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

type page struct {
	Status                   core.GameStatus
	Reason                   core.LossCondition
	Round                    int
	CarbonEmissions          int
	LastRoundAssetMix        assets.AssetMix
	LastRoundGridStability   core.GridStability
	LastRoundPriceVolatility core.PriceVolatility
	TakeoverPool             assets.AssetMix
	Players                  []playerInfo
}

//go:embed assets/game.html.tmpl
var templateSrc []byte

var pageTemplate = template.Must(template.New("page").Parse(string(templateSrc)))

func renderGame(g engine.ProceduralGameState) ([]byte, error) {
	actions := g.PossibleActions()
	gs := g.Game()
	tmplInput := page{
		Status:                   gs.Status,
		Reason:                   gs.Reason,
		Round:                    gs.Round,
		CarbonEmissions:          gs.CarbonEmissions,
		LastRoundGridStability:   gs.LastSnapshot.GridStability,
		LastRoundPriceVolatility: gs.LastSnapshot.PriceVolatility,
		LastRoundAssetMix:        gs.LastSnapshot.AssetMix,
		TakeoverPool:             gs.TakeoverPool,
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
	log.Printf("Game: %+v", gs)
	log.Printf("Actions: %+v", actions)
	log.Printf("Template Input: %+v", tmplInput)
	var b bytes.Buffer
	if err := pageTemplate.Execute(&b, tmplInput); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func GameRenderHandler(g *engine.ProceduralGameState) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("Content-Type", "text/html")
		b, err := renderGame(*g)
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			resp.Write([]byte(err.Error()))
			log.Printf("Render error: %v", err)
			return
		}
		resp.WriteHeader(http.StatusOK)
		resp.Write(b)
	}
}

type logWriter struct{}

func (_ logWriter) Write(p []byte) (n int, err error) {
	log.Print(string(p))
	return len(p), nil
}

func main() {
	var netAddr string
	flag.StringVar(&netAddr, "addr", "localhost:0", "Address in host:port format. If the port is 0 or empty, an unused port will be selected.")
	flag.Parse()

	eventLogger := eventlog.NewJsonLogger(logWriter{})
	g, err := engine.NewProceduralGame(4, params.Default, eventLogger)
	if err != nil {
		log.Fatalf("Unable to create Game: %v", err)
	}
	http.HandleFunc("/game", GameRenderHandler(g))
	listener, err := net.Listen("tcp", netAddr)
	if err != nil {
		log.Fatalf("Unable to start TCP listener: %v", err)
	}
	log.Printf("Listening on %s", listener.Addr().String())
	server := http.Server{}
	server.Serve(listener)
}
