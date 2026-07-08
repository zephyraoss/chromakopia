package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zephyraoss/chromakopia/internal/catalog"
	"github.com/zephyraoss/chromakopia/internal/dataset"
	"github.com/zephyraoss/chromakopia/internal/fpcalc"
	"github.com/zephyraoss/chromakopia/internal/server"
)

const metadataWatchInterval = 30 * time.Second

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

type modeSet struct {
	identify bool
	metadata bool
}

func parseMode(s string) (modeSet, error) {
	switch s {
	case "identify":
		return modeSet{identify: true}, nil
	case "metadata":
		return modeSet{metadata: true}, nil
	case "both":
		return modeSet{identify: true, metadata: true}, nil
	}
	return modeSet{}, fmt.Errorf("invalid --mode %q (want identify, metadata, or both)", s)
}

func validateFlags(mode modeSet, datasetPrefix, metadataDB, metadataURL string) error {
	if mode.identify && datasetPrefix == "" {
		return errors.New("identify mode requires --dataset (or CHROMAKOPIA_DATASET)")
	}
	if mode.metadata && metadataDB == "" {
		return errors.New("metadata mode requires --metadata-db (or CHROMAKOPIA_METADATA_DB)")
	}
	if mode.metadata && metadataURL != "" {
		return errors.New("--metadata-url is only for identify mode; metadata mode serves the catalog from --metadata-db itself")
	}
	return nil
}

func run() error {
	var (
		modeFlag = flag.String("mode", envOr("CHROMAKOPIA_MODE", "identify"),
			"serving mode: identify, metadata, or both; env CHROMAKOPIA_MODE")
		datasetPrefix = flag.String("dataset", os.Getenv("CHROMAKOPIA_DATASET"),
			"CKAF dataset prefix (path without .ckd/.cki/.ckm extension), identify mode; env CHROMAKOPIA_DATASET")
		metadataDB = flag.String("metadata-db", os.Getenv("CHROMAKOPIA_METADATA_DB"),
			"mbforge metadata database file (local path); env CHROMAKOPIA_METADATA_DB")
		metadataURL = flag.String("metadata-url", os.Getenv("CHROMAKOPIA_METADATA_URL"),
			"base URL of a chromakopia metadata-mode node to join identify metadata from; env CHROMAKOPIA_METADATA_URL")
		listen = flag.String("listen", envOr("CHROMAKOPIA_LISTEN", ":3000"),
			"listen address; env CHROMAKOPIA_LISTEN")
		fpcalcPath = flag.String("fpcalc", envOr("CHROMAKOPIA_FPCALC", "fpcalc"),
			"fpcalc executable path; env CHROMAKOPIA_FPCALC")
	)
	flag.Parse()

	mode, err := parseMode(*modeFlag)
	if err != nil {
		return err
	}
	if err := validateFlags(mode, *datasetPrefix, *metadataDB, *metadataURL); err != nil {
		return err
	}

	var store *catalog.Store
	if mode.metadata {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		store, err = catalog.Open(ctx, *metadataDB)
		cancel()
		if err != nil {
			return err
		}
		defer store.Close()
		log.Printf("metadata db opened: %s", *metadataDB)
	}

	var identify *server.Server
	if mode.identify {
		ds, err := dataset.Open(*datasetPrefix)
		if err != nil {
			return fmt.Errorf("open dataset %s: %w", *datasetPrefix, err)
		}
		defer ds.Close()
		log.Printf("dataset opened prefix=%s records=%d metadata_map=%t", *datasetPrefix, ds.RecordCount(), ds.HasMetadataMap())

		identify = &server.Server{
			Dataset: ds,
			Fpcalc:  &fpcalc.Runner{Path: *fpcalcPath},
		}

		switch {
		case *metadataURL != "":
			if *metadataDB != "" {
				log.Printf("both --metadata-url and --metadata-db set; joining via remote %s", *metadataURL)
			}
			identify.Metadata = catalog.NewClient(*metadataURL)
			log.Printf("identify metadata joined from remote node: %s", *metadataURL)
		case store != nil:
			identify.Metadata = store
		case *metadataDB != "":
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			s, err := catalog.Open(ctx, *metadataDB)
			cancel()
			if err != nil {
				log.Printf("metadata db unavailable, serving MBIDs only: %v", err)
			} else {
				defer s.Close()
				store = s
				identify.Metadata = s
				log.Printf("metadata db opened: %s", *metadataDB)
			}
		}
	}

	watchCtx, stopWatch := context.WithCancel(context.Background())
	defer stopWatch()
	if store != nil {
		go store.Watch(watchCtx, metadataWatchInterval)
		hup := make(chan os.Signal, 1)
		signal.Notify(hup, syscall.SIGHUP)
		go func() {
			for range hup {
				log.Printf("SIGHUP: reopening metadata db")
				ctx, cancel := context.WithTimeout(watchCtx, 30*time.Second)
				if err := store.Reopen(ctx); err != nil {
					log.Printf("metadata db reopen failed (keeping previous handle): %v", err)
				}
				cancel()
			}
		}()
	}

	var catalogHTTP *catalog.Store
	if mode.metadata {
		catalogHTTP = store
	}
	httpServer := &http.Server{
		Addr:              *listen,
		Handler:           server.Routes(identify, catalogHTTP),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("chromakopia listening on %s mode=%s", *listen, *modeFlag)
		errCh <- httpServer.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-stop:
		log.Printf("received %s, shutting down", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return httpServer.Shutdown(ctx)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
