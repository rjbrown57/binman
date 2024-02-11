package binman

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/rjbrown57/binman/pkg/logging"
)

var (
	ErrNotFound   = errors.New("Release not found")
	ErrBadRequest = errors.New("Bad Request")
	ErrUnknown    = errors.New("Unknown Failure")
)

const (
	v1QueryEndpoint = "v1/query"
	healthEndpoint  = "/healthz"
	metricsEndpoint = "/metrics"
	fileEndpoint    = "/binMan/"
	protocolHeader  = "X-Forwarded-Proto"
)

// https://gin-gonic.com/docs/examples/binding-and-validation/
type BinmanQuery struct {
	Architechure string `json:"architechure" binding:"required"`
	Repo         string `json:"repo" binding:"required"`
	Source       string `json:"source" binding:"required"`
	Version      string `json:"version"`
}

func (q *BinmanQuery) SendQuery(bmurl string) (BinmanQueryResponse, error) {

	resp := BinmanQueryResponse{}

	j, err := json.Marshal(q)
	if err != nil {
		return resp, err
	}

	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", bmurl, v1QueryEndpoint), bytes.NewReader(j))
	if err != nil {
		return resp, err
	}

	req.Header.Set("Content-Type", "application/json")

	r, err := client.Do(req)
	if err != nil {
		return resp, err
	}

	switch r.StatusCode {
	case http.StatusOK:
	case http.StatusBadRequest:
		return resp, ErrBadRequest
	case http.StatusNotFound:
		return resp, ErrNotFound
	default:
		return resp, ErrUnknown
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return resp, err
	}

	err = json.Unmarshal(b, &resp)

	r.Body.Close()

	return resp, err
}

type BinmanQueryResponse struct {
	Repo    string
	DlUrl   string
	Version string `json:"version,omitempty"`
}

// curl -X POST -H "Content-Type: application/json" -d "$payload"  binman.default.svc.cluster.local:9091/v1/query
// example payload {"architechure":"amd64","source":"github.com","repo":"anchore/syft"}
func queryPost(m map[string]BinmanRelease) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r BinmanRelease
		var exists bool

		// We need to validate the query here
		q := BinmanQuery{}
		if err := c.BindJSON(&q); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}

		// If we have synced the requested repo proceed
		if r, exists = m[q.Repo]; exists {
			// If the user has supplied a version, and it's not the latest we need to do a lookup
			if q.Version != "" && q.Version != r.Version {
				err := r.FetchReleaseData(q.Version)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
			}

			qr := BinmanQueryResponse{
				Repo:    q.Repo,
				Version: r.Version,
				// Use c.Request.Header[protocolHeader][0] to respond to the client with the same scheme
				DlUrl: fmt.Sprintf("%s://%s%s", c.Request.Header[protocolHeader][0], c.Request.Host, r.ArtifactPath),
			}
			c.IndentedJSON(http.StatusOK, qr)
		} else {
			c.JSON(http.StatusNotFound, q)
		}
	}
}

func prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func healthzFunc() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.String(http.StatusOK, "%s", "ok")
	}
}

func PopulateLatestMap(bm *BMConfig) map[string]BinmanRelease {
	m := make(map[string]BinmanRelease)

	// Collect latest stored release data for reach release
	for _, rel := range bm.Releases {
		err := rel.FetchReleaseData()
		if err != nil {
			log.Warnf("Failed to populate query cache for %s %s", rel.Repo, err)
		}
		m[rel.Repo] = rel
	}
	return m
}

func WatchServe(bm *BMConfig) {
	log.Infof("Serving /metrics on %s", bm.Config.Watch.Port)

	gin.SetMode(gin.ReleaseMode)

	// Set custom Logger
	r := gin.New()
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/healthz", metricsEndpoint},
	}))
	r.Use(gin.Recovery())

	// https://github.com/gin-gonic/gin/issues/2809
	// https://github.com/gin-gonic/gin/blob/master/docs/doc.md#dont-trust-all-proxies
	r.SetTrustedProxies([]string{"127.0.0.1", "192.168.1.2", "10.0.0.0/8"})

	r.GET(healthEndpoint, healthzFunc())
	r.GET(metricsEndpoint, prometheusHandler())

	if bm.Config.Watch.FileServer {
		r.POST(v1QueryEndpoint, queryPost(bm.Config.Watch.LatestVersionMap))
		r.StaticFS(fileEndpoint, http.Dir(bm.Config.ReleasePath))
	}

	log.Fatalf("%v", r.Run(":"+bm.Config.Watch.Port))
}
