# mcr
Dependency-free Minecraft remote console client written in Golang

# usage
```
import (
	"log"
	"os"
	"time"

	"github.com/jake-young-dev/mcr"
)

func main() {
	//create new client with minecraft server address and nonmandatory options if the default values need to be changed
	client := mcr.NewClient(os.Getenv("rcon_address"))

	//connect to server and authenticate with password
	err := client.Connect(os.Getenv("rcon_password"))
	if err != nil {
		panic(err)
	}
	defer client.Close() //always call close to clean up your connections

	response, err := client.Command("list") //run "list" command on minecraft server
	if err != nil {
		panic(err)
	}

	log.Println(response)
}
```

# notes
- To prevent using connections prematurely the client does not connect to the server on creation, the Connect method must be called
- To cleanup connections after use call the Close method, it is recommended to defer the Close after the call to Connect
- The client can be reused after closing by calling the Connect method again