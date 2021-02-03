package rtc

import (
	"fmt"
	"testing"
	"time"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
	"mgkj/pkg/rtc/rtpengine"
	"mgkj/pkg/rtc/transport"
)

func TestRTPEngineAcceptAndRead(t *testing.T) {
	connCh, err := rtpengine.Serve(6789)
	if err != nil {
		t.Fatal("TestRTPEngineAcceptAndRead ", err)
	}

	go func() {
		for {
			select {
			case rtpTransport := <-connCh:
				fmt.Println("accept new conn from connCh", rtpTransport.RemoteAddr().String())
				go func() {
					for {
						// must read otherwise can't get new conn
						pkt, _ := rtpTransport.ReadRTP()
						fmt.Println("read rtp", pkt)
					}
				}()
			}
		}
	}()

	for i := 0; i < 1; i++ {
		rawPkt := []byte{
			0x90, 0xe0, 0x69, 0x8f, 0xd9, 0xc2, 0x93, 0xda, 0x1c, 0x64,
			0x27, 0x82, 0x00, 0x01, 0x00, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0x98, 0x36, 0xbe, 0x88, 0x9e,
		}

		rtp := &rtp.Packet{}
		rtpTransport := transport.NewOutRTPTransport("awsome", "0.0.0.0:6789")
		if err := rtp.Unmarshal(rawPkt); err == nil {
			rtpTransport.WriteRTP(rtp)
		} else {
			fmt.Println("rtpTransport.WriteRTP ", err)
		}
		time.Sleep(time.Second)
	}
}

func TestRTPEngineAcceptKCPAndRead(t *testing.T) {
	connCh, err := rtpengine.ServeWithKCP(1234, "key", "salt")
	if err != nil {
		t.Fatal("TestRTPEngineAcceptKCPAndRead ", err)
	}
	go func() {
		for {
			select {
			case rtpTransport := <-connCh:
				fmt.Println("accept new conn over kcp from connCh", rtpTransport.RemoteAddr().String())
				go func() {
					for {
						// must read otherwise can't get new conn
						pkt, _ := rtpTransport.ReadRTP()
						fmt.Println("read rtp over kcp", pkt)
					}
				}()
			}
		}
	}()

	for i := 0; i < 1; i++ {
		rawPkt := []byte{
			0x90, 0xe0, 0x69, 0x8f, 0xd9, 0xc2, 0x93, 0xda, 0x1c, 0x64,
			0x27, 0x82, 0x00, 0x01, 0x00, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0x98, 0x36, 0xbe, 0x88, 0x9e,
		}

		rtp := &rtp.Packet{}
		rtpTransport := transport.NewOutRTPTransportWithKCP("awsome", "0.0.0.0:1234", "key", "salt")
		if err := rtp.Unmarshal(rawPkt); err == nil {
			rtpTransport.WriteRTP(rtp)
		} else {
			fmt.Println("rtpTransport.WriteRTP ", err)
		}
		time.Sleep(time.Second)
	}
}

func TestWebRTCTransportP2P(t *testing.T) {
	options := make(map[string]interface{})
	options["codec"] = "vp8"

	// new pub
	pub := transport.NewWebRTCTransport("pub", options)
	if pub == nil {
		t.Fatal("pub == nil")
	}

	// pub add track
	_, err := pub.AddTrack(476325762, webrtc.DefaultPayloadTypeVP8, "video", "pion")
	if err != nil {
		t.Fatalf("pub.AddTrack err=%v", err)
	}

	// pub create offer
	offer, err := pub.Offer()
	if err != nil {
		t.Fatalf("pub.Offer err=%v", err)
	}

	// new sub
	sub := transport.NewWebRTCTransport("sub", options)

	// sub answer offer
	options = make(map[string]interface{})
	options["publish"] = "true"
	answer, err := sub.Answer(offer, options)
	if err != nil {
		t.Fatalf("err=%v answer=%v", err, answer)
	}

	// pub set remote sdp
	err = pub.SetRemoteSDP(answer)
	if err != nil {
		t.Fatalf("err=%v", err)
	}

	go func() {
		for {
			rtp, err := sub.ReadRTP()
			fmt.Printf("rtp=%v err=%v\n", rtp, err)
		}
	}()

	count := 0
	for {
		if count > 1 {
			return
		}
		rawPkt := []byte{
			0x90, 0xe0, 0x69, 0x8f, 0xd9, 0xc2, 0x93, 0xda, 0x1c, 0x64,
			0x27, 0x82, 0x00, 0x01, 0x00, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0x98, 0x36, 0xbe, 0x88, 0x9e,
		}

		rtp := &rtp.Packet{}
		err := rtp.Unmarshal(rawPkt)
		if err != nil {
			t.Fatal("rtp.Unmarshal", err)
		}
		err = pub.WriteRTP(rtp)
		if err != nil {
			t.Fatal("pub.WriteRTP ", err)
		}
		time.Sleep(time.Second)
		count++
	}
}
