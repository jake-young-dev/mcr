package mcr_test

import (
	"encoding/binary"
	"net"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jake-young-dev/mcr"
)

var _ = Describe("Mcr", func() {
	var (
		testingClient *mcr.Client //main testing client
		recv, serv    net.Conn    //testing server and client using net.Pipe
		testCmd       string      //command to send to test server
		wg            sync.WaitGroup
	)

	BeforeEach(func() {
		//create client and server with Pipe
		serv, recv = net.Pipe()
		//create main testing client with fake address
		testingClient = mcr.NewClient("testing", mcr.WithClient(recv))
		//arbitrary command for tests
		testCmd = "test command"
		wg = sync.WaitGroup{}
	})

	Describe("Sending packet to server", func() {
		Context("with simple command", func() {
			It("should receive the command back from server", func() {
				//create go routine to send command
				wg.Add(1)
				go func(t string) {
					res, err := testingClient.Command(t)
					//no error
					Expect(err).To(BeNil())
					//response should match command
					Expect(res).To(Equal(t))
					wg.Done()
				}(testCmd)

				//read command from testingClient
				var resHead mcr.Headers
				err := binary.Read(serv, binary.LittleEndian, &resHead)
				Expect(err).To(BeNil())

				response := make([]byte, 14)
				_, err = serv.Read(response)
				Expect(err).To(BeNil())
				//remove trailing bytes while confirming command request, these are cleaned in the client methods
				Expect(string(response[:len(response)-2])).To(Equal(testCmd))

				//create response packet, reply with command
				p, err := testingClient.CreatePacket([]byte(testCmd), resHead.Type)
				Expect(err).To(BeNil())
				_, err = serv.Write(p)
				Expect(err).To(BeNil())

				//wait for routine to wrap up
				wg.Wait()

				//close client
				err = testingClient.Close()
				serv.Close()
				recv.Close()
				Expect(err).To(BeNil())

			})
		})
	})
})
