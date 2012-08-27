package quotedriver

import (
	"github.com/fluffle/golog/logging"
	"github.com/fluffle/sp0rkle/lib/db"
	"github.com/fluffle/sp0rkle/lib/quotes"
	//	"github.com/fluffle/sp0rkle/lib/util"
	//	"github.com/fluffle/sp0rkle/sp0rkle/base"
	//	"labix.org/v2/mgo/bson"
	//	"strings"
	"time"
)

const driverName string = "quotes"

type rateLimit struct {
	badness  time.Duration
	lastsent time.Time
}

type quoteDriver struct {
	*quotes.QuoteCollection

	// Data for rate limiting quote lookups per-nick
	limits map[string]*rateLimit

	// logging object
	l logging.Logger
}

func QuoteDriver(db *db.Database, l logging.Logger) *quoteDriver {
	qc := quotes.Collection(db, l)
	return &quoteDriver{
		QuoteCollection: qc,
		limits:          make(map[string]*rateLimit),
		l:               l,
	}
}

func (qd *quoteDriver) Name() string {
	return driverName
}

func (qd *quoteDriver) rateLimit(nick string) bool {
	lim, ok := qd.limits[nick]
	if !ok {
		lim = new(rateLimit)
		qd.limits[nick] = lim
	}
	// limit to 1 quote every 15 seconds, burst to 4 quotes
	elapsed := time.Now().Sub(lim.lastsent)
	if lim.badness += 15*time.Second - elapsed; lim.badness < 0 {
		lim.badness = 0
	}
	if lim.badness > 60*time.Second {
		return true
	}
	lim.lastsent = time.Now()
	return false
}