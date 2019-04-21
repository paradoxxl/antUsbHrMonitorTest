package main

import (
	"bufio"
	"context"
	"flag"
	"github.com/google/gousb"
	"github.com/half2me/antgo/message"
	"log"
	"os"
	"strconv"
)

var (
	argPid = flag.String("pid", "0x1009", "the pid of the ant+ usb stick")
	argVid = flag.String("vid", "0x0fcf", "the vd of the ant+ usb stick")
)

var (
	pid gousb.ID
	vid gousb.ID
)

func main() {
	flag.Parse()

	iPid, err := strconv.ParseInt(*argPid, 0, 16)
	if err != nil {
		log.Fatal(err)
	}
	iVid, err := strconv.ParseInt(*argVid, 0, 16)
	if err != nil {
		log.Fatal(err)
	}

	pid = gousb.ID(iPid)
	vid = gousb.ID(iVid)

	ctx := gousb.NewContext()
	ctx.Debug(99)
	defer ctx.Close()

	dev, err := ctx.OpenDeviceWithVIDPID(vid, pid)

	if err != nil {
		log.Fatal("oen device ", err)
	}
	if dev == nil {
		log.Fatal("device is nil")
	}
	defer dev.Close()

	log.Print("Enabling autodetach")
	dev.SetAutoDetach(true)

	log.Printf("Setting configuration %d...", 1)
	cfg, err := dev.Config(1)
	if err != nil {
		log.Fatalf("dev.Config(%d): %v", 1, err)
	}

	log.Printf("Claiming interface %d (alt setting %d)...", 0, 0)
	intf, err := cfg.Interface(0, 0)
	if err != nil {
		log.Fatalf("cfg.Interface(%d, %d): %v", 0, 0, err)
	}

	log.Printf("Using in-endpoint %d...", 1)
	inep, err := intf.InEndpoint(1)
	if err != nil {
		log.Fatalf("dev.InEndpoint(): %s", err)
	}

	log.Println(inep.String())

	log.Printf("Using out-endpoint %d...", 1)
	outep, err := intf.OutEndpoint(1)
	if err != nil {
		log.Fatalf("dev.InEndpoint(): %s", err)
	}

	log.Println(outep.String())

	readstr, err := inep.NewStream(64, 1)
	if err != nil {
		log.Fatal(err)
	}
	defer readstr.Close()

	opCtx := context.Background()
	buf := make([]byte, 64)
	s := make(chan interface{})

	go func() {
		for {
			select {
			case <-s:
				return
			default:
				n, err := readstr.ReadContext(opCtx, buf)
				if err != nil {
					log.Fatal(err)
				}

				if n < 12 {
					continue
				}

				if buf[0] == message.MESSAGE_TX_SYNC {
					if buf[2] == message.MESSAGE_TYPE_BROADCAST {
						eventTime := (uint16(buf[9]) << 8) | uint16(buf[8])
						hbCount := buf[10]
						hr := buf[11]

						if buf[1] > 8 && buf[12] == 0xe0 {
							log.Printf("HR: %v\tBeatCount: %v\teventTime: %v [%2x %2x]\tflag %2x", hr, hbCount, eventTime, buf[9], buf[8], buf[12])
						} else {
							log.Printf("HR: %v\tBeatCount: %v\teventTime: %v [%2x %2x]", hr, hbCount, eventTime, buf[9], buf[8])

						}

					}
				}
				//log.Println("Read: ", buf[:n])
			}
		}
	}()

	startRxScanMode(outep)

	r := bufio.NewReader(os.Stdin)
	r.ReadLine()
	s <- true
}

func startRxScanMode(ep *gousb.OutEndpoint) {
	if _, err := ep.Write(message.SystemResetMessage()); err != nil {
		log.Fatal("startRxScanMode - SystemResetMessage ", err)
	}
	if _, err := ep.Write(message.SetNetworkKeyMessage(0, []byte(message.ANTPLUS_NETWORK_KEY))); err != nil {
		log.Fatal("startRxScanMode - SetNetworkKeyMessage ", err)
	}
	if _, err := ep.Write(message.AssignChannelMessage(0, message.CHANNEL_TYPE_ONEWAY_RECEIVE)); err != nil {
		log.Fatal("startRxScanMode - AssignChannelMessage ", err)

	}
	if _, err := ep.Write(message.SetChannelIdMessage(0)); err != nil {
		log.Fatal("startRxScanMode - SetChannelIdMessage ", err)

	}
	if _, err := ep.Write(message.SetChannelRfFrequencyMessage(0, 2457)); err != nil {
		log.Fatal("startRxScanMode - SetChannelRfFrequencyMessage ", err)

	}
	if _, err := ep.Write(message.EnableExtendedMessagesMessage(true)); err != nil {
		log.Fatal("startRxScanMode - EnableExtendedMessagesMessage ", err)

	}
	if _, err := ep.Write(message.LibConfigMessage(true, true, true)); err != nil {
		log.Fatal("startRxScanMode - LibConfigMessage ", err)

	}
	if _, err := ep.Write(message.OpenRxScanModeMessage()); err != nil {
		log.Fatal("startRxScanMode - OpenRxScanModeMessage ", err)

	}
}
