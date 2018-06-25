package slackbot

import (
	"context"
	"fmt"

	"github.com/flw-cn/slack"
)

// key is unexported so other packages cannot access these keys directly or by mimicking their values.
// This ensures that messages and bots can only be added to or retrieved from the context via these functions.
type key int

const (
	// BotContext is the context key for the bot context entry
	BotContext = iota
	// MessageContext is the context key for the message context entry
	MessageContext
	// NamedCaptureContextKey is the key for named captures
	NamedCaptureContextKey
)

// BotFromContext creates a Bot from provided Context
func BotFromContext(ctx context.Context) *Bot {
	if result, ok := ctx.Value(contextKey(BotContext)).(*Bot); ok {
		return result
	}
	fmt.Printf("Got a nil bot from context: %#v\n", ctx)
	return nil
}

// AddBotToContext sets the bot reference in context and returns the newly derived context
func AddBotToContext(ctx context.Context, bot *Bot) context.Context {
	return context.WithValue(ctx, contextKey(BotContext), bot)
}

// MessageFromContext gets the message from the provided context
func MessageFromContext(ctx context.Context) *slack.MessageEvent {
	if result, ok := ctx.Value(contextKey(MessageContext)).(*slack.MessageEvent); ok {
		return result
	}
	return nil
}

// AddMessageToContext sets the Slack message event reference in context and returns the newly derived context
func AddMessageToContext(ctx context.Context, msg *slack.MessageEvent) context.Context {
	return context.WithValue(ctx, contextKey(MessageContext), msg)
}

// NamedCapturesFromContext returns any NamedCaptures parsed from regexp
func NamedCapturesFromContext(ctx context.Context) NamedCaptures {
	if result, ok := ctx.Value(contextKey(NamedCaptureContextKey)).(NamedCaptures); ok {
		return result
	}
	return NamedCaptures{}
}
