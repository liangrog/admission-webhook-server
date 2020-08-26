package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/trilogy-group/admission-webhook-server/pkg/admission/podnodesselector"
	"github.com/trilogy-group/admission-webhook-server/pkg/utils"
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

	mux := http.NewServeMux()

	log.Print("Registering handlers ...")
	registerAllHandlers(mux)

	// Configure server
	server := &http.Server{
		Addr:           utils.GetEnvVal(ENV_LISTEN_PORT, listenPort),
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1048576
	}

	// Start server
	log.Print("Starting admission webhook server ...")

	cert := filepath.Join(tlsDir, tlsCert)
	key := filepath.Join(tlsDir, tlsKey)
	log.Fatal(server.ListenAndServeTLS(cert, key))
}

// handleRoot handles root path (i.e. /)
func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello from admission webhook server , you have hit : %q", html.EscapeString(r.URL.Path))
}

// registerAllHandlers registers handlers for all path
func registerAllHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/", handleRoot)
	podnodesselector.Register(mux)
}
