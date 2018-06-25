// Package slackbot hopes to ease development of Slack bots by adding helpful
// methods and a mux-router style interface to the github.com/essentialkaos/slack package.
//
// Incoming Slack RTM events are mapped to a handler in the following form:
// 	bot.Hear("(?i)how are you(.*)").MessageHandler(HowAreYouHandler)
//
// The package adds Reply and ReplyWithAttachments methods:
//	func HowAreYouHandler(ctx context.Context, bot *slackbot.Bot, evt *slack.MessageEvent) {
// 		bot.Reply(evt, "A bit tired. You get it? A bit?", slackbot.WithTyping)
//	}
//
//	func HowAreYouAttachmentsHandler(ctx context.Context, bot *slackbot.Bot, evt *slack.MessageEvent) {
// 		txt := "Beep Beep Boop is a ridiculously simple hosting platform for your Slackbots."
// 		attachment := slack.Attachment{
// 			Pretext:   "We bring bots to life. :sunglasses: :thumbsup:",
// 			Title:     "Host, deploy and share your bot in seconds.",
// 			TitleLink: "https://beepboophq.com/",
// 			Text:      txt,
// 			Fallback:  txt,
// 			ImageURL:  "https://storage.googleapis.com/beepboophq/_assets/bot-1.22f6fb.png",
// 			Color:     "#7CD197",
// 		}
//
//		attachments := []slack.Attachment{attachment}
//		bot.ReplyWithAttachments(evt, attachments, slackbot.WithTyping)
//	}
//
// The slackbot package exposes  github.com/essentialkaos/slack RTM and Client objects
// enabling a consumer to interact with the lower level package directly:
// 	func HowAreYouHandler(ctx context.Context, bot *slackbot.Bot, evt *slack.MessageEvent) {
// 		bot.RTM.NewOutgoingMessage("Hello", "#random")
// 	}
//
//
// Project home and samples: https://github.com/BeepBoopHQ/go-slackbot
package slackbot

import (
	"fmt"
	"log"
	"time"

	"context"

	"github.com/nlopes/slack"
)

const (
	// WithTyping sends a message with typing indicator
	WithTyping bool = true
	// WithoutTyping sends a message without typing indicator
	WithoutTyping bool = false

	maxTypingSleep time.Duration = time.Millisecond * 2000
)

var botUserID string

// New constructs a new Bot using the slackToken to authorize against the Slack service.
func New(slackToken string) *Bot {
	b := &Bot{Client: slack.New(slackToken)}
	return b
}

// Bot is a bot
type Bot struct {
	SimpleRouter
	// Routes to be matched, in order.
	routes []*Route
	// unhandledEventsHandlers are event handlers for unknown events
	unhandledEventsHandlers []EventHandler
	// channelJoinEventsHandlers are event handlers for channel join events
	channelJoinEventsHandlers []ChannelJoinHandler
	// Slack UserID of the bot UserID
	botUserID string
	// Slack API
	Client *slack.Client
	RTM    *slack.RTM
}

// Run listens for incoming slack RTM events, matching them to an appropriate handler.
func (b *Bot) Run(rtmopts bool) {

	options := slack.RTMOptions{}
	options.UseRTMStart = rtmopts

	b.RTM = b.Client.NewRTMWithOptions(&options)

	go b.RTM.ManageConnection()
	for {
		select {
		case msg := <-b.RTM.IncomingEvents:
			ctx := context.Background()
			ctx = AddBotToContext(ctx, b)
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				log.Printf("Connected: %#v\n", ev.Info.User)
				b.setBotID(ev.Info.User.ID)
				botUserID = ev.Info.User.ID
			case *slack.MessageEvent:
				// ignore messages from the current user, the bot user
				if b.botUserID == ev.User {
					continue
				}

				ctx = AddMessageToContext(ctx, ev)
				var match RouteMatch
				if matched, newCtx := b.Match(ctx, &match); matched {
					match.Handler(newCtx)
				}
			case *slack.ChannelJoinedEvent:
				if len(b.channelJoinEventsHandlers) > 0 {
					for _, h := range b.channelJoinEventsHandlers {
						var handler ChannelJoinMatch
						handler.Handler = h
						go handler.Handle(ctx, b, &ev.Channel)
					}
				}
			case *slack.GroupJoinedEvent:
				if len(b.channelJoinEventsHandlers) > 0 {
					for _, h := range b.channelJoinEventsHandlers {
						var handler ChannelJoinMatch
						handler.Handler = h
						go handler.Handle(ctx, b, &ev.Channel)
					}
				}

			case *slack.InvalidAuthEvent:
				log.Printf("Invalid credentials")
				break

			default:
				// Ignore other events..
				// fmt.Printf("Unexpected: %v\n", msg.Data)
			}
		}
	}
}

