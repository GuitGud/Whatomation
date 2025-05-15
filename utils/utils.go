package utils

import (
	"context"
	"fmt"
	"log"
	"time"
	"zapsender/whatsapp"

	"go.mau.fi/whatsmeow"
	"google.golang.org/protobuf/proto"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// MensagemBombing envia mensagens repetidamente para um número específico
// count: número de mensagens a enviar (0 para infinito)
// delayMs: atraso entre mensagens em milissegundos
// Retorna estatísticas sobre o envio
func MensagemBombing(client *whatsapp.Client, numero string, mensagem string, count int, delayMs int) (enviadas int, falhas int) {
	enviadas = 0
	falhas = 0

	// Verificar se o delay é razoável (pelo menos 100ms)
	if delayMs < 100 {
		delayMs = 100 // Mínimo de 100ms para evitar sobrecarga
	}

	fmt.Printf("Iniciando envio de mensagens para %s\n", numero)
	fmt.Printf("Pressione Ctrl+C para interromper\n")

	// Se count for 0, considerar infinito
	isInfinite := (count == 0)
	i := 0

	for isInfinite || i < count {
		err := client.SendTextMessage(numero, mensagem)

		if err != nil {
			fmt.Printf("Erro ao enviar mensagem #%d: %v\n", i+1, err)
			falhas++
		} else {
			fmt.Printf("Mensagem #%d enviada com sucesso!\n", i+1)
			enviadas++
		}

		// Incrementar contador
		i++

		// Esperar o tempo definido
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	fmt.Printf("\nResumo do envio:\n")
	fmt.Printf("- Total tentado: %d\n", enviadas+falhas)
	fmt.Printf("- Enviadas com sucesso: %d\n", enviadas)
	fmt.Printf("- Falhas: %d\n", falhas)

	return
}

// SendScheduledMessage envia uma mensagem programada para um grupo às 10:15 no horário de Brasília
func SendScheduledMessage(client *whatsmeow.Client) {
	// Parse JID do grupo
	groupJID, err := types.ParseJID("120363319804897565@g.us")
	if err != nil {
		log.Printf("Erro ao parsear JID do grupo: %v", err)
		return
	}

	// Define a mensagem a ser enviada
	text := "Olá, grupo! Esta é sua mensagem diária às 10:15 ⏰"
	msg := &waProto.Message{
		Conversation: &text,
	}

	// Defina o fuso horário de Brasília
	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		log.Fatalf("Erro ao carregar fuso horário de São Paulo: %v", err)
	}

	// Loop para verificar a hora a cada 1 minuto
	for {
		// Hora atual no horário de Brasília
		now := time.Now().In(location)

		// Verifica se já passou das 10:15 AM
		if now.Hour() == 10 && now.Minute() == 15 {
			// Envia a mensagem ao grupo
			_, err := client.SendMessage(context.Background(), groupJID, msg)
			if err != nil {
				log.Printf("Erro ao enviar mensagem para o grupo: %v", err)
			} else {
				log.Println("Mensagem enviada com sucesso para o grupo.")
			}

			// Espera 24 horas antes de tentar novamente (para o próximo dia às 10:15)
			nextSendTime := time.Date(now.Year(), now.Month(), now.Day()+1, 10, 15, 0, 0, location)
			duration := nextSendTime.Sub(now)
			fmt.Printf("Aguardando %v até o próximo envio...\n", duration)
			time.Sleep(duration)
		}

		// Espera 1 minuto antes de verificar novamente
		fmt.Println("Verificando a hora novamente em 1 minuto...")
		time.Sleep(1 * time.Minute)
	}
}

// Configuração para o respondedor
type ResponderConfig struct {
	TargetUserJID string // JID do usuário para monitorar (formato: 5511999999999@s.whatsapp.net)
	GroupJID      string // JID do grupo para monitorar (formato: 123456789@g.us)
	Response      string // Mensagem que será enviada como resposta
}

// MonitorarUsuarioEmGrupo inicia uma goroutine que monitora mensagens
// de um usuário específico em um grupo específico
func MonitorarUsuarioEmGrupo(client *whatsmeow.Client, config ResponderConfig) {
	// Verificando os JIDs
	targetUserJID, err := types.ParseJID(config.TargetUserJID)
	if err != nil {
		log.Fatalf("Erro ao parsear JID do usuário alvo: %v", err)
	}

	groupJID, err := types.ParseJID(config.GroupJID)
	if err != nil {
		log.Fatalf("Erro ao parsear JID do grupo: %v", err)
	}

	// Confirmar que groupJID é realmente um grupo
	if groupJID.Server != "g.us" {
		log.Fatalf("O JID fornecido não é um grupo: %s", config.GroupJID)
	}

	// Criar um canal para receber eventos de mensagem
	eventHandler := func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			// Verificar se a mensagem veio do grupo que estamos monitorando
			if v.Info.Chat.String() == groupJID.String() {
				// Verificar se o remetente é o usuário que estamos monitorando
				if v.Info.Sender.String() == targetUserJID.String() {
					log.Printf("Mensagem recebida do usuário alvo no grupo alvo: %s", v.Message.GetConversation())

					// Responder ao grupo
					go func() {
						_, err := client.SendMessage(context.Background(), groupJID, &waProto.Message{
							Conversation: proto.String(config.Response),
						})
						if err != nil {
							log.Printf("Erro ao enviar mensagem de resposta: %v", err)
						} else {
							log.Println("Resposta enviada com sucesso!")
						}
					}()
				}
			}
		}
	}

	// Registrar o handler de eventos
	client.AddEventHandler(eventHandler)
	log.Printf("Monitoramento iniciado para o usuário %s no grupo %s", config.TargetUserJID, config.GroupJID)
}
