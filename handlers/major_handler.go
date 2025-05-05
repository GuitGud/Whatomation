package handlers

import (
	"fmt"

	"go.mau.fi/whatsmeow/types/events"
)

// MessageHandler processa eventos de mensagem
func MessageHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Printf("Mensagem recebida de %s: %s\n", v.Info.Sender.String(), v.Message.GetConversation())
	case *events.Connected:
		fmt.Println("Conexão estabelecida com sucesso")
	case *events.Disconnected:
		fmt.Println("Desconectado do WhatsApp")
	case *events.LoggedOut:
		fmt.Println("Sessão encerrada, necessário novo login")
	}
}
