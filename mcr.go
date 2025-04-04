package mcr

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
)

const (
	//rcon packet type values
	FailurePacket = int32(-1)
	CommandPacket = int32(2)
	AuthPacket    = int32(3)

	//tcp constants
	Protocol          = "tcp"
	PacketRequestSize = 10 //size of headers plus padding bytes, not including Size header per RCon standard
	PacketHeaderSize  = 8  //size of headers not including Size header per RCon standard
	PacketPaddingSize = 2  //size of padding required after body

	//default values
	ResetID        = 1
	DefaultCap     = 100
	DefaultTimeout = time.Second * 10
	DefaultPort    = 61695
)

// remote console response headers
type headers struct {
	Size      int32 //size of packet
	RequestID int32 //client-side request id
	Type      int32 //type of packet
}

// command response returned to client
type response struct {
	RequestID int32 //client-side request id
	Type      int32
	Body      string //response from server
}

// remote console client
type Client struct {
	connection net.Conn      //server connection
	requestID  int32         //self-incrementing request counter used for unique request id's
	address    string        //server address
	port       int           //server port
	timeout    time.Duration //timeout for connection
	cap        int32         //request id capacity before resetting it
}

type IClient interface {
	//main rcon methods
	Connect(password string) error
	Command(cmd string) (string, error)
	Send(cmd string) error
	Close() error
	//filtered methods
	sendAndRecv(packet []byte) (*response, error)
	send(packet []byte) error
	createPacket(body []byte, packetType int32) ([]byte, error)
	authenticate(password []byte) error
	incrementRequestID()
}

// creates a new remote console client configured with the supplied options. The client does not connect to the server until the
// Connect method is called to authenticate the client. Check the README for information on default values
func NewClient(addr string, opts ...Option) *Client {
	c := &Client{
		connection: nil,
		requestID:  ResetID,
		address:    addr,
		port:       DefaultPort,
		timeout:    DefaultTimeout,
		cap:        DefaultCap,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// connects to server and authenticates the client. Ensure to call, or defer the call to, the Close method
// to clean up the connection
func (c *Client) Connect(password string) error {
	if c.connection == nil {
		connection, err := net.DialTimeout(Protocol, fmt.Sprintf("%s:%d", c.address, c.port), c.timeout)
		if err != nil {
			return err
		}

		c.connection = connection
	}

	err := c.authenticate([]byte(password))
	if err != nil {
		return err
	}

	return nil
}

// sends a command to the server and returns the server response, an error is returned if the client has
// not connected to the server before attempting to send a command
func (c *Client) Command(cmd string) (string, error) {
	if c.connection == nil {
		return "", errors.New("the Connect method must be called before commands can be run")
	}

	packet, err := c.createPacket([]byte(cmd), CommandPacket)
	if err != nil {
		return "", err
	}

	res, err := c.sendAndRecv(packet)
	if err != nil {
		return "", err
	}

	return res.Body, nil
}

// sends a command to the server without waiting for the response, an error is returned if the client has
// not connected to the server before attempting to send a command
func (c *Client) Send(cmd string) error {
	if c.connection == nil {
		return errors.New("the Connect method must be called before commands can be run")
	}

	packet, err := c.createPacket([]byte(cmd), CommandPacket)
	if err != nil {
		return err
	}

	return c.send(packet)
}

// closes remote console connection, nil's out the connection value in client struct, and resets the request id
func (c *Client) Close() error {
	c.requestID = ResetID
	if c.connection != nil {
		err := c.connection.Close()
		if err != nil {
			return err
		}
		c.connection = nil
	}
	return nil
}

// constructs and sends the tcp packet to the server and parses the response data, requestID is incremented
// after each packet is sent
func (c *Client) sendAndRecv(packet []byte) (*response, error) {
	_, err := c.connection.Write(packet)
	if err != nil {
		return nil, err
	}

	var res headers
	err = binary.Read(c.connection, binary.LittleEndian, &res)
	if err != nil {
		return nil, err
	}

	payload := make([]byte, res.Size-PacketHeaderSize) //read body size (total size - header size)
	err = binary.Read(c.connection, binary.LittleEndian, &payload)
	if err != nil {
		return nil, err
	}

	//remove byte padding
	payload = payload[:len(payload)-2]

	c.incrementRequestID()

	return &response{
		RequestID: res.RequestID,
		Type:      res.Type,
		Body:      string(payload),
	}, nil
}

// constructs and sends the tcp packet to the server without waiting for a response, requestID is incremented
// after each packet is sent
func (c *Client) send(packet []byte) error {
	_, err := c.connection.Write(packet)
	if err != nil {
		return err
	}
	c.incrementRequestID()

	return nil
}

// creates remote console packet including the body and packet type returning the packet bytes. These bytes
// can be sent directly to the server.
func (c *Client) createPacket(body []byte, packetType int32) ([]byte, error) {
	length := len(body) + PacketRequestSize

	//packet structure
	//[Length] length of packet: int32
	//[RequestID] client set id for each request used to track responses: int32
	//[Type] request packet type: int32
	//[Body] body of request/response: Null-terminated ASCII String
	//[Padding] body must be terminated by two null bytes

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
	err = binary.Write(&buffer, binary.LittleEndian, [PacketPaddingSize]byte{}) //padding
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// sends authentication packet to server. This must be called before
// any commands can be run and returns an error if the supplied password is incorrect
func (c *Client) authenticate(password []byte) error {
	packet, err := c.createPacket(password, AuthPacket)
	if err != nil {
		return err
	}

	res, err := c.sendAndRecv(packet)
	if err != nil {
		return err
	}

	if res.RequestID == FailurePacket { //request id is set to -1 if auth fails
		return errors.New("authentication failed")
	}

	return nil
}

// a simple handler for requestID header, the requestID is incremented after each packet sent to the server
// and is reset once it exceeds IDCap to prevent any overflowing issues
func (c *Client) incrementRequestID() {
	c.requestID++
	if c.requestID > c.cap {
		c.requestID = ResetID
	}
}
