package mcr

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

/*
Listen, this file really got away from me. With the goal of high code coverage this file has turned into an abomination, I tried
to group tests with each other but I have entirely lost track at this point. These tests should be rewritten to use a TestingController
that will call nested tTestFunction funcs while handling proper setup/teardown using setup() and teardown() functions. This would allow test
scaffolding to only be written once instead of in every test function. When that rewrite will come, who knows. I will continue to add tests
until it drives me crazy enough to rewrite.
*/

// command to generate code coverage and run tests
// go test -cover -coverprofile coverage.out
// open coverage file in browser
// go tool cover -html coverage.out

// test creating a new client with default options
func TestNewClientDefaults(t *testing.T) {
	tc := NewClient("test")

	if tc.Address() != "test" {
		t.Fatal("address does not match on creation")
	}

	if tc.Cap() != DefaultCap {
		t.Fatal("default cap not set correctly")
	}

	if tc.Port() != DefaultPort {
		t.Fatal("default port not set")
	}

	if tc.RequestID() != ResetID {
		t.Fatal("default request id not set")
	}

	if tc.Timeout() != DefaultTimeout {
		t.Fatal("default timeout not set")
	}
}

// testing sending command without connecting
func TestCommandError(t *testing.T) {
	tc := NewClient("test")
	_, err := tc.Command("hi")
	if !errors.Is(err, ErrClientNotConnected) {
		t.Fatal("proper connection handling failing on command send")
	}
}

func TestCommandNoResError(t *testing.T) {
	tc := NewClient("test")
	err := tc.CommandNoResponse("hi")
	if !errors.Is(err, ErrClientNotConnected) {
		t.Fatal("proper connection handling failing on command send, no response")
	}
}

// test case creating a new client and mock connection and sending a command using the Command
// method. Command value is sent in server reply to confirm data integrity
func TestRemoteCommand(t *testing.T) {
	var (
		testingClient Client   //main testing client
		recv, serv    net.Conn //testing server and client using net.Pipe
		testCmd       string   //command to send to test server
		wg            sync.WaitGroup
	)

	//create client and server with Pipe
	serv, recv = net.Pipe()
	//create main testing client with fake address
	testingClient = NewClient("testing", WithConnection(recv))
	//arbitrary command for tests
	testCmd = "test command"
	wg = sync.WaitGroup{}

	//create go routine to send command
	wg.Add(1)
	ec := make(chan error)
	go func(testing string) {
		res, err := testingClient.Command(testing)
		if err != nil {
			ec <- err
			wg.Done()
			return
		}
		//response should match command
		if !strings.EqualFold(res, testing) {
			ec <- errors.New("response from server does not match client request")
			wg.Done()
			return
		}
		ec <- nil
		wg.Done()
	}(testCmd)

	//read command from testingClient
	var resHead headers
	err := binary.Read(serv, binary.LittleEndian, &resHead)
	if err != nil {
		t.Fatal(err)
	}

	response := make([]byte, 14)
	_, err = serv.Read(response)
	if err != nil {
		t.Fatal(err)
	}
	//remove trailing bytes while confirming command request, these are cleaned in the client methods
	if !strings.EqualFold(string(response[:len(response)-2]), testCmd) {
		t.Fatal("the server did not recieve the matching command sent from client")
	}

	//create response packet, reply with command
	p, err := testingClient.createPacket([]byte(testCmd), resHead.Type)
	if err != nil {
		t.Fatal(err)
	}
	_, err = serv.Write(p)
	if err != nil {
		t.Fatal(err)
	}

	//wait for routine to wrap up
	check := <-ec
	wg.Wait()
	close(ec)
	if check != nil {
		t.Fatal(check)
	}

	//close client
	err = testingClient.Close()
	if err != nil {
		t.Fatal(err)
	}

	serv.Close()
	recv.Close()
}

func TestConnectOverflow(t *testing.T) {
	var (
		testingClient Client   //main testing client
		recv, serv    net.Conn //testing server and client using net.Pipe
	)

	//create client and server with Pipe
	serv, recv = net.Pipe()
	//create main testing client with fake address
	testingClient = NewClient("testing", WithConnection(recv))

	overflow := make([]byte, math.MaxInt32)
	err := testingClient.authenticate(overflow)
	if err != ErrIntOverflow {
		t.Fatal("password authentication integer overflow")
	}

	//close client
	err = testingClient.Close()
	if err != nil {
		t.Fatal(err)
	}

	serv.Close()
	recv.Close()
}

