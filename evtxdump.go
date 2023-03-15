package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"rawsec-evtx/evtx"
	"rawsec-evtx/log"
	"strconv"
	"strings"
)

const version = "1.0"

func main() {
	var strEventIds string
	flag.StringVar(&strEventIds, "e", "", "Comma seperated event IDs")

	flag.Usage = func() {
		fmt.Printf("%s\nUsage of %s: %[1]s [OPTIONS] FILES...\n", version, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	flag.Parse()

	var eventIds []interface{}
	for _, i := range strings.Split(strEventIds, ",") {
		if _, err := strconv.ParseInt(i, 10, 64); err == nil {
			eventIds = append(eventIds, i)
		}
	}

	for _, evtxFile := range flag.Args() {
		ef, err := evtx.OpenDirty(evtxFile)
		if err != nil {
			log.Error(err)
			continue
		}

		name := strings.TrimSuffix(evtxFile, filepath.Ext(evtxFile)) + ".json"
		f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			log.Error(err)
			continue
		}

		_, err = f.WriteString("[")
		if err != nil {
			log.Error(err)
			_ = f.Close()
			continue
		}

		for e := range ef.UnorderedEvents() {
			if e == nil {
				continue
			}

			if eventIds != nil && !e.IsEventID(eventIds...) {
				continue
			}

			_, err = f.WriteString(string(evtx.ToJSON(e)) + ",")
			if err != nil {
				log.Error(err)
				break
			}
		}

		if err == nil {
			_, err = f.WriteString("null]")
			if err != nil {
				log.Error(err)
			}
		}

		_ = f.Close()
	}
}
