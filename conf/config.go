package conf

import (
	"bytes"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Username string `toml:"Username"`
	Password string `toml:"Password"`
	GateAddr string `toml:"GateAddr"`
}

func ConfOfB1() (userName, password, gateAddr string) {
	file, err := os.Open("./config.toml")
	defer file.Close()
	if err != nil {
		panic(err)
	}

	var conf Config

	buf := bytes.NewBufferString("")
	_, err = buf.ReadFrom(file)
	if err != nil {
		panic(err)
	}

	_, err = toml.Decode(buf.String(), &conf)
	if err != nil {
		panic(err)
	}

	return conf.Username, conf.Password, conf.GateAddr
}
