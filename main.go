package main

import (
	"net/http"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/liangrog/admission-webhook-server/pkg/admission/podnodesselector"
	"github.com/liangrog/admission-webhook-server/pkg/utils"
)

const (
	// Setting log level.
	// Available log levels:
	// panic (zerolog.PanicLevel, 5)
	// fatal (zerolog.FatalLevel, 4)
	// error (zerolog.ErrorLevel, 3)
	// warn (zerolog.WarnLevel, 2)
	// info (zerolog.InfoLevel, 1)
	// debug (zerolog.DebugLevel, 0)
	// trace (zerolog.TraceLevel, -1)
	ENV_LOG_LEVEL = "LOG_LEVEL"
)

const (
	// SSL mount dir
	ENV_TLS_DIR   = "TLS_DIR"
	defaultTlsDir = "/run/secrets/tls"

	// SSL certificate filename
	ENV_TLS_CERT_FILENAME = "TLS_CERT_FILENAME"
	defaultTlsCert        = "tls.crt"

	// SSL private key filename
	ENV_TLS_KEY_FILENAME = "TLS_KEY_FILENAME"
	defaultTlsKey        = "tls.key"

	// Port to be listen to by the server
	port = ":8443"
)

func main() {
	// Setting global log level. Default info level
	zerolog.SetGlobalLevel(utils.GetZeroLogLevel(utils.GetEnvVal(ENV_LOG_LEVEL, "info")))

	log.Info().Msg("Registering handlers...")

	// New mux server
	mux := http.NewServeMux()

	// Register webhook handlers
	registerHandlers(mux)

	// Config server
	server := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	log.Info().Msg("Starting admission webhook server...")

	// Set ssl certificate and private key
	tlsDir := utils.GetEnvVal(ENV_TLS_DIR, defaultTlsDir)
	tlsCert := utils.GetEnvVal(ENV_TLS_CERT_FILENAME, defaultTlsCert)
	tlsKey := utils.GetEnvVal(ENV_TLS_KEY_FILENAME, defaultTlsKey)
	cert := filepath.Join(tlsDir, tlsCert)
	key := filepath.Join(tlsDir, tlsKey)

	log.Debug().Msgf("Preparing to load TLS certificate [%s] and private key [%s]", cert, key)

	// Serve
	log.Fatal().Err(server.ListenAndServeTLS(cert, key))
}

// Register all admission webhook handlers
func registerHandlers(mux *http.ServeMux) {
	podnodesselector.Register(mux)
}
