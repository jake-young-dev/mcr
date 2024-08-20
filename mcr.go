package mcr

/*
Remote console client used to interact with Minecraft servers, the byte order for minecraft rcon is little
endian and the packet structure is defined as follows:

[Length] length of packet: int32
[RequestID] client set id for each request used to track responses: int32
[Type] request packet type: int32
[Body] body of request/response: Null-terminated ASCII String
[Padding] body must be terminated by two null bytes

DISCLAIMER
This code has not been tested with commands that return data exceeding 4096 bytes and may not work.
*/

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"time"
)

const (
	//type value for failed authentication requests
	FailureType = int32(-1)
	_           = iota //1 is unused
	//type value for command requests
	CommandType //2
	//type value for authentication requests
	AuthenticationType //3

	//tcp constants
	Protocol = "tcp"
	Timeout  = time.Second * 10

	//request id reset value
	ResetID = 1
)

// remote console response headers
type headers struct {
	size      int32 //size of packet
	requestID int32 //client-side request id
}

// remote console response returned to client
type response struct {
	requestID int32  //client-side request id
	body      string //response from server
}

// minecraft remote console client
type client struct {
	server    net.Conn //server connection
	requestID int32    //self-incrementing request counter used for unique request id's
	address   string   //server address
}

type Client interface {
	Connect(address, password string) error
	Command(cmd string) (string, error)
	Close() error
	send(packet []byte) (*response, error)
	authenticate(password []byte) error
	createPacket(body []byte, packetType int32) ([]byte, error)
	incrementRequestID()
}

// creates and returns a new remote console client. The Connect method must be called before the client
// can be used to send commands
func NewClient(addr string) *client {
	return &client{
		server:    nil,
		requestID: ResetID,
		address:   addr,
	}
}

// connects to minecraft server and authenticates the client. Ensure to call or defer the call to the Close method
// to clean up the connection
func (c *client) Connect(password string) error {
	connection, err := net.DialTimeout(Protocol, c.address, Timeout)
	if err != nil {
		return err
	}

	c.server = connection

	err = c.authenticate([]byte(password))
	if err != nil {
		return err
	}

	return nil
}

// sends a command to the minecraft server and return the server response, command examples can be found on the
// minecraft wiki: https://minecraft.wiki/w/Commands
func (c *client) Command(cmd string) (string, error) {
	packet, err := c.createPacket([]byte(cmd), CommandType)
	if err != nil {
		return "", err
	}

	res, err := c.send(packet)
	if err != nil {
		return "", err
	}

	return res.body, nil
}

// closes remote console connection and resets the request id. The remote console client can be reused by calling
// the Connect method again
func (c *client) Close() error {
	c.requestID = ResetID
	return c.server.Close()
}

// sends a remote console packet to the minecraft server and parse response data, the requestID is incremented
// after each packet is sent
func (c *client) send(packet []byte) (*response, error) {
	_, err := c.server.Write(packet)
	if err != nil {
		return nil, err
	}

	var res headers
	err = binary.Read(c.server, binary.LittleEndian, &res)
	if err != nil {
		return nil, err
	}

	payload := make([]byte, res.size-8)
	err = binary.Read(c.server, binary.LittleEndian, &payload)
	if err != nil {
		return nil, err
	}

	c.incrementRequestID()

	return &response{
		requestID: res.requestID,
		body:      string(payload),
	}, nil
}

// sends authentication packet to minecraft server. This must be called before
// any commands can be run and returns an error if the supplied password is incorrect
func (c *client) authenticate(password []byte) error {
	packet, err := c.createPacket(password, AuthenticationType)
	if err != nil {
		return err
	}

	res, err := c.send(packet)
	if err != nil {
		return err
	}

	if res.requestID == FailureType {
		return errors.New("authentication failed")
	}

	return nil
}

// creates remote console packet using the body data based on the packetType value
func (c *client) createPacket(body []byte, packetType int32) ([]byte, error) {
	length := len(body) + 10 //length of body plus extra for headers

	//RCon packet structure
	//[Length] int32
	//[RequestID] int32
	//[Type] int32
	//[Body] Null-terminated ASCII String

	var buffer bytes.Buffer
	err := binary.Write(&buffer, binary.LittleEndian, int32(length))
	if err != nil {
		return nil, err
	}
	err = binary.Write(&buffer, binary.LittleEndian, c.requestID)
	if err != nil {
		return nil, err
	}
	err = binary.Write(&buffer, binary.LittleEndian, packetType)
	if err != nil {
		return nil, err
	}
	err = binary.Write(&buffer, binary.LittleEndian, body)
	if err != nil {
		return nil, err
	}
	err = binary.Write(&buffer, binary.LittleEndian, [2]byte{}) //padding
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// a simple handler for requestID header, the requestID is incremented after each packet sent to the server
// and is reset once it exceeds 100 to prevent any overflowing issues
func (c *client) incrementRequestID() {
	c.requestID++
	if c.requestID > 100 {
		c.requestID = ResetID
	}
}
