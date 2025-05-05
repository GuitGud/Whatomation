package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"zapsender/config"
	"zapsender/handlers"
	"zapsender/whatsapp"
)

func main() {
	// Carregar configurações
	cfg := config.NewConfig()

	// Criar diretório para sessão se não existir
	if _, err := os.Stat(cfg.SessionPath); os.IsNotExist(err) {
		err = os.MkdirAll(cfg.SessionPath, 0755)
		if err != nil {
			fmt.Printf("Erro ao criar diretório de sessão: %v\n", err)
			return
		}
	}

	// Inicializar cliente WhatsApp
	client, err := whatsapp.NewClient(cfg.SessionPath)
	if err != nil {
		fmt.Printf("Erro ao criar cliente WhatsApp: %v\n", err)
		return
	}

	// Configurar handler de eventos
	client.SetEventHandler(handlers.MessageHandler)

	// Conectar ao WhatsApp
	err = client.Connect()
	if err != nil {
		fmt.Printf("Erro ao conectar: %v\n", err)
		return
	}

	fmt.Println("Cliente WhatsApp conectado!")

	// Configurar captura de sinais para desconexão limpa
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Interface simples de linha de comando
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("\nDigite 'enviar <número> <mensagem>' para enviar uma mensagem")
		fmt.Println("Exemplo: enviar 5511998765432 Olá, tudo bem?")
		fmt.Println("Digite 'sair' para encerrar")

		for scanner.Scan() {
			text := scanner.Text()

			if text == "sair" {
				c <- os.Interrupt
				break
			}

			if strings.HasPrefix(text, "enviar ") {
				parts := strings.SplitN(text[7:], " ", 2)
				if len(parts) != 2 {
					fmt.Println("Formato inválido. Use: enviar <número> <mensagem>")
					continue
				}

				number := parts[0]
				message := parts[1]

				// Formatar número para padrão WhatsApp se não estiver formatado
				if !strings.Contains(number, "@") {
					number = number + "@s.whatsapp.net"
				}

				err := client.SendTextMessage(number, message)
				if err != nil {
					fmt.Printf("Erro ao enviar mensagem: %v\n", err)
				} else {
					fmt.Println("Mensagem enviada com sucesso!")
				}
			} else {
				fmt.Println("Comando não reconhecido")
			}
		}
	}()

	// Aguardar sinal de término
	<-c
	fmt.Println("\nEncerrando cliente...")
	client.Close()
	fmt.Println("Cliente encerrado. Até mais!")
}
