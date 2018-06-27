package slackbot

import (
	"context"

	"github.com/flw-cn/slack"
)

// MessageType represents a message type
type MessageType int

const (
	DirectMessage MessageType = iota // someone message me one by one
	DirectMention                    // someone mention me in a DirectMessage
	Message                          // normal message, just like channel chat message
	Mention                          // someone mention me in a Message
	Ambient                          // what?
	FileShared                       // someone shared a file
)

// Handler is a handler
type Handler func(context.Context)

// ChannelJoinHandler handles channel join events
type ChannelJoinHandler func(context.Context, *Bot, *slack.Channel)

// EventHandler handles events in a generic fashion
type EventHandler func(context.Context, *Bot, *slack.RTMEvent)

// MessageHandler is a message handler
type MessageHandler func(ctx context.Context, bot *Bot, msg *slack.MessageEvent)

// Preprocessor is a preprocessor
type Preprocessor func(context.Context) context.Context

// Matcher type for matching message routes
type Matcher interface {
	Match(context.Context) (bool, context.Context)
	SetBotID(botID string)
}

// NamedCaptures is a container for any named captures in our context
type NamedCaptures struct {
	m map[string]string
}

// Get returns a value from a key lookup
func (nc NamedCaptures) Get(key string) string {
	v, ok := nc.m[key]
	if !ok {
		return ""
	}
	return v
}
