package telegram

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tg"
	"github.com/idkroff/tg-intersector/internal/flow"
	getter "github.com/idkroff/tg-intersector/internal/set-getter"
)

type IntersectorClient struct {
	Options          IntersectorClientOptions
	Intersector      *telegram.Client
	UpdateDispatcher tg.UpdateDispatcher
}

type IntersectorClientOptions struct {
	API_ID   int
	API_HASH string
	Phone    string
	Password string
	AuthFlow string
}

func New(options IntersectorClientOptions) IntersectorClient {
	sessionDir := filepath.Join("session", flow.SessionFolderName(options.Phone))
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		log.Fatalf("error creating session dir: %s", err)
	}

	sessionStorage := &telegram.FileSessionStorage{
		Path: filepath.Join(sessionDir, "session.json"),
	}

	d := tg.NewUpdateDispatcher()
	client := telegram.NewClient(
		options.API_ID,
		options.API_HASH,
		telegram.Options{UpdateHandler: d, SessionStorage: sessionStorage},
	)

	return IntersectorClient{
		Options:          options,
		Intersector:      client,
		UpdateDispatcher: d,
	}
}

func (client *IntersectorClient) Authorize(ctx context.Context) {
	if client.Options.AuthFlow == "code" {
		if err := auth.NewFlow(
			auth.Constant(client.Options.Phone, client.Options.Password, auth.CodeAuthenticatorFunc(flow.CodePrompt)),
			auth.SendCodeOptions{},
		).Run(ctx, client.Intersector.Auth()); err != nil {
			log.Fatalf("unable to authirize: %s", err)
		}
	} else if client.Options.AuthFlow == "qr" {
		if _, err := client.Intersector.QR().Auth(
			ctx,
			qrlogin.OnLoginToken(client.UpdateDispatcher),
			flow.ShowQR,
		); err != nil {
			log.Fatalf("unable to auth via qr: %s", err)
		}
	}

	status, err := client.Intersector.Auth().Status(ctx)
	if err != nil {
		log.Fatalf("error: unable to check auth status: %s", err)
	}

	if !status.Authorized {
		log.Fatal("error: not authorized")
	}

	log.Println("Logged in successfully")
}

func (client *IntersectorClient) RunFetch() {
	if err := client.Intersector.Run(context.Background(), func(ctx context.Context) error {
		log.Println("starting auth")
		client.Authorize(ctx)

		api := client.Intersector.API()

		dialogsClass, err := api.MessagesGetDialogs(
			ctx,
			&tg.MessagesGetDialogsRequest{OffsetPeer: &tg.InputPeerChannel{}, Limit: 10000},
		)
		if err != nil {
			return fmt.Errorf("cannot fetch dialogs: %w", err)
		}

		dialogs, ok := dialogsClass.(*tg.MessagesDialogsSlice)
		if !ok {
			return fmt.Errorf("cannot cast dialogs class")
		}

		dialogsInfo := ""
		log.Println(fmt.Sprintf("%d dialogs fetched", len(dialogs.Dialogs)))
		for _, dialogClass := range dialogs.Dialogs {
			dialog, ok := dialogClass.(*tg.Dialog)
			if !ok {
				log.Println("unable to cast dialog class")
				continue
			}

			log.Println(reflect.TypeOf(dialog.Peer))

			peerClass := dialog.Peer
			switch peerFetched := peerClass.(type) {
			case *tg.PeerChat:
				chatID := peerFetched.GetChatID()
				fullChats, err := api.MessagesGetChats(ctx, []int64{chatID})
				if err != nil {
					log.Println(fmt.Sprintf("unable to fetch chat: %d", chatID))
					continue
				}

				log.Println(chatID, fullChats)

				fullChatsList := fullChats.GetChats()
				if len(fullChatsList) != 1 {
					log.Println(fmt.Sprintf("unable to fetch chat: %d (list len not 1)", chatID))
					continue
				}
				fullChat, ok := fullChatsList[0].AsFull()
				if !ok {
					log.Println(fmt.Sprintf("unable to cast chat: %d to full chat", chatID))
					continue
				}

				dialogsInfo += fmt.Sprintf("%d: %s\n", fullChat.GetID(), fullChat.GetTitle())
			case *tg.PeerChannel:
				inputChannel := &tg.InputChannel{
					ChannelID:  peerFetched.GetChannelID(),
					AccessHash: 0,
				}
				channels, err := api.ChannelsGetChannels(ctx, []tg.InputChannelClass{inputChannel})

				if err != nil {
					log.Printf("failed to fetch channel: %d: %s\n", peerFetched.GetChannelID(), err)
				}

				if len(channels.GetChats()) == 0 {
					log.Printf("channel not found: %d: %s\n", peerFetched.GetChannelID(), err)
				}

				channel := channels.GetChats()[0].(*tg.Channel)
				dialogsInfo += fmt.Sprintf("%d: %s\n", channel.GetID(), channel.GetTitle())
			}
		}

		if err := os.WriteFile("chats.txt", []byte(dialogsInfo), 0666); err != nil {
			return fmt.Errorf("unable to save chats to file: %w", err)
		}

		log.Println("ok")

		return nil
	}); err != nil {
		log.Fatalf("Error while running API: %s", err)
	}
}

