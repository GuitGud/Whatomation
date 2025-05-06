package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"zapsender/config"
	"zapsender/handlers"
	"zapsender/utils"
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

	// Interface de linha de comando melhorada
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("\nComandos disponíveis:")
		fmt.Println("- enviar <número> <mensagem> : Envia uma mensagem")
		fmt.Println("- bomba <número> <mensagem> <quantidade> <delay_ms> : Envio repetido de mensagens")
		fmt.Println("- sair : Encerra o programa")

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
			} else if strings.HasPrefix(text, "bomba ") {
				// Formato: bomba <número> <mensagem> <quantidade> <delay_ms>
				comandoCompleto := text[6:] // Remove "bomba "
				partes := strings.SplitN(comandoCompleto, " ", 4)

				if len(partes) < 4 {
					fmt.Println("Formato inválido. Use: bomba <número> <mensagem> <quantidade> <delay_ms>")
					fmt.Println("Exemplo: bomba 5511999999999 'Olá mundo!' 10 500")
					continue
				}

				numero := partes[0]
				mensagem := partes[1]
				qtd, err := strconv.Atoi(partes[2])
				if err != nil {
					fmt.Println("Quantidade deve ser um número")
					continue
				}

				delay, err := strconv.Atoi(partes[3])
				if err != nil {
					fmt.Println("Delay deve ser um número (em milissegundos)")
					continue
				}

				// Formatar número para padrão WhatsApp se não estiver formatado
				if !strings.Contains(numero, "@") {
					numero = numero + "@s.whatsapp.net"
				}

				fmt.Printf("Iniciando bombardeio para %s: %d mensagens com intervalo de %dms\n",
					numero, qtd, delay)

				// Iniciar bombardeio em uma goroutine separada
				go func() {
					utils.MensagemBombing(client, numero, mensagem, qtd, delay)
				}()

				fmt.Println("Bombardeio iniciado em segundo plano!")
				fmt.Println("Você pode continuar usando outros comandos.")

			} else {
				fmt.Println("Comando não reconhecido. Comandos disponíveis:")
				fmt.Println("- enviar <número> <mensagem>")
				fmt.Println("- bomba <número> <mensagem> <quantidade> <delay_ms>")
				fmt.Println("- sair")
			}
		}
	}()

	go func() {
		utils.SendScheduledMessage(client.WAClient)
	}()

	// Aguardar sinal de término
	<-c
	fmt.Println("\nEncerrando cliente...")
	client.Close()
	fmt.Println("Cliente encerrado. Até mais!")
}
