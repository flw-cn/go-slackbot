package slackbot

// Package slackbot hopes to ease development of Slack bots by adding helpful
// methods and a mux-router style interface to the github.com/flw-cn/slack package.
//
// Incoming Slack RTM events are mapped to a handler in the following form:
// 	bot.Hear("(?i)how are you(.*)").MessageHandler(HowAreYouHandler)
//
// The package adds Reply, ReplyInThread and ReplyWithAttachments methods:
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
// The slackbot package exposes github.com/flw-cn/slack RTM and Client objects
// enabling a consumer to interact with the lower level package directly:
// 	func HowAreYouHandler(ctx context.Context, bot *slackbot.Bot, evt *slack.MessageEvent) {
// 		bot.RTM.NewOutgoingMessage("Hello", "#random")
// 	}
