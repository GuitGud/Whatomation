package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"zapsender/config"
	"zapsender/handlers"
	"zapsender/utils"
	"zapsender/whatsapp"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// ResponderConfig estrutura para configurar o monitoramento de usuário em grupo
type ResponderConfig struct {
	TargetUserJID string // JID do usuário para monitorar (formato: 5511999999999@s.whatsapp.net)
	GroupJID      string // JID do grupo para monitorar (formato: 123456789@g.us)
	Response      string // Mensagem que será enviada como resposta
}

// MonitorarUsuarioEmGrupo inicia uma goroutine que monitora mensagens
// de um usuário específico em um grupo específico
func MonitorarUsuarioEmGrupo(client *whatsapp.Client, config ResponderConfig) {
	targetUserJID, err := types.ParseJID(config.TargetUserJID)
	if err != nil {
		log.Printf("Erro ao parsear JID do usuário alvo: %v", err)
		return
	}

	groupJID, err := types.ParseJID(config.GroupJID)
	if err != nil {
		log.Printf("Erro ao parsear JID do grupo: %v", err)
		return
	}

	if groupJID.Server != "g.us" {
		log.Printf("O JID fornecido não é um grupo válido: %s", config.GroupJID)
		return
	}

	eventHandler := func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			if v.Info.Sender.String() == targetUserJID.String() {
				log.Printf("Mensagem recebida do usuário alvo no grupo: %s", v.Message.GetConversation())

				go func() {
					err := client.SendTextMessage(targetUserJID.String(), config.Response)
					if err != nil {
						log.Printf("Erro ao enviar resposta automática: %v", err)
					} else {
						log.Println("Resposta automática enviada com sucesso!")
					}
				}()
			}
		}
	}

	client.AddCustomEventHandler(eventHandler)
	log.Printf("Monitoramento iniciado para o usuário %s no grupo %s", config.TargetUserJID, config.GroupJID)
}

func FirstResponder(client *whatsapp.Client, numero string, mensagem string) {
	if !strings.Contains(numero, "@") {
		numero = numero + "@s.whatsapp.net"
	}

	err := client.SendTextMessage(numero, mensagem)
	if err != nil {
		log.Printf("Erro ao enviar mensagem: %v", err)
	} else {
		log.Println("Mensagem enviada com sucesso!")
	}
}

func main() {
	cfg := config.NewConfig()

	if _, err := os.Stat(cfg.SessionPath); os.IsNotExist(err) {
		err = os.MkdirAll(cfg.SessionPath, 0755)
		if err != nil {
			fmt.Printf("Erro ao criar diretório de sessão: %v\n", err)
			return
		}
	}

	client, err := whatsapp.NewClient(cfg.SessionPath)
	if err != nil {
		fmt.Printf("Erro ao criar cliente WhatsApp: %v\n", err)
		return
	}

	client.SetEventHandler(handlers.MessageHandler)

	err = client.Connect()
	if err != nil {
		fmt.Printf("Erro ao conectar: %v\n", err)
		return
	}

	fmt.Println("Cliente WhatsApp conectado!")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	var monitoresAtivos []ResponderConfig

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("\nComandos disponíveis:")
		fmt.Println("- enviar <número> <mensagem>")
		fmt.Println("- bomba <número> <mensagem> <quantidade> <delay_ms>")
		fmt.Println("- monitorar <número_usuário> <id_grupo> <resposta>")
		fmt.Println("- listar_monitores")
		fmt.Println("- sair")

		for scanner.Scan() {
			text := scanner.Text()

			if text == "sair" {
				c <- os.Interrupt
				break
			} else if strings.HasPrefix(text, "enviar ") {
				parts := strings.SplitN(text[7:], " ", 2)
				if len(parts) != 2 {
					fmt.Println("Formato inválido. Use: enviar <número> <mensagem>")
					continue
				}

				number := parts[0]
				message := parts[1]

				if !strings.Contains(number, "@") {
					number = number + "@s.whatsapp.net"
				}

				err := client.SendTextMessage(number, message)
				if err != nil {
					fmt.Printf("Erro ao enviar mensagem: %v\n", err)
				} else {
					fmt.Println("Mensagem enviada com sucesso!")
				}
			} else if strings.HasPrefix(text, "bomba ") {
				comandoCompleto := text[6:]
				partes := strings.SplitN(comandoCompleto, " ", 4)

				if len(partes) < 4 {
					fmt.Println("Formato inválido. Use: bomba <número> <mensagem> <quantidade> <delay_ms>")
					continue
				}

				numero := partes[0]
				mensagem := partes[1]
				qtd, err := strconv.Atoi(partes[2])
				if err != nil {
					fmt.Println("Quantidade inválida.")
					continue
				}

				delay, err := strconv.Atoi(partes[3])
				if err != nil {
					fmt.Println("Delay inválido.")
					continue
				}

				if !strings.Contains(numero, "@") {
					numero = numero + "@s.whatsapp.net"
				}

				fmt.Printf("Enviando %d mensagens para %s com intervalo de %dms\n", qtd, numero, delay)

				go func() {
					utils.MensagemBombing(client, numero, mensagem, qtd, delay)
				}()

			} else if strings.HasPrefix(text, "monitorar ") {
				comandoCompleto := text[10:]
				partes := strings.SplitN(comandoCompleto, " ", 3)

				if len(partes) < 3 {
					fmt.Println("Formato inválido. Use: monitorar <número_usuário> <id_grupo> <resposta>")
					continue
				}

				numeroUsuario := partes[0]
				idGrupo := partes[1]
				resposta := partes[2]

				if !strings.Contains(numeroUsuario, "@") {
					numeroUsuario = numeroUsuario + "@s.whatsapp.net"
				}

				config := ResponderConfig{
					TargetUserJID: numeroUsuario,
					GroupJID:      idGrupo,
					Response:      resposta,
				}

				monitoresAtivos = append(monitoresAtivos, config)
				go MonitorarUsuarioEmGrupo(client, config)

				fmt.Printf("Monitoramento iniciado para o usuário %s no grupo %s\n", numeroUsuario, idGrupo)

			} else if text == "listar_monitores" {
				if len(monitoresAtivos) == 0 {
					fmt.Println("Não há monitores ativos no momento.")
				} else {
					fmt.Println("\nMonitores ativos:")
					for i, monitor := range monitoresAtivos {
						fmt.Printf("%d. Usuário: %s | Grupo: %s | Resposta: %s\n",
							i+1, monitor.TargetUserJID, monitor.GroupJID, monitor.Response)
					}
				}
			} else {
				fmt.Println("Comando não reconhecido. Comandos disponíveis:")
				fmt.Println("- enviar <número> <mensagem>")
				fmt.Println("- bomba <número> <mensagem> <quantidade> <delay_ms>")
				fmt.Println("- monitorar <número_usuário> <id_grupo> <resposta>")
				fmt.Println("- listar_monitores")
				fmt.Println("- sair")
			}
		}
	}()

	go func() {
		utils.SendScheduledMessage(client.WAClient)
	}()

	<-c
	fmt.Println("\nEncerrando cliente...")
	client.Close()
	fmt.Println("Cliente encerrado. Até logo!")
}
