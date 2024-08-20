# mcr
Dependency-free Minecraft remote console client written in Golang

# usage
```
import (
	"log"
	"os"

	"github.com/jake-young-dev/mcr"
)

func main() {
	//create new client
	client := mcr.NewClient(os.Getenv("rcon_address"))

	//connect to server and authenticate with password
	err := client.Connect(os.Getenv("rcon_password"))
	if err != nil {
		panic(err)
	}
	defer client.Close() //always call close to clean up your connections

	response, err := client.Command("list")
	if err != nil {
		panic(err)
	}

	log.Println(response)
}
```

# notes
- To prevent consuming extra resources the Minecraft server is only connected to after calling the Connect method
- The Close method must be called to clean up the client after use
- The client can be reused after closing by calling the Connect method again