package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

const timeFormat = "2006-01-02"

type TransAction struct {
	Target string `json:"target"`
	Amount int64  `json:"amount"` // amount or unit * X < Limit
	Base   int64  `json:"base"`
	Unit   int64  `json:"unit"`
	Limit  int64  `json:"limit"`
}

func (a *TransAction) Execute(item *Item) error {
	acc := item.acc
	amount := a.Amount
	if amount == 0 {
		balance, _ := acc.TotalBalance()
		amount = (balance - a.Base) / a.Unit * a.Unit
	}
	if amount > a.Limit {
		amount = a.Limit
	}
	if amount > 0 {
		log.Println("Start transfer", amount)
		tr, err := acc.NewTransactionWithNick(a.Target, amount)
		if err != nil {
			return nil
		}
		log.Println("Execute transfer", tr)
		recpt, err := acc.CommitTransaction(tr, item.Password2)
		log.Println("Finish transfer ReceptId:", recpt, err)
	}
	return nil
}

type SlackAction struct {
	Channel string `json:"channel"`
	Message string `json:"message"`
}

func (a *SlackAction) Execute(item *Item) error {
	return SendSlackMessage(config.SlackUrl, a.Message+"  ("+item.Name)
}

type LogAction struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

func (a *LogAction) Execute(item *Item) error {
	log.Println(a.Level, a.Message)
	return nil
}

type Action struct {
	Type        string       `json:"type"`      // balance | history
	Op          string       `json:"op"`        // ">" | "<" | "match"
	Threshold   int64        `json:"threshold"` // integer type
	Match       string       `json:"match"`     // string type
	BalanceItem string       `json:"balance_item"`
	Interval    int          `json:"interval"` // hours
	Trans       *TransAction `json:"trans"`
	Slack       *SlackAction `json:"slack"`
	Log         *LogAction   `json:"log"`
}

type Item struct {
	Name        string    `json:"name"`
	Login       string    `json:"login"`
	Password2   string    `json:"password2"`
	Spreadsheet string    `json:"spreadsheet"`
	Actions     []*Action `json:"actions"`
	SaveStatus  string    `json:"save_status"`
	acc         Account
	status      ItemStatus
}

type ItemStatus struct {
	Balance        int64                `json:"balance"`
	LastSuccessful time.Time            `json:"last_successful"`
	LastExecution  map[string]time.Time `json:"last_event_exec"`
}

type Config struct {
	GoogleCredential string  `json:"google_credential"`
	SlackUrl         string  `json:"slack_url"` // incoming webhook url: https://hooks.slack.com/services/XXXXXXXXX
	Items            []*Item `json:"items"`
}

func load(path string) (*Config, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var c Config
	err = json.Unmarshal(raw, &c)
	if err != nil {
		return nil, err
	}
	return &c, err
}

func (s *Spreadsheet) EnsureSheetByName(name string) (*Sheet, error) {
	sheet, err := s.SheetByName(name)
	if err == nil {
		return sheet, nil
	}
	return s.AddSheet(name, 10, 2)
}

func (a *Action) Execute(item *Item) (err error) {
	log.Println("execute!", a)
	if a.Trans != nil {
		err = a.Trans.Execute(item)
	}
	if a.Slack != nil {
		err = a.Slack.Execute(item)
	}
	if a.Log != nil {
		err = a.Log.Execute(item)
	}
	return
}

var config *Config

func ensureLogin(item *Item) error {
	if item.acc != nil {
		return nil
	}
	// login
	acc, err := login(item.Login)
	if err != nil {
		return err
	}
	item.acc = acc
	// load status
	item.status.LastExecution = map[string]time.Time{}
	if item.SaveStatus != "" {
		if bytes, err := ioutil.ReadFile(item.SaveStatus); err == nil {
			_ = json.Unmarshal(bytes, &item.status)
		}
	}
	item.status.Balance, _ = acc.TotalBalance()
	return nil
}

