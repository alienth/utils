package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Packet struct {
	Source struct {
		Layers struct {
			Frame       []string `json:"frame.number"`
			Time        []string `json:"frame.time"`
			Stream      []string `json:"tcp.stream"`
			Query       []string `json:"pgsql.query"`
			ReponseType []string `json:"pgsql.type"`
		} `json:"layers"`
	} `json:"_source"`
	Index string `json:"_index"`
}

type Query struct {
	Query string
	Time  time.Time
}

func main() {
	// tshark -r /tmp/dump -Y 'pgsql.query' -e 'tcp.stream' -e 'frame.number' -e 'pgsql.query' -T fields
	args := strings.Split("-r filename -Y pgsql.query -e frame.number -e frame.time -e tcp.stream -e pgsql.query -e pgsql.type -T json", " ")
	args[1] = os.Args[1]

	cmd := exec.Command("tshark", args...)
	stdout, err := cmd.Output()
	if err != nil {
		log.Fatal("Error running tshark: ", err)
	}

	var packets []Packet

	err = json.Unmarshal(stdout, &packets)
	if err != nil {
		fmt.Println(err)
		log.Fatal("Error unmarshalling: ", err)
	}

	fmt.Printf("Found %d packets.\n", len(packets))

	LastQueryTimeByStream := make(map[int]Query)
	for _, p := range packets {
		data := p.Source.Layers
		frameTime, err := time.Parse("Jan  2, 2006 15:04:05.000000000 MST", data.Time[0])
		if err != nil {
			fmt.Println(err)
		}
		stream, _ := strconv.Atoi(data.Stream[0])

		if query, ok := LastQueryTimeByStream[stream]; ok {
			complete := false
			for _, responseType := range data.ReponseType {
				if responseType == "Command completion" {
					complete = true
				}
			}
			if complete {
				responseTime := frameTime.Sub(query.Time)
				fmt.Println(responseTime)
				if responseTime.Seconds() > 0.001 {
					fmt.Println(query.Time, data.Frame[0], stream, responseTime.Seconds())
					fmt.Println(query.Query)
					// fmt.Println(responseTime.Seconds())
				}
				delete(LastQueryTimeByStream, stream)
			}
		}
		if len(data.Query) > 0 {
			var query Query
			query.Query = data.Query[0]
			query.Time = frameTime
			LastQueryTimeByStream[stream] = query
		}

	}

}
