package slackbot

import (
	"regexp"

	"context"

	"github.com/flw-cn/slack"
)

// Route represents a route
type Route struct {
	handler      Handler
	err          error
	matchers     []Matcher
	subrouter    Router
	preprocessor Preprocessor
	botUserID    string
	talkToSelf   bool // if set, the bot can reply to its own messages
}

func (r *Route) setBotID(botID string) {
	r.botUserID = botID
	for _, matcher := range r.matchers {
		matcher.SetBotID(botID)
	}
}

// RouteMatch stores information about a matched route.
type RouteMatch struct {
	Route   *Route
	Handler Handler
}

// EventMatch stores information about a matched event
type EventMatch struct {
	Handler EventHandler
}

// ChannelJoinMatch stores information about a channel joined event
type ChannelJoinMatch struct {
	Handler ChannelJoinHandler
}

// Handle calls the handler with provided parameters
func (cjm *ChannelJoinMatch) Handle(ctx context.Context, b *Bot, channel *slack.Channel) {
	cjm.Handler(ctx, b, channel)
}

// Handle handles any unspecified RTM events
func (em *EventMatch) Handle(ctx context.Context, b *Bot, ev *slack.RTMEvent) {
	em.Handler(ctx, b, ev)
}

// Match matches
func (r *Route) Match(ctx context.Context, match *RouteMatch) (bool, context.Context) {
	if r.err != nil {
		return false, ctx
	}

	if r.handler == nil && r.subrouter == nil {
		return false, ctx
	}

	if ev := MessageFromContext(ctx); ev != nil && !r.talkToSelf && r.botUserID == ev.User {
		return false, ctx
	}

	if r.preprocessor != nil {
		ctx = r.preprocessor(ctx)
	}
	for _, m := range r.matchers {
		var matched bool
		if matched, ctx = m.Match(ctx); !matched {
			return false, ctx
		}
	}

	// if this route contains a subrouter, invoke the subrouter match
	if r.subrouter != nil {
		return r.subrouter.Match(ctx, match)
	}

	match.Route = r
	match.Handler = r.handler
	return true, ctx
}

func (r *Route) TalkToSelf() *Route {
	r.talkToSelf = true
	return r
}

func (r *Route) NoTalkToSelf() *Route {
	r.talkToSelf = false
	return r
}

// Hear adds a matcher for the message text
func (r *Route) Hear(regex string) *Route {
	r.addRegexpMatcher(regex)
	return r
}

// Messages sets the types of Messages we want to handle
func (r *Route) Messages(types ...MessageType) *Route {
	_ = r.addTypesMatcher(types...)
	return r
}

// Handler sets a handler for the route.
func (r *Route) Handler(handler Handler) *Route {
	if r.err == nil {
		r.handler = handler

	}
	return r
}

// MessageHandler is a message handler
func (r *Route) MessageHandler(fn MessageHandler) *Route {
	return r.Handler(func(ctx context.Context) {
		bot := BotFromContext(ctx)
		msg := MessageFromContext(ctx)
		fn(ctx, bot, msg)
	})
}

// Preprocess preproccesses
func (r *Route) Preprocess(fn Preprocessor) *Route {
	r.preprocessor = fn
	return r
}

// Subrouter creates a subrouter
func (r *Route) Subrouter() Router {
	r.subrouter = &SimpleRouter{err: r.err}
	return r.subrouter
}

// AddMatcher adds a matcher to the route.
func (r *Route) AddMatcher(m Matcher) *Route {
	r.matchers = append(r.matchers, m)
	return r
}

func (r *Route) Err() error {
	return r.err
}

// RegexpMatcher is a regexp matcher
type RegexpMatcher struct {
	regex     *regexp.Regexp
	botUserID string
}

// Match matches a message
func (rm *RegexpMatcher) Match(ctx context.Context) (bool, context.Context) {
	msg := MessageFromContext(ctx)
	// A message be receded by a direct mention. For simplicity sake, strip out any potention direct mentions first
	text := StripDirectMention(msg.Text)
	// now consider stripped text against regular expression
	matched, matches := namedRegexpParse(text, rm.regex)
	if !matched {
		return false, ctx
	}
	var namedCaptures = NamedCaptures{}
	namedCaptures.m = make(map[string]string)
	for k, v := range matches {
		namedCaptures.m[k] = v
	}
	newCtx := context.WithValue(ctx, namedCaptureContext, namedCaptures)

	return true, newCtx
}

// SetBotID sets the bot id
func (rm *RegexpMatcher) SetBotID(botID string) {
	rm.botUserID = botID
}

// addRegexpMatcher adds a host or path matcher and builder to a route.
func (r *Route) addRegexpMatcher(regex string) {
	re, err := regexp.Compile(regex)
	if err != nil {
		r.err = err
		return
	}

	r.AddMatcher(&RegexpMatcher{regex: re})
}

// TypesMatcher is a type matcher
type TypesMatcher struct {
	types     []MessageType
	botUserID string
}

// Match matches
func (tm *TypesMatcher) Match(ctx context.Context) (bool, context.Context) {
	msg := MessageFromContext(ctx)
	bot := BotFromContext(ctx)
	for _, t := range tm.types {
		switch t {
		case DirectMessage:
			if IsDirectMessage(bot.RTM, msg) {
				return true, ctx
			}
		case DirectMention:
			if IsDirectMention(bot.RTM, msg, botUserID) {
				return true, ctx
			}
		case Message:
			if IsMessage(bot.RTM, msg) {
				return true, ctx
			}
		case Mention:
			if IsMentioned(msg, botUserID) {
				return true, ctx
			}
		}
	}
	return false, ctx
}

// SetBotID sets the botid
func (tm *TypesMatcher) SetBotID(botID string) {
	tm.botUserID = botID
}

// addTypesMatcher adds a host or path matcher and builder to a route.
func (r *Route) addTypesMatcher(types ...MessageType) error {
	if r.err != nil {
		return r.err
	}

	r.AddMatcher(&TypesMatcher{types: types, botUserID: r.botUserID})
	return nil
}
