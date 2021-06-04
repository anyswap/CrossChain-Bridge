package worker

import (
	"os"
	"strings"

	"github.com/anyswap/CrossChain-Bridge/cmd/utils"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/fsnotify/fsnotify"
)

// AddTokenPairDynamically add token pair dynamically
func AddTokenPairDynamically() {
	pairsDir := tokens.GetTokenPairsDir()
	if pairsDir == "" {
		log.Warn("token pairs dir is empty")
		return
	}

	watch, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error("fsnotify.NewWatcher failed", "err", err)
		return
	}

	err = watch.Add(pairsDir)
	if err != nil {
		log.Error("watch.Add token pairs dir failed", "err", err)
		return
	}

	utils.TopWaitGroup.Add(1)
	go startWatcher(watch)
}

func startWatcher(watch *fsnotify.Watcher) {
	log.Info("start fsnotify watch")
	defer func() {
		log.Info("stop fsnotify watch")
		_ = watch.Close()
		utils.TopWaitGroup.Done()
	}()

	ops := []fsnotify.Op{
		fsnotify.Create,
		fsnotify.Write,
	}

	for {
		select {
		case <-utils.CleanupChan:
			return
		case ev, ok := <-watch.Events:
			if !ok {
				continue
			}
			log.Trace("fsnotify watch event", "event", ev)
			for _, op := range ops {
				if ev.Op&op == op {
					err := addTokenPair(ev.Name)
					if err != nil {
						log.Info("addTokenPair error", "configFile", ev.Name, "err", err)
					}
					break
				}
			}
		case werr, ok := <-watch.Errors:
			if !ok {
				continue
			}
			log.Warn("fsnotify watch error", "err", werr)
		}
	}
}

func addTokenPair(fileName string) error {
	if !strings.HasSuffix(fileName, ".toml") {
		return nil
	}
	fileStat, _ := os.Stat(fileName)
	// ignore if file is not exist, or is directory, or is empty file
	if fileStat == nil || fileStat.IsDir() || fileStat.Size() == 0 {
		return nil
	}
	pairConfig, err := tokens.AddPairConfig(fileName)
	if err != nil {
		return err
	}
	log.Info("addTokenPair success", "configFile", fileName, "pairID", pairConfig.PairID)
	return nil
}
