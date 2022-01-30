package polybar

import (
	"errors"
	"fmt"
	"html"
	"os"
	"path"
	"strings"

	"github.com/Pauloo27/go-mpris"
	"github.com/Pauloo27/gotroller"
	"github.com/Pauloo27/gotroller/cli/utils"
	"github.com/fsnotify/fsnotify"
	"github.com/godbus/dbus/v5"
	"github.com/joho/godotenv"
)

var (
	maxTitleSize  int
	maxArtistSize int
)

func loadMaxSizes() {
	home, err := os.UserHomeDir()
	if err == nil {
		godotenv.Load(path.Join(home, ".config", "gotroller.env"))
	}
	maxTitleSize = utils.AtoiOrDefault(os.Getenv("GOTROLLER_MAX_TITLE_SIZE"), 30)
	maxArtistSize = utils.AtoiOrDefault(os.Getenv("GOTROLLER_MAX_ARTIST_SIZE"), 20)
}

func handleError(err error, message string) {
	if err != nil {
		fmt.Println(Span{UNDERLINE, "#ff0000", message}.String())
		os.Exit(-1)
	}
}

func gotrollerCLI(command string) string {
	return fmt.Sprintf("gotroller %s", command)
}

func startMainLoop(playerSelectCommand string) {
	player, err := gotroller.GetBestPlayer()
	if err != nil {
		if errors.Is(err, gotroller.ErrDisabled{}) {
			playerSelectorAction := ActionButton{LEFT_CLICK, gotroller.MENU, playerSelectCommand}
			fmt.Printf("%s\n", playerSelectorAction.String())
			return
		}
		handleError(err, "Cannot get best player")
	}
	if player == nil {
		fmt.Println("Nothing playing...")
		return
	}

	update := func() {
		printToPolybar(playerSelectCommand, player)
	}

	update()
	mprisCh := make(chan *dbus.Signal)
	err = player.OnSignal(mprisCh)
	handleError(err, "Cannot listen to mpris signals")

	preferedPlayerCh := make(chan fsnotify.Event)
	gotroller.ListenToChanges(preferedPlayerCh)

	go func() {
		for range preferedPlayerCh {
			// prefered player changed
			os.Exit(0)
		}
	}()

	for sig := range mprisCh {
		if sig.Name == "org.freedesktop.DBus.NameOwnerChanged" {
			// player exitted
			if len(sig.Body) == 3 && sig.Body[0] == "org.mpris.MediaPlayer2.mpv" {
				os.Exit(0)
			}
		}
		update()
	}
}

func printToPolybar(playerSelectCommand string, player *mpris.Player) {
	metadata, err := player.GetMetadata()
	handleError(err, "Cannot get player metadata")

	status, err := player.GetPlaybackStatus()
	handleError(err, "Cannot get playback status")

	volume, err := player.GetVolume()
	handleError(err, "Cannot get volume")

	stopped := false

	var icon string
	switch status {
	case mpris.PlaybackPaused:
		icon = gotroller.PAUSED
	case mpris.PlaybackStopped:
		icon = gotroller.STOPPED
		stopped = true
	default:
		icon = gotroller.PLAYING
	}

	var title string
	if rawTitle, ok := metadata["xesam:title"]; ok {
		title = rawTitle.Value().(string)
	}

	var artist string
	if rawArtist, ok := metadata["xesam:artist"]; ok {
		switch rawArtist.Value().(type) {
		case string:
			artist = rawArtist.Value().(string)
		case []string:
			artist = strings.Join(rawArtist.Value().([]string), ", ")
		}
	}

	fullTitle := utils.EnforceSize(title, maxTitleSize)
	if artist != "" {
		fullTitle += " from " + utils.EnforceSize(artist, maxArtistSize)
	}
	// since lainon.life radios' uses HTML notation in the "japanese" chars
	// we need to decode them
	fullTitle = html.UnescapeString(fullTitle)

	playerSelectorAction := ActionButton{LEFT_CLICK, gotroller.MENU, playerSelectCommand}

	playPause := ActionButton{LEFT_CLICK, icon, gotrollerCLI("play-pause")}

	// previous + restart
	previous := ActionOver(
		ActionButton{LEFT_CLICK, gotroller.PREVIOUS, gotrollerCLI("prev")},
		RIGHT_CLICK, gotrollerCLI("position 0"), // TODO:
	)

	next := ActionButton{LEFT_CLICK, gotroller.NEXT, gotrollerCLI("next")}

	volumeAction := ActionOver(
		ActionButton{SCROLL_UP, fmt.Sprintf("%s %.f%%", gotroller.VOLUME, volume*100), gotrollerCLI("volume +0.05")},
		SCROLL_DOWN,
		gotrollerCLI("volume -0.05"),
	)

	if stopped {
		fmt.Printf("%s %s\n", playerSelectorAction.String(), icon)
	} else {
		// Print everything
		fmt.Printf("%s %s %s %s %s %s\n",
			playerSelectorAction.String(),
			fullTitle,
			// restart contains previous
			previous.String(),
			playPause.String(),
			next.String(),
			volumeAction.String(),
		)
	}
}
