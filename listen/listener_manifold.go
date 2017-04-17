package listen

import log "github.com/Sirupsen/logrus"

type Listener struct {
	processing bool
	queued     bool
	begin      chan BeginEvent
}

type Update uint32

type BeginEvent struct {
	AfterPause bool //else as deferred
}

const (
	Triggered Update = 1 << iota
	Requested
	ProcessingComplete
)

func NewListener(triggers <-chan struct{}, completions <-chan struct{}) *Listener {

	begin := make(chan BeginEvent, 1)

	l := &Listener{begin:begin}

	go func() {
		defer close(l.begin)
		for {
			select {
			case <-completions:
				l.triggered(ProcessingComplete)
			case <-triggers:
				log.Info("Change detected")
				l.triggered(Triggered)
			}
		}
	}()

	return l
}

func (l *Listener) BeginChannel() <-chan BeginEvent {
	return l.begin
}

func (l *Listener) TriggerNow() {
	l.triggered(Requested)
}

func (l *Listener) Queue() {
	l.triggered(Triggered)
}

//single threaded
func (l *Listener) triggered(u Update) {
	switch u {
	case Triggered, Requested:
		if l.processing {
			log.Infof("Processing queued")
			l.queued = true
		} else {
			log.Infof("Processing triggered")
			l.processing = true
			l.begin <- BeginEvent{u == Triggered}
		}
	case ProcessingComplete:
		if l.processing {
			l.processing = false
			log.Infof("Processing complete")
			if l.queued {
				log.Infof("Queued processing triggered")
				l.queued = false
				l.processing = true
				l.begin <- BeginEvent{false}
			}
		} else {
			log.Errorf("Not marked as processing at completion of processing")
		}
	}
}