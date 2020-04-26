package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func min(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

func fixTime(t time.Time) int64 {
	const lifetime = 86400
	const step = 120
	now := time.Now().UnixNano() / int64(time.Second)
	tt := t.UnixNano() / int64(time.Second)
	return (tt + min((now-tt)/step*step, lifetime)) * 1000
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] metrics_dir\n", os.Args[0])
		flag.PrintDefaults()
	}
	// all := flag.Bool("a", false, "full scan")

	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		return
	}

	dir := flag.Arg(0)

	var metrics [][]byte

	var total int64
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	for _, f := range files {
		name := filepath.Base(strings.TrimSuffix(f, filepath.Ext(f)))
		if name == "test" {
			continue
		}
		buf, err := ioutil.ReadFile(f)
		if err != nil {
			continue
		}
		var status struct {
			Balance        int64                `json:"balance"`
			LastSuccessful time.Time            `json:"last_successful"`
			LastExecution  map[string]time.Time `json:"last_event_exec"`
		}
		err = json.Unmarshal(buf, &status)
		if err != nil {
			continue
		}

		total += status.Balance
		t := fixTime(status.LastSuccessful)
		metrics = append(metrics, []byte(fmt.Sprintf("bank_balance{account=\"%v\"} %v %v", name, status.Balance, t)))
	}
	metrics = append(metrics, []byte(fmt.Sprintf("bank_balance_total %v", total)))
	metrics = append(metrics, []byte{})

	ioutil.WriteFile(filepath.Join(dir, "metrics"), bytes.Join(metrics, []byte("\n")), 0644)
}
