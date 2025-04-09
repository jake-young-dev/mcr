# mcr
[![tests](https://github.com/jake-young-dev/mcr/actions/workflows/test.yaml/badge.svg?branch=master&event=push)](https://github.com/jake-young-dev/mcr/actions/workflows/test.yaml) <br />
mcr is a dependency-free remote console (RCon) package written in Golang following the [Source](https://developer.valvesoftware.com/wiki/Source_RCON_Protocol) protocol.

# Usage
```
import (
	"log"
	"os"

	"github.com/jake-young-dev/mcr"
)

func main() {
	//create new client to server address on port 9876
	client := mcr.NewClient(os.Getenv("rcon_address"), mcr.WithPort(9876))

	//connect to server and authenticate with password
	err := client.Connect(os.Getenv("rcon_password"))
	if err != nil {
		panic(err)
	}
	defer client.Close() //always call close to clean up your connections

	response, err := client.Command("list") //run "list" command on server
	if err != nil {
		panic(err)
	}

	log.Println(response)
}
```

# Default Options
- Timeout is defaulted 10 seconds
- Port is defaulted to 61695

# Security
- RCon is an inherently insecure protocol that sends passwords in plaintext. I recommend using a VPN or keeping the connection local when possible.
- All code is checked with [gosec](https://github.com/securego/gosec) as an added security measure
