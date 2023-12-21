package binman

import (
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	db "github.com/rjbrown57/binman/pkg/db"
	"github.com/rjbrown57/binman/pkg/downloader"
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
func StartWatch(config string) {

	// Watch mode always uses json style logging
	log.ConfigureLog(true, 0)

	var downloadChan = make(chan downloader.DlMsg)
	var dwg sync.WaitGroup

	dbOptions := db.DbConfig{
		Dwg:    &dwg,
		DbChan: make(chan db.DbMsg),
	}

	if checkNewDb("") {
		log.Debugf("Initializing DB")
		populateDB(dbOptions, config)
	}

	go db.RunDB(dbOptions)

	// create the base config
	watchConfig := NewGHBMConfig(SetBaseConfig(config))
	watchConfig.setWatchConfig(dbOptions.Dwg, dbOptions.DbChan)

	// Start watch mode http
	go watchServe(watchConfig.Config.Watch, watchConfig.Config.ReleasePath)

	// start download workers
	log.Debugf("launching %d download workers", watchConfig.Config.NumWorkers)
	for worker := 1; worker <= watchConfig.Config.NumWorkers; worker++ {
		go downloader.GetDownloader(downloadChan, worker)
	}

	log.Debugf("watch config = %+v", watchConfig.Config.Watch)

	go getSpinner(true)

	go func() {
		for {

			c := make(chan BinmanMsg)
			output := make(map[string][]BinmanMsg)
			var wg sync.WaitGroup

			log.Infof("Binman watch sync start of %d releases", len(watchConfig.Releases))

			for _, rel := range watchConfig.Releases {
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
					output["Synced"] = append(output["Synced"], msg)
					continue
				}

				if msg.err.Error() == "Noupdate" {
					output["Up to Date"] = append(output["Up to Date"], msg)
					continue
				}

				output["Error"] = append(output["Error"], msg)
				if msg.rel.cleanupOnFailure {
					err := os.RemoveAll(msg.rel.publishPath)
					if err != nil {
						log.Debugf("Unable to clean up %s - %s", msg.rel.publishPath, err)
					}
					log.Debugf("cleaned %s\n", msg.rel.publishPath)
					log.Debugf("Final release data  %+v\n", msg.rel)
				}
			}

			log.Infof("Binman watch iteration complete")
			time.Sleep(time.Duration(watchConfig.Config.Watch.Frequency) * time.Second)

			watchConfig.metrics.Reset()
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
