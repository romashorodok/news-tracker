package prebuiltemplate

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
)

type ConfigFlag []NewsFeedConfig

func (c *ConfigFlag) Set(arg string) error {
	var config NewsFeedConfig
	if err := json.Unmarshal([]byte(arg), &config); err != nil {
		return errors.Join(errors.New("unable deserialize config."), err)
	}
	*c = append(*c, config)
	return nil
}

func (c *ConfigFlag) String() string {
	return fmt.Sprint(*c)
}

var _ flag.Value = (*ConfigFlag)(nil)
