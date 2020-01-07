package main

import (
	"log"
	"net/http"
	"path/filepath"

	"github.com/liangrog/admission-webhook-server/pkg/admission/podnodesselector"
	"github.com/liangrog/admission-webhook-server/pkg/utils"
)

// TLS secrets
const (
	tlsDir  = `/run/secrets/tls`
	tlsCert = `tls.crt`
	tlsKey  = `tls.key`
)

// Port to listen to
const (
	ENV_LISTEN_PORT = "LISTEN_PORT"
	listenPort      = ":8443"
)

func main() {
	cert := filepath.Join(tlsDir, tlsCert)
	key := filepath.Join(tlsDir, tlsKey)

	mux := http.NewServeMux()

	log.Print("Registering handlers...")
	registerAllHandlers(mux)

	// Config server
	server := &http.Server{
		Addr:    utils.GetEnvVal(ENV_LISTEN_PORT, listenPort),
		Handler: mux,
	}

	// Serve
	log.Print("Starting admission webhook server...")
	log.Fatal(server.ListenAndServeTLS(cert, key))
}

// Register all admission handlers
func registerAllHandlers(mux *http.ServeMux) {
	podnodesselector.Register(mux)
}
