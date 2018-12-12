package config

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/juju/errors"
	//"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	APIKey string `json:"api_key"`
	SharedSecret string `json:"shared_secret"`
	ArchiveDir string `json:"archive_dir"`
	TagsetPrefix string `json:"tagsetprefix"`
	VisibilityPrefix string `json:"visibilityprefix"`
	StateFile string `json:"statefile"`
	//TagReplacements map[string]map[string]string `json:"tag_replacements"`
	//BlockedTags map[string]string `json:"blocked_tags"`
	//ConvertFiles map[string][]string `json:"convert_files"`
	//TransferService *TransferService `json:"transfer_service"`
}

//type TransferService struct {
//	Password string `json:"password"`
//	DropboxDirMapping map[string]string `json:"dropbox_dir_mapping"`
//}

func LoadTo(filepath string, target interface{}) error {
	bytes, err := ioutil.ReadFile(filepath)

	if err != nil {
		return errors.Trace(err)
	}

	err = json.Unmarshal(bytes, target)

	if err != nil {
		logrus.Debugf("Loaded config %s", filepath)
	}

	return err
}

func Load(filepath string) (*Config, error) {
	rv := new(Config)
	return rv, LoadTo(filepath, rv)
}