func ensureLogout(item *Item) error {
	if item.acc == nil {
		return nil
	}
	// save
	if item.SaveStatus != "" {
		if bytes, err := json.Marshal(&item.status); err == nil {
			_ = ioutil.WriteFile(item.SaveStatus, bytes, 0644)
		}
	}
	err := item.acc.Logout()
	item.acc = nil
	return err
}

func main() {
	configPath := flag.String("c", "config.json", "config.json path")
	flag.Parse()

	c, err := load(*configPath)
	if err != nil {
		fmt.Println("Login error.", err)
		return
	}
	config = c
	target := flag.Arg(0)

	var sss *Service
	if config.GoogleCredential != "" {
		sss, err = NewService(config.GoogleCredential)
		if err != nil {
			log.Println("NewService() error:", err)
		}
	}

	for _, item := range config.Items {
		if target != "" && item.Name != target {
			continue
		}
		log.Println("Start:", item.Name)
		// login
		err := ensureLogin(item)
		if err != nil {
			log.Println("login error:", item.Name, err)
			continue
		}
		defer ensureLogout(item)
		log.Println("balance:", item.Name, item.status.Balance)

		var lastError error

		for _, act := range item.Actions {
			actStr := fmt.Sprint(act.Type, act.Op, act.Threshold, ":", act.Match) // TODO grouping
			if item.status.LastExecution[actStr].Add(time.Duration(act.Interval) * time.Hour).After(time.Now()) {
				continue
			}
			firing := false
			switch act.Type {
			case "balance":
				balance := item.status.Balance
				if act.BalanceItem != "" {
					for _, item2 := range config.Items {
						if item.Name == act.BalanceItem {
							err := ensureLogin(item2)
							if err != nil {
								fmt.Println("login error:", item.Name, err)
								lastError = err
								break
							}
							defer ensureLogout(item2)
							balance = item2.status.Balance
							break
						}
					}
				}
				if act.Op == ">" && balance > act.Threshold {
					firing = true
				} else if act.Op == "<" && balance < act.Threshold {
					firing = true
				}
			case "history":
				recent, _ := item.acc.Recent()
				for _, t := range recent {
					if strings.Contains(t.Description, act.Match) {
						// TODO: dup check.
						firing = true
					}
				}
			case "always":
				firing = true
			case "error":
				firing = lastError != nil
			}
			if firing {
				item.status.LastExecution[actStr] = time.Now()
				err := act.Execute(item)
				if err != nil {
					fmt.Println("Execute() error:", err, act)
					lastError = err
				}
			}
		}

		recent, _ := item.acc.Recent()

		// spreadsheet
		sheetAndName := strings.Split(item.Spreadsheet, ":")
		if len(sheetAndName) == 2 && sss != nil {
			ss, err := sss.Spreadsheet(sheetAndName[0])
			if err != nil {
				fmt.Println("Spreadsheet() error:", err)
				return
			}
			s, err := ss.EnsureSheetByName(sheetAndName[1])
			if err != nil {
				fmt.Println("SheetByName() error:", err)
				return
			}

			// Last Row
			last, _ := s.Read(fmt.Sprintf("A%d:E%d", s.RowCount(), s.RowCount()))

			var values [][]interface{}
			for _, t := range recent {
				log.Println(t.Date, t.Amount, t.Balance, t.Description)
				var in, out string
				if t.Amount < 0 {
					out = fmt.Sprint(-t.Amount)
				} else {
					in = fmt.Sprint(t.Amount)
				}
				row := []interface{}{t.Date.Format(timeFormat), in, out, t.Description, fmt.Sprint(t.Balance)}
				values = append(values, row)

				if len(last) > 0 && len(last[0]) >= 5 {
					lastRow := last[0]
					if row[0].(string) == lastRow[0].(string) && lastRow[1].(string) == in &&
						lastRow[2].(string) == out && row[4].(string) == lastRow[4].(string) {
						// match last
						values = values[:0]
					}
				}
			}
			if len(values) > 0 {
				err = s.Append(values)
				if err != nil {
					fmt.Println("Update error:", err)
				}
			}
		}
		if lastError == nil {
			item.status.LastSuccessful = time.Now()
		}
	}
}
