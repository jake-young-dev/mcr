package mcr_test

import (
	"net"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jake-young-dev/mcr"
)

var _ = Describe("Mcr", func() {
	var (
		testc                    *mcr.Client
		tc, ts                   net.Conn
		testCmd                  string
		w                        sync.WaitGroup
		responseHeader, response []byte
	)

	BeforeEach(func() {
		ts, tc = net.Pipe()
		testc = mcr.NewClient("testing", mcr.WithClient(tc))
		// tests = mcr.NewClient("testing", mcr.WithClient(ts))
		testCmd = "test command"
		w = sync.WaitGroup{}
		responseHeader = make([]byte, 12)
		response = make([]byte, 250)
	})

	Describe("Sending packet to server", func() {
		Context("with simple command", func() {
			It("should receive the command", func() {

				w.Add(1)
				go func(t string) {
					res, err := testc.Command(t)
					Expect(err).To(BeNil())

					t += "\x00\x00"
					Expect(strings.TrimSpace(res)).To(Equal(t))
					w.Done()
				}(testCmd)

				// log.Println("reading")
				ts.Read(responseHeader)
				ts.Read(response)
				// log.Println(string(response))

				// time.Sleep(time.Second * 2)

				// log.Println("responding")
				ts.Write(responseHeader)
				// log.Println("1 sent")
				_, err := ts.Write(response)
				Expect(err).To(BeNil())

				// log.Println("waiting")
				w.Wait()

				err = testc.Close()
				Expect(err).To(BeNil())

			})
		})
	})
})