// OnChannelJoin handles ChannelJoin events
func (b *Bot) OnChannelJoin(h ChannelJoinHandler) {
	b.channelJoinEventsHandlers = append(b.channelJoinEventsHandlers, h)
}

// Reply replies to a message event with a simple message.
func (b *Bot) Reply(evt *slack.MessageEvent, msg string, typing bool) {
	if typing {
		b.Type(evt, msg)
	}
	b.RTM.SendMessage(b.RTM.NewOutgoingMessage(msg, evt.Channel))
}

// ReplyWithAttachments replys to a message event with a Slack Attachments message.
func (b *Bot) ReplyWithAttachments(evt *slack.MessageEvent, attachments []slack.Attachment, typing bool) {
	params := slack.PostMessageParameters{AsUser: true}
	params.Attachments = attachments

	_, _, _ = b.Client.PostMessage(evt.Msg.Channel, "", params)
}

// ReplyAsThread
func (b *Bot) ReplyInThread(evt *slack.MessageEvent, msg string, typing bool) {
	params := slack.PostMessageParameters{AsUser: true}

	if evt.ThreadTimestamp == "" {
		params.ThreadTimestamp = evt.Timestamp
	} else {
		params.ThreadTimestamp = evt.ThreadTimestamp
	}

	b.Client.PostMessage(evt.Msg.Channel, msg, params)
}

// ReplyInThreadWithAttachments replys to a message event inside a thread with a Slack Attachments message.
func (b *Bot) ReplyInThreadWithAttachments(evt *slack.MessageEvent, attachments []slack.Attachment, typing bool) {
	params := slack.PostMessageParameters{AsUser: true}
	params.Attachments = attachments

	if evt.ThreadTimestamp == "" {
		params.ThreadTimestamp = evt.Timestamp
	} else {
		params.ThreadTimestamp = evt.ThreadTimestamp
	}

	b.Client.PostMessage(evt.Msg.Channel, "", params)
}

// Type sends a typing message and simulates delay (max 2000ms) based on message size.
func (b *Bot) Type(evt *slack.MessageEvent, msg interface{}) {
	msgLen := msgLen(msg)

	sleepDuration := time.Minute * time.Duration(msgLen) / 3000
	if sleepDuration > maxTypingSleep {
		sleepDuration = maxTypingSleep
	}

	b.RTM.SendMessage(b.RTM.NewTypingMessage(evt.Channel))
	time.Sleep(sleepDuration)
}

// BotUserID fetches the botUserID.
func (b *Bot) BotUserID() string {
	return b.botUserID
}

func (b *Bot) setBotID(ID string) {
	b.botUserID = ID
	b.SimpleRouter.SetBotID(ID)

	for _, route := range b.routes {
		route.setBotID(ID)
	}
}

// msgLen gets lenght of message and attachment messages. Unsupported types return 0.
func msgLen(msg interface{}) (msgLen int) {
	switch m := msg.(type) {
	case string:
		msgLen = len(m)
	case []slack.Attachment:
		msgLen = len(fmt.Sprintf("%#v", m))
	}
	return
}
