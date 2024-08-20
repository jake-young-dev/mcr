package mcr

import (
	"bytes"
	"encoding/binary"
	"errors"
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
	Timeout           = time.Second * 10
	RequestPacketSize = 10 //size of headers plus padding bytes
	PacketHeaderSize  = 8  //size of headers
	PacketPaddingSize = 2  //size of padding required after body

	//request id values
	ResetID = 1
	IDCap   = 100
)

// remote console response headers
type headers struct {
	Size      int32 //size of packet
	RequestID int32 //client-side request id
	Type      int32 //type of packet
}

// command response returned to client
type Response struct {
	RequestID int32  //client-side request id
	Body      string //response from server
}

// minecraft remote console client
type Client struct {
	server    net.Conn //server connection
	requestID int32    //self-incrementing request counter used for unique request id's
	address   string   //server address
}

type IClient interface {
	Connect(password string) error
	Command(cmd string) (string, error)
	Close() error
	send(packet []byte) (*Response, error)
	authenticate(password []byte) error
	createPacket(body []byte, packetType int32) ([]byte, error)
	incrementRequestID()
}

// creates and returns a new remote console client using the supplied address (addr). The Connect method must be called
// before the client can be used to send commands
func NewClient(addr string) *Client {
	return &Client{
		server:    nil,
		requestID: ResetID,
		address:   addr,
	}
}

// connects to minecraft server and authenticates the client. Ensure to call or defer the call to the Close method
// to clean up the connection
func (c *Client) Connect(password string) error {
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

// sends a command to the minecraft server and returns the server response, an error is returned if the client has
// not connected to the server before attempting to send a command. Command examples can be found on the
// minecraft wiki: https://minecraft.wiki/w/Commands
func (c *Client) Command(cmd string) (string, error) {
	if c.server == nil {
		return "", errors.New("the Connect method must be called before commands can be run")
	}

	packet, err := c.createPacket([]byte(cmd), CommandPacket)
	if err != nil {
		return "", err
	}

	res, err := c.send(packet)
	if err != nil {
		return "", err
	}

	return res.Body, nil
}

// closes remote console connection, nil's out the server value in client struct, and resets the request id. The remote
// console client can be reused by calling the Connect method again
func (c *Client) Close() error {
	c.requestID = ResetID
	err := c.server.Close()
	if err != nil {
		return err
	}
	c.server = nil
	return nil
}

// constructs and sends the tcp packet to the minecraft server and parses the response data, requestID is incremented
// after each packet is sent
func (c *Client) send(packet []byte) (*Response, error) {
	_, err := c.server.Write(packet)
	if err != nil {
		return nil, err
	}

	var res headers
	err = binary.Read(c.server, binary.LittleEndian, &res)
	if err != nil {
		return nil, err
	}

	payload := make([]byte, res.Size-PacketHeaderSize) //read body size (total size - header size)
	err = binary.Read(c.server, binary.LittleEndian, &payload)
	if err != nil {
		return nil, err
	}

	c.incrementRequestID()

	return &Response{
		RequestID: res.RequestID,
		Body:      string(payload),
	}, nil
}

// sends authentication packet to minecraft server. This must be called before
// any commands can be run and returns an error if the supplied password is incorrect
func (c *Client) authenticate(password []byte) error {
	packet, err := c.createPacket(password, AuthPacket)
	if err != nil {
		return err
	}

	res, err := c.send(packet)
	if err != nil {
		return err
	}

	if res.RequestID == FailurePacket {
		return errors.New("authentication failed")
	}

	return nil
}

// creates remote console packet using the body data based on the packetType value
func (c *Client) createPacket(body []byte, packetType int32) ([]byte, error) {
	length := len(body) + RequestPacketSize

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

// a simple handler for requestID header, the requestID is incremented after each packet sent to the server
// and is reset once it exceeds 100 to prevent any overflowing issues
func (c *Client) incrementRequestID() {
	c.requestID++
	if c.requestID > IDCap {
		c.requestID = ResetID
	}
}
