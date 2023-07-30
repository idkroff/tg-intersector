package flow

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tg"
)

func CodePrompt(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
	fmt.Print("Enter code: ")
	code, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
}

func ShowQR(ctx context.Context, token qrlogin.Token) error {
	url := fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=%s", token.URL())
	fmt.Printf("Open %s using your phone\n", url)
	return nil
}
