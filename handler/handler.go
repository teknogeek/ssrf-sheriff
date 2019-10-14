package handler

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"path"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/teknogeek/ssrf-sheriff/httpserver"
	"go.uber.org/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SerializableResponse is a generic type which both can be safely serialized to both XML and JSON
type SerializableResponse struct {
	SecretToken string `json:"token" xml:"token"`
}

// SSRFSheriffRouter is a wrapper around mux.Router to handle HTTP requests to the sheriff, with logging
type SSRFSheriffRouter struct {
	logger    *zap.Logger
	ssrfToken string
}

// NewHTTPServer provides a new HTTP server listener
func NewHTTPServer(
	mux *mux.Router,
	cfg config.Provider,
) *http.Server {

	return &http.Server{
		Addr:    cfg.Get("http.address").String(),
		Handler: mux,
	}
}

// NewSSRFSheriffRouter returns a new SSRFSheriffRouter which is used to route and handle all HTTP requests
func NewSSRFSheriffRouter(
	logger *zap.Logger,
	cfg config.Provider,
) *SSRFSheriffRouter {
	return &SSRFSheriffRouter{
		logger:    logger,
		ssrfToken: cfg.Get("ssrf_token").String(),
	}
}

// StartServer starts the HTTP server
func StartServer(server *http.Server, lc fx.Lifecycle) {
	h := httpserver.NewHandle(server)
	lc.Append(fx.Hook{
		OnStart: h.Start,
		OnStop:  h.Shutdown,
	})
}

// PathHandler is the main handler for all inbound requests
func (s *SSRFSheriffRouter) PathHandler(w http.ResponseWriter, r *http.Request) {
	fileExtension := filepath.Ext(r.URL.Path)
	contentType := mime.TypeByExtension(fileExtension)
	var response string

	switch fileExtension {
	case ".json":
		res, _ := json.Marshal(SerializableResponse{SecretToken: s.ssrfToken})
		response = string(res)
	case ".xml":
		res, _ := xml.Marshal(SerializableResponse{SecretToken: s.ssrfToken})
		response = string(res)
	case ".html":
		tmpl := readTemplateFile("html.html")
		response = fmt.Sprintf(tmpl, s.ssrfToken, s.ssrfToken)
	case ".csv":
		tmpl := readTemplateFile("csv.csv")
		response = fmt.Sprintf(tmpl, s.ssrfToken)
	case ".txt":
		response = fmt.Sprintf("token=%s", s.ssrfToken)

	// TODO: dynamically generate these formats with the secret token rendered in the media
	case ".gif":
		response = readTemplateFile("gif.gif")
	case ".png":
		response = readTemplateFile("png.png")
	case ".jpg", ".jpeg":
		response = readTemplateFile("jpeg.jpg")
	case ".mp3":
		response = readTemplateFile("mp3.mp3")
	case ".mp4":
		response = readTemplateFile("mp4.mp4")
	default:
		response = s.ssrfToken
	}

	if contentType == "" {
		contentType = "text/plain"
	}

	s.logger.Info("New inbound HTTP request",
		zap.String("IP", r.RemoteAddr),
		zap.String("Path", r.URL.Path),
		zap.String("Response Content-Type", contentType),
		zap.Any("Request Headers", r.Header),
	)

	responseBytes := []byte(response)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Secret-Token", s.ssrfToken)
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}

func readTemplateFile(templateFileName string) string {
	data, err := ioutil.ReadFile(path.Join("templates", path.Clean(templateFileName)))
	if err != nil {
		return ""
	}
	return string(data)
}

// NewServerRouter returns a new mux.Router for handling any HTTP request to /.*
func NewServerRouter(s *SSRFSheriffRouter) *mux.Router {
	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(s.PathHandler)
	return router
}

// NewConfigProvider returns a config.Provider for YAML configuration
func NewConfigProvider() (config.Provider, error) {
	return config.NewYAMLProviderFromFiles("config/base.yaml")
}

// NewLogger returns a new *zap.Logger
func NewLogger() (*zap.Logger, error) {
	zapConfig := zap.NewProductionConfig()
	zapConfig.Encoding = "console"
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapConfig.DisableStacktrace = true

	return zapConfig.Build()
}
