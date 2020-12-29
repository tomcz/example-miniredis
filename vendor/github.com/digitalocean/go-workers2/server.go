package workers

import (
	"context"
	"fmt"
	"net/http"
)

var globalHTTPServer *http.Server

func GlobalAPIHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/stats", globalApiServer.Stats)
	mux.HandleFunc("/retries", globalApiServer.Retries)
	return mux
}

func StartAPIServer(port int) {
	Logger.Println("APIs are available at", fmt.Sprintf("http://localhost:%v/", port))

	globalHTTPServer = &http.Server{Addr: fmt.Sprint(":", port), Handler: GlobalAPIHandler()}
	if err := globalHTTPServer.ListenAndServe(); err != nil {
		Logger.Println(err)
	}
}

func StopAPIServer() {
	if globalHTTPServer != nil {
		globalHTTPServer.Shutdown(context.Background())
	}
}
