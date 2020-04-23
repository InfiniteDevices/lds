package lds

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/brocaar/chirpstack-api/go/gw"
	"github.com/golang/protobuf/ptypes/duration"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/evio"
)

// NSClient is a raw UDP client
type NSClient struct {
	Server string
	Port   int

	connected bool
	udpEvents evio.Events
}

type pfpacket struct {
	Time string  `json:"time"`
	TMMS uint64  `json:"tmms"`
	TMST uint32  `json:"tmst"`
	Chan uint32  `json:"chan"`
	RFCH uint32  `json:"rfch"`
	Freq float32 `json:"freq"`
	Stat int32   `json:"stat"`
	Modu string  `json:"modu"`
	DatR string  `json:"datr"`
	CorR string  `json:"codr"`
	RSSI int32   `json:"rssi"`
	LSNR float64 `json:"lsnr"`
	Size uint32  `json:"size"`
	Data string  `json:"data"`
}

type pfproto struct {
	RXPK []pfpacket `json:"rxpk"`
}

// IsConnected checks if listening for incoming UDP
func (client *NSClient) IsConnected() bool {
	return client.connected
}

type udpPacketCallback func(payload []byte) error

// Connect starts listening incoming UDP
func (client *NSClient) Connect(onReceive udpPacketCallback) error {

	client.udpEvents.Data = func(c evio.Conn, in []byte) (out []byte, action evio.Action) {
		onReceive(in)
		out = nil
		return
	}

	bindpoint := fmt.Sprintf("udp://0.0.0.0:%d", client.Port)
	log.Infoln("UDP listening bindpoint=", bindpoint)
	go evio.Serve(client.udpEvents, bindpoint)

	client.connected = true
	return nil
}

func (client *NSClient) send(bytes []byte) error {
	ip := net.ParseIP(client.Server)

	if ip == nil {
		return errors.New("bad network server IP")
	}

	addr := net.UDPAddr{
		IP:   ip,
		Port: client.Port,
	}

	conn, err := net.DialUDP("udp", nil, &addr)

	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(bytes)
	return err
}

func toMilliseconds(d *duration.Duration) uint64 {
	return uint64(d.Seconds)*1000 + uint64(d.Nanos)/1000
}

func (client *NSClient) sendWithPayload(payload []byte, gwMAC string, rxInfo *gw.UplinkRXInfo, txInfo *gw.UplinkTXInfo) error {

	phyBase := base64.StdEncoding.EncodeToString(payload)

	gps := rxInfo.GetTimeSinceGpsEpoch()
	utc := time.Now().Format(time.RFC3339)
	mod := txInfo.GetLoraModulationInfo()

	packet := pfpacket{}
	packet.Time = utc
	packet.TMMS = toMilliseconds(gps) / 1000
	packet.TMST = uint32(toMilliseconds(gps) / 1000 / 1000)
	packet.Chan = rxInfo.GetChannel()
	packet.RFCH = rxInfo.GetRfChain()
	packet.Freq = float32(txInfo.GetFrequency()) / 1000000.0
	packet.Stat = 1
	packet.Modu = "LORA"
	packet.DatR = fmt.Sprintf("SF%dBW%d", mod.SpreadingFactor, mod.GetBandwidth())
	packet.CorR = mod.GetCodeRate()
	packet.RSSI = rxInfo.GetRssi()
	packet.LSNR = rxInfo.GetLoraSnr()
	packet.Size = uint32(len(payload))
	packet.Data = phyBase

	proto := pfproto{RXPK: []pfpacket{packet}}

	packetJSON, err := json.Marshal(proto)

	log.Debugf("Marshalled upstream JSON %s", packetJSON)

	if err != nil {
		return err
	}

	version := byte(0x02)
	token := rand.Int()
	tokenlsb := byte(token & 0x00FF)
	tokenmsb := byte((token & 0xFF00) >> 8)
	id := byte(0x00)
	header := []byte{version, tokenmsb, tokenlsb, id}

	gwbytes, err := hex.DecodeString(gwMAC)

	if err != nil {
		return err
	}

	jsonbytes := []byte(packetJSON)
	datagram := bytes.Join([][]byte{header, gwbytes, jsonbytes}, []byte{})

	client.send(datagram)

	return nil
}

// UDPParsePacket extract metadata and physial payload from a packet
func UDPParsePacket(packet []byte, result *map[string]interface{}) (bool, error) {
	var data struct {
		version int8
		token   int16
		id      int8
	}

	buf := bytes.NewReader(packet)
	err := binary.Read(buf, binary.LittleEndian, &data)

	if err != nil {
		return false, err
	}

	// PULL_RESP == 0x03
	if data.id != 0x03 {
		return false, nil
	}

	jsonBytes := packet[4:]
	jsonString := string(jsonBytes)
	fmt.Printf("Incoming JSON %s", jsonString)
	*result = make(map[string]interface{})

	return false, nil
}
