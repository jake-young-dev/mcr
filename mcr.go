// mcr is an RCon client that provides useful methods for connecting to and managing
// game servers that support the source protocol
package mcr

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"time"
)

const (
	//packet types are used to represent the status of the packet

	//the server encountered an error while handling the last request
	FailurePacket = int32(-1)
	//represents any command structs as well as there responses
	CommandPacket = int32(2)
	//used for any packets in the authentication handshake
	AuthPacket = int32(3)

	//tcp constants
	Protocol          = "tcp"
	PacketRequestSize = 10 //size of headers plus padding bytes, not including Size header per RCon standard
	PacketHeaderSize  = 8  //size of headers not including Size header per RCon standard
	PacketPaddingSize = 2  //size of padding required after body

	//default client configuration values
	ResetID        = 1
	DefaultCap     = 100
	DefaultTimeout = time.Second * 10
	DefaultPort    = 61695
)

var (
	ErrClientNotConnected = errors.New("client not connected. The Connect method must be called before commands can be run")
	ErrIntOverflow        = errors.New("integer overflowed 32 bits")
)

// header represents the network packets sent from the client and received from the server
type header struct {
	Size      int32 //size of packet
	RequestID int32 //client-side request id
	Type      int32 //type of packet
}

// response contains the fields sent in server responses
type response struct {
	RequestID int32  //client-side request id
	Type      int32  //packet type
	Body      string //response body from server
}

// Client contains the configuration and methods for the RCon connection
type Client struct {
	connection net.Conn      //server connection
	requestID  int32         //self-incrementing request counter used for unique request id's
	address    string        //server address
	port       int           //server port
	timeout    time.Duration //timeout for connection
	cap        int32         //request id capacity before resetting it
}

type client interface {
	Connect(password string) error
	Command(cmd string) (string, error)
	CommandNoResponse(cmd string) error
	Close() error
	RequestID() int32
	SetRequestID(id int32)
	Timeout() time.Duration
	Connection() net.Conn
	Port() int
	Address() string
	//filtered methods
	sendAndRecv(packet []byte) (*response, error)
	send(packet []byte) error
	createPacket(body []byte, packetType int32) ([]byte, error)
	authenticate(password []byte) error
	incrementRequestID()
	safeIntConversion(n int) (int32, error)
}

// NewClient creates a new remote console client configured with the supplied options. The Connect method must be called before the server
// can be interacted with.
func NewClient(addr string, opts ...Option) client {
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

// Connect sends the connection request to the server and authenticates the client. Ensure to call,
// or defer the call to, the Close method to clean up the connection after use
func (c *Client) Connect(password string) error {
	if c.connection == nil {
		connection, err := net.DialTimeout(Protocol, net.JoinHostPort(c.address, fmt.Sprint(c.port)), c.timeout)
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

// Command sends the payload to the server and waits for the server response. The response packet is parsed
// and the body data is returned, an error is returned if the client has not connected before sending the
// packet.
func (c *Client) Command(cmd string) (string, error) {
	if c.connection == nil {
		return "", ErrClientNotConnected
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

// CommandNoResponse sends a payload to the server without waiting for a response, the client must be connected
// to the server before any commands can be sent.
func (c *Client) CommandNoResponse(cmd string) error {
	if c.connection == nil {
		return ErrClientNotConnected
	}

	packet, err := c.createPacket([]byte(cmd), CommandPacket)
	if err != nil {
		return err
	}

	return c.send(packet)
}

// Close disconnects from the server and resets the clients request id.
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

// RequestID returns the current packets request ID.
func (c *Client) RequestID() int32 {
	return c.requestID
}

// SetRequestID of the current packet.
func (c *Client) SetRequestID(id int32) {
	c.requestID = id
}

// Timeout returns the current clients connection timeout.
func (c *Client) Timeout() time.Duration {
	return c.timeout
}

// Connection returns the underlying client connection, this value cannot be updated after the server has been
// connected to. Instead, a new client must be created.
func (c *Client) Connection() net.Conn {
	return c.connection
}

// Port returns the server port, this value cannot be updated after the connection is made. Instead, a new
// client should be created
func (c *Client) Port() int {
	return c.port
}

// Address returns the current server address, this value cannot be updated after the connection is made. Instead, a new
// client should be created
func (c *Client) Address() string {
	return c.address
}

// constructs and sends the tcp packet to the server and parses the response data, requestID is incremented
// after each packet is sent
func (c *Client) sendAndRecv(packet []byte) (*response, error) {
	_, err := c.connection.Write(packet)
	if err != nil {
		return nil, err
	}

	var res header
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
	payload = payload[:len(payload)-PacketPaddingSize]

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
	length, err := c.safeIntConversion(len(body) + PacketRequestSize)
	if err != nil {
		return nil, err
	}

	//packet structure
	//[Length] length of packet: int32
	//[RequestID] client set id for each request used to track responses: int32
	//[Type] request packet type: int32
	//[Body] body of request/response: Null-terminated ASCII String
	//[Padding] body must be terminated by two null bytes
	head := header{
		Size:      length,
		RequestID: c.requestID,
		Type:      packetType,
	}
	buffer := make([]byte, 0, binary.Size(head)+len(body)+2)
	buffer, err = binary.Append(buffer, binary.LittleEndian, head)
	if err != nil {
		return nil, err
	}
	buffer = append(buffer, body...)
	buffer = append(buffer, make([]byte, PacketPaddingSize)...)

	return buffer, nil
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

// prevents integer overflow errors when converting "int" to "int32" to ensure safe conversion
func (c *Client) safeIntConversion(n int) (int32, error) {
	if n > math.MaxInt32 || n < math.MinInt32 {
		return 0, ErrIntOverflow
	}

	return int32(n), nil
}
