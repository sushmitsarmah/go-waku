package rpc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/rpc/v2"
	logging "github.com/ipfs/go-log"
	"github.com/status-im/go-waku/waku/v2/node"
)

var log = logging.Logger("wakurpc")

type WakuRpc struct {
	node   *node.WakuNode
	server *http.Server
}

func NewWakuRpc(node *node.WakuNode, address string, port int) *WakuRpc {
	s := rpc.NewServer()
	s.RegisterCodec(NewSnakeCaseCodec(), "application/json")
	s.RegisterCodec(NewSnakeCaseCodec(), "application/json;charset=UTF-8")

	err := s.RegisterService(&DebugService{node}, "Debug")
	if err != nil {
		log.Error(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/jsonrpc", s.ServeHTTP)

	listenAddr := fmt.Sprintf("%s:%d", address, port)

	server := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	return &WakuRpc{node: node, server: server}
}

func (r *WakuRpc) Start() {
	log.Info("Rpc server started at ", r.server.Addr)
	log.Info("server stopped ", r.server.ListenAndServe())
}

func (r *WakuRpc) Stop(ctx context.Context) error {
	return r.server.Shutdown(ctx)
}