func (client *IntersectorClient) RunIntersection() {
	if err := client.Intersector.Run(context.Background(), func(ctx context.Context) error {
		log.Println("starting auth")
		client.Authorize(ctx)

		api := client.Intersector.API()

		chatSet1 := getter.GetSet()
		chatSet2 := getter.GetSet()

		usersSet1 := map[int64]bool{}
		usersSet2 := map[int64]bool{}

		for _, channelAlias := range chatSet1 {
			participants, err := fetchParticipantsFromChannel(
				ctx,
				api,
				channelAlias,
			)
			if err != nil {
				log.Printf("unable to handle %s: %s", channelAlias, err)
				continue
			}

			for _, p := range participants {
				usersSet1[p] = true
			}
		}

		for _, channelAlias := range chatSet2 {
			participants, err := fetchParticipantsFromChannel(
				ctx,
				api,
				channelAlias,
			)
			if err != nil {
				log.Printf("unable to handle %s: %s", channelAlias, err)
				continue
			}

			for _, p := range participants {
				usersSet2[p] = true
			}
		}

		intersectedUsers := []int64{}
		for k := range usersSet1 {
			if _, ok := usersSet2[k]; ok {
				intersectedUsers = append(intersectedUsers, k)
			}
		}

		log.Println(fmt.Sprint(intersectedUsers))

		return nil
	}); err != nil {
		log.Fatalf("Error while running API: %s", err)
	}
}

func fetchParticipantsFromChannel(ctx context.Context, api *tg.Client, channelAlias string) ([]int64, error) {
	users := []int64{}

	var inputChannel *tg.InputChannel
	if strings.HasPrefix(channelAlias, "id_") {
		channelID, err := strconv.Atoi(channelAlias[3:])
		if err != nil {
			return nil, fmt.Errorf("unable to parse channel id: %s: %w", channelAlias, err)
		}

		inputChannel = &tg.InputChannel{
			ChannelID:  int64(channelID),
			AccessHash: 0,
		}
	} else {
		peer, err := api.ContactsResolveUsername(ctx, channelAlias)
		if err != nil {
			return nil, fmt.Errorf("resolve peer %s errored: %w", channelAlias, err)
		}

		inputChannel = &tg.InputChannel{
			ChannelID:  peer.Chats[0].GetID(),
			AccessHash: 0,
		}
	}

	channels, err := api.ChannelsGetChannels(ctx, []tg.InputChannelClass{inputChannel})

	if err != nil {
		log.Printf("failed to fetch channel: %s: %s\n", channelAlias, err)
	}

	if len(channels.GetChats()) == 0 {
		log.Printf("channel not found: %s: %s\n", channelAlias, err)
	}

	channel := channels.GetChats()[0].(*tg.Channel)

	participantsClass, err := api.ChannelsGetParticipants(ctx, &tg.ChannelsGetParticipantsRequest{
		Channel: &tg.InputChannel{ChannelID: channel.ID, AccessHash: channel.AccessHash},
		Filter:  &tg.ChannelParticipantsSearch{},
	})
	if err != nil {
		return nil, fmt.Errorf("get participants errored: %w", err)
	}

	var participants *tg.ChannelsChannelParticipants
	switch participantsFetched := participantsClass.(type) {
	case *tg.ChannelsChannelParticipants:
		participants = participantsFetched
	}

	log.Printf("channel fetched: %s [%d users]\n", channelAlias, participants.Count)

	for _, participant := range participants.Participants {
		switch user := participant.(type) {
		case *tg.ChannelParticipant:
			users = append(users, user.UserID)
		}
	}

	return users, nil
}
