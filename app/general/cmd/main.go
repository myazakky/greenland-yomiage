package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	config "github.com/chun37/greenland-yomiage/general/internal/config"
	"github.com/chun37/greenland-yomiage/general/internal/handler"
	"github.com/chun37/greenland-yomiage/general/internal/initialize"
	"github.com/chun37/greenland-yomiage/general/internal/listener"
	"github.com/chun37/greenland-yomiage/general/internal/speaker"
)

// Variables used for command line parameters
var (
	Token            string
	GuildID          string
	YomiageChannelID string
)

func init() {
	Token = os.Getenv("DISCORD_TOKEN")
	GuildID = os.Getenv("DISCORD_GUILD_ID")
	YomiageChannelID = os.Getenv("DISCORD_YOMIAGE_CH_ID")

	if Token == "" {
		panic("環境変数`DISCORD_TOKEN`がセットされていません")
	}
	if GuildID == "" {
		panic("環境変数`DISCORD_GUILD_ID`がセットされていません")
	}
	if YomiageChannelID == "" {
		panic("環境変数`DISCORD_YOMIAGE_CH_ID`がセットされていません")
	}
}

func main() {
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		log.Fatalf("認証に失敗しました: %+v\n", err)
	}

	dg.Identify.Intents = discordgo.IntentsAll
	if err := dg.Open(); err != nil {
		log.Fatalf("コネクションを確立できませんでした: %+v\n", err)
	}

	cfg := config.Config{
		TargetChannelID: YomiageChannelID,
	}
	externalDeps := initialize.NewExternalDependencies()
	usecases := initialize.NewUsecases(externalDeps)
	hp := initialize.NewHandlerProps(cfg, usecases)

	messages := make(chan speaker.SpeechMessage, 10)
	soundPacket := make(chan *discordgo.Packet, 1)
	quiet := make(chan struct{})

	hdr := handler.New(hp, messages, soundPacket)
	dg.AddHandler(hdr.TTS(messages, quiet))
	dg.AddHandler(hdr.Disconnect)

	interactionHandler, _ := hdr.Interaction(dg, GuildID)
	dg.AddHandler(interactionHandler)

	spkr := speaker.NewSpeaker(usecases.TTSUsecase, messages, quiet)
	listener := listener.NewListener(soundPacket, quiet)
	go spkr.Run()
	go listener.Run()

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("Removing commands...")
	// // We need to fetch the commands, since deleting requires the command ID.
	// // We are doing this from the returned commands on line 375, because using
	// // this will delete all the commands, which might not be desirable, so we
	// // are deleting only the commands that we added.
	// registeredCommands, err := s.ApplicationCommands(s.State.User.ID, *GuildID)
	// if err != nil {
	// 	log.Fatalf("Could not fetch registered commands: %v", err)
	// }

	/*for _, scid := range slashCommandIDs {
		err := dg.ApplicationCommandDelete(dg.State.User.ID, GuildID, scid)
		if err != nil {
			log.Panicf("Cannot delete command: %+v", err)
		}
	}*/

	log.Println("Gracefully shutting down.")

	// Cleanly close down the Discord session.
	dg.Close()
}
