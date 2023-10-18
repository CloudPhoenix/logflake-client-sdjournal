package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CloudPhoenix/logflake-client-go/logflake"
	"github.com/coreos/go-systemd/sdjournal"
)

func main() {
	l := logflake.New(os.Getenv("LOGFLAKE_APPID"))
	if l.AppKey == "" {
		log.Fatalln(" env LOGFLAKE_APPID not found")
	}
	r, err := sdjournal.NewJournalReader(sdjournal.JournalReaderConfig{
		NumFromTail: 1,
		Formatter: func(entry *sdjournal.JournalEntry) (string, error) {
			data, _ := json.Marshal(entry)
			log.Println(string(data))
			params := map[string]interface{}{}
			for k, v := range entry.Fields {
				params[k] = v
			}
			var level logflake.LogLevel
			switch entry.Fields["PRIORITY"] {
			// 0: emerg -- 1: alert -- 2: crit
			case "0":
			case "1":
			case "2":
				level = logflake.LevelFatal
			// 3: err
			case "3":
				level = logflake.LevelError
			// 4: warning
			case "4":
				level = logflake.LevelWarn
			// 5: notice -- 6: info
			case "5":
			case "6":
				level = logflake.LevelInfo
			default:
			case "7": // 7: debug
				level = logflake.LevelDebug
			}
			l.SendLog(logflake.Log{
				Correlation: entry.Fields["_SYSTEMD_UNIT"],
				Content:     entry.Fields["MESSAGE"],
				Params:      params,
				Level:       level,
			})
			return "", nil
		},
	})
	if err != nil {
		log.Fatalln(err)
	}
	if r == nil {
		log.Fatalln("Got a nil reader")
	}
	defer r.Close()

	timeout := make(chan time.Time, 1)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig
		timeout <- time.Now()
	}()

	if err = r.Follow(timeout, os.Stdout); err != sdjournal.ErrExpired {
		log.Fatalln("Error during follow:", err)
	}
}
