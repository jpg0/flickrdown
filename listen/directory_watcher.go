package listen

import (
	"github.com/fsnotify/fsnotify"
	"github.com/jpg0/flickrup/config"
	log "github.com/Sirupsen/logrus"
	"github.com/juju/errors"
	"path/filepath"
)

func Watch(cfg *config.Config) (<-chan struct {}, error){
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.Trace(err)
	}
	c := make(chan struct{})

	us := NewUploadStatus(cfg.WatchDir)

	go func() {
		for {
			select {
			case e := <-watcher.Events:
				//log.Debugf("Detected Change:", e)
				if !us.IsStatusFile(cfg.WatchDir + string(filepath.Separator) + e.Name) {
					c <- struct{}{}
				}
			case err := <-watcher.Errors:
				log.Error("error:", err)
			}
		}
	}()

	err = watcher.Add(cfg.WatchDir)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return c, nil
}