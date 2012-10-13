package bot

import (
	"fmt"
	"github.com/fluffle/goevent/event"
	"github.com/fluffle/goirc/client"
	"github.com/fluffle/golog/logging"
	"github.com/fluffle/sp0rkle/sp0rkle/base"
	"os/exec"
	"strings"
)

type BotHandler func(*Sp0rkle, *base.Line)

// NOTE: Nothing but the bot should register for IRC events!
func (bot *Sp0rkle) RegisterHandlers(r event.EventRegistry) {
	// Generic shim to wrap an irc event into a bot event.
	forward_event := func(name string) event.Handler {
		return client.NewHandler(func(irc *client.Conn, line *client.Line) {
			getState(irc).Dispatch("bot_"+name, Line(line))
		})
	}

	r.AddHandler(forward_event("privmsg"), "privmsg")
	r.AddHandler(forward_event("action"), "action")
	// These are mostly for the seen plugin.
	r.AddHandler(forward_event("join"), "join")
	r.AddHandler(forward_event("part"), "part")
	r.AddHandler(forward_event("kick"), "kick")
	r.AddHandler(forward_event("quit"), "quit")
	r.AddHandler(forward_event("nick"), "nick")

}

// Unboxer for bot handlers.
func NewHandler(f BotHandler) event.Handler {
	return event.NewHandler(func(ev ...interface{}) {
		f(ev[0].(*Sp0rkle), ev[1].(*base.Line))
	})
}

func bot_connected(line *base.Line) {
	for _, c := range bot.channels {
		logging.Info("Joining %s on startup.\n", c)
		irc.Join(c)
	}
}

func bot_disconnected(line *base.Line) {
	bot.Quit <- bot.quit
	logging.Info("Disconnected...")
}

func bot_command(l *base.Line) {
	if cmd := commands.Match(l.Args[1]); l.Addressed && cmd != nil {
		cmd.Execute(l)
	}
}

// Retrieve the bot from irc.State.
func getState(irc *client.Conn) *Sp0rkle {
	return irc.State.(*Sp0rkle)
}

func bot_rebuild(line *base.Line) {
	if bot.rbnick == "" || bot.rbnick != line.Nick { return }
	if !strings.HasPrefix(line.Args[1], "rebuild") { return }
	if bot.rbpw != "" && line.Args[1] != "rebuild "+bot.rbpw { return }

	// Ok, we should be good to rebuild now.
	irc.Notice(line.Nick, "Beginning rebuild")
	cmd := exec.Command("go", "get", "-u", "github.com/fluffle/sp0rkle/sp0rkle")
	out, err := cmd.CombinedOutput()
	if err != nil {
		irc.Notice(line.Nick, fmt.Sprintf("Rebuild failed: %s", err))
		for _, l := range strings.Split(string(out), "\n") {
			irc.Notice(line.Nick, l)
		}
		return
	}
	bot.quit = true
	bot.reexec = true
	irc.Quit("Restarting with new build.")
}

func bot_shutdown(line *base.Line) {
	if bot.rbnick == "" || bot.rbnick != line.Nick { return }
	if !strings.HasPrefix(line.Args[1], "shutdown") { return }
	if bot.rbpw != "" && line.Args[1] != "shutdown "+bot.rbpw { return }
	bot.quit = true
	bot.Conn.Quit("Shutting down.")
}

func bot_help(line *base.Line) {
	s := strings.Join(strings.Fields(line.Args[1])[1:], " ")
	if cmd := commands.Match(s); cmd != nil {
		bot.ReplyN(line, cmd.Help())
	} else if len(s) == 0 {
		bot.ReplyN(line, "https://github.com/fluffle/sp0rkle/wiki " +
			"-- pull requests welcome ;-)")
	} else {
		bot.ReplyN(line, "Unrecognised command '%s'.", s)
	}
}