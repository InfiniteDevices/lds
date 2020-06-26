package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/brocaar/lorawan"
	log "github.com/sirupsen/logrus"

	"github.com/iegomez/lds/lds"
)

type redisConf struct {
	Addr     string `toml:"addr"`
	Password string `toml:"password"`
	DB       int    `toml:"db"`
}

type tomlConfig struct {
	MQTT        mqtt           `toml:"mqtt"`
	Forwarder   forwarder      `toml:"forwarder"`
	Band        band           `toml:"band"`
	Device      device         `toml:"device"`
	GW          gateway        `toml:"gateway"`
	DR          dataRate       `toml:"data_rate"`
	RXInfo      rxInfo         `toml:"rx_info"`
	RawPayload  rawPayload     `toml:"raw_payload"`
	EncodedType []*encodedType `toml:"encoded_type"`
	LogLevel    string         `toml:"log_level"`
	RedisConf   redisConf      `toml:"redis"`
	Provisioner provisioner    `toml:"provisioner"`
}

// Configuration holders.
var (
	confFile *string
	config   *tomlConfig
)

// Configuration files loading and saving.
var (
	openFile     bool
	files        []os.FileInfo
	saveFile     bool
	saveFilename string
	mwOpen       = true
)

func importConf() {

	//When config hasn't been initialized we need to provide fresh zero instances with some defaults.
	//Decoding the conf file will override any present option.
	if config == nil {
		cMqtt := mqtt{}

		cForwarder := forwarder{}

		cGw := gateway{}

		cDev := device{
			MType: lorawan.UnconfirmedDataUp,
		}

		cBand := band{}

		cDr := dataRate{}

		cRx := rxInfo{}

		cPl := rawPayload{
			MaxExecTime: 100,
		}

		et := []*encodedType{}

		p := provisioner{}

		config = &tomlConfig{
			MQTT:        cMqtt,
			Forwarder:   cForwarder,
			Band:        cBand,
			Device:      cDev,
			GW:          cGw,
			DR:          cDr,
			RXInfo:      cRx,
			RawPayload:  cPl,
			EncodedType: et,
			Provisioner: p,
		}
	}

	if _, err := toml.DecodeFile(*confFile, &config); err != nil {
		log.Println(err)
		return
	}

	l, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(l)
	}

	//Try to set redis.
	lds.StartRedis(config.RedisConf.Addr, config.RedisConf.Password, config.RedisConf.DB)

	//Fill string representations of numeric values.
	config.DR.BitRateS = strconv.Itoa(config.DR.BitRate)
	config.RXInfo.ChannelS = strconv.Itoa(config.RXInfo.Channel)
	config.RXInfo.CrcStatusS = strconv.Itoa(config.RXInfo.CrcStatus)
	config.RXInfo.FrequencyS = strconv.Itoa(config.RXInfo.Frequency)
	config.RXInfo.LoRASNRS = strconv.FormatFloat(config.RXInfo.LoRaSNR, 'f', -1, 64)
	config.RXInfo.RfChainS = strconv.Itoa(config.RXInfo.RfChain)
	config.RXInfo.RssiS = strconv.Itoa(config.RXInfo.Rssi)

	//Set default script when it's not present.
	if config.RawPayload.Script == "" {
		config.RawPayload.Script = defaultScript
	}
}

func exportConf(filename string) {
	if !strings.Contains(filename, ".toml") {
		filename = fmt.Sprintf("%s.toml", filename)
	}
	f, err := os.Create(filename)
	if err != nil {
		log.Errorf("export error: %s", err)
		return
	}
	encoder := toml.NewEncoder(f)
	err = encoder.Encode(config)
	if err != nil {
		log.Errorf("export error: %s", err)
		return
	}
	log.Infof("exported conf file %s", f.Name())
	*confFile = f.Name()

}
