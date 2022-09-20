package main

import (
	bot2 "bot/bot"
	"log"
	"os"
	"time"
)

func main() {

	bot := bot2.NewGoCQBot("ws://localhost:3333", ".")
	loadGoCQBot(bot)

	for {
		//TODO 控制台输入
		time.Sleep(1000)
	}
}

func loadGoCQBot(bot *bot2.GoCQ) {
	log.Println("MUGBot启动中...")
	err := bot.Start()
	//bot.AddListener(event.Message, func(iEvent event.IEvent) error {
	//	e := iEvent.(*event.MessageEvent)
	//	log.Println(*e.RawMsg)
	//	return nil
	//})
	if err != nil {
		log.Panicln(err)
	}

	bot2.GoCQBot = bot
}

func initPlugins() error {
	fileInfo, err := os.Stat("plugins")
	if os.IsNotExist(err) {
		err = os.Mkdir("plugins", 0755)
	} else {
		if !fileInfo.IsDir() {
			err = os.Remove("plugins")
		}
	}

	if err != nil {
		return err
	}

	return nil
}
