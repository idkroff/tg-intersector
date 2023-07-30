package telegram

import (
	"context"
	"log"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tg"
	"github.com/idkroff/tg-intersector/internal/flow"
)

type IntersectorClient struct {
	API_ID   int
	API_HASH string
	Phone    string
	Password string
	AuthFlow string
}

func (iClient *IntersectorClient) Run() {
	d := tg.NewUpdateDispatcher()
	client := telegram.NewClient(
		iClient.API_ID,
		iClient.API_HASH,
		telegram.Options{UpdateHandler: d},
	)

	if err := client.Run(context.Background(), func(ctx context.Context) error {
		api := client.API()
		_ = api

		if iClient.AuthFlow == "code" {
			// TODO: add pasword field
			if err := auth.NewFlow(
				auth.Constant(iClient.Phone, iClient.Password, auth.CodeAuthenticatorFunc(flow.CodePrompt)),
				auth.SendCodeOptions{},
			).Run(ctx, client.Auth()); err != nil {
				log.Fatalf("unable to authirize: %s", err)
			}
		} else if iClient.AuthFlow == "qr" {
			if _, err := client.QR().Auth(
				ctx,
				qrlogin.OnLoginToken(d),
				flow.ShowQR,
			); err != nil {
				log.Fatalf("unable to auth via qr: %s", err)
			}
		}

		status, err := client.Auth().Status(ctx)
		if err != nil {
			return err
		}

		if !status.Authorized {
			log.Fatal("error: not authorized")
		}

		log.Println("Logged in successfully")

		return nil
	}); err != nil {
		log.Fatalf("Error while running API: %s", err)
	}
}