// testing sending a command to the server without waiting for a response
func TestRemoteCommandNoResponse(t *testing.T) {
	var (
		testingClient Client   //main testing client
		recv, serv    net.Conn //testing server and client using net.Pipe
		testCmd       string   //command to send to test server
		wg            sync.WaitGroup
	)

	//create client and server with Pipe
	serv, recv = net.Pipe()
	//create main testing client with fake address
	testingClient = NewClient("testing", WithConnection(recv))
	//arbitrary command for tests
	testCmd = "test command"
	wg = sync.WaitGroup{}

	//create go routine to send command
	wg.Add(1)
	ec := make(chan error)
	go func(testing string) {
		err := testingClient.CommandNoResponse(testing)
		if err != nil {
			ec <- err
			wg.Done()
			return
		}

		ec <- nil
		wg.Done()
	}(testCmd)

	//read command from testingClient
	var resHead headers
	err := binary.Read(serv, binary.LittleEndian, &resHead)
	if err != nil {
		t.Fatal(err)
	}

	response := make([]byte, 14)
	_, err = serv.Read(response)
	if err != nil {
		t.Fatal(err)
	}
	//remove trailing bytes while confirming command request, these are cleaned in the client methods
	if !strings.EqualFold(string(response[:len(response)-2]), testCmd) {
		t.Fatal("the server did not recieve the matching command sent from client")
	}

	//wait for routine to wrap up
	check := <-ec
	wg.Wait()
	close(ec)
	if check != nil {
		t.Fatal(check)
	}

	//close client
	err = testingClient.Close()
	if err != nil {
		t.Fatal(err)
	}

	serv.Close()
	recv.Close()
}

// testing the requestID handling ensuring it is reset once it overflows the cap
func TestRequestIDReset(t *testing.T) {
	testingClient := NewClient("testing")
	testingClient.SetRequestID(DefaultCap)
	testingClient.incrementRequestID()
	if testingClient.RequestID() != ResetID {
		t.Fatal("request id did not properly reset")
	}
	//close client
	err := testingClient.Close()
	if err != nil {
		t.Fatal(err)
	}
}

// testing the WithCap option
func TestCapOption(t *testing.T) {
	testingClient := NewClient("testing", WithCap(20))
	testingClient.SetRequestID(20)
	testingClient.incrementRequestID()
	if testingClient.RequestID() != 1 {
		t.Fatal("custom request id did not properly reset")
	}
	//close client
	err := testingClient.Close()
	if err != nil {
		t.Fatal(err)
	}
}

// testing cap getter/setters
func TestReqIDGetSet(t *testing.T) {
	tc := NewClient("testing")
	tc.SetRequestID(66)
	if tc.RequestID() != 66 {
		t.Fatal("request id getter/setter values do not match")
	}
}

// testing implmenting a custom timeout for the client
func TestTimeoutOption(t *testing.T) {
	testingClient := NewClient("testing", WithTimeout(time.Second*5))
	if testingClient.Timeout() != time.Second*5 {
		t.Fatal("timeout value did not update when supplying the timeout")
	}
	//close client
	err := testingClient.Close()
	if err != nil {
		t.Fatal(err)
	}
}

// testing using a different port
func TestPortOption(t *testing.T) {
	testPort := 9876
	tc := NewClient("test", WithPort(testPort))
	if tc.Port() != testPort {
		t.Fatal("ports did not match")
	}
}

// testing using custom connection
func TestConnectionOption(t *testing.T) {
	srv, _ := net.Pipe()
	tc := NewClient("test", WithConnection(srv))
	if tc.Connection() != srv {
		t.Fatal("connection was not updated")
	}
}

// testing int conversions
func TestIntConversionFail(t *testing.T) {
	tv := math.MaxInt32 + 1
	tc := NewClient("test")
	_, err := tc.safeIntConversion(tv)
	if err == nil {
		t.Fatal("integer overflow")
	}
}

// testing address getter
func TestAddrGetter(t *testing.T) {
	mock := "test"
	tc := NewClient(mock)

	if tc.Address() != mock {
		t.Fatal("address value does not match getter response")
	}
}

// testing timeout getter/setter
func TestTimeoutGetSet(t *testing.T) {
	tc := NewClient("test")
	to := time.Second * 30
	tc.SetTimeout(to)

	if tc.Timeout() != to {
		t.Fatal("timeout setter not matching getter value")
	}
}

// testing cap getter/setter
func TestCapGetSet(t *testing.T) {
	tc := NewClient("test")
	tp := int32(66)

	tc.SetCap(tp)

	if tc.Cap() != tp {
		t.Fatal("cap setter not matching getter value")
	}
}

