
// This go code will not compile. It is the main handler for anonymorse.
// Any other information can be reasonably assumed (e.g. how the main 
//    method calls this functionq)

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func wshandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsupgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Failed to set websocket upgrade: %+v", err)
		return
	}

	// new connected client
	// add them to clients
	clients[r.RemoteAddr] = WSClient{WAITING, r.RemoteAddr, "", conn}

	fmt.Println("NEW CONNECTION: " + r.RemoteAddr)

	fmt.Println("There are now " + strconv.Itoa(len(clients)) + " clients online")

	for {
		// clients start as waiting
		// try to find them a match
		if clients[r.RemoteAddr].Status == WAITING {
			// they're waiting
			// see if anyone else is waiting
			for k := range clients {
				if clients[k].Status == WAITING && clients[k].Address != r.RemoteAddr {
					fmt.Println("Pairing Clients: " + r.RemoteAddr + "  &  " + clients[k].Address)

					// there is another waiting client
					// pair them together
					clients[r.RemoteAddr] = WSClient{
						CONNECTED,
						r.RemoteAddr,
						clients[k].Address,
						conn,
					}

					clients[k] = WSClient{
						CONNECTED,
						clients[k].Address,
						r.RemoteAddr,
						clients[k].Connection,
					}

					// they are now paired
					conn.WriteMessage(1, []byte("connected"))
					clients[k].Connection.WriteMessage(1, []byte("connected"))

					fmt.Println("  ^ Success")

					break
				}
			}
		}

		t, msg, err := conn.ReadMessage()

		// fmt.Println(t)

		if err != nil {
			fmt.Println("DISCONNECT: " + r.RemoteAddr)
			// kill the connection
			if clients[r.RemoteAddr].Status == CONNECTED {
				// inform their partner
				partner := clients[r.RemoteAddr].ConnectedTo
				
				fmt.Println("  Informing their partner, " + partner)

				clients[partner].Connection.WriteMessage(1, []byte("disconnected"))

				clients[partner] = WSClient{
					DISCONNECTED,
					clients[partner].Address,
					"",
					clients[partner].Connection,
				}
			}

			// delete them
			delete(clients, r.RemoteAddr)

			return
		}

		split := strings.Split(string(msg), ":")

		event := split[0]
		id := ""

		if len(split) > 1 {
			id = split[1]
		}

		switch event {
		case "beep_start":
			// inform their partner
			if clients[r.RemoteAddr].Status == CONNECTED {
				partner := clients[r.RemoteAddr].ConnectedTo
				clients[partner].Connection.WriteMessage(t, []byte("beep_start:" + id))
			}
			break
		case "beep_end":
			// inform their partner
			if clients[r.RemoteAddr].Status == CONNECTED {
				partner := clients[r.RemoteAddr].ConnectedTo
				clients[partner].Connection.WriteMessage(t, []byte("beep_end:" + id))
			}
			break
		case "disconnect":
			// tell their partner that they disconnected
			fmt.Println("  " + r.RemoteAddr + " disconnected")

			if clients[r.RemoteAddr].Status == CONNECTED {
				partner := clients[r.RemoteAddr].ConnectedTo
				clients[partner].Connection.WriteMessage(t, []byte("disconnected"))

				// set their partner's state
				clients[partner] = WSClient{
					DISCONNECTED,
					clients[partner].Address,
					"",
					clients[partner].Connection,
				}
			}

			// set their state to disconnected
			clients[r.RemoteAddr] = WSClient{
				DISCONNECTED,
				clients[r.RemoteAddr].Address,
				"",
				clients[r.RemoteAddr].Connection,
			}
			break
		case "enter_queue":
			// set their state to waiting
			fmt.Println("  " + r.RemoteAddr + " has entered the queue")
			clients[r.RemoteAddr] = WSClient{
				WAITING,
				clients[r.RemoteAddr].Address,
				"",
				clients[r.RemoteAddr].Connection,
			}
			break
		}

		conn.WriteMessage(t, msg)
	}
}
