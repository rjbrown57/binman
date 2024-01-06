package binman

import (
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/rjbrown57/binman/pkg/logging"
)

func healthzFunc() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}
}

func watchServe(config Watch, releasePath string) {
	log.Infof("Serving /metrics on %s", config.Port)
	http.HandleFunc("/healthz", healthzFunc())
	http.Handle("/metrics", promhttp.Handler())

	if config.FileServer && config.Sync {
		http.Handle("/", http.FileServer(http.Dir(releasePath)))
	}

	log.Fatalf("%v", http.ListenAndServe(":"+config.Port, nil))
}

// Start watch command to expose metrics and sync on a schedule
func StartWatch(bm *BMConfig) {

	log.Debugf("watch config = %+v", bm.Config.Watch)

	go getSpinner(true)

	go func() {
		for {

			c := make(chan BinmanMsg)
			var wg sync.WaitGroup

			log.Infof("Binman watch sync start of %d releases", len(bm.Releases))

			for _, rel := range bm.Releases {
				wg.Add(1)
				go goSyncRepo(rel, c, &wg)
			}

			go func(c chan BinmanMsg, wg *sync.WaitGroup) {
				wg.Wait()
				close(c)
			}(c, &wg)

			// Process results
			for msg := range c {

				if msg.err == nil {
					log.Infof("%s synced new release %s", msg.rel.Repo, msg.rel.Version)
					continue
				}

				if msg.err.Error() == "Noupdate" {
					log.Infof("%s - %s is up to date", msg.rel.Repo, msg.rel.Version)
					continue
				}

				log.Infof("Issue syncing %s - %s", msg.rel.Repo, msg.err)

				if msg.rel.cleanupOnFailure {
					err := os.RemoveAll(msg.rel.PublishPath)
					if err != nil {
						log.Debugf("Unable to clean up %s - %s", msg.rel.PublishPath, err)
					}
					log.Debugf("cleaned %s\n", msg.rel.PublishPath)
					log.Debugf("Final release data  %+v\n", msg.rel)
				}
			}

			log.Infof("Binman watch iteration complete")
			time.Sleep(time.Duration(bm.Config.Watch.Frequency) * time.Second)

			bm.metrics.Reset()
		}
	}()

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	// Block until a signal is received.
	sig := <-sigs
	log.Infof("Terimnating binman on %s", sig)
}