// testing overflowing packet size
func TestCreatePacketTooBig(t *testing.T) {
	tc := NewClient("test")

	d := make([]byte, math.MaxInt32)

	_, err := tc.createPacket(d, CommandPacket)
	if !errors.Is(err, ErrIntOverflow) {
		t.Fatal("integer overflow allowed")
	}
}

func TestAuthenticatePacketTooBig(t *testing.T) {
	tc := NewClient("test")
	d := make([]byte, math.MaxInt32)

	err := tc.authenticate(d)
	if !errors.Is(err, ErrIntOverflow) {
		t.Fatal("integer overflow allowed")
	}
}

func TestSendCommandOverflow(t *testing.T) {
	tc := NewClient("test", WithConnection(&net.TCPConn{}))
	d := make([]byte, math.MaxInt32)

	_, err := tc.Command(string(d))
	if !errors.Is(err, ErrIntOverflow) {
		t.Fatal("integer overflow allowed")
	}
}

func TestSendCommandWriteFail(t *testing.T) {
	var (
		testingClient Client   //main testing client
		recv, serv    net.Conn //testing server and client using net.Pipe
	)

	//create client and server with Pipe
	serv, recv = net.Pipe()
	//create main testing client with fake address
	testingClient = NewClient("testing", WithConnection(recv))

	//close pipe to force error
	recv.Close()

	err := testingClient.send([]byte("hi"))
	if err != io.ErrClosedPipe {
		t.Fatal(err)
	}

	//close client
	err = testingClient.Close()
	if err != nil {
		t.Fatal(err)
	}

	serv.Close()
}

func TestSendAndRcvCommandWriteFail(t *testing.T) {
	var (
		testingClient Client   //main testing client
		recv, serv    net.Conn //testing server and client using net.Pipe
	)

	//create client and server with Pipe
	serv, recv = net.Pipe()
	//create main testing client with fake address
	testingClient = NewClient("testing", WithConnection(recv))

	//close pipe to force error
	recv.Close()

	_, err := testingClient.sendAndRecv([]byte("hi"))
	if err != io.ErrClosedPipe {
		t.Fatal(err)
	}

	//close client
	err = testingClient.Close()
	if err != nil {
		t.Fatal(err)
	}

	serv.Close()
}

func TestSendNoResOverflow(t *testing.T) {
	tc := NewClient("test", WithConnection(&net.TCPConn{}))
	d := make([]byte, math.MaxInt32)

	err := tc.CommandNoResponse(string(d))
	if !errors.Is(err, ErrIntOverflow) {
		t.Fatal("integer overflow allowed")
	}
}

// testing authentication using the Connect method
func TestAuthentication(t *testing.T) {
	var (
		testingClient Client   //main testing client
		recv, serv    net.Conn //testing server and client using net.Pipe
		testPwd       string   //command to send to test server
		wg            sync.WaitGroup
	)

	//create client and server
	serv, recv = net.Pipe()
	//create main testing client with fake address
	testingClient = NewClient("testing", WithConnection(recv))
	//fake testing password
	testPwd = "password"
	wg = sync.WaitGroup{}

	//create go routine to send command
	wg.Add(1)
	ec := make(chan error)
	go func(tp string) {
		err := testingClient.Connect(tp)
		if err != nil {
			ec <- err
			wg.Done()
			return
		}
		ec <- nil
		wg.Done()
	}(testPwd)

	//read command from testingClient
	var resHead headers
	err := binary.Read(serv, binary.LittleEndian, &resHead)
	if err != nil {
		t.Fatal(err)
	}

	response := make([]byte, 10)
	_, err = serv.Read(response)
	if err != nil {
		t.Fatal(err)
	}
	//remove trailing bytes while confirming command request, these are cleaned in the client methods
	if !strings.EqualFold(string(response[:len(response)-2]), testPwd) {
		t.Fatal("the server did not recieve the matching command sent from client")
	}

	//create response packet, reply with command
	p, err := testingClient.createPacket([]byte(testPwd), 2) //hardcode auth response
	if err != nil {
		t.Fatal(err)
	}
	_, err = serv.Write(p)
	if err != nil {
		t.Fatal(err)
	}

	//wait for routine to wrap up
	check := <-ec
	wg.Wait()
	close(ec)
	if check != nil {
		t.Fatal(check)
	}

	//close client
	err = testingClient.Close()
	if err != nil {
		t.Fatal(err)
	}

	serv.Close()
	recv.Close()
}
