package main

import (
	"log"
	"net/http"
	"path/filepath"

	"github.com/liangrog/admission-webhook-server/pkg/admission/podnodesselector"
	"github.com/liangrog/admission-webhook-server/pkg/utils"
)

// Port to listen to
const (
	ENV_LISTEN_PORT = "LISTEN_PORT"
	listenPort      = ":8443"
)

// TLS secrets
const (
	ENV_TLS_DIR = "TLS_DIR"
	tlsCert     = `tls.crt`
	tlsKey      = `tls.key`
)

func main() {
	tlsDir := utils.GetEnvVal(ENV_TLS_DIR, "/run/secrets/tls")
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
