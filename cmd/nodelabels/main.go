package main

import (
	"context"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/CyCoreSystems/nodelabels"
	"github.com/ericchiang/k8s"
	"github.com/pkg/errors"
)

var nodeKey = "sip"
var nodeVal = "proxy"
var filterKey = ""
var filterVal = ""

var desiredCount = 2
var checkInterval = 2 * time.Minute

func main() {
	var err error

	if os.Getenv("COUNT") != "" {
		desiredCount, err = strconv.Atoi(os.Getenv("COUNT"))
		if err != nil {
			log.Printf("failed to interpret count from COUNT=%s", os.Getenv("COUNT"))
			os.Exit(1)
		}
	}

	if os.Getenv("NODE_KEY") != "" {
		nodeKey = os.Getenv("NODE_KEY")
	}
	if os.Getenv("NODE_VAL") != "" {
		nodeVal = os.Getenv("NODE_VAL")
	}
	if os.Getenv("FILTER_KEY") != "" {
		filterKey = os.Getenv("FILTER_KEY")
	}
	if os.Getenv("FILTER_VAL") != "" {
		filterVal = os.Getenv("FILTER_VAL")
	}

	for {
		err = run(nodeKey, nodeVal, filterKey, filterVal)
		if errors.Cause(err) != io.EOF {
			log.Println("manager died:", err)
			os.Exit(1)
		}
	}
}

func run(nodeKey, nodeVal, filterKey, filterVal string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kc, err := k8s.NewInClusterClient()
	if err != nil {
		log.Println("failed to get k8s client:", err)
		os.Exit(1)
	}

	sig := make(chan struct{}, 1)

	m := nodelabels.NewFilteredManager(kc, nodeKey, nodeVal, filterKey, filterVal)

	go checker(ctx, m, sig)

	return m.Watch(ctx, sig)
}

func checker(ctx context.Context, m nodelabels.Manager, sig chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(checkInterval):
		case <-sig:
		}

		if err := m.Reconcile(ctx, desiredCount); err != nil {
			log.Println("failed to reconcile node count:", err)
		}
	}
}
