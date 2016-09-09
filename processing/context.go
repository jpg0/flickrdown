package processing

import (
	"github.com/jpg0/flickrup/config"
	"time"
)

type ProcessingContext struct {
	File TaggedFile
	Visibilty string
	Config *config.Config
	ArchiveSubdir string
	UploadedId string
	OverrideDateTaken time.Time
}

func (pc ProcessingContext) DateTaken() time.Time {
	if pc.OverrideDateTaken.IsZero() {
		return pc.File.RealDateTaken()
	} else {
		return pc.OverrideDateTaken
	}
}

func NewProcessingContext(config *config.Config, file TaggedFile) *ProcessingContext {
	return &ProcessingContext{
		Visibilty: "public",
		Config: config,
		File: file,
	}
}