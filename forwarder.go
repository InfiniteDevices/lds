package main

import (
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/iegomez/lds/lds"

    "gioui.org/layout"
    "gioui.org/widget/material"
)

// cNSClient is a direct NetworkServer connection handle
var cNSClient lds.NSClient

type forwarder struct {
	Server string `toml:"nserver"`
	Port   string `toml:"nsport"`
}

func forwarderForm(gtx *layout.Context, th *material.Theme) layout.FlexChild {
	return layout.Rigid( func() {
		th.Caption("forwarder").Layout(gtx)
	})
}

func beginForwarderForm() {
/*! imgui.Begin("Forwarder")
	imgui.Separator()
	imgui.PushItemWidth(250.0)
	imgui.InputText("Network Server", &config.Forwarder.Server)
	imgui.InputText("UDP Port", &config.Forwarder.Port)

	if mqttClient == nil || !mqttClient.IsConnected() {
		if !cNSClient.IsConnected() {
			if imgui.Button("Connect") {
				forwarderConnect()
			}
		} else {
			imgui.Text("UDP Listening")
		}
	}
	//Add popus for file administration.
	beginOpenFile()
	beginSaveFile()
	imgui.End()*/
}

func forwarderConnect() error {
	port, err := strconv.Atoi(config.Forwarder.Port)

	if err != nil {
		log.Warn("network server UDP port must be a number")
		return err
	}

	cNSClient.Server = config.Forwarder.Server
	cNSClient.Port = port
	cNSClient.Connect(config.GW.MAC, onIncomingDownlink)
	log.Infoln("UDP Forwarder started (MQTT disabled)")

	return nil
}
