package mcr

import (
	"encoding/binary"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// run with "go test -cover -v" to show coverage and to list the tests ran

// test case creating a new client and mock connection and sending a command using the Command
// method. Command value is sent in server reply to confirm data integrity
func TestRemoteCommand(t *testing.T) {
	var (
		testingClient *Client  //main testing client
		recv, serv    net.Conn //testing server and client using net.Pipe
		testCmd       string   //command to send to test server
		wg            sync.WaitGroup
	)

	//create client and server with Pipe
	serv, recv = net.Pipe()
	//create main testing client with fake address
	testingClient = NewClient("testing")
	testingClient.connection = recv //use mock connector
	//arbitrary command for tests
	testCmd = "test command"
	wg = sync.WaitGroup{}

	//create go routine to send command
	wg.Add(1)
	ec := make(chan error)
	go func(testing string) {
		res, err := testingClient.Command(testing)
		//no error
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

// testing the requestID handling ensuring it is reset once it overflows the cap
func TestRequestIDReset(t *testing.T) {
	testingClient := NewClient("testing")
	testingClient.requestID = testingClient.cap
	testingClient.incrementRequestID()
	if testingClient.requestID != ResetID {
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
	testingClient.requestID = testingClient.cap
	testingClient.incrementRequestID()
	if testingClient.requestID != ResetID {
		t.Fatal("custom request id did not properly reset")
	}
	//close client
	err := testingClient.Close()
	if err != nil {
		t.Fatal(err)
	}
}

// testing implmenting a custom timeout for the client
func TestTimeoutOption(t *testing.T) {
	testingClient := NewClient("testing", WithTimeout(time.Second*5))
	if testingClient.timeout != time.Second*5 {
		t.Fatal("timeout value did not update when supplying the timeout")
	}
	//close client
	err := testingClient.Close()
	if err != nil {
		t.Fatal(err)
	}
}

// testing authentication using the Connect method
func TestAuthenticationUsingConnect(t *testing.T) {
	var (
		testingClient *Client  //main testing client
		recv, serv    net.Conn //testing server and client using net.Pipe
		testPwd       string   //command to send to test server
		wg            sync.WaitGroup
	)

	//create client and server with Pipe
	serv, recv = net.Pipe()
	//create main testing client with fake address
	testingClient = NewClient("testing")
	testingClient.connection = recv //use mock connector
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